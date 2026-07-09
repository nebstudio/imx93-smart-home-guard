package applink

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Envelope struct {
	Type      string          `json:"type"`
	Timestamp int64           `json:"timestamp"`
	Data      json.RawMessage `json:"data,omitempty"`
}

type Command struct {
	Type string
	Data json.RawMessage
}

type Server struct {
	mu      sync.RWMutex
	clients map[*appConn]bool

	latestStatus json.RawMessage
	hasStatus    bool

	latestSystemState json.RawMessage
	hasSystemState    bool
	latestConfig      json.RawMessage
	hasConfig         bool

	commands chan Command
}

func New() *Server {
	return &Server{
		clients:  make(map[*appConn]bool),
		commands: make(chan Command, 16),
	}
}

func (s *Server) Commands() <-chan Command {
	return s.commands
}

func (s *Server) RegisterHandler(mux *http.ServeMux) {
	mux.HandleFunc("/ws/app", s.handleWS)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("APP WebSocket升级失败: %v", err)
		return
	}
	conn := &appConn{ws: ws}

	s.addClient(conn)
	defer s.removeClient(conn)

	for {
		var env Envelope
		if err := ws.ReadJSON(&env); err != nil {
			return
		}
		select {
		case s.commands <- Command{Type: env.Type, Data: env.Data}:
		default:
			log.Println("警告: 指令队列已满，丢弃一条来自APP的待处理消息")
		}
	}
}

func (s *Server) addClient(conn *appConn) {
	s.mu.Lock()
	s.clients[conn] = true
	status := s.latestStatus
	hasStatus := s.hasStatus
	systemState := s.latestSystemState
	hasSystemState := s.hasSystemState
	config := s.latestConfig
	hasConfig := s.hasConfig
	s.mu.Unlock()

	log.Printf("APP客户端已连接，当前在线数: %d", s.clientCount())

	if hasStatus {
		conn.send(Envelope{Type: "device_status", Timestamp: nowMillis(), Data: status})
	}
	if hasSystemState {
		conn.send(Envelope{Type: "system_state", Timestamp: nowMillis(), Data: systemState})
	}
	if hasConfig {
		conn.send(Envelope{Type: "config_state", Timestamp: nowMillis(), Data: config})
	}
}

func (s *Server) removeClient(conn *appConn) {
	s.mu.Lock()
	delete(s.clients, conn)
	s.mu.Unlock()
	log.Printf("APP客户端已断开，当前在线数: %d", s.clientCount())
}

func (s *Server) clientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

func (s *Server) BroadcastStatus(payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("序列化状态失败: %v", err)
		return
	}

	s.mu.Lock()
	s.latestStatus = data
	s.hasStatus = true
	s.mu.Unlock()

	s.broadcast(Envelope{Type: "device_status", Timestamp: nowMillis(), Data: data})
}

func (s *Server) BroadcastEvent(kind, message string) {
	data, _ := json.Marshal(map[string]string{"kind": kind, "message": message})
	s.broadcast(Envelope{Type: "device_event", Timestamp: nowMillis(), Data: data})
}

func (s *Server) BroadcastChatTranscript(text string, isUser bool) {
	data, _ := json.Marshal(map[string]any{"text": text, "is_user": isUser})
	s.broadcast(Envelope{Type: "agent_chat_transcript", Timestamp: nowMillis(), Data: data})
}

func (s *Server) BroadcastAgentState(state string) {
	data, _ := json.Marshal(map[string]string{"state": state})
	s.broadcast(Envelope{Type: "agent_state", Timestamp: nowMillis(), Data: data})
}

func (s *Server) BroadcastAgentConversationState(active bool) {
	data, _ := json.Marshal(map[string]bool{"active": active})
	s.broadcast(Envelope{Type: "agent_conversation_state", Timestamp: nowMillis(), Data: data})
}

func (s *Server) BroadcastSystemState(systemEnabled, voiceEnabled bool) {
	data, err := json.Marshal(map[string]bool{
		"system_enabled": systemEnabled,
		"voice_enabled":  voiceEnabled,
	})
	if err != nil {
		log.Printf("序列化系统开关状态失败: %v", err)
		return
	}

	s.mu.Lock()
	s.latestSystemState = data
	s.hasSystemState = true
	s.mu.Unlock()

	s.broadcast(Envelope{Type: "system_state", Timestamp: nowMillis(), Data: data})
}

func (s *Server) BroadcastConfig(staticAlertAfterSeconds, fireThreshold, smokeThreshold int) {
	data, err := json.Marshal(map[string]int{
		"static_alert_after_seconds": staticAlertAfterSeconds,
		"fire_threshold":             fireThreshold,
		"smoke_threshold":            smokeThreshold,
	})
	if err != nil {
		log.Printf("序列化参数配置失败: %v", err)
		return
	}

	s.mu.Lock()
	s.latestConfig = data
	s.hasConfig = true
	s.mu.Unlock()

	s.broadcast(Envelope{Type: "config_state", Timestamp: nowMillis(), Data: data})
}

func (s *Server) BroadcastFrame(frameBase64 string, posture string, person bool) {
	data, _ := json.Marshal(map[string]any{
		"frame":   frameBase64,
		"posture": posture,
		"person":  person,
	})
	s.broadcast(Envelope{Type: "camera_frame", Timestamp: nowMillis(), Data: data})
}

func (s *Server) broadcast(env Envelope) {
	s.mu.RLock()
	clients := make([]*appConn, 0, len(s.clients))
	for c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.RUnlock()

	for _, c := range clients {
		c.send(env)
	}
}

func nowMillis() int64 {
	return time.Now().UnixMilli()
}

type appConn struct {
	ws *websocket.Conn
	mu sync.Mutex
}

func (c *appConn) send(env Envelope) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.ws.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := c.ws.WriteJSON(env); err != nil {
		log.Printf("发送消息给APP客户端失败: %v", err)
	}
}
