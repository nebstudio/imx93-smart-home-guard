package actuator

import (
	"fmt"
	"time"

	"imx93-guard/serialio"
	"imx93-guard/statemachine"
)

const (
	pinLightRed    = 7
	pinLightYellow = 8
	pinLightGreen  = 9
)

const (
	servoIndexWindow = 1
	servoIndexDoor   = 2
	servoIndexGarage = 3

	angleWindowClosed = 90
	angleWindowOpen   = 180

	angleDoorClosed = 90
	angleDoorOpen   = 180

	angleGarageClosed = 90
	angleGarageOpen   = 0
)

const (
	lcdEmojiSmiley    = 0
	lcdEmojiFrownie   = 1
	lcdEmojiSurprised = 2
	lcdEmojiNeutral   = 3
)

type Actuator struct {
	client *serialio.Client

	blinkState bool
	lastBlink  time.Time

	buzzOn        bool
	buzzPhaseFrom time.Time

	fanOn bool

	lastLightColor string

	manualLightOverride bool

	lastEmoji int

	lcdSurprisedUntil time.Time

	manualEmojiUntil time.Time

	windowOpen bool
	doorOpen   bool
	garageOpen bool
}

func (a *Actuator) LightColor() string {
	return a.lastLightColor
}

func LogicalLightColor(behavior, env statemachine.State) string {
	if env != "" {
		return "red"
	}
	switch behavior {
	case statemachine.StateNormal:
		return "green"
	case statemachine.StateMonitoring:
		return "yellow"
	case statemachine.StateFallAlert, statemachine.StateStaticAlert:
		return "red"
	default:
		return "off"
	}
}

func (a *Actuator) FanOn() bool {
	return a.fanOn
}

func (a *Actuator) HasManualLightOverride() bool {
	return a.manualLightOverride
}

func (a *Actuator) SetFan(on bool) error {
	speed := 0
	if on {
		speed = 255
	}
	if err := a.client.FanControl(0, speed); err != nil {
		return err
	}
	a.fanOn = on
	return nil
}

func (a *Actuator) SetLight(color string) error {
	red, yellow, green := false, false, false
	switch color {
	case "red":
		red = true
	case "yellow":
		yellow = true
	case "green":
		green = true
	case "off":

	default:
		return fmt.Errorf("未知灯光颜色: %s", color)
	}
	if err := a.setLights(red, yellow, green); err != nil {
		return err
	}
	a.manualLightOverride = true
	return nil
}

func New(client *serialio.Client) *Actuator {
	return &Actuator{client: client, lastEmoji: -1}
}

func (a *Actuator) WindowOpen() bool { return a.windowOpen }
func (a *Actuator) DoorOpen() bool   { return a.doorOpen }
func (a *Actuator) GarageOpen() bool { return a.garageOpen }

func (a *Actuator) SetWindow(open bool) error {
	angle := angleWindowClosed
	if open {
		angle = angleWindowOpen
	}
	if err := a.client.SetServoAngle(servoIndexWindow, angle); err != nil {
		return err
	}
	a.windowOpen = open
	return nil
}

func (a *Actuator) SetDoor(open bool) error {
	angle := angleDoorClosed
	if open {
		angle = angleDoorOpen
	}
	if err := a.client.SetServoAngle(servoIndexDoor, angle); err != nil {
		return err
	}
	a.doorOpen = open
	return nil
}

func (a *Actuator) SetGarage(open bool) error {
	angle := angleGarageClosed
	if open {
		angle = angleGarageOpen
	}
	if err := a.client.SetServoAngle(servoIndexGarage, angle); err != nil {
		return err
	}
	a.garageOpen = open
	return nil
}

func (a *Actuator) ApplyState(behavior, env statemachine.State, now time.Time) error {
	isAlertState := env != "" || behavior == statemachine.StateFallAlert || behavior == statemachine.StateStaticAlert

	if a.manualLightOverride {
		if isAlertState {

			a.manualLightOverride = false
		} else {

			return nil
		}
	}

	a.updateLcdEmoji(behavior, env, now)

	if env != "" {
		return a.applyEnvAlert(env, now)
	}
	return a.applyBehaviorState(behavior, now)
}

func (a *Actuator) updateLcdEmoji(behavior, env statemachine.State, now time.Time) {
	isAlert := env != "" || behavior == statemachine.StateFallAlert || behavior == statemachine.StateStaticAlert

	if isAlert {

		a.manualEmojiUntil = time.Time{}
	} else if now.Before(a.manualEmojiUntil) {

		return
	}

	var targetEmoji int
	if isAlert {
		if a.lcdSurprisedUntil.IsZero() {

			a.lcdSurprisedUntil = now.Add(3 * time.Second)
		}
		if now.Before(a.lcdSurprisedUntil) {
			targetEmoji = lcdEmojiSurprised
		} else {
			targetEmoji = lcdEmojiFrownie
		}
	} else {
		a.lcdSurprisedUntil = time.Time{}
		if behavior == statemachine.StateMonitoring {
			targetEmoji = lcdEmojiNeutral
		} else {
			targetEmoji = lcdEmojiSmiley
		}
	}

	if targetEmoji == a.lastEmoji {
		return
	}
	if err := a.client.ShowLcdEmoji(targetEmoji); err != nil {

		return
	}
	a.lastEmoji = targetEmoji
}

