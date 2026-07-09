"""
姿态识别常驻进程（供 Go 主程序作为子进程管理）。

职责：持续读摄像头帧 -> MoveNet Lightning INT8 推理 -> 提取17个关键点
     -> 用简单几何规则判断"是否有人/站姿/倒地姿态" -> 每帧向 stdout 打一行 JSON。

设计原则：
- 这个脚本只做"感知"，不做任何决策阈值判断(那是Go状态机的职责)，
  只是把姿态判断结果(站/坐/倒地/无人)喂出去，具体要不要报警、报警多久
  超时复位，全部交给 Go 那边的 statemachine 处理。
- 每一帧都完整独立输出一行 JSON，Go 端按行读取即可，没有粘包问题。
- stdout 只输出 JSON 数据行；所有调试信息/警告走 stderr，避免污染协议流。

输出协议（每行一个 JSON 对象）：
  {"ts": 1720000000.123, "person": true, "posture": "standing", "conf": 0.62}

  person:  是否检测到人(基于关键点整体置信度阈值判断)
  posture: "standing" | "sitting" | "lying" | "unknown"
           person=false 时 posture 恒为 "none"
  conf:    本次姿态判断所依据的关键点平均置信度(0~1)，供 Go 端记录/调试参考
  frame:   仅在开启"画面流"模式时才会出现这个字段(base64编码的JPEG，
           已经画好人体框+骨架线+姿态文字标注)。默认不开启，不消耗额外
           CPU/带宽，这是"点击播放才传输"设计的核心：APP不点播放按钮，
           这个字段永远不会出现，摄像头照常只做轻量的姿态判断供状态机使用。

画面流控制（stdin，一行一个JSON指令）：
  {"cmd": "stream_on"}   开启画面流(开始画框/编码JPEG/在输出中附带frame字段)
  {"cmd": "stream_off"}  关闭画面流(恢复到只做轻量判断，不再编码画面)
  用独立线程非阻塞读取stdin，不影响主循环的实时推理节奏。
"""

import base64
import json
import sys
import threading
import time

import cv2
import numpy as np

try:
    from tflite_runtime.interpreter import Interpreter
except ImportError:
    from tensorflow.lite.python.interpreter import Interpreter

MODEL_PATH = "movenet_lightning_int8.tflite"
CAMERA_INDEX = 0
FRAME_WIDTH = 640
FRAME_HEIGHT = 480
INPUT_SIZE = 192

KP_NOSE = 0
KP_L_SHOULDER, KP_R_SHOULDER = 5, 6
KP_L_HIP, KP_R_HIP = 11, 12
KP_L_KNEE, KP_R_KNEE = 13, 14
KP_L_ANKLE, KP_R_ANKLE = 15, 16

PERSON_CONF_THRESHOLD = 0.25

SKELETON_EDGES = [
    (5, 6),
    (5, 7), (7, 9),
    (6, 8), (8, 10),
    (5, 11), (6, 12),
    (11, 12),
    (11, 13), (13, 15),
    (12, 14), (14, 16),
    (0, 5), (0, 6),
]

KEYPOINT_DRAW_THRESHOLD = 0.3

POSTURE_LABEL_CN = {
    "standing": "站立",
    "sitting": "坐姿",
    "lying": "倒地",
    "none": "无人",
}

JPEG_QUALITY = 60

def log(msg):
    """调试/状态信息统一走 stderr，不与 stdout 的 JSON 数据流混在一起。"""
    print(msg, file=sys.stderr, flush=True)

def preprocess(frame_bgr):
    """BGR摄像头帧 -> MoveNet输入格式：resize到192x192，BGR转RGB，uint8。"""
    resized = cv2.resize(frame_bgr, (INPUT_SIZE, INPUT_SIZE))
    rgb = cv2.cvtColor(resized, cv2.COLOR_BGR2RGB)
    return np.expand_dims(rgb, axis=0).astype(np.uint8)

