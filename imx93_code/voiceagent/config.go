package voiceagent

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	WSURL  string
	APIKey string
	Model  string

	Voice string

	AudioCaptureDevice  string
	AudioPlaybackDevice string
	AudioPlaybackGain   float64

	RecvTimeout      time.Duration
	InputSampleRate  int
	OutputSampleRate int
}

func (c Config) Configured() bool {
	return c.APIKey != ""
}

func LoadConfigFromEnv() (Config, error) {
	cfg := Config{
		WSURL:               getEnvDefault("DASHSCOPE_WS_URL", "wss://dashscope.aliyuncs.com/api-ws/v1/realtime"),
		APIKey:              os.Getenv("DASHSCOPE_API_KEY"),
		Model:               getEnvDefault("DASHSCOPE_MODEL", "qwen3.5-omni-plus-realtime"),
		Voice:               getEnvDefault("DASHSCOPE_VOICE", "Tina"),
		AudioCaptureDevice:  getEnvDefault("AUDIO_CAPTURE_DEVICE", "plughw:0,0"),
		AudioPlaybackDevice: getEnvDefault("AUDIO_PLAYBACK_DEVICE", "plughw:1,0"),
		RecvTimeout:         5 * time.Second,
		InputSampleRate:     16000,
		OutputSampleRate:    24000,
	}

	gain := 2.0
	if v := os.Getenv("AUDIO_PLAYBACK_GAIN"); v != "" {
		if parsed, err := parseFloat(v); err == nil {
			gain = parsed
		}
	}
	cfg.AudioPlaybackGain = gain

	if cfg.APIKey == "" {
		return cfg, fmt.Errorf("缺少环境变量: DASHSCOPE_API_KEY")
	}
	return cfg, nil
}

func getEnvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
