package voiceagent

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os/exec"
	"time"
)

const (
	sampleRate  = 16000
	frameMs     = 20
	frameBytes  = sampleRate * 2 * frameMs / 1000
	preRollFrms = 10
)

type VADResult struct {
	PCM      []byte
	Started  bool
	Duration float64
	RMS      int
	Reason   string
}

func RecordUntilSilence(device string, maxSeconds, startTimeoutSec float64, silenceMs int, energyThreshold int) (VADResult, error) {
	cmd := exec.Command("arecord",
		"-D", device, "-q",
		"-f", "S16_LE", "-r", fmt.Sprintf("%d", sampleRate), "-c", "1",
		"-t", "raw",
	)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return VADResult{}, fmt.Errorf("创建arecord输出管道失败: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return VADResult{}, fmt.Errorf("启动arecord失败: %w", err)
	}
	defer func() {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
	}()

	var frames [][]byte
	var ring [][]byte
	speechStarted := false
	silenceFrames := 0
	requiredSilenceFrames := silenceMs / frameMs
	if requiredSilenceFrames < 1 {
		requiredSilenceFrames = 1
	}

	startTime := time.Now()
	lastReason := "timeout"

	buf := make([]byte, frameBytes)
	for {
		elapsed := time.Since(startTime).Seconds()
		if elapsed >= maxSeconds {
			lastReason = "max_seconds"
			break
		}
		if !speechStarted && elapsed >= startTimeoutSec {
			lastReason = "start_timeout"
			break
		}

		n, readErr := readFull(stdout, buf)
		if readErr != nil || n < frameBytes {
			lastReason = "audio_end"
			break
		}
		frame := make([]byte, frameBytes)
		copy(frame, buf)

		isSpeech := isSpeechEnergy(frame, energyThreshold)

		if !isSpeech && isSpeechEnergy(frame, maxInt(energyThreshold*2, 80)) {
			isSpeech = true
		}

		if !speechStarted {
			ring = append(ring, frame)
			if len(ring) > preRollFrms {
				ring = ring[1:]
			}
			if isSpeech {
				speechStarted = true
				frames = append(frames, ring...)
				ring = nil
				silenceFrames = 0
			}
		} else {
			frames = append(frames, frame)
			if isSpeech {
				silenceFrames = 0
			} else {
				silenceFrames++
				if silenceFrames >= requiredSilenceFrames {
					lastReason = "silence"
					break
				}
			}
		}
	}

	pcm := []byte{}
	if speechStarted {
		pcm = joinBytes(frames)
	}
	rms, _ := pcmLevel(pcm)

	return VADResult{
		PCM:      pcm,
		Started:  speechStarted,
		Duration: float64(len(pcm)) / float64(sampleRate*2),
		RMS:      rms,
		Reason:   lastReason + "/energy",
	}, nil
}

func readFull(r interface{ Read([]byte) (int, error) }, buf []byte) (int, error) {
	total := 0
	for total < len(buf) {
		n, err := r.Read(buf[total:])
		total += n
		if err != nil {
			return total, err
		}
		if n == 0 {
			break
		}
	}
	if total < len(buf) {
		return total, fmt.Errorf("读取不足: 期望%d字节，实际%d字节", len(buf), total)
	}
	return total, nil
}

func pcmLevel(pcm []byte) (rms int, maxV int) {
	if len(pcm) == 0 {
		return 0, 0
	}
	n := len(pcm) / 2
	if n == 0 {
		return 0, 0
	}
	var sumSquares float64
	for i := 0; i < n; i++ {
		sample := int16(binary.LittleEndian.Uint16(pcm[i*2 : i*2+2]))
		v := float64(sample)
		sumSquares += v * v
		absV := int(sample)
		if absV < 0 {
			absV = -absV
		}
		if absV > maxV {
			maxV = absV
		}
	}
	rms = int(math.Sqrt(sumSquares / float64(n)))
	return rms, maxV
}

func isSpeechEnergy(frame []byte, threshold int) bool {
	rms, _ := pcmLevel(frame)
	return rms >= threshold
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func AmplifyPCM(audio []byte, gain float64) []byte {
	if gain <= 1.0 {
		return audio
	}
	out := make([]byte, len(audio))
	n := len(audio) / 2
	for i := 0; i < n; i++ {
		sample := int16(binary.LittleEndian.Uint16(audio[i*2 : i*2+2]))
		scaled := float64(sample) * gain
		if scaled > 32767 {
			scaled = 32767
		} else if scaled < -32768 {
			scaled = -32768
		}
		binary.LittleEndian.PutUint16(out[i*2:i*2+2], uint16(int16(scaled)))
	}
	return out
}

func PlayPCM(audio []byte, device string, gain float64, sampleRateHz int) error {
	if len(audio) == 0 {
		return nil
	}
	audio = AmplifyPCM(audio, gain)

	cmd := exec.Command("aplay",
		"-D", device, "-q",
		"-f", "S16_LE", "-r", fmt.Sprintf("%d", sampleRateHz), "-c", "1",
		"-t", "raw",
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("创建aplay输入管道失败: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动aplay失败: %w", err)
	}

	if _, err := bytes.NewReader(audio).WriteTo(stdin); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("写入音频数据失败: %w", err)
	}
	_ = stdin.Close()

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()
	select {
	case err := <-done:
		return err
	case <-time.After(20 * time.Second):
		_ = cmd.Process.Kill()
		return fmt.Errorf("aplay 播放超时")
	}
}

func joinBytes(chunks [][]byte) []byte {
	total := 0
	for _, c := range chunks {
		total += len(c)
	}
	out := make([]byte, 0, total)
	for _, c := range chunks {
		out = append(out, c...)
	}
	return out
}
