# Fold embedded Tailscale development build, 20 October 2026

Status: the visible Tailscale settings and VPN protection candidate is being
validated offline. Earlier candidates confirmed profile/controller startup after
IPC framing repair. Full embedded Tailscale enrollment, automatic VPN routing,
and the new settings page still require device acceptance. Root is restoring the
simpler MeshArc path while this candidate is completed.
## Source changes

ClashBox 2.1.0 was unavailable for this build. The retained candidate backports
the native changes into version `1.7.6-jw-ts` (`1007061`), bundle
`org.xbgroup.clashboxLTS`. The candidate includes the 32-byte TLS repair. Current installation and routed-service acceptance must be verified on the Fold.
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

## Native IPC readiness repair

The installed startup candidate logged `Connection refused` during UI interface
injection before the VPN core process started. Interface initialization now runs
inside `SocketStubService.startService`, after awaited native initialization,
IPC startup and stub listener setup, and before writing the ready lock. This
also covers background core startup. The premature UI initialization call is
removed. Interface injection checks socket readiness and retries up to twelve
times with 250 ms delays, retaining the final error and logging successful network
counts.

The UI core readiness check now requires the matching request lock and a live
native socket, with forty attempts separated by 250 ms. A missing lock returns
false. Device proof of the repaired startup sequence remains pending.

The signed readiness candidate is retained at
`/home/jonathan/artifacts/clashbox-fold-ipc-ready-20261020/ClashBox-JW-1.7.6-jw-ts-Ultimate-signed.hap`.
Its SHA-256 is
`72494b054c1a64e7eccc01b623f64ae26de842f191088e07a5630f641d08db96`.
Its native library hash is unchanged from the ACL candidate. Bytecode inspection
confirms both new success and bounded-retry failure diagnostic strings.
[Readiness package/sign job](https://nix.thewilners.com/machines/armstrong/logs/build-sign-clashbox-fold-ready.txt)
passed with protected signature verification.

## IPC stream framing repair

The installed readiness candidate reached native IPC, then logged method 28
(`SetNetInterfaces`) followed by `unexpected end of JSON input`. The native
handler read only once into a 10,240-byte buffer. A local stream socket may split
JSON at any byte; network-property payloads can also exceed that buffer. This
can reject interface initialization and block the selected profile from loading.

The handler now decodes one complete JSON document from the stream, bounded to
4 MiB and a ten-second read deadline. Empty connect-only readiness probes return
without error logs. Response framing and subscription behavior remain unchanged.
Regression tests cover method 28 with a string parameter containing encoded
network properties larger than 10 KiB, fragmented into 17-byte reads. Tests also
reject incomplete and oversized input and accept an empty readiness probe.

[IPC request regression test](https://nix.thewilners.com/machines/armstrong/logs/test-clashbox-fold-ipc-params.txt)
and [native library build](https://nix.thewilners.com/machines/armstrong/logs/build-clashbox-fold-ipc-framing.txt)
passed. The unstripped native library SHA-256 is
`401b94ba6357017afb1fe7706fd077cc9b35c11d6434cbd3eea7501ba266dbda`.
The signed HAP is retained at
`/home/jonathan/artifacts/clashbox-fold-ipc-framing-20261020/ClashBox-JW-1.7.6-jw-ts-Ultimate-signed.hap`.
Its SHA-256 is
`42c2d2c118171a91e142feb2e66e878c7927c3aa7d09a915b5300312dd743e3b`.
The packaged native SHA-256 is
`74ae6d20bfd4a7549a21367c437c0cb4f8769e294640ad5f7dff4313388bbcd1`.
[IPC framing package/sign job](https://nix.thewilners.com/machines/armstrong/logs/build-sign-clashbox-fold-ipc-framing.txt)
passed with protected `verify-app` success. The packaged library contains the new
request decoder. Device acceptance of this parser repair remains pending.

## Required device acceptance

Coordinate installation of the signed settings candidate below while preserving application data. Keep the working MeshArc recovery path until the cutover is explicitly coordinated. Confirm the process
stays alive after launch and profile load, with no new Faultlogger crash. Verify
Google and Baidu through the mixed port after restarting with the existing saved
profile. Activate an embedded Tailscale profile, confirm interface discovery and
registration, and test uncached DNS, HTTPS, tailnet destinations and the intended
China/US routing. Verify the single system VPN arrangement and browser traffic
path. Embedded Tailscale runtime success remains unconfirmed until these checks
complete.


## Visible Tailscale settings and protected VPN startup

The previous UI contained no Tailscale settings. Settings now has a visible
**Tailscale · 内置组网** entry. The page reads the active profile's existing node
name, hostname, state directory and control server. Node name and state directory
are read-only, retaining the enrolled identity and existing references. The page
provides a masked optional enrollment key, browser sign-in, connection status,
Tailscale addresses, peer count, health messages and an accept-subnet-routes switch.
The status RPC excludes keys and peer identities.

Apply preserves existing proxy passwords, Hysteria2 protocol settings, China DNS
and domain rules. It adds a direct embedded Tailscale node only when absent,
chains tailnet-addressed proxy servers through it when no different explicit
proxy chain exists, and prioritizes the tailnet CIDR rule. A first-write backup
and persistent per-profile settings remain in the app sandbox. Saved settings
are reapplied before profile loading, including after subscription updates.
Existing authentication keys remain unchanged when the enrollment field is blank.
A native profile-load error now raises an error instead of reporting success.

Native IPC response connections now remain open until an asynchronous reply, or
until the client disconnects for protect/log subscriptions. The prior unconditional
close invalidated the protect callback after StartTUN returned. Writes are serialized
and each protect event owns its response. The ArkTS receiver retains incomplete
frames across stream reads. A native ready acknowledgement confirms hook installation
before VPN startup completes. The selected profile is then reloaded to recreate
Tailscale sockets under the protected connection, including after VPN restart.

The offline checks include eleven tests of the production ArkTS methods and profile
helper: navigation visibility, enrollment/status responses, fragmented protect
messages, saved profile settings across subscription replacement, Hysteria2 and
China policy preservation, readiness ordering, and profile failure cleanup. JSON
serialization stands in for the YAML library in the profile helper tests. The
actual ArkTS compiler validates the page and imported production YAML helper.
Two Go core tests verify status forwarding through the real auto-close wrapper
and its availability after the real `ParseProxy` path. Native IPC framing and
asynchronous/streaming lifetime tests pass, including the Go race detector.

Full profile replacement now validates the new configuration before replacing the
live profile. It explicitly closes old Tailscale adapters before applying new
adapters, releasing the tsnet state lock and old unprotected sockets. The core's
DNS registration uses identity checks, so closing an old adapter preserves the
new adapter's resolver registration. The auto-close wrapper forwards the optional
status method. Profile load failures propagate through startup, clear the running
flag and stop the newly acquired VPN before reporting failure.

Device UI, interactive enrollment, VPN cutover and routed-service acceptance
remain pending. The source tests and signed package do not establish device
success. The Fold was not changed by this build lane.
