#!/usr/bin/env bash
set -euo pipefail

# Validate a snapshot without installing it or accessing real AWS files.
repo_dir="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_dir"
dist="${1:-dist}"
case "$(uname -s)" in
  Darwin) os=darwin ;;
  Linux) os=linux ;;
  *) echo 'Archive smoke test requires macOS or Linux.' >&2; exit 1 ;;
esac
case "$(uname -m)" in
  x86_64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) echo 'Unsupported smoke-test architecture.' >&2; exit 1 ;;
esac
archives=("$dist"/awss_*_"${os}_${arch}".tar.gz)
if [[ ${#archives[@]} -ne 1 || ! -f "${archives[0]}" ]]; then
  echo 'Expected exactly one native release archive.' >&2
  exit 1
fi
archive="${archives[0]}"
version="${archive##*/awss_}"
version="${version%_${os}_${arch}.tar.gz}"
tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

tar -xzf "$archive" -C "$tmpdir"
[[ "$("$tmpdir/awss" --version)" == "awss version $version" ]]
for file in LICENSE README.md completions/awss.bash completions/_awss completions/awss.fish; do
  test -s "$tmpdir/$file"
done
"$tmpdir/awss" list --config-file testdata/aws/config --credentials-file testdata/aws/credentials | grep -qx production
ruby -c "$dist/packages/awss.rb"
bash -n "$dist/packages/PKGBUILD"
echo "Release archive and package recipes passed smoke checks ($os/$arch)."
