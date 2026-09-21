// Package terminal manages the tmux sessions that outlive browser attachments.
package terminal

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"sync"

	"github.com/creack/pty"
)

var tmuxIdentifier = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// Config describes the one fixed terminal used by the first prototype.
type Config struct {
	TmuxPath    string
	SocketName  string
	SessionName string
	AgentPath   string
	AgentArgs   []string
	WorkingDir  string
}

// Runner executes tmux control commands. It is injectable so command
// construction and idempotency can be tested without changing real sessions.
type Runner interface {
	Run(context.Context, string, ...string) error
}

// TerminalAttachment is one temporary PTY client attached to a persistent
// terminal session.
type TerminalAttachment interface {
	io.ReadWriteCloser
	Resize(cols, rows uint16) error
}

// Session is the lifecycle boundary used by the Web attachment handler.
type Session interface {
	Ensure(context.Context, uint16, uint16) error
	OpenAttachment(context.Context, uint16, uint16) (TerminalAttachment, error)
}

// ExecRunner runs commands directly without a shell.
type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) error {
	return exec.CommandContext(ctx, name, args...).Run()
}

// Output runs a tmux inspection command and returns its stdout.
func (ExecRunner) Output(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

// Manager owns a tmux session but not any individual browser attachment.
type Manager struct {
	config   Config
	runner   Runner
	startPTY func(*exec.Cmd, *pty.Winsize) (*os.File, error)
	activeMu sync.Mutex
	active   *Attachment
}

// NewManager validates the fixed command and tmux identifiers.
func NewManager(config Config, runner Runner) (*Manager, error) {
	if runner == nil {
		return nil, fmt.Errorf("terminal runner is required")
	}
	if config.TmuxPath == "" || config.AgentPath == "" || config.WorkingDir == "" {
		return nil, fmt.Errorf("tmux path, agent path, and working directory are required")
	}
	if !tmuxIdentifier.MatchString(config.SocketName) {
		return nil, fmt.Errorf("invalid tmux socket name %q", config.SocketName)
	}
	if !tmuxIdentifier.MatchString(config.SessionName) {
		return nil, fmt.Errorf("invalid tmux session name %q", config.SessionName)
	}
	return &Manager{config: config, runner: runner, startPTY: pty.StartWithSize}, nil
}

// Ensure creates the agent session once. An existing session is reused.
func (manager *Manager) Ensure(ctx context.Context, cols, rows uint16) error {
	if cols < 2 || rows < 1 || cols > 500 || rows > 300 {
		return fmt.Errorf("terminal size %dx%d is outside prototype limits", cols, rows)
	}

	base := manager.baseArgs()
	target := "=" + manager.config.SessionName
	if err := manager.runner.Run(ctx, manager.config.TmuxPath, append(base, "has-session", "-t", target)...); err != nil {
		args := append(base,
			"new-session", "-d",
			"-s", manager.config.SessionName,
			"-x", strconv.Itoa(int(cols)),
			"-y", strconv.Itoa(int(rows)),
			"-c", manager.config.WorkingDir,
			"--", "/usr/bin/env", "-u", "NO_COLOR", manager.config.AgentPath,
		)
		args = append(args, manager.config.AgentArgs...)
		if err := manager.runner.Run(ctx, manager.config.TmuxPath, args...); err != nil {
			return fmt.Errorf("create tmux session %q: %w", manager.config.SessionName, err)
		}
	}

	// The iPhone has no physical mouse wheel. The Web client translates a
	// vertical touch gesture into native wheel events so tmux can expose its
	// own scrollback without turning terminal output into application data.
	if err := manager.runner.Run(ctx, manager.config.TmuxPath, append(base, "set-option", "-g", "mouse", "on")...); err != nil {
		return fmt.Errorf("enable tmux scrollback gestures: %w", err)
	}
	// tmux normally forwards wheel events to applications in the alternate
	// screen. Codex uses that screen, so reserve wheel-up for tmux history and
	// let copy mode handle subsequent up/down events natively.
	wheelUp := append(base,
		"bind-key", "-T", "root", "WheelUpPane",
		"copy-mode", "-e", "-u",
	)
	if err := manager.runner.Run(ctx, manager.config.TmuxPath, wheelUp...); err != nil {
		return fmt.Errorf("configure tmux scrollback gesture: %w", err)
	}
	return nil
}

// AttachCommand returns a direct tmux invocation for a temporary PTY client.
func (manager *Manager) AttachCommand() (string, []string) {
	return manager.config.TmuxPath, append(manager.baseArgs(), "attach-session", "-t", "="+manager.config.SessionName)
}

// OpenAttachment starts a temporary tmux client under a PTY. Closing this PTY
// detaches the client; the tmux session and agent keep running.
func (manager *Manager) OpenAttachment(ctx context.Context, cols, rows uint16) (TerminalAttachment, error) {
	if cols < 2 || rows < 1 || cols > 500 || rows > 300 {
		return nil, fmt.Errorf("terminal size %dx%d is outside prototype limits", cols, rows)
	}

	name, args := manager.AttachCommand()
	command := exec.CommandContext(ctx, name, args...)
	command.Env = append(os.Environ(), "TERM=xterm-256color")
	terminalFile, err := manager.startPTY(command, &pty.Winsize{Cols: cols, Rows: rows})
	if err != nil {
		return nil, fmt.Errorf("start tmux attachment PTY: %w", err)
	}
	go func() { _ = command.Wait() }()
	attachment := &Attachment{file: terminalFile}
	attachment.onClose = func() {
		manager.activeMu.Lock()
		defer manager.activeMu.Unlock()
		if manager.active == attachment {
			manager.active = nil
		}
	}

	manager.activeMu.Lock()
	previous := manager.active
	manager.active = attachment
	manager.activeMu.Unlock()
	if previous != nil {
		_ = previous.Close()
	}
	return attachment, nil
}

func (manager *Manager) baseArgs() []string {
	return []string{"-L", manager.config.SocketName, "-f", "/dev/null"}
}

// Attachment is one temporary tmux attach client.
type Attachment struct {
	file      *os.File
	closeOnce sync.Once
	closeErr  error
	onClose   func()
}

func (attachment *Attachment) Read(buffer []byte) (int, error) {
	return attachment.file.Read(buffer)
}

func (attachment *Attachment) Write(data []byte) (int, error) {
	return attachment.file.Write(data)
}

func (attachment *Attachment) Close() error {
	attachment.closeOnce.Do(func() {
		attachment.closeErr = attachment.file.Close()
		if attachment.onClose != nil {
			attachment.onClose()
		}
	})
	return attachment.closeErr
}

func (attachment *Attachment) Resize(cols, rows uint16) error {
	if cols < 2 || rows < 1 || cols > 500 || rows > 300 {
		return fmt.Errorf("terminal size %dx%d is outside prototype limits", cols, rows)
	}
	return pty.Setsize(attachment.file, &pty.Winsize{Cols: cols, Rows: rows})
}
