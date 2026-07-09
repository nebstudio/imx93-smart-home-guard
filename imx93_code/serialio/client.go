package serialio

import (
	"bufio"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

type Client struct {
	port   serial.Port
	reader *bufio.Reader

	timeout time.Duration

	mu sync.Mutex
}

func Open(portName string, baud int) (*Client, error) {
	mode := &serial.Mode{
		BaudRate: baud,
		Parity:   serial.NoParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open(portName, mode)
	if err != nil {
		return nil, fmt.Errorf("打开串口 %s 失败: %w", portName, err)
	}

	c := &Client{
		port:    port,
		reader:  bufio.NewReader(port),
		timeout: 2 * time.Second,
	}

	time.Sleep(2 * time.Second)
	c.drainInput()

	return c, nil
}

func (c *Client) Close() error {
	return c.port.Close()
}

func (c *Client) drainInput() {
	c.port.ResetInputBuffer()
	_ = c.port.SetReadTimeout(300 * time.Millisecond)
	for {
		line, err := c.reader.ReadString('\n')
		if err != nil {

			return
		}
		_ = line
	}
}

func (c *Client) sendRaw(cmd string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.port.SetReadTimeout(c.timeout); err != nil {
		return "", err
	}
	if _, err := c.port.Write([]byte(cmd + "\n")); err != nil {
		return "", fmt.Errorf("写入指令 %q 失败: %w", cmd, err)
	}
	line, err := c.reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("读取指令 %q 的回复失败: %w", cmd, err)
	}
	return strings.TrimSpace(line), nil
}

func (c *Client) Ping() error {
	resp, err := c.sendRaw("PING?")
	if err != nil {
		return err
	}
	if resp != "PONG" {
		return fmt.Errorf("PING 期望收到 PONG，实际收到: %q", resp)
	}
	return nil
}

func (c *Client) ReadAnalog(analogIndex int) (int, error) {
	resp, err := c.sendRaw(fmt.Sprintf("A%d?", analogIndex))
	if err != nil {
		return 0, err
	}
	return parseValueReply(resp, fmt.Sprintf("A%d", analogIndex))
}

func (c *Client) ReadDigital(pin int) (int, error) {
	resp, err := c.sendRaw(fmt.Sprintf("D%d?", pin))
	if err != nil {
		return 0, err
	}
	return parseValueReply(resp, fmt.Sprintf("D%d", pin))
}

func (c *Client) ReadUltrasonicCm() (int, error) {
	resp, err := c.sendRaw("U?")
	if err != nil {
		return 0, err
	}
	return parseValueReply(resp, "U")
}

func (c *Client) WriteDigital(pin int, val bool) error {
	v := 0
	if val {
		v = 1
	}
	return c.expectOK(fmt.Sprintf("D%d,%d", pin, v))
}

func (c *Client) WritePWM(pin int, duty int) error {
	return c.expectOK(fmt.Sprintf("P%d,%d", pin, duty))
}

func (c *Client) SetServoAngle(servoIndex int, angle int) error {
	return c.expectOK(fmt.Sprintf("S%d,%d", servoIndex, angle))
}

func (c *Client) Buzz(freqHz int, durationMs int) error {
	return c.expectOK(fmt.Sprintf("B%d,%d", freqHz, durationMs))
}

func (c *Client) FanControl(dir int, speed int) error {
	return c.expectOK(fmt.Sprintf("M%d,%d", dir, speed))
}

func (c *Client) ShowLcdLine(line int, text string) error {
	return c.expectOK(fmt.Sprintf("L%d,%s", line, text))
}

func (c *Client) ClearLcd() error {
	return c.expectOK("LC")
}

func (c *Client) ShowLcdEmoji(emojiIndex int) error {
	return c.expectOK(fmt.Sprintf("LE%d", emojiIndex))
}

func (c *Client) expectOK(cmd string) error {
	resp, err := c.sendRaw(cmd)
	if err != nil {
		return err
	}
	if resp != "OK" {
		return fmt.Errorf("指令 %q 期望收到 OK，实际收到: %q", cmd, resp)
	}
	return nil
}

func parseValueReply(resp string, expectPrefix string) (int, error) {
	if strings.HasPrefix(resp, "ERR") {
		return 0, fmt.Errorf("固件返回错误: %s", resp)
	}
	parts := strings.SplitN(resp, ",", 2)
	if len(parts) != 2 || parts[0] != expectPrefix {
		return 0, fmt.Errorf("回复格式不符合预期，期望前缀 %q，实际收到: %q", expectPrefix, resp)
	}
	val, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("解析数值失败: %q: %w", resp, err)
	}
	return val, nil
}
