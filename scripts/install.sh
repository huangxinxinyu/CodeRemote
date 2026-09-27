#!/bin/sh

set -eu

usage() {
    cat <<'EOF'
Install Code Remote for the current macOS user.

Usage: sh scripts/install.sh [--source DIR] [--bin-dir DIR]

When run from a repository checkout, the installer builds that checkout.
When piped to sh, it downloads the latest main branch first.
The default destination is $HOME/.local/bin.
EOF
}

fail() {
    printf 'Code Remote install: %s\n' "$1" >&2
    exit 1
}

source_dir=
bin_dir=${HOME:?HOME is required}/.local/bin
temporary_source=
temporary_binary=

cleanup() {
    if [ -n "$temporary_binary" ]; then
        rm -f "$temporary_binary"
    fi
    if [ -n "$temporary_source" ]; then
        rm -rf "$temporary_source"
    fi
}

trap cleanup EXIT
trap 'exit 1' HUP INT TERM

while [ "$#" -gt 0 ]; do
    case "$1" in
        --source|--bin-dir)
            [ "$#" -ge 2 ] || fail "$1 requires a directory"
            case "$1" in
                --source) source_dir=$2 ;;
                --bin-dir) bin_dir=$2 ;;
            esac
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        *) fail "unknown option: $1" ;;
    esac
done

[ "$(uname -s)" = Darwin ] || fail 'this installer currently supports macOS only'

if [ -z "$source_dir" ]; then
    case "$0" in
        */scripts/install.sh)
            candidate_dir=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
            if [ -f "$candidate_dir/go.mod" ]; then
                source_dir=$candidate_dir
            fi
            ;;
    esac
fi

if [ -z "$source_dir" ]; then
    command -v git >/dev/null 2>&1 || fail 'Git is required to download the source'
    temporary_source=$(mktemp -d "${TMPDIR:-/tmp}/code-remote-install.XXXXXX")
    repo_url=${CODE_REMOTE_REPO_URL:-https://github.com/huangxinxinyu/CodeRemote.git}
    printf 'Downloading Code Remote source...\n'
    git clone --quiet --depth 1 "$repo_url" "$temporary_source/source"
    source_dir=$temporary_source/source
fi

source_dir=$(CDPATH= cd -- "$source_dir" && pwd) || fail 'source directory does not exist'
[ -f "$source_dir/go.mod" ] && [ -d "$source_dir/cmd/daemon" ] || fail 'source directory is not a Code Remote checkout'

if ! command -v go >/dev/null 2>&1 || ! command -v tmux >/dev/null 2>&1; then
    command -v brew >/dev/null 2>&1 || fail 'Homebrew is required to install missing Go or tmux; see https://brew.sh/'
    if ! command -v go >/dev/null 2>&1; then
        brew install go
    fi
    if ! command -v tmux >/dev/null 2>&1; then
        brew install tmux
    fi
fi

command -v go >/dev/null 2>&1 || fail 'Go was installed but is not on PATH'
command -v tmux >/dev/null 2>&1 || fail 'tmux was installed but is not on PATH'

mkdir -p "$bin_dir"
bin_dir=$(CDPATH= cd -- "$bin_dir" && pwd)
temporary_binary=$(mktemp "$bin_dir/.code-remote-daemon.XXXXXX")
printf 'Building Code Remote...\n'
(cd "$source_dir" && go build -trimpath -o "$temporary_binary" ./cmd/daemon)
chmod 755 "$temporary_binary"
mv -f "$temporary_binary" "$bin_dir/code-remote-daemon"
temporary_binary=

printf 'Installed %s\n' "$bin_dir/code-remote-daemon"
if ! command -v tailscale >/dev/null 2>&1 && [ ! -x /Applications/Tailscale.app/Contents/MacOS/Tailscale ]; then
    printf 'Next: install Tailscale on the Mac and iPhone, then join the same tailnet.\n'
fi
if ! command -v codex >/dev/null 2>&1 && ! command -v claude >/dev/null 2>&1; then
    printf 'Next: install and sign in to Codex CLI or Claude Code on this Mac.\n'
fi
printf 'Start the daemon with: %s -listen 127.0.0.1:8080 -agent codex -cwd /absolute/path/to/workspace\n' "$bin_dir/code-remote-daemon"
printf 'For private iPhone access, see the Tailscale setup in the README.\n'
