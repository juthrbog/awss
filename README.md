# awss

A fast, interactive AWS profile and region switcher. Like [kubectx](https://github.com/ahmetb/kubectx) for AWS.

> **Status:** Early development — profile listing, switching, and shell integration work. Interactive picker coming soon.

## Usage

```bash
awss list              # list all profiles
awss <name>            # switch to named profile (requires shell integration)
awss -                 # switch to previous profile (coming soon)
awss -c                # print current profile and region (coming soon)
awss -r                # interactive region picker (coming soon)
```

## How it works

`awss` sets `AWS_PROFILE` (and optionally `AWS_REGION`) in your current shell via a thin shell wrapper. It does not resolve or cache credentials — the AWS SDK handles that transparently.

```bash
# Bash — add to ~/.bashrc:
eval "$(awss init bash)"

# Zsh — add to ~/.zshrc:
eval "$(awss init zsh)"

# Fish — add to ~/.config/fish/config.fish:
awss init fish | source
```

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
