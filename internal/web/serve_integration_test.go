package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// TestServeIntegration is opt-in because it requires a running daemon and a
// configured tailnet Serve endpoint. It validates the real HTTPS/WSS proxy,
// not a local httptest server.
func TestServeIntegration(t *testing.T) {
	websocketURL := os.Getenv("CODE_REMOTE_WSS_URL")
	if websocketURL == "" {
		t.Skip("CODE_REMOTE_WSS_URL is not set")
	}
	parsed, err := url.Parse(websocketURL)
	if err != nil {
		t.Fatal(err)
	}
	originScheme := "https"
	if parsed.Scheme == "ws" {
		originScheme = "http"
	}
	origin := originScheme + "://" + parsed.Host

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	connection, response, err := websocket.Dial(ctx, websocketURL, &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{origin}},
	})
	if err != nil {
		if response != nil {
			t.Fatalf("WSS dial failed with HTTP %d: %v", response.StatusCode, err)
		}
		t.Fatal(err)
	}
	defer connection.CloseNow()

	request := wireMessage{
		Version: protocolVersion,
		Type:    "terminal.attach",
		Payload: mustMarshal(attachPayload{Cols: 60, Rows: 30}),
	}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatal(err)
	}
	if err := connection.Write(ctx, websocket.MessageText, data); err != nil {
		t.Fatal(err)
	}
	message, err := readWireMessage(ctx, connection)
	if err != nil {
		t.Fatal(err)
	}
	if message.Type != "terminal.attached" {
		t.Fatalf("first server message type = %q, want terminal.attached", message.Type)
	}
	if err := connection.Close(websocket.StatusNormalClosure, "probe complete"); err != nil {
		t.Fatal(err)
	}
}
