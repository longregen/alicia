package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"strings"
	"sync"
)

type Transport interface {
	Send(ctx context.Context, message any) error
	Receive() <-chan Message
	Close() error
	IsConnected() bool
}

type Message struct {
	Data  []byte
	Error error
}

type StdioTransport struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	receiveCh chan Message
	closeCh   chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup
	mu        sync.RWMutex
	connected bool
}

func validateCommand(command string, args []string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("command cannot be empty")
	}

	shellMetaChars := regexp.MustCompile(`[;&|$` + "`" + `\(\)<>]`)
	if shellMetaChars.MatchString(command) {
		return "", fmt.Errorf("command contains invalid characters")
	}

	cmdPath, err := exec.LookPath(command)
	if err != nil {
		return "", fmt.Errorf("command not found: %s", command)
	}

	for i, arg := range args {
		if shellMetaChars.MatchString(arg) {
			return "", fmt.Errorf("argument %d contains invalid characters", i)
		}
		// Some commands (like git, curl) can be tricked with flags like --config to execute arbitrary code
		lowerArg := strings.ToLower(arg)
		if strings.HasPrefix(lowerArg, "--exec") ||
			strings.HasPrefix(lowerArg, "--config=") ||
			strings.HasPrefix(lowerArg, "-c=") {
			return "", fmt.Errorf("argument %d contains potentially dangerous flag", i)
		}
	}

	return cmdPath, nil
}

func NewStdioTransport(command string, args []string, env []string) (*StdioTransport, error) {
	cmdPath, err := validateCommand(command, args)
	if err != nil {
		return nil, fmt.Errorf("invalid command: %w", err)
	}

	cmd := exec.Command(cmdPath, args...)
	if env != nil {
		cmd.Env = env
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		stdin.Close()
		stdout.Close()
		return nil, fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	transport := &StdioTransport{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
		receiveCh: make(chan Message, 10),
		closeCh:   make(chan struct{}),
		connected: false,
	}

	if err := cmd.Start(); err != nil {
		stdin.Close()
		stdout.Close()
		stderr.Close()
		return nil, fmt.Errorf("failed to start command: %w", err)
	}

	transport.mu.Lock()
	transport.connected = true
	transport.mu.Unlock()

	transport.wg.Add(2)

	go transport.readLoop()
	go transport.readStderr()
	go transport.monitorProcess()

	return transport, nil
}

func (t *StdioTransport) Send(ctx context.Context, message any) error {
	t.mu.RLock()
	if !t.connected {
		t.mu.RUnlock()
		return fmt.Errorf("transport not connected")
	}
	t.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	data = append(data, '\n')

	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

func (t *StdioTransport) Receive() <-chan Message {
	return t.receiveCh
}

func (t *StdioTransport) Close() error {
	var err error
	t.closeOnce.Do(func() {
		close(t.closeCh)

		t.mu.Lock()
		t.connected = false
		t.mu.Unlock()

		if t.stdin != nil {
			t.stdin.Close()
		}

		if t.cmd != nil && t.cmd.Process != nil {
			if killErr := t.cmd.Process.Kill(); killErr != nil {
				err = killErr
			}
		}

		if t.stdout != nil {
			t.stdout.Close()
		}

		if t.stderr != nil {
			t.stderr.Close()
		}

		// Use a goroutine to avoid blocking Close() indefinitely while waiting for senders
		go func() {
			t.wg.Wait()
			close(t.receiveCh)
		}()
	})
	return err
}

func (t *StdioTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected
}

func (t *StdioTransport) readLoop() {
	defer t.wg.Done()

	scanner := bufio.NewScanner(t.stdout)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for {
		select {
		case <-t.closeCh:
			return
		default:
			if !scanner.Scan() {
				if err := scanner.Err(); err != nil {
					select {
					case t.receiveCh <- Message{Error: fmt.Errorf("scanner error: %w", err)}:
					case <-t.closeCh:
					}
				}
				return
			}

			data := scanner.Bytes()
			if len(data) == 0 {
				continue
			}

			// Scanner reuses the buffer, so we need a copy
			dataCopy := make([]byte, len(data))
			copy(dataCopy, data)

			select {
			case t.receiveCh <- Message{Data: dataCopy}:
			case <-t.closeCh:
				return
			}
		}
	}
}

func (t *StdioTransport) readStderr() {
	scanner := bufio.NewScanner(t.stderr)
	for scanner.Scan() {
		_ = scanner.Text()
	}
}

func (t *StdioTransport) monitorProcess() {
	defer t.wg.Done()

	if err := t.cmd.Wait(); err != nil {
		t.mu.Lock()
		if t.connected {
			t.connected = false
			select {
			case t.receiveCh <- Message{Error: fmt.Errorf("process exited: %w", err)}:
			case <-t.closeCh:
			}
		}
		t.mu.Unlock()
	}
}
