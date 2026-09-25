package terminal

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	directorypkg "github.com/huangxinxinyu/CodeRemote/internal/directory"
)

const maxCatalogSessions = 12

var (
	managedSessionIdentifier = regexp.MustCompile(`^session-[a-f0-9]{12}$`)
	nativeSessionIdentifier  = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{7,127}$`)
	// ErrSessionLimit is returned before creating more persistent agent sessions.
	ErrSessionLimit = errors.New("terminal session limit reached")
	// ErrInvalidWorkingDirectory is returned when a requested cwd cannot be used.
	ErrInvalidWorkingDirectory = errors.New("invalid working directory")
)

// Info is the product-owned metadata for one persistent agent terminal.
type Info struct {
	ID               string
	AgentID          string
	WorkingDirectory string
	Title            string
	CreatedAt        time.Time
}

// CatalogConfig configures a set of independent sessions sharing one agent and cwd.
type CatalogConfig struct {
	TmuxPath           string
	SocketName         string
	InitialSessionName string
	AgentID            string
	AgentPath          string
	WorkingDir         string
}

// CatalogRunner supports both tmux mutations and discovery.
type CatalogRunner interface {
	Runner
	Output(context.Context, string, ...string) ([]byte, error)
}

// Catalog owns the known Code Remote tmux sessions on one dedicated socket.
type Catalog struct {
	config   CatalogConfig
	runner   CatalogRunner
	mu       sync.RWMutex
	sessions map[string]*Manager
	infos    map[string]Info
}

// NewCatalog recovers managed tmux sessions and always exposes the initial session.
func NewCatalog(ctx context.Context, config CatalogConfig, runner CatalogRunner) (*Catalog, error) {
	if runner == nil {
		return nil, fmt.Errorf("terminal catalog runner is required")
	}
	if config.AgentID == "" {
		return nil, fmt.Errorf("terminal catalog agent id is required")
	}
	initial, err := newCatalogManager(config, config.InitialSessionName, runner)
	if err != nil {
		return nil, err
	}
	catalog := &Catalog{
		config:   config,
		runner:   runner,
		sessions: map[string]*Manager{config.InitialSessionName: initial},
		infos: map[string]Info{config.InitialSessionName: {
			ID:               config.InitialSessionName,
			AgentID:          config.AgentID,
			WorkingDirectory: config.WorkingDir,
		}},
	}
	catalog.recover(ctx)
	return catalog, nil
}

func newCatalogManager(config CatalogConfig, sessionID string, runner Runner) (*Manager, error) {
	return newCatalogManagerWithArgs(config, sessionID, nil, config.WorkingDir, runner)
}

func newCatalogManagerWithArgs(config CatalogConfig, sessionID string, agentArgs []string, workingDir string, runner Runner) (*Manager, error) {
	return NewManager(Config{
		TmuxPath:    config.TmuxPath,
		SocketName:  config.SocketName,
		SessionName: sessionID,
		AgentPath:   config.AgentPath,
		AgentArgs:   append([]string(nil), agentArgs...),
		WorkingDir:  workingDir,
	}, runner)
}

func (catalog *Catalog) recover(ctx context.Context) {
	args := []string{"-L", catalog.config.SocketName, "-f", "/dev/null", "list-sessions", "-F", "#{session_name}\t#{session_created}\t#{@code-remote-cwd}"}
	output, err := catalog.runner.Output(ctx, catalog.config.TmuxPath, args...)
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		id := fields[0]
		if id != catalog.config.InitialSessionName && !managedSessionIdentifier.MatchString(id) {
			continue
		}
		workingDir := catalog.config.WorkingDir
		if len(fields) >= 3 && strings.TrimSpace(fields[2]) != "" {
			workingDir = fields[2]
		}
		manager, managerErr := newCatalogManagerWithArgs(catalog.config, id, nil, workingDir, catalog.runner)
		if managerErr != nil {
			continue
		}
		createdUnix, _ := strconv.ParseInt(fields[1], 10, 64)
		createdAt := time.Time{}
		if createdUnix > 0 {
			createdAt = time.Unix(createdUnix, 0).UTC()
		}
		catalog.sessions[id] = manager
		catalog.infos[id] = Info{
			ID:               id,
			AgentID:          catalog.config.AgentID,
			WorkingDirectory: workingDir,
			CreatedAt:        createdAt,
		}
	}
}

// List returns stable metadata ordered from oldest to newest.
func (catalog *Catalog) List() []Info {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	catalog.refreshTitles(ctx)

	catalog.mu.RLock()
	defer catalog.mu.RUnlock()
	infos := make([]Info, 0, len(catalog.infos))
	for _, info := range catalog.infos {
		infos = append(infos, info)
	}
	sort.Slice(infos, func(left, right int) bool {
		if infos[left].CreatedAt.Equal(infos[right].CreatedAt) {
			return infos[left].ID < infos[right].ID
		}
		if infos[left].CreatedAt.IsZero() {
			return true
		}
		if infos[right].CreatedAt.IsZero() {
			return false
		}
		return infos[left].CreatedAt.Before(infos[right].CreatedAt)
	})
	return infos
}

func (catalog *Catalog) refreshTitles(ctx context.Context) {
	args := []string{"-L", catalog.config.SocketName, "-f", "/dev/null", "list-panes", "-a", "-F", "#{session_name}\t#{pane_title}"}
	output, err := catalog.runner.Output(ctx, catalog.config.TmuxPath, args...)
	if err != nil {
		return
	}

	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.SplitN(line, "\t", 2)
		if len(fields) != 2 {
			continue
		}
		info, ok := catalog.infos[fields[0]]
		if !ok {
			continue
		}
		info.Title = strings.TrimSpace(fields[1])
		catalog.infos[fields[0]] = info
	}
}

// Get resolves a known terminal without creating or ending any session.
func (catalog *Catalog) Get(id string) (Session, bool) {
	catalog.mu.RLock()
	defer catalog.mu.RUnlock()
	manager, ok := catalog.sessions[id]
	return manager, ok
}

// Delete ends one product-managed tmux session and removes it from the live
// catalog. Repeating the same deletion is safe and does not target arbitrary
// sessions outside this catalog.
func (catalog *Catalog) Delete(ctx context.Context, id string) error {
	catalog.mu.Lock()
	defer catalog.mu.Unlock()
	if _, ok := catalog.sessions[id]; !ok {
		return nil
	}

	args := []string{"-L", catalog.config.SocketName, "-f", "/dev/null", "kill-session", "-t", "=" + id}
	if err := catalog.runner.Run(ctx, catalog.config.TmuxPath, args...); err != nil {
		return fmt.Errorf("end terminal session %q: %w", id, err)
	}
	delete(catalog.sessions, id)
	delete(catalog.infos, id)
	return nil
}

// Create starts and registers a new independent agent session.
func (catalog *Catalog) Create(ctx context.Context, cols, rows uint16) (Info, error) {
	return catalog.CreateInDirectory(ctx, catalog.config.WorkingDir, cols, rows)
}

// CreateInDirectory starts a new independent agent session in a validated cwd.
func (catalog *Catalog) CreateInDirectory(ctx context.Context, workingDir string, cols, rows uint16) (Info, error) {
	canonical, err := directorypkg.Resolve(workingDir, catalog.config.WorkingDir)
	if err != nil {
		return Info{}, fmt.Errorf("%w: %v", ErrInvalidWorkingDirectory, err)
	}
	return catalog.create(ctx, "", nil, canonical, cols, rows)
}

// Resume starts the provider's native resume command in a new tmux terminal.
// The thread title is metadata returned by Codex itself, not inferred from TUI text.
func (catalog *Catalog) Resume(ctx context.Context, nativeSessionID, title string, cols, rows uint16) (Info, error) {
	return catalog.ResumeInDirectory(ctx, nativeSessionID, title, catalog.config.WorkingDir, cols, rows)
}

// ResumeInDirectory starts a native Codex thread in its selected workspace.
func (catalog *Catalog) ResumeInDirectory(ctx context.Context, nativeSessionID, title, workingDir string, cols, rows uint16) (Info, error) {
	if catalog.config.AgentID != "codex" {
		return Info{}, fmt.Errorf("native resume is not available for agent %q", catalog.config.AgentID)
	}
	if !nativeSessionIdentifier.MatchString(nativeSessionID) {
		return Info{}, fmt.Errorf("invalid native session id")
	}
	canonical, err := directorypkg.Resolve(workingDir, catalog.config.WorkingDir)
	if err != nil {
		return Info{}, fmt.Errorf("%w: %v", ErrInvalidWorkingDirectory, err)
	}
	return catalog.create(ctx, strings.TrimSpace(title), []string{"resume", nativeSessionID}, canonical, cols, rows)
}

func (catalog *Catalog) create(ctx context.Context, title string, agentArgs []string, workingDir string, cols, rows uint16) (Info, error) {
	catalog.mu.RLock()
	count := len(catalog.sessions)
	catalog.mu.RUnlock()
	if count >= maxCatalogSessions {
		return Info{}, ErrSessionLimit
	}

	var id string
	for attempt := 0; attempt < 8; attempt++ {
		random := make([]byte, 6)
		if _, err := rand.Read(random); err != nil {
			return Info{}, fmt.Errorf("generate terminal id: %w", err)
		}
		candidate := "session-" + hex.EncodeToString(random)
		catalog.mu.RLock()
		_, exists := catalog.sessions[candidate]
		catalog.mu.RUnlock()
		if !exists {
			id = candidate
			break
		}
	}
	if id == "" {
		return Info{}, fmt.Errorf("generate unique terminal id")
	}

	manager, err := newCatalogManagerWithArgs(catalog.config, id, agentArgs, workingDir, catalog.runner)
	if err != nil {
		return Info{}, err
	}
	if err := manager.Ensure(ctx, cols, rows); err != nil {
		return Info{}, err
	}
	metadataArgs := []string{"-L", catalog.config.SocketName, "-f", "/dev/null", "set-option", "-t", id, "@code-remote-cwd", workingDir}
	if err := catalog.runner.Run(ctx, catalog.config.TmuxPath, metadataArgs...); err != nil {
		return Info{}, fmt.Errorf("record terminal working directory: %w", err)
	}
	info := Info{
		ID:               id,
		AgentID:          catalog.config.AgentID,
		WorkingDirectory: workingDir,
		Title:            title,
		CreatedAt:        time.Now().UTC(),
	}
	catalog.mu.Lock()
	catalog.sessions[id] = manager
	catalog.infos[id] = info
	catalog.mu.Unlock()
	return info, nil
}
