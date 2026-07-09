package posesensor

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sync"
	"time"
)

type Snapshot struct {
	Timestamp float64 `json:"ts"`
	Person    bool    `json:"person"`
	Posture   string  `json:"posture"`
	Conf      float64 `json:"conf"`

	Frame string `json:"frame,omitempty"`
}

const (
	PostureNone     = "none"
	PostureStanding = "standing"
	PostureSitting  = "sitting"
	PostureLying    = "lying"
)

type Sensor struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	mu     sync.RWMutex
	latest Snapshot

	hasData bool

	stdinMu sync.Mutex

	errMu    sync.Mutex
	lastErr  error
	stopped  bool
	doneChan chan struct{}
}

func Start(pythonExec, scriptPath, workDir string) (*Sensor, error) {
	cmd := exec.Command(pythonExec, scriptPath)
	cmd.Dir = workDir

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("创建 stdout 管道失败: %w", err)
	}
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("创建 stdin 管道失败: %w", err)
	}

	cmd.Stderr = nil

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动姿态识别子进程失败: %w", err)
	}

	s := &Sensor{
		cmd:      cmd,
		stdin:    stdin,
		doneChan: make(chan struct{}),
	}

	go s.readLoop(stdout)

	return s, nil
}

func (s *Sensor) SetStreamEnabled(enabled bool) error {
	s.stdinMu.Lock()
	defer s.stdinMu.Unlock()

	cmd := "stream_off"
	if enabled {
		cmd = "stream_on"
	}
	_, err := s.stdin.Write([]byte(fmt.Sprintf(`{"cmd":%q}`+"\n", cmd)))
	return err
}

func (s *Sensor) readLoop(stdout io.Reader) {
	defer close(s.doneChan)

	scanner := bufio.NewScanner(stdout)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var snap Snapshot
		if err := json.Unmarshal(line, &snap); err != nil {

			continue
		}
		s.mu.Lock()
		s.latest = snap
		s.hasData = true
		s.mu.Unlock()
	}

	if err := scanner.Err(); err != nil {
		s.setErr(fmt.Errorf("读取姿态识别子进程输出出错: %w", err))
		return
	}

	s.setErr(fmt.Errorf("姿态识别子进程输出流已结束(进程可能已退出)"))
}

func (s *Sensor) setErr(err error) {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	s.lastErr = err
}

func (s *Sensor) Err() error {
	s.errMu.Lock()
	defer s.errMu.Unlock()
	return s.lastErr
}

func (s *Sensor) Latest() (Snapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.latest, s.hasData
}

func (s *Sensor) IsFresh(maxAge time.Duration) bool {
	snap, ok := s.Latest()
	if !ok {
		return false
	}
	age := time.Since(time.Unix(0, int64(snap.Timestamp*1e9)))
	return age <= maxAge
}

func (s *Sensor) Stop() error {
	s.errMu.Lock()
	if s.stopped {
		s.errMu.Unlock()
		return nil
	}
	s.stopped = true
	s.errMu.Unlock()

	_ = s.stdin.Close()
	if s.cmd.Process != nil {
		_ = s.cmd.Process.Kill()
	}
	<-s.doneChan
	return s.cmd.Wait()
}
