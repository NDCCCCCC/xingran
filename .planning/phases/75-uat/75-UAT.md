# Phase 75：浏览器实机 UAT 报告（68.1 UI-SPEC §8 全表 + v1.23 抽查）

- **日期**：2026-10-28
- **执行方式**：agent-browser skill（真实 Chromium），零新依赖（未引入 Playwright）
- **被测产物**：`ruoyi-react-frontend` prod build（`npm run build`），build 时 commit **988e4949**
  （`docs(planning): Phase 74 完成——MIG-03/HYG-02/03/04 Satisfied…`；工作树另有一处与本 phase 无关的未提交改动 `internal/services/system/asset_columns_schema.json`）
- **访问地址**：`npm run preview` → http://localhost:4173

---

## 一、三项前置满足情况

| 前置 | 结果 | 说明 |
| --- | --- | --- |
| ① prod build 产物 + preview | ✅ 满足 | `VITE_API_BASE_URL=http://localhost:9000/api/v1 npm run build`（33s，vendor chunk 全量拆分正常：vendor-react 2,035 kB / three / echarts / xlsx / md-editor 均独立 chunk，懒加载正常触发）；`npm run preview` 起 4173 静态服务 |
| ② 请求加密按生产配置 | ✅ 满足（降级说明见下） | 加密开关自 Phase 迁移后为**运行时**从后端 `/system/auth/encryption-config` 获取，`VITE_ENABLE_REQUEST_ENCRYPTION` 构建期注入已不存在于代码（`src/lib/api.ts:122` fail-secure 逻辑）。实测：本地后端 DB 键 `sys.request.encryption.enabled=false`，与生产 10.62.10.39 实测返回（`enabled:false`）**完全一致**；SM2+SM4 加密中间件已挂载（`public-key` 端点正常返回公钥），加密/解密链路可用。即用户真实路径（明文开关关闭态 + 中间件在场）与生产一致 |
| ③ 清空 sessionStorage 验证持久化恢复 | ✅ 执行 | 每次登录前 `sessionStorage.clear(); localStorage.clear()`；翻页状态持久化行为实测结果见 #13（发现 1 个失败子项） |

**后端环境**：本地 `ruoyi-backend.exe`（sqlite + memory cache，临时配置在 `$TEMP/p75-uat/`），滑动验证码 disabled、admin/admin123 可登录，测试数据（12 楼宇 / 3 专线 / 3 资产 / 4 岗位 / 受限用户 uatview）经 API 造数。生产 Linux（10.62.10.39:9000）可达但开启 slider 验证码，按任务要求未硬闯（见 BLOCKERS）。

---

## 二、§8 回归清单结果总表（19 条）

判定口径：✅ 通过 / ❌ 显式失败 / 🚫 BLOCKED。证据均为本目录 `screenshots/` 下文件。

