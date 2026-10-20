# Fold embedded Tailscale development build, 20 October 2026

Status: native compilation, release packaging, signing and binary inspection pass.
Physical device acceptance is pending. The latest signed candidate has not been
validated on the Fold because wireless HDC transport became unavailable. Earlier
candidates crashed during native startup. Do not promote this candidate as a
working or recommended release until the device checks below pass.

## Source changes

ClashBox 2.1.0 was unavailable for this build. The retained candidate backports
the native changes into version `1.7.6-jw-ts` (`1007061`), bundle
`org.xbgroup.clashboxLTS`. The 32-byte TLS repair has not been installed on the
Fold.
The HarmonyOS bridge supplies local interface names, indices, MTUs and addresses
from `connection.getAllNets()` and `getConnectionProperties()` before loading a
profile. The former neighbor-table implementation described remote neighbors.
The core patch routes the embedded Tailscale interface enumeration through this
cache under the `ohos` build tag, avoiding Linux netlink enumeration. The wrapper
also follows the current core HTTP/statistics APIs and preserves idle cleanup.
Device confirmation of the netlink permission repair remains pending.

Pinned native sources:

- Go fork `jgowdy/go`: `1a087d05b5cf9573876b18812d8d5516f16bbe57`.
- `xfz347/ClashBox-Core`: `1cecba5c30b33c40e88fa0056ee674ad27a63fc3`.
- `xfz347/ClashBox-gvisor-ohos`: `4dd36187f7714e0da03017394a566062ca8594d8`.

The build scripts fetch those revisions and apply the retained patches. Native
compilation uses the HarmonyOS 26 ARM64 Clang/sysroot on Dropshot. HAP packaging
uses the existing Monaco DevEco 26 project and protected signing helper. Signing
credentials and device-specific profiles remain outside this repository.

## ARM64 runtime TLS repair

The fork's `runtime.load_g` and `runtime.save_g` TLS descriptor helpers saved
registers R1/LR in R10/R11, violating the helper register preservation contract.
They also called the Fold's musl TLS descriptor resolver without protecting the
Go caller's frame pointer slot at SP-8. The resolver pushes x1/x2 below SP.

The first stack-based repair reserved 16 bytes and stored LR at old SP-8. Fresh
Fold fault data showed x29 pointing into `_cgo_topofstack` instructions. The next
C wrapper read instruction bytes through that corrupted frame pointer and
crashed. Fault record: `cppcrash-org.xbgroup.clashboxLTS-20220238-20261020183440`.

The corrected helpers reserve **32 bytes**, save R1 at old SP-32 and LR at old
SP-24, and restore both registers and SP. Old SP-8 stays untouched. The musl
resolver can push below the adjusted SP. The Python source patch accepts the
original implementation and first repair and is idempotent for the final form.
Disassembly confirms the corrected reservation/restoration in both helpers.

## Exact retained candidate

Local artifact directory:
`/home/jonathan/artifacts/clashbox-fold-native-tls32-20261020`.

| File | SHA-256 |
| --- | --- |
| `ClashBox-JW-1.7.6-jw-ts-Ultimate-signed.hap` | `8bc94b47197fbfff9b02450b9abef2b030a43b528bc72ed8982925068c100398` |
| `libflclash.so` before packaging | `fbd6b92915248b22910653b04fd3957b6481e399d05b3c01896c9da036a4d368` |
| `libflclash-packaged.so` extracted from signed HAP | `ee2d8e5a472a75115eb1f8db7131f9d8af87bc711cd95a23c5293cc7df74e03c` |

Monaco's signer hash and the retrieved HAP hash match. DevEco strips native debug
information during packaging, so the packaged library hash differs from the
native build output. The packaged library contains exactly two corrected TLS
helper instruction sequences, zero first-repair sequences, and zero original
R10/R11 save sequences. The signed HAP metadata confirms the intended version.

Tracked jobs, each with exit code 0:

- [Native build](https://nix.thewilners.com/machines/armstrong/logs/build-clashbox-fold-native-tls32.txt)
- [Release package and protected signing](https://nix.thewilners.com/machines/armstrong/logs/build-sign-clashbox-fold-tls32.txt)

Both succeeded entries were verified in the published dashboard manifest.
The HarmonyOS interface-cache unit test passes on the Dropshot host:
[test-clashbox-fold-anet](https://nix.thewilners.com/machines/armstrong/logs/test-clashbox-fold-anet.txt).
This host test verifies cache behavior; device interface API behavior remains
pending. Shell syntax, patch migration/idempotency and whitespace checks pass.

## Required device acceptance

Restore wireless HDC first. Install the retained signed candidate while
preserving application data. Confirm the process stays alive after launch and
profile load, with no new Faultlogger crash. Activate an embedded Tailscale
profile, confirm interface discovery and registration, and test uncached DNS,
HTTPS, tailnet destinations and the intended China/US routing. Verify the single
system VPN arrangement and browser traffic path. Runtime success remains
unconfirmed until these checks complete.
