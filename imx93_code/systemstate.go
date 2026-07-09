package main

import "sync/atomic"

type systemState struct {
	systemEnabled atomic.Bool
	voiceEnabled  atomic.Bool

	voiceConfigured bool
}

func newSystemState(voiceConfigured, voiceEnabledDefault bool) *systemState {
	s := &systemState{voiceConfigured: voiceConfigured}
	s.systemEnabled.Store(true)
	s.voiceEnabled.Store(voiceConfigured && voiceEnabledDefault)
	return s
}

func (s *systemState) SystemEnabled() bool { return s.systemEnabled.Load() }
func (s *systemState) VoiceEnabled() bool  { return s.voiceEnabled.Load() }
func (s *systemState) VoiceConfigured() bool { return s.voiceConfigured }

func (s *systemState) SetSystemEnabled(v bool) {
	s.systemEnabled.Store(v)
}

func (s *systemState) SetVoiceEnabled(v bool) bool {
	if v && !s.voiceConfigured {
		return false
	}
	s.voiceEnabled.Store(v)
	return true
}
