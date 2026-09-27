#!/bin/sh

set -eu

repo_root=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
test_root=$(mktemp -d)
trap 'rm -rf "$test_root"' EXIT HUP INT TERM

bin_dir="$test_root/bin with spaces"
mkdir -p "$bin_dir"
printf '#!/bin/sh\nexit 42\n' > "$bin_dir/code-remote-daemon"
chmod +x "$bin_dir/code-remote-daemon"

sh "$repo_root/scripts/install.sh" --source "$repo_root" --bin-dir "$bin_dir" > "$test_root/install.log"
"$bin_dir/code-remote-daemon" -version > "$test_root/version.txt"
if ! grep -q '^code-remote-daemon ' "$test_root/version.txt"; then
    printf 'FAIL: installed executable did not report its version.\n'
    exit 1
fi

cat "$repo_root/scripts/install.sh" | sh -s -- --source "$repo_root" --bin-dir "$bin_dir" > "$test_root/reinstall.log"
"$bin_dir/code-remote-daemon" -version > /dev/null

cat "$repo_root/scripts/install.sh" | CODE_REMOTE_REPO_URL="file://$repo_root" sh -s -- --bin-dir "$test_root/from-clone" > "$test_root/clone.log"
"$test_root/from-clone/code-remote-daemon" -version > /dev/null

printf 'PASS: installer replaces an existing binary and builds from a piped source checkout.\n'
