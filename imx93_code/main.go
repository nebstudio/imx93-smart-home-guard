package main

import (
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"

	"imx93-guard/actuator"
	"imx93-guard/applink"
	"imx93-guard/posesensor"
	"imx93-guard/serialio"
	"imx93-guard/statemachine"
	"imx93-guard/voiceagent"
)

const (
	analogSmokeIndex = 1
	analogFlameIndex = 2

	pollInterval = 200 * time.Millisecond

	poseMaxAge = 2 * time.Second

	voiceListenSeconds = 6.0

	agentListenSeconds      = 8.0
	agentListenStartTimeout = 6.0
)

func main() {
	portName := flag.String("port", "/dev/cu.usbserial-1130", "Arduino 串口设备路径")
	baud := flag.Int("baud", 115200, "串口波特率")
	verbose := flag.Bool("verbose", false, "打印每一轮的原始传感器读数(调试用，日志会很多)")
	poseEnabled := flag.Bool("pose", false, "是否启用摄像头姿态识别子进程(降低静止/跌倒误报)")
	posePython := flag.String("pose-python", "/mnt/tfcard/npu_setup/movenet_test/venv/bin/python3", "姿态识别子进程使用的 Python 解释器路径")
	poseScript := flag.String("pose-script", "/mnt/tfcard/npu_setup/movenet_test/pose_worker.py", "姿态识别脚本路径")
	voiceEnabled := flag.Bool("voice", false, "是否启用语音确认(告警时主动问话，免唤醒词监听回应)")
	voiceEnvPath := flag.String("voice-env", "/mnt/tfcard/npu_setup/voice_agent/.env", "语音确认所需API密钥的.env文件路径")
	applinkEnabled := flag.Bool("applink", false, "是否启用APP直连功能(板子自身充当WebSocket服务器)")
	applinkAddr := flag.String("applink-addr", ":8080", "APP直连功能的监听地址")
	flag.Parse()

	if *voiceEnabled || *applinkEnabled {

		if err := godotenv.Load(*voiceEnvPath); err != nil {
			log.Printf("警告: 加载语音确认.env文件失败(路径=%s): %v", *voiceEnvPath, err)
		}
	}

	client, err := serialio.Open(*portName, *baud)
	if err != nil {
		log.Fatalf("打开串口失败: %v", err)
	}
	defer client.Close()

	if err := client.Ping(); err != nil {
		log.Fatalf("与 Arduino 通信失败(PING测试未通过): %v", err)
	}
	log.Println("已连接 Arduino，PING 测试通过")

	var poseSensor *posesensor.Sensor
	if *poseEnabled {
		workDir := scriptDir(*poseScript)
		ps, err := posesensor.Start(*posePython, *poseScript, workDir)
		if err != nil {

			log.Printf("警告: 启动姿态识别子进程失败，将仅依赖超声波判断(误报率会更高): %v", err)
		} else {
			poseSensor = ps
			defer poseSensor.Stop()
			log.Println("姿态识别子进程已启动")
		}
	}

	var voiceCfg voiceagent.Config
	if *voiceEnabled || *applinkEnabled {
		cfg, err := voiceagent.LoadConfigFromEnv()
		if err != nil {

			log.Printf("警告: 语音配置加载失败，语音确认和Agent对话都将不可用: %v", err)
			*voiceEnabled = false
		} else {
			voiceCfg = cfg
			if *voiceEnabled {
				log.Println("语音确认已启用")
			}
		}
	}

	var link *applink.Server
	if *applinkEnabled {
		link = applink.New()
		mux := http.NewServeMux()
		link.RegisterHandler(mux)
		go func() {
			log.Printf("APP直连服务已启动，监听 %s (端点: /ws/app)", *applinkAddr)
			if err := http.ListenAndServe(*applinkAddr, mux); err != nil {

				log.Printf("警告: APP直连服务启动失败: %v", err)
			}
		}()
	}

	sm := statemachine.New(statemachine.DefaultConfig())
	act := actuator.New(client)
	agentConv := newAgentConversationController()
	garageCloser := newGarageAutoCloser()

	sysState := newSystemState(voiceCfg.Configured(), *voiceEnabled)

	if link != nil {

		link.BroadcastSystemState(sysState.SystemEnabled(), sysState.VoiceEnabled())
		broadcastConfigState(link, sm)
	}

	var lastBehavior, lastEnv statemachine.State
	var lastManualScenario string
	voiceResultChan := make(chan bool, 1)
	voiceAskInFlight := false
	systemPaused := false

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for now := range ticker.C {

		if link != nil {
			drainAppCommands(link, sm, act, now, &lastManualScenario, voiceCfg, sysState, agentConv, poseSensor)
		}

		if !sysState.SystemEnabled() {
			if !systemPaused {

				if err := act.StopAlarm(); err != nil {
					log.Printf("系统关闭时停止蜂鸣器/风扇出错: %v", err)
				}
				if err := act.SetLight("off"); err != nil {
					log.Printf("系统关闭时熄灭指示灯出错: %v", err)
				}
				systemPaused = true
				log.Println("系统总开关已关闭：跳过传感器判断与执行器动作，仅保持APP连接")
			}

			if link != nil {
				link.BroadcastStatus(pausedStatusPayload(lastBehavior, lastEnv, act, lastManualScenario))
			}
			continue
		}
		systemPaused = false

		snapshot, dist, err := collectSensors(client, now)
		if err != nil {
			log.Printf("采集传感器数据出错: %v", err)
			continue
		}

		attachPoseData(&snapshot, poseSensor)

		garageCloser.tick(dist, act, link)

		select {
		case responded := <-voiceResultChan:
			snapshot.VoiceCancelAlert = responded
			voiceAskInFlight = false
		default:
		}

		behavior, env := sm.Update(snapshot)

		if err := act.ApplyState(behavior, env, now); err != nil {
			log.Printf("下发执行器指令出错: %v", err)
			continue
		}

		justEnteredAlert := behavior != lastBehavior && (behavior == statemachine.StateFallAlert || behavior == statemachine.StateStaticAlert)
		if sysState.VoiceEnabled() && justEnteredAlert && !voiceAskInFlight {
			voiceAskInFlight = true
			go askVoiceConfirm(voiceCfg, behavior, voiceResultChan)
		}

		if *verbose {
			log.Printf("轮询: 车库距离=%dcm 有效=%v 姿态可用=%v 有人=%v 姿态=%s | 行为=%s 环境=%s",
				dist.Cm, dist.Valid,
				snapshot.PoseAvailable, snapshot.PosePerson, snapshot.PosePosture,
				behavior, env)
		}

		if behavior != lastBehavior || env != lastEnv {
			log.Printf("状态变化: 行为=%s 环境=%s (烟雾=%d 火焰=%d 姿态可用=%v 有人=%v 姿态=%s)",
				behavior, env, snapshot.SmokeADC, snapshot.FlameADC,
				snapshot.PoseAvailable, snapshot.PosePerson, snapshot.PosePosture)
			lastBehavior, lastEnv = behavior, env
		}

		if link != nil {
			link.BroadcastStatus(deviceStatusPayload(behavior, env, snapshot, dist, act, lastManualScenario, sysState))
			forwardCameraFrame(link, poseSensor)
		}
	}
}

