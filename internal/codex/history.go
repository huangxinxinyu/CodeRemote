// Package codex exposes provider-native conversation metadata without parsing
// Codex session files or terminal screen text.
package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	directorypkg "github.com/huangxinxinyu/CodeRemote/internal/directory"
)

const (
	initializeRequestID = 0
	threadListRequestID = 1
)

// Thread is the small, read-only subset of Codex thread metadata used by the
// mobile picker. Name and Preview both come from the native app-server API.
type Thread struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	Preview   string    `json:"preview,omitempty"`
	CreatedAt time.Time `json:"created_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// DisplayName prefers Codex's user-facing name and only falls back to its own
// preview when a newly-created thread has not been named yet.
func (thread Thread) DisplayName() string {
	if name := strings.TrimSpace(thread.Name); name != "" {
		return name
	}
	if preview := strings.TrimSpace(thread.Preview); preview != "" {
		return preview
	}
	return "未命名对话"
}

// History starts a short-lived local app-server connection for each list
// request. It reuses the Mac user's existing Codex login and CODEX_HOME.
type History struct {
	CLIPath    string
	WorkingDir string
}

// List returns recent interactive Codex threads scoped to the requested cwd.
// An empty request falls back to the daemon's configured working directory.
func (history History) List(ctx context.Context, workingDirectory string) ([]Thread, error) {
	if history.CLIPath == "" || history.WorkingDir == "" {
		return nil, fmt.Errorf("codex path and working directory are required")
	}
	workingDirectory, err := directorypkg.Resolve(workingDirectory, history.WorkingDir)
	if err != nil {
		return nil, fmt.Errorf("resolve Codex history working directory: %w", err)
	}
	requestCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	command := exec.CommandContext(requestCtx, history.CLIPath, "app-server")
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("open codex app-server stdin: %w", err)
	}
	stdout, err := command.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("open codex app-server stdout: %w", err)
	}
	if err := command.Start(); err != nil {
		return nil, fmt.Errorf("start codex app-server: %w", err)
	}
	cleanup := func() {
		_ = stdin.Close()
		if command.Process != nil {
			_ = command.Process.Kill()
		}
		_ = command.Wait()
	}

	encoder := json.NewEncoder(stdin)
	if err := encoder.Encode(map[string]any{
		"method": "initialize",
		"id":     initializeRequestID,
		"params": map[string]any{"clientInfo": map[string]string{
			"name": "code_remote", "title": "Code Remote", "version": "0.1.0",
		}},
	}); err != nil {
		cleanup()
		return nil, fmt.Errorf("initialize codex app-server: %w", err)
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	if _, err := readRPCResult(scanner, initializeRequestID); err != nil {
		cleanup()
		return nil, fmt.Errorf("initialize codex app-server: %w", err)
	}
	if err := encoder.Encode(map[string]any{"method": "initialized", "params": map[string]any{}}); err != nil {
		cleanup()
		return nil, err
	}
	if err := encoder.Encode(map[string]any{
		"method": "thread/list",
		"id":     threadListRequestID,
		"params": map[string]any{
			"limit": 50, "sortKey": "updated_at", "sortDirection": "desc", "cwd": workingDirectory,
		},
	}); err != nil {
		cleanup()
		return nil, fmt.Errorf("request Codex thread list: %w", err)
	}

	threads, err := decodeThreadListScanner(scanner, threadListRequestID)
	if err != nil {
		cleanup()
		return nil, err
	}
	_ = stdin.Close()
	if err := command.Wait(); err != nil && requestCtx.Err() != nil {
		return nil, fmt.Errorf("wait for codex app-server: %w", requestCtx.Err())
	}
	return threads, nil
}

type rpcEnvelope struct {
	ID     *int            `json:"id,omitempty"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func readRPCResult(scanner *bufio.Scanner, requestID int) (json.RawMessage, error) {
	for scanner.Scan() {
		var envelope rpcEnvelope
		if err := json.Unmarshal(scanner.Bytes(), &envelope); err != nil {
			continue
		}
		if envelope.ID == nil || *envelope.ID != requestID {
			continue
		}
		if envelope.Error != nil {
			return nil, fmt.Errorf("Codex app-server error %d: %s", envelope.Error.Code, envelope.Error.Message)
		}
		return envelope.Result, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.ErrUnexpectedEOF
}

func decodeThreadList(reader io.Reader, requestID int) ([]Thread, error) {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	return decodeThreadListScanner(scanner, requestID)
}

func decodeThreadListScanner(scanner *bufio.Scanner, requestID int) ([]Thread, error) {
	result, err := readRPCResult(scanner, requestID)
	if err != nil {
		return nil, err
	}
	var payload struct {
		Data []struct {
			ID        string `json:"id"`
			Name      string `json:"name"`
			Preview   string `json:"preview"`
			CreatedAt int64  `json:"createdAt"`
			UpdatedAt int64  `json:"updatedAt"`
		} `json:"data"`
	}
	if err := json.Unmarshal(result, &payload); err != nil {
		return nil, fmt.Errorf("decode Codex thread list: %w", err)
	}
	threads := make([]Thread, 0, len(payload.Data))
	for _, item := range payload.Data {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		threads = append(threads, Thread{
			ID:        item.ID,
			Name:      item.Name,
			Preview:   item.Preview,
			CreatedAt: unixTime(item.CreatedAt),
			UpdatedAt: unixTime(item.UpdatedAt),
		})
	}
	return threads, nil
}

func unixTime(value int64) time.Time {
	if value <= 0 {
		return time.Time{}
	}
	return time.Unix(value, 0).UTC()
}
