# awss

A fast, interactive AWS profile and region switcher. Like [kubectx](https://github.com/ahmetb/kubectx) for AWS.

## Install

Release binaries are on the [latest release page](https://github.com/juthrbog/awss/releases/latest).
You can also install with Homebrew, Go, or a `PKGBUILD` for Arch Linux. The
package is not on the AUR yet.

### Homebrew (macOS and Linux)

```bash
brew install juthrbog/tap/awss
```

The formula installs the binary and the bash, zsh, and fish completions. It
supports Intel/amd64 and ARM64. You must still
[set up shell integration](#set-up-shell-integration).

### Go install or local build

You need **Go 1.27.3 or newer**.

```bash
go install github.com/juthrbog/awss@latest
# Or, from a local checkout:
go install .
```

Put `$(go env GOBIN)` on your `PATH`. If `GOBIN` is not set, use
`$(go env GOPATH)/bin`. Run `awss --version` to see the installed build.

### Arch Linux

The `awss` package is not on the AUR yet. To build it from the release recipe:

1. From the [Releases page](https://github.com/juthrbog/awss/releases), download
   `PKGBUILD`, `SRCINFO`, `awss.rb`, and `packaging_checksums.txt`. You need
   `awss.rb` (the Homebrew formula) only because the checksum file lists it.
2. Run `sha256sum -c packaging_checksums.txt`.
3. Read the `PKGBUILD`.
4. As a regular user, run `makepkg -si`. The build needs Go 1.27.3 or newer.

The package builds from a checksummed source archive. It installs the binary and
the completions. For the maintainer steps, see
[AUR release steps](docs/releasing.md#one-time-aur-setup-and-updates).

### Binary downloads

From the [Releases page](https://github.com/juthrbog/awss/releases), download your
archive and `checksums.txt`. The archives have the name
`awss_<version>_<os>_<arch>.tar.gz` (`.zip` for Windows):

| OS | Architectures |
|---|---|
| `linux` | `amd64`, `arm64` |
| `darwin` (macOS) | `amd64` (Intel), `arm64` (Apple Silicon) |
| `windows` | `amd64`, `arm64` |

For example, on Linux (replace the version and architecture with your download):

```bash
archive=awss_0.1.1_linux_amd64.tar.gz
# Compare the downloaded archive against its entry in the checksum manifest.
grep "  ${archive}$" checksums.txt | sha256sum -c -
mkdir -p awss-download
tar -xzf "$archive" -C awss-download
mkdir -p "$HOME/.local/bin"
install -m 755 awss-download/awss "$HOME/.local/bin/awss"
```

On macOS, use `shasum -a 256 -c -` instead of `sha256sum -c -`. Add
`$HOME/.local/bin` to your `PATH`. The archive also has a `completions/`
directory. You can install those files yourself, or see
[Shell completions](#shell-completions).

On Windows, run this command in PowerShell. Then compare the result with the
matching line in `checksums.txt`:

```powershell
(Get-FileHash .\awss_<version>_windows_<arch>.zip -Algorithm SHA256).Hash
```

Extract the ZIP and add the directory that contains `awss.exe` to your user
`PATH`. On Windows, commands such as `awss list` and `awss login` work. Shell
integration for PowerShell and CMD does not exist. To switch profiles, use the
Linux binary in WSL.

## Set up shell integration

A program cannot change the environment of the shell that started it. Shell
integration is a small shell function, named `awss`, that runs the binary and
applies its output to your current shell. You need it to switch profiles and
regions.

Add the line for your shell to its startup file:

```bash
# Bash: add to ~/.bashrc
eval "$(awss init bash)"

# Zsh: add to ~/.zshrc
eval "$(awss init zsh)"
```

```fish
# Fish: add to ~/.config/fish/config.fish
awss init fish | source
```

Start a new shell, or run the line in your current shell. Then run `awss`.

The function calls the binary by its full path. After you upgrade awss, reload
shell integration or start a new shell.

If you have no AWS profiles yet, run `awss login --start-url <your start URL>`
(see [SSO login](#sso-login)) or `aws configure --profile <name>`.

## Usage

```bash
awss                   # fuzzy profile picker
awss list              # list all profiles
awss <name>            # switch to named profile
awss select <name>     # switch to named profile, also if the name is a subcommand
awss -                 # switch to previous profile
awss -c                # print current profile and region (--current)
awss -r                # interactive region picker (--region)
awss -r eu-west-1      # switch region directly
awss login             # log in to IAM Identity Center and generate profiles
awss init <shell>      # print shell integration for bash, zsh, or fish
awss completion <shell>  # print a completion script
awss --version         # print the version
```

Switching a profile or region needs [shell integration](#set-up-shell-integration).

If a profile has the same name as a subcommand (`list`, `select`, `login`,
`init`, `completion`, or `help`), `awss <name>` runs the subcommand. Use
`awss select <name>` or `awss -- <name>` instead.

### The picker

When you run `awss` in a terminal, it opens a picker that filters as you type.
Use ↑ and ↓ to move, and press Enter to select. The picker marks the active
`AWS_PROFILE` with `(current)` and selects it first. Press Esc or Ctrl+C to
cancel. Cancelling does not change your shell.

If you have no profiles, `awss` writes setup guidance to stderr. Your shell does
not change.

When input comes from a pipe or a file (for example, `awss < /dev/null`), bare
`awss` prints the profiles as plain text. Use `awss list` to print the list in a
terminal.

### Without shell integration

Without shell integration, a profile selection prints export statements and
does not change your shell. The picker writes to stderr, so you can use stdout
with `eval`. The output uses bash and zsh syntax. In fish, add `--shell fish`:

```fish
awss select production --shell fish | source
```

## What awss changes

`awss` changes environment variables in your current shell:

- It sets `AWS_PROFILE` to the profile that you select.
- It sets `AWS_REGION` to the region of the profile. If the profile has no
  region, it unsets `AWS_REGION`. A region that you set by hand does not stay.
- If you switch profiles or regions with `--config-file` or
  `--credentials-file`, it also sets `AWS_CONFIG_FILE` or
  `AWS_SHARED_CREDENTIALS_FILE`. See [Custom AWS file paths](#custom-aws-file-paths).

`awss` does not read or cache credentials. The AWS CLI and SDKs do that.

## Previous profile and current state

`awss -` goes back to the previous profile. Run it again to toggle between two
profiles.

Before each profile switch, by name or from the picker, `awss` saves the current
`AWS_PROFILE`. All shells share this history file:
`$XDG_CACHE_HOME/awss/previous` (default `~/.cache/awss/previous`). The history
does not change when you:

- Select the profile that is already active.
- Cancel the picker.
- Switch regions.
- Switch with no `AWS_PROFILE` set. In this case there is no profile to save.

When you go back to a profile, `awss` uses the region from the profile. It does
not bring back a region that you set by hand. If there is no history, or the
profile no longer exists, `awss -` prints an error and your shell does not
change.

`awss -c` (or `awss --current`) prints `AWS_PROFILE (AWS_REGION)` from the
current environment. A missing value shows as `<unset>`. The command does not
look up the default region of the profile. It does not accept `--config-file`
or `--credentials-file`.

## Region switching

`awss -r` (or `awss --region`) opens the same picker for AWS regions. It marks
the current `AWS_REGION`. The picker needs a terminal.

To switch to a region directly, run `awss -r eu-west-1` (or
`awss --region=eu-west-1`). This also works when input comes from a pipe or a
file. A region switch changes `AWS_REGION`, and the AWS file paths if you
give `--config-file` or `--credentials-file`. `AWS_PROFILE` and the
previous-profile history do not change.

The region list is built into awss. It includes the commercial, China, GovCloud,
and European Sovereign Cloud regions. awss does not call AWS to get the list.
Selecting a region does not enable it in your account.

## Custom AWS file paths

Use `--config-file` and `--credentials-file` to choose the AWS files. You do not
have to set environment variables first:

```bash
awss --config-file /path/to/config --credentials-file /path/to/credentials
awss list --config-file /path/to/config --credentials-file /path/to/credentials
awss production --config-file /path/to/config --credentials-file /path/to/credentials
```

awss finds each file separately, in this order: the flag, then the AWS
environment variable, then the default.

| File | Flag | Environment variable | Default |
|---|---|---|---|
| Config | `--config-file` | `AWS_CONFIG_FILE` | `~/.aws/config` |
| Credentials | `--credentials-file` | `AWS_SHARED_CREDENTIALS_FILE` | `~/.aws/credentials` |

The flags work with these commands, before or after the subcommand:

- The picker, `awss <name>`, `awss select`, and `awss -`.
- `awss -r` (the region picker and `awss -r <region>`).
- `awss list` and profile completion. These do not change your environment.
- `awss login`.

`awss -c` stops with an error if you give the flags, because it reads only the
environment.

When you select a profile or a region, shell integration also exports the file
paths that you gave. Later AWS CLI and SDK calls then use the same files. awss
exports relative paths as absolute paths. If you cancel a picker, nothing
changes.

The flags do not copy or overwrite files. `awss login` writes to the files that
you choose instead of the defaults. `--config-file` is the AWS config file. It
is not the awss `config.yaml` file. See [Config file](#config-file).

## Shell completions

If you installed with Homebrew or from the Arch `PKGBUILD`, the completions are
already installed. Skip this section.

Otherwise, first [set up shell integration](#set-up-shell-integration). Then add
the lines for your shell to its startup file:

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

`awss <TAB>` and `awss select <TAB>` complete profile names from your AWS files.
`awss -r <TAB>` completes regions. Completion does not change your shell or the
history. After you upgrade awss, reload shell integration and completions, or
start a new shell.

## SSO login

`awss login` logs in to AWS IAM Identity Center (SSO) with a device login in
your browser. Then it writes one profile for each account and role to your AWS
config file (default `~/.aws/config`).

- Each profile that awss writes has the key `awss_managed = true`. awss calls
  these managed profiles.
- awss never changes a profile without that key. Your hand-written profiles
  stay as they are.
- New profiles get `region` set to the SSO region. awss does not change
  `region` later, so you can edit it.

Each login replaces all managed profiles for the same start URL. awss removes a
managed profile when it no longer matches a role. This also applies when you
use `--account` or `--role`: awss removes the managed profiles outside the
filter. If no account or role matches, awss does not change the profiles in
your config file. It still removes expired managed keys for the start URL from
your credentials file.
Managed profiles for other start URLs do not change.

The login stops with an error if a generated profile name is the same as one of
these profiles:

- A hand-written profile.
- A managed profile for a different start URL.

With `--sts`, awss also checks the credentials file. A clash in either file
leaves both files unchanged. Rename your profile or change `--template`. The
SSO token that awss caches in `~/.aws/sso/cache` stays.

awss compares start URLs exactly, character for character. If you change the
start URL of an org, the managed profiles for the old URL stay, and they block
the next login. Delete those profiles from your AWS config file, and from your
credentials file if you use `--sts`. Then log in again.

```bash
awss login --start-url https://example.awsapps.com/start
awss login --sso-region eu-west-1                       # region of the Identity Center instance (default us-east-1)
awss login --account prod --account dev                 # only these account names
awss login --role AdministratorAccess                   # only these role names
awss login --template '{{.Account.Name}}-{{.Role}}'     # control profile names
awss login --sts                                        # also write STS keys to the credentials file
awss login --force                                      # log in again even if the token is valid
awss login --no-browser                                 # print the login URL instead
awss login --browser                                    # only open the start URL, write nothing
```

`--account` and `--role` match exact names. You can give each flag more than
once. Run `awss login --help` for the short flag names. In `awss login`, `-r` is
`--role`. In other commands, `-r` is `--region`.

`--sts` is for tools that cannot read SSO profiles. It writes short-lived keys
to your AWS credentials file (default `~/.aws/credentials`), with the same
profile names. On the next login, awss removes the managed keys that expired.

### Config file

To skip typing the start URL, put your orgs in the awss config file. awss looks
for it in this order:

1. The path in `AWSS_CONFIG`.
2. `$XDG_CONFIG_HOME/awss/config.yaml`.
3. `~/.config/awss/config.yaml`.

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

`awss login` uses the default org. `awss login --org sandbox` uses another org.
If you have only one org, you can leave out `default_org`.

`--sso-region` and `--template` override the values for the org.
`--start-url` is different: awss does not use any org from `config.yaml`, and
you cannot use it with `--org`. The file must still be valid YAML.

### Profile name templates

Templates use Go `text/template` syntax. The default is `{{.Account.Name}}-{{.Role}}`.

| Variable | Meaning |
|---|---|
| `{{.Account.ID}}` | full AWS account number |
| `{{.Account.Name}}` | account name as shown in the SSO portal |
| `{{.ShortAccountID}}` | last four digits of the account number |
| `{{.Role}}` | role (permission set) name |

Functions: `trimPrefix "p"` removes a prefix. `replace "a" "b"` replaces all occurrences.

Each template must give a different name for each account and role. If the same
role exists in more than one account, include `{{.Account.Name}}` or
`{{.Account.ID}}`. If two profiles get the same name, one overwrites the other
without a warning.

## Development

Fake AWS files for tests are in [`testdata/aws/config`](testdata/aws/config) and
[`testdata/aws/credentials`](testdata/aws/credentials). Both files have several
profiles with invalid `FAKE_KEY_*` and `FAKE_SECRET_*` credentials. Use them to
test listing, picking, switching, and completion. Do not use them for AWS API
calls or SSO login. Do not put real credentials in these files.

| Profile | Region | Test case |
|---|---|---|
| `default` | `us-west-2` | Present in both files |
| `production` | `us-east-1` | Switch to a different region; present in both files |
| `staging` | None | Clears `AWS_REGION`; present in both files |
| `dev-only` | `ap-northeast-1` | Credentials-only profile and region fallback |
| `dev-sso` | `eu-west-1` | Config-only SSO profile; session block is not a profile |
| `cross-account` | `ap-southeast-1` | Config-only assume-role profile |

To test the picker with no environment setup, run this from the repository root:

```bash
go run . --config-file ./testdata/aws/config --credentials-file ./testdata/aws/credentials
```

You can also use the helper script. It copies the files into a temporary
directory and sets `AWS_CONFIG_FILE` and `AWS_SHARED_CREDENTIALS_FILE`. It does
not touch `~/.aws`.

```bash
# Bash or zsh, from the repository root:
eval "$(./scripts/dev-env.sh)"
go run . list
go run . select production                # prints exports; does not apply them
go run .                                  # test the picker
# With shell integration loaded, use awss / awss production / awss - instead.
eval "$(./scripts/dev-env.sh teardown)"   # unset file overrides and remove copies
```

You can point the two environment variables directly at the fixture files. For
commands that write to AWS files, use temporary copies. The helper manages only
the file variables. It does not restore an earlier profile, region, or
previous-profile history after the test.

The Go tests make sure that awss finds the fixture profiles and regions and
reads the exact fake credential values. Gitleaks and Trivy secret scans still
run on these files. They need no exclusion for the fixtures.

## License

Apache-2.0
