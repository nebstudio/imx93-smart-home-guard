package statemachine

import (
	"testing"
	"time"
)

func poseSnap(t time.Time, person bool, posture string) SensorSnapshot {
	return SensorSnapshot{
		Time:          t,
		FlameADC:      1023,
		PoseAvailable: true,
		PosePerson:    person,
		PosePosture:   posture,
	}
}

func feedPose(m *Machine, base time.Time, startOffset, interval time.Duration, person bool, posture string, count int) (State, State) {
	var behavior, env State
	for i := 0; i < count; i++ {
		behavior, env = m.Update(poseSnap(base.Add(startOffset+time.Duration(i)*interval), person, posture))
	}
	return behavior, env
}

func newMonitoringMachine(cfg Config, base time.Time) *Machine {
	m := New(cfg)
	m.Update(poseSnap(base, true, PostureStanding))
	return m
}

func TestNormalToMonitoring_WhenPersonDetected(t *testing.T) {
	base := time.Now()
	m := New(DefaultConfig())
	behavior, _ := m.Update(poseSnap(base, true, PostureStanding))
	if behavior != StateMonitoring {
		t.Fatalf("检测到人应进入 MONITORING，实际: %s", behavior)
	}
}

func TestFallAlert_LyingConfirmedOverFrames_Triggers(t *testing.T) {
	cfg := DefaultConfig()
	base := time.Now()
	m := newMonitoringMachine(cfg, base)

	behavior, _ := feedPose(m, base, 200*time.Millisecond, 200*time.Millisecond, true, PostureLying, cfg.FallConfirmReadings)
	if behavior != StateFallAlert {
		t.Fatalf("连续多帧倒地应触发 FALL_ALERT，实际: %s", behavior)
	}
}

func TestFallAlert_SingleFrameLying_DoesNotTrigger(t *testing.T) {
	cfg := DefaultConfig()
	base := time.Now()
	m := newMonitoringMachine(cfg, base)

	behavior, _ := m.Update(poseSnap(base.Add(200*time.Millisecond), true, PostureLying))
	if behavior == StateFallAlert {
		t.Fatalf("单帧倒地不应触发 FALL_ALERT(需连续多帧确认)，实际已误报")
	}
	if behavior != StateMonitoring {
		t.Fatalf("期望仍在 MONITORING，实际: %s", behavior)
	}
}

func TestFallAlert_LyingInterruptedByStanding_ResetsConfirm(t *testing.T) {
	cfg := DefaultConfig()
	base := time.Now()
	m := newMonitoringMachine(cfg, base)

	m.Update(poseSnap(base.Add(200*time.Millisecond), true, PostureLying))
	m.Update(poseSnap(base.Add(400*time.Millisecond), true, PostureStanding))
	behavior, _ := m.Update(poseSnap(base.Add(600*time.Millisecond), true, PostureLying))
	if behavior == StateFallAlert {
		t.Fatalf("倒地确认被站立打断后不应立即触发 FALL_ALERT")
	}
}

func TestStaticAlert_PostureUnchangedTooLong_Triggers(t *testing.T) {
	cfg := DefaultConfig()
	cfg.StaticAlertAfter = 1 * time.Second
	base := time.Now()
	m := newMonitoringMachine(cfg, base)

	m.Update(poseSnap(base.Add(200*time.Millisecond), true, PostureSitting))
	behavior, _ := m.Update(poseSnap(base.Add(1500*time.Millisecond), true, PostureSitting))
	if behavior != StateStaticAlert {
		t.Fatalf("姿态长时间不变应触发 STATIC_ALERT，实际: %s", behavior)
	}
}

func TestStaticAlert_PostureChanges_ResetsTimer(t *testing.T) {
	cfg := DefaultConfig()
	cfg.StaticAlertAfter = 1 * time.Second
	base := time.Now()
	m := newMonitoringMachine(cfg, base)

	m.Update(poseSnap(base.Add(200*time.Millisecond), true, PostureSitting))
	m.Update(poseSnap(base.Add(900*time.Millisecond), true, PostureStanding))
	behavior, _ := m.Update(poseSnap(base.Add(1500*time.Millisecond), true, PostureStanding))
	if behavior == StateStaticAlert {
		t.Fatalf("姿态在超时窗口内变化过，不应触发 STATIC_ALERT")
	}
}

func TestMonitoring_PersonLeaves_ReturnsToNormal(t *testing.T) {
	cfg := DefaultConfig()
	base := time.Now()
	m := newMonitoringMachine(cfg, base)

	behavior, _ := feedPose(m, base, 200*time.Millisecond, 200*time.Millisecond, false, PostureNone, cfg.StaticPoseGraceReadings)
	if behavior != StateNormal {
		t.Fatalf("人离开后应退回 NORMAL，实际: %s", behavior)
	}
}

func TestMonitoring_BriefPersonMiss_DoesNotResetToNormal(t *testing.T) {
	cfg := DefaultConfig()
	base := time.Now()
	m := newMonitoringMachine(cfg, base)

	behavior, _ := m.Update(poseSnap(base.Add(200*time.Millisecond), false, PostureNone))
	if behavior != StateMonitoring {
		t.Fatalf("短暂一帧漏检不应退回 NORMAL，实际: %s", behavior)
	}
}

