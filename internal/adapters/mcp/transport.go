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

// Transport defines the interface for MCP communication
type Transport interface {
	// Send sends a message to the MCP server
	Send(ctx context.Context, message any) error

	// Receive returns a channel for receiving messages from the MCP server
	Receive() <-chan Message

	// Close closes the transport
	Close() error

	// IsConnected returns true if the transport is connected
	IsConnected() bool
}

// Message represents a message received from the transport
type Message struct {
	Data  []byte
	Error error
}

// StdioTransport implements Transport using stdio (stdin/stdout)
type StdioTransport struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	receiveCh chan Message
	closeCh   chan struct{}
	closeOnce sync.Once
	wg        sync.WaitGroup // tracks goroutines that send on receiveCh
	mu        sync.RWMutex
	connected bool
}

// validateCommand validates a command and its arguments to prevent command injection.
// It ensures the command exists as an executable and validates arguments for safety.
func validateCommand(command string, args []string) (string, error) {
	// Reject empty commands
	if command == "" {
		return "", fmt.Errorf("command cannot be empty")
	}

	// Reject commands containing shell metacharacters that could enable injection
	shellMetaChars := regexp.MustCompile(`[;&|$` + "`" + `\(\)<>]`)
	if shellMetaChars.MatchString(command) {
		return "", fmt.Errorf("command contains invalid characters")
	}

	// Use LookPath to resolve the command to its full path
	// This validates the command exists and is executable
	cmdPath, err := exec.LookPath(command)
	if err != nil {
		return "", fmt.Errorf("command not found: %s", command)
	}

	// Validate arguments don't contain shell injection patterns
	for i, arg := range args {
		if shellMetaChars.MatchString(arg) {
			return "", fmt.Errorf("argument %d contains invalid characters", i)
		}
		// Prevent flag injection for commands that may execute arbitrary code
		// Some commands (like git, curl) can be tricked with flags like --config
		// For safety, reject arguments that look like flags attempting to set dangerous options
		lowerArg := strings.ToLower(arg)
		if strings.HasPrefix(lowerArg, "--exec") ||
			strings.HasPrefix(lowerArg, "--config=") ||
			strings.HasPrefix(lowerArg, "-c=") {
			return "", fmt.Errorf("argument %d contains potentially dangerous flag", i)
		}
	}

	return cmdPath, nil
}

// NewStdioTransport creates a new stdio transport
func NewStdioTransport(command string, args []string, env []string) (*StdioTransport, error) {
	// Validate and resolve the command path to prevent command injection
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

	// Track goroutines that may send to receiveCh
	transport.wg.Add(2) // readLoop and monitorProcess

	// Start reading from stdout
	go transport.readLoop()

	// Start reading from stderr (for logging)
	go transport.readStderr()

	// Monitor process exit
	go transport.monitorProcess()

	return transport, nil
}

// Send sends a message to the MCP server
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

	// MCP uses newline-delimited JSON
	data = append(data, '\n')

	if _, err := t.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write message: %w", err)
	}

	return nil
}

// Receive returns a channel for receiving messages
func (t *StdioTransport) Receive() <-chan Message {
	return t.receiveCh
}

// Close closes the transport
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

		// Wait for all senders to finish, then close receiveCh
		// Use a goroutine to avoid blocking Close() indefinitely
		go func() {
			t.wg.Wait()
			close(t.receiveCh)
		}()
	})
	return err
}

// IsConnected returns true if the transport is connected
func (t *StdioTransport) IsConnected() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.connected
}

// readLoop reads messages from stdout
func (t *StdioTransport) readLoop() {
	defer t.wg.Done()

	scanner := bufio.NewScanner(t.stdout)
	// Set a larger buffer size for scanner to handle large messages
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024) // 1MB max

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

			// Make a copy of the data since scanner reuses the buffer
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

// readStderr reads and logs stderr output
func (t *StdioTransport) readStderr() {
	scanner := bufio.NewScanner(t.stderr)
	for scanner.Scan() {
		// In production, this should use proper logging
		// For now, we'll just ignore stderr unless there's an error
		_ = scanner.Text()
	}
}

// monitorProcess monitors the process and closes the transport if it exits
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
