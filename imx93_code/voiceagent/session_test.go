package voiceagent

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

type fakeQwenServer struct {
	t      *testing.T
	server *httptest.Server
	script func(clientEvent map[string]any) []map[string]any
}

func newFakeQwenServer(t *testing.T, script func(map[string]any) []map[string]any) *fakeQwenServer {
	upgrader := websocket.Upgrader{}
	f := &fakeQwenServer{t: t, script: script}

	f.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("测试服务端升级WebSocket失败: %v", err)
			return
		}
		defer conn.Close()

		for {
			_, data, err := conn.ReadMessage()
			if err != nil {
				return
			}
			var ev map[string]any
			if err := json.Unmarshal(data, &ev); err != nil {
				t.Errorf("测试服务端解析客户端事件失败: %v", err)
				return
			}
			for _, reply := range f.script(ev) {
				out, _ := json.Marshal(reply)
				if err := conn.WriteMessage(websocket.TextMessage, out); err != nil {
					return
				}
			}
		}
	}))
	return f
}

func (f *fakeQwenServer) wsURL() string {
	return "ws" + strings.TrimPrefix(f.server.URL, "http")
}

func (f *fakeQwenServer) close() {
	f.server.Close()
}

func testConfig(wsURL string) Config {
	return Config{
		WSURL:            wsURL,
		APIKey:           "test-key",
		Model:            "qwen3.5-omni-plus-realtime",
		Voice:            "Ethan",
		RecvTimeout:      3 * time.Second,
		InputSampleRate:  16000,
		OutputSampleRate: 24000,
	}
}

func TestUpdateSession_Accepted(t *testing.T) {
	fake := newFakeQwenServer(t, func(ev map[string]any) []map[string]any {
		if ev["type"] == "session.update" {
			return []map[string]any{{"type": "session.updated"}}
		}
		return nil
	})
	defer fake.close()

	c, err := dial(testConfig(fake.wsURL()))
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer c.close()

	if err := c.updateSession("你是测试助手", nil, true); err != nil {
		t.Fatalf("期望 session.update 成功，实际报错: %v", err)
	}
}

func TestUpdateSession_Rejected(t *testing.T) {
	fake := newFakeQwenServer(t, func(ev map[string]any) []map[string]any {
		if ev["type"] == "session.update" {
			return []map[string]any{{
				"type": "error",
				"error": map[string]any{
					"code":    "COMMON_ERROR",
					"message": "Voice 'Bogus' is not supported.",
				},
			}}
		}
		return nil
	})
	defer fake.close()

	c, err := dial(testConfig(fake.wsURL()))
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer c.close()

	err = c.updateSession("你是测试助手", nil, true)
	if err == nil {
		t.Fatal("期望 session.update 被拒绝时返回错误，实际没有")
	}
	if !strings.Contains(err.Error(), "COMMON_ERROR") {
		t.Errorf("错误信息应包含错误码，实际: %v", err)
	}
}

func TestCollectTurn_PlainTextResponse(t *testing.T) {
	audioB64 := base64.StdEncoding.EncodeToString([]byte{1, 2, 3, 4})
	fake := newFakeQwenServer(t, func(ev map[string]any) []map[string]any {
		if ev["type"] != "response.create" {
			return nil
		}
		return []map[string]any{
			{"type": "response.audio_transcript.delta", "delta": "你好"},
			{"type": "response.audio.delta", "delta": audioB64},
			{"type": "response.audio_transcript.done", "transcript": "你好，有什么可以帮您？"},
			{"type": "response.done"},
		}
	})
	defer fake.close()

	c, err := dial(testConfig(fake.wsURL()))
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer c.close()

	if err := c.createResponse(); err != nil {
		t.Fatalf("发送response.create失败: %v", err)
	}

	res, err := c.collectTurn(nil)
	if err != nil {
		t.Fatalf("collectTurn失败: %v", err)
	}
	if res.AssistantText != "你好，有什么可以帮您？" {
		t.Errorf("期望完整文本为最终transcript，实际: %q", res.AssistantText)
	}
	if string(res.audio) != string([]byte{1, 2, 3, 4}) {
		t.Errorf("音频数据不匹配: %v", res.audio)
	}
	if len(res.InvokedTools) != 0 {
		t.Errorf("普通响应不应该有工具调用，实际: %v", res.InvokedTools)
	}
}

