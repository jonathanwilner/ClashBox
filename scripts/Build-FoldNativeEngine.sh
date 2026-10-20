#!/usr/bin/env bash
set -euo pipefail
work="${CLASHBOX_NATIVE_WORK:-$HOME/src/clashbox-fold-native-20261020}"
project="${CLASHBOX_PROJECT_ROOT:-$(cd "$(dirname "$0")/.." && pwd)}"
cd "$project/proxy_core/src/flclash"
if [ ! -f core/go.mod ]; then
 curl -fL --retry 2 https://codeload.github.com/xfz347/ClashBox-Core/tar.gz/1cecba5c30b33c40e88fa0056ee674ad27a63fc3 -o "$work/core.tar.gz"
 mkdir -p core; tar -xzf "$work/core.tar.gz" --strip-components=1 -C core
 patch -p1 -d core < "$project/scripts/patches/core-harmonyos-interfaces.patch"
fi
if ! grep -q 'func (p \*autoCloseProxyAdapter) EmbeddedStatus' core/adapter/outbound/tailscale_status.go 2>/dev/null; then
 rm -f core/adapter/outbound/tailscale_status.go core/adapter/outbound/tailscale_status_test.go core/adapter/tailscale_status_test.go
 patch -p1 -d core < "$project/scripts/patches/core-tailscale-settings-status.patch"
fi
if [ ! -f gvisor-ohos/go.mod ]; then
 curl -fL --retry 2 https://codeload.github.com/xfz347/ClashBox-gvisor-ohos/tar.gz/4dd36187f7714e0da03017394a566062ca8594d8 -o "$work/gvisor.tar.gz"
 mkdir -p gvisor-ohos; tar -xzf "$work/gvisor.tar.gz" --strip-components=1 -C gvisor-ohos
fi
native="$HOME/Library/Huawei/Sdk/HarmonyOS-26.0.0/openharmony/native"
export PATH="$work/go/bin:$PATH" GOTOOLCHAIN=local
export GOOS=linux GOARCH=arm64 CGO_ENABLED=1
export CC="$native/llvm/bin/clang" CXX="$native/llvm/bin/clang++"
export CGO_CFLAGS="--target=aarch64-linux-ohos --sysroot=$native/sysroot -Wno-error"
export CGO_LDFLAGS="--target=aarch64-linux-ohos --sysroot=$native/sysroot"
go build -mod=mod -buildmode=c-shared -tags 'ohos with_gvisor' -o "$work/libflclash.so" .
shasum -a 256 "$work/libflclash.so"
