package voiceagent

import (
	"encoding/base64"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

var debugFrames = os.Getenv("VOICEAGENT_DEBUG") == "1"

type turnResult struct {
	UserText      string
	AssistantText string
	audio         []byte
	InvokedTools  []string
}

const (
	turnReadTimeout    = 15 * time.Second
	turnOverallTimeout = 60 * time.Second
	maxToolRounds      = 4
)

type pendingCall struct {
	name   string
	callID string
	args   string
}

func (c *client) collectTurn(tools []Tool) (turnResult, error) {
	overallDeadline := time.Now().Add(turnOverallTimeout)

	var res turnResult
	var transcriptDelta strings.Builder
	var pending []pendingCall
	toolRounds := 0

	for {
		if time.Now().After(overallDeadline) {
			return res, fmt.Errorf("等待模型响应整体超时")
		}

		ev, err := c.recvEvent()
		if err != nil {
			return res, fmt.Errorf("读取服务端响应失败: %w", err)
		}

		if debugFrames {
			log.Printf("[voiceagent debug] <<< %s", ev.Type)
		}

		switch ev.Type {
		case "conversation.item.input_audio_transcription.completed":

			if t := ev.str("transcript"); t != "" {
				res.UserText = t
			}

		case "response.audio.delta":
			if d := ev.str("delta"); d != "" {
				if raw, err := base64.StdEncoding.DecodeString(d); err == nil {
					res.audio = append(res.audio, raw...)
				}
			}

		case "response.audio_transcript.delta", "response.text.delta":
			if d := ev.str("delta"); d != "" {
				transcriptDelta.WriteString(d)
			}

		case "response.audio_transcript.done", "response.text.done":

			if t := ev.str("transcript"); t != "" {
				res.AssistantText = t
			} else if t := ev.str("text"); t != "" {
				res.AssistantText = t
			}

		case "response.function_call_arguments.done":
			pending = append(pending, pendingCall{
				name:   ev.str("name"),
				callID: ev.str("call_id"),
				args:   ev.str("arguments"),
			})

		case "response.done":
			if len(pending) > 0 && toolRounds < maxToolRounds {
				toolRounds++
				if err := c.executeAndReturnTools(tools, pending, &res); err != nil {
					return res, err
				}
				pending = nil
				if err := c.createResponse(); err != nil {
					return res, fmt.Errorf("触发工具结果后的最终响应失败: %w", err)
				}
				continue
			}

			goto done

		case "error":
			return res, fmt.Errorf("实时语音服务端错误: %s", errorMessage(ev))
		}
	}

done:
	if res.AssistantText == "" {
		res.AssistantText = strings.TrimSpace(transcriptDelta.String())
	}
	return res, nil
}

func (c *client) executeAndReturnTools(tools []Tool, calls []pendingCall, res *turnResult) error {
	for _, call := range calls {
		res.InvokedTools = append(res.InvokedTools, call.name)

		tool, ok := findTool(tools, call.name)
		if !ok {
			log.Printf("模型请求了未知工具: %q，回传占位结果", call.name)
			if err := c.sendToolResult(call.callID, "该功能暂不可用"); err != nil {
				return fmt.Errorf("回传未知工具结果失败: %w", err)
			}
			continue
		}

		args := parseArgs(call.args)
		output, err := tool.Handler(args)
		if err != nil {
			log.Printf("工具 %q 执行出错(回传给模型): %v", call.name, err)
			output = fmt.Sprintf("执行失败: %v", err)
		}
		if output == "" {
			output = "已完成"
		}
		if err := c.sendToolResult(call.callID, output); err != nil {
			return fmt.Errorf("回传工具 %q 结果失败: %w", call.name, err)
		}
	}
	return nil
}
