package actuator

import (
	"testing"

	"imx93-guard/statemachine"
)

func TestLightColorFromBools(t *testing.T) {
	cases := []struct {
		red, yellow, green bool
		want               string
	}{
		{true, false, false, "red"},
		{false, true, false, "yellow"},
		{false, false, true, "green"},
		{false, false, false, "off"},

		{true, true, false, "red"},
		{true, false, true, "red"},
	}
	for _, c := range cases {
		got := lightColorFromBools(c.red, c.yellow, c.green)
		if got != c.want {
			t.Errorf("lightColorFromBools(%v,%v,%v) = %q, want %q",
				c.red, c.yellow, c.green, got, c.want)
		}
	}
}

func TestLogicalLightColor_StableAcrossBlinkCycle(t *testing.T) {
	cases := []struct {
		behavior statemachine.State
		env      statemachine.State
		want     string
	}{
		{statemachine.StateNormal, "", "green"},
		{statemachine.StateMonitoring, "", "yellow"},
		{statemachine.StateFallAlert, "", "red"},
		{statemachine.StateStaticAlert, "", "red"},
		{statemachine.StateMonitoring, statemachine.StateFireAlert, "red"},
		{statemachine.StateNormal, statemachine.StateSmokeAlert, "red"},
		{statemachine.StateNormal, statemachine.StateEmergency, "red"},
	}
	for _, c := range cases {

		for i := 0; i < 5; i++ {
			got := LogicalLightColor(c.behavior, c.env)
			if got != c.want {
				t.Errorf("LogicalLightColor(%s, %s) 第%d次调用 = %q, want %q",
					c.behavior, c.env, i, got, c.want)
			}
		}
	}
}
