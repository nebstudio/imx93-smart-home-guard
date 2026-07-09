package main

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"imx93-guard/actuator"
	"imx93-guard/statemachine"
	"imx93-guard/voiceagent"
)

func buildAgentTools(act *actuator.Actuator, sm *statemachine.Machine) []voiceagent.Tool {
	return []voiceagent.Tool{
		{
			Name:        "control_fan",
			Description: "控制家里的风扇开关。用户说“开风扇”“把风扇关了”之类的话时调用。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"state": map[string]any{
						"type":        "string",
						"enum":        []string{"on", "off"},
						"description": "on表示打开风扇，off表示关闭风扇",
					},
				},
				"required": []string{"state"},
			},
			Handler: func(args map[string]any) (string, error) {
				on := stringArg(args, "state") == "on"
				if err := act.SetFan(on); err != nil {
					return "", err
				}
				if on {
					return "风扇已打开", nil
				}
				return "风扇已关闭", nil
			},
		},
		{
			Name:        "control_light",
			Description: "控制家里的指示灯颜色。用户想切换灯光提示或关灯时调用。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"color": map[string]any{
						"type":        "string",
						"enum":        []string{"red", "yellow", "green", "off"},
						"description": "灯光颜色，off表示熄灭",
					},
				},
				"required": []string{"color"},
			},
			Handler: func(args map[string]any) (string, error) {
				color := stringArg(args, "color")
				if err := act.SetLight(color); err != nil {
					return "", err
				}
				return fmt.Sprintf("指示灯已切换为%s", color), nil
			},
		},
		{
			Name:        "control_window",
			Description: "控制家里的窗户开关。用户说“开窗”“把窗户关了”之类的话时调用。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"state": map[string]any{
						"type":        "string",
						"enum":        []string{"open", "close"},
						"description": "open表示打开窗户，close表示关闭窗户",
					},
				},
				"required": []string{"state"},
			},
			Handler: func(args map[string]any) (string, error) {
				open := stringArg(args, "state") == "open"
				if err := act.SetWindow(open); err != nil {
					return "", err
				}
				if open {
					return "窗户已打开", nil
				}
				return "窗户已关闭", nil
			},
		},
		{
			Name:        "control_door",
			Description: "控制家里的门开关。用户说“开门”“把门关了”之类的话时调用。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"state": map[string]any{
						"type":        "string",
						"enum":        []string{"open", "close"},
						"description": "open表示打开门，close表示关闭门",
					},
				},
				"required": []string{"state"},
			},
			Handler: func(args map[string]any) (string, error) {
				open := stringArg(args, "state") == "open"
				if err := act.SetDoor(open); err != nil {
					return "", err
				}
				if open {
					return "门已打开", nil
				}
				return "门已关闭", nil
			},
		},
		{
			Name:        "control_garage",
			Description: "控制家里的车库门开关。用户说“开车库门”“把车库关了”之类的话时调用。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"state": map[string]any{
						"type":        "string",
						"enum":        []string{"open", "close"},
						"description": "open表示打开车库门，close表示关闭车库门",
					},
				},
				"required": []string{"state"},
			},
			Handler: func(args map[string]any) (string, error) {
				open := stringArg(args, "state") == "open"
				if err := act.SetGarage(open); err != nil {
					return "", err
				}
				if open {
					return "车库门已打开", nil
				}
				return "车库门已关闭", nil
			},
		},
		{
			Name:        "get_weather",
			Description: "查询当前天气情况。用户问天气/温度/是否下雨时调用。",
			Handler: func(args map[string]any) (string, error) {
				return queryWeather(), nil
			},
		},
		{
			Name: "show_emotion",
			Description: "在屏幕上表个情，让对话更生动。可以在合适的时候主动调用" +
				"(比如听到用户开心的事就笑一下，听到用户不开心/担心的事就换成" +
				"关心的表情，或者被逗到了就用惊讶表情)，不需要每句话都调用，" +
				"自然地用就行，就像人聊天时会有表情变化一样。",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"emotion": map[string]any{
						"type": "string",
						"enum": []string{
							actuator.EmotionSmiley,
							actuator.EmotionFrownie,
							actuator.EmotionSurprised,
							actuator.EmotionNeutral,
						},
						"description": "happy=开心笑脸 sad=关心/难过表情 surprised=惊讶 neutral=平静",
					},
				},
				"required": []string{"emotion"},
			},
			Handler: func(args map[string]any) (string, error) {

				if sm != nil {
					behavior := sm.BehaviorState()
					env := sm.EnvState()
					isAlert := env != "" || behavior == statemachine.StateFallAlert || behavior == statemachine.StateStaticAlert
					if isAlert {
						return "现在有安全告警，屏幕需要保持告警显示，不能切换表情", nil
					}
				}
				emotion := stringArg(args, "emotion")
				if err := act.ShowEmotion(emotion, 6); err != nil {
					return "", err
				}
				return "表情已切换", nil
			},
		},
		{
			Name: voiceagent.StopToolName,
			Description: "用户想结束/停止这次对话时调用(比如说“不聊了”“再见”“先这样吧”)。" +
				"调用后你还会有机会说最后一句话，请在那句话里自然地道别、" +
				"提一句“有需要随时叫我”，不要生硬地直接结束。不需要参数。",
			Handler: func(args map[string]any) (string, error) {

				return "对话即将结束，请用温暖自然的语气跟用户说一句告别语，" +
					"可以提一句有需要随时叫我，不要说“已停止”这种系统提示语气。", nil
			},
		},
	}
}

func stringArg(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return v
	}
	return ""
}

func queryWeather() string {
	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("https://wttr.in/?format=%C+%t")
	if err != nil {
		return fallbackWeatherReply()
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallbackWeatherReply()
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256))
	if err != nil || len(strings.TrimSpace(string(body))) == 0 {
		return fallbackWeatherReply()
	}

	return fmt.Sprintf("当前天气：%s", strings.TrimSpace(string(body)))
}

func fallbackWeatherReply() string {
	return "暂时无法连接天气服务，不过居家环境的温湿度可以通过传感器数据查看。"
}
