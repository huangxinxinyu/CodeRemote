package terminal

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

type catalogRunner struct {
	commands   []recordedCommand
	sessions   map[string]bool
	listOutput string
	paneOutput string
}

func (runner *catalogRunner) Run(_ context.Context, name string, args ...string) error {
	runner.commands = append(runner.commands, recordedCommand{name: name, args: append([]string(nil), args...)})
	if len(args) >= 7 && args[4] == "has-session" {
		id := strings.TrimPrefix(args[6], "=")
		if !runner.sessions[id] {
			return errors.New("session not found")
		}
	}
	if len(args) >= 9 && args[4] == "new-session" {
		for index, value := range args {
			if value == "-s" && index+1 < len(args) {
				runner.sessions[args[index+1]] = true
			}
		}
	}
	if len(args) >= 9 && args[4] == "set-option" && strings.HasPrefix(args[6], "=") {
		return errors.New("no such session")
	}
	return nil
}

func (runner *catalogRunner) Output(_ context.Context, _ string, args ...string) ([]byte, error) {
	if slices.Contains(args, "list-panes") {
		return []byte(runner.paneOutput), nil
	}
	return []byte(runner.listOutput), nil
}

func TestCatalogRecoversOnlyCodeRemoteSessions(t *testing.T) {
	t.Parallel()

	runner := &catalogRunner{
		sessions:   map[string]bool{"prototype": true, "session-a1b2c3d4e5f6": true},
		listOutput: "prototype\t1789959000\nsession-a1b2c3d4e5f6\t1789959100\nforeign\t1789959200\n",
	}
	catalog, err := NewCatalog(context.Background(), CatalogConfig{
		TmuxPath:           "tmux",
		SocketName:         "code-remote",
		InitialSessionName: "prototype",
		AgentID:            "codex",
		AgentPath:          "/usr/local/bin/codex",
		WorkingDir:         "/tmp/project",
	}, runner)
	if err != nil {
		t.Fatal(err)
	}

	infos := catalog.List()
	if len(infos) != 2 {
		t.Fatalf("List() returned %d sessions, want 2: %#v", len(infos), infos)
	}
	if _, ok := catalog.Get("prototype"); !ok {
		t.Fatal("prototype session was not recovered")
	}
	if _, ok := catalog.Get("session-a1b2c3d4e5f6"); !ok {
		t.Fatal("managed session was not recovered")
	}
	if _, ok := catalog.Get("foreign"); ok {
		t.Fatal("foreign tmux session entered the Code Remote catalog")
	}
}

func TestCatalogRefreshesNativePaneTitleWhenListing(t *testing.T) {
	t.Parallel()

	runner := &catalogRunner{
		sessions:   map[string]bool{"prototype": true},
		listOutput: "prototype\t1789959000\n",
	}
	catalog, err := NewCatalog(context.Background(), CatalogConfig{
		TmuxPath:           "tmux",
		SocketName:         "code-remote",
		InitialSessionName: "prototype",
		AgentID:            "codex",
		AgentPath:          "/usr/local/bin/codex",
		WorkingDir:         "/tmp/code-remote",
	}, runner)
	if err != nil {
		t.Fatal(err)
	}

	runner.paneOutput = "prototype\t问候用户 | code-remote\n"
	infos := catalog.List()
	if len(infos) != 1 {
		t.Fatalf("List() returned %d sessions, want 1", len(infos))
	}
	payload, err := json.Marshal(infos[0])
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err := json.Unmarshal(payload, &fields); err != nil {
		t.Fatal(err)
	}
	if got, _ := fields["Title"].(string); got != "问候用户 | code-remote" {
		t.Fatalf("List()[0].Title = %q, want native tmux pane title", got)
	}
}

