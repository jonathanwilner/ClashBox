# ClashBox

This fork produces the signed `ClashBox JW` HarmonyOS build used on Jonathan
Wilner's devices. It tracks upstream `master` and keeps device-specific fixes
as a small reviewable patch stack.

- Published builds: <https://remote.thewilners.com/haps/clashbox/>
- Update manifest: <https://remote.thewilners.com/haps/clashbox/latest.json>
- Upstream: <https://github.com/xiaobaigroup/ClashBox>

See [docs/JW_RELEASE.md](docs/JW_RELEASE.md) for the build, signing, rollout,
and rollback workflow. Signing keys, profiles, passwords, and built packages
are intentionally excluded from git.

#### 介绍

ClashBox是一个HarmonyOS NEXT(OpenHarmony)平台的代理软件，使用改版的ClashMate内核

注意：本仓库仅包含前端部分，改版的后端部分暂不开源

#### 食用方法

需要使用Auto-installer(https://github.com/likuai2010/auto-installer/)进行安装
