package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"

	webui "github.com/huangxinxinyu/CodeRemote/internal/web"
)

type daemonConfig struct {
	listen           string
	agent            string
	workingDirectory string
	showVersion      bool
}

func parseConfig(args []string) (daemonConfig, error) {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return daemonConfig{}, fmt.Errorf("get current directory: %w", err)
	}

	var config daemonConfig
	flags := flag.NewFlagSet("code-remote-daemon", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&config.listen, "listen", "127.0.0.1:8080", "loopback or Tailscale listen address")
	flags.StringVar(&config.agent, "agent", "codex", "prototype agent: codex or claude")
	flags.StringVar(&config.workingDirectory, "cwd", workingDirectory, "agent working directory")
	flags.BoolVar(&config.showVersion, "version", false, "print version information")
	if err := flags.Parse(args); err != nil {
		return daemonConfig{}, err
	}
	if flags.NArg() != 0 {
		return daemonConfig{}, fmt.Errorf("unexpected positional arguments")
	}
	if config.showVersion {
		return config, nil
	}
	if config.agent != "codex" && config.agent != "claude" {
		return daemonConfig{}, fmt.Errorf("agent must be codex or claude")
	}
	if err := webui.ValidateListenAddress(config.listen); err != nil {
		return daemonConfig{}, err
	}

	config.workingDirectory, err = filepath.Abs(config.workingDirectory)
	if err != nil {
		return daemonConfig{}, fmt.Errorf("resolve working directory: %w", err)
	}
	info, err := os.Stat(config.workingDirectory)
	if err != nil {
		return daemonConfig{}, fmt.Errorf("inspect working directory: %w", err)
	}
	if !info.IsDir() {
		return daemonConfig{}, fmt.Errorf("working directory %q is not a directory", config.workingDirectory)
	}
	return config, nil
}
