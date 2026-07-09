package voiceagent

import (
	"encoding/binary"
	"testing"
)

func makeSinePCM(amplitude int16, numSamples int) []byte {
	buf := make([]byte, numSamples*2)
	for i := 0; i < numSamples; i++ {
		v := amplitude
		if i%2 == 1 {
			v = -amplitude
		}
		binary.LittleEndian.PutUint16(buf[i*2:i*2+2], uint16(v))
	}
	return buf
}

func TestPcmLevel_Silence(t *testing.T) {
	silence := make([]byte, 640)
	rms, maxV := pcmLevel(silence)
	if rms != 0 || maxV != 0 {
		t.Errorf("静音数据应该RMS=0 maxV=0，实际 rms=%d maxV=%d", rms, maxV)
	}
}

func TestPcmLevel_KnownAmplitude(t *testing.T) {

	pcm := makeSinePCM(1000, 100)
	rms, maxV := pcmLevel(pcm)
	if rms != 1000 {
		t.Errorf("期望RMS=1000，实际=%d", rms)
	}
	if maxV != 1000 {
		t.Errorf("期望maxV=1000，实际=%d", maxV)
	}
}

func TestIsSpeechEnergy(t *testing.T) {
	loud := makeSinePCM(500, 100)
	quiet := makeSinePCM(10, 100)

	if !isSpeechEnergy(loud, 100) {
		t.Error("响亮的音频应该被判定为语音")
	}
	if isSpeechEnergy(quiet, 100) {
		t.Error("安静的音频不应该被判定为语音")
	}
}

func TestAmplifyPCM_GainOne_NoChange(t *testing.T) {
	pcm := makeSinePCM(1000, 10)
	out := AmplifyPCM(pcm, 1.0)
	if string(out) != string(pcm) {
		t.Error("gain=1.0时应该原样返回")
	}
}

func TestAmplifyPCM_DoublesAmplitude(t *testing.T) {
	pcm := makeSinePCM(1000, 10)
	out := AmplifyPCM(pcm, 2.0)
	rms, _ := pcmLevel(out)
	if rms != 2000 {
		t.Errorf("放大2倍后期望RMS=2000，实际=%d", rms)
	}
}

func TestAmplifyPCM_ClipsAtMax(t *testing.T) {

	pcm := makeSinePCM(20000, 10)
	out := AmplifyPCM(pcm, 2.0)

	n := len(out) / 2
	for i := 0; i < n; i++ {
		sample := int16(binary.LittleEndian.Uint16(out[i*2 : i*2+2]))
		if i%2 == 0 {
			if sample != 32767 {
				t.Errorf("正样本期望被削波到32767，实际=%d", sample)
			}
		} else {
			if sample != -32768 {
				t.Errorf("负样本期望被削波到-32768，实际=%d", sample)
			}
		}
	}
}
