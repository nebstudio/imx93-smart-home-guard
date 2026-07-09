package voiceagent

const StopToolName = "stop_conversation"

type ConverseResult struct {
	Heard         bool
	UserText      string
	AssistantText string
	StopRequested bool
	InvokedTools  []string
	Reason        string
}

func Converse(cfg Config, tools []Tool, listenSeconds, startTimeoutSeconds float64) (ConverseResult, error) {
	vad, err := RecordUntilSilence(cfg.AudioCaptureDevice, listenSeconds+2, startTimeoutSeconds, 800, 120)
	if err != nil {
		return ConverseResult{}, err
	}
	if !vad.Started {
		return ConverseResult{Heard: false, Reason: vad.Reason}, nil
	}

	c, err := dial(cfg)
	if err != nil {
		return ConverseResult{}, err
	}
	defer c.close()

	if err := c.updateSession(systemInstructions, tools, true); err != nil {
		return ConverseResult{}, err
	}
	if err := c.appendAudio(vad.PCM); err != nil {
		return ConverseResult{}, err
	}
	if err := c.commitAudio(); err != nil {
		return ConverseResult{}, err
	}
	if err := c.createResponse(); err != nil {
		return ConverseResult{}, err
	}

	res, err := c.collectTurn(tools)
	if err != nil {
		return ConverseResult{}, err
	}
	if err := PlayPCM(res.audio, cfg.AudioPlaybackDevice, cfg.AudioPlaybackGain, cfg.OutputSampleRate); err != nil {
		return ConverseResult{
			Heard:         true,
			UserText:      res.UserText,
			AssistantText: res.AssistantText,
			StopRequested: containsTool(res.InvokedTools, StopToolName),
			InvokedTools:  res.InvokedTools,
			Reason:        "vad_speech",
		}, err
	}

	return ConverseResult{
		Heard:         true,
		UserText:      res.UserText,
		AssistantText: res.AssistantText,
		StopRequested: containsTool(res.InvokedTools, StopToolName),
		InvokedTools:  res.InvokedTools,
		Reason:        "vad_speech",
	}, nil
}

func ChatText(cfg Config, tools []Tool, text string, speakAudio bool) (ConverseResult, error) {
	c, err := dial(cfg)
	if err != nil {
		return ConverseResult{}, err
	}
	defer c.close()

	if err := c.updateSession(systemInstructions, tools, speakAudio); err != nil {
		return ConverseResult{}, err
	}
	if err := c.sendUserText(text); err != nil {
		return ConverseResult{}, err
	}
	if err := c.createResponse(); err != nil {
		return ConverseResult{}, err
	}

	res, err := c.collectTurn(tools)
	if err != nil {
		return ConverseResult{}, err
	}

	if speakAudio {
		if err := PlayPCM(res.audio, cfg.AudioPlaybackDevice, cfg.AudioPlaybackGain, cfg.OutputSampleRate); err != nil {
			return ConverseResult{
				Heard:         true,
				UserText:      text,
				AssistantText: res.AssistantText,
				StopRequested: containsTool(res.InvokedTools, StopToolName),
				InvokedTools:  res.InvokedTools,
			}, err
		}
	}

	return ConverseResult{
		Heard:         true,
		UserText:      text,
		AssistantText: res.AssistantText,
		StopRequested: containsTool(res.InvokedTools, StopToolName),
		InvokedTools:  res.InvokedTools,
	}, nil
}

func containsTool(tools []string, name string) bool {
	for _, t := range tools {
		if t == name {
			return true
		}
	}
	return false
}
