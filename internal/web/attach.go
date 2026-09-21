package web

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/coder/websocket"
	terminalpkg "github.com/huangxinxinyu/CodeRemote/internal/terminal"
)

const (
	protocolVersion     = 1
	maxClientMessage    = 64 << 10
	terminalReadBuffer  = 16 << 10
	websocketWriteLimit = 10 * time.Second
)

type TerminalAttachment = terminalpkg.TerminalAttachment
type TerminalSession = terminalpkg.Session

type wireMessage struct {
	Version int             `json:"v"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

type attachPayload struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

type inputPayload struct {
	AttachmentID string `json:"attachment_id"`
	Data         string `json:"data_base64"`
}

type resizePayload struct {
	AttachmentID string `json:"attachment_id"`
	Cols         uint16 `json:"cols"`
	Rows         uint16 `json:"rows"`
}

// NewAttachHandler bridges one WebSocket to one temporary tmux PTY client.
func NewAttachHandler(session TerminalSession) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !SameOrigin(request) {
			http.Error(response, "WebSocket origin is not allowed", http.StatusForbidden)
			return
		}

		connection, err := websocket.Accept(response, request, &websocket.AcceptOptions{
			InsecureSkipVerify: true, // SameOrigin above is intentionally stricter.
			CompressionMode:    websocket.CompressionDisabled,
		})
		if err != nil {
			return
		}
		defer connection.CloseNow()
		connection.SetReadLimit(maxClientMessage)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		message, err := readWireMessage(ctx, connection)
		if err != nil || message.Version != protocolVersion || message.Type != "terminal.attach" {
			closeProtocolError(connection, "first message must be terminal.attach v1")
			return
		}
		var attach attachPayload
		if err := decodeStrict(message.Payload, &attach); err != nil || !validTerminalSize(attach.Cols, attach.Rows) {
			closeProtocolError(connection, "invalid terminal.attach payload")
			return
		}
		if err := session.Ensure(ctx, attach.Cols, attach.Rows); err != nil {
			writeTerminalError(ctx, connection, "failed to start terminal")
			return
		}
		terminal, err := session.OpenAttachment(ctx, attach.Cols, attach.Rows)
		if err != nil {
			writeTerminalError(ctx, connection, "failed to attach terminal")
			return
		}
		defer terminal.Close()

		attachmentID, err := newAttachmentID()
		if err != nil {
			writeTerminalError(ctx, connection, "failed to create attachment")
			return
		}
		if err := writeWireMessage(ctx, connection, "terminal.attached", map[string]any{
			"attachment_id": attachmentID,
		}); err != nil {
			return
		}

		outputDone := make(chan struct{})
		go func() {
			defer close(outputDone)
			streamTerminalOutput(ctx, connection, terminal, attachmentID)
		}()

		for {
			message, err := readWireMessage(ctx, connection)
			if err != nil {
				cancel()
				return
			}
			if message.Version != protocolVersion {
				closeProtocolError(connection, "unsupported protocol version")
				return
			}

			switch message.Type {
			case "terminal.input":
				var input inputPayload
				if err := decodeStrict(message.Payload, &input); err != nil || input.AttachmentID != attachmentID {
					closeProtocolError(connection, "invalid terminal.input payload")
					return
				}
				data, err := base64.StdEncoding.DecodeString(input.Data)
				if err != nil || len(data) > maxClientMessage {
					closeProtocolError(connection, "invalid terminal input")
					return
				}
				if _, err := terminal.Write(data); err != nil {
					return
				}
			case "terminal.resize":
				var resize resizePayload
				if err := decodeStrict(message.Payload, &resize); err != nil || resize.AttachmentID != attachmentID || !validTerminalSize(resize.Cols, resize.Rows) {
					closeProtocolError(connection, "invalid terminal.resize payload")
					return
				}
				if err := terminal.Resize(resize.Cols, resize.Rows); err != nil {
					return
				}
			default:
				closeProtocolError(connection, "unsupported terminal message")
				return
			}

			select {
			case <-outputDone:
				return
			default:
			}
		}
	})
}

func streamTerminalOutput(ctx context.Context, connection *websocket.Conn, terminal TerminalAttachment, attachmentID string) {
	buffer := make([]byte, terminalReadBuffer)
	var sequence uint64
	for {
		count, err := terminal.Read(buffer)
		if count > 0 {
			sequence++
			if writeErr := writeWireMessage(ctx, connection, "terminal.output", map[string]any{
				"attachment_id": attachmentID,
				"seq":           sequence,
				"data_base64":   base64.StdEncoding.EncodeToString(buffer[:count]),
			}); writeErr != nil {
				return
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) && ctx.Err() == nil {
				writeTerminalError(ctx, connection, "terminal output ended unexpectedly")
			}
			_ = connection.Close(websocket.StatusNormalClosure, "terminal attachment ended")
			return
		}
	}
}

func readWireMessage(ctx context.Context, connection *websocket.Conn) (wireMessage, error) {
	messageType, data, err := connection.Read(ctx)
	if err != nil {
		return wireMessage{}, err
	}
	if messageType != websocket.MessageText {
		return wireMessage{}, fmt.Errorf("binary control message is not allowed")
	}
	var message wireMessage
	if err := decodeStrict(data, &message); err != nil {
		return wireMessage{}, err
	}
	return message, nil
}

func writeWireMessage(ctx context.Context, connection *websocket.Conn, messageType string, payload any) error {
	writeCtx, cancel := context.WithTimeout(ctx, websocketWriteLimit)
	defer cancel()
	data, err := json.Marshal(wireMessage{
		Version: protocolVersion,
		Type:    messageType,
		Payload: mustMarshal(payload),
	})
	if err != nil {
		return err
	}
	return connection.Write(writeCtx, websocket.MessageText, data)
}

func writeTerminalError(ctx context.Context, connection *websocket.Conn, message string) {
	_ = writeWireMessage(ctx, connection, "terminal.error", map[string]string{"message": message})
}

func closeProtocolError(connection *websocket.Conn, message string) {
	_ = connection.Close(websocket.StatusPolicyViolation, message)
}

func decodeStrict(data []byte, destination any) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		return fmt.Errorf("multiple JSON values are not allowed")
	}
	return nil
}

func mustMarshal(value any) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil {
		panic(err)
	}
	return data
}

func newAttachmentID() (string, error) {
	data := make([]byte, 16)
	if _, err := rand.Read(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(data), nil
}

func validTerminalSize(cols, rows uint16) bool {
	return cols >= 2 && cols <= 500 && rows >= 1 && rows <= 300
}
