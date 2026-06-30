package player

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"time"
)

const (
	socketTimeout   = 5 * time.Second
	commandTimeout  = 2 * time.Second
	mpvSocketPrefix = "mpv-socket-"
)

// MPVPlayer implements Player interface using mpv with IPC
type MPVPlayer struct {
	socketPath string
	conn       net.Conn
	cmd        *exec.Cmd
	mu         sync.Mutex
	requestID  int
	dial       socketDialer
}

type socketDialer func(network string, address string, timeout time.Duration) (net.Conn, error)

// mpvCommand represents a JSON-RPC command to mpv
type mpvCommand struct {
	Command   []interface{} `json:"command"`
	RequestID int           `json:"request_id,omitempty"`
}

// mpvResponse represents a JSON-RPC response from mpv
type mpvResponse struct {
	RequestID int             `json:"request_id"`
	Error     string          `json:"error"`
	Data      json.RawMessage `json:"data"`
}

// NewMPV creates a new MPV player instance
func NewMPV() Player {
	return &MPVPlayer{}
}

// IsAvailable checks if mpv is installed
func (p *MPVPlayer) IsAvailable() bool {
	_, err := exec.LookPath("mpv")
	return err == nil
}

// Play starts playing the given URL
func (p *MPVPlayer) Play(ctx context.Context, url string, onStart StartFunc) error {
	p.mu.Lock()
	if err := p.cleanup(); err != nil {
		p.mu.Unlock()
		return fmt.Errorf("cleanup existing player: %w", err)
	}

	socketPath, err := p.createSocketPath()
	if err != nil {
		p.mu.Unlock()
		return fmt.Errorf("create socket path: %w", err)
	}

	cmd := exec.CommandContext(ctx, "mpv",
		"--no-video",
		"--really-quiet",
		"--no-terminal",
		fmt.Sprintf("--input-ipc-server=%s", socketPath),
		"--idle=yes",
		url,
	)

	if err := cmd.Start(); err != nil {
		p.mu.Unlock()
		return fmt.Errorf("start mpv: %w", err)
	}
	p.cmd = cmd
	p.socketPath = socketPath
	p.conn = nil
	p.mu.Unlock()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	if err := p.waitForSocket(ctx, socketPath, done); err != nil {
		p.cleanupPlayback(cmd)
		return fmt.Errorf("connect to mpv: %w", err)
	}

	conn, err := p.dialSocket(socketPath)
	if err != nil {
		p.cleanupPlayback(cmd)
		return fmt.Errorf("dial socket: %w", err)
	}

	p.mu.Lock()
	if p.cmd != cmd {
		p.mu.Unlock()
		conn.Close()
		return fmt.Errorf("playback stopped")
	}
	p.conn = conn
	p.mu.Unlock()

	if onStart != nil {
		if err := onStart(PlaybackSession{
			PID:        cmd.Process.Pid,
			SocketPath: socketPath,
		}); err != nil {
			p.cleanupPlayback(cmd)
			return fmt.Errorf("start playback session: %w", err)
		}
	}

	select {
	case <-ctx.Done():
		p.cleanupPlayback(cmd)
		return ctx.Err()
	case err := <-done:
		p.cleanupPlayback(cmd)
		if err != nil {
			// mpv exits with code 0 on normal completion
			if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 0 {
				return nil
			}
			return fmt.Errorf("mpv process: %w", err)
		}
		return nil
	}
}

// Pause pauses the current playback
func (p *MPVPlayer) Pause() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return fmt.Errorf("no active playback")
	}

	return p.sendCommand([]interface{}{"set_property", "pause", true})
}

// Resume resumes the paused playback
func (p *MPVPlayer) Resume() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil {
		return fmt.Errorf("no active playback")
	}

	return p.sendCommand([]interface{}{"set_property", "pause", false})
}

// Stop stops the current playback
func (p *MPVPlayer) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.cleanup()
}

// Status returns the current playback status
func (p *MPVPlayer) Status() string {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.conn == nil || p.cmd == nil {
		return "stopped"
	}

	// Check if process is still running
	if p.cmd.Process != nil {
		if err := p.cmd.Process.Signal(os.Signal(nil)); err != nil {
			return "stopped"
		}
	}

	// Query pause state
	paused, err := p.getProperty("pause")
	if err != nil {
		return "unknown"
	}

	if paused == "true" {
		return "paused"
	}
	return "playing"
}

// Close closes the player and cleans up resources
func (p *MPVPlayer) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.cleanup()
}

