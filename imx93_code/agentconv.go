package main

import (
	"log"
	"sync"
	"time"

	"imx93-guard/actuator"
	"imx93-guard/applink"
	"imx93-guard/statemachine"
	"imx93-guard/voiceagent"
)

var micLock sync.Mutex

const agentConversationMaxDuration = 5 * time.Minute

type agentConversationController struct {
	mu      sync.Mutex
	active  bool
	stopCh  chan struct{}
	doneCh  chan struct{}
}

func newAgentConversationController() *agentConversationController {
	return &agentConversationController{}
}

func (c *agentConversationController) IsActive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.active
}

func (c *agentConversationController) Start(cfg voiceagent.Config, link *applink.Server, act *actuator.Actuator, sm *statemachine.Machine) {
	c.mu.Lock()
	if c.active {
		c.mu.Unlock()
		return
	}
	c.active = true
	c.stopCh = make(chan struct{})
	c.doneCh = make(chan struct{})
	stopCh := c.stopCh
	doneCh := c.doneCh
	c.mu.Unlock()

	link.BroadcastAgentConversationState(true)
	go c.runLoop(cfg, link, act, sm, stopCh, doneCh)
}

func (c *agentConversationController) Stop() {
	c.mu.Lock()
	if !c.active {
		c.mu.Unlock()
		return
	}
	stopCh := c.stopCh
	c.mu.Unlock()

	close(stopCh)
	<-c.doneCh
}

func (c *agentConversationController) markStopped(link *applink.Server) {
	c.mu.Lock()
	c.active = false
	c.mu.Unlock()
	link.BroadcastAgentConversationState(false)
}

func (c *agentConversationController) runLoop(cfg voiceagent.Config, link *applink.Server, act *actuator.Actuator, sm *statemachine.Machine, stopCh, doneCh chan struct{}) {
	defer close(doneCh)
	defer c.markStopped(link)

	deadline := time.Now().Add(agentConversationMaxDuration)

	for {
		select {
		case <-stopCh:
			link.BroadcastAgentState("idle")
			return
		default:
		}

		if time.Now().After(deadline) {
			log.Println("自然对话已达最长持续时长，自动停止(避免遗忘关闭导致持续消耗语音API额度)")
			link.BroadcastAgentState("idle")
			link.BroadcastEvent("agent_conversation_timeout", "对话已持续较长时间，已自动停止")
			return
		}

		if !micLock.TryLock() {
			select {
			case <-stopCh:
				link.BroadcastAgentState("idle")
				return
			case <-time.After(500 * time.Millisecond):
				continue
			}
		}

		stopped := c.runOneTurn(cfg, link, act, sm, stopCh)
		micLock.Unlock()

		if stopped {
			return
		}
	}
}

func (c *agentConversationController) runOneTurn(cfg voiceagent.Config, link *applink.Server, act *actuator.Actuator, sm *statemachine.Machine, stopCh chan struct{}) bool {
	link.BroadcastAgentState("listening")

	tools := buildAgentTools(act, sm)
	result, err := voiceagent.Converse(cfg, tools, agentListenSeconds, agentListenStartTimeout)
	if err != nil {
		log.Printf("自然对话出错: %v", err)
		link.BroadcastAgentState("idle")

		select {
		case <-stopCh:
			return true
		case <-time.After(2 * time.Second):
		}
		return false
	}

	if !result.Heard {

		link.BroadcastAgentState("idle")
		return false
	}

	log.Printf("自然对话识别到用户说话: %q (调用工具=%v)", result.UserText, result.InvokedTools)
	link.BroadcastChatTranscript(result.UserText, true)
	link.BroadcastAgentState("thinking")
	link.BroadcastChatTranscript(result.AssistantText, false)
	link.BroadcastAgentState("idle")

	return result.StopRequested
}
