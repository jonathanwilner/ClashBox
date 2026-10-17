# Ultimate fold development build

Source: https://github.com/xiaobaigroup/ClashBox/tree/master at e036a6a3.
Private integration branch: jw/ultimate-dev-20261016.
Version: 1.7.6 (1007060). Bundle: org.xbgroup.clashboxLTS.
Target: ARM64; module declares phone, tablet, and 2in1.
Minimum and target SDK: HarmonyOS 6.1 API 23, using the existing Monaco SDK shim.

Build on Monaco with scripts/Build-JwRelease.ps1 in an isolated source directory.
The 16 October 2026 build directory is
C:\Users\jonathan\src\deveco\ClashBox-Ultimate-20261016.
Tracking session on Armstrong: build-clashbox-ultimate.

## Native engine provenance

Hvigor compiles ArkTS and the C++ interface. It packages the upstream prebuilt
proxy_core/libs/arm64-v8a/libflclash.so, SHA-256
651f88c622c4ecdc578243129940badce5a1468bfaabdc4dec01bc2f86949b3d.
The Go engine is not rebuilt by this task. Upstream references gvisor-ohos
73e918980a3e7d997aed8f9a8698059a8baa19a4, which its remote refuses as
"not our ref". This prevents a reproducible full native-engine source build.
The source archive excludes the unused Go source submodules.
The retained xb_components revision is 8c84b2ca7392819fdfcb04c28ca99e3ebff6037d.

## Signing and acceptance

Huawei developer authentication was restored on 16 October 2026. The account
uses de.devecostudio.huawei.com and connect-api-dre.cloud.huawei.com.
The China API returned misleading empty lists and rejected registration with
205390089 (developer region not match). The European API confirms the Ultimate
was already registered. Use --region de with the nixos-amd profile helper.
International accounts use the authenticated session team because the team-list
endpoint is China-only.

Use the protected Monaco signing helper from nixos-amd, verify-app, then
install and launch through HDC. Check VPN authorization and actual routing
before claiming the build works on the device. Publish the signed development
artifact only after verification; do not update the stable latest manifest
before physical acceptance. Preserve the existing installed app and its data.

## Build result

DevEco assembleApp succeeded on 16 October 2026 with exit code 0.
Published log: https://nix.thewilners.com/machines/armstrong/logs/build-clashbox-ultimate.txt
Unsigned HAP: /home/jonathan/artifacts/clashbox-ultimate-20261016/ClashBox-JW-1.7.6-arm64-unsigned.hap
Size: 120933347 bytes.
SHA-256: 9ee02787c2dd67908fa5e4c7291f78f40744a7e14d9c543fb82f4e268dd97378
Package metadata confirms version 1.7.6, versionCode 1007060, and 2in1 support.
ARM64 engine and interface libraries are present. The component dependency
also contributes two x86_64 libraries; this is not a complete x86_64 build.
The retained build output above is unsigned. The signed device-specific copy is described below.

## Reliable Huawei callback

Use `scripts/huawei-auth-callback.mjs AUTH_JSON STATUS_JSON de` on Monaco before
opening the DevEco authorization page with port 8888. Keep the process running.
The listener binds only to 127.0.0.1 and remains available for 30 minutes.
It exchanges the temporary token inside the callback request, accepts Huawei's
POST body, and supports a plain-text JWT response. It saves the session only
after the userInfo access token is present. Status output contains no tokens.
An existing auth file is backed up with a timestamp before replacement.
Use `cn` only when authorizing a mainland account. The callback's site field,
when provided, selects the exchange region.

The previous five-minute receiver timed out before login completed. Separating
callback capture from exchange also allowed short-lived tokens to expire.
The new flow was validated against Huawei: callback status authenticated and
certificate/device requests both returned Huawei code 0. Device registration,
signing, and installation remain pending the physical device connection.

## Signed Ultimate development download, 17 October 2026 UTC

Download: https://remote.thewilners.com/haps/ClashBox-JW-1.7.6-Ultimate-signed.hap

Huawei certificate: ClashBox Ultimate debug 20261017.
Certificate expiry: 17 October 2027 at 02:52:41 UTC.
The embedded profile contains exactly the user-supplied Ultimate UDID and
org.xbgroup.clashboxLTS. Private signing material remains on Monaco under
Downloads/hap-resign-work/clashbox-ultimate-20261017.

DevEco sign-app and verify-app passed. The extracted profile's CMS signature
passed OpenSSL verification with -noverify (content signature only; no separate
CA trust-store validation). Public download SHA-256 matches the local artifact:
bc54764d152fe8469dbeeb93c234c0b7136818f9968ceaa132b375760dff5b30.
Tracked session: sign-clashbox-ultimate, exit 0.

On the Ultimate, download the HAP and use the native Terminal with the existing
local HDC connection. Replace PORT and the downloaded file path:

```sh
hdc tconn 127.0.0.1:PORT
hdc -t 127.0.0.1:PORT install -r "/path/to/ClashBox-JW-1.7.6-Ultimate-signed.hap"
```

Installation, launch, VPN authorization, and traffic routing on the physical
Ultimate remain unverified. Do not uninstall the existing app to resolve a
signature mismatch: retain its data and investigate the installed signer first.
The stable latest manifest remains unchanged.