func TestPoseUnavailable_NoNewAlert(t *testing.T) {
	cfg := DefaultConfig()
	base := time.Now()
	m := newMonitoringMachine(cfg, base)

	for i := 0; i < 10; i++ {
		behavior, _ := m.Update(SensorSnapshot{
			Time:          base.Add(time.Duration(i+1) * 200 * time.Millisecond),
			FlameADC:      1023,
			PoseAvailable: false,
		})
		if behavior == StateFallAlert || behavior == StateStaticAlert {
			t.Fatalf("姿态不可用时不应产生行为告警，实际: %s", behavior)
		}
	}
}

func TestVoiceConfirm_CancelsFallAlert(t *testing.T) {
	cfg := DefaultConfig()
	base := time.Now()
	m := newMonitoringMachine(cfg, base)

	behavior, _ := feedPose(m, base, 200*time.Millisecond, 200*time.Millisecond, true, PostureLying, cfg.FallConfirmReadings)
	if behavior != StateFallAlert {
		t.Fatalf("前置条件应先进入 FALL_ALERT，实际: %s", behavior)
	}

	behavior, _ = m.Update(SensorSnapshot{
		Time:             base.Add(5 * time.Second),
		FlameADC:         1023,
		VoiceCancelAlert: true,
	})
	if behavior != StateNormal {
		t.Fatalf("语音确认用户无恙后期望回到 NORMAL，实际: %s", behavior)
	}
}

func TestVoiceConfirm_DoesNotCancelFireAlert(t *testing.T) {
	cfg := DefaultConfig()
	base := time.Now()
	m := New(cfg)

	_, env := m.Update(SensorSnapshot{
		Time: base, FlameADC: 10,
	})
	if env != StateFireAlert {
		t.Fatalf("前置条件应先进入FIRE_ALERT，实际: %s", env)
	}

	_, env = m.Update(SensorSnapshot{
		Time: base.Add(200 * time.Millisecond), FlameADC: 10,
		VoiceCancelAlert: true,
	})
	if env != StateFireAlert {
		t.Fatalf("语音确认不应能取消火警，期望仍为FIRE_ALERT，实际: %s", env)
	}
}

func TestApplyManualScenario_Fall(t *testing.T) {
	m := New(DefaultConfig())
	m.ApplyManualScenario(ManualScenarioFall, time.Now())
	if m.BehaviorState() != StateFallAlert {
		t.Fatalf("期望进入FALL_ALERT，实际: %s", m.BehaviorState())
	}
}

func TestApplyManualScenario_Static(t *testing.T) {
	m := New(DefaultConfig())
	m.ApplyManualScenario(ManualScenarioStatic, time.Now())
	if m.BehaviorState() != StateStaticAlert {
		t.Fatalf("期望进入STATIC_ALERT，实际: %s", m.BehaviorState())
	}
}

func TestApplyManualScenario_Clear(t *testing.T) {
	m := New(DefaultConfig())
	m.ApplyManualScenario(ManualScenarioFall, time.Now())
	m.ApplyManualScenario(ManualScenarioClear, time.Now())
	if m.BehaviorState() != StateNormal {
		t.Fatalf("期望回到NORMAL，实际: %s", m.BehaviorState())
	}
	if m.EnvState() != "" {
		t.Fatalf("期望环境告警也被清除，实际: %s", m.EnvState())
	}
}

func TestApplyManualScenario_FollowedByNormalTimeout(t *testing.T) {

	cfg := DefaultConfig()
	cfg.AlertTimeout = 1 * time.Second
	m := New(cfg)

	base := time.Now()
	m.ApplyManualScenario(ManualScenarioFall, base)

	behavior, _ := m.Update(SensorSnapshot{Time: base.Add(2 * time.Second), FlameADC: 1023})
	if behavior != StateNormal {
		t.Fatalf("手动触发的告警超时后应该自动清除，实际: %s", behavior)
	}
}

func TestUpdateConfig_AppliesOnlyProvidedFields(t *testing.T) {
	cfg := DefaultConfig()
	m := New(cfg)

	newSeconds := 10
	m.UpdateConfig(ConfigPatch{StaticAlertAfterSeconds: &newSeconds})

	got := m.ConfigSnapshot()
	if got.StaticAlertAfter != 10*time.Second {
		t.Errorf("期望StaticAlertAfter=10s，实际=%v", got.StaticAlertAfter)
	}
	if got.FireThreshold != cfg.FireThreshold {
		t.Errorf("未提供的字段FireThreshold应保持不变，期望=%d，实际=%d", cfg.FireThreshold, got.FireThreshold)
	}
}

func TestUpdateConfig_StaticAlertAfterSecondsConverted(t *testing.T) {
	m := New(DefaultConfig())
	seconds := 10
	m.UpdateConfig(ConfigPatch{StaticAlertAfterSeconds: &seconds})

	got := m.ConfigSnapshot().StaticAlertAfter
	if got != 10*time.Second {
		t.Errorf("期望StaticAlertAfter=10s，实际=%v", got)
	}
}
