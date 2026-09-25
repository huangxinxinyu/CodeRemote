package web

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
)

type fakeSession struct {
	attachment *fakeAttachment
	ensured    chan terminalSize
	returned   chan struct{}
}

func newFakeSession() *fakeSession {
	return &fakeSession{
		attachment: newFakeAttachment(),
		ensured:    make(chan terminalSize, 1),
		returned:   make(chan struct{}, 1),
	}
}

func (session *fakeSession) Ensure(_ context.Context, cols, rows uint16) error {
	session.ensured <- terminalSize{cols: cols, rows: rows}
	return nil
}

func (session *fakeSession) OpenAttachment(_ context.Context, _, _ uint16) (TerminalAttachment, error) {
	return session.attachment, nil
}

func (session *fakeSession) ReturnToLive(_ context.Context) error {
	session.returned <- struct{}{}
	return nil
}

func TestAttachHandlerReturnsFromHistoryBeforeNativeInput(t *testing.T) {
	t.Parallel()
	session := newFakeSession()
	server := httptest.NewServer(NewHandler(NewAttachHandler(session)))
	t.Cleanup(server.Close)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	connection, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/terminals/prototype/attach", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{server.URL}}})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.CloseNow() })
	writeJSON(t, ctx, connection, `{"v":1,"type":"terminal.attach","payload":{"cols":46,"rows":19}}`)
	attached := readMessage(t, ctx, connection)
	if attached.Type != "terminal.attached" {
		t.Fatalf("message = %q, want terminal.attached", attached.Type)
	}
	id := attachmentID(t, attached.Payload)
	writeJSON(t, ctx, connection, `{"v":1,"type":"terminal.focus","payload":{"attachment_id":"`+id+`"}}`)
	writeJSON(t, ctx, connection, `{"v":1,"type":"terminal.input","payload":{"attachment_id":"`+id+`","data_base64":"`+base64.StdEncoding.EncodeToString([]byte("hello"))+`"}}`)
	select {
	case <-session.returned:
	case <-ctx.Done():
		t.Fatal("terminal focus did not leave history mode")
	}
	select {
	case input := <-session.attachment.input:
		if string(input) != "hello" {
			t.Fatalf("terminal input = %q, want hello", input)
		}
	case <-ctx.Done():
		t.Fatal("native input did not reach terminal")
	}
}

type fakeAttachment struct {
	reader    *io.PipeReader
	output    *io.PipeWriter
	input     chan []byte
	resizes   chan terminalSize
	closed    chan struct{}
	closeOnce sync.Once
}

func newFakeAttachment() *fakeAttachment {
	reader, writer := io.Pipe()
	return &fakeAttachment{
		reader:  reader,
		output:  writer,
		input:   make(chan []byte, 1),
		resizes: make(chan terminalSize, 1),
		closed:  make(chan struct{}),
	}
}

func (attachment *fakeAttachment) Read(buffer []byte) (int, error) {
	return attachment.reader.Read(buffer)
}

func (attachment *fakeAttachment) Write(data []byte) (int, error) {
	copyOfData := append([]byte(nil), data...)
	attachment.input <- copyOfData
	return len(data), nil
}

func (attachment *fakeAttachment) Resize(cols, rows uint16) error {
	attachment.resizes <- terminalSize{cols: cols, rows: rows}
	return nil
}

func (attachment *fakeAttachment) Close() error {
	attachment.closeOnce.Do(func() {
		close(attachment.closed)
		_ = attachment.reader.Close()
		_ = attachment.output.Close()
	})
	return nil
}

type terminalSize struct {
	cols uint16
	rows uint16
}

type testMessage struct {
	Version int             `json:"v"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

func TestAttachHandlerBridgesTerminalBytesAndResize(t *testing.T) {
	t.Parallel()

	session := newFakeSession()
	server := httptest.NewServer(NewHandler(NewAttachHandler(session)))
	t.Cleanup(server.Close)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	connection, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(server.URL, "http")+"/api/v1/terminals/prototype/attach", &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{server.URL}},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = connection.CloseNow() })

	writeJSON(t, ctx, connection, `{"v":1,"type":"terminal.attach","payload":{"cols":60,"rows":30}}`)
	attached := readMessage(t, ctx, connection)
	if attached.Type != "terminal.attached" || !strings.Contains(string(attached.Payload), "attachment_id") {
		t.Fatalf("attached message = %#v", attached)
	}

	select {
	case size := <-session.ensured:
		if size != (terminalSize{cols: 60, rows: 30}) {
			t.Fatalf("Ensure size = %#v", size)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	wantOutput := []byte("你好\r\n\x1b[32mready\x1b[0m")
	if _, err := session.attachment.output.Write(wantOutput); err != nil {
		t.Fatal(err)
	}
	output := readMessage(t, ctx, connection)
	if output.Type != "terminal.output" {
		t.Fatalf("output type = %q", output.Type)
	}
	var outputPayload struct {
		Data string `json:"data_base64"`
	}
	if err := json.Unmarshal(output.Payload, &outputPayload); err != nil {
		t.Fatal(err)
	}
	decoded, err := base64.StdEncoding.DecodeString(outputPayload.Data)
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != string(wantOutput) {
		t.Fatalf("terminal output = %q, want %q", decoded, wantOutput)
	}

	writeJSON(t, ctx, connection, `{"v":1,"type":"terminal.input","payload":{"attachment_id":"`+attachmentID(t, attached.Payload)+`","data_base64":"`+base64.StdEncoding.EncodeToString([]byte("测试\n"))+`"}}`)
	select {
	case input := <-session.attachment.input:
		if string(input) != "测试\n" {
			t.Fatalf("terminal input = %q", input)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	writeJSON(t, ctx, connection, `{"v":1,"type":"terminal.resize","payload":{"attachment_id":"`+attachmentID(t, attached.Payload)+`","cols":80,"rows":24}}`)
	select {
	case size := <-session.attachment.resizes:
		if size != (terminalSize{cols: 80, rows: 24}) {
			t.Fatalf("Resize size = %#v", size)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}

	if err := connection.Close(websocket.StatusNormalClosure, "test complete"); err != nil {
		t.Fatal(err)
	}
	select {
	case <-session.attachment.closed:
	case <-ctx.Done():
		t.Fatal("PTY attachment remained open after WebSocket closed")
	}
}

func writeJSON(t *testing.T, ctx context.Context, connection *websocket.Conn, message string) {
	t.Helper()
	if err := connection.Write(ctx, websocket.MessageText, []byte(message)); err != nil {
		t.Fatal(err)
	}
}

func readMessage(t *testing.T, ctx context.Context, connection *websocket.Conn) testMessage {
	t.Helper()
	_, data, err := connection.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var message testMessage
	if err := json.Unmarshal(data, &message); err != nil {
		t.Fatal(err)
	}
	return message
}

func attachmentID(t *testing.T, payload json.RawMessage) string {
	t.Helper()
	var value struct {
		AttachmentID string `json:"attachment_id"`
	}
	if err := json.Unmarshal(payload, &value); err != nil {
		t.Fatal(err)
	}
	if value.AttachmentID == "" {
		t.Fatal("attachment_id is empty")
	}
	return value.AttachmentID
}