func TestCollectTurn_ToolCall_ExecutesHandlerAndReturnsResult(t *testing.T) {
	handlerCalled := false
	var receivedArgs map[string]any

	tools := []Tool{
		{
			Name: "control_fan",
			Handler: func(args map[string]any) (string, error) {
				handlerCalled = true
				receivedArgs = args
				return "风扇已打开", nil
			},
		},
	}

	round := 0
	fake := newFakeQwenServer(t, func(ev map[string]any) []map[string]any {
		switch ev["type"] {
		case "response.create":
			round++
			if round == 1 {

				return []map[string]any{
					{
						"type":    "response.function_call_arguments.done",
						"name":    "control_fan",
						"call_id": "call_1",
						"arguments": `{"state":"on"}`,
					},
					{"type": "response.done"},
				}
			}

			return []map[string]any{
				{"type": "response.audio_transcript.done", "transcript": "好的，风扇已经为您打开。"},
				{"type": "response.done"},
			}
		case "conversation.item.create":

			item, _ := ev["item"].(map[string]any)
			if item["type"] != "function_call_output" {
				t.Errorf("期望回传function_call_output，实际: %v", item["type"])
			}
			if item["output"] != "风扇已打开" {
				t.Errorf("期望回传Handler的真实返回值，实际: %v", item["output"])
			}
		}
		return nil
	})
	defer fake.close()

	c, err := dial(testConfig(fake.wsURL()))
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer c.close()

	if err := c.createResponse(); err != nil {
		t.Fatalf("发送response.create失败: %v", err)
	}

	res, err := c.collectTurn(tools)
	if err != nil {
		t.Fatalf("collectTurn失败: %v", err)
	}

	if !handlerCalled {
		t.Fatal("期望本地Handler被真实调用，实际没有")
	}
	if receivedArgs["state"] != "on" {
		t.Errorf("期望Handler收到解析后的参数state=on，实际: %v", receivedArgs)
	}
	if len(res.InvokedTools) != 1 || res.InvokedTools[0] != "control_fan" {
		t.Errorf("期望InvokedTools记录control_fan，实际: %v", res.InvokedTools)
	}
	if res.AssistantText != "好的，风扇已经为您打开。" {
		t.Errorf("期望最终回复是工具执行后的那一轮文本，实际: %q", res.AssistantText)
	}
}

func TestCollectTurn_UnknownTool_DoesNotCrash(t *testing.T) {
	round := 0
	fake := newFakeQwenServer(t, func(ev map[string]any) []map[string]any {
		if ev["type"] == "response.create" {
			round++
			if round == 1 {
				return []map[string]any{
					{
						"type":      "response.function_call_arguments.done",
						"name":      "unknown_tool_xyz",
						"call_id":   "call_1",
						"arguments": `{}`,
					},
					{"type": "response.done"},
				}
			}
			return []map[string]any{
				{"type": "response.audio_transcript.done", "transcript": "这个功能暂时还做不到。"},
				{"type": "response.done"},
			}
		}
		return nil
	})
	defer fake.close()

	c, err := dial(testConfig(fake.wsURL()))
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer c.close()

	if err := c.createResponse(); err != nil {
		t.Fatalf("发送response.create失败: %v", err)
	}

	res, err := c.collectTurn(nil)
	if err != nil {
		t.Fatalf("即使工具未知，collectTurn也不应该报错，实际: %v", err)
	}
	if res.AssistantText != "这个功能暂时还做不到。" {
		t.Errorf("期望模型能继续生成回复，实际: %q", res.AssistantText)
	}
}

func TestCollectTurn_ServerError_PropagatesAsError(t *testing.T) {
	fake := newFakeQwenServer(t, func(ev map[string]any) []map[string]any {
		if ev["type"] == "response.create" {
			return []map[string]any{
				{"type": "error", "error": map[string]any{"code": "RATE_LIMIT", "message": "太快了"}},
			}
		}
		return nil
	})
	defer fake.close()

	c, err := dial(testConfig(fake.wsURL()))
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer c.close()

	if err := c.createResponse(); err != nil {
		t.Fatalf("发送response.create失败: %v", err)
	}

	_, err = c.collectTurn(nil)
	if err == nil {
		t.Fatal("期望服务端错误被传播为error，实际没有")
	}
	if !strings.Contains(err.Error(), "RATE_LIMIT") {
		t.Errorf("错误信息应包含服务端错误码，实际: %v", err)
	}
}
