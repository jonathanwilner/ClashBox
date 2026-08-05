# ClashBox JW release workflow

`master` mirrors the upstream source branch. `jw/harmony-current` carries the
maintained build and device fixes. Rebase that branch onto `upstream/master`
before each release and review every submodule pointer explicitly.

## Release contents

- Current upstream ClashBox application and prebuilt ARM64 proxy core.
- Current reviewed `xb_components` revision.
- HarmonyOS 6.1 API 23 ARM64 packaging using Monaco's isolated API 23 SDK
  metadata shim with DevEco Studio 26 tools.
- Correct notification helper argument ordering.
- China-reachable latency probe and Wilner-hosted update manifest.
- A monotonically increasing `versionCode` and numeric `versionName`.

The native proxy-core and gVisor source submodules are not advanced unless a
matching `libflclash.so` has been rebuilt and tested. A dirty or incomplete
native submodule is a release blocker.

## Build on Monaco

Use DevEco Studio 26 and its explicit tools. From the repository root:

```powershell
powershell -ExecutionPolicy Bypass -File .\scripts\Build-JwRelease.ps1
```

The script installs OHPM dependencies from Huawei's China registry and creates
an unsigned release HAP. It never reads or embeds signing credentials.

## Sign

Sign with the stable Wilner application key and a Huawei development profile
that authorizes every target device. Supply passwords through process
environment variables or a user-scoped secret store, never through git:

```powershell
$env:CLASHBOX_KEY_PASSWORD = '<from secret store>'
$env:CLASHBOX_STORE_PASSWORD = '<from secret store>'
powershell -ExecutionPolicy Bypass -File .\scripts\Sign-JwRelease.ps1 `
  -UnsignedHap '<path>' -OutputHap '<path>' -KeyAlias '<alias>' `
  -Certificate '<certificate.cer>' -Profile '<profile.p7b>' `
  -Keystore '<keystore.p12>'
```

The signer verifies the completed HAP and prints its SHA-256 digest.

## Rollout

1. Retain the last signed HAP as rollback material.
2. Install with `hdc install -r` on one device first.
3. Verify package metadata, launch, profile import, VLESS activation, IPv4 and
   IPv6 routing, DNS behavior, China/global split routing, and notifications.
4. Publish the HAP, `SHA256SUMS`, and `latest.json` under
   `https://remote.thewilners.com/haps/clashbox/`.
5. Roll the exact verified digest to the remaining authorized devices.

The bundle name stays `org.xbgroup.clashboxLTS` so the Wilner-signed build can
update the existing Wilner-signed installation without discarding app data.
Officially signed upstream packages are not signature-compatible with this
update path.
