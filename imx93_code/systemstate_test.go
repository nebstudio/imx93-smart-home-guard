package main

import (
	"sync"
	"testing"
)

func TestNewSystemState_DefaultsSystemEnabledTrue(t *testing.T) {
	s := newSystemState(true, false)
	if !s.SystemEnabled() {
		t.Error("系统总开关默认应为开启状态(与调整前始终工作的行为一致)")
	}
}

func TestNewSystemState_VoiceDefaultRespectsStartupFlagAndConfig(t *testing.T) {
	cases := []struct {
		name        string
		configured  bool
		startupFlag bool
		want        bool
	}{
		{"未配置+未启用flag", false, false, false},
		{"未配置+启用flag", false, true, false},
		{"已配置+未启用flag", true, false, false},
		{"已配置+启用flag", true, true, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := newSystemState(c.configured, c.startupFlag)
			if got := s.VoiceEnabled(); got != c.want {
				t.Errorf("期望VoiceEnabled()=%v，实际=%v", c.want, got)
			}
		})
	}
}

func TestSetVoiceEnabled_RejectsEnableWhenNotConfigured(t *testing.T) {
	s := newSystemState(false, false)
	ok := s.SetVoiceEnabled(true)
	if ok {
		t.Error("语音配置未加载时，打开语音开关应该被拒绝")
	}
	if s.VoiceEnabled() {
		t.Error("被拒绝的打开请求不应该真正生效")
	}
}

func TestSetVoiceEnabled_DisableAlwaysSucceeds(t *testing.T) {
	s := newSystemState(false, false)
	ok := s.SetVoiceEnabled(false)
	if !ok {
		t.Error("关闭语音开关不需要配置校验，应该总是成功")
	}
}

func TestSetVoiceEnabled_AcceptsEnableWhenConfigured(t *testing.T) {
	s := newSystemState(true, false)
	ok := s.SetVoiceEnabled(true)
	if !ok {
		t.Fatal("语音配置已加载时，打开语音开关应该成功")
	}
	if !s.VoiceEnabled() {
		t.Error("成功的打开请求应该真正生效")
	}
}

func TestSetSystemEnabled_TogglesCorrectly(t *testing.T) {
	s := newSystemState(false, false)
	s.SetSystemEnabled(false)
	if s.SystemEnabled() {
		t.Error("SetSystemEnabled(false)后应为关闭状态")
	}
	s.SetSystemEnabled(true)
	if !s.SystemEnabled() {
		t.Error("SetSystemEnabled(true)后应为开启状态")
	}
}

func TestSystemState_ConcurrentAccess(t *testing.T) {
	s := newSystemState(true, false)
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		go func(n int) {
			defer wg.Done()
			s.SetSystemEnabled(n%2 == 0)
		}(i)
		go func() {
			defer wg.Done()
			_ = s.SystemEnabled()
			_ = s.VoiceEnabled()
		}()
	}
	wg.Wait()
}
