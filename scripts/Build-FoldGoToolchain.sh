#!/usr/bin/env bash
set -euo pipefail
work="${CLASHBOX_NATIVE_WORK:-$HOME/src/clashbox-fold-native-20261020}"
script_dir="$(cd "$(dirname "$0")" && pwd)"
mkdir -p "$work"
cd "$work"
if [ ! -d go/src ]; then
  curl -fL --retry 2 https://codeload.github.com/jgowdy/go/tar.gz/1a087d05b5cf9573876b18812d8d5516f16bbe57 -o go-source.tar.gz
  mkdir -p go
  tar -xzf go-source.tar.gz --strip-components=1 -C go
fi
patch_python="$(command -v python3 || true)"
if [ -x /usr/bin/python3 ]; then patch_python=/usr/bin/python3; fi
"$patch_python" "$script_dir/patches/go-arm64-tls-preserve-registers.py" "$work/go"
if [ ! -x go/bin/go ]; then
  printf 'go1.27.0\n' > go/VERSION
  cd go/src
  if [ ! -x "$work/bootstrap/go/bin/go" ]; then
    mkdir -p "$work/bootstrap"
    curl -fL --retry 2 https://go.dev/dl/go1.25.0.darwin-arm64.tar.gz -o "$work/bootstrap.tar.gz"
    tar -xzf "$work/bootstrap.tar.gz" -C "$work/bootstrap"
  fi
  GOROOT_BOOTSTRAP="$work/bootstrap/go" GOTOOLCHAIN=local CGO_ENABLED=0 ./make.bash
fi
"$work/go/bin/go" version
