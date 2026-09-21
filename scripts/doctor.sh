#!/bin/sh

set -u

missing=0

check_command() {
    label="$1"
    command_name="$2"
    if command_path=$(command -v "$command_name" 2>/dev/null); then
        printf 'ok      %-12s %s\n' "$label" "$command_path"
    else
        printf 'missing %-12s %s\n' "$label" "$command_name"
        missing=1
    fi
}

check_command "Go" "go"
check_command "tmux" "tmux"
check_command "Codex" "codex"
check_command "Claude" "claude"

if tailscale_path=$(command -v tailscale 2>/dev/null); then
    printf 'ok      %-12s %s\n' "Tailscale" "$tailscale_path"
elif [ -x /Applications/Tailscale.app/Contents/MacOS/Tailscale ]; then
    printf 'ok      %-12s %s\n' "Tailscale" "/Applications/Tailscale.app/Contents/MacOS/Tailscale"
else
    printf 'missing %-12s %s\n' "Tailscale" "install the macOS client and join the target tailnet"
    missing=1
fi

if [ "$missing" -ne 0 ]; then
    printf '\nInstall the missing prerequisites described in docs/development.md.\n'
    exit 1
fi

printf '\nDevelopment prerequisites are present.\n'
