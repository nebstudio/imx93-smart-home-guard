# 基于 i.MX 93 的智能家居行为监护与多模态 AI 交互系统

全国大学生嵌入式芯片与系统设计大赛作品。以恩智浦 i.MX 93（A55 CPU + Ethos-U65 NPU）为核心计算平台，在本地边缘侧完成 AI 推理与逻辑决策，面向独居老人与智能家居场景，实现异常行为监护、环境安全告警与多模态 AI 交互。

## 系统架构

```
i.MX 93 主控（核心大脑）
  ├─ 视觉 AI：摄像头 + MoveNet 姿态识别（跑在 Ethos-U65 NPU，本地推理）
  ├─ 语音 AI：大模型自然对话与语音控制
  ├─ 决策：状态机（跌倒 / 静止 / 火焰 / 烟雾 / 车库）
  └─ 通信：WebSocket 直连 App + 远程推送
        │  USB 串口（文本行协议）
        ▼
Arduino UNO + Sensor Shield（IO 扩展，仅执行硬件动作，不含业务逻辑）
        │
        ▼
传感器与执行器：超声波 / 火焰 / 烟雾 / 三色灯 / 蜂鸣器 / 三舵机（门窗车库）/ 风扇 / LCD
```

所有 AI 推理、状态机决策、网络通信均在 i.MX 93 上完成；Arduino 仅作为 USB 转 GPIO/PWM/ADC 的扩展电路。

## 目录结构

| 目录 | 说明 | 语言/平台 |
|------|------|-----------|
| `arduino_code/` | Arduino UNO 固件（IO 扩展，串口指令协议） | C++ / PlatformIO |
| `imx93_code/` | i.MX 93 主控程序（状态机、串口、语音 Agent、执行器、App 直连） | Go |
| `pose_worker/` | 摄像头姿态识别子进程（MoveNet 推理） | Python |
| `app/` | 上位机手机 App（实时监控、设备控制、告警展示） | Flutter / Dart |
| `deploy/` | 部署相关（systemd 开机自启服务单元） | - |

## 功能概览

- **跌倒 / 长时间静止监护**：摄像头 AI 姿态识别，本地判定，误报低。
- **环境安全告警**：火焰、烟雾多传感融合，声光报警 + 自动排烟。
- **多模态 AI 交互**：语音大模型自然对话，可语音控制家中设备、情绪陪伴。
- **门窗车库控制**：三路舵机控制窗户、门、车库门；车库支持车辆驶离后自动关闭。
- **本地 + 远程双告警**：离线时本地声光报警；联网时推送通知到家人手机。
- **配套 App**：实时状态、设备控制、参数调节、事件记录。

## 构建说明

- **Arduino 固件**：PlatformIO 打开 `arduino_code/`，编译烧录到 Arduino UNO。
- **i.MX 93 主控**：`cd imx93_code && GOOS=linux GOARCH=arm64 go build -o imx93-guard-arm64 .`
- **姿态识别**：`pip install -r pose_worker/requirements.txt`，需 MoveNet tflite 模型。
- **手机 App**：`cd app && flutter build apk --release`

> 运行所需的密钥（语音大模型 API Key 等）通过环境变量 / 编译参数注入，不包含在本仓库中。
> App 连接主控的地址通过 `--dart-define=DEVICE_HOST=... --dart-define=DEVICE_PORT=...` 配置。
