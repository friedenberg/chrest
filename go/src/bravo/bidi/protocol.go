package bidi

import (
	"bufio"
	"crypto/rand"
	"crypto/sha1"
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
	"time"
)

type request struct {
	ID     int64           `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params,omitempty"`
}

type response struct {
	Type      string          `json:"type"`
	ID        int64           `json:"id"`
	Result    json.RawMessage `json:"result,omitempty"`
	ErrorType string          `json:"error,omitempty"`
	Message   string          `json:"message,omitempty"`
}

const (
	opcodeText  = 0x1
	opcodeClose = 0x8
	opcodePing  = 0x9
	opcodePong  = 0xA

	wsGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

	// Maximum frames to skip (events, non-matching responses) before
	// giving up. Prevents infinite loops if the server never responds.
	maxSkippedFrames = 1000

	dialTimeout = 5 * time.Second
	readTimeout = 30 * time.Second
)

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

	conn, err := net.DialTimeout("tcp", hostPort, dialTimeout)
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
	resp.Body.Close()

	if resp.StatusCode != 101 {
		conn.Close()
		return nil, fmt.Errorf("bidi upgrade failed: HTTP %d %s", resp.StatusCode, resp.Status)
	}

	// Validate Sec-WebSocket-Accept per RFC 6455 section 4.1.
	expectedAccept := computeAcceptKey(wsKey)
	if got := resp.Header.Get("Sec-WebSocket-Accept"); got != expectedAccept {
		conn.Close()
		return nil, fmt.Errorf("bidi: invalid Sec-WebSocket-Accept: got %q, want %q", got, expectedAccept)
	}

	log.Printf("bidi: connected")
	return &Conn{conn: conn, br: br}, nil
}

// computeAcceptKey computes the expected Sec-WebSocket-Accept value.
func computeAcceptKey(key string) string {
	h := sha1.New()
	h.Write([]byte(key + wsGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// writeFrame writes a WebSocket text frame with masking (client must mask).
func (c *Conn) writeFrame(payload []byte) error {
	// Text frame, FIN bit set.
	header := []byte{0x80 | opcodeText}

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

// writePong sends a WebSocket pong frame with the given payload.
func (c *Conn) writePong(payload []byte) error {
	header := []byte{0x80 | opcodePong}

	length := len(payload)
	if length > 125 {
		length = 125
	}
	header = append(header, byte(length)|0x80)

	mask := make([]byte, 4)
	if _, err := rand.Read(mask); err != nil {
		return err
	}
	header = append(header, mask...)

	masked := make([]byte, length)
	for i := 0; i < length; i++ {
		masked[i] = payload[i] ^ mask[i%4]
	}

	if _, err := c.conn.Write(header); err != nil {
		return err
	}
	_, err := c.conn.Write(masked)
	return err
}

// readFrame reads a WebSocket frame, handling control frames (close, ping).
// Returns the payload for data frames only.
func (c *Conn) readFrame() ([]byte, error) {
	for {
		hdr := make([]byte, 2)
		if _, err := io.ReadFull(c.br, hdr); err != nil {
			return nil, err
		}

		opcode := hdr[0] & 0x0F
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

		switch opcode {
		case opcodeClose:
			return nil, fmt.Errorf("bidi: server sent close frame")
		case opcodePing:
			if err := c.writePong(payload); err != nil {
				return nil, fmt.Errorf("bidi: pong failed: %w", err)
			}
			continue
		case opcodePong:
			continue
		default:
			return payload, nil
		}
	}
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

	// Set a read deadline so we don't block forever.
	c.conn.SetReadDeadline(time.Now().Add(readTimeout))
	defer c.conn.SetReadDeadline(time.Time{})

	if err := c.writeFrame(payload); err != nil {
		return nil, fmt.Errorf("bidi send: %w", err)
	}

	for skipped := 0; skipped < maxSkippedFrames; skipped++ {
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
			return nil, fmt.Errorf("bidi error %s: %s", resp.ErrorType, resp.Message)
		}

		return resp.Result, nil
	}

	return nil, fmt.Errorf("bidi: exceeded %d skipped frames waiting for response to %s (id=%d)", maxSkippedFrames, method, id)
}

// Close closes the underlying connection.
func (c *Conn) Close() error {
	return c.conn.Close()
}