def classify_posture(keypoints):
    """
    根据17个关键点(每个点为 [y, x, score]，均为0~1归一化坐标)判断姿态。

    核心思路：站立/坐着时，躯干接近垂直，肩膀明显高于髋部，髋部明显高于膝盖；
    倒地时，人体拉平摊开，肩、髋、膝在图像里的垂直落差(y方向)会大幅缩小，
    同时人体水平方向的跨度(x方向)会明显变大。这是最简单可靠的几何规则，
    不需要训练额外的分类器。

    返回: (person_detected: bool, posture: str, conf: float)
    """
    l_sh, r_sh = keypoints[KP_L_SHOULDER], keypoints[KP_R_SHOULDER]
    l_hip, r_hip = keypoints[KP_L_HIP], keypoints[KP_R_HIP]
    l_knee, r_knee = keypoints[KP_L_KNEE], keypoints[KP_R_KNEE]
    l_ankle, r_ankle = keypoints[KP_L_ANKLE], keypoints[KP_R_ANKLE]

    core_points = [l_sh, r_sh, l_hip, r_hip]
    avg_conf = float(np.mean([p[2] for p in core_points]))

    if avg_conf < PERSON_CONF_THRESHOLD:
        return False, "none", avg_conf

    def mid(p1, p2):
        return ((p1[0] + p2[0]) / 2.0, (p1[1] + p2[1]) / 2.0)

    shoulder_y, shoulder_x = mid(l_sh, r_sh)
    hip_y, hip_x = mid(l_hip, r_hip)

    knee_conf_ok = l_knee[2] > 0.2 and r_knee[2] > 0.2
    if knee_conf_ok:
        knee_y, knee_x = mid(l_knee, r_knee)
    else:
        knee_y, knee_x = hip_y, hip_x

    torso_vertical = abs(hip_y - shoulder_y)
    torso_horizontal = abs(hip_x - shoulder_x)

    hip_knee_vertical = abs(knee_y - hip_y) if knee_conf_ok else None

    if torso_vertical < 0.06 or torso_vertical < torso_horizontal * 0.6:
        return True, "lying", avg_conf

    if hip_knee_vertical is not None and hip_knee_vertical < 0.08:
        return True, "sitting", avg_conf

    return True, "standing", avg_conf

def draw_annotations(frame_bgr, keypoints, person, posture, conf):
    """
    在原始摄像头帧(BGR，未缩放到模型输入尺寸)上画出：
    - 人体骨架线(关键点之间按SKELETON_EDGES连线)
    - 每个高置信度关键点画一个小圆点
    - 一个近似人体框(取所有高置信度关键点的坐标范围，外扩一点边距)
    - 左上角文字标注：姿态判断结果 + 置信度

    直接在传入的frame_bgr上原地绘制并返回，不额外分配新图像，
    减少内存拷贝(画面流场景下每帧都要做这个操作，性能敏感)。
    """
    h, w = frame_bgr.shape[:2]

    if not person:
        cv2.putText(frame_bgr, "未检测到人", (12, 28),
                    cv2.FONT_HERSHEY_SIMPLEX, 0.7, (160, 160, 160), 2, cv2.LINE_AA)
        return frame_bgr

    pts = []
    for kp in keypoints:
        y, x, score = kp
        pts.append((int(x * w), int(y * h), float(score)))

    for a, b in SKELETON_EDGES:
        xa, ya, sa = pts[a]
        xb, yb, sb = pts[b]
        if sa >= KEYPOINT_DRAW_THRESHOLD and sb >= KEYPOINT_DRAW_THRESHOLD:
            cv2.line(frame_bgr, (xa, ya), (xb, yb), (80, 220, 100), 2, cv2.LINE_AA)

    xs, ys = [], []
    for x, y, score in pts:
        if score >= KEYPOINT_DRAW_THRESHOLD:
            cv2.circle(frame_bgr, (x, y), 4, (60, 180, 255), -1, cv2.LINE_AA)
            xs.append(x)
            ys.append(y)

    if xs and ys:
        pad_x = int((max(xs) - min(xs)) * 0.15) + 10
        pad_y = int((max(ys) - min(ys)) * 0.1) + 10
        x0 = max(0, min(xs) - pad_x)
        y0 = max(0, min(ys) - pad_y)
        x1 = min(w - 1, max(xs) + pad_x)
        y1 = min(h - 1, max(ys) + pad_y)

        box_color = (60, 200, 60) if posture != "lying" else (50, 50, 230)
        cv2.rectangle(frame_bgr, (x0, y0), (x1, y1), box_color, 2, cv2.LINE_AA)

        label = f"{POSTURE_LABEL_CN.get(posture, posture)} {conf:.0%}"
        label_y = max(0, y0 - 8)
        cv2.putText(frame_bgr, label, (x0, label_y),
                    cv2.FONT_HERSHEY_SIMPLEX, 0.6, box_color, 2, cv2.LINE_AA)

    return frame_bgr

