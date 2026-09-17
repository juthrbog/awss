#!/usr/bin/env bash
set -euo pipefail

repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"
mkdir -p .generated/completions
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

# Generate on the build host; never try to execute a cross-compiled binary.
go build -o "$tmpdir/awss" .
"$tmpdir/awss" completion bash > .generated/completions/awss.bash
"$tmpdir/awss" completion zsh > .generated/completions/_awss
"$tmpdir/awss" completion fish > .generated/completions/awss.fish
