# awss

A fast, interactive AWS profile and region switcher. Like [kubectx](https://github.com/ahmetb/kubectx) for AWS.

> **Status:** Early development. Interactive profile picking, listing, switching, shell integration, and SSO login work.

## Usage

```bash
awss                   # fuzzy profile picker (requires shell integration to switch)
awss list              # list all profiles
awss <name>            # switch to named profile (requires shell integration)
awss login             # log in to IAM Identity Center and generate profiles
awss -                 # switch to previous profile (coming soon)
awss -c                # print current profile and region (coming soon)
awss -r                # interactive region picker (coming soon)
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

```bash
go install github.com/juthrbog/awss@latest
```

## Development

```bash
eval "$(./scripts/dev-env.sh)"           # create fixture AWS config and activate
go run . list                             # verify profiles
go run . select production                # test switching
eval "$(./scripts/dev-env.sh teardown)"   # deactivate and clean up
```

## License

Apache-2.0
