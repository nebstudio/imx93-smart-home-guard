import cv2
import time

cap = cv2.VideoCapture(0)
cap.set(cv2.CAP_PROP_FRAME_WIDTH, 640)
cap.set(cv2.CAP_PROP_FRAME_HEIGHT, 480)

if not cap.isOpened():
    print("摄像头打开失败")
    exit(1)

print("摄像头已打开，实际分辨率:", cap.get(cv2.CAP_PROP_FRAME_WIDTH), "x", cap.get(cv2.CAP_PROP_FRAME_HEIGHT))

for _ in range(5):
    cap.read()

t0 = time.perf_counter()
ok, frame = cap.read()
t1 = time.perf_counter()

if ok:
    print(f"读帧成功，shape={frame.shape}, dtype={frame.dtype}, 耗时={(t1-t0)*1000:.1f}ms")
    cv2.imwrite("test_frame.jpg", frame)
    print("已保存 test_frame.jpg")
else:
    print("读帧失败")

cap.release()
