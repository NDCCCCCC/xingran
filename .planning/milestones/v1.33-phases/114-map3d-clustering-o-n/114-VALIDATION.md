---
phase: 114
slug: map3d-clustering-o-n
status: approved
nyquist_compliant: true
wave_0_complete: false
created: 2026-09-12
---

# Phase 114 — Validation Strategy

> Per-phase validation contract for feedback sampling during execution.

---

## Test Infrastructure

| Property | Value |
|----------|-------|
| **Framework** | Vitest ^4.0.18（jsdom, globals: true, testTimeout 15s, maxWorkers 4） |
| **Config file** | `xingran-react-frontend/vitest.config.ts` |
| **Quick run command** | `cd xingran-react-frontend && npx vitest run src/pages/operations/building-spaces-3d` |
| **Full suite command** | `cd xingran-react-frontend && npx vitest run`（七 gate 另含 lint / type-check / build） |
| **Estimated runtime** | ~10s（目录级实测 2026-09-12） |

---

## Sampling Rate

- **After every task commit:** `npx vitest run src/pages/operations/building-spaces-3d`（~10s）
- **After every plan wave:** `npx vitest run` 全量 + `npm run lint` + `npm run type-check`
- **Before `/gsd:verify-work`:** Full suite must be green
- **Max feedback latency:** 15 seconds

---

## Per-Task Verification Map

| Task ID | Plan | Wave | Requirement | Threat Ref | Secure Behavior | Test Type | Automated Command | File Exists | Status |
|---------|------|------|-------------|------------|-----------------|-----------|-------------------|-------------|--------|
| 114-01-01 | 01 | 1 | MAP3D-01/02/03 | — | N/A（纯算法） | unit（参考实现对照+种子随机+边界+500ms 性能冒烟） | `npx vitest run src/pages/operations/building-spaces-3d/__tests__/cluster.test.ts` | ❌ W0 | ⬜ pending |
| 114-02-01 | 02 | 2 | MAP3D-01 | — | N/A | grep 断言（两组件各 ≤1 处 pointToOverlayPixel） | `grep -c "pointToOverlayPixel" HubeiMap.tsx HubeiMapGL.tsx` ≤1 each | ✅（命令即验证） | ⬜ pending |
| 114-02-02 | 02 | 2 | MAP3D-04 | — | N/A | 回归 + type-check（组件级自动化性价比低，如实标注） | `npx vitest run src/pages/operations/building-spaces-3d` + `npm run type-check` | ✅ | ⬜ pending |
| 114-03-01 | 03 | 3 | MAP3D-06 | — | N/A | grep 断言 + 全套件 + lint/build | `grep -r "BuildingMarkers\|CityMarkers" src/` 零命中；`grep -r "@uiw/react-baidu-map" src/` 零命中 | ✅（命令即验证） | ⬜ pending |
| 114-03-02 | 03 | 3 | MAP3D-05 | — | N/A | unit（cleanup spy：removeEventListener + 函数引用一致） | `npx vitest run src/pages/operations/building-spaces-3d/__tests__/hubei-map-cleanup.test.tsx` | ❌ W0 | ⬜ pending |
| — | — | — | MAP3D-02 | — | N/A | 人工（jsdom 无法测真实 WebGL 地图长任务） | 人工项见下表 | — | ⬜ pending |

*Status: ⬜ pending · ✅ green · ❌ red · ⚠️ flaky*

> 注：114-01-01 的单测仅承担 MAP3D-02 的自动侧兜底（500ms 性能冒烟 + 一致性对照）；MAP3D-02 的正式验收仍为下方人工项。

---

## Wave 0 Requirements

- [ ] `xingran-react-frontend/src/pages/operations/building-spaces-3d/__tests__/cluster.test.ts` — MAP3D-01/02/03（旧 O(n²) 参考实现对照 + 种子随机 + 边界 + 500ms 宽松性能冒烟）
- [ ] `xingran-react-frontend/src/pages/operations/building-spaces-3d/__tests__/hubei-map-cleanup.test.tsx` — MAP3D-05 cleanup spy 测试（Plan 03 Task 2 落地，独立文件）
- 测试基建无需安装（Vitest/jsdom/mock 体系已就绪；`src/test/utils/renderWithProviders` 可复用）

---

## Manual-Only Verifications

| Behavior | Requirement | Why Manual | Test Instructions |
|----------|-------------|------------|-------------------|
| n=1000 楼宇缩放/倾斜切换无秒级主线程长任务 | MAP3D-02 | BMapGL 为外部 WebGL SDK，jsdom 无法执行真实地图渲染；单测侧以合成 1000 点 + 宽松预算兜底（见 114-01-01 性能冒烟） | 开发机浏览器打开 building-spaces-3d 页（需 `VITE_BAIDU_MAP_AK`），导入/构造千级楼宇，连续缩放+倾斜，Performance 面板确认无 >1s 长任务，聚类视觉与改造前一致 |

---

## Validation Sign-Off

- [x] All tasks have `<automated>` verify or Wave 0 dependencies
- [x] Sampling continuity: no 3 consecutive tasks without automated verify
- [x] Wave 0 covers all MISSING references
- [x] No watch-mode flags
- [x] Feedback latency < 15s
- [x] `nyquist_compliant: true` set in frontmatter

**Approval:** approved（2026-09-12 checker 复核——六项 sign-off 全部满足）
