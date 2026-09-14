package websocket

import (
	"bufio"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	goryuctx "github.com/arthurlch/goryu/goryuctx"
)

func TestAcceptKey(t *testing.T) {
	if got := acceptKey("dGhlIHNhbXBsZSBub25jZQ=="); got != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("acceptKey = %q", got)
	}
}

type fakeHijacker struct {
	header http.Header
	conn   net.Conn
	rw     *bufio.ReadWriter
}

func (f *fakeHijacker) Header() http.Header                          { return f.header }
func (f *fakeHijacker) Write(b []byte) (int, error)                  { return f.conn.Write(b) }
func (f *fakeHijacker) WriteHeader(int)                              {}
func (f *fakeHijacker) Hijack() (net.Conn, *bufio.ReadWriter, error) { return f.conn, f.rw, nil }

func maskedTextFrame(payload string) []byte {
	mask := []byte{0x12, 0x34, 0x56, 0x78}
	p := []byte(payload)
	frame := []byte{0x81, byte(0x80 | len(p))}
	frame = append(frame, mask...)
	for i, b := range p {
		frame = append(frame, b^mask[i%4])
	}
	return frame
}

func newPipeConn(t *testing.T) (*Conn, net.Conn) {
	t.Helper()
	server, client := net.Pipe()
	c := &Conn{
		conn: server,
		rw:   bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server)),
	}
	return c, client
}

func TestRejectsUnmaskedClientFrame(t *testing.T) {
	c, client := newPipeConn(t)
	go func() {
		client.Write([]byte{0x81, 0x02, 'h', 'i'})
	}()
	if _, _, err := c.ReadMessage(); err == nil {
		t.Fatal("expected error for unmasked client frame")
	}
}

func TestRejectsOversizedControlFrame(t *testing.T) {
	c, client := newPipeConn(t)
	go func() {
		frame := []byte{0x89, 0x80 | 126, 0x00, 0x80}
		frame = append(frame, []byte{0, 0, 0, 0}...)
		frame = append(frame, make([]byte, 128)...)
		client.Write(frame)
	}()
	if _, _, err := c.ReadMessage(); err == nil {
		t.Fatal("expected protocol error for oversized ping frame")
	}
}

func TestUpgradeAndEcho(t *testing.T) {
	server, client := net.Pipe()
	fh := &fakeHijacker{
		header: http.Header{},
		conn:   server,
		rw:     bufio.NewReadWriter(bufio.NewReader(server), bufio.NewWriter(server)),
	}
	req := httptest.NewRequest("GET", "/ws", nil)
	req.Header.Set("Upgrade", "websocket")
	req.Header.Set("Connection", "Upgrade")
	req.Header.Set("Sec-WebSocket-Key", "dGhlIHNhbXBsZSBub25jZQ==")

	go func() {
		conn, err := Upgrade(goryuctx.NewContext(fh, req))
		if err != nil {
			return
		}
		mt, msg, err := conn.ReadMessage()
		if err != nil || mt != TextMessage {
			return
		}
		_ = conn.WriteText(string(msg))
	}()

	cr := bufio.NewReader(client)
	var accept string
	for {
		line, err := cr.ReadString('\n')
		if err != nil {
			t.Fatalf("reading handshake: %v", err)
		}
		if strings.HasPrefix(line, "Sec-WebSocket-Accept:") {
			accept = strings.TrimSpace(strings.TrimPrefix(line, "Sec-WebSocket-Accept:"))
		}
		if line == "\r\n" {
			break
		}
	}
	if accept != "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=" {
		t.Fatalf("handshake accept = %q", accept)
	}

	if _, err := client.Write(maskedTextFrame("ping")); err != nil {
		t.Fatalf("client write: %v", err)
	}

	head := make([]byte, 2)
	if _, err := io.ReadFull(cr, head); err != nil {
		t.Fatalf("read echo head: %v", err)
	}
	if head[0] != 0x81 {
		t.Fatalf("echo not a final text frame: %#x", head[0])
	}
	n := int(head[1] & 0x7f)
	body := make([]byte, n)
	if _, err := io.ReadFull(cr, body); err != nil {
		t.Fatalf("read echo body: %v", err)
	}
	if string(body) != "ping" {
		t.Fatalf("echo = %q", string(body))
	}
}
