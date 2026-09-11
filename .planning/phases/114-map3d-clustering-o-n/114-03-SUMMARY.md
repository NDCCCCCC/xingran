---
phase: 114-map3d-clustering-o-n
plan: 03
subsystem: ui
tags: [react, typescript, vitest, dead-code, event-cleanup, baidu-map, performance, manual-verification]

# Dependency graph
requires:
  - phase: 114-02
    provides: 两组件 zoomend/tiltend 具名 handler + effect cleanup（同引用 removeEventListener）落地形态；clusterBuildings 消费链
provides:
  - MAP3D-06 完成态：BuildingMarkers.tsx / CityMarkers.tsx 已删除，src/ 内零引用，@uiw/react-baidu-map 零 import（Phase 120 DEAD-02 僵尸依赖移除解锁）
  - MAP3D-05 回归守护：hubei-map-cleanup.test.tsx 3 用例（GL 版 zoomend/tiltend 同引用移除、BMap 版 zoomend 同引用移除、AK 缺失守卫冒烟）
  - MAP3D-02 收口：⚡ Auto-approved 人工性能验证（HUMAN-UAT 候补步骤全文留档）+ Plan 01 500ms 性能冒烟自动兜底
affects: [Phase 120 DEAD-02（@uiw/react-baidu-map 卸载）, Phase 120 死代码批次（components/utils.ts 仅测试引用函数清理）]

# Tech tracking
tech-stack:
  added: [] # 零新增依赖（git rm 删除 + 测试文件，无包变更）
  patterns: [window 全局 fake namespace mock（BaiduMapScript.test.ts 风格）, 可 new 构造器 mock（独立 function 表达式标识符传入 vi.fn——躲 prefer-arrow-callback 自动改写）, vi.stubEnv + 动态 import（模块加载期求值的 env 常量测试）, 突变校验（临时破坏 cleanup 引用一致性证明测试判别力）]

key-files:
  created:
    - xingran-react-frontend/src/pages/operations/building-spaces-3d/__tests__/hubei-map-cleanup.test.tsx
  modified: [] # Task 1 为删除：components/BuildingMarkers.tsx、components/CityMarkers.tsx

key-decisions:
  - "死组件删除严格限界：仅 git rm 两个 .tsx，components/ 平行第二套 utils.ts/constants.ts/types.ts 与 components/__tests__/utils.test.ts、src/types/global.d.ts:57（HubeiMapGL type-only import）全部原样——components/utils.ts 沦为仅测试引用的导出不顺带清理（归 Phase 120 死代码批次）"
  - "fake namespace 构造器用『独立 function 表达式 + 标识符传入 vi.fn』——内联 vi.fn(function(){...}) 会被 lint-staged 的 eslint --fix(prefer-arrow-callback) 静默改写回箭头形态，箭头不可 new 致全量套件下 2 用例必红（0db74c3 修复，本地 --fix 预演验证不再改写）"
  - "守卫用例 vi.resetModules() 后必须 mockClear 被 mock 的 BaiduMapScript——resetModules 不重置 mock 注册表，脏 mock 携带前序用例调用记录会假红"
  - "TDD RED 门的等效证明用突变校验：cleanup 临时改内联箭头形态（引用不一致）→ GL 用例精确变红 → git checkout 恢复复绿；实现本体已在 Plan 02（f748fa7/a6b3fdd）落地，本 plan 无实现步骤，测试纯回归守护"
  - "MAP3D-02 人工验证按 AUTO_MODE 规则自动批准：AK 已配置（.env.development），HUMAN-UAT 步骤全文留档供用户按 VALIDATION.md Manual-Only 表执行；自动侧兜底为 Plan 01 的 1000 点 500ms 性能冒烟（实测毫秒级）"

patterns-established:
  - "Pattern 可 new 的 vi.fn mock：构造器实现写独立 function 表达式再标识符引用，绝不内联（prefer-arrow-callback 会改写内联 function 回调）"
  - "Pattern 模块级 env 常量测试：vi.stubEnv 先于 beforeAll 动态 import；变体路径用 resetModules + restub + try/finally"

requirements-completed: [MAP3D-02, MAP3D-05, MAP3D-06]

# Metrics
duration: 54min
completed: 2026-09-12
---

# Phase 114 Plan 03: 死组件删除 + cleanup spy 测试 + MAP3D-02 人工验证收口 Summary

**删除 @uiw/react-baidu-map 仅有的两处消费方（MAP3D-06，解锁 Phase 120 DEAD-02），新增 3 用例 cleanup spy 测试锁定 MAP3D-05 监听器生命周期（同引用 removeEventListener），MAP3D-02 人工性能验证按自主模式自动批准并留档 HUMAN-UAT 步骤**

## Performance

- **Duration:** 54 min
- **Started:** 2026-09-11T18:23:03Z
- **Completed:** 2026-09-11T19:17:29Z
- **Tasks:** 3（Task 3 checkpoint 按 AUTO_MODE 自动批准）
- **Files:** 1 created / 2 deleted

## Accomplishments

