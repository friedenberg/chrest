package cdp

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
	ID     int64           `json:"id"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *rpcError) Error() string {
	return fmt.Sprintf("CDP error %d: %s", e.Code, e.Message)
}

// Conn wraps a WebSocket and provides synchronous CDP JSON-RPC calls.
type Conn struct {
	ws  *websocket.Conn
	seq atomic.Int64
	mu  sync.Mutex // serializes reads (CDP responses are ordered)
}

// Dial connects to a CDP WebSocket endpoint.
func Dial(url string) (*Conn, error) {
	ws, err := websocket.Dial(url, "", "http://localhost")
	if err != nil {
		return nil, fmt.Errorf("cdp dial: %w", err)
	}
	return &Conn{ws: ws}, nil
}

// Send sends a CDP method call and returns the result.
func (c *Conn) Send(method string, params any) (json.RawMessage, error) {
	id := c.seq.Add(1)

	var rawParams json.RawMessage
	if params != nil {
		var err error
		if rawParams, err = json.Marshal(params); err != nil {
			return nil, fmt.Errorf("cdp marshal params: %w", err)
		}
	}

	req := request{ID: id, Method: method, Params: rawParams}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := websocket.JSON.Send(c.ws, req); err != nil {
		return nil, fmt.Errorf("cdp send: %w", err)
	}

	// Read responses until we find ours (skip events).
	for {
		var resp response
		if err := websocket.JSON.Receive(c.ws, &resp); err != nil {
			return nil, fmt.Errorf("cdp receive: %w", err)
		}

		if resp.ID == id {
			if resp.Error != nil {
				return nil, resp.Error
			}
			return resp.Result, nil
		}
		// Response for a different ID or an event — skip.
	}
}

// Close closes the WebSocket connection.
func (c *Conn) Close() error {
	return c.ws.Close()
}
