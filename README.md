# awss

A fast, interactive AWS profile and region switcher. Like [kubectx](https://github.com/ahmetb/kubectx) for AWS.

> **Status:** Early development — profile listing works, interactive picker and shell integration coming soon.

## Usage

```bash
awss list              # list all profiles
awss <name>            # switch to named profile (coming soon)
awss -                 # switch to previous profile (coming soon)
awss -c                # print current profile and region (coming soon)
awss -r                # interactive region picker (coming soon)
```

## How it works

`awss` sets `AWS_PROFILE` (and optionally `AWS_REGION`) in your current shell via a thin shell wrapper. It does not resolve or cache credentials — the AWS SDK handles that transparently.

```bash
# Add to your .bashrc / .zshrc:
source <(awss init bash)   # or zsh/fish
```

## Install

```bash
go install github.com/juthrbog/awss@latest
```

## License

Apache-2.0
