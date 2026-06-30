package player

import (
	"encoding/json"
	"net"
	"testing"
)

func TestMPVIPCClientPause(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	done := make(chan error, 1)
	go func() {
		cmd, err := readTestCommand(serverConn)
		if err != nil {
			done <- err
			return
		}

		if got := cmd.Command[0]; got != "set_property" {
			t.Errorf("command name = %v, want set_property", got)
		}
		if got := cmd.Command[1]; got != "pause" {
			t.Errorf("property = %v, want pause", got)
		}
		if got := cmd.Command[2]; got != true {
			t.Errorf("pause value = %v, want true", got)
		}

		done <- writeTestResponse(serverConn, cmd.RequestID, nil)
	}()

	client := &MPVIPCClient{conn: clientConn}
	if err := client.Pause(); err != nil {
		t.Fatalf("Pause() error = %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("server error = %v", err)
	}
}

func TestMPVIPCClientStatus(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	done := make(chan error, 1)
	go func() {
		cmd, err := readTestCommand(serverConn)
		if err != nil {
			done <- err
			return
		}

		if got := cmd.Command[0]; got != "get_property" {
			t.Errorf("command name = %v, want get_property", got)
		}
		if got := cmd.Command[1]; got != "pause" {
			t.Errorf("property = %v, want pause", got)
		}

		done <- writeTestResponse(serverConn, cmd.RequestID, json.RawMessage("true"))
	}()

	client := &MPVIPCClient{conn: clientConn}
	status, err := client.Status()
	if err != nil {
		t.Fatalf("Status() error = %v", err)
	}
	if status != "paused" {
		t.Fatalf("Status() = %q, want paused", status)
	}
	if err := <-done; err != nil {
		t.Fatalf("server error = %v", err)
	}
}

func TestMPVIPCClientStopWritesQuitWithoutWaitingForResponse(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	defer clientConn.Close()
	defer serverConn.Close()

	done := make(chan error, 1)
	go func() {
		cmd, err := readTestCommand(serverConn)
		if err != nil {
			done <- err
			return
		}

		if got := cmd.Command[0]; got != "quit" {
			t.Errorf("command name = %v, want quit", got)
		}

		done <- nil
	}()

	client := &MPVIPCClient{conn: clientConn}
	if err := client.Stop(); err != nil {
		t.Fatalf("Stop() error = %v", err)
	}
	if err := <-done; err != nil {
		t.Fatalf("server error = %v", err)
	}
}

func readTestCommand(conn net.Conn) (mpvCommand, error) {
	var cmd mpvCommand
	err := json.NewDecoder(conn).Decode(&cmd)
	return cmd, err
}

func writeTestResponse(conn net.Conn, requestID int, data json.RawMessage) error {
	response := mpvResponse{
		RequestID: requestID,
		Error:     "success",
		Data:      data,
	}
	return json.NewEncoder(conn).Encode(response)
}
