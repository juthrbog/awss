# awss

A fast, interactive AWS profile and region switcher. Like [kubectx](https://github.com/ahmetb/kubectx) for AWS.

> **Status:** Early development. Profile and region picking, previous-profile switching, shell completions, and SSO login work.

## Usage

```bash
awss                   # fuzzy profile picker (requires shell integration to switch)
awss list              # list all profiles
awss <name>            # switch to named profile (requires shell integration)
awss login             # log in to IAM Identity Center and generate profiles
awss -                 # switch to previous profile
awss -c                # print current profile and region (--current)
awss -r                # interactive region picker (--region)
awss -r eu-west-1      # switch region directly
```

With terminal input, `awss` opens a fuzzy-filterable picker. Type to filter, use
↑/↓ to navigate, and Enter to select. The active `AWS_PROFILE` is marked
`(current)` and selected initially. Esc or Ctrl+C cancels without changing your
shell. With no profiles configured, the interactive command prints setup guidance
to stderr and leaves your shell unchanged.

With non-terminal input (for example, `awss < /dev/null`), bare `awss` lists
profiles as plain text. Use `awss list` to force listing even in a terminal.
Without shell integration, selecting a profile prints export statements rather
than changing your shell; the picker UI goes to stderr, keeping stdout safe for
`eval`.

## How it works

`awss` sets `AWS_PROFILE` (and optionally `AWS_REGION`) in your current shell via a thin shell wrapper. It does not resolve or cache credentials. The AWS SDK handles that transparently.

```bash
# Bash — add to ~/.bashrc:
eval "$(awss init bash)"

# Zsh — add to ~/.zshrc:
eval "$(awss init zsh)"

# Fish — add to ~/.config/fish/config.fish:
awss init fish | source
```

## Custom AWS file paths

Use `--config-file` and `--credentials-file` to choose AWS files without setting
environment variables first:

```bash
awss --config-file ./testdata/aws/config --credentials-file ./testdata/aws/credentials
awss list --config-file /path/to/config --credentials-file /path/to/credentials
awss production --config-file /path/to/config --credentials-file /path/to/credentials
```

Each file is resolved independently: **flag → AWS environment variable → default**.
The defaults are `~/.aws/config` and `~/.aws/credentials`; the environment variables
are `AWS_CONFIG_FILE` and `AWS_SHARED_CREDENTIALS_FILE`, respectively.

The flags work with the picker, listing, selection, profile completion, and SSO
login, and may appear before or after a subcommand. Listing and completion do not
change your environment. When you select a profile, shell integration also sets
the explicitly supplied file paths in your shell, so subsequent AWS CLI/SDK calls
use the same files. Relative flag paths are exported as absolute paths. Cancelling
the picker changes nothing.

Specifying a path does not copy or overwrite files. `awss login` still writes
profiles and updates managed credentials, but uses the selected paths instead of
the defaults. `--config-file` refers to the AWS INI file, **not** awss's own
`config.yaml` (which is controlled by `AWSS_CONFIG`).

## Previous profile and current state

`awss -` returns to the previously active profile; repeat it to toggle between
two profiles. Both named and interactive profile switches record the current
`AWS_PROFILE` before switching. History is shared across shell sessions in
`$XDG_CACHE_HOME/awss/previous` (default `~/.cache/awss/previous`). Selecting the
same profile, cancelling a picker, or switching regions leaves history alone.
If `AWS_PROFILE` is unset, there is no current profile to save.

Returning to a profile uses its configured region, not a previous manual region
override. Missing history or a deleted profile produces an error without changing
your shell.

`awss -c` / `awss --current` prints `AWS_PROFILE (AWS_REGION)` from the current
environment. Missing values appear as `<unset>`; it does not infer profile defaults.

## Region switching

`awss -r` / `awss --region` opens the same fuzzy picker for AWS regions and marks
the current `AWS_REGION`. Use `awss -r eu-west-1` (or `--region=eu-west-1`) to
switch directly, including with non-terminal input. The interactive form requires
a terminal. Selection only changes `AWS_REGION`, leaving `AWS_PROFILE` and
previous-profile history intact.

The region list is built in and includes commercial, China, GovCloud, and European
Sovereign Cloud regions. No AWS API call is needed, and selecting a region does
not enable it in your account.

## Shell completions

After loading shell integration, enable completion in your shell's startup file:

```bash
# Bash (~/.bashrc): install and load bash-completion first.
source <(awss completion bash)

# Zsh (~/.zshrc): run compinit first, unless your shell framework already does.
autoload -Uz compinit
compinit
source <(awss completion zsh)
```

```fish
# Fish (~/.config/fish/config.fish):
awss completion fish | source
```

`awss <TAB>` and `awss select <TAB>` complete profile names from your AWS files;
`awss -r <TAB>` completes regions. Completion does not change your shell or history.
After upgrading, reload both shell integration and completions (or start a new shell).

## SSO login

`awss login` runs the IAM Identity Center device login in your browser, then writes one profile per account and role into `~/.aws/config`. Each profile it writes carries `awss_managed = true`. Profiles without that key are never touched, so your hand-written profiles are safe. Managed profiles that no longer match a role are removed on the next login.

```bash
awss login --start-url https://example.awsapps.com/start
awss login --account prod --role AdministratorAccess    # filter what gets written
awss login --template '{{.Account.Name}}-{{.Role}}'      # control profile names
awss login --sts                                          # also write STS keys to ~/.aws/credentials
awss login --force                                        # log in again even if the token is valid
awss login --no-browser                                   # print the login URL instead
```

`--sts` is for tools that cannot read SSO profiles. It writes short-lived keys into `~/.aws/credentials` under the same profile names. Expired managed keys are cleaned up on the next login.

### Config file

To avoid typing the start URL, put your orgs in `~/.config/awss/config.yaml` (or `$XDG_CONFIG_HOME/awss/config.yaml`, or the path in `AWSS_CONFIG`):

```yaml
default_org: example
orgs:
  example:
    start_url: https://example.awsapps.com/start
    sso_region: us-east-1
    template: '{{.Account.Name | trimPrefix "example-"}}-{{.Role}}'
  sandbox:
    start_url: https://d-0123456789.awsapps.com/start
```

Then `awss login` uses the default org, and `awss login --org sandbox` picks another. Flags override config values. With one org configured, `default_org` can be left out.

### Profile name templates

Templates use Go `text/template` syntax. The default is `{{.Account.Name}}-{{.Role}}`.

| Variable | Meaning |
|---|---|
| `{{.Account.ID}}` | full AWS account number |
| `{{.Account.Name}}` | account name as shown in the SSO portal |
| `{{.ShortAccountID}}` | last four digits of the account number |
| `{{.Role}}` | role (permission set) name |

Functions: `trimPrefix "p"` strips a prefix, `replace "a" "b"` replaces all occurrences.

## Install

**Distribution status:** [v0.1.0](https://github.com/juthrbog/awss/releases/tag/v0.1.0)
is available as binary downloads and through Homebrew. AUR submission remains
manual; see [the release guide](docs/releasing.md).

### Go install / local build

Requires **Go 1.27.1 or newer**:

```bash
go install github.com/juthrbog/awss@latest
# Or, from a local checkout:
go install .
```

Ensure `$(go env GOBIN)` is on your `PATH`, or `$(go env GOPATH)/bin` if `GOBIN`
is unset. Check the installed build with `awss --version`.

### Homebrew (macOS and Linux)

```bash
brew install juthrbog/tap/awss
```

The formula installs the binary and bash/zsh/fish completions for Intel/amd64
and ARM64. Shell integration still needs to be enabled below.

### Arch Linux / AUR

After the `awss` AUR package has been submitted:

```bash
yay -S awss
```

Alternatively, download the release's `PKGBUILD`, `SRCINFO`, `awss.rb`, and
`packaging_checksums.txt` from the [Releases page](https://github.com/juthrbog/awss/releases).
Review the recipe, verify `sha256sum -c packaging_checksums.txt`, and run
`makepkg -si` as a regular user. It builds from a checksummed source archive and
installs the binary and completions. See [AUR release steps](docs/releasing.md#one-time-aur-setup-and-updates).

### Binary downloads

Download your archive and `checksums.txt` from the
[Releases page](https://github.com/juthrbog/awss/releases). Assets are named
`awss_<version>_<os>_<arch>.tar.gz` (`.zip` for Windows):

| OS | Architectures |
|---|---|
| `linux` | `amd64`, `arm64` |
| `darwin` (macOS) | `amd64` (Intel), `arm64` (Apple Silicon) |
| `windows` | `amd64`, `arm64` |

For example, on Linux (replace the version and architecture with your download):

```bash
archive=awss_0.1.0_linux_amd64.tar.gz
# Compare the downloaded archive against its entry in the checksum manifest.
grep "  ${archive}$" checksums.txt | sha256sum -c -
mkdir -p awss-download
tar -xzf "$archive" -C awss-download
mkdir -p "$HOME/.local/bin"
install -m 755 awss-download/awss "$HOME/.local/bin/awss"
```

On macOS use `shasum -a 256 -c -` instead of `sha256sum -c -`. Add
`$HOME/.local/bin` to your `PATH`. Keep the included `completions/` files if you
want to install them manually, or generate them using [Shell completions](#shell-completions).

On Windows, compare `(Get-FileHash .\\awss_<version>_windows_amd64.zip -Algorithm SHA256).Hash`
with the matching entry in `checksums.txt`, extract the ZIP, and add the directory
containing `awss.exe` to your user `PATH`. The Windows binary supports commands
such as listing and SSO login, but **PowerShell/CMD shell integration is not
implemented**. Use the Linux binary inside WSL for integrated profile switching.

### Enable shell integration

Installation alone cannot change your parent shell. Add the appropriate line to
your shell's startup file:

```bash
# ~/.bashrc
eval "$(awss init bash)"
# ~/.zshrc
eval "$(awss init zsh)"
```

```fish
# ~/.config/fish/config.fish
awss init fish | source
```

Start a new shell (or evaluate the corresponding line now), then run `awss`.
After upgrading, reload shell integration so it uses the new binary. Configure
AWS profiles first, or use the [local test fixtures](#development).

## Development

Reusable fake AWS files live in [`testdata/aws/config`](testdata/aws/config) and
[`testdata/aws/credentials`](testdata/aws/credentials). Both contain multiple
profiles with deliberately invalid `FAKE_KEY_*` / `FAKE_SECRET_*` credentials.
They are for local listing, picking, switching, and completion tests—not AWS API
calls or SSO authentication. Never put real credentials in these files.

| Profile | Region | Test case |
|---|---|---|
| `default` | `us-west-2` | Present in both files |
| `production` | `us-east-1` | Switch to a different region; present in both files |
| `staging` | None | Clears `AWS_REGION`; present in both files |
| `dev-only` | `ap-northeast-1` | Credentials-only profile and region fallback |
| `dev-sso` | `eu-west-1` | Config-only SSO profile; session block is not a profile |
| `cross-account` | `ap-southeast-1` | Config-only assume-role profile |

For a quick picker test with no environment-variable setup:

```bash
go run . --config-file ./testdata/aws/config --credentials-file ./testdata/aws/credentials
```

Or use the helper to copy these into a temporary directory and set
`AWS_CONFIG_FILE` / `AWS_SHARED_CREDENTIALS_FILE` without touching `~/.aws`:

```bash
# Bash or zsh, from the repository root:
eval "$(./scripts/dev-env.sh)"
go run . list
go run . select production                # prints exports; does not apply them
go run .                                 # test the picker
# With shell integration loaded, use awss / awss production / awss - instead.
eval "$(./scripts/dev-env.sh teardown)"    # unset file overrides and remove copies
```

You can also point those two environment variables directly at the fixture files,
but use temporary copies for commands that write to AWS files. The helper only
manages the file overrides; it does not restore an earlier profile, region, or
previous-profile history after testing.

The Go tests validate fixture discovery, regions, and exact fake credential
values. Gitleaks and Trivy secret scanning remain enabled for these files; no
fixture-wide secret-scanner exclusion is needed.

## License

Apache-2.0