- **MAP3D-06 完成**：`components/BuildingMarkers.tsx`（107 行）与 `CityMarkers.tsx`（144 行）删除——二者是 `@uiw/react-baidu-map` 在 src/ 仅有的两处 import，删除后 `grep "@uiw/react-baidu-map" src/` 零命中，Phase 120 DEAD-02 僵尸依赖卸载解锁（本 plan 不执行 npm uninstall）
- **删除雷区零触碰**：`grep "BuildingMarkers\|CityMarkers" src/` 零命中；components/ 平行第二套 utils.ts / constants.ts / types.ts、components/__tests__/utils.test.ts、src/types/global.d.ts:57 的 HubeiMapGL type-only import 全部原样存在（存活消费方 BuildingModel3D/AddressInput 不受影响）
- **MAP3D-05 回归守护落地**：`hubei-map-cleanup.test.tsx` 3 用例——GL 版 zoomend/tiltend 挂载各一次且 unmount 后以同一 handler 引用移除、BMap 版 zoomend 同引用移除、AK 缺失守卫路径不发起脚本加载不挂监听不抛错；突变校验证明判别力（引用不一致形态下精确变红）
- **MAP3D-02 收口**：⚡ Auto-approved（autonomous mode）——人工浏览器性能验证步骤全文留档（见下方 Human-UAT 段），自动侧由 Plan 01 的 1000 点合成夹具 <500ms 性能冒烟（实测毫秒级）+ 聚类一致性 15 用例兜底
- **七 gate 前端项全绿**：全量 vitest 553 文件 3800 用例通过、lint 0 errors、type-check exit 0、build 通过（1m13s）——D-04 零回归、D-02 不倒退、D-03 entry gzip 不推高

## Task Commits

Each task was committed atomically:

1. **Task 1: 删除死组件 BuildingMarkers/CityMarkers（MAP3D-06）** - `d57177a` (feat)
2. **Task 2: cleanup spy 测试锁定 MAP3D-05 监听器生命周期** - `9d40f5f` (test)
3. **Task 2 修复: 构造器 mock 改独立 function 表达式** - `0db74c3` (fix)
4. **Task 3: MAP3D-02 人工性能验证** - ⚡ Auto-approved (autonomous mode)，无文件改动无提交

## MAP3D-02 Human-UAT 候补（人工验证步骤全文）

⚡ Auto-approved (autonomous mode) —— 以下步骤供用户后续按 VALIDATION.md Manual-Only 表执行（VITE_BAIDU_MAP_AK 已在 .env.development 配置，环境就绪）：

1. 启动前端开发服务器：`cd xingran-react-frontend && npm run dev`
2. 登录系统，导航到楼宇空间 3D 页面（operations/building-spaces-3d）
3. 确保楼宇数据达千级（若生产数据不足，可在楼宇管理中批量导入测试楼宇，或接受现有数据量验证）
4. 打开 DevTools Performance 面板录制，连续执行：滚轮缩放（zoom 8→10→8 往返数次）→ WebGL 版切换 3D 倾斜（tilt 拖拽）→ 普通版/WebGL 版 Switch 来回切换
5. 检查录制结果：主线程无 >1s 长任务（改造前每次 zoomend 有秒级卡死）；交互跟手无明显掉帧段
6. 聚类视觉一致性：缩放到楼宇重叠区域，确认聚类圆圈数量/位置/数字与改造前直觉一致（成员顺序、tooltip 前排楼宇、点击侧栏列表顺序无肉眼可见漂移）
7. 控制台无新增报错/警告（特别留意快速切换地图版本开关时的 unmount 相关警告——cleanup spy 测试已在 jsdom 侧锁定同引用移除）

## Files Created/Deleted

- `xingran-react-frontend/src/pages/operations/building-spaces-3d/__tests__/hubei-map-cleanup.test.tsx` - [新建] MAP3D-05 cleanup spy 测试（211 行，3 用例；fake 地图 + function 形态构造器 mock + 突变校验留痕注释）
- `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/BuildingMarkers.tsx` - [删除] 107 行（@uiw/react-baidu-map 消费方，零外部引用）
- `xingran-react-frontend/src/pages/operations/building-spaces-3d/components/CityMarkers.tsx` - [删除] 144 行（同上）

## Decisions Made