// sendCommand sends a JSON-RPC command to mpv
func (p *MPVPlayer) sendCommand(command []interface{}) error {
	if p.conn == nil {
		return fmt.Errorf("not connected to mpv")
	}

	p.requestID++
	cmd := mpvCommand{
		Command:   command,
		RequestID: p.requestID,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("marshal command: %w", err)
	}

	data = append(data, '\n')

	// Set write deadline
	if err := p.conn.SetWriteDeadline(time.Now().Add(commandTimeout)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}

	if _, err := p.conn.Write(data); err != nil {
		return fmt.Errorf("write command: %w", err)
	}

	// Read response
	if err := p.conn.SetReadDeadline(time.Now().Add(commandTimeout)); err != nil {
		return fmt.Errorf("set read deadline: %w", err)
	}

	decoder := json.NewDecoder(p.conn)
	var response mpvResponse
	if err := decoder.Decode(&response); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}

	if response.Error != "success" && response.Error != "" {
		return fmt.Errorf("mpv error: %s", response.Error)
	}

	return nil
}

// getProperty gets a property value from mpv
func (p *MPVPlayer) getProperty(property string) (string, error) {
	if p.conn == nil {
		return "", fmt.Errorf("not connected to mpv")
	}

	p.requestID++
	cmd := mpvCommand{
		Command:   []interface{}{"get_property", property},
		RequestID: p.requestID,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return "", fmt.Errorf("marshal command: %w", err)
	}

	data = append(data, '\n')

	if err := p.conn.SetWriteDeadline(time.Now().Add(commandTimeout)); err != nil {
		return "", fmt.Errorf("set write deadline: %w", err)
	}

	if _, err := p.conn.Write(data); err != nil {
		return "", fmt.Errorf("write command: %w", err)
	}

	if err := p.conn.SetReadDeadline(time.Now().Add(commandTimeout)); err != nil {
		return "", fmt.Errorf("set read deadline: %w", err)
	}

	decoder := json.NewDecoder(p.conn)
	var response mpvResponse
	if err := decoder.Decode(&response); err != nil {
		return "", fmt.Errorf("decode response: %w", err)
	}

	if response.Error != "success" && response.Error != "" {
		return "", fmt.Errorf("mpv error: %s", response.Error)
	}

	return string(response.Data), nil
}

// cleanup closes connections and kills the process
func (p *MPVPlayer) cleanup() error {
	var errs []error

	if p.conn != nil {
		if err := p.conn.Close(); err != nil {
			errs = append(errs, fmt.Errorf("close connection: %w", err))
		}
		p.conn = nil
	}

	// Kill process
	if p.cmd != nil && p.cmd.Process != nil {
		if err := p.cmd.Process.Kill(); err != nil {
			// Ignore "process already finished" errors
			if err.Error() != "os: process already finished" {
				errs = append(errs, fmt.Errorf("kill process: %w", err))
			}
		}
		p.cmd = nil
	}

	// Remove socket file
	if p.socketPath != "" {
		if err := os.Remove(p.socketPath); err != nil && !os.IsNotExist(err) {
			errs = append(errs, fmt.Errorf("remove socket: %w", err))
		}
		p.socketPath = ""
	}

	if len(errs) > 0 {
		return errs[0]
	}
	return nil
}

func (p *MPVPlayer) cleanupPlayback(cmd *exec.Cmd) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.cmd != cmd {
		return nil
	}

	return p.cleanup()
}

func (p *MPVPlayer) dialSocket(socketPath string) (net.Conn, error) {
	dial := net.DialTimeout
	if p.dial != nil {
		dial = p.dial
	}
	return dial("unix", socketPath, socketTimeout)
}

// createSocketPath creates a unique socket path
func (p *MPVPlayer) createSocketPath() (string, error) {
	file, err := os.CreateTemp("", mpvSocketPrefix)
	if err != nil {
		return "", err
	}
	socketPath := file.Name()
	if err := file.Close(); err != nil {
		return "", err
	}
	if err := os.Remove(socketPath); err != nil {
		return "", err
	}
	return socketPath, nil
}

// waitForSocket waits for the mpv socket to be created
func (p *MPVPlayer) waitForSocket(ctx context.Context, socketPath string, done <-chan error) error {
	timeout := time.After(3 * time.Second)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-done:
			if err != nil {
				return fmt.Errorf("mpv exited before socket was ready: %w", err)
			}
			return fmt.Errorf("mpv exited before socket was ready")
		case <-timeout:
			return fmt.Errorf("timeout waiting for socket")
		case <-ticker.C:
			if _, err := os.Stat(socketPath); err == nil {
				return nil
			}
		}
	}
}
