# Fold embedded Tailscale development build, 20 October 2026

Status: native compilation, release packaging, signing and binary inspection pass.
The corrected 32-byte TLS candidate launches and exposes the core API on the Fold.
A separate inbound access-control regression caused immediate proxy disconnects.
A runtime ACL patch restored Google and Baidu HTTPS with status 200. The durable
ACL candidate still requires installation and restart acceptance. Embedded
Tailscale registration and routing acceptance remain pending.

## Source changes

ClashBox 2.1.0 was unavailable for this build. The retained candidate backports
the native changes into version `1.7.6-jw-ts` (`1007061`), bundle
`org.xbgroup.clashboxLTS`. The 32-byte TLS repair is installed on the Fold.
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

## Previous TLS-only candidate

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

## Inbound access-control compatibility repair

The newer core rejects accepted mixed-port clients when `lan-allowed-ips` is
null. Older ClashBox settings omit this field. The wrapper passed the resulting
nil list to `inbound.SetAllowedIPs`, which closed connections before HTTP CONNECT
parsing. This produced immediate disconnects and no tracked proxy connections.

`defaultLanAllowedIPs` restores the core default (`0.0.0.0/0`, `::/0`) for nil
lists during raw profile mapping and listener refresh. Explicit empty lists and
custom prefix lists retain their policy. `allow-lan` and bind-address settings
continue to control listener exposure. The JSON regression test covers omitted,
null, explicit empty and custom prefix inputs, including IPv4 and IPv6 clients.

The runtime ACL patch has confirmed Google and Baidu HTTPS status 200 through
the Fold mixed port. The new signed candidate makes the fix durable across profile
reload and application restart. Candidate installation and restart verification
remain pending.

## Signed ACL candidate

Local artifact directory:
`/home/jonathan/artifacts/clashbox-fold-native-acl-20261020`.

| File | SHA-256 |
| --- | --- |
| `ClashBox-JW-1.7.6-jw-ts-Ultimate-signed.hap` | `306687454ee67e6f286834b6cd8879dc40d4fdc42d6225b12e9a19404feee84e` |
| `libflclash.so` before packaging | `a4fc86188a4aea2ab95aec0ff7ae46473a73966e717dea629bf37f7f6de4d0dc` |
| `libflclash-packaged.so` extracted from signed HAP | `10ce54492ec5498aef1f0bdccefeaa73e29035e8ba2b50e33a41caec1819e705` |

Metadata confirms `1.7.6-jw-ts` / `1007061`. The packaged library retains exactly
two corrected 32-byte TLS helper sequences and zero previous helper sequences.
The focused JSON ACL regression test, native build and protected package/sign job
passed. Published dashboard jobs:

- [ACL regression test](https://nix.thewilners.com/machines/armstrong/logs/test-clashbox-fold-inbound-acl.txt)
- [Native ACL build](https://nix.thewilners.com/machines/armstrong/logs/build-clashbox-fold-native-acl.txt)
- [ACL release package and protected signing](https://nix.thewilners.com/machines/armstrong/logs/build-sign-clashbox-fold-acl.txt)

## Interface bridge permission and startup repair

The isolated embedded Tailscale probe reported `HarmonyOS local link properties
have not been supplied`. Interface injection previously ran only before UI
profile loading or IP checks. A profile loaded through the HTTP API bypasses
those calls.

The entry source omitted `ohos.permission.GET_NETWORK_INFO`, which the installed
HarmonyOS 26 SDK requires for `getAllNets` and `getConnectionProperties`. Inspection
of the previous signed HAP's merged module metadata confirmed that a dependency
already supplied this permission. The entry omission alone therefore does not
explain the installed failure. The permission is now explicit in entry source.

The entry module now requests the permission. Interface injection runs when the
view model initializes and after a successful core restart, as well as before UI
profile loads. This supplies the cache for profiles loaded through the HTTP API.
Bridge failures now retain the error name, message and structured error in logs.
Device confirmation of interface population and Tailscale registration is pending.

The signed startup candidate is retained at
`/home/jonathan/artifacts/clashbox-fold-interface-startup-20261020/ClashBox-JW-1.7.6-jw-ts-Ultimate-signed.hap`.
Its SHA-256 is
`2b8f42a6f9960578fa08cfc154f0b1f6c61d79db970bfc5499801cc5f8fad5de`.
The packaged native hash remains
`10ce54492ec5498aef1f0bdccefeaa73e29035e8ba2b50e33a41caec1819e705`.
The signed HAP contains the updated diagnostic string and permission metadata.
[Startup bridge package/sign job](https://nix.thewilners.com/machines/armstrong/logs/build-sign-clashbox-fold-interface.txt)
passed with protected signature verification. Device acceptance is pending.

## Required device acceptance

Install the ACL candidate while preserving application data. Confirm the process
stays alive after launch and profile load, with no new Faultlogger crash. Verify
Google and Baidu through the mixed port after restarting with the existing saved
profile. Activate an embedded Tailscale profile, confirm interface discovery and
registration, and test uncached DNS, HTTPS, tailnet destinations and the intended
China/US routing. Verify the single system VPN arrangement and browser traffic
path. Embedded Tailscale runtime success remains unconfirmed until these checks
complete.