- fake namespace 构造器 mock 用独立 function 表达式（`const MapCtor = function () {...}` + `vi.fn(MapCtor)`）而非内联——lint-staged 的 eslint --fix(prefer-arrow-callback) 会把内联 `vi.fn(function(){...})` 改写回箭头形态，导致 `new BMapGL.Map(...)` 抛 "is not a constructor"；该坑在全量套件（而非目录级）复跑时才暴露，修复后本地预演 --fix 确认标识符引用形态不被改写
- 守卫用例断言前对被 mock 的 BaiduMapScript `mockClear()`——`vi.resetModules()` 不重置 mock 注册表，脏 mock 会携带前序用例（test-ak 路径）的调用记录造成假红
- 守卫用例加断言 `loadBaiduMapGLScript` 未被调用（强于计划条款的"不挂监听"——守卫生效时连脚本加载尝试都不应发生）
- TDD RED 门等效证明：实现本体已在 Plan 02 落地（本 plan 无实现步骤），以突变校验（临时改坏 cleanup → 测试精确变红 → 恢复复绿）证明判别力，替代自然 RED

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] cleanup 测试构造器 mock 被 lint-staged 改写致全量套件 2 用例红**
- **Found during:** Task 2（全量套件复跑时暴露；目录级运行通过是因为当时磁盘文件还是提交前版本）
- **Issue:** 提交时 lint-staged 执行 eslint --fix，prefer-arrow-callback 规则把 `vi.fn(function () { return fakeMap; })` 内联回调静默改写为箭头形态；箭头函数不可 `new`，组件 `new BMapGL.Map(...)` 抛 TypeError → initMap 走 catch → addEventListener 零调用 → waitFor 超时断言失败（553 文件全量跑仅此文件 2 用例红）
- **Fix:** 构造器实现改为独立 function 表达式、以标识符传入 vi.fn（该形态不被 prefer-arrow-callback 改写）；JSDoc 留痕两点注意事项
- **Files modified:** `__tests__/hubei-map-cleanup.test.tsx`
- **Commit:** `0db74c3`

### TDD Gate Compliance

- Task 2 标记 tdd="true"，但 `<files>` 仅含测试文件、无实现步骤——行为本体（具名 handler + 同引用 cleanup）已由 Plan 02（f748fa7/a6b3fdd）落地，计划原文即声明该断言"改造后必绿——这就是 MAP3D-05 的回归守护"
- RED 门以突变校验等效执行：临时将 HubeiMapGL cleanup 改为内联箭头形态（引用不一致）→ GL 用例精确变红（1 failed | 2 passed）→ `git checkout --` 恢复 → 3/3 复绿
- 提交序列为单个 `test(114-03)` 提交（9d40f5f）+ 后续 `fix(114-03)`（0db74c3），无独立 feat 提交——与"无实现步骤"的任务性质一致

## Issues Encountered

- 全量套件单次运行约 10-12 分钟，超出单命令 600s 超时一次（转入后台完成，exit 0），非测试问题
- 新增测试文件带来 3 条 lint warning（`_zoomControl` unused + `(window as any)` ×2，均与既有测试风格一致），lint gate 口径为 0 errors，基线 1358 → 1361 warnings 属测试文件豁免惯例，不影响 gate

## Verification Results

- **grep 断言（MAP3D-06）**：`grep -rl "BuildingMarkers\|CityMarkers" src/` 零命中；`grep -rl "@uiw/react-baidu-map" src/` 零命中；两死组件文件不存在；components/utils.ts、constants.ts、types.ts、__tests__/utils.test.ts、global.d.ts 全部原样
- **cleanup spy 测试**：`npx vitest run src/.../hubei-map-cleanup.test.tsx` 3/3 绿（含 GL 同引用移除、BMap 同引用移除、AK 守卫冒烟）；突变校验（改坏 cleanup）→ 精确 1 用例红 → 恢复复绿
- **目录级回归**：`npx vitest run src/pages/operations/building-spaces-3d` 11 文件 82 用例全绿（既有 79 + 新增 3）
- **全量套件**：`npx vitest run` 553 文件 3800 用例全绿（D-04 零回归口径）
- **lint**：`npm run lint` exit 0（0 errors / 1361 warnings，均为既有风格豁免）
- **type-check**：`npm run type-check` exit 0
- **build**：`npm run build` 通过（1m13s，entry gzip 无推高——本 plan 零运行时源码变更，仅删死代码 + 加测试）

## User Setup Required

None - no external service configuration required（VITE_BAIDU_MAP_AK 已在 .env.development 配置，Human-UAT 环境就绪）。

## Known Stubs

None - 无占位/未接线代码（删除 + 测试，无新生产代码路径）。

## Next Phase Readiness

- Phase 114 三 plan 全部完成，可进入 `/gsd:verify-work`
- Phase 120 DEAD-02 解锁：@uiw/react-baidu-map 在 src/ 已零 import，可直接 `npm uninstall @uiw/react-baidu-map`（届时按 plan 走 legitimacy gate）
- Phase 120 死代码批次候补：`components/utils.ts` 的 getBuildingMarkerColors / generateClusterIconSVG 等导出删除死组件后沦为仅测试引用（knip 可见，非七 gate）
- MAP3D-02 唯一遗留：Human-UAT 候补步骤（见上）待用户空闲时执行，自动侧兜底已绿，不阻塞 verify-work

## Self-Check: PASSED

- `__tests__/hubei-map-cleanup.test.tsx` 存在（211 行 ≥ min_lines 80，含 "toHaveBeenCalledWith" 与 "mock.calls" key_links 标记）
- BuildingMarkers.tsx / CityMarkers.tsx 不存在
- 提交 d57177a / 9d40f5f / 0db74c3 均在 git log 中确认
- 全量 vitest / lint / type-check / build 四 gate 复跑全绿

---
*Phase: 114-map3d-clustering-o-n*
*Completed: 2026-09-12*
