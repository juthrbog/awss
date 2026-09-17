# Releasing awss

## What is automated

- `.goreleaser.yaml` builds CGO-disabled Go binaries for Linux, macOS, and Windows,
  each on amd64 and arm64. Windows archives are ZIP; the others are tar.gz.
- Each binary archive contains the executable, README, Apache-2.0 license, and
  bash/zsh/fish completions. Releases also include a prefixed source archive and
  `checksums.txt` (SHA-256).
- `tools/package-release` verifies all seven archives against the checksum file,
  then generates `awss.rb`, `PKGBUILD`, `SRCINFO`, and
  `packaging_checksums.txt`. The Homebrew formula supports macOS and Linux, on
  Intel and ARM64. The AUR recipe builds from the checksummed source archive.
- A push of a `v*` tag runs `.github/workflows/release.yml`. Tests and lint must
  pass before GoReleaser creates a **draft** release. The workflow attaches and
  validates the package recipes, including a download/checksum round trip, before
  publishing the release.
- Stable releases update the Homebrew tap when its deploy key or token is configured.
  Prereleases do not change the stable formula. AUR submission is manual.
- Pushes to `main` and pull requests run build, vet, and race tests on Linux,
  macOS, and Windows; lint/security checks run on Linux. A separate Linux job
  builds a full GoReleaser snapshot with packaging smoke tests. Snapshots never
  publish and need no publishing secrets.

The package generator deliberately produces a traditional Homebrew formula,
not a cask, without relying on GoReleaser's deprecated `brews` integration.

## One-time Homebrew setup

The public [juthrbog/homebrew-tap](https://github.com/juthrbog/homebrew-tap)
repository is initialized, and a repository-scoped SSH deploy key has been
provisioned for publishing. The first stable release, `v0.1.0`, published the
initial formula.

For recreating or rotating this setup:

1. Initialize the public tap repository's `main` branch.
2. Generate a dedicated SSH key pair. Add the **public** key under the tap's
   **Settings → Deploy keys**, with **Allow write access** enabled. Store the
   **private** key as the `HOMEBREW_TAP_SSH_KEY` Actions secret in `juthrbog/awss`.
   Never commit either a private key or a token, or reuse a personal SSH key.
3. Make sure branch protection permits that key to update `Formula/awss.rb`, or
   adapt publication to use a reviewed tap PR.

Alternatively, use a fine-grained GitHub token restricted to the tap with
**Contents: read and write**, stored as `HOMEBREW_TAP_TOKEN` in `juthrbog/awss`.
The SSH key takes precedence if both are configured. The ordinary `GITHUB_TOKEN`
only publishes releases in `juthrbog/awss`; it cannot write to the separate tap.
With neither publishing credential configured, the release still publishes, but
its workflow emits a warning and leaves Homebrew publication pending.

The configured key is named **awss release publisher** in the tap's deploy-key
settings. To rotate it, replace the Actions secret with a new dedicated key and
remove the old deploy key from the tap after verifying publication. Deploy keys
do not automatically expire; revoke access when it is no longer needed.

After each stable release updates the tap, verify on macOS and Linux:

```bash
brew install juthrbog/tap/awss
brew test juthrbog/tap/awss
awss --version
```

You can alternatively copy the release's `awss.rb` into the tap as
`Formula/awss.rb` and commit it manually. Never publish a snapshot formula: its
URLs intentionally refer to a nonexistent snapshot release.

## One-time AUR setup and updates

The repository includes a PKGBUILD template; every release generates an actual
`PKGBUILD` and `SRCINFO` with its version, source URL, and SHA-256 checksum.
The release asset is named `SRCINFO` because GitHub rewrites leading-dot asset
names. After verifying checksums, copy it to `.SRCINFO` for AUR use; regenerate
it with `makepkg --printsrcinfo` after making any recipe changes.
No AUR account, SSH key, or automatic AUR push is configured here.

For a stable release, download `PKGBUILD`, `SRCINFO`, and
`packaging_checksums.txt` from that release. To verify the full recipe manifest,
also download `awss.rb`. In the download directory:

```bash
sha256sum -c packaging_checksums.txt
cp SRCINFO .SRCINFO
makepkg --verifysource
makepkg --cleanbuild --syncdeps
makepkg --printsrcinfo > .SRCINFO
```

Run `makepkg` as a regular user on Arch Linux. The recipe needs Go 1.27.1 or
newer and targets `x86_64` / `aarch64`. Review the generated `.SRCINFO` and test
installing the package before submitting/updating the `awss` AUR repository with
your own AUR account. Check that the name is available or coordinate with its
maintainer; do not overwrite someone else's package.

Only after that submission will `yay -S awss` be a supported install path.
Until then, users can build the release's PKGBUILD locally using `makepkg -si`.

## Local preflight (no publication)

Install GoReleaser **v2.18.2**, matching both workflow pins. From the repository:

```bash
go test -race ./...
go vet ./...
goreleaser check
goreleaser release --snapshot --clean
go run ./tools/package-release -dist dist
bash scripts/check-release.sh
```

The smoke test checks the native archive's version, fixture profile discovery,
completions, and Homebrew/PKGBUILD syntax. It uses Ruby and bash but does not
install anything, modify real AWS files, or contact AWS. Other targets are
cross-compiled, not executed by this check. Test Windows binaries on Windows
and test the PKGBUILD on Arch before claiming native runtime coverage.

GoReleaser's source archive comes from Git, so commit intended source changes
before validating that archive. Binary snapshot builds can include working-tree
changes. `dist/` and `.generated/` are ignored build outputs.

## Publish a release

After merging, confirming CI is green, and configuring the desired registries:

```bash
git switch main
git pull --ff-only
git tag -a v0.1.0 -m 'Release v0.1.0'  # example: choose the intended version
git push origin v0.1.0
```

Use `vMAJOR.MINOR.PATCH` tags; a suffix such as `-rc.1` makes a prerelease. The
package generator accepts these versions, converting hyphens to dots for Arch's
`pkgver`. Build metadata (`+...`) is not supported. Never retag a published
release: publish a new version instead.

Watch the Release workflow, confirm all archives and recipes are attached, and
verify their checksums. A failure before the final publish step leaves a draft
for inspection. A tap-update failure after publication does not remove the
GitHub release; repair the tap setup and update its formula from that release.

No signing/notarization credentials are configured. Checksums detect corruption,
but are not signatures. Binaries are not Apple-notarized or Windows Authenticode
signed; do not disable operating-system security controls to work around this.
