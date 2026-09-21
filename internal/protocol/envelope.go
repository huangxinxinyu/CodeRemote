// Package protocol defines transport-neutral message metadata shared by the Go
// services. The Swift equivalents and wire fixtures will be added when the
// first terminal transport is implemented.
package protocol

import "encoding/json"

const Version = 1

// Envelope is the JSON control-message shape described in docs/protocol.md.
// Terminal bytes remain encoded inside message-specific payloads for v1.
type Envelope struct {
	Version   int             `json:"v"`
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	HostID    string          `json:"host_id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
}
