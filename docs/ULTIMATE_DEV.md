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

Signing requires a current Huawei developer login and a profile authorizing
the exact physical device. The saved Monaco session failed refresh on
16 October 2026, and HDC listed no connected devices. A fresh Huawei login
page was opened in Monaco Chrome. Obtain the current wireless debugging
address, verify the device model and UDID privately, then issue the profile.
Never commit authentication snapshots, signing passwords, keys, or profiles.

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
The package remains unsigned and is not ready to install on a retail device.
