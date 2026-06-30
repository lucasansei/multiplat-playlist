package player

import (
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type MPVIPCClient struct {
	conn      net.Conn
	requestID int
}

func DialMPVIPC(socketPath string) (*MPVIPCClient, error) {
	conn, err := net.DialTimeout("unix", socketPath, socketTimeout)
	if err != nil {
		return nil, fmt.Errorf("dial mpv socket: %w", err)
	}

	return &MPVIPCClient{conn: conn}, nil
}

func (c *MPVIPCClient) Pause() error {
	return c.sendCommand([]interface{}{"set_property", "pause", true})
}

func (c *MPVIPCClient) Resume() error {
	return c.sendCommand([]interface{}{"set_property", "pause", false})
}

func (c *MPVIPCClient) Stop() error {
	return c.writeCommand([]interface{}{"quit"})
}

func (c *MPVIPCClient) Status() (string, error) {
	paused, err := c.getProperty("pause")
	if err != nil {
		return "", err
	}

	if paused == "true" {
		return "paused", nil
	}
	return "playing", nil
}

func (c *MPVIPCClient) Close() error {
	if c.conn == nil {
		return nil
	}

	err := c.conn.Close()
	c.conn = nil
	if err != nil {
		return fmt.Errorf("close mpv socket: %w", err)
	}
	return nil
}

func (c *MPVIPCClient) sendCommand(command []interface{}) error {
	_, err := c.roundTrip(command)
	return err
}

func (c *MPVIPCClient) getProperty(property string) (string, error) {
	data, err := c.roundTrip([]interface{}{"get_property", property})
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (c *MPVIPCClient) roundTrip(command []interface{}) (json.RawMessage, error) {
	if c.conn == nil {
		return nil, fmt.Errorf("not connected to mpv")
	}

	if err := c.writeCommand(command); err != nil {
		return nil, err
	}

	if err := c.conn.SetReadDeadline(time.Now().Add(commandTimeout)); err != nil {
		return nil, fmt.Errorf("set read deadline: %w", err)
	}

	decoder := json.NewDecoder(c.conn)
	for {
		var response mpvResponse
		if err := decoder.Decode(&response); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}

		if response.RequestID != c.requestID {
			continue
		}

		if response.Error != "success" && response.Error != "" {
			return nil, fmt.Errorf("mpv error: %s", response.Error)
		}

		return response.Data, nil
	}
}

func (c *MPVIPCClient) writeCommand(command []interface{}) error {
	if c.conn == nil {
		return fmt.Errorf("not connected to mpv")
	}

	c.requestID++
	cmd := mpvCommand{
		Command:   command,
		RequestID: c.requestID,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("marshal command: %w", err)
	}
	data = append(data, '\n')

	if err := c.conn.SetWriteDeadline(time.Now().Add(commandTimeout)); err != nil {
		return fmt.Errorf("set write deadline: %w", err)
	}

	if _, err := c.conn.Write(data); err != nil {
		return fmt.Errorf("write command: %w", err)
	}

	return nil
}
