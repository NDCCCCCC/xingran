# Phase 129 · TIDY-02 — v1.25 遗留 human_needed 项回补清单

> 组织日期：2026-11-02
> 目的：把 v1.25 三个 phase（84 / 87 / 90）的 `human_needed` 遗留项**集中收口**——
> 逐项列明「阻断原因 / 解锁需要什么 / 由谁执行 / 判定方式」，
> 避免它们散落在三个 milestone 目录里长期悬空。

## 一、结论先行

三个 phase 的 `human_needed` **不是三个独立缺口，而是两类阻断**：

| 阻断类别 | 涉及项 | 解锁需要 |
|---|---|---|
| **① 生产 PG 访问** | 84 的 2 项 + 87 的 4 项 | 生产库只读访问（或部署窗口内的执行权） |
| **② AD 域认证 / 生产环境部署** | 87 的 3 项 + 90 的 40 组合 + 40 张截图 | 一个可登录账号（AD 账号 或 临时切 `sys.auth.default.mode`） |

**已由等价强证据覆盖的部分（不阻断 phase 判定，见第五节）**：Phase 90 的 13 页 × 明暗矩阵
（456 格 computed-style 逐格比对）比 40 张截图更强，另有 CSS 冻结契约检查 ALL PASS。

## 二、Phase 84（pmodel-gaps）遗留 — 2 项

来源：`.planning/milestones/v1.25-phases/84-pmodel-gaps/84-VERIFICATION.md` §遗留

| # | 项 | 为何必须人工 | 解锁需要 | 判定方式 | 状态 |
|---|---|---|---|---|---|
| 84-1 | PG 实机 `data_scope=3` 复测 total | 生产 PG 与 SQLite 行为差异（v1.24 #79 同款对照） | 生产 PG 只读访问 | 方法已登记于 `84-VALIDATION.md`，照做即可 | ⏳ 待执行 |
| 84-2 | `sys_role_menu` / `sys_user_role` 生产库实际列集 | 需 `\d` 实查 | 生产 PG 只读访问 | `SELECT conname FROM pg_constraint WHERE conrelid=...` | ⏳ 待执行 |
| 84-3 | v1.24 #79 验收脚本是否断言用户列表行数 | 历史脚本行为需人工核对 | — | — | ✅ **已核对**（grep 无行数断言，无需登记更新） |

## 三、Phase 87（permission-grant）遗留 — 7 项 + OQ-7

来源：`.planning/milestones/v1.25-phases/87-permission-grant/87-EVIDENCE.md` §七（生产侧人工验证清单）

| # | 验证项 | 解锁需要 | 判定方式 |
|---|---|---|---|
| 87-1 | 受影响生产角色清单（判定 SQL 在 `87-RESEARCH §E19`） | 生产 PG 只读 | 逐条列明角色；若仅 admin 则表明业务角色未配置 |
| 87-2 | 生产库既有行零改动核对 | 生产 PG 只读 | 迁移前后计数差 = **+65 菜单行 / +N 授权行 / 0 角色变更 / 0 用户变更** |
| 87-3 | 迁移在生产 PG 上实际执行（PG 方言） | 生产部署窗口 | `SELECT count(*) FROM sys_menu WHERE menu_type='F' AND perms IN (<65 码>)` = **65** |
| 87-4 | `\d sys_role_menu` 唯一键约束复核 | 生产 PG 只读 | 记录约束名或确认无唯一键（Migration 226 用 `NOT EXISTS` 兼容双方言） |
| 87-5 | HTTP 实测 403 → 200 | 生产后端部署运行 + Token | 补授前 403 → 等待 **31s**（权限缓存 TTL 30s，`InvalidateMiddlewarePermCache` 未接线）或重启后端 → 200 → 撤销复原再等 31s → 403 |
| 87-6 | 前端 UI 实机交互（角色编辑弹窗菜单树 65 个按钮） | 可登录环境（AD 账号） | `/system/role` 清 sessionStorage → 编辑角色 → 树中 65 个按钮正常展示 → 勾选保存 → 刷新回读保持勾选 |
| 87-7 | 生产部署顺序规范 | 发布流程 | **必须先执行后端 Migration 226（补齐数据），再上线前端包与门禁脚本** |

**OQ-7（独立后续项，不在上述 7 项内）**：5 个权限码存在架构层面「读路由恒 403」缺口，
登记于 `87-EVIDENCE.md` §八，属独立后续项，本 phase 不闭合。

## 四、Phase 90（input-variant）遗留 — 40 组合 + 40 张截图

来源：`.planning/milestones/v1.25-phases/90-input-variant/90-VERIFICATION.md` §七