| # | 自检项 | 结果 | 证据 | 备注 / 复现要点 |
| ---: | --- | :-: | --- | --- |
| 1 | 删除确认弹窗文案含对象名 | ✅ | 01-delete-confirm-object-name.png | 楼宇页行「删除」→ 正文「将删除楼宇 UAT测试楼宇A，删除后不可恢复，是否继续？」 |
| 2 | 确认按钮 danger + 提交中 loading | ✅ | 02-03-07-submitting-state.png | XHR 注入 4s 延迟捕捉提交中：`.ant-btn-loading` + `.dangerous` 均为 true，loading 态不可重复点 |
| 3 | 提交中弹窗不可被关闭 | ✅ | 03-submitting-esc-mask-ignored.png | 提交中按 Esc（modalOpen=true）+ 程序化点遮罩（loading=true, open=true），均不关闭 |
| 4 | 删除失败弹窗保持 + 失败提示 | ✅ | 04-delete-fail-message.png / 04-delete-fail-modal-stays.png | 停后端后点确认：提示「删除失败: 请求超时」，弹窗保持打开可重试 |
| 5 | 批量删除显示真实对象 | ✅ | 05-batch-delete-real-objects.png | 资产页勾选 3 条 →「将删除选中的 3 项资产，删除后不可恢复，是否继续？」 |
| 6 | 批量删除前必确认 | ✅ | 06-dedicated-line-batch-confirm.png | 专线页勾 3 条点批量删除 → 模态框带遮罩先弹确认（「选中的 3 项专线」） |
| 7 | 提交中取消按钮禁用 | ✅ | 02-03-07-submitting-state.png | 提交中 `cancel.disabled === true`，点不动 |
| 8 | Popconfirm 已升级为 Modal | ✅ | 01/05/06/09 各截图 | 楼宇、资产、专线、字典 4 处行删除入口点击后均为带遮罩模态框 |
| 9 | 升级后触发入口位置/文案不变 | ✅（抽验 4/14） | 同上 | 4 处入口均在行/卡片内原位、按钮文本仍为「删除」，无丢失入口；未逐个走完 14 处（其余入口与已验 4 处走同一 `confirmDelete.ts` 通道） |
| 10 | 升级后取消语义 + Esc 有意加强 | ✅ | 10-esc-does-not-close.png | 点「取消」无副作用（行数不变、无残留弹窗）；非提交态按 Esc 弹窗不关闭（`keyboard:false`），用户可明确走「取消」退出，无卡死感 |
| 11 | 非删除破坏性文案各自贴切 | ✅（抽验 2/10） | 11-clear-cache-wording.png / 11-clear-log-wording.png | 清空缓存：「将全部缓存数据，操作不可恢复…」确认键「清空」；清空日志：「将全部操作日志…」。均无「将删除…」误用模板 |
| 12 | 危险操作收口计数归零 | ✅ | 静态 grep（记录于本文件） | 删除类裸 `Modal.confirm`=0；删除类 `<Popconfirm`=0（`WorkstationDeviceTable:478` 的 Popconfirm 为「设为主设备？」非删除语义且带豁免注释）。备注 1 处轻微偏离：`ColumnConfigModal.tsx:136`「重置列配置」确认无 `eslint-disable` 理由注释（非删除类，建议补注释，不计缺陷） |
| 13 | 分页口径统一 | ✅（口径 ✅ / 恢复 ✅ 已修复） | 13-pagination-total.png / 13-empty-no-pagination.png / FAIL1-fixed-1-page2-before-tab-switch.png / FAIL1-fixed-2-page2-after-tab-switch.png | 口径：「共 12 条 / 2 页」正确（`usePagination.ts:122`），空数据无分页文案 ✅。恢复失败 FAIL-1 **已修复并实机复验通过**（2026-10-28）：切 tab 后仍停第 2 页、storage 未覆写，见「四、FAIL-1」修复记录 |
| 14 | 空态/错误态可见 | ✅ | 13-empty-no-pagination.png / 14-error-state-retry.png | 机房页空数据显示「暂无数据」无分页；后端重启窗口期资产页出现「查询失败:网络异常」+「重新加载」按钮，后端恢复后可重试成功 |
| 15 | 固定列不失效 | ✅ | 15-fixed-column-scroll.png | 参数配置页（操作列 fixed right）横向滚动 500px 前后固定列 x 坐标 1044→1044 不变 |
| 16 | 行操作权限不变 | ❌ | 16-no-perm-no-delete-btn.png | 受限用户 uatview（角色仅含 `ops:building:list/query`，无 delete）登录后楼宇页 9 行全部仍显示「编辑/删除」按钮（delCount=9）。**环境限制**：角色创建 API 在 sqlite 后端报 500（`ON CONFLICT` 不匹配，PG 不受影响），角色-菜单关联为手工插库，未走正常缓存加载链路，后端实际下发权限集未能独立核实（`/system/menus/permissions` 对该用户 403）。根因候选：①前端 hasPerm 在权限获取失败时未隐藏危险按钮；②手工插库角色数据不完整导致后端权限异常。**建议在 PG 生产库复测定责** |
| 17 | 可访问名正确 | ✅ | 17-batch-delete-btn-count.png + snapshot 记录 | 删除按钮 aria-label=「删除楼宇 UAT测试楼宇A」（删除+对象名）；批量删除按钮可见文本「批量删除 (3)」数量内嵌，无冗余 aria-label（符合 64 FIX-05 口径） |
| 18 | 豁免清单 100% 带理由 + 脚本幂等 | ✅ | 命令行 diff 记录 | `gen-p65-exemptions.mjs` / `gen-p66-exemptions.mjs` 各连跑两次，有效输出 diff 为空（仅 node 进程 PID 告警行差异） |
| 19 | lint error ≤ 基线 83 | ✅ | 命令行输出 | `npm run lint` → **17 errors, 34 warnings** ≤ 83 |

