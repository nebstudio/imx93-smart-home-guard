package main

import (
	"testing"
	"time"

	"imx93-guard/actuator"
	"imx93-guard/statemachine"
	"imx93-guard/voiceagent"
)

func TestBuildAgentTools_ContainsExpectedTools(t *testing.T) {
	tools := buildAgentTools(nil, nil)

	expected := []string{"control_fan", "control_light", "get_weather", "show_emotion", voiceagent.StopToolName}
	for _, name := range expected {
		found := false
		for _, tool := range tools {
			if tool.Name == name {
				found = true
				if tool.Description == "" {
					t.Errorf("工具 %s 的描述不能为空(模型依赖描述判断何时调用)", name)
				}
				if tool.Handler == nil {
					t.Errorf("工具 %s 缺少Handler", name)
				}
				break
			}
		}
		if !found {
			t.Errorf("期望工具清单包含 %s，实际未找到", name)
		}
	}
}

func TestBuildAgentTools_StopTool_NoParametersRequired(t *testing.T) {
	tools := buildAgentTools(nil, nil)
	for _, tool := range tools {
		if tool.Name != voiceagent.StopToolName {
			continue
		}
		result, err := tool.Handler(map[string]any{})
		if err != nil {
			t.Fatalf("停止对话工具应该总是能成功执行，实际报错: %v", err)
		}
		if result == "" {
			t.Error("停止对话工具应该返回非空的确认文案")
		}
		return
	}
	t.Fatal("未找到停止对话工具")
}

func TestControlFan_Tool_ParametersSchema(t *testing.T) {
	tools := buildAgentTools(nil, nil)
	for _, tool := range tools {
		if tool.Name != "control_fan" {
			continue
		}
		params := tool.Parameters
		if params == nil {
			t.Fatal("control_fan 应该声明 Parameters")
		}
		required, ok := params["required"].([]string)
		if !ok || len(required) == 0 || required[0] != "state" {
			t.Errorf("control_fan 应该要求必填参数 state，实际: %v", params["required"])
		}
		return
	}
	t.Fatal("未找到 control_fan 工具")
}

func TestStringArg_MissingKey_ReturnsEmpty(t *testing.T) {
	if got := stringArg(map[string]any{}, "state"); got != "" {
		t.Errorf("缺失的参数应该返回空字符串，实际: %q", got)
	}
}

func TestStringArg_WrongType_ReturnsEmpty(t *testing.T) {
	if got := stringArg(map[string]any{"state": 123}, "state"); got != "" {
		t.Errorf("类型不匹配时应该返回空字符串(不panic)，实际: %q", got)
	}
}

func TestShowEmotionTool_ParametersSchema(t *testing.T) {
	tools := buildAgentTools(nil, nil)
	for _, tool := range tools {
		if tool.Name != "show_emotion" {
			continue
		}
		params := tool.Parameters
		if params == nil {
			t.Fatal("show_emotion 应该声明 Parameters")
		}
		required, ok := params["required"].([]string)
		if !ok || len(required) == 0 || required[0] != "emotion" {
			t.Errorf("show_emotion 应该要求必填参数 emotion，实际: %v", params["required"])
		}
		return
	}
	t.Fatal("未找到 show_emotion 工具")
}

func TestShowEmotionTool_RejectedDuringRealAlert(t *testing.T) {
	sm := statemachine.New(statemachine.DefaultConfig())
	now := time.Now()

	sm.ApplyManualScenario(statemachine.ManualScenarioFall, now)
	if sm.BehaviorState() != statemachine.StateFallAlert {
		t.Fatalf("测试前置条件失败：期望进入FALL_ALERT，实际: %s", sm.BehaviorState())
	}

	tools := buildAgentTools(nil, sm)
	for _, tool := range tools {
		if tool.Name != "show_emotion" {
			continue
		}
		result, err := tool.Handler(map[string]any{"emotion": actuator.EmotionSmiley})
		if err != nil {
			t.Fatalf("即使拒绝也不应该返回error(应该用回复文案让模型知道原因)，实际: %v", err)
		}
		if result == "表情已切换" {
			t.Fatal("真实告警期间不应该允许表情切换成功，但看起来切换生效了")
		}
		return
	}
	t.Fatal("未找到 show_emotion 工具")
}

func TestQueryWeather_NeverPanicsAndReturnsNonEmpty(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("queryWeather不应该panic，实际: %v", r)
		}
	}()
	reply := queryWeather()
	if reply == "" {
		t.Error("queryWeather应该总是返回非空字符串(真实结果或兜底文案)")
	}
}