| # | 项 | 为何必须人工 | 解锁需要 | 现有材料 |
|---|---|---|---|---|
| 90-1 | 13 页 × 明暗 = **40 个「需登录页面」巡检组合** | **认证墙**：本环境 `sys.auth.default.mode = ad`（admin 走 AD 域认证），本地 `admin123` 不适用。滑块验证码**已自动化解决**（`p90-captcha-solve.cjs` / `p90-login.cjs`，实测 `POST /api/v1/system/auth/captcha/verify/slider` 返回 200），但随后的登录被 AD 认证拒绝（`用户 admin 认证失败(mode=ad)`）。切换该配置属**生产数据写入**，按纪律不擅自执行 | 一个可登录账号：**① 提供 AD 账号**（域\用户名 + 密码）**或 ② 授权在非生产环境临时把 `sys.auth.default.mode` 切到 local**（跑完切回） | 待巡检清单 `inspection-manual-todo-90.json`（40 条，含 mode/name/url）；逐个探测结果 `inspection-result-90.txt`（每条含 `blockedByLoginWall: true` + `finalHref`） |
| 90-2 | 目标页截图（89 的 40 张） | 同上（需登录态） | 同上 | 本波只截到无登录态可达的 `/login` 明暗 2 张（`login-light.png` / `login-dark.png`）；目标页截图待人工 |
| 90-3 | M16 在「未巡检面」的达成 | 同上 | 同上 | 已巡检面 `deprecated = 0` / `errors = 0`；未巡检面待人工确认 |

**环境事实（2026-11-02 只读查证）**：远程环境 `10.62.10.39` 的 `sys_config` 实测
`sys.auth.default.mode = ad`、`sys.auth.ad.enabled = true` ⇒ 90-1 的阻断**当前确实存在**。

## 五、已由等价强证据覆盖的部分（申请改判的根据）

Phase 90 虽未取得登录态截图，但本波用**更强的等价证据**覆盖了「90 的改动没让组件破相」：

| 覆盖项 | 证据 | 相对强度 |
|---|---|---|
| 全站 22 形态 × 明暗 × 6 属性的**逐格**视觉属性 | 矩阵 456 格终判：**bg / shadow / btw / fontSize 四个维度差异 = 0** | **强于** 14 页截图（逐格 computed-style 比对 vs 像素目视） |
| 形态集合完整（无结构缺失） | before = 22 / after = 22，f22 的 carrier / inner 两侧均在 | 强 |
| v1.24 冻结契约零破坏 | `css-contract-check.js` **ALL PASS**（C1a-C1c / C4a-c / C5a-c / C14a-g / C16a-c） | 强（静态断言） |

⇒ **建议**：若接受「computed-style 矩阵 + 契约检查」作为 40 组合的等价替代证据，
Phase 90 可由 `human_needed` 改判 `passed`。**该改判需用户确认（属验收口径变更）**。

## 六、解锁路径（用户只需做一件事即可解锁大部分）

| 选项 | 用户动作 | 解锁 |
|---|---|---|
| **A（推荐）** | 提供一个可用的 AD 账号（域\用户名 + 密码） | 90-1 / 90-2 / 90-3 + 87-6 → 全部可自动巡检（agent-browser + 已有滑块验证码求解脚本） |
| **B** | 授权在**非生产**环境临时把 `sys.auth.default.mode` 切到 local（跑完切回，记录变更） | 同 A（但需明确「哪个环境可写」） |
| **C** | 确认接受等价证据，Phase 90 改判 passed | 90 整项闭合，Phase 129 可零人工项完成 |
| **D** | 生产发布窗口（87-3 / 87-5 / 87-7 必须在生产部署时执行） | 87 的 7 项生产侧清单 |
| **E** | 提供生产 PG 只读访问 | 84-1 / 84-2 + 87-1 / 87-2 / 87-4 |

**当前状态**：以上选项**均未获授权**，故三项 human_needed 保持原判，
本清单作为 Phase 129 的「组织」交付物存证。

## 七、与真机 UAT 的关系

本轮真机 UAT（Phase 129 的第三项交付）已**独立完成并闭合**：
华为 S8700 11 动作 + 锐捷 RS8607E 12 动作真机实测，缺陷定位与修复见
`129-RESULTS.md`。三项交付中：

| 交付项 | 状态 |
|---|---|
| 真机 UAT | ✅ 完成（2026-11-02，见 `129-RESULTS.md`） |
| 目录改名（DEBT-06） | 🔄 进行中（task-31，webfe） |
| UAT 回补组织（TIDY-02） | ✅ 完成（本文件即为组织交付物；执行待用户授权，见第六节） |