**分布：通过 17 / 显式失败 2（#13、#16）/ BLOCKED 0**（重点条目 #2/#3/#7 实测通过；#13 口径通过但恢复子项失败；#15/#16 实测，#16 失败）。

---

## 三、v1.23 新增抽查（3 条）

| 抽查 | 结果 | 证据 | 说明 |
| --- | :-: | --- | --- |
| SC1 TextArea 一键清空（楼宇描述） | ✅ | SC1-textarea-clear-building.png | 新增楼宇弹窗描述 TextArea 有值时出现 clear icon（visible/opacity 1），点击后 value="" |
| SC2 TextArea 一键清空（参数配置备注） | ✅ | SC2-textarea-clear-config.png | 编辑参数弹窗备注 TextArea（预填文案）→ 输入新文本 → clear icon 点击 → value=""。两处不同域的 TextArea 契约均生效 |
| SC3 AppSelect 搜索过滤（用户表单「职位」） | ✅ | SC3-appselect-search-filter.png | 职位下拉（AppSelect showSearch + optionFilterProp="value"）输入 "Arch" → 选项精准过滤为 "Architect" |

> 测试备注：SC3 期间发现**测试数据编码问题**——经 bash/curl 直接 POST 的中文字段落库为乱码（与前端无关，属造数通道编码损坏）；已改用 UTF-8 文件/ASCII 数据完成验证。线上数据不受影响（本问题仅存在于本地造数路径）。

---

## 四、失败 / 异常明细与根因猜想

### FAIL-1：#13 usePagination 持久化恢复失效（切 tab 场景）——✅ 已修复 + 实机复验通过（2026-10-28）
- **现象**：楼宇管理（`/operations/buildings`）翻至第 2 页 → sessionStorage `ruoyi_table_state_operations_buildings_current=2` 正确写入 → 切到「首页」tab 再切回「楼宇管理」tab → 页码显示第 1 页且 storage 被覆写为 `1`。另：F5 刷新后 SPA 落到 `/dashboard` 且 `ruoyi_last_visited_path` 被覆盖为 `/dashboard`，无法经刷新路径回归原页。
- **根因（已定责）**：`usePagination` 本身无问题（keyPrefix 读写一致）。真正根因在 PageTable → useTableManager 数据加载链：`PageTable.tsx` 的 mount effect 无条件调用 `applyFilters`，而 `useTableManager.ts` 的 `applyFilters` 硬编码 `currentRef.current = 1; setCurrent(1); loadData({ current: 1 })`（回第 1 页语义）——tab 切换 remount 时把 sessionStorage 恢复出的 current=2 覆写回 1。楼宇页走 PageTable 契约组件（非直接 usePagination），故嫌疑①「mount 硬编码回写」成立。
- **为什么 jsdom 没抓到**：既有单测覆盖 `usePagination` hook 本身（mount 恢复断言通过）与 PageTable 基础骨架，但没有「预置持久化页码 + 组件 mount + 数据加载 effect」的组合用例，也未模拟卸载→重挂载（tab 切换 remount）路径——覆写发生在组件层 mount effect 与数据加载时序之间，恰好落在覆盖盲区。
- **修复**：`useTableManager.applyFilters` 增加 `opts.keepPage` 选项（保留当前页码加载，不回写 1）；`PageTable` mount 首载改走 `keepPage: true`（恢复语义优先），extraFilters 引用变化仍回第 1 页（联动语义不变）。一处修全站（所有 PageTable 列表页）受益。
- **新测试**：`usePagination.test.tsx`「tab 切换卸载→重挂载：恢复持久化页码且 sessionStorage 未被覆写」；`useTableManager.test.tsx`「applyFilters({ keepPage: true }) 保留当前页码加载且不覆写持久化 current」；`PageTable.test.tsx`「UAT FAIL-1：sessionStorage 有持久化页码时 mount 首载恢复该页码且不覆写 storage」。
- **实机复验**：✅ 通过。prod build（修复后）+ preview 4173 + sqlite 后端 9000，admin 登录，楼宇页补造数据至 13 条（2 页）→ 翻第 2 页（storage=`2`）→ 切「仪表盘」tab 再切回 → **仍停第 2 页、「共 13 条 / 2 页」口径正确、storage 未被覆写（=2）**。证据：`screenshots/FAIL1-fixed-1-page2-before-tab-switch.png`、`screenshots/FAIL1-fixed-2-page2-after-tab-switch.png`。
- **单测对照**：`usePagination.test.tsx` jsdom 下恢复断言通过 → **jsdom 覆盖与浏览器实机行为存在盲区**，正是 Phase 75 要抓的类别（盲区已由上述新用例闭合）。