func broadcastConfigState(link *applink.Server, sm *statemachine.Machine) {
	cfg := sm.ConfigSnapshot()
	link.BroadcastConfig(int(cfg.StaticAlertAfter/time.Second), cfg.FireThreshold, cfg.SmokeThreshold)
}

func pausedStatusPayload(lastBehavior, lastEnv statemachine.State, act *actuator.Actuator, manualScenario string) map[string]any {
	return map[string]any{
		"behavior":        string(lastBehavior),
		"env":             string(lastEnv),
		"distance_cm":     0,
		"distance_valid":  false,
		"smoke_adc":       0,
		"flame_adc":       1023,
		"pose_available":  false,
		"pose_person":     false,
		"pose_posture":    "",
		"light_color":     act.LightColor(),
		"fan_on":          act.FanOn(),
		"window_open":     act.WindowOpen(),
		"door_open":       act.DoorOpen(),
		"garage_open":     act.GarageOpen(),
		"manual_scenario": manualScenario,
		"system_enabled":  false,
		"voice_enabled":   false,
	}
}

func forwardCameraFrame(link *applink.Server, poseSensor *posesensor.Sensor) {
	if poseSensor == nil {
		return
	}
	pose, ok := poseSensor.Latest()
	if !ok || pose.Frame == "" {
		return
	}
	link.BroadcastFrame(pose.Frame, pose.Posture, pose.Person)
}

