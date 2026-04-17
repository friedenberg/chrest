package bidi

import (
	"bufio"
	"crypto/rand"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
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

// Conn wraps a raw TCP connection with WebSocket framing for BiDi JSON-RPC.
type Conn struct {
	conn net.Conn
	br   *bufio.Reader
	seq  atomic.Int64
	mu   sync.Mutex
}

// Dial connects to a BiDi WebSocket endpoint.
// The baseURL is the URL Firefox logs (e.g. ws://127.0.0.1:PORT).
// Firefox requires the /session path for BiDi connections.
func Dial(baseURL string) (*Conn, error) {
	wsURL := strings.TrimRight(baseURL, "/") + "/session"
	log.Printf("bidi: dialing %s", wsURL)

	host := strings.TrimPrefix(wsURL, "ws://")
	pathIdx := strings.Index(host, "/")
	hostPort := host
	path := "/"
	if pathIdx >= 0 {
		hostPort = host[:pathIdx]
		path = host[pathIdx:]
	}

	conn, err := net.Dial("tcp", hostPort)
	if err != nil {
		return nil, fmt.Errorf("bidi tcp dial: %w", err)
	}

	keyBytes := make([]byte, 16)
	if _, err := rand.Read(keyBytes); err != nil {
		conn.Close()
		return nil, fmt.Errorf("bidi key gen: %w", err)
	}
	wsKey := base64.StdEncoding.EncodeToString(keyBytes)

	upgradeReq := fmt.Sprintf(
		"GET %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Key: %s\r\n"+
			"Sec-WebSocket-Version: 13\r\n"+
			"\r\n",
		path, hostPort, wsKey,
	)

	if _, err := conn.Write([]byte(upgradeReq)); err != nil {
		conn.Close()
		return nil, fmt.Errorf("bidi upgrade write: %w", err)
	}

	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("bidi upgrade read: %w", err)
	}

	if resp.StatusCode != 101 {
		resp.Body.Close()
		conn.Close()
		return nil, fmt.Errorf("bidi upgrade failed: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	log.Printf("bidi: connected")
	return &Conn{conn: conn, br: br}, nil
}

// writeFrame writes a WebSocket text frame with masking (client must mask).
func (c *Conn) writeFrame(payload []byte) error {
	// Text frame, FIN bit set.
	header := []byte{0x81}

	// Payload length with mask bit set.
	length := len(payload)
	switch {
	case length <= 125:
		header = append(header, byte(length)|0x80)
	case length <= 65535:
		header = append(header, 126|0x80, byte(length>>8), byte(length))
	default:
		buf := make([]byte, 8)
		binary.BigEndian.PutUint64(buf, uint64(length))
		header = append(header, 127|0x80)
		header = append(header, buf...)
	}

	// Masking key.
	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	header = append(header, mask...)

	// Mask the payload.
	masked := make([]byte, len(payload))
	for i, b := range payload {
		masked[i] = b ^ mask[i%4]
	}

	if _, err := c.conn.Write(header); err != nil {
		return err
	}
	_, err := c.conn.Write(masked)
	return err
}

// readFrame reads a WebSocket frame and returns the payload.
func (c *Conn) readFrame() ([]byte, error) {
	// Read first 2 bytes: FIN/opcode + mask/length.
	hdr := make([]byte, 2)
	if _, err := io.ReadFull(c.br, hdr); err != nil {
		return nil, err
	}

	masked := hdr[1]&0x80 != 0
	length := uint64(hdr[1] & 0x7F)

	switch length {
	case 126:
		buf := make([]byte, 2)
		if _, err := io.ReadFull(c.br, buf); err != nil {
			return nil, err
		}
		length = uint64(binary.BigEndian.Uint16(buf))
	case 127:
		buf := make([]byte, 8)
		if _, err := io.ReadFull(c.br, buf); err != nil {
			return nil, err
		}
		length = binary.BigEndian.Uint64(buf)
	}

	var mask []byte
	if masked {
		mask = make([]byte, 4)
		if _, err := io.ReadFull(c.br, mask); err != nil {
			return nil, err
		}
	}

	payload := make([]byte, length)
	if _, err := io.ReadFull(c.br, payload); err != nil {
		return nil, err
	}

	if masked {
		for i := range payload {
			payload[i] ^= mask[i%4]
		}
	}

	return payload, nil
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
	payload, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("bidi marshal request: %w", err)
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.writeFrame(payload); err != nil {
		return nil, fmt.Errorf("bidi send: %w", err)
	}

	for {
		frame, err := c.readFrame()
		if err != nil {
			return nil, fmt.Errorf("bidi receive: %w", err)
		}

		var resp response
		if err := json.Unmarshal(frame, &resp); err != nil {
			return nil, fmt.Errorf("bidi unmarshal response: %w", err)
		}

		if resp.ID != id {
			continue
		}

		if resp.Type == "error" {
			return nil, fmt.Errorf("bidi error %s: %s", resp.Error, resp.Message)
		}

		return resp.Result, nil
	}
}

// Close closes the underlying connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}
