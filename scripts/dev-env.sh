#!/usr/bin/env bash
set -euo pipefail

# Teardown mode: unset env vars and remove temp dir.
if [[ "${1:-}" == "teardown" ]]; then
    if [[ -n "${AWS_CONFIG_FILE:-}" && "$AWS_CONFIG_FILE" == */awss-dev-* ]]; then
        rm -rf "$(dirname "$AWS_CONFIG_FILE")"
        echo "unset AWS_CONFIG_FILE"
        echo "unset AWS_SHARED_CREDENTIALS_FILE"
        if command -v gum &>/dev/null; then
            gum style --foreground 212 "awss dev environment removed." >&2
        else
            echo "awss dev environment removed." >&2
        fi
    else
        echo "No awss dev environment active." >&2
    fi
    exit 0
fi

# Setup mode: create fixture files and print exports.

# Reuse existing temp dir if already active.
if [[ -n "${AWS_CONFIG_FILE:-}" && "$AWS_CONFIG_FILE" == */awss-dev-* && -d "$(dirname "$AWS_CONFIG_FILE")" ]]; then
    tmpdir="$(dirname "$AWS_CONFIG_FILE")"
else
    tmpdir="$(mktemp -d /tmp/awss-dev-XXXXXX)"
fi

# Share the committed fixtures with the automated tests. Resolve relative to
# this script so setup also works when invoked outside the repository root.
script_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
cp "$script_dir/../testdata/aws/config" "$tmpdir/config"
cp "$script_dir/../testdata/aws/credentials" "$tmpdir/credentials"
chmod 600 "$tmpdir/config" "$tmpdir/credentials"

echo "export AWS_CONFIG_FILE=\"$tmpdir/config\""
echo "export AWS_SHARED_CREDENTIALS_FILE=\"$tmpdir/credentials\""

if command -v gum &>/dev/null; then
    gum style --foreground 212 "awss dev environment active ($tmpdir)" >&2
    gum style --faint 'Run: eval "$(./scripts/dev-env.sh teardown)" to deactivate' >&2
else
    echo "awss dev environment active ($tmpdir)" >&2
    echo 'Run: eval "$(./scripts/dev-env.sh teardown)" to deactivate' >&2
fi
