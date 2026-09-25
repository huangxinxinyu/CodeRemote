package terminal

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"reflect"
	"testing"

	"github.com/creack/pty"
)

type recordedCommand struct {
	name string
	args []string
}

type recordingRunner struct {
	commands   []recordedCommand
	hasSession bool
	modeOutput string
}

func (runner *recordingRunner) Run(_ context.Context, name string, args ...string) error {
	runner.commands = append(runner.commands, recordedCommand{name: name, args: append([]string(nil), args...)})
	if len(args) >= 5 && args[4] == "has-session" && !runner.hasSession {
		return errors.New("session not found")
	}
	return nil
}

func (runner *recordingRunner) Output(_ context.Context, name string, args ...string) ([]byte, error) {
	runner.commands = append(runner.commands, recordedCommand{name: name, args: append([]string(nil), args...)})
	return []byte(runner.modeOutput), nil
}

func TestReturnToLiveOnlyCancelsTmuxCopyMode(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name       string
		modeOutput string
		wantCancel bool
	}{
		{name: "history", modeOutput: "1\n", wantCancel: true},
		{name: "live", modeOutput: "0\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			runner := &recordingRunner{modeOutput: test.modeOutput}
			manager, err := NewManager(Config{TmuxPath: "tmux", SocketName: "code-remote", SessionName: "prototype", AgentPath: "codex", WorkingDir: "/tmp"}, runner)
			if err != nil {
				t.Fatal(err)
			}
			if err := manager.ReturnToLive(context.Background()); err != nil {
				t.Fatal(err)
			}
			want := []recordedCommand{{name: "tmux", args: []string{"-L", "code-remote", "-f", "/dev/null", "display-message", "-p", "-t", "=prototype:0.0", "#{pane_in_mode}"}}}
			if test.wantCancel {
				want = append(want, recordedCommand{name: "tmux", args: []string{"-L", "code-remote", "-f", "/dev/null", "send-keys", "-X", "-t", "=prototype:0.0", "cancel"}})
			}
			if !reflect.DeepEqual(runner.commands, want) {
				t.Fatalf("commands = %#v, want %#v", runner.commands, want)
			}
		})
	}
}

func TestEnsureCreatesMissingTmuxSession(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{}
	manager, err := NewManager(Config{
		TmuxPath:    "/opt/homebrew/bin/tmux",
		SocketName:  "code-remote",
		SessionName: "prototype",
		AgentPath:   "/usr/local/bin/codex",
		WorkingDir:  "/tmp/project with spaces",
	}, runner)
	if err != nil {
		t.Fatal(err)
	}

	if err := manager.Ensure(context.Background(), 60, 30); err != nil {
		t.Fatal(err)
	}

	want := []recordedCommand{
		{name: "/opt/homebrew/bin/tmux", args: []string{"-L", "code-remote", "-f", "/dev/null", "has-session", "-t", "=prototype"}},
		{name: "/opt/homebrew/bin/tmux", args: []string{"-L", "code-remote", "-f", "/dev/null", "new-session", "-d", "-s", "prototype", "-x", "60", "-y", "30", "-c", "/tmp/project with spaces", "--", "/usr/bin/env", "-u", "NO_COLOR", "/usr/local/bin/codex"}},
		{name: "/opt/homebrew/bin/tmux", args: []string{"-L", "code-remote", "-f", "/dev/null", "set-option", "-g", "mouse", "on"}},
		{name: "/opt/homebrew/bin/tmux", args: []string{"-L", "code-remote", "-f", "/dev/null", "bind-key", "-T", "root", "WheelUpPane", "copy-mode", "-e", "-u"}},
	}
	if !reflect.DeepEqual(runner.commands, want) {
		t.Fatalf("commands = %#v, want %#v", runner.commands, want)
	}
}

func TestEnsureReusesExistingTmuxSession(t *testing.T) {
	t.Parallel()

	runner := &recordingRunner{hasSession: true}
	manager, err := NewManager(Config{
		TmuxPath:    "tmux",
		SocketName:  "code-remote",
		SessionName: "prototype",
		AgentPath:   "codex",
		WorkingDir:  "/tmp/project",
	}, runner)
	if err != nil {
		t.Fatal(err)
	}

	if err := manager.Ensure(context.Background(), 60, 30); err != nil {
		t.Fatal(err)
	}
	if len(runner.commands) != 3 {
		t.Fatalf("Ensure ran %d commands, want has-session plus touch scroll configuration", len(runner.commands))
	}
	wantMouse := recordedCommand{
		name: "tmux",
		args: []string{"-L", "code-remote", "-f", "/dev/null", "set-option", "-g", "mouse", "on"},
	}
	if !reflect.DeepEqual(runner.commands[1], wantMouse) {
		t.Fatalf("mouse configuration = %#v, want %#v", runner.commands[1], wantMouse)
	}
	wantWheelUp := recordedCommand{
		name: "tmux",
		args: []string{"-L", "code-remote", "-f", "/dev/null", "bind-key", "-T", "root", "WheelUpPane", "copy-mode", "-e", "-u"},
	}
	if !reflect.DeepEqual(runner.commands[2], wantWheelUp) {
		t.Fatalf("wheel-up configuration = %#v, want %#v", runner.commands[2], wantWheelUp)
	}
}

