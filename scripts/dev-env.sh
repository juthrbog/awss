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

cat > "$tmpdir/config" <<'EOF'
[default]
region = us-west-2

[profile production]
region = us-east-1
output = json

[profile staging]
region = eu-west-1

[profile dev-sso]
sso_session = my-sso
sso_account_id = 111122223333
sso_role_name = ReadOnly
region = us-east-1

[profile cross-account]
role_arn = arn:aws:iam::123456789012:role/CrossAccountAdmin
source_profile = production
region = ap-southeast-1

[sso-session my-sso]
sso_start_url = https://myorg.awsapps.com/start
sso_region = us-east-1
EOF

cat > "$tmpdir/credentials" <<'EOF'
[default]
aws_access_key_id = FAKE_KEY_DEFAULT
aws_secret_access_key = FAKE_SECRET_DEFAULT

[production]
aws_access_key_id = FAKE_KEY_PRODUCTION
aws_secret_access_key = FAKE_SECRET_PRODUCTION

[dev-only]
aws_access_key_id = FAKE_KEY_DEV_ONLY
aws_secret_access_key = FAKE_SECRET_DEV_ONLY
EOF

echo "export AWS_CONFIG_FILE=\"$tmpdir/config\""
echo "export AWS_SHARED_CREDENTIALS_FILE=\"$tmpdir/credentials\""

if command -v gum &>/dev/null; then
    gum style --foreground 212 "awss dev environment active ($tmpdir)" >&2
    gum style --faint 'Run: eval "$(./scripts/dev-env.sh teardown)" to deactivate' >&2
else
    echo "awss dev environment active ($tmpdir)" >&2
    echo 'Run: eval "$(./scripts/dev-env.sh teardown)" to deactivate' >&2
fi