func deviceStatusPayload(behavior, env statemachine.State, snapshot statemachine.SensorSnapshot, dist DistanceReading, act *actuator.Actuator, manualScenario string, sysState *systemState) map[string]any {
	lightColor := actuator.LogicalLightColor(behavior, env)
	if act.HasManualLightOverride() {

		lightColor = act.LightColor()
	}
	return map[string]any{
		"behavior":        string(behavior),
		"env":             string(env),
		"distance_cm":     dist.Cm,
		"distance_valid":  dist.Valid,
		"smoke_adc":       snapshot.SmokeADC,
		"flame_adc":       snapshot.FlameADC,
		"pose_available":  snapshot.PoseAvailable,
		"pose_person":     snapshot.PosePerson,
		"pose_posture":    snapshot.PosePosture,
		"light_color":     lightColor,
		"fan_on":          act.FanOn(),
		"window_open":     act.WindowOpen(),
		"door_open":       act.DoorOpen(),
		"garage_open":     act.GarageOpen(),
		"manual_scenario": manualScenario,
		"system_enabled":  sysState.SystemEnabled(),
		"voice_enabled":   sysState.VoiceEnabled(),
	}
}

func drainAppCommands(link *applink.Server, sm *statemachine.Machine, act *actuator.Actuator, now time.Time, lastManualScenario *string, voiceCfg voiceagent.Config, sysState *systemState, agentConv *agentConversationController, poseSensor *posesensor.Sensor) {
	const maxPerTick = 5
	for i := 0; i < maxPerTick; i++ {
		select {
		case cmd := <-link.Commands():
			handleAppCommand(cmd, sm, act, now, lastManualScenario, link, voiceCfg, sysState, agentConv, poseSensor)
		default:
			return
		}
	}
}