func TestNewManagerRejectsUnsafeIdentifiers(t *testing.T) {
	t.Parallel()

	_, err := NewManager(Config{
		TmuxPath:    "tmux",
		SocketName:  "../../other",
		SessionName: "prototype; kill-server",
		AgentPath:   "codex",
		WorkingDir:  "/tmp/project",
	}, &recordingRunner{})
	if err == nil {
		t.Fatal("NewManager succeeded with unsafe tmux identifiers")
	}
}

func TestAttachCommandDoesNotStartAgent(t *testing.T) {
	t.Parallel()

	manager, err := NewManager(Config{
		TmuxPath:    "tmux",
		SocketName:  "code-remote",
		SessionName: "prototype",
		AgentPath:   "codex",
		WorkingDir:  "/tmp/project",
	}, &recordingRunner{})
	if err != nil {
		t.Fatal(err)
	}

	name, args := manager.AttachCommand()
	wantArgs := []string{"-L", "code-remote", "-f", "/dev/null", "-u", "attach-session", "-t", "=prototype"}
	if name != "tmux" || !reflect.DeepEqual(args, wantArgs) {
		t.Fatalf("AttachCommand() = %q %#v, want %q %#v", name, args, "tmux", wantArgs)
	}
}

func TestOpenAttachmentStartsTemporaryPTYAtRequestedSize(t *testing.T) {
	t.Parallel()

	manager, err := NewManager(Config{
		TmuxPath:    "tmux",
		SocketName:  "code-remote",
		SessionName: "prototype",
		AgentPath:   "codex",
		WorkingDir:  "/tmp/project",
	}, &recordingRunner{})
	if err != nil {
		t.Fatal(err)
	}

	temporary, err := os.CreateTemp(t.TempDir(), "fake-pty")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = temporary.Close() })

	var gotName string
	var gotArgs []string
	var gotSize pty.Winsize
	manager.startPTY = func(command *exec.Cmd, size *pty.Winsize) (*os.File, error) {
		gotName = command.Args[0]
		gotArgs = append([]string(nil), command.Args[1:]...)
		gotSize = *size
		return temporary, nil
	}

	attachment, err := manager.OpenAttachment(context.Background(), 72, 28)
	if err != nil {
		t.Fatal(err)
	}
	if gotName != "tmux" {
		t.Fatalf("PTY command = %q, want tmux", gotName)
	}
	wantArgs := []string{"-L", "code-remote", "-f", "/dev/null", "-u", "attach-session", "-t", "=prototype"}
	if !reflect.DeepEqual(gotArgs, wantArgs) {
		t.Fatalf("PTY args = %#v, want %#v", gotArgs, wantArgs)
	}
	if gotSize.Cols != 72 || gotSize.Rows != 28 {
		t.Fatalf("PTY size = %#v, want 72x28", gotSize)
	}
	if err := attachment.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestOpenAttachmentClosesPreviousClient(t *testing.T) {
	t.Parallel()

	manager, err := NewManager(Config{
		TmuxPath:    "tmux",
		SocketName:  "code-remote",
		SessionName: "prototype",
		AgentPath:   "codex",
		WorkingDir:  "/tmp/project",
	}, &recordingRunner{})
	if err != nil {
		t.Fatal(err)
	}

	files := make([]*os.File, 2)
	for index := range files {
		files[index], err = os.CreateTemp(t.TempDir(), "fake-pty")
		if err != nil {
			t.Fatal(err)
		}
	}
	nextFile := 0
	manager.startPTY = func(_ *exec.Cmd, _ *pty.Winsize) (*os.File, error) {
		file := files[nextFile]
		nextFile++
		return file, nil
	}

	first, err := manager.OpenAttachment(context.Background(), 60, 30)
	if err != nil {
		t.Fatal(err)
	}
	second, err := manager.OpenAttachment(context.Background(), 60, 30)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })

	if _, err := first.Write([]byte("must be closed")); err == nil {
		t.Fatal("first attachment remained writable after replacement")
	}
}
