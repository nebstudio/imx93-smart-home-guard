package serialio

import (
	"bufio"
	"fmt"
	"sync"
	"testing"
	"time"

	"go.bug.st/serial"
)

type fakeArduinoPort struct {
	mu      sync.Mutex
	pending []byte
	closed  bool
}

func (f *fakeArduinoPort) SetMode(mode *serial.Mode) error { return nil }

func (f *fakeArduinoPort) Read(p []byte) (int, error) {

	for {
		f.mu.Lock()
		if len(f.pending) > 0 {
			n := copy(p, f.pending)
			f.pending = f.pending[n:]
			f.mu.Unlock()
			return n, nil
		}
		closed := f.closed
		f.mu.Unlock()
		if closed {
			return 0, fmt.Errorf("port closed")
		}
		time.Sleep(time.Millisecond)
	}
}

func (f *fakeArduinoPort) Write(p []byte) (int, error) {
	cmd := string(p)
	reply := replyFor(cmd)

	f.mu.Lock()
	f.pending = append(f.pending, []byte(reply)...)
	f.mu.Unlock()
	return len(p), nil
}

func (f *fakeArduinoPort) Drain() error            { return nil }
func (f *fakeArduinoPort) ResetInputBuffer() error { return nil }
func (f *fakeArduinoPort) ResetOutputBuffer() error { return nil }
func (f *fakeArduinoPort) SetDTR(dtr bool) error   { return nil }
func (f *fakeArduinoPort) SetRTS(rts bool) error   { return nil }
func (f *fakeArduinoPort) GetModemStatusBits() (*serial.ModemStatusBits, error) {
	return &serial.ModemStatusBits{}, nil
}
func (f *fakeArduinoPort) SetReadTimeout(t time.Duration) error { return nil }
func (f *fakeArduinoPort) Close() error {
	f.mu.Lock()
	f.closed = true
	f.mu.Unlock()
	return nil
}
func (f *fakeArduinoPort) Break(d time.Duration) error { return nil }

func replyFor(cmd string) string {
	switch {
	case len(cmd) > 0 && cmd[0] == 'D' && len(cmd) > 1 && cmd[len(cmd)-2] != '?':
		return "OK\n"
	case len(cmd) > 1 && cmd[len(cmd)-2] == '?' && cmd[0] == 'D':

		return fmt.Sprintf("%s,1\n", trimSuffixQuestion(cmd))
	case len(cmd) > 0 && cmd[0] == 'U':
		return "U,80\n"
	case len(cmd) > 0 && cmd[0] == 'A':
		return fmt.Sprintf("%s,512\n", trimSuffixQuestion(cmd))
	case len(cmd) > 0 && (cmd[0] == 'P' || cmd[0] == 'S' || cmd[0] == 'B' || cmd[0] == 'M'):
		return "OK\n"
	default:
		return "ERR,unknown\n"
	}
}

func trimSuffixQuestion(cmd string) string {

	trimmed := cmd
	for len(trimmed) > 0 && (trimmed[len(trimmed)-1] == '\n' || trimmed[len(trimmed)-1] == '?') {
		trimmed = trimmed[:len(trimmed)-1]
	}
	return trimmed
}

func newTestClient() *Client {
	port := &fakeArduinoPort{}
	return &Client{
		port:    port,
		reader:  bufio.NewReader(port),
		timeout: 2 * time.Second,
	}
}

func TestClient_ConcurrentAccess_NoRaceAndNoReplyMismatch(t *testing.T) {
	c := newTestClient()

	var wg sync.WaitGroup
	errCh := make(chan error, 200)

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			d, err := c.ReadUltrasonicCm()
			if err != nil {
				errCh <- fmt.Errorf("ReadUltrasonicCm: %w", err)
				return
			}
			if d != 80 {
				errCh <- fmt.Errorf("ReadUltrasonicCm回复错位，期望80，实际%d", d)
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			if err := c.WriteDigital(7, i%2 == 0); err != nil {
				errCh <- fmt.Errorf("WriteDigital: %w", err)
				return
			}
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			if err := c.Buzz(2000, 100); err != nil {
				errCh <- fmt.Errorf("Buzz: %w", err)
				return
			}
			if _, err := c.ReadAnalog(1); err != nil {
				errCh <- fmt.Errorf("ReadAnalog: %w", err)
				return
			}
		}
	}()

	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Error(err)
	}
}