func handleAppCommand(cmd applink.Command, sm *statemachine.Machine, act *actuator.Actuator, now time.Time, lastManualScenario *string, link *applink.Server, voiceCfg voiceagent.Config, sysState *systemState, agentConv *agentConversationController, poseSensor *posesensor.Sensor) {

	switch cmd.Type {
	case "system_toggle":
		var t struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.Unmarshal(cmd.Data, &t); err != nil {
			log.Printf("解析系统开关指令失败: %v", err)
			return
		}
		sysState.SetSystemEnabled(t.Enabled)
		log.Printf("收到APP系统开关指令: enabled=%v", t.Enabled)
		link.BroadcastSystemState(sysState.SystemEnabled(), sysState.VoiceEnabled())
		return

	case "voice_toggle":
		var t struct {
			Enabled bool `json:"enabled"`
		}
		if err := json.Unmarshal(cmd.Data, &t); err != nil {
			log.Printf("解析语音开关指令失败: %v", err)
			return
		}
		ok := sysState.SetVoiceEnabled(t.Enabled)
		log.Printf("收到APP语音开关指令: enabled=%v 生效=%v", t.Enabled, ok)
		if !ok {
			link.BroadcastEvent("command_failed", "语音功能配置未加载(缺少.env密钥)，无法开启")
		}
		link.BroadcastSystemState(sysState.SystemEnabled(), sysState.VoiceEnabled())
		return

	case "config_update":
		var patch statemachine.ConfigPatch
		if err := json.Unmarshal(cmd.Data, &patch); err != nil {
			log.Printf("解析参数调节指令失败: %v", err)
			return
		}
		sm.UpdateConfig(patch)
		log.Println("收到APP参数调节指令，已应用新的阈值配置")
		broadcastConfigState(link, sm)
		return
	}

	switch cmd.Type {
	case "scenario_command":
		var sc struct {
			Scenario string `json:"scenario"`
		}
		if err := json.Unmarshal(cmd.Data, &sc); err != nil {
			log.Printf("解析场景指令失败: %v", err)
			return
		}
		log.Printf("收到APP场景指令: %s", sc.Scenario)
		switch sc.Scenario {
		case "fall":
			*lastManualScenario = sc.Scenario
			sm.ApplyManualScenario(statemachine.ManualScenarioFall, now)
		case "static":
			*lastManualScenario = sc.Scenario
			sm.ApplyManualScenario(statemachine.ManualScenarioStatic, now)
		case "clear":
			sm.ApplyManualScenario(statemachine.ManualScenarioClear, now)
			*lastManualScenario = ""
		case "alert_send":
			handleAlertSendScenario(link, act)
		default:
			log.Printf("未知场景指令: %s", sc.Scenario)
		}

	case "device_control_command":
		var dc struct {
			Action string `json:"action"`
		}
		if err := json.Unmarshal(cmd.Data, &dc); err != nil {
			log.Printf("解析设备控制指令失败: %v", err)
			return
		}
		log.Printf("收到APP设备控制指令: %s", dc.Action)
		if err := applyDeviceControl(act, dc.Action); err != nil {
			log.Printf("执行设备控制指令失败: %v", err)
		}

	case "agent_conversation_start":

		if !sysState.VoiceEnabled() {
			link.BroadcastEvent("command_failed", "语音功能未启用，无法开始对话")
			return
		}
		agentConv.Start(voiceCfg, link, act, sm)

	case "agent_conversation_stop":

		agentConv.Stop()

	case "agent_chat_text":

		var tc struct {
			Text string `json:"text"`
		}
		if err := json.Unmarshal(cmd.Data, &tc); err != nil {
			log.Printf("解析文字对话指令失败: %v", err)
			return
		}
		link.BroadcastChatTranscript(tc.Text, true)
		handleAgentChatText(tc.Text, link, act, voiceCfg, sm)

	case "camera_stream_start":
		if poseSensor == nil {
			link.BroadcastEvent("command_failed", "摄像头姿态识别未启用，无法开启监控画面")
			return
		}
		if err := poseSensor.SetStreamEnabled(true); err != nil {
			log.Printf("开启摄像头画面流失败: %v", err)
			link.BroadcastEvent("command_failed", "开启监控画面失败")
		}

	case "camera_stream_stop":
		if poseSensor != nil {
			if err := poseSensor.SetStreamEnabled(false); err != nil {
				log.Printf("关闭摄像头画面流失败: %v", err)
			}
		}

	default:
		log.Printf("收到APP未知指令类型: %s", cmd.Type)
	}
}

func handleAlertSendScenario(link *applink.Server, act *actuator.Actuator) {
	link.BroadcastEvent("alert_sent", "已发送至紧急联系人")

	if err := act.NotifyLocalConfirm(); err != nil {
		log.Printf("发送场景本地确认动作失败(不影响APP端已展示的确认提示): %v", err)
	}
}