class StreamToggle:
    """
    在独立线程中监听stdin指令(stream_on/stream_off)，用一个线程安全的
    布尔标志控制主循环是否需要画框+编码JPEG。用单独线程而不是在主循环里
    轮询stdin，是因为stdin.readline()是阻塞调用，不能放进每帧都要跑的
    推理主循环里，否则会拖慢正常的姿态判断节奏。
    """

    def __init__(self):
        self._enabled = False
        self._lock = threading.Lock()
        thread = threading.Thread(target=self._listen, daemon=True)
        thread.start()

    def _listen(self):
        for line in sys.stdin:
            line = line.strip()
            if not line:
                continue
            try:
                cmd = json.loads(line)
            except json.JSONDecodeError:
                continue
            action = cmd.get("cmd")
            if action == "stream_on":
                with self._lock:
                    self._enabled = True
                log("画面流已开启")
            elif action == "stream_off":
                with self._lock:
                    self._enabled = False
                log("画面流已关闭")

    @property
    def enabled(self):
        with self._lock:
            return self._enabled

def main():
    cap = cv2.VideoCapture(CAMERA_INDEX)
    cap.set(cv2.CAP_PROP_FRAME_WIDTH, FRAME_WIDTH)
    cap.set(cv2.CAP_PROP_FRAME_HEIGHT, FRAME_HEIGHT)

    if not cap.isOpened():
        log("错误: 摄像头打开失败")
        sys.exit(1)

    interpreter = Interpreter(model_path=MODEL_PATH, num_threads=2)
    interpreter.allocate_tensors()
    input_details = interpreter.get_input_details()
    output_details = interpreter.get_output_details()

    log(f"pose_worker 已启动，摄像头={CAMERA_INDEX}, 模型输入={input_details[0]['shape']}")

    stream_toggle = StreamToggle()

    for _ in range(5):
        cap.read()

    while True:
        ok, frame = cap.read()
        if not ok:
            log("警告: 读帧失败，跳过本轮")
            time.sleep(0.1)
            continue

        input_tensor = preprocess(frame)
        interpreter.set_tensor(input_details[0]['index'], input_tensor)
        interpreter.invoke()

        keypoints = interpreter.get_tensor(output_details[0]['index'])[0][0]

        person, posture, conf = classify_posture(keypoints)

        result = {
            "ts": time.time(),
            "person": person,
            "posture": posture,
            "conf": round(conf, 3),
        }

        if stream_toggle.enabled:
            annotated = draw_annotations(frame, keypoints, person, posture, conf)
            ok_encode, buf = cv2.imencode(
                ".jpg", annotated, [int(cv2.IMWRITE_JPEG_QUALITY), JPEG_QUALITY]
            )
            if ok_encode:
                result["frame"] = base64.b64encode(buf).decode("ascii")

        print(json.dumps(result), flush=True)

if __name__ == "__main__":
    main()
