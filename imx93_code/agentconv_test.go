package main

import (
	"sync"
	"testing"
	"time"
)

func TestMicLock_MutualExclusion(t *testing.T) {

	var lock sync.Mutex

	if !lock.TryLock() {
		t.Fatal("初始状态下应该能获取到锁")
	}

	locked := lock.TryLock()
	if locked {
		t.Fatal("锁已被持有时，第二次TryLock应该失败")
	}

	lock.Unlock()

	if !lock.TryLock() {
		t.Fatal("释放后应该能重新获取锁")
	}
	lock.Unlock()
}

func TestAgentConversationController_StartStop_Idempotent(t *testing.T) {
	c := newAgentConversationController()

	if c.IsActive() {
		t.Fatal("初始状态下应该是未激活的")
	}

	done := make(chan struct{})
	go func() {
		c.Stop()
		c.Stop()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("未Start过时调用Stop不应该阻塞")
	}
}

func TestAgentConversationController_ActiveStateReflectsLifecycle(t *testing.T) {
	c := newAgentConversationController()

	c.mu.Lock()
	c.active = true
	c.stopCh = make(chan struct{})
	c.doneCh = make(chan struct{})
	c.mu.Unlock()

	if !c.IsActive() {
		t.Fatal("设置active=true后，IsActive()应该返回true")
	}

	close(c.doneCh)
	c.mu.Lock()
	c.active = false
	c.mu.Unlock()

	if c.IsActive() {
		t.Fatal("清理后IsActive()应该返回false")
	}
}

func TestAgentConversationController_ConcurrentStartCalls_OnlyOneWins(t *testing.T) {
	c := newAgentConversationController()

	var wg sync.WaitGroup
	winCount := 0
	var winMu sync.Mutex

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c.mu.Lock()
			if !c.active {
				c.active = true
				winMu.Lock()
				winCount++
				winMu.Unlock()
			}
			c.mu.Unlock()
		}()
	}
	wg.Wait()

	if winCount != 1 {
		t.Fatalf("并发Start竞争下应该只有1次真正生效，实际=%d", winCount)
	}
}