func handleAgentChatText(text string, link *applink.Server, act *actuator.Actuator, voiceCfg voiceagent.Config, sm *statemachine.Machine) {
	tools := buildAgentTools(act, sm)
	result, err := voiceagent.ChatText(voiceCfg, tools, text, false)
	if err != nil {
		log.Printf("文字对话请求模型失败: %v", err)
		link.BroadcastChatTranscript("抱歉，暂时无法处理这条消息，请稍后再试。", false)
		link.BroadcastAgentState("idle")
		return
	}

	log.Printf("文字对话: 调用工具=%v 回复=%q", result.InvokedTools, result.AssistantText)
	link.BroadcastChatTranscript(result.AssistantText, false)
	link.BroadcastAgentState("idle")
}

func applyDeviceControl(act *actuator.Actuator, action string) error {
	switch action {
	case "fan_on":
		return act.SetFan(true)
	case "fan_off":
		return act.SetFan(false)
	case "light_red":
		return act.SetLight("red")
	case "light_yellow":
		return act.SetLight("yellow")
	case "light_green":
		return act.SetLight("green")
	case "light_off":
		return act.SetLight("off")
	case "window_open":
		return act.SetWindow(true)
	case "window_close":
		return act.SetWindow(false)
	case "door_open":
		return act.SetDoor(true)
	case "door_close":
		return act.SetDoor(false)
	case "garage_open":
		return act.SetGarage(true)
	case "garage_close":
		return act.SetGarage(false)
	default:
		log.Printf("未知设备控制动作: %s", action)
		return nil
	}
}

func askVoiceConfirm(cfg voiceagent.Config, behavior statemachine.State, resultChan chan<- bool) {

	micLock.Lock()
	defer micLock.Unlock()

	prompt := "检测到您可能摔倒了，您还好吗？"
	if behavior == statemachine.StateStaticAlert {
		prompt = "您好，检测到您已经很长时间没有动了，您还好吗？"
	}

	result, err := voiceagent.AskConfirm(cfg, prompt, voiceListenSeconds)
	if err != nil {
		log.Printf("语音确认执行出错(按无回应处理): %v", err)
		resultChan <- false
		return
	}

	if result.Responded {
		log.Printf("语音确认收到回应: asr=%q reply=%q", result.ASRText, result.ChatText)
	} else {
		log.Printf("语音确认未收到回应(reason=%s)", result.Reason)
	}
	resultChan <- result.Responded
}

func scriptDir(scriptPath string) string {
	return filepath.Dir(scriptPath)
}

func attachPoseData(snapshot *statemachine.SensorSnapshot, ps *posesensor.Sensor) {
	if ps == nil {
		return
	}
	if err := ps.Err(); err != nil {
		log.Printf("姿态识别子进程异常: %v", err)
		return
	}
	if !ps.IsFresh(poseMaxAge) {
		return
	}
	pose, ok := ps.Latest()
	if !ok {
		return
	}
	snapshot.PoseAvailable = true
	snapshot.PosePerson = pose.Person
	snapshot.PosePosture = pose.Posture
}

type DistanceReading struct {
	Cm    int
	Valid bool
}

func collectSensors(client *serialio.Client, now time.Time) (statemachine.SensorSnapshot, DistanceReading, error) {
	snapshot := statemachine.SensorSnapshot{Time: now}
	var dist DistanceReading

	distance, err := client.ReadUltrasonicCm()
	if err != nil {
		log.Printf("读取超声波失败: %v", err)
	} else if distance >= 0 {
		dist.Cm = distance
		dist.Valid = true
	}

	smoke, err := client.ReadAnalog(analogSmokeIndex)
	if err != nil {
		log.Printf("读取烟雾传感器失败: %v", err)
	} else {
		snapshot.SmokeADC = smoke
	}

	flame, err := client.ReadAnalog(analogFlameIndex)
	if err != nil {
		log.Printf("读取火焰传感器失败: %v", err)
	} else {
		snapshot.FlameADC = flame
	}

	return snapshot, dist, nil
}