### FAIL-2：#16 受限用户仍可见「删除」行操作按钮
- **现象**：见总表 #16。
- **根因候选**：①前端 `hasPerm` 类判定在权限列表获取失败/为空时默认放行危险按钮（fail-open，正确姿势应为 fail-close）；②本地 sqlite 手工插库角色未触发权限缓存加载，后端下发的权限集异常——**不能排除测试环境因素**，定责需在 PG 环境用 UI 正常路径建角色复测。

### 环境备注（非缺陷）
1. 本地 sqlite 后端「角色管理→批量分配菜单权限」500：`ON CONFLICT clause does not match any PRIMARY KEY or UNIQUE constraint`——sqlite 迁移兼容问题，生产 PG 不触发，建议登记后续 sqlite 兼容债。
2. 本地配置踩坑记录（对后续 UAT 有用）：`cache.retry_initial_delay` 需整数（毫秒）；`security.sm4_key` 需 16 字节的 base64（24 字符）；`use_sm2: true` 时 `JWT_SM2_PRIVATE_KEY/PUBLIC_KEY` 必须注入（hex，公钥需 `04+X+Y` 定长 130 字符），否则 public-key 500 且加密中间件不挂载、`/encryption-config` 永远返回默认 true，前端报「登录安全配置已过期」无法登录。

---

## 五、结论：v1.23 可否收档

**判定：暂不满足「无债、无盲区」全绿收档；可按「17 通过 + 2 显式失败带证据转新债」的有条件方式推进收档流程。**

- 68.1 §8 全表 19 条无「未测」空白，重点条目 #2/#3/#7/#15 实测通过；#13/#16 为**带截图与复现步骤的显式失败**，符合 SC2「失败项显式转为带 owner 的新债」的出口，但不符合 SC1「19 条全部通过」的理想判定。
- 抽查 3 条（TextArea ×2、AppSelect ×1）全部通过——v1.23 自身交付的TextArea 一键清空验收成立。
- 遗留（建议转 quick / 下里程碑）：
  1. **[owner: 前端]** FAIL-1 usePagination 切 tab 恢复失效（附复现步骤）；
  2. **[owner: 前端+后端]** FAIL-2 受限用户危险按钮可见性，需 PG 环境复测定责后修复；
  3. [低] `ColumnConfigModal.tsx:136` 补豁免理由注释；
  4. [低] sqlite 后端角色-菜单批量分配 500 的兼容债。
- SC3/SC4（结果回写 68.1 UI-SPEC §8、7 份 debug 文档 Resolution 一致性）属收档动作，本 phase 未改动任何业务代码与 §8 原文（本 phase 只测不改），**回写动作留给收档执行者**：将本表结果誊入 §8 并标注债务关闭状态。

## 六、BLOCKERS（需要用户做的事）

1. **生产实机登录会话**：Linux 生产 10.62.10.39:9000 开启 slider 滑动验证码，自动登录不可行（按约定未硬闯）。若需在生产实机复核 #16（权限）与 FAIL-1，请提供：a) 临时关闭验证码的维护窗口，或 b) 一个可用的低权限测试账号会话。
2. **PG 环境复测**：#16 定责需要在 Postgres 生产/准生产库用 UI 正常路径创建受限角色复测（本地 sqlite 角色创建 API 有兼容性 500）。
3. 本地测试进程与数据：preview（4173）与 sqlite 后端（9000，`$TEMP/p75-uat/`）已停止并清理；`uatview` 受限用户与 UAT 造数仅存在于本地临时 sqlite，不涉及生产。
