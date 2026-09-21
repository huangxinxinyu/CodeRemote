// Command daemon runs the private Web UI and bridges one prototype agent
// terminal through a persistent tmux session.
package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/huangxinxinyu/CodeRemote/internal/buildinfo"
	"github.com/huangxinxinyu/CodeRemote/internal/terminal"
	webui "github.com/huangxinxinyu/CodeRemote/internal/web"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "code-remote-daemon: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	config, err := parseConfig(args)
	if err != nil {
		return err
	}
	if config.showVersion {
		fmt.Printf("code-remote-daemon %s\n", buildinfo.Version)
		return nil
	}

	tmuxPath, err := exec.LookPath("tmux")
	if err != nil {
		return fmt.Errorf("find tmux: %w", err)
	}
	agentPath, err := exec.LookPath(config.agent)
	if err != nil {
		return fmt.Errorf("find %s: %w", config.agent, err)
	}

	manager, err := terminal.NewManager(terminal.Config{
		TmuxPath:    tmuxPath,
		SocketName:  "code-remote",
		SessionName: "prototype",
		AgentPath:   agentPath,
		WorkingDir:  config.workingDirectory,
	}, terminal.ExecRunner{})
	if err != nil {
		return err
	}

	listener, err := net.Listen("tcp", config.listen)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", config.listen, err)
	}
	defer listener.Close()

	server := &http.Server{
		Handler: webui.NewHandlerWithContext(webui.NewAttachHandler(manager), webui.RuntimeContext{
			AgentID:          config.agent,
			WorkingDirectory: config.workingDirectory,
			TerminalID:       "prototype",
		}),
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	shutdownFinished := make(chan struct{})
	go func() {
		defer close(shutdownFinished)
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	fmt.Printf("Code Remote listening on http://%s (agent=%s, cwd=%s)\n", config.listen, config.agent, config.workingDirectory)
	err = server.Serve(listener)
	if errors.Is(err, http.ErrServerClosed) {
		<-shutdownFinished
		return nil
	}
	return err
}
