package websocket

// NOTE: This is a minimal WebSocket implementation for goryu.
// It is not intended to be a full-featured WebSocket library, but rather a simple way to
// upgrade an HTTP connection to a WebSocket connection and read/write messages.
//  For more advanced features I will consider using a dedicated WebSocket library.
// or expanding this implementation in the future.
// Therefore I might be not be adding I will see

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	goryuctx "github.com/arthurlch/goryu/goryuctx"
)

const (
	TextMessage       = 0x1
	BinaryMessage     = 0x2
	continuationFrame = 0x0
	CloseMessage      = 0x8
	pingMessage       = 0x9
	pongMessage       = 0xA

	acceptMagic    = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	maxMessageSize = 32 << 20

	defaultIdleTimeout  = 60 * time.Second
	defaultWriteTimeout = 10 * time.Second
)

var (
	ErrClosed   = errors.New("websocket: connection closed")
	errProtocol = errors.New("websocket: protocol error")
)

type Conn struct {
	conn      net.Conn
	rw        *bufio.ReadWriter
	mu        sync.Mutex
	idle      time.Duration
	writeWait time.Duration
}

func Upgrade(c *goryuctx.Context) (*Conn, error) {
	r := c.Request
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return nil, errors.New("websocket: not an upgrade request")
	}
	if !headerHasToken(r.Header.Get("Connection"), "upgrade") {
		return nil, errors.New("websocket: missing Connection upgrade")
	}
	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		return nil, errors.New("websocket: missing Sec-WebSocket-Key")
	}
	hj, ok := c.Writer.(http.Hijacker)
	if !ok {
		return nil, errors.New("websocket: response writer does not support hijacking")
	}
	conn, rw, err := hj.Hijack()
	if err != nil {
		return nil, err
	}
	resp := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey(key) + "\r\n\r\n"
	if _, err := rw.WriteString(resp); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if err := rw.Flush(); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return &Conn{conn: conn, rw: rw, idle: defaultIdleTimeout, writeWait: defaultWriteTimeout}, nil
}

func (c *Conn) SetReadDeadline(t time.Time) error  { return c.conn.SetReadDeadline(t) }
func (c *Conn) SetWriteDeadline(t time.Time) error { return c.conn.SetWriteDeadline(t) }
func (c *Conn) SetIdleTimeout(d time.Duration)     { c.idle = d }
func (c *Conn) SetWriteTimeout(d time.Duration)    { c.writeWait = d }

func (c *Conn) ReadMessage() (int, []byte, error) {
	var message []byte
	var messageType int
	fragmented := false
	for {
		op, payload, fin, err := c.readFrame()
		if err != nil {
			return 0, nil, err
		}
		switch op {
		case pingMessage:
			if err := c.writeFrame(pongMessage, payload); err != nil {
				return 0, nil, err
			}
			continue
		case pongMessage:
			continue
		case CloseMessage:
			_ = c.writeFrame(CloseMessage, closeEcho(payload))
			return CloseMessage, payload, ErrClosed
		case TextMessage, BinaryMessage:
			if fragmented {
				return 0, nil, errProtocol
			}
			messageType = op
			message = payload
		case continuationFrame:
			if !fragmented {
				return 0, nil, errProtocol
			}
			if len(message)+len(payload) > maxMessageSize {
				return 0, nil, errors.New("websocket: message too large")
			}
			message = append(message, payload...)
		default:
			return 0, nil, errProtocol
		}
		if fin {
			return messageType, message, nil
		}
		fragmented = true
	}
}

func (c *Conn) WriteMessage(messageType int, data []byte) error {
	return c.writeFrame(messageType, data)
}

func (c *Conn) WriteText(s string) error {
	return c.writeFrame(TextMessage, []byte(s))
}

func (c *Conn) Close() error {
	_ = c.writeFrame(CloseMessage, nil)
	return c.conn.Close()
}

func (c *Conn) readFrame() (op int, payload []byte, fin bool, err error) {
	if c.idle > 0 {
		_ = c.conn.SetReadDeadline(time.Now().Add(c.idle))
	}
	var head [2]byte
	if _, err = io.ReadFull(c.rw, head[:]); err != nil {
		return
	}
	if head[0]&0x70 != 0 {
		err = errProtocol
		return
	}
	fin = head[0]&0x80 != 0
	op = int(head[0] & 0x0f)
	masked := head[1]&0x80 != 0
	length := uint64(head[1] & 0x7f)
	switch length {
	case 126:
		var ext [2]byte
		if _, err = io.ReadFull(c.rw, ext[:]); err != nil {
			return
		}
		length = uint64(binary.BigEndian.Uint16(ext[:]))
	case 127:
		var ext [8]byte
		if _, err = io.ReadFull(c.rw, ext[:]); err != nil {
			return
		}
		length = binary.BigEndian.Uint64(ext[:])
	}
	if op >= CloseMessage && (!fin || length > 125) {
		err = errProtocol
		return
	}
	if !masked {
		err = errors.New("websocket: unmasked client frame")
		return
	}
	if length > maxMessageSize {
		err = errors.New("websocket: frame too large")
		return
	}
	var mask [4]byte
	if _, err = io.ReadFull(c.rw, mask[:]); err != nil {
		return
	}
	payload = make([]byte, length)
	if _, err = io.ReadFull(c.rw, payload); err != nil {
		return
	}
	for i := range payload {
		payload[i] ^= mask[i%4]
	}
	return
}

func (c *Conn) writeFrame(op int, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.writeWait > 0 {
		_ = c.conn.SetWriteDeadline(time.Now().Add(c.writeWait))
	}

	header := []byte{byte(0x80 | op)}
	n := len(payload)
	switch {
	case n < 126:
		header = append(header, byte(n))
	case n < 65536:
		header = append(header, 126, byte(n>>8), byte(n))
	default:
		header = append(header, 127)
		var ext [8]byte
		binary.BigEndian.PutUint64(ext[:], uint64(n))
		header = append(header, ext[:]...)
	}
	if _, err := c.rw.Write(header); err != nil {
		return err
	}
	if _, err := c.rw.Write(payload); err != nil {
		return err
	}
	return c.rw.Flush()
}

func closeEcho(payload []byte) []byte {
	if len(payload) >= 2 {
		return payload[:2]
	}
	return []byte{0x03, 0xe8}
}

func acceptKey(key string) string {
	h := sha1.New()
	_, _ = io.WriteString(h, key+acceptMagic)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func headerHasToken(header, token string) bool {
	for _, part := range strings.Split(header, ",") {
		if strings.EqualFold(strings.TrimSpace(part), token) {
			return true
		}
	}
	return false
}
