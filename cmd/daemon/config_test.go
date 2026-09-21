package main

import "testing"

func TestParseConfigAcceptsPrototypeOptions(t *testing.T) {
	t.Parallel()

	workingDirectory := t.TempDir()
	config, err := parseConfig([]string{
		"-listen", "100.78.102.15:8080",
		"-agent", "claude",
		"-cwd", workingDirectory,
	})
	if err != nil {
		t.Fatal(err)
	}
	if config.listen != "100.78.102.15:8080" || config.agent != "claude" || config.workingDirectory != workingDirectory {
		t.Fatalf("parseConfig() = %#v", config)
	}
}

func TestParseConfigRejectsOrdinaryLANListener(t *testing.T) {
	t.Parallel()

	_, err := parseConfig([]string{"-listen", "192.168.1.20:8080", "-cwd", t.TempDir()})
	if err == nil {
		t.Fatal("parseConfig accepted an ordinary LAN listener")
	}
}

func TestParseConfigRejectsArbitraryCommand(t *testing.T) {
	t.Parallel()

	_, err := parseConfig([]string{"-agent", "/bin/sh", "-cwd", t.TempDir()})
	if err == nil {
		t.Fatal("parseConfig accepted an arbitrary agent command")
	}
}
