package statemachine

import "time"

type State string

const (
	StateNormal      State = "NORMAL"
	StateMonitoring  State = "MONITORING"
	StateFallAlert   State = "FALL_ALERT"
	StateStaticAlert State = "STATIC_ALERT"
	StateFireAlert   State = "FIRE_ALERT"
	StateSmokeAlert  State = "SMOKE_ALERT"
	StateEmergency   State = "EMERGENCY"
)

const (
	PostureNone     = "none"
	PostureStanding = "standing"
	PostureSitting  = "sitting"
	PostureLying    = "lying"
)

type Config struct {

	StaticAlertAfter time.Duration

	FireThreshold int

	SmokeThreshold int

	AlertTimeout time.Duration

	FallConfirmReadings int

	StaticPoseGraceReadings int
}

func DefaultConfig() Config {
	return Config{
		StaticAlertAfter:        30 * time.Second,
		FireThreshold:           200,
		SmokeThreshold:          600,
		AlertTimeout:            15 * time.Second,
		FallConfirmReadings:     3,
		StaticPoseGraceReadings: 5,
	}
}

type SensorSnapshot struct {
	Time          time.Time
	SmokeADC      int
	FlameADC      int
	TouchReleased bool

	PoseAvailable bool
	PosePerson    bool
	PosePosture   string

	VoiceCancelAlert bool
}

type ManualScenario string

const (
	ManualScenarioFall   ManualScenario = "fall"
	ManualScenarioStatic ManualScenario = "static"
	ManualScenarioClear  ManualScenario = "clear"
)

type Machine struct {
	cfg Config

	behaviorState State
	envState      State

	alertEnteredAt time.Time

	lyingConfirmCount int

	staticPosture  string
	staticSince    time.Time
	poseMissStreak int
}

func New(cfg Config) *Machine {
	return &Machine{
		cfg:           cfg,
		behaviorState: StateNormal,
	}
}

type ConfigPatch struct {
	StaticAlertAfterSeconds *int `json:"static_alert_after_seconds,omitempty"`
	FireThreshold           *int `json:"fire_threshold,omitempty"`
	SmokeThreshold          *int `json:"smoke_threshold,omitempty"`
}

func (m *Machine) ConfigSnapshot() Config {
	return m.cfg
}

func (m *Machine) UpdateConfig(patch ConfigPatch) {
	if patch.StaticAlertAfterSeconds != nil {
		m.cfg.StaticAlertAfter = time.Duration(*patch.StaticAlertAfterSeconds) * time.Second
	}
	if patch.FireThreshold != nil {
		m.cfg.FireThreshold = *patch.FireThreshold
	}
	if patch.SmokeThreshold != nil {
		m.cfg.SmokeThreshold = *patch.SmokeThreshold
	}
}

func (m *Machine) BehaviorState() State {
	return m.behaviorState
}

func (m *Machine) EnvState() State {
	return m.envState
}

func (m *Machine) Update(s SensorSnapshot) (State, State) {
	m.updateEnvState(s)
	m.updateBehaviorState(s)
	return m.behaviorState, m.envState
}

func (m *Machine) ApplyManualScenario(scenario ManualScenario, now time.Time) {
	switch scenario {
	case ManualScenarioFall:
		m.transitionTo(StateFallAlert, now)
	case ManualScenarioStatic:
		m.transitionTo(StateStaticAlert, now)
	case ManualScenarioClear:
		m.transitionTo(StateNormal, now)
		m.envState = ""
	}
}

func (m *Machine) updateEnvState(s SensorSnapshot) {

	fire := s.FlameADC < m.cfg.FireThreshold
	smoke := s.SmokeADC > m.cfg.SmokeThreshold

	switch {
	case fire && smoke:
		m.enterEnvState(StateEmergency, s.Time)
	case fire:
		m.enterEnvState(StateFireAlert, s.Time)
	case smoke:
		m.enterEnvState(StateSmokeAlert, s.Time)
	default:

		if m.envState != "" && m.shouldClearAlert(s) {
			m.envState = ""
		}
	}
}

func (m *Machine) enterEnvState(s State, now time.Time) {
	if m.envState != s {
		m.envState = s
		m.alertEnteredAt = now
	}
}

func (m *Machine) updateBehaviorState(s SensorSnapshot) {

	if m.behaviorState == StateFallAlert || m.behaviorState == StateStaticAlert {
		if m.shouldClearBehaviorAlert(s) {
			m.transitionTo(StateNormal, s.Time)
		}
		return
	}

	personPresent := s.PoseAvailable && s.PosePerson
	if personPresent {
		m.poseMissStreak = 0
	} else {
		if m.behaviorState == StateMonitoring {
			m.poseMissStreak++
			if m.poseMissStreak >= m.cfg.StaticPoseGraceReadings {
				m.transitionTo(StateNormal, s.Time)
			}
		}

		return
	}

	switch m.behaviorState {
	case StateNormal:
		m.transitionTo(StateMonitoring, s.Time)

	case StateMonitoring:

		if s.PosePosture == PostureLying {
			m.lyingConfirmCount++
			if m.lyingConfirmCount >= m.cfg.FallConfirmReadings {
				m.transitionTo(StateFallAlert, s.Time)
			}

		} else {
			m.lyingConfirmCount = 0
		}

		if m.staticPosture == s.PosePosture && m.staticPosture != "" {
			if m.staticSince.IsZero() {
				m.staticSince = s.Time
			} else if s.Time.Sub(m.staticSince) > m.cfg.StaticAlertAfter {
				m.transitionTo(StateStaticAlert, s.Time)
			}
		} else {
			m.staticPosture = s.PosePosture
			m.staticSince = s.Time
		}
	}
}

func (m *Machine) shouldClearAlert(s SensorSnapshot) bool {
	if s.TouchReleased {
		return true
	}
	return s.Time.Sub(m.alertEnteredAt) > m.cfg.AlertTimeout
}

func (m *Machine) shouldClearBehaviorAlert(s SensorSnapshot) bool {
	return s.VoiceCancelAlert || m.shouldClearAlert(s)
}

func (m *Machine) transitionTo(s State, now time.Time) {
	if m.behaviorState != s {
		m.behaviorState = s
		m.alertEnteredAt = now
		m.staticSince = time.Time{}
		m.staticPosture = ""
		m.lyingConfirmCount = 0
		m.poseMissStreak = 0
	}
}
