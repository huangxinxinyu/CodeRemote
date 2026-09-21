#!/bin/sh

set -eu

repo_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
fake_bin=$(mktemp -d)
build_dir=$(mktemp -d)
generated_root_binary=0

cleanup() {
    rm -rf "$fake_bin" "$build_dir"
    if [ "$generated_root_binary" -eq 1 ]; then
        rm -f "$repo_root/daemon"
    fi
}

trap cleanup EXIT HUP INT TERM

for command_name in go tmux codex claude tailscale; do
    printf '#!/bin/sh\nexit 0\n' > "$fake_bin/$command_name"
    chmod +x "$fake_bin/$command_name"
done

if ! doctor_output=$(PATH="$fake_bin:$PATH" sh "$repo_root/scripts/doctor.sh" 2>&1); then
    printf '%s\n' "$doctor_output"
    printf 'FAIL: Web development prerequisites should pass with the declared tools.\n'
    exit 1
fi

printf 'PASS: Web development prerequisites use only the declared tools.\n'

bootstrap_plan=$(make -n -C "$repo_root" bootstrap)
if printf '%s\n' "$bootstrap_plan" | grep -q 'legacy-generate'; then
    printf '%s\n' "$bootstrap_plan"
    printf 'FAIL: Default bootstrap still generates the removed native client.\n'
    exit 1
fi

legacy_terms=$(printf '%s%s|%s(%s|%s)|%s/%s' 'x' 'code' 'swift' 'ui' 'term' 'apps' 'ios')
if rg --hidden -n -i "$legacy_terms" "$repo_root" --glob '!.git/**' --glob '!bin/**'; then
    printf 'FAIL: Repository still contains removed native-client assets or references.\n'
    exit 1
fi

if [ -d "$repo_root/apps" ]; then
    printf 'FAIL: Removed native-client directory still exists.\n'
    exit 1
fi

printf 'PASS: Repository excludes removed native-client tooling.\n'

if [ -e "$repo_root/daemon" ]; then
    printf 'FAIL: Cannot test build output while %s already exists.\n' "$repo_root/daemon"
    exit 1
fi

make -C "$repo_root" build BUILD_DIR="$build_dir" >/dev/null
if [ -e "$repo_root/daemon" ]; then
    generated_root_binary=1
    printf 'FAIL: Default build wrote a daemon binary to the repository root.\n'
    exit 1
fi

if [ ! -x "$build_dir/code-remote-daemon" ]; then
    printf 'FAIL: Default build did not create code-remote-daemon in BUILD_DIR.\n'
    exit 1
fi

printf 'PASS: Build output stays in the configured build directory.\n'
