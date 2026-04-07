# AWS Profile Picker — Research

## Overview

Tool similar to [kubectx/kubens](https://github.com/ahmetb/kubectx) for switching AWS profiles and regions interactively.

---

## AWS Default Credential Chain

The SDK resolves credentials in this order, stopping at the first match:

1. **Environment variables** — `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`
2. **Web identity token** — `AWS_WEB_IDENTITY_TOKEN_FILE` + `AWS_ROLE_ARN` (used in EKS/IRSA)
3. **Shared config/credentials files** — `~/.aws/credentials` and `~/.aws/config` for the active profile
   - Sub-chain: web identity → SSO → role assumption via `source_profile` → role assumption via `credential_source` → `credential_process` → static keys
4. **ECS container credentials** — `AWS_CONTAINER_CREDENTIALS_RELATIVE_URI` / `AWS_CONTAINER_CREDENTIALS_FULL_URI`
5. **EC2 Instance Metadata (IMDS)** — `http://169.254.169.254` (IMDSv2 preferred)

### Key Environment Variables

#### Profile Selection

| Variable | Purpose | Notes |
|---|---|---|
| `AWS_PROFILE` | Select named profile | **Standard. Use this.** |
| `AWS_DEFAULT_PROFILE` | Legacy alias for `AWS_PROFILE` | CLI v1 only. Avoid. |
| `AWS_SHARED_CREDENTIALS_FILE` | Override credentials file path | Default: `~/.aws/credentials` |
| `AWS_CONFIG_FILE` | Override config file path | Default: `~/.aws/config` |

#### Region Selection

| Variable | Purpose | Notes |
|---|---|---|
| `AWS_REGION` | AWS region for requests | **Standard. Use this.** |
| `AWS_DEFAULT_REGION` | CLI-specific region | Lower precedence than `AWS_REGION` |

Region resolution order:
1. `--region` CLI flag
2. `AWS_REGION`
3. `AWS_DEFAULT_REGION`
4. `region` in active profile in `~/.aws/config`

#### Credentials

| Variable | Purpose |
|---|---|
| `AWS_ACCESS_KEY_ID` | Access key ID |
| `AWS_SECRET_ACCESS_KEY` | Secret access key |
| `AWS_SESSION_TOKEN` | Session token (temporary credentials) |
| `AWS_ROLE_ARN` | Role ARN for web identity / assume-role |
| `AWS_ROLE_SESSION_NAME` | Session name for assumed role |
| `AWS_WEB_IDENTITY_TOKEN_FILE` | Path to OIDC token file |

#### Container / EC2

| Variable | Purpose |
|---|---|
| `AWS_CONTAINER_CREDENTIALS_RELATIVE_URI` | Relative URI for ECS credential endpoint |
| `AWS_CONTAINER_CREDENTIALS_FULL_URI` | Full URI for credential endpoint |
| `AWS_CONTAINER_AUTHORIZATION_TOKEN` | Auth token for container credentials |
| `AWS_EC2_METADATA_DISABLED` | Set `true` to skip IMDS |

#### Other

| Variable | Purpose |
|---|---|
| `AWS_MAX_ATTEMPTS` | Max retry attempts |
| `AWS_RETRY_MODE` | `standard`, `legacy`, or `adaptive` |
| `AWS_ENDPOINT_URL` | Global custom endpoint |
| `AWS_CA_BUNDLE` | Custom CA certificate bundle path |

### Config File Format

**`~/.aws/credentials`** — credential settings only:
```ini
[default]
aws_access_key_id = AKIA...
aws_secret_access_key = secret...

[production]
aws_access_key_id = AKIA...
aws_secret_access_key = secret...
```

**`~/.aws/config`** — everything else (and optionally credentials):
```ini
[default]
region = us-west-2
output = json

[profile production]
region = us-east-1
output = text

[profile dev-sso]
sso_session = my-sso
sso_account_id = 111122223333
sso_role_name = DeveloperAccess
region = us-west-2

[profile cross-account]
role_arn = arn:aws:iam::999999999999:role/AdminRole
source_profile = production
region = eu-west-1

[sso-session my-sso]
sso_start_url = https://myorg.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access
```

**Critical difference:** Named profiles use `[myprofile]` in credentials but `[profile myprofile]` in config. The `[default]` section uses no prefix in either file.

When the same credential keys exist in both files for the same profile, the credentials file takes precedence.

### SSO Profiles

#### Recommended format (refreshable token):
```ini
[profile my-sso-profile]
sso_session = my-sso
sso_account_id = 111122223333
sso_role_name = ReadOnly
region = us-west-2

[sso-session my-sso]
sso_start_url = https://myorg.awsapps.com/start
sso_region = us-east-1
sso_registration_scopes = sso:account:access
```

- Token cached at `~/.aws/sso/cache/` keyed by session name
- Supports automatic token refresh
- Multiple profiles can share one `[sso-session]` block

#### Legacy format (non-refreshable):
```ini
[profile my-legacy-sso]
sso_start_url = https://myorg.awsapps.com/start
sso_region = us-east-1
sso_account_id = 111122223333
sso_role_name = ReadOnly
```

- Fixed 8-hour session, no auto-refresh

### Assume Role Profiles

#### Via source_profile (most common):
```ini
[profile cross-account]
role_arn = arn:aws:iam::123456789012:role/RoleName
source_profile = base-credentials
external_id = optional-external-id
mfa_serial = arn:aws:iam::111111111111:mfa/username
duration_seconds = 3600
```

#### Via credential_source (EC2/ECS/Lambda):
```ini
[profile container-role]
role_arn = arn:aws:iam::123456789012:role/RoleName
credential_source = EcsContainer  # or Environment, Ec2InstanceMetadata
```

`source_profile` and `credential_source` are mutually exclusive.

---

## Existing Tools

### kubectx/kubens (reference design)

- **Stars:** ~19.6K
- **Language:** Originally bash (~254 lines), rewritten in Go (v0.9.0, 2020)
- **Mechanism:** Modifies `~/.kube/config` directly (the `current-context` field)
- **Rewrite motivations:** Cross-platform support, maintainability, contributor pool, plugin distribution (krew), 8-15x performance improvement
- **UX patterns:**
  - No args + TTY + fzf → interactive picker
  - No args + no TTY → list contexts
  - `kubectx <name>` → switch
  - `kubectx -` → switch to previous (state stored in `$XDG_CACHE_HOME/kubectx`)
  - `kubectx -c` → print current
  - `kubectx -s <name>` → isolated sub-shell
- **Architecture:** Clean `Op` interface pattern, `cobra`-style flag parsing, `kyaml` for YAML manipulation, `go-isatty` for TTY detection

### AWS Profile Switchers

| Tool | Language | Stars | Approach | Limitations |
|---|---|---|---|---|
| **aws-vault** | Go | ~9K | Security vault. `exec <profile> -- <cmd>` spawns subprocess with temp STS creds | Not a switcher — requires prefixing every command. No interactive picker. |
| **granted** | Go | ~1.7K | Shell wrapper exports env vars. Frecency ranking. Browser console access. | Requires shell alias setup. |
| **awsume** | Python | ~23 | Shell eval trick. MFA session caching. Plugin system. | Python dependency. Installation issues. |
| **awsp** (johnnyopao) | Node.js | ~380 | Interactive list, sets `AWS_PROFILE` via shell wrapper | Node.js dependency. Only sets `AWS_PROFILE`. |
| **awsp** (antonbabenko) | Bash | ~90 | Minimal bash + fzf | No SSO handling. Fragile. |

### How Third-Party Tools Inject Credentials

**aws-vault:** Stores long-term keys in OS keychain. `exec` spawns subprocess with `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `AWS_REGION`, `AWS_VAULT`, `AWS_CREDENTIAL_EXPIRATION`. Optional `--server` flag runs local IMDS server for auto-refresh.

**granted:** Shell alias wraps `assumego` binary. By default exports `AWS_PROFILE`. With `--export-all-env-vars` / `-x` exports full credential env vars. `-unset` flag clears them.

**awsume:** Sources into current shell. Exports `AWS_ACCESS_KEY_ID`, `AWS_SECRET_ACCESS_KEY`, `AWS_SESSION_TOKEN`, `AWS_SECURITY_TOKEN` (legacy), `AWS_REGION`, `AWS_DEFAULT_REGION`, `AWSUME_PROFILE`, `AWSUME_EXPIRATION`. When profile uses `credential_source`, exports `AWS_PROFILE` instead.

---

## The Parent Shell Problem

A child process cannot modify its parent shell's environment. Three patterns:

### Pattern 1: Shell wrapper + eval (recommended)
Binary prints `export` statements to stdout. A thin shell function sources the output.

```bash
# Shell wrapper (~15 lines)
awss() {
  local output
  output=$(command awss select "$@")
  if [ $? -eq 0 ]; then
    eval "$output"
  fi
}
```

**Pros:** Session-scoped, stays in current shell, proven (Granted, awsume)
**Cons:** One-time shell setup required

### Pattern 2: Subprocess exec
`tool exec <profile> -- <command>` spawns a new shell with env vars injected.

**Pros:** No shell setup needed
**Cons:** Always in a nested shell, confusing shell state

### Pattern 3: Modify config file directly
Write to `~/.aws/config` (like kubectx writes to kubeconfig).

**Pros:** No shell wrapper needed, works everywhere
**Cons:** Global mutation — affects ALL terminal sessions. No `current-profile` field in AWS config (unlike kubeconfig's `current-context`).

---

## Language Decision

### Recommendation: Go

| Factor | Go | Rust | Bash+Gum | Python |
|---|---|---|---|---|
| Distribution | Single static binary, trivial cross-compile | Single binary, harder cross-compile | Requires gum installed | Distribution nightmare |
| TUI | Charm (bubbletea/bubbles/lipgloss) — best-in-class | ratatui — excellent but overkill | `gum filter` — one liner | textual — slow startup |
| AWS SDK | v2, mature, production-grade | Less mature | N/A (manual INI parsing) | boto3 — great but ~100ms import |
| Startup | ~5ms | ~5ms | ~50ms | ~200ms |
| Shell completions | cobra generates free | clap generates free | Manual | Manual |
| Reference impl | Granted proves the pattern | None | Fragile for complex configs | awsume has install issues |

### Why Go wins:
1. **Charm ecosystem v2** — `charm.land/bubbles/v2` list component gives fuzzy-filterable interactive picker out of the box
2. **Single binary** — `GOOS=darwin GOARCH=arm64 go build`, no runtime deps
3. **Granted proves the architecture** — Go + shell wrapper + AWS SDK v2
4. **cobra** — CLI framework with free shell completion generation
5. **~5ms startup** — feels instant
6. **AWS SDK v2 config** — handles SSO, assume-role chains, credential process, all config/credentials file parsing (no standalone INI parser needed)

---

## Proposed Architecture

```
awss (Go binary)
├── cmd/                        # cobra commands
│   ├── root.go                 # default: interactive picker
│   ├── select.go               # select profile (outputs export statements)
│   ├── list.go                 # list profiles
│   ├── current.go              # print current profile/region
│   └── region.go               # select region
├── internal/
│   ├── config/                 # AWS config/credentials file parsing
│   ├── picker/                 # bubbletea interactive picker
│   └── state/                  # previous profile state ($XDG_CACHE_HOME/awss, fallback ~/.cache/awss)
├── shell/                      # shell wrapper scripts
│   ├── awss.bash
│   ├── awss.zsh
│   └── awss.fish
└── main.go

UX patterns (mirroring kubectx):
  awss                          # interactive picker (fuzzy filter)
  awss <name>                   # switch to named profile
  awss -                        # switch to previous profile
  awss -c / --current           # print current profile and region
  awss -l / --list              # list all profiles
  awss -r / --region            # interactive region picker
  awss -r <region>              # switch region directly
```

### Shell wrapper approach:
- Go binary prints `export AWS_PROFILE=x` and `export AWS_REGION=y` to stdout
- Shell wrapper function evals the output in the parent shell
- Ship wrappers for bash, zsh, fish
- User adds `source <(awss init bash)` to `.bashrc` (self-installing wrapper)

### Key libraries:
- `github.com/spf13/cobra` v1.10.x — CLI framework
- `charm.land/bubbletea/v2` — TUI framework (v2; vanity import replaces `github.com/charmbracelet/bubbletea`)
- `charm.land/bubbles/v2` — TUI components incl. list with filtering (v2; vanity import)
- `charm.land/lipgloss/v2` — TUI styling (v2; vanity import)
- `github.com/aws/aws-sdk-go-v2/config` — AWS config/credentials file loading (eliminates need for standalone INI parser)
- `github.com/mattn/go-isatty` v0.0.20 — TTY detection (may be unnecessary if bubbletea v2 handles this internally)
