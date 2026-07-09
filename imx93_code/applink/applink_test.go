package applink

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

var testDialer = websocket.DefaultDialer

func newTestHTTPServer(t *testing.T, s *Server) *httptest.Server {
	mux := http.NewServeMux()
	s.RegisterHandler(mux)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func wsURL(httpURL string) string {
	return "ws" + httpURL[len("http"):] + "/ws/app"
}

func TestBroadcastStatus_DeliversToConnectedClient(t *testing.T) {
	s := New()
	srv := newTestHTTPServer(t, s)

	conn, _, err := testDialer.Dial(wsURL(srv.URL), nil)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	waitForClientCount(t, s, 1)

	s.BroadcastStatus(map[string]any{"behavior": "MONITORING"})

	var env Envelope
	if err := conn.ReadJSON(&env); err != nil {
		t.Fatalf("读取消息失败: %v", err)
	}
	if env.Type != "device_status" {
		t.Errorf("期望type=device_status，实际=%s", env.Type)
	}
	var data map[string]any
	_ = json.Unmarshal(env.Data, &data)
	if data["behavior"] != "MONITORING" {
		t.Errorf("期望behavior=MONITORING，实际=%v", data["behavior"])
	}
}

func TestNewClient_ReceivesCachedStatusImmediately(t *testing.T) {
	s := New()
	s.BroadcastStatus(map[string]any{"behavior": "NORMAL"})

	srv := newTestHTTPServer(t, s)

	conn, _, err := testDialer.Dial(wsURL(srv.URL), nil)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	var env Envelope
	if err := conn.ReadJSON(&env); err != nil {
		t.Fatalf("新连接应该立即收到缓存的状态: %v", err)
	}
	if env.Type != "device_status" {
		t.Errorf("期望type=device_status，实际=%s", env.Type)
	}
}

func TestCommands_ReceivesFromClient(t *testing.T) {
	s := New()
	srv := newTestHTTPServer(t, s)

	conn, _, err := testDialer.Dial(wsURL(srv.URL), nil)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	data, _ := json.Marshal(map[string]string{"scenario": "fall"})
	if err := conn.WriteJSON(Envelope{Type: "scenario_command", Data: data}); err != nil {
		t.Fatalf("发送指令失败: %v", err)
	}

	select {
	case cmd := <-s.Commands():
		if cmd.Type != "scenario_command" {
			t.Fatalf("期望type=scenario_command，实际=%s", cmd.Type)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("超时：服务器没有收到客户端指令")
	}
}

func TestBroadcastStatus_NoClients_DoesNotPanic(t *testing.T) {
	s := New()
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("没有客户端时广播不应该panic，实际: %v", r)
		}
	}()
	s.BroadcastStatus(map[string]any{"behavior": "NORMAL"})
	s.BroadcastEvent("test", "测试事件")
	s.BroadcastChatTranscript("测试", true)
	s.BroadcastAgentState("idle")
}

func waitForClientCount(t *testing.T, s *Server, want int) {
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if s.clientCount() == want {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("超时：客户端数量未达到期望值 %d", want)
}

func TestNewClient_ReceivesCachedSystemStateAndConfig(t *testing.T) {
	s := New()
	s.BroadcastSystemState(true, false)
	s.BroadcastConfig(30, 200, 600)

	srv := newTestHTTPServer(t, s)

	conn, _, err := testDialer.Dial(wsURL(srv.URL), nil)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	gotTypes := map[string]json.RawMessage{}
	for i := 0; i < 2; i++ {
		var env Envelope
		if err := conn.ReadJSON(&env); err != nil {
			t.Fatalf("读取消息失败: %v", err)
		}
		gotTypes[env.Type] = env.Data
	}

	sysData, ok := gotTypes["system_state"]
	if !ok {
		t.Fatal("新连接应该立即收到缓存的system_state")
	}
	var sys map[string]bool
	_ = json.Unmarshal(sysData, &sys)
	if sys["system_enabled"] != true || sys["voice_enabled"] != false {
		t.Errorf("system_state内容不符: %v", sys)
	}

	cfgData, ok := gotTypes["config_state"]
	if !ok {
		t.Fatal("新连接应该立即收到缓存的config_state")
	}
	var cfg map[string]int
	_ = json.Unmarshal(cfgData, &cfg)
	if cfg["static_alert_after_seconds"] != 30 || cfg["fire_threshold"] != 200 {
		t.Errorf("config_state内容不符: %v", cfg)
	}
}

func TestBroadcastSystemState_DeliversToConnectedClient(t *testing.T) {
	s := New()
	srv := newTestHTTPServer(t, s)

	conn, _, err := testDialer.Dial(wsURL(srv.URL), nil)
	if err != nil {
		t.Fatalf("连接失败: %v", err)
	}
	defer conn.Close()

	waitForClientCount(t, s, 1)

	s.BroadcastSystemState(false, true)

	var env Envelope
	if err := conn.ReadJSON(&env); err != nil {
		t.Fatalf("读取消息失败: %v", err)
	}
	if env.Type != "system_state" {
		t.Fatalf("期望type=system_state，实际=%s", env.Type)
	}
	var data map[string]bool
	_ = json.Unmarshal(env.Data, &data)
	if data["system_enabled"] != false || data["voice_enabled"] != true {
		t.Errorf("system_state内容不符: %v", data)
	}
}
