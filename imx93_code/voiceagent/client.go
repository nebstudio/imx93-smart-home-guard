package voiceagent

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

type client struct {
	cfg  Config
	conn *websocket.Conn
}

type event struct {
	Type string
	raw  map[string]any
}

func (e event) str(key string) string {
	if v, ok := e.raw[key].(string); ok {
		return v
	}
	return ""
}

func dial(cfg Config) (*client, error) {
	url := cfg.WSURL + "?model=" + cfg.Model
	header := http.Header{}
	header.Set("Authorization", "Bearer "+cfg.APIKey)

	dialer := websocket.Dialer{HandshakeTimeout: 20 * time.Second}
	conn, resp, err := dialer.Dial(url, header)
	if err != nil {
		if resp != nil {
			return nil, fmt.Errorf("连接实时语音服务失败: %w (HTTP %d)", err, resp.StatusCode)
		}
		return nil, fmt.Errorf("连接实时语音服务失败: %w", err)
	}
	return &client{cfg: cfg, conn: conn}, nil
}

func (c *client) close() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
	}
}

func (c *client) sendEvent(obj map[string]any) error {
	if c.conn == nil {
		return fmt.Errorf("WebSocket 未连接")
	}
	if _, ok := obj["event_id"]; !ok {
		obj["event_id"] = "event_" + uuid.NewString()
	}
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("序列化客户端事件失败: %w", err)
	}
	return c.conn.WriteMessage(websocket.TextMessage, data)
}

func (c *client) recvEvent() (event, error) {
	if c.conn == nil {
		return event{}, fmt.Errorf("WebSocket 未连接")
	}
	_ = c.conn.SetReadDeadline(time.Now().Add(c.cfg.RecvTimeout))
	msgType, data, err := c.conn.ReadMessage()
	if err != nil {
		return event{}, err
	}
	if msgType != websocket.TextMessage {

		return event{Type: ""}, nil
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return event{}, fmt.Errorf("解析服务端事件失败: %w (原始: %s)", err, string(data))
	}
	t, _ := raw["type"].(string)
	return event{Type: t, raw: raw}, nil
}

func (c *client) updateSession(instructions string, tools []Tool, speakAudio bool) error {
	modalities := []string{"text", "audio"}
	if !speakAudio {
		modalities = []string{"text"}
	}
	session := map[string]any{
		"modalities":          modalities,
		"voice":               c.cfg.Voice,
		"input_audio_format":  "pcm",
		"output_audio_format": "pcm",
		"instructions":        instructions,
		"turn_detection":      nil,
	}
	if schema := toolSchema(tools); schema != nil {
		session["tools"] = schema
	}

	if err := c.sendEvent(map[string]any{
		"type":    "session.update",
		"session": session,
	}); err != nil {
		return err
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		ev, err := c.recvEvent()
		if err != nil {
			return fmt.Errorf("等待 session.updated 失败: %w", err)
		}
		switch ev.Type {
		case "session.updated", "session.created":

			if ev.Type == "session.updated" {
				return nil
			}
		case "error":
			return fmt.Errorf("session.update 被拒绝: %s", errorMessage(ev))
		}
	}
	return fmt.Errorf("等待 session.updated 超时")
}

func (c *client) appendAudio(pcm []byte) error {
	return c.sendEvent(map[string]any{
		"type":  "input_audio_buffer.append",
		"audio": base64.StdEncoding.EncodeToString(pcm),
	})
}

func (c *client) commitAudio() error {
	return c.sendEvent(map[string]any{"type": "input_audio_buffer.commit"})
}

func (c *client) sendUserText(text string) error {
	return c.sendEvent(map[string]any{
		"type": "conversation.item.create",
		"item": map[string]any{
			"type": "message",
			"role": "user",
			"content": []map[string]any{
				{"type": "input_text", "text": text},
			},
		},
	})
}

func (c *client) sendToolResult(callID, output string) error {
	return c.sendEvent(map[string]any{
		"type": "conversation.item.create",
		"item": map[string]any{
			"type":    "function_call_output",
			"call_id": callID,
			"output":  output,
		},
	})
}

func (c *client) createResponse() error {
	return c.sendEvent(map[string]any{"type": "response.create"})
}

func errorMessage(ev event) string {
	if errObj, ok := ev.raw["error"].(map[string]any); ok {
		msg, _ := errObj["message"].(string)
		code, _ := errObj["code"].(string)
		if code != "" {
			return fmt.Sprintf("%s (code=%s)", msg, code)
		}
		return msg
	}
	return "未知错误"
}