func TestCatalogCreatesIndependentAgentSession(t *testing.T) {
	t.Parallel()

	workingDirectory := filepath.Join(t.TempDir(), "project with spaces")
	if err := os.Mkdir(workingDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	workingDirectory, err := filepath.EvalSymlinks(workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	runner := &catalogRunner{
		sessions:   map[string]bool{"prototype": true},
		listOutput: "prototype\t1789959000\n",
	}
	catalog, err := NewCatalog(context.Background(), CatalogConfig{
		TmuxPath:           "tmux",
		SocketName:         "code-remote",
		InitialSessionName: "prototype",
		AgentID:            "codex",
		AgentPath:          "/usr/local/bin/codex",
		WorkingDir:         workingDirectory,
	}, runner)
	if err != nil {
		t.Fatal(err)
	}

	created, err := catalog.Create(context.Background(), 64, 28)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`^session-[a-f0-9]{12}$`).MatchString(created.ID) {
		t.Fatalf("created terminal id = %q, want safe random session id", created.ID)
	}
	if created.AgentID != "codex" || created.WorkingDirectory != workingDirectory {
		t.Fatalf("created terminal = %#v, want configured agent and cwd", created)
	}
	if _, ok := catalog.Get(created.ID); !ok {
		t.Fatal("created session is not attachable")
	}
	if got := len(catalog.List()); got != 2 {
		t.Fatalf("List() returned %d sessions after create, want 2", got)
	}

	var createCommand string
	for _, command := range runner.commands {
		if strings.Contains(strings.Join(command.args, " "), " new-session ") {
			createCommand = strings.Join(command.args, " ")
		}
	}
	if createCommand == "" {
		t.Fatal("creating a catalog session did not create a tmux session")
	}
	for _, required := range []string{created.ID, workingDirectory, "/usr/bin/env -u NO_COLOR /usr/local/bin/codex"} {
		if !strings.Contains(createCommand, required) {
			t.Errorf("tmux create command %q does not contain %q", createCommand, required)
		}
	}
}

func TestCatalogDeletesOnlySelectedSessionIdempotently(t *testing.T) {
	t.Parallel()

	runner := &catalogRunner{
		sessions: map[string]bool{
			"prototype":            true,
			"session-a1b2c3d4e5f6": true,
		},
		listOutput: "prototype\t1789959000\nsession-a1b2c3d4e5f6\t1789959100\n",
	}
	catalog, err := NewCatalog(context.Background(), CatalogConfig{
		TmuxPath:           "tmux",
		SocketName:         "code-remote",
		InitialSessionName: "prototype",
		AgentID:            "codex",
		AgentPath:          "/usr/local/bin/codex",
		WorkingDir:         "/tmp/project",
	}, runner)
	if err != nil {
		t.Fatal(err)
	}

	if err := catalog.Delete(context.Background(), "session-a1b2c3d4e5f6"); err != nil {
		t.Fatal(err)
	}
	if _, ok := catalog.Get("session-a1b2c3d4e5f6"); ok {
		t.Fatal("deleted session remains attachable")
	}
	if _, ok := catalog.Get("prototype"); !ok {
		t.Fatal("deleting one session removed another session")
	}
	if err := catalog.Delete(context.Background(), "session-a1b2c3d4e5f6"); err != nil {
		t.Fatalf("repeated delete returned %v, want idempotent success", err)
	}

	var killCommands []recordedCommand
	for _, command := range runner.commands {
		if slices.Contains(command.args, "kill-session") {
			killCommands = append(killCommands, command)
		}
	}
	if len(killCommands) != 1 {
		t.Fatalf("kill commands = %#v, want one exact tmux deletion", killCommands)
	}
	want := []string{"-L", "code-remote", "-f", "/dev/null", "kill-session", "-t", "=session-a1b2c3d4e5f6"}
	if !slices.Equal(killCommands[0].args, want) {
		t.Fatalf("kill args = %#v, want %#v", killCommands[0].args, want)
	}
}

func TestCatalogCreatesSessionInSelectedWorkingDirectory(t *testing.T) {
	t.Parallel()

	defaultDirectory := t.TempDir()
	selectedDirectory := t.TempDir()
	canonicalSelectedDirectory, err := filepath.EvalSymlinks(selectedDirectory)
	if err != nil {
		t.Fatal(err)
	}
	runner := &catalogRunner{sessions: map[string]bool{"prototype": true}}
	catalog, err := NewCatalog(context.Background(), CatalogConfig{
		TmuxPath:           "tmux",
		SocketName:         "code-remote",
		InitialSessionName: "prototype",
		AgentID:            "codex",
		AgentPath:          "/usr/local/bin/codex",
		WorkingDir:         defaultDirectory,
	}, runner)
	if err != nil {
		t.Fatal(err)
	}

	created, err := catalog.CreateInDirectory(context.Background(), selectedDirectory, 64, 28)
	if err != nil {
		t.Fatal(err)
	}
	if created.WorkingDirectory != canonicalSelectedDirectory {
		t.Fatalf("created working directory = %q, want %q", created.WorkingDirectory, canonicalSelectedDirectory)
	}

	var createArgs, metadataArgs []string
	for _, command := range runner.commands {
		if slices.Contains(command.args, "new-session") {
			createArgs = command.args
		}
		if slices.Contains(command.args, "@code-remote-cwd") {
			metadataArgs = command.args
		}
	}
	if index := slices.Index(createArgs, "-c"); index < 0 || index+1 >= len(createArgs) || createArgs[index+1] != canonicalSelectedDirectory {
		t.Fatalf("tmux create args = %#v, want selected cwd after -c", createArgs)
	}
	if !slices.Contains(metadataArgs, canonicalSelectedDirectory) {
		t.Fatalf("tmux metadata args = %#v, want selected cwd", metadataArgs)
	}
}

func TestCatalogRejectsUnavailableWorkingDirectory(t *testing.T) {
	t.Parallel()

	defaultDirectory := t.TempDir()
	filePath := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(filePath, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	runner := &catalogRunner{sessions: map[string]bool{"prototype": true}}
	catalog, err := NewCatalog(context.Background(), CatalogConfig{
		TmuxPath:           "tmux",
		SocketName:         "code-remote",
		InitialSessionName: "prototype",
		AgentID:            "codex",
		AgentPath:          "/usr/local/bin/codex",
		WorkingDir:         defaultDirectory,
	}, runner)
	if err != nil {
		t.Fatal(err)
	}

	for _, path := range []string{filepath.Join(t.TempDir(), "missing"), filePath} {
		_, err := catalog.CreateInDirectory(context.Background(), path, 64, 28)
		if !errors.Is(err, ErrInvalidWorkingDirectory) {
			t.Errorf("CreateInDirectory(%q) error = %v, want ErrInvalidWorkingDirectory", path, err)
		}
	}
}

func TestCatalogRecoversSessionWorkingDirectoryMetadata(t *testing.T) {
	t.Parallel()

	defaultDirectory := t.TempDir()
	recoveredDirectory := t.TempDir()
	runner := &catalogRunner{
		sessions:   map[string]bool{"prototype": true, "session-a1b2c3d4e5f6": true},
		listOutput: "prototype\t1789959000\t\nsession-a1b2c3d4e5f6\t1789959100\t" + recoveredDirectory + "\n",
	}
	catalog, err := NewCatalog(context.Background(), CatalogConfig{
		TmuxPath:           "tmux",
		SocketName:         "code-remote",
		InitialSessionName: "prototype",
		AgentID:            "codex",
		AgentPath:          "/usr/local/bin/codex",
		WorkingDir:         defaultDirectory,
	}, runner)
	if err != nil {
		t.Fatal(err)
	}

	infos := catalog.List()
	for _, info := range infos {
		if info.ID == "session-a1b2c3d4e5f6" && info.WorkingDirectory != recoveredDirectory {
			t.Fatalf("recovered working directory = %q, want %q", info.WorkingDirectory, recoveredDirectory)
		}
	}
}

func TestCatalogResumesNativeCodexThreadInIndependentTerminal(t *testing.T) {
	t.Parallel()

	workingDirectory := t.TempDir()
	canonicalWorkingDirectory, err := filepath.EvalSymlinks(workingDirectory)
	if err != nil {
		t.Fatal(err)
	}
	runner := &catalogRunner{sessions: map[string]bool{}}
	catalog, err := NewCatalog(context.Background(), CatalogConfig{
		TmuxPath:           "tmux",
		SocketName:         "code-remote",
		InitialSessionName: "prototype",
		AgentID:            "codex",
		AgentPath:          "/usr/local/bin/codex",
		WorkingDir:         "/tmp/project",
	}, runner)
	if err != nil {
		t.Fatal(err)
	}

	info, err := catalog.ResumeInDirectory(context.Background(), "019abcde-1234-7000-8000-123456789abc", "重构移动端 UI", workingDirectory, 80, 24)
	if err != nil {
		t.Fatal(err)
	}
	if info.Title != "重构移动端 UI" {
		t.Fatalf("resumed title = %q, want native Codex title", info.Title)
	}
	if info.WorkingDirectory != canonicalWorkingDirectory {
		t.Fatalf("resumed cwd = %q, want %q", info.WorkingDirectory, canonicalWorkingDirectory)
	}

	var createCommand string
	for _, command := range runner.commands {
		if strings.Contains(strings.Join(command.args, " "), " new-session ") {
			createCommand = strings.Join(command.args, " ")
		}
	}
	for _, required := range []string{"/usr/local/bin/codex resume 019abcde-1234-7000-8000-123456789abc", info.ID, canonicalWorkingDirectory} {
		if !strings.Contains(createCommand, required) {
			t.Errorf("resume command %q does not contain %q", createCommand, required)
		}
	}
}