func (a *Actuator) applyBehaviorState(behavior statemachine.State, now time.Time) error {
	switch behavior {
	case statemachine.StateNormal:

		return a.setLights(false, false, true)

	case statemachine.StateMonitoring:

		blink := a.tickBlink(now, 500*time.Millisecond)
		return a.setLights(false, blink, false)

	case statemachine.StateFallAlert:

		blink := a.tickBlink(now, 300*time.Millisecond)
		if err := a.setLights(blink, false, false); err != nil {
			return err
		}
		return a.tickBuzzer(now, 2000, 300, 400)

	case statemachine.StateStaticAlert:

		blink := a.tickBlink(now, 800*time.Millisecond)
		if err := a.setLights(blink, false, false); err != nil {
			return err
		}
		return a.tickBuzzer(now, 1000, 150, 800)

	default:
		return fmt.Errorf("未知行为状态: %s", behavior)
	}
}

func (a *Actuator) applyEnvAlert(env statemachine.State, now time.Time) error {
	switch env {
	case statemachine.StateFireAlert, statemachine.StateSmokeAlert:

		blink := a.tickBlink(now, 300*time.Millisecond)
		if err := a.setLights(blink, false, false); err != nil {
			return err
		}
		return a.tickBuzzer(now, 2500, 250, 350)

	case statemachine.StateEmergency:

		blink := a.tickBlink(now, 150*time.Millisecond)
		if err := a.setLights(blink, false, false); err != nil {
			return err
		}
		if err := a.tickBuzzer(now, 3000, 400, 150); err != nil {
			return err
		}
		return a.SetFan(true)

	default:
		return fmt.Errorf("未知环境告警状态: %s", env)
	}
}

func (a *Actuator) tickBlink(now time.Time, period time.Duration) bool {
	if now.Sub(a.lastBlink) >= period {
		a.blinkState = !a.blinkState
		a.lastBlink = now
	}
	return a.blinkState
}

func (a *Actuator) tickBuzzer(now time.Time, freqHz, onMs, offMs int) error {
	phaseLen := time.Duration(onMs) * time.Millisecond
	if !a.buzzOn {
		phaseLen = time.Duration(offMs) * time.Millisecond
	}

	if a.buzzPhaseFrom.IsZero() || now.Sub(a.buzzPhaseFrom) >= phaseLen {
		a.buzzOn = !a.buzzOn
		a.buzzPhaseFrom = now
		if a.buzzOn {
			return a.client.Buzz(freqHz, onMs)
		}

	}
	return nil
}

func (a *Actuator) setLights(red, yellow, green bool) error {
	if err := a.client.WriteDigital(pinLightRed, red); err != nil {
		return err
	}
	if err := a.client.WriteDigital(pinLightYellow, yellow); err != nil {
		return err
	}
	if err := a.client.WriteDigital(pinLightGreen, green); err != nil {
		return err
	}
	a.lastLightColor = lightColorFromBools(red, yellow, green)
	return nil
}

func lightColorFromBools(red, yellow, green bool) string {
	switch {
	case red:
		return "red"
	case yellow:
		return "yellow"
	case green:
		return "green"
	default:
		return "off"
	}
}

func (a *Actuator) StopAlarm() error {
	if err := a.client.Buzz(0, 0); err != nil {
		return err
	}
	return a.SetFan(false)
}

const (
	EmotionSmiley    = "happy"
	EmotionFrownie   = "sad"
	EmotionSurprised = "surprised"
	EmotionNeutral   = "neutral"
)

func emotionNameToCode(name string) (int, bool) {
	switch name {
	case EmotionSmiley:
		return lcdEmojiSmiley, true
	case EmotionFrownie:
		return lcdEmojiFrownie, true
	case EmotionSurprised:
		return lcdEmojiSurprised, true
	case EmotionNeutral:
		return lcdEmojiNeutral, true
	default:
		return 0, false
	}
}

func (a *Actuator) ShowEmotion(emotionName string, durationSeconds int) error {
	code, ok := emotionNameToCode(emotionName)
	if !ok {
		return fmt.Errorf("未知表情名称: %s", emotionName)
	}
	if durationSeconds <= 0 {
		durationSeconds = 5
	}
	if err := a.client.ShowLcdEmoji(code); err != nil {
		return err
	}
	a.lastEmoji = code
	a.manualEmojiUntil = time.Now().Add(time.Duration(durationSeconds) * time.Second)
	return nil
}

func (a *Actuator) NotifyLocalConfirm() error {
	for i := 0; i < 2; i++ {
		if err := a.SetLight("green"); err != nil {
			return err
		}
		time.Sleep(150 * time.Millisecond)
		if err := a.SetLight("off"); err != nil {
			return err
		}
		time.Sleep(150 * time.Millisecond)
	}
	a.manualLightOverride = false
	return nil
}
