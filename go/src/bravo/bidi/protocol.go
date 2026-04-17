package bidi

import (
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/net/websocket"
)

type request struct {
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type response struct {
	Type    string          `json:"type"`
	ID      int64           `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   string          `json:"error,omitempty"`
	Message string          `json:"message,omitempty"`
}

// Conn wraps a WebSocket and provides synchronous BiDi JSON-RPC calls.
type Conn struct {
	ws  *websocket.Conn
	seq atomic.Int64
	mu  sync.Mutex
}

// Dial connects to a BiDi WebSocket endpoint.
func Dial(url string) (*Conn, error) {
	ws, err := websocket.Dial(url, "", "http://localhost")
	if err != nil {
		return nil, fmt.Errorf("bidi dial: %w", err)
	}
	return &Conn{ws: ws}, nil
}

// Send sends a BiDi method call and returns the result.
func (c *Conn) Send(method string, params any) (json.RawMessage, error) {
	id := c.seq.Add(1)

	var rawParams json.RawMessage
	if params != nil {
		var err error
		if rawParams, err = json.Marshal(params); err != nil {
			return nil, fmt.Errorf("bidi marshal params: %w", err)
		}
	}

	req := request{ID: id, Method: method, Params: rawParams}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := websocket.JSON.Send(c.ws, req); err != nil {
		return nil, fmt.Errorf("bidi send: %w", err)
	}

	for {
		var resp response
		if err := websocket.JSON.Receive(c.ws, &resp); err != nil {
			return nil, fmt.Errorf("bidi receive: %w", err)
		}

		if resp.ID != id {
			// Event or response for a different ID — skip.
			continue
		}

		if resp.Type == "error" {
			return nil, fmt.Errorf("bidi error %s: %s", resp.Error, resp.Message)
		}

		return resp.Result, nil
	}
}

// Close closes the WebSocket connection.
func (c *Conn) Close() error {
	return c.ws.Close()
}
