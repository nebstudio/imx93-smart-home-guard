package voiceagent

import (
	"fmt"
)

const systemInstructions = "你是小智，一个陪伴在老人身边的智能家居助手，性格温暖、耐心、" +
	"有点像家里贴心的孙辈。你可以调用工具真实控制家里的设备(风扇、指示灯)、" +
	"查询天气，或者在用户明确表达想结束对话时调用停止工具。\n" +
	"说话风格：像日常聊天一样自然、口语化，不要用书面语或客服腔；" +
	"每次回答控制在1到2句话，简短但不生硬；可以适当带点情绪(开心、关心、" +
	"俏皮)，但不要过度夸张；不要机械地复述用户说的话，也不要用" +
	"“已收到”“指令已执行”这类系统提示语气。\n" +
	"如果对话与跌倒/长时间静止告警相关，先用一句话简短安抚情绪，" +
	"再给出实用的安全建议；不要确诊、不要开药、不要给专业医疗意见。\n" +
	"如果用户表达了结束对话的意思(比如“不聊了”“先这样吧”“再见”)，" +
	"调用停止工具，并在这一轮里先自然地说一句暖心的告别语再结束，" +
	"比如提一句“有需要随时叫我”，不要生硬地直接沉默。\n" +
	"你的回答会被语音朗读出来，所以只能用完全口语化的表达：不要用" +
	"“爷爷/奶奶”这种带斜杠的书面写法去同时指代多种称呼，也不要用" +
	"括号、星号等任何朗读不出来的符号；不确定对方称呼时，直接说" +
	"“您”，不要罗列多个称呼。"

type AskConfirmResult struct {
	Responded bool
	ASRText   string
	ChatText  string
	Reason    string
}

func Say(cfg Config, text string) error {
	c, err := dial(cfg)
	if err != nil {
		return err
	}
	defer c.close()

	if err := c.updateSession(systemInstructions, nil, true); err != nil {
		return fmt.Errorf("开启会话失败: %w", err)
	}
	if err := c.sendUserText(fmt.Sprintf("请直接播报这句话，不要添加其它内容：%s", text)); err != nil {
		return fmt.Errorf("发送播报文本失败: %w", err)
	}
	if err := c.createResponse(); err != nil {
		return fmt.Errorf("触发播报响应失败: %w", err)
	}

	res, err := c.collectTurn(nil)
	if err != nil {
		return fmt.Errorf("播报失败: %w", err)
	}
	return PlayPCM(res.audio, cfg.AudioPlaybackDevice, cfg.AudioPlaybackGain, cfg.OutputSampleRate)
}

func AskConfirm(cfg Config, prompt string, listenSeconds float64) (AskConfirmResult, error) {
	c, err := dial(cfg)
	if err != nil {
		return AskConfirmResult{}, err
	}
	defer c.close()

	if err := c.updateSession(systemInstructions, nil, true); err != nil {
		return AskConfirmResult{}, fmt.Errorf("开启会话失败: %w", err)
	}

	if err := c.sendUserText(fmt.Sprintf("请直接播报这句话，不要添加其它内容：%s", prompt)); err != nil {
		return AskConfirmResult{}, fmt.Errorf("发送确认问题失败: %w", err)
	}
	if err := c.createResponse(); err != nil {
		return AskConfirmResult{}, fmt.Errorf("触发确认问题响应失败: %w", err)
	}
	promptRes, err := c.collectTurn(nil)
	if err != nil {
		return AskConfirmResult{}, fmt.Errorf("播报确认问题失败: %w", err)
	}
	if err := PlayPCM(promptRes.audio, cfg.AudioPlaybackDevice, cfg.AudioPlaybackGain, cfg.OutputSampleRate); err != nil {
		return AskConfirmResult{}, fmt.Errorf("播放确认问题失败: %w", err)
	}

	vad, err := RecordUntilSilence(cfg.AudioCaptureDevice, listenSeconds+2, listenSeconds, 800, 120)
	if err != nil {
		return AskConfirmResult{}, fmt.Errorf("监听回应失败: %w", err)
	}
	if !vad.Started {
		return AskConfirmResult{Responded: false, Reason: vad.Reason}, nil
	}

	if err := c.appendAudio(vad.PCM); err != nil {
		return AskConfirmResult{}, fmt.Errorf("上传回应音频失败: %w", err)
	}
	if err := c.commitAudio(); err != nil {
		return AskConfirmResult{}, fmt.Errorf("提交回应音频失败: %w", err)
	}
	if err := c.createResponse(); err != nil {
		return AskConfirmResult{}, fmt.Errorf("触发回应响应失败: %w", err)
	}

	chatRes, err := c.collectTurn(nil)
	if err != nil {
		return AskConfirmResult{}, fmt.Errorf("获取回应处理结果失败: %w", err)
	}
	if err := PlayPCM(chatRes.audio, cfg.AudioPlaybackDevice, cfg.AudioPlaybackGain, cfg.OutputSampleRate); err != nil {

		return AskConfirmResult{
			Responded: true,
			ASRText:   chatRes.UserText,
			ChatText:  chatRes.AssistantText,
			Reason:    "vad_speech",
		}, fmt.Errorf("播放回复音频失败(不影响回应判定): %w", err)
	}

	return AskConfirmResult{
		Responded: true,
		ASRText:   chatRes.UserText,
		ChatText:  chatRes.AssistantText,
		Reason:    "vad_speech",
	}, nil
}
