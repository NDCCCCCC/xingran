---
phase: "87"
slug: "permission-grant"
created: "2026-10-30"
---

# Phase 87 模式地图（PATTERNS）

> **用途**：给 `gsd-planner` 的施工图 ——「要改哪些文件 + 照哪个文件的第几行抄」。
> **与 87-RESEARCH.md 的分工**：研究文档回答「为什么这么做 / 清单是什么 / 坑在哪」；本文件回答「文件在哪一行、抄哪一段」。
> **溯源纪律**：`[VERIFIED: 路径:行号]` = 本会话用 Read 工具**实际打开该文件**并逐字引用；`[CITED]` = 引自既有 artifact（研究 / note / 验收报告）；`[ASSUMED]` = 推导未实证。
> **本会话新增实证**（本文件独有，研究文档未覆盖）：见 P-1 的「visible 落库实测」与 P-3 的「`IN ?` 展开实测」。

---

## Files to Create

> 共 **3** 个新文件。

### 1. `internal/core/db/migrations/migration_226_action_permission_f_type_seed.go`

**职责**：Phase 87 的播种载体 —— 在 `sys_menu` 新增 **72 条** `menu_type='F'` 的动作级权限行，并按「持有父 C 型菜单」精准授权给 `sys_role_menu`。

**为什么在本 phase 建**：研究结论「链路 ①②③④ 全部已就绪，唯一缺的是『F 型码在 `sys_menu` 里有一行』」`[CITED: 87-RESEARCH.md:92]`。用户已锁定「F 型动作权限的播种载体 = 写 migration」。

**硬约束**：
- 函数名 `Migrate226ActionPermissionFTypeSeed(db *gorm.DB) error`；文件名 `migration_226_action_permission_f_type_seed.go`。
- **编号 226 空闲已实证**：全仓 grep `migration_226|Migrate226|runAndRecord(226` → `No matches found`；`registry.go` 内最大版本为 `{225, "schema_migrations_table"}` `[VERIFIED: internal/core/db/migrations/registry.go:49]`。
- **必须双方言**（sqlite 是 dev 运行时也是测试运行时）：**禁止** `isPostgreSQL(db)` 守卫（`migration_202` 那样的守卫会让 sqlite 测试全部跳过 = 假绿）`[CITED: 87-RESEARCH.md:854]`。
- **禁止**写 Java 风格 `frame` / `cache` 列 —— 逐字依据：`// **绝对禁止** Java sys_menu 风格的 frame / cache 字段(Go sys_menu 用 Meta JSONB 表达)` `[VERIFIED: internal/core/db/migrations/migration_202_port_write_audit.go:27]`。
- 文件内**必须**包含 `// F_TYPE_SEED_BEGIN` / `// F_TYPE_SEED_END` 标记块（供校验脚本识别 F 型），见 P-4。

### 2. `internal/core/db/migrations/migration_226_action_permission_f_type_seed_test.go`

**职责**：sqlite 内存库 + 原生 DDL 的迁移单测，覆盖「F 行恰好 1 条 / 幂等 / 精准授权 / 父缺失降级 / 多父集合 / visible 落库值」。

**为什么在本 phase 建**：87-RESEARCH §Validation Architecture 的 Test Infrastructure 明写「仿 `internal/core/db/migrations/migration_218_ad_reconciliation_permission_test.go:23-70`」`[CITED: 87-RESEARCH.md:867]`；`87-VALIDATION.md` 的 87-NN-01..05 五条自动化断言全部指向该文件 `[CITED: .planning/phases/87-permission-grant/87-VALIDATION.md:45-49]`。

**硬约束**：
- **禁止用 GORM `AutoMigrate`** 建表 —— 逐字依据：`// 原生 DDL（不走 GORM AutoMigrate）：sqlite 测试路径不经 sanitizeSQLiteModelDefaults，model 里的 type:uuid 等 PG-only 片段会直接报错（同 migration_217 测试口径）` `[VERIFIED: internal/core/db/migrations/migration_218_ad_reconciliation_permission_test.go:18-20]`。
- 表名/列集照抄 `migrate218Statements` 三张表（`sys_menu` / `sys_role` / `sys_role_menu`）。

### 3. `ruoyi-react-frontend/src/pages/system/role/__tests__/permission.grant.test.tsx`

**职责**：角色授权闭环组件测试 —— F 型节点在角色页树中可见 → 可勾选 → 提交的 `menuIds` **含该 F 节点 id**（且因 antd 父子联动同时含父 C 菜单 id）。

**为什么在本 phase 建**：`87-VALIDATION.md` 的 87-NN-06 自动化断言指向该文件 `[CITED: .planning/phases/87-permission-grant/87-VALIDATION.md:50]`；87-RESEARCH §E17 的 T-1..T-4 四条用例归属该文件 `[CITED: 87-RESEARCH.md:709-712]`。

**硬约束**：
- **只新建，不得改动**同目录的 `permission.gate.test.tsx`（v1.24 三态门控契约回归基线）。
- 测试文件放在 `src/pages/system/role/__tests__/` 下（与 `permission.gate.test.tsx` 同目录），因为 `check-gating-coverage.mjs` 的页面归属规则**排除** `__tests__/`、`*.test.*`、`*.spec.*` `[VERIFIED: ruoyi-react-frontend/scripts/check-gating-coverage.mjs:11]`，故新增测试文件**不改变** 76 页恒等式。

---

## Files to Modify

> 共 **6** 个既有文件。改动点逐条列出。

### 1. `internal/core/db/migrations/registry.go`

**职责**：`ActiveMigrations` 版本清单（与 `database.go` 调用点集合对账用）。

**为什么在本 phase 改**：新增版本 226 必须登记，否则 `MissingRecords` 对账误报。

**改法**：在 `{225, "schema_migrations_table"},` 之后追加一行：

```go
	{225, "schema_migrations_table"},
	{226, "action_permission_f_type_seed"},
}
```

**逐字依据（维护约定，不许漏改）**：`// 维护约定：database.go 调用块每新增/移除一个 MigrateNNN 调用点，本清单必须同步增删，// 否则 MissingRecords 对账会误报。记录表只读不控制执行 —— 本清单**不参与**任何迁移 // 是否运行的判断。` `[VERIFIED: internal/core/db/migrations/registry.go:17-19]`。

**连带效应（需在验收报告登记，不必额外改代码）**：`migration_225` 每次启动会用 `ActiveMigrations` 批量 seed 记录行 `[VERIFIED: internal/core/db/migrations/migration_225_schema_migrations_table.go:57-74]`，因此 226 会在 225 跑完后被写入 `sys_schema_migrations`（即使 226 本身失败）。该表**只读不控制执行**，无功能影响。

### 2. `internal/core/db/database.go`

**职责**：启动期迁移调用块（PG 分支在 advisory-lock 内 / sqlite 分支独立）。

**为什么在本 phase 改**：新 migration 不被调用就永不执行。

**改法 —— 两个分支各加一条**：

**(a) PG 分支**：插在 `runAndRecord(224, ...)` 块之后、PG 分支的 `} else {` 之前（当前 :1154-1157 之间）：

```go
		// Phase 87 RBAC-05: 动作级 F 型权限播种 (72 条, 双方言, 幂等, 精准授权)。
		// **非阻断**：缺 F 行只让普通角色写操作 403 / 前端按钮不显示，
		// 不影响登录 / 入队 / 排水主链路；父菜单缺失时迁移内部降级返回 nil。
		if err := d.runAndRecord(226, "action_permission_f_type_seed", migrations.Migrate226ActionPermissionFTypeSeed); err != nil {
			applogger.Errorf("动作级 F 型权限播种失败 (非阻断,留待下次启动): %v", err)
		}
```

**(b) sqlite 分支**：插在 `runAndRecord(224, ...)` 块之后（当前 :1240-1242 之后）、`ensureSQLiteReconciliationViews()` 之前：

```go
		// sqlite 分支: Phase 87 动作级 F 型权限播种 (与 PG 分支同款调用 + 同款非阻断档)。
		// sqlite 既是 dev 运行时也是测试运行时；只注册 PG 分支会让 dev 环境缺权限行。
		if err := d.runAndRecord(226, "action_permission_f_type_seed", migrations.Migrate226ActionPermissionFTypeSeed); err != nil {
			applogger.Errorf("动作级 F 型权限播种失败 (非阻断,留待下次启动): %v", err)
		}
```

**逐字依据（非阻断档位）**：
- PG 分支同款注释逐字：`// **非阻断**：缺权限行只让普通角色对账端点 403 / 前端 Tab 不渲染，// 不影响登录 / 入队 / 排水主链路；父菜单缺失时迁移内部降级返回 nil。` `[VERIFIED: internal/core/db/database.go:1118-1119]`
- sqlite 分支同款注释逐字：`// sqlite 既是 dev 运行时也是测试运行时；只注册 PG 分支会让 dev 环境缺权限行。` `[VERIFIED: internal/core/db/database.go:1214]`
- `runAndRecord` 语义：fn 的错误**原样返回**；记录 upsert 失败只 WARN 不阻断；**不做任何 skip / 版本比较**（迁移每次启动无条件重跑 ⇒ 幂等是硬要求）`[VERIFIED: internal/core/db/database.go:1318-1319 / 1320-1335]`。

### 3. `ruoyi-react-frontend/scripts/check-permission-codes.mjs`

**职责**：TS ↔ Go 常量 ↔ `sys_menu` 菜单种子的四类漂移双向校验（a/b/c/d）。

**为什么在本 phase 改**：RBAC-06 要求「校验脚本覆盖本轮新铺开面并接入 CI 门禁」。当前脚本有**两个盲区**必须补：
1. **白名单永久化**：78 项白名单只输出警告、仍 exit 0 `[VERIFIED: ruoyi-react-frontend/scripts/check-permission-codes.mjs:449-455]`。
2. **Go 侧不分 F/C**：Go migration 的所有权限码字面量一律进 `untyped`（c 类存在性），**永不进 `fType`**（d 类只对 SQL 生效）`[VERIFIED: check-permission-codes.mjs:374-381]` ⇒ Go 播种的 F 行永远进不了 d 类。

**改法（三处，精确到行）**：

**(a) 清空白名单**（替换 :46-143 整段注释 + 数组内容）：

```js
/**
 * 新码待播种白名单 —— Phase 87 播种完成后**永久清空**。
 *
 * 85-02（RBAC-04）拆分出的一批新码在播种 migration 落地前需要临时豁免 c 类；
 * Phase 87 已落地播种 migration（internal/core/db/migrations/migration_226_action_permission_f_type_seed.go），
 * 白名单必须为空且**由脚本硬断言守护**（非空 ⇒ exit 1），防止漂移被永久掩盖。
 *
 * 如需临时豁免，必须同批在 .planning/phases/87-permission-grant/87-STATE.md 登记。
 */
const PENDING_SEED_WHITELIST = [];

const PENDING_SEED_SET = new Set(PENDING_SEED_WHITELIST.map((w) => w.code));
```

**保留 `PENDING_SEED_SET` 与 c 类的白名单分支**（`[VERIFIED: check-permission-codes.mjs:145]` 与 `:430-433]`）—— 清空后该分支永不触发，删掉反而破坏结构一致性。

**(b) 白名单非空 ⇒ 硬失败**（替换 :449-455）：

```js
  // 白名单硬断言（RBAC-06 防永久化）：87 播种已落地，白名单必须为空。
  // 非空即 exit 1 —— 保留「仅警告」会让 c 类漂移被永久掩盖。
  if (PENDING_SEED_WHITELIST.length > 0) {
    errors.push(
      `PENDING_SEED_WHITELIST 非空（${PENDING_SEED_WHITELIST.length} 项）—— Phase 87 已播种，白名单必须永久清空` +
        `（如需临时豁免须同批在 87-STATE.md 登记）`,
    );
  }
```

**位置不变**（在 `if (errors.length > 0)` 判定之前，即 :457 之前）⇒ 直接进 `errors` 即 exit 1。

**(c) Go 侧 F 型声明块解析**（在 :374-381 的 Go 分支内追加）：

```js
    } else {
      // Go migration：权限码以字符串字面量出现（类型未知 → 仅参与存在性判断）
      const re = /["']([a-z][a-z0-9_]*(?::[A-Za-z0-9_]+)+)["']/g;
      let m;
      while ((m = re.exec(text)) !== null) {
        menuUntyped.add(m[1]);
      }
      // Phase 87 RBAC-06: 「F_TYPE_SEED_BEGIN/END」块内的码视为
      // **已在 sys_menu 声明的 F 型按钮** → 进 menuFType（参与 d 类校验），
      // 而不是无类型的 untyped（仅参与 c 类存在性）。
      const fBlockRe = /\/\/ F_TYPE_SEED_BEGIN[\s\S]*?\/\/ F_TYPE_SEED_END/g;
      let fb;
      while ((fb = fBlockRe.exec(text)) !== null) {
        const innerRe = /["']([a-z][a-z0-9_]*(?::[A-Za-z0-9_]+)+)["']/g;
        let bm;
        while ((bm = innerRe.exec(fb[0])) !== null) {
          menuFType.add(bm[1]);
        }
      }
    }
```

**(d) 【必需】放宽权限码正则的段内字符类（连字符）** —— **这是清空白名单的硬前提**

**问题（本会话实测发现，研究文档未覆盖）**：`ops:floor-plan:edit` **含连字符**，而现有两处正则的段内字符类都不含 `-`：

- `const PERM_LITERAL = /^[a-z][a-z0-9_]*(?::[A-Za-z0-9_]+)+$/;` `[VERIFIED: check-permission-codes.mjs:44]`
- Go 侧扫描：`const re = /["']([a-z][a-z0-9_]*(?::[A-Za-z0-9_]+)+)["']/g;` `[VERIFIED: check-permission-codes.mjs:376]`

⇒ 该码**既进不了 `menuUntyped`（c 类存在性）也进不了 `menuFType`（d 类）** ⇒ **清空白名单后它的 c 类必然 exit 1**，无论 migration 里写不写这个字面量。

**本会话实测（临时探针，跑完已删除）**：

| 度量 | 结果 |
|---|---|
| 72 码在**原** `PERM_LITERAL` 下的漏网 | **1 个** → `["ops:floor-plan:edit"]` |
| 72 码在**原** Go 侧正则下的捕获 | **71 / 72**（漏 `ops:floor-plan:edit`） |
| 72 码在**放宽后**两条正则下的漏网 | **0 个**（72/72 全部命中） |
| 放宽后 `PERM_LITERAL` 在整个 `internal/core/db/migrations/` 目录**新接受**的字面量 | **0 个** ⇒ 零误报风险（SQL 侧 `parseSqlMenuPerms` 与 Go 侧均无新增命中） |

**改法（两处必改 + 一处新增，一并放宽）**：

```js
// :44 —— 段内允许连字符（ops:floor-plan:edit）
const PERM_LITERAL = /^[a-z][a-z0-9_]*(?::[A-Za-z0-9_-]+)+$/;

// :376 —— Go 侧扫描正则同步放宽
const re = /["']([a-z][a-z0-9_]*(?::[A-Za-z0-9_-]+)+)["']/g;

// (c) 新增的 F 型块 innerRe —— 同样放宽
const innerRe = /["']([a-z][a-z0-9_]*(?::[A-Za-z0-9_-]+)+)["']/g;
```

**验证方法（planner 必须跑）**：改完后 `node scripts/check-permission-codes.mjs` 必须 **exit 0 且 `✗` 计数 = 0**。本会话已用脚本副本完整模拟「清白名单 + F 块注入 72 码 + 放宽正则」组合，结果 `EXIT=0`、漂移 `0` 项（该组合未引入任何 d 类误报）。

**判据效果（planner 必须验证的四条）**：
- c 类：`menuAllScope = menuFType ∪ menuUntyped` `[VERIFIED: check-permission-codes.mjs:408-410]` ⇒ 72 码在 Go 文件里有字面量 ⇒ 自动满足 c 类。
- d 类：`menuScope` 从 `menuFType` 取 `[VERIFIED: check-permission-codes.mjs:407]` ⇒ 72 码进 d 类 ⇒ 若某个码从 TS 移除即红。
- **F 块内只能放 72 条动作码**：放父 perms（如 `system:role`）不会误报（scope 过滤为 `system:role:`，父码不以该前缀开头），但放**非 72 码**（如 `ops:floor-plan:list` —— 该码 TS 未声明）会立刻触发 d 类 exit 1。
- `collectMigrationFiles` 会遍历 `internal/core/db/migrations/` 下所有 `.go` / `.sql`（跳过 `archive/`）`[VERIFIED: check-permission-codes.mjs:336-356]` ⇒ 新 migration 文件放在该目录即被扫到，**不需要**改扫描逻辑。

### 4. `ruoyi-react-frontend/package.json`

**职责**：npm scripts 定义（`lint` / `lint:ci` 等）。

**为什么在本 phase 改**：RBAC-06 要求「接入 CI 门禁」；当前 `lint:ci` 链**不含** `check-permission-codes.mjs`。

**改法**：把 :14 的 `lint:ci` 整行替换为（**在 `check-gating-coverage.mjs` 之后插入**，保持与 86 的门控覆盖脚本相邻）：

```json
    "lint:ci": "eslint . --max-warnings 0 && node scripts/assert-lint-budget.mjs && node scripts/check-exemption-reasons.mjs && node scripts/check-gating-coverage.mjs && node scripts/check-permission-codes.mjs && node scripts/check-hardcoded-colors.mjs && node scripts/css-contract-check.js && node ../scripts/check-route-registry.mjs",
```

**既有做法依据（脚本怎么被挂进 `lint:ci` 的范式）**：仓库根 `scripts/check-route-registry.mjs` **已经在 `lint:ci` 链尾**，调用形式是 `node ../scripts/check-route-registry.mjs`（相对 frontend 目录）`[VERIFIED: ruoyi-react-frontend/package.json:14]`；该脚本自身注释逐字：`// 本脚本位于仓库根 scripts/ 目录（由 frontend lint:ci 以 \`node ../scripts/check-route-registry.mjs\` 调用），// 仓库根 = 本脚本目录的上一级。` `[VERIFIED: scripts/check-route-registry.mjs:24-25]`。

⇒ **87 的 `check-permission-codes.mjs` 在 frontend 目录内**（`ruoyi-react-frontend/scripts/`），故调用形式是 `node scripts/check-permission-codes.mjs`（**不带** `../`）。它是「frontend 目录内 + `node scripts/xxx.mjs`」这一路的样例，而 `check-route-registry.mjs` 是「仓库根 + `node ../scripts/xxx.mjs`」那一路的样例。**两条路都已在 `lint:ci` 链里有先例，照抄即可。**

**⚠️ 路径正确性已实证**：`check-permission-codes.mjs` 用 `const repoRoot = path.resolve(frontendRoot, "..");` 定位仓库根 `[VERIFIED: check-permission-codes.mjs:37]`，`MIGRATIONS_DIR = path.join(repoRoot, "internal", "core", "db", "migrations")` `[VERIFIED: check-permission-codes.mjs:41]` ⇒ 在 CI（`working-directory: ruoyi-react-frontend`）下也能找到 `internal/core/db/migrations`。

### 5. `.github/workflows/frontend-build.yml`

**职责**：前端 CI 门禁 workflow（Lint / Type check / Test / Build 四步）。

**为什么在本 phase 改**：当前 Lint 步只跑 `npm run lint`（**不含**任何 `scripts/*.mjs` 校验）⇒ 即使 `lint:ci` 加了脚本，CI 也不执行。

**改法**：把 :44-45 替换为：

```yaml
      - name: Lint (lint:ci)
        run: npm run lint:ci
```

**逐字依据（当前状态）**：
```yaml
      - name: Lint
        run: npm run lint
```
`[VERIFIED: .github/workflows/frontend-build.yml:44-45]`

**工作目录无风险**：该 workflow 的 `defaults.run.working-directory: ruoyi-react-frontend` `[VERIFIED: .github/workflows/frontend-build.yml:23-25]` ⇒ `node ../scripts/check-route-registry.mjs` 与 `node scripts/check-permission-codes.mjs` 两条相对路径都能解析。

**注**：本仓 `.github/workflows/` 只有 `frontend-build.yml` 一个 workflow `[CITED: 87-RESEARCH.md:529]`，故「接入 CI 门禁」只需改这一个文件。

### 6. `scripts/check-ci-local.sh`

**职责**：本地复刻 CI 判定步骤（backend 段 + frontend 段）。

**为什么在本 phase 改**：保持「本地绿 = CI 绿」——workflow 改跑 `lint:ci` 后，本地复刻脚本必须同步。

**改法**：把 :117 的一行改掉：

```bash
  for s in format:check lint lint:ci type-check test build; do
```

**逐字依据（当前状态）**：`  for s in format:check lint type-check test build; do` `[VERIFIED: scripts/check-ci-local.sh:117]`

**⚠️ 副作用（须知悉，不阻塞）**：`lint:ci` 内含 `eslint . --max-warnings 0`，与原有的 `lint` 条目重复执行一次 eslint —— 本地会慢几秒，**语义无冲突**。脚本用 `npm run "$s" --silent` 逐条跑，任一条失败即 `fail` 并最终 `exit 1` `[VERIFIED: scripts/check-ci-local.sh:117-120 / 124-131]`。

---

## Files Not to Touch

> 本 phase 的**红线**。改动任一项即视为超出 Phase 87 范围。

### A. 前端真相源（87 不新增任何权限码）

| 文件 | 理由（逐字依据） |
|---|---|
| `ruoyi-react-frontend/src/constants/permissionCodes.ts` | 文件头逐字：`// 单一真相源（v1.24 PERM-06）。` + `// 校验：\`node scripts/check-permission-codes.mjs\` 做**双向**校验` `[VERIFIED: ruoyi-react-frontend/src/constants/permissionCodes.ts:2-9]`。**78 个码已全部声明**（TS 78/78 齐 `[CITED: 87-RESEARCH.md:509]`）⇒ 87 无需新增任何码。文件内 85-02 注释已预告：`// ⚠️ 这些码的 sys_menu F 型菜单当前为 0 声明（Phase 87 播种后才有授权来源），// c 类由 check-permission-codes.mjs 的 PENDING_SEED_WHITELIST 显式豁免；// 87 播种 migration 落地后必须清空该白名单。` `[VERIFIED: permissionCodes.ts:33-35]` ⇒ 87 的动作是**清白名单**，不是改 TS。 |

### B. 86 线的门控覆盖恒等式

| 文件 | 理由（逐字依据） |
|---|---|
| `ruoyi-react-frontend/scripts/gating-exemptions.json` | 51 条豁免白名单。`check-gating-coverage.mjs` 的硬断言逐字：`// ① 覆盖率恒等式：已门控（不含豁免）+ 豁免 == EXPECTED_PAGE_COUNT（76）；` `[VERIFIED: ruoyi-react-frontend/scripts/check-gating-coverage.mjs:35]`。改豁免条目会破坏 `25 + 51 = 76` 恒等式。 |
| `ruoyi-react-frontend/scripts/check-gating-coverage.mjs` | 两个口径常量逐字：`const EXPECTED_PAGE_COUNT = 76;` / `const EXPECTED_EXEMPTION_COUNT = 51;` `[VERIFIED: check-gating-coverage.mjs:61-63]`。87 消解 **0** 条豁免 `[CITED: 87-RESEARCH.md:794]`。 |

### C. v1.24 三态门控契约回归基线

| 文件 | 理由 |
|---|---|
| `ruoyi-react-frontend/src/pages/system/role/__tests__/permission.gate.test.tsx` | **只读，不得改动**。文件头逐字：`// ⚠️ 禁止用 \`permissions.length === 0\` 判「未就绪」——空权限集与未加载完在数据上 //    不可区分。` `[VERIFIED: permission.gate.test.tsx:9-11]`。87-NN-13 的自动化断言就是「跑这个文件且它零改动」`[CITED: 87-VALIDATION.md:57]`。 |

### D. 既有验收报告（不得回改历史结论）

| 文件 | 理由 |
|---|---|
| `.planning/phases/86-frontend-gating/86-VERIFICATION.md`（及 85/84 同类报告） | 历史 phase 的验收结论是只读证据；87 的结论写进 `87-VERIFICATION.md`。 |

### E. 后端权限链路（只读消费，链路已通电）

| 文件 | 理由（逐字依据） |
|---|---|
| `pkg/middleware/permission.go` | 放行判定点逐字：`WHERE ur.user_id = ? AND m.perms = ? AND m.status = ?` `[CITED: 87-RESEARCH.md:626-630]`。已由 v1.24 Phase 78 落地，**勿重做** `[CITED: 87-RESEARCH.md:133]`。 |
| `pkg/permission/config.go` | 权限码常量 223 个已齐（78/78 新码已有）`[CITED: 87-RESEARCH.md:509]`。87 **不新增码**。 |
| `internal/services/system/menu_service.go` | `GetTree` 无 `menu_type` 过滤 ⇒ F 行已在角色页可见 `[CITED: 87-RESEARCH.md:605]`。87 不改后端授权链路。 |
| `internal/services/system/role_service.go` | `assignRoleMenusAndDepts` 写 `sys_role_menu` 已就绪 `[CITED: 87-RESEARCH.md:622]`。 |
| `internal/api/v1/system/role_router.go` / `role_handler.go` | 授权端点已存在（`POST /system/roles/:id/update`）`[CITED: 87-RESEARCH.md:558]`。 |
| `internal/models/base.go` / `menu.go` / `role.go` | 枚举与模型不改。逐字：`MenuTypeButton MenuType = "F" // 按钮` `[VERIFIED: internal/models/base.go:86]`、`VisibleHidden VisibleType = 0 // 隐藏` `[VERIFIED: internal/models/base.go:94]`、`Perms *string \`gorm:"size:100" json:"perms,omitempty"\`` `[VERIFIED: internal/models/menu.go:71]`、`RoleID string \`gorm:"type:uuid;not null;index:idx_sys_role_menu_role_id_menu_id" json:"roleId"\`` `[VERIFIED: internal/models/role.go:25]`。 |

### F. 既有迁移与 helper（不改，只读）

| 文件 | 理由 |
|---|---|
| `internal/core/db/migrations/migration_221_manager_sync_menus.go` | 已播种 `ops:manager:{rule,org}:{edit,delete}` 四个 F 行 + 两个 C 型菜单 `[CITED: 87-RESEARCH.md:482-483]`。87 **不重复播种**这 6 条（避免同 perms 重复行，见 OQ-2）。 |
| `internal/core/db/migrations/migration_202_port_write_audit.go` | PG-only + 按 `menu_name` 定位 + `GrantNewMenuToRolesHavingParent` —— 是**反面范式**（见 P-7）。 |
| `internal/core/db/migrations/menu_grant_helpers.go` | **仅作反面参考**（见 P-7）。 |
| `internal/core/db/migrations/menu_catalog_seed.sql` | 239 条菜单种子的入库产物来源；87 只新增 F 行，不改种子文件。 |
| `internal/core/db/migrations/migration_225_schema_migrations_table.go` | 记录表只读不控制执行（见 registry.go 条目下的连带效应说明）。 |

### G. 前端角色页（授权 UI 已就绪，零改动）

| 文件 | 理由（逐字依据） |
|---|---|
| `ruoyi-react-frontend/src/pages/system/role/index.tsx` | 已有 antd `Tree`（**不是** TreeSelect）`[CITED: 87-RESEARCH.md:555]`；`onCheck` 已把 key 写入 `editForm.setFieldsValue({ menuIds: keyArray })` `[VERIFIED: ruoyi-react-frontend/src/pages/system/role/index.tsx:442-446]`。87 不改。 |
| `ruoyi-react-frontend/src/pages/system/role/hooks/useRoleActions.ts` | `menuIds: (values.menuIds as string[]) || []` 已进提交体 `[VERIFIED: ruoyi-react-frontend/src/pages/system/role/hooks/useRoleActions.ts:190]`。 |
| `ruoyi-react-frontend/src/pages/system/role/hooks/useRoleData.ts` | `processTreeData` 只映射 key/title/value/children，不按 menuType 过滤 `[VERIFIED: ruoyi-react-frontend/src/pages/system/role/hooks/useRoleData.ts:103-111]`。 |

---

## Patterns to Follow

> 共 **7** 条。每条给出「源文件:行号」+ 可直接复制的骨架。

### P-1 · migration 双方言骨架（照 `migration_218_ad_reconciliation_permission.go` + `migration_221_manager_sync_menus.go`）

**源**：`internal/core/db/migrations/migration_218_ad_reconciliation_permission.go:39-108`（单条 F 行 + 精准授权）与 `migration_221_manager_sync_menus.go:60-127, 167-197`（种子表 + 逐条 ensure）。

**签名与事务边界**：
- 顶层入口 `func Migrate226ActionPermissionFTypeSeed(db *gorm.DB) error` —— **无显式事务**（migration_218 / 221 都没有）；「事务边界」= 每条种子独立成败，单条失败 `log.Printf` + `continue`。
- 入口第一行固定 `log.Println("Running migration 226: action-level F type permission seed")`（照 :40 范式）。
- 入口最后一行固定 `log.Println("Migration 226 completed: action-level F type permissions seeded")` + `return nil`（照 :107-108 范式）。

**幂等写法（按 perms 查 → 命中复用 ID）**：逐字范式
```go
	var existing models.Menu
	permErr := db.Where("perms = ?", adReconciliationListPerms).First(&existing).Error
	hasPermRow := permErr == nil
	if permErr != nil && !errors.Is(permErr, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询既有对账权限行失败: %w", permErr)
	}
```
`[VERIFIED: migration_218:42-48]`

⇒ **87 的加强版**（加 `menu_type='F'` 约束，因为库内存在同 perms 的 C 型行，如 `ops:manager:rule:list`）：

```go
// ensureActionPermissionMenu 幂等创建 / 复用一个 F 型动作权限行，返回菜单 ID。
//
// 幂等口径：**按 perms + menu_type='F' 查**。
// migration_218:44 只按 perms 查（无 type 约束），87 加严 —— 库内存在同 perms 的
// C 型行（如 ops:manager:rule:list 是 C 型页面本体码），不加约束会误复用 C 行。
func ensureActionPermissionMenu(db *gorm.DB, seed actionPermissionSeed, parentID string) (string, error) {
	var existing models.Menu
	err := db.Where("perms = ? AND menu_type = ?", seed.perms, models.MenuTypeButton).First(&existing).Error
	if err == nil {
		return existing.ID, nil // 命中既有行 → 复用 ID，跳过创建（授权照常执行）
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return "", fmt.Errorf("查询动作权限 %s 失败: %w", seed.perms, err)
	}

	emptyPath := ""
	icon := "#"
	perms := seed.perms
	button := &models.Menu{
		MenuName: seed.menuName,
		ParentID: &parentID,
		Path:     &emptyPath,              // F 型 path 不参与 routeGenerator
		MenuType: models.MenuTypeButton,   // 'F'
		Visible:  models.VisibleHidden,    // 0 —— ⚠️ 见下方「visible 落库实测」
		Status:   models.MenuStatusNormal, // 0
		Perms:    &perms,                  // 单码
		Icon:     &icon,
		OrderNum: seed.orderNum,
		Remark:   "Phase 87: 动作级权限按钮",
	}
	if err := db.Create(button).Error; err != nil {
		return "", fmt.Errorf("创建动作权限 %s 失败: %w", seed.perms, err)
	}
	// ⚠️ GORM 对「零值 + default tag」字段会省略该列（见下方实测），
	//    Visible 的 tag 是 default:1 ⇒ Create 后库里是 visible=1。
	//    AGENTS.md 要求播种的 F 行 visible=0（隐藏）⇒ 立即显式归零。
	if err := db.Model(&models.Menu{}).Where("id = ?", button.ID).
		Update("visible", models.VisibleHidden).Error; err != nil {
		return "", fmt.Errorf("归零动作权限 %s 的 visible 失败: %w", seed.perms, err)
	}
	return button.ID, nil
}
```

**⚠️ visible 落库实测（本会话新增实证，研究文档未覆盖）**

`models.Menu.Visible` 的 tag 是 `gorm:"default:1;index:idx_sys_menu_parent_status_visible"` `[VERIFIED: internal/models/menu.go:69]`。GORM 对「零值 + 带 default tag」的字段在 INSERT 时**省略该列**，让 DB 落默认值。实测（本会话在 `internal/core/db/migrations/` 内存 sqlite 上跑临时探针，跑完已删除探针文件）：

| 建行方式 | 实际落库 `visible` | 实际落库 `status` |
|---|---|---|
| `db.Create(button)`（migration_218 / 202 / 221 同款） | **1** | 0 |
| `db.Select("*").Create(button)` | **1** | 0 |
| 原生 SQL `INSERT ... visible, status) VALUES (..., 0, 0, ...)` | **0** | 0 |

⇒ **三条硬推论（planner 必须照办）**：
1. **migration_218 / 202 / 221 播种的 F 行，库里 `visible` 实际是 1（显示），不是 0**。migration_218 自己的注释也承认了：`// 注：Visible/Status 为零值 + 模型带 default tag 时 GORM 会省略列、落 DB 默认值，// 与 migration_163 同款行为；按钮（F 型）的可见性不参与前端渲染判定，无语义影响。` `[VERIFIED: migration_218:85-86]`。
2. **AGENTS.md 的 `Exception - Menu visibility uses 1 = visible, 0 = hidden` + 播种 F 行须 `visible=0` 的要求，用 `db.Create` 达不到** ⇒ 87 必须显式归零（本 PATTERNS 推荐 `Create` + `Update("visible", 0)`；等价方案是原生 INSERT，但要自己填 UUID）。
3. `Status` 无此问题（tag `default:0`，零值即 0，三条路径实测都是 0）`[VERIFIED: internal/models/menu.go:70]`。

**该项不影响权限判定**（`visible` 只影响导航栏显示，F 行因 `menu_type IN ('M','C')` 过滤不进导航树 `[CITED: 87-RESEARCH.md:839]`），但 AGENTS.md 是硬约束 ⇒ 归零。

**父菜单解析（按 perms 精确定位）**：

```go
// resolveActionPermissionParents 按 perms 精确定位父 C 型菜单集合。
//
// 解析规则：perms = <父码> AND menu_type = 'C' AND status = 0
//   - 命中 0 条 → 降级跳过该条（非阻断）
//   - 命中 1 条 → 唯一父
//   - 命中 ≥2 条 → 父集合全部（授权按「持有任一父」，F 行挂第一条）
//
// 特例（parentPerms 为空时）：按 menu_name + menu_type='C' + component 定位 ——
//   ops:asset 的父「资产列表」perms 为 NULL（migration_147 建时不写 Perms），
//   component = 'operations/assets/index'。
func resolveActionPermissionParents(db *gorm.DB, seed actionPermissionSeed) ([]models.Menu, error) {
	var parents []models.Menu
	q := db.Model(&models.Menu{}).
		Where("menu_type = ? AND status = ?", models.MenuTypeMenu, models.MenuStatusNormal)
	if seed.parentPerms != "" {
		q = q.Where("perms = ?", seed.parentPerms)
	} else {
		q = q.Where("menu_name = ?", seed.parentName)
		if seed.parentComponent != "" {
			q = q.Where("component = ?", seed.parentComponent)
		}
	}
	// Order("id ASC")：命中多条时确定「挂哪一条」，与 GORM First 的主键升序口径一致
	if err := q.Order("id ASC").Find(&parents).Error; err != nil {
		return nil, fmt.Errorf("查询父菜单失败(%s): %w", seed.perms, err)
	}
	return parents, nil
}
```

**⚠️ 禁止按 `menu_name` 定位父菜单（越权风险）**：库内每个 ops 域 C 菜单都有**两份同名同 perms 的副本**（live `status=0` / stopped `status=1`），按名字授权会命中停用旧菜单 `[CITED: 87-RESEARCH.md:352-358]`。

**错误非阻断处理（逐字范式）**：
```go
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("Warning: 未找到 AD 域控配置父菜单(perms=%s), migration_218 跳过创建与授权 "+
				"(部署清单须管理员手建按钮权限并授权): %v", adConfigListPerms, err)
			return nil
		}
		return fmt.Errorf("查询 AD 域控配置父菜单失败: %w", err)
	}
```
`[VERIFIED: migration_218:58-65]`

⇒ 87 的逐条版本（**单条失败不阻断其余 71 条**，逐字照抄研究 §Code Examples E1 的形态）：

```go
	for _, seed := range actionPermissionSeeds {
		parents, err := resolveActionPermissionParents(db, seed)
		if err != nil {
			log.Printf("Migration 226: WARNING 查询父菜单失败, 跳过 %s (非阻断): %v", seed.perms, err)
			continue
		}
		if len(parents) == 0 {
			log.Printf("Migration 226: WARNING 父菜单未找到, 跳过 %s (非阻断, 留待下次启动)", seed.perms)
			continue
		}

		menuID, err := ensureActionPermissionMenu(db, seed, parents[0].ID)
		if err != nil {
			log.Printf("Migration 226: WARNING 创建 %s 失败 (非阻断): %v", seed.perms, err)
			continue
		}

		parentIDs := make([]string, 0, len(parents))
		for _, p := range parents {
			parentIDs = append(parentIDs, p.ID)
		}
		if err := grantActionPermissionToParentHolders(db, menuID, parentIDs); err != nil {
			log.Printf("Migration 226: WARNING 授权 %s 失败 (非阻断): %v", seed.perms, err)
			continue
		}
	}
```

**⚠️ 入口级硬断言（F 型声明块 ↔ 种子表逐条一致）**：`actionPermissionFTypeSeedCodes` 与 `actionPermissionSeeds` 是两份数据，必须锁死一致性，否则校验脚本的 d 类与实际播种脱节。在 migrate 入口加：

```go
	// 硬断言：F 型声明块（供 check-permission-codes.mjs 的 d 类消费）与种子表逐条一致。
	// 两者漂移 ⇒ 脚本认为已播种、库里却没有（或反之）。
	if len(actionPermissionFTypeSeedCodes) != len(actionPermissionSeeds) {
		return fmt.Errorf("F 型声明块(%d 条) 与种子表(%d 条) 条数不一致",
			len(actionPermissionFTypeSeedCodes), len(actionPermissionSeeds))
	}
	for i, s := range actionPermissionSeeds {
		if s.perms != actionPermissionFTypeSeedCodes[i] {
			return fmt.Errorf("第 %d 条种子 perms=%q 与 F 型声明块 %q 不一致",
				i, s.perms, actionPermissionFTypeSeedCodes[i])
		}
	}
```

### P-2 · migration 测试骨架（照 `migration_218_ad_reconciliation_permission_test.go:23-70`）

**源**：`internal/core/db/migrations/migration_218_ad_reconciliation_permission_test.go:23-90`（建表 DDL + 三个 helper）。

**照抄四件套**：

```go
// ① 建表 DDL 变量（原生 SQL，**不走 GORM AutoMigrate**）
var migrate226Statements = []string{
	`CREATE TABLE sys_menu (
	    id TEXT PRIMARY KEY,
	    menu_name TEXT,
	    parent_id TEXT,
	    order_num INTEGER DEFAULT 0,
	    path TEXT,
	    component TEXT,
	    menu_type TEXT DEFAULT 'M',
	    visible INTEGER DEFAULT 1,
	    status INTEGER DEFAULT 0,
	    perms TEXT,
	    icon TEXT,
	    remark TEXT DEFAULT '',
	    meta TEXT,
	    created_by TEXT DEFAULT '',
	    updated_by TEXT DEFAULT '',
	    version INTEGER DEFAULT 0,
	    created_at DATETIME,
	    updated_at DATETIME,
	    deleted_at DATETIME
	)`,
	`CREATE TABLE sys_role (
	    id TEXT PRIMARY KEY,
	    role_name TEXT,
	    status INTEGER DEFAULT 0
	)`,
	`CREATE TABLE sys_role_menu (
	    role_id TEXT,
	    menu_id TEXT
	)`,
}

// ② 内存 sqlite 库
func freshSQLiteDBForMigrate226(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	for i, stmt := range migrate226Statements {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("建表 %d 失败: %v", i, err)
		}
	}
	return db
}

// ③ 批量执行 SQL（任一条失败即终止用例）
func execMigrate226(t *testing.T, db *gorm.DB, stmts []string) {
	t.Helper()
	for i, stmt := range stmts {
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("执行第 %d 条 SQL 失败: %v\nSQL: %s", i, err, stmt)
		}
	}
}

// ④ 单值 COUNT
func countMigrate226Rows(t *testing.T, db *gorm.DB, query string, args ...interface{}) int64 {
	t.Helper()
	var n int64
	if err := db.Raw(query, args...).Scan(&n).Error; err != nil {
		t.Fatalf("统计失败 (%s): %v", query, err)
	}
	return n
}
```

**逐字依据**：`dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"` `[VERIFIED: migration_218_test.go:59]`；三张表的 DDL `[VERIFIED: migration_218_test.go:24-53]`；三个 helper `[VERIFIED: migration_218_test.go:57-90]`。

**必写的 6 个用例（对齐 `87-VALIDATION.md` 的 87-NN-01..05 + P-1 的 visible 实测）**：

| 用例名 | 断言 | 对应 87-NN |
|---|---|---|
| `TestMigrate226_SeedsAllCodes` | 72 码在 `sys_menu` 各有且仅有 1 行，且 `menu_type='F' AND status=0 AND visible=0 AND perms 单码` | 01 |
| `TestMigrate226_Idempotent` | 连跑 2 次 ⇒ `sys_menu` 行数不变、`sys_role_menu` 行数不变 | 02 |
| `TestMigrate226_GrantsOnlyParentHolders` | 持父 C 菜单的角色得授权；不持的不得；**停用旧菜单（status=1）不得命中** | 03 |
| `TestMigrate226_MissingParentDegradesGracefully` | 返回 nil、不 panic、`sys_menu` 行数不变、`sys_role_menu` 行数不变 | 04 |
| `TestMigrate226_MultiParent` | `network:command` 的 2 个父 C 菜单，持有任一者都获授权 | 05 |
| `TestMigrate226_VisibleIsZero` | 新建的 F 行 `visible = 0`（锁住 P-1 的归零逻辑，防止回退成 GORM 默认 1） | 新增（AGENTS.md P1） |

**造数范式（照抄）**：
```go
	execMigrate218(t, db, []string{
		// 父菜单形态对齐真实库：menu_name='域控配置' + perms='ops:ad:config:list'
		`INSERT INTO sys_menu (id, menu_name, menu_type, visible, status, path, order_num, perms)
		 VALUES ('menu-adcfg', '域控配置', 'C', 1, 0, 'ad-config', 3, 'ops:ad:config:list')`,
		`INSERT INTO sys_role (id, role_name, status) VALUES ('role-a', 'AD管理员', 0)`,
		`INSERT INTO sys_role (id, role_name, status) VALUES ('role-b', '普通角色', 0)`,
		// 只有角色 A 关联了父菜单
		`INSERT INTO sys_role_menu (role_id, menu_id) VALUES ('role-a', 'menu-adcfg')`,
	})
```
`[VERIFIED: migration_218_test.go:95-104]`

**清理范式**：migration 测试**不清理**（每次用例独立内存库，`dsn` 带 `t.Name()` 隔离）。若在同一内存库里跑多轮，用 `countMigrate226Rows` 做前后差值断言（照 `TestPhase59_Migration218_MissingParentAfterCreation` 的 `before` / `after` 写法）`[VERIFIED: migration_218_test.go:189-201]`。

### P-3 · 精确授权写法（照 `migration_218` / `migration_221` 的 `NOT EXISTS`）

**源**：`migration_218:96-105` 与 `migration_221:200-211`（两者 SQL 完全同构）。

**单父版本逐字**：
```go
	if err := db.Exec(
		`INSERT INTO sys_role_menu (role_id, menu_id)
		 SELECT rm.role_id, ? FROM sys_role_menu rm
		 WHERE rm.menu_id = ?
		   AND NOT EXISTS (
		       SELECT 1 FROM sys_role_menu x
		       WHERE x.role_id = rm.role_id AND x.menu_id = ?)`,
		menuID, parent.ID, menuID).Error; err != nil {
		return fmt.Errorf("精准授权对账查询按钮权限失败: %w", err)
	}
```
`[VERIFIED: migration_218:96-105]`

**87 的多父版本（`network:command` 有 2 个父）**：

```go
// grantActionPermissionToParentHolders 双方言幂等精准授权：
// 授予「持有父 C 型菜单任一」的角色。
//
// ⚠️ 禁止使用 GrantNewMenuToRolesHavingParent（见 P-7）。
// ⚠️ 禁止使用 ON CONFLICT DO NOTHING —— 依赖 (role_id, menu_id) 唯一约束，
//    而 models/role.go:24-27 的 GORM tag 只是普通索引（非 unique），
//    生产 PG 是否真有该唯一约束未实证（87-RESEARCH OQ-1）⇒ 一律用 NOT EXISTS。
func grantActionPermissionToParentHolders(db *gorm.DB, menuID string, parentIDs []string) error {
	if len(parentIDs) == 0 {
		return fmt.Errorf("父菜单集合为空,拒绝授权 %s", menuID)
	}
	return db.Exec(
		`INSERT INTO sys_role_menu (role_id, menu_id)
		 SELECT rm.role_id, ? FROM sys_role_menu rm
		 WHERE rm.menu_id IN ?
		   AND NOT EXISTS (
		       SELECT 1 FROM sys_role_menu x
		       WHERE x.role_id = rm.role_id AND x.menu_id = ?)`,
		menuID, parentIDs, menuID).Error
}
```

**⚠️ `IN ?` 切片展开已本会话实测可用（本会话新增实证）**：在内存 sqlite 上跑临时探针（跑完已删除探针文件）验证：

| 场景 | 实测结果 |
|---|---|
| 2 个父 + 2 个角色（各持一个父）+ `IN ?` 展开 | 授权行数 = **2**（持有任一父都获授）✔ |
| 同一条 SQL 重跑一次（幂等） | 授权行数 = **2**（未新增）✔ |

⇒ GORM 的 `IN ?` 在 `db.Exec` 中会展开为 `IN (?,?)`，**sqlite 与 PG 双方言通用**。

**`sys_role_menu` 无唯一约束 ⇒ 必须用 `NOT EXISTS`**：逐字依据 `RoleID string \`gorm:"type:uuid;not null;index:idx_sys_role_menu_role_id_menu_id" json:"roleId"\`` / `MenuID string \`gorm:"type:uuid;not null;index:idx_sys_role_menu_role_id_menu_id" json:"menuId"\`` `[VERIFIED: internal/models/role.go:25-26]` —— 是 `index:`（普通索引）而非 `uniqueIndex:`。

### P-4 · 校验脚本范式（照 `check-permission-codes.mjs` 自身）

**源**：`ruoyi-react-frontend/scripts/check-permission-codes.mjs`（468 行，本会话全量读过）。

**四类判定的判据位置（planner 改动时不要动这四处语义）**：

| 类别 | 语义 | 位置 |
|---|---|---|
| a | TS 声明了，但后端常量没有 → 前端永远拿不到该权限 | `[VERIFIED: :414-419]` |
| b | 后端常量有，但 TS 没声明 → 前端漏了按钮门控 | `[VERIFIED: :420-425]` |
| c | TS 声明了，但 sys_menu 未声明 → 该权限永远不会被授予 | `[VERIFIED: :426-435]` |
| d | sys_menu 声明了 F 型按钮，但 TS 没声明 → 前端漏了按钮门控 | `[VERIFIED: :436-441]` |

**作用域口径**：每个 TS 模块的权限码共享同一前缀；**前缀本身是页面本体权限（C 型），不参与比对**（比对作用域 = 前缀 + `":"`）`[VERIFIED: :18-20, :396-403]`。

**⚠️ 权限码正则的连字符陷阱（本会话实测发现）**：`PERM_LITERAL`（`:44`）与 Go 侧扫描正则（`:376`）的段内字符类 `[A-Za-z0-9_]` **不含 `-`** ⇒ `ops:floor-plan:edit` 两处都匹配不上 ⇒ 它既进不了 `menuUntyped` 也不进 `menuFType` ⇒ **清空白名单后它的 c 类必然 exit 1**。
⇒ **必须**把两处（+ 新增的 F 块 `innerRe`）的段内字符类改成 `[A-Za-z0-9_-]`。实测放宽后 72 码 72/72 全部命中，且在整个 migrations 目录**新接受的字面量 = 0 个**（零误报）。详见 Files to Modify §3(d)。

**`PENDING_SEED_WHITELIST` 结构（改前必知）**：每条三个字段 `{ code, reason, evidence }`；派生出 `PENDING_SEED_SET` `[VERIFIED: :57-145]`；**仅作用于 c 类**（命中时打印 `⏳ … 在 85 新码待播种白名单内（c 类跳过；87 播种后必须清空白名单）`）`[VERIFIED: :430-433]`。

**⚠️ 白名单当前实际行为（本会话实测 baseline）**：
```
$ cd ruoyi-react-frontend && node scripts/check-permission-codes.mjs
EXIT=0
⏳ 出现次数 = 72
⚠ PENDING_SEED_WHITELIST 非空（78 项）——仅限 85-02 拆分 → 87 播种的窗口期。…
✓ 双向零漂移（TS ↔ 后端常量 ↔ sys_menu 菜单种子）
```
⇒ **72 次 `⏳` = 72 条真正待播种的码**（靠白名单豁免 c 类）；另外 6 条 `ops:manager:*` 因 `migration_221` 的 Go 字面量已在 `untyped` 集合里，不走白名单也能过 c 类。

⇒ **清空白名单后的硬指标**：72 码必须**在 `migration_226_action_permission_f_type_seed.go` 的源码里出现字符串字面量**（`_test.go` 里的字面量也会被扫到 —— `collectMigrationFiles` 收所有 `.go` `[VERIFIED: :351]`），否则 c 类 exit 1。**建议只把字面量写在非测试文件里**，避免「测试文件让 c 类假绿」。

**脚本被挂进 `lint:ci` 的既有做法**：见 Files to Modify §4（`node scripts/xxx.mjs` 与 `node ../scripts/xxx.mjs` 两条路径都已有先例）。

**Go 侧 F 型识别块（脚本侧改法）**：见 Files to Modify §3(c)。

### P-5 · 组件测试范式（照 `permission.gate.test.tsx`，**只读不改**）

**源**：`ruoyi-react-frontend/src/pages/system/role/__tests__/permission.gate.test.tsx`（207 行，本会话全量读过）。

**三件套用法（逐字）**：

```tsx
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, waitFor, fireEvent, act } from "@testing-library/react";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { mockApiBatch, setGenericFallback } from "@/test/utils/createApiMock";
import { QueryClient } from "@tanstack/react-query";
import { useMenuStore } from "@/store/menuStore";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import RoleManagement from "../index";

/** 设置权限集与显式就绪位（构造 ready / loading / error 三态） */
function setPermissionState(
  permissions: string[],
  state: { loaded?: boolean; error?: Error } = {}
): void {
  useMenuStore.setState({
    permissions,
    permissionsLoaded: state.loaded ?? true,
    permissionsError: state.error ?? null,
  });
}

function renderRole() {
  // role 页的 useRoleActions 依赖 react-query 的缓存失效器（useQueryClient）
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(<RoleManagement />, { queryClient: qc });
}
```
`[VERIFIED: permission.gate.test.tsx:17-58]`

**⚠️ `vi.mock` 必须静态放在文件顶层**（Vitest 模块图约定：函数体内的 `vi.mock` 对已缓存模块无效）`[VERIFIED: ruoyi-react-frontend/src/test/utils/createApiMock.ts:15-27]`。

**mock 端点批次（87 新文件需要的最小集）**：
```tsx
    mockApiBatch([
      { endpoint: "/system/roles/list",
        response: { data: { list: [ROLE_ROW], total: 1, current: 1, pageSize: 10 } } },
      { endpoint: "/system/roles/statistics",
        response: { data: { total: 1, active: 1, inactive: 0 } } },
      { endpoint: "/system/menus/tree-select", response: { data: MENU_TREE_WITH_F_TYPE } },
      { endpoint: "/system/departments/tree-select", response: { data: [] } },
      { endpoint: "/system/menus/role-menu-tree-select/r1",
        response: { data: { checkedKeys: ["f-add"] } } },
      { endpoint: "/system/departments/role-dept-tree-select/r1",
        response: { data: { checkedKeys: [] } } },
    ]);
    setGenericFallback({ data: {} });
```
**依据**：
- 四个基础端点照抄 `permission.gate.test.tsx:70-81`。
- `/system/menus/role-menu-tree-select/{roleId}` 的调用点逐字：`const result = (await post(\`/system/menus/role-menu-tree-select/${roleId}\`)) as { data: { checkedKeys: string[] } };` `[VERIFIED: ruoyi-react-frontend/src/pages/system/role/index.tsx:137-139]` ⇒ **URL 含角色 id，必须逐条注册精确 endpoint**（`mockApiBatch` 是精确匹配）。
- `/system/departments/role-dept-tree-select/${roleId}` 同理 `[VERIFIED: ruoyi-react-frontend/src/pages/system/role/index.tsx:166-167]`。

**树数据形态（`processTreeData` 只映射这四个字段）**：
```tsx
const MENU_TREE_WITH_F_TYPE = [
  {
    key: "menu-role",
    title: "角色管理",
    value: "menu-role",
    children: [
      { key: "f-add", title: "新增", value: "f-add" },   // ← F 型节点
    ],
  },
];
```
依据：`key: node.key || node.id || ""`、`title: node.title || node.menuName || ""`、`value: node.value || node.id || ""` `[VERIFIED: ruoyi-react-frontend/src/pages/system/role/hooks/useRoleData.ts:104-107]`。

**4 条 87 用例（照 87-RESEARCH §E17 的 T-1..T-4）**：

| # | 用例 | 断言 | 交互路径 |
|---|---|---|---|
| T-1 | 授权闭环正向 | 提交体 `menuIds` **含 F 节点 key** | 点行「编辑」→ 在 Tree 里勾 F 节点 → 点「确定」→ 断言 `POST /system/roles/r1/update` 的 body |
| T-2 | 父联动语义 | 提交的 `menuIds` **同时含父 C 菜单 id** | 同上（antd `checkStrictly` 默认 false） |
| T-3 | F 节点在树中可见 | F 节点 `title`（"新增"）出现在 DOM | 打开编辑弹窗即可 |
| T-4 | 回读一致 | mock `checkedKeys: ["f-add"]` ⇒ 该 F 节点呈勾选态 | 打开编辑弹窗 |

**交互路径的三个坑（planner 必须处理）**：
1. **「编辑」行按钮需要权限**：`screen.getByRole("button", { name: "编辑" })` 只有在权限集含 `PERMISSION_CODES.systemRole.edit` 时才渲染（否则 `ActionButtons` 返回 null，连「操作」下拉都不渲染）`[VERIFIED: permission.gate.test.tsx:97-101 / 163-169]` ⇒ **测试里必须 `setPermissionState([..., PERMISSION_CODES.systemRole.edit, ...])`**。
2. **antd Tree 的勾选交互**：`Tree` 的 `checkable` + `onCheck` 回调在 `index.tsx:442-446` ⇒ 勾选后 `editForm.setFieldsValue({ menuIds: keyArray })`；测试里用 `fireEvent.click` 打在 `.ant-tree-checkbox` 上（antd 6 的 Tree 节点结构），或用 `screen.getByText("新增")` 定位标题后向上找 checkbox。**建议在测试里先 `screen.getByText("新增")` 确认节点存在（T-3），再 `container.querySelectorAll(".ant-tree-checkbox")` 按序号勾选**。
3. **提交断言**：编辑态提交走 `await post(\`/system/roles/${editingRole.id}/update\`, {...roleData, id: editingRole.id})` `[VERIFIED: ruoyi-react-frontend/src/pages/system/role/hooks/useRoleActions.ts:194-198]` ⇒ 断言 `mockApiBatch` 里 `/system/roles/r1/update` 这个 endpoint spy 的第 1 个调用参数体。

**超时**：整页 + antd Table 慢 → `describe(..., () => {...}, 30000)` `[VERIFIED: permission.gate.test.tsx:207]`。

### P-6 · 「脚本挂进 `lint:ci`」的既有做法（照 `check-route-registry.mjs`）

**源**：`scripts/check-route-registry.mjs`（105 行）+ `ruoyi-react-frontend/package.json:14`。

**做法拆解（planner 照抄结构，不照抄内容）**：
1. 脚本自带**定位注释**（说明自己在哪、被谁调用）：`// 本脚本位于仓库根 scripts/ 目录（由 frontend lint:ci 以 \`node ../scripts/check-route-registry.mjs\` 调用），// 仓库根 = 本脚本目录的上一级。` `[VERIFIED: scripts/check-route-registry.mjs:24-25]`。
2. 脚本用 `fileURLToPath(import.meta.url)` 自定位，不依赖 cwd：`const scriptDir = path.dirname(fileURLToPath(import.meta.url));` `[VERIFIED: scripts/check-route-registry.mjs:26]`。
3. 失败形态固定：`console.error("[check-route-registry] FAIL")` + 逐条 `errors.forEach` + `process.exit(1)`；成功形态 `console.log("[check-route-registry] OK " + ...)` `[VERIFIED: scripts/check-route-registry.mjs:100-105]`。
4. 挂载点 = `package.json` 的 `lint:ci` 链（`&&` 串联，`node ../scripts/check-route-registry.mjs` 在**链尾**）`[VERIFIED: package.json:14]`。
5. CI 侧 workflow 跑 `npm run lint:ci`（87 改后）⇒ 本地链 = CI 链。

**⇒ 87 的 `check-permission-codes.mjs` 已在 frontend 内**（`ruoyi-react-frontend/scripts/`），其定位代码已存在：`const scriptDir = path.dirname(fileURLToPath(import.meta.url)); const frontendRoot = path.resolve(scriptDir, ".."); const repoRoot = path.resolve(frontendRoot, "..");` `[VERIFIED: check-permission-codes.mjs:35-37]` ⇒ **不需改定位代码**，只需加进 `lint:ci` 链。

### P-7 · 反面参考：`GrantNewMenuToRolesHavingParent`（**为什么不能用**）

**源**：`internal/core/db/migrations/menu_grant_helpers.go:26-50`（本会话全量读过）。

**逐字源码**：
```go
func GrantNewMenuToRolesHavingParent(db *gorm.DB, parentMenuName string, newMenuID string) error {
	// SQLite 跳过:SQLite 不支持 PG 的 ::uuid cast + ON CONFLICT DO NOTHING;
	// helper 是 PG 启动期 migration 调用点,SQLite 测试路径不消费。
	if !isPostgreSQL(db) {
		return nil
	}
	const sql = `
INSERT INTO sys_role_menu (role_id, menu_id)
SELECT rm.role_id, $1::uuid
FROM sys_role_menu rm
JOIN sys_menu m ON rm.menu_id = m.id
WHERE m.menu_name = $2
ON CONFLICT DO NOTHING
`
	return db.Exec(sql, newMenuID, parentMenuName).Error
}
```
`[VERIFIED: menu_grant_helpers.go:26-50]`

**三条禁用理由（逐条带证据）**：
1. **PG-only，sqlite 静默 noop**：逐字 `// SQLite 跳过:SQLite 不支持 PG 的 ::uuid cast + ON CONFLICT DO NOTHING;` `[VERIFIED: menu_grant_helpers.go:27-28]` ⇒ sqlite 分支的 migration 会**跳过授权而不报错** ⇒ sqlite 测试永远绿 ⇒ 假绿。
2. **按 `menu_name` 匹配 ⇒ 越权授予**：SQL 的 `WHERE m.menu_name = $2` 不带 `status = 0` ⇒ 会命中同名停用旧菜单（每个 ops 域都有 `status=1` 的同名副本）`[CITED: 87-RESEARCH.md:352-358]`。
3. **`ON CONFLICT DO NOTHING` 依赖未实证的唯一约束**：helper 注释声称依赖 `(role_id, menu_id)` 复合 PK / 唯一约束 `[VERIFIED: menu_grant_helpers.go:22-23]`，但模型侧只是普通索引 `[VERIFIED: internal/models/role.go:25-26]` ⇒ 一律改用 `NOT EXISTS`。

**另有反向断言禁止 `fmt.Sprintf`**：`// C6 反向断言:源代码不得含 fmt.Sprintf(SQL 字符串插值已参数化)。` / `if strings.Contains(s, "fmt.Sprintf") { t.Fatalf("helper source must NOT contain fmt.Sprintf (SQL must be parameterized, C6)") }` `[VERIFIED: internal/core/db/migrations/menu_grant_helpers_test.go:128-131]`
⇒ **87 的授权 SQL 必须是参数化占位（`?`），不得 `fmt.Sprintf` 拼串。**

---

## Reusable Fixtures

> 可直接复用的夹具 / 工具清单。planner 写 PLAN 的「验收 / 夹具」段时照抄。

### F-1 · `.planning/notes/2026-10-29-v124-perm-fixture.md`（10 步夹具范式 + 4 个已知坑）

**性质**：生产库只读 + 单事务夹具清理的可重复执行清单（本会话读了 §0/§1/§4 三段）。

**10 步范式（逐条带出处）**：
| 步 | 内容 | 出处 |
|---|---|---|
| 0 | 环境事实：PG `10.62.10.34:5432/ruoyi`（生产）；`.env` 不可读，口令须由环境变量注入 | §1.1（:37-61）`[VERIFIED]` |
| 1 | 夹具工具：**仓库外**临时 Go 程序（`pgx/v5`），目录 `%TEMP%\pgfix\`，含 `main.go` / `10_fixture_role.sql` / `11_fixture_user.sql` / `v_simulate_perm_check.sql` / `90_cleanup.sql` / `run_all.sh`；**验证后必须整体删除** | §1.1（:39-53）`[VERIFIED]` |
| 2 | 超管 bypass 口径核实（代码级）：判定键是 `r.role_key = 'admin'`；夹具 role_key 用 `p87_*` 前缀，**绝不撞名**；验证账号**只挂**本受限角色 | §1.2（:69-83）`[VERIFIED]` |
| 3 | 造数（只 INSERT、幂等 `WHERE NOT EXISTS`、不改任何既有行） | §2.1-2.3（:153-251）`[CITED]` |
| 4 | DB 层前后对照（只读 SQL 复刻中间件三级判定：`step1_exact` / `step2_child_inherit` / `step3_module_match` / `strict_exact_pass`） | §3.4 / §3.8.3 `[CITED]` |
| 5 | 「无数据变更」判据：写接口调用前后 `SELECT count(*)` 目标表一致 | `[CITED]` |
| 6 | 清理 SQL（单事务，先子后父）：`sys_user_post` → `sys_user_role` → `sys_user` → `sys_role_menu` → `sys_role_dept` → `sys_role` | §4（:691-719）`[VERIFIED]` |
| 7 | 清理后自检（必须全 0）：`SELECT (SELECT count(*) FROM sys_role WHERE role_key=…), (… sys_user …), (… sys_role_menu JOIN sys_role …), (… sys_user_role JOIN sys_user …)` | §4（:712-718）`[VERIFIED]` |
| 8 | 全库总量对照：清理前 `sys_role=7 / sys_user=4232 / sys_user_role=4242 / sys_role_menu=353`；差值恰为夹具行数 ⇒ 其余既有行零改动 | §3.8.6 / §4.1（:748-749）`[VERIFIED]` |
| 9 | 并发干扰硬规则：取证前后各跑一次权限集快照，两次数值一致才认；对照实验与取证**必须串行** | §3.9（:653-686）`[CITED]` |
| 10 | 已知坑（4 条） | 见下 |

**4 个已知坑（逐字）**：
1. `sys_menu` 查 **`perms = 'system:user'` 精确匹配**，**禁用 `LIKE 'system:user%'`**（会捞进 2 条隐藏 C 菜单）—— 逐字：`// ⚠️ **坑点**：库里还有两条**隐藏 C 型**菜单 \`system:user:profile\` … 与 \`system:user:settings\` … // **造数 SQL 一律用 \`perms = 'system:user'\` 精确匹配**；若误用 \`perms LIKE 'system:user%'\` 会把这两条一起绑进来。` `[VERIFIED: :126-128]`
2. `sys_role_menu` **只有普通索引非 unique** ⇒ 一律 `NOT EXISTS`（与 P-3 一致）。
3. **模型未映射 ≠ 库表无列**（生产库 `sys_role_menu` 实有 `id` / `created_at`）。
4. **`sys_user.id` 是 `uuid`（非 `user_id`）、`nickname`（非 `nick_name`）**。

**清理 SQL 逐字（可直接复制，只把夹具标识换成 `p87_*`）**：
```sql
-- 1. 用户关联
DELETE FROM sys_user_post
WHERE user_id IN (SELECT id FROM sys_user WHERE username = 'v124_auditor_test');

DELETE FROM sys_user_role
WHERE user_id IN (SELECT id FROM sys_user WHERE username = 'v124_auditor_test');

-- 2. 测试用户本体
DELETE FROM sys_user WHERE username = 'v124_auditor_test';

-- 3. 角色关联
DELETE FROM sys_role_menu
WHERE role_id IN (SELECT id FROM sys_role WHERE role_key = 'v124_audit_test');

DELETE FROM sys_role_dept
WHERE role_id IN (SELECT id FROM sys_role WHERE role_key = 'v124_audit_test');

-- 4. 受限角色本体
DELETE FROM sys_role WHERE role_key = 'v124_audit_test';

-- 5. 清理后自检（应全为 0）
SELECT (SELECT count(*) FROM sys_role WHERE role_key = 'v124_audit_test') AS role_left,
       (SELECT count(*) FROM sys_user WHERE username = 'v124_auditor_test') AS user_left,
       (SELECT count(*) FROM sys_role_menu rm JOIN sys_role r ON r.id = rm.role_id
         WHERE r.role_key = 'v124_audit_test') AS role_menu_left,
       (SELECT count(*) FROM sys_user_role ur JOIN sys_user u ON u.id = ur.user_id
         WHERE u.username = 'v124_auditor_test') AS user_role_left;
```
`[VERIFIED: :691-719]`

**⇒ 87 的用法**：夹具 `role_key` 改 `p87_probe_test` / `username` 改 `p87_probe_user`；**播种的 F 行不在清理范围内**（它们是 phase 交付物，不是夹具）⇒ 清理只删 `sys_role` / `sys_user` / `sys_user_role` / `sys_role_menu`（夹具那几行），**不得**删 `sys_menu` 的 72 条新行。

### F-2 · `.planning/notes/2026-10-29-pg-data-scope-verification.md`（Phase 84 遗留的 PG 实机验证清单范式）

**性质**：「本机无 PG / 无 `psql` / `.env` 不可读 ⇒ 无法代跑」的**人工验证清单**范式 —— 结构可复用，内容换成 87 的判定项。

**结构骨架（逐字）**：
- §0 为什么必须人工做（不可代跑的原因表）`[VERIFIED: :10-18]`
- §1 关键口径（读代码得出，勿凭直觉）`[VERIFIED: :21-48]`
- §2 执行步骤：**步骤 A 基线快照（只读，先做）** → **步骤 B 夹具准备（二选一：B-0 复用路径（推荐，零写入）/ B-1 建夹具路径（需写库，验收后必须清理））** → **步骤 C 应用层实测** `[VERIFIED: :51-120]`
- 收口方式：产出回填到 phase 的 `xx-VERIFICATION.md` 的「遗留（人工验证项，不阻塞 phase 完成）」表 + `status` 由 `human_needed` 改为 `passed` `[VERIFIED: :4]`

**⇒ 87 的复用点**：把 87 的三项人工验证（受影响生产角色清单 / 生产库既有行零改动核对 / HTTP 实测 403→200）按这个骨架写成 `87-MANUAL-VERIFICATION.md` 形态（planner 决定文件名），逐字照抄「只读基线快照 → 夹具 → 实测 → 清理自检」四段式。

**两条必须遵守的约束（逐字）**：`// ⚠️ B1/B2 违反的两个约束必须遵守：// ① 目标用户**不得已有其他角色**（否则 MAX 取值被污染，结论失真）// ② **不得修改任何既有行的密码 / dept_id / status** —— 夹具只加关联，不改既有数据` `[VERIFIED: :116-118]`。

### F-3 · `internal/core/db/migrations/menu_grant_helpers.go`（**仅作反面参考**）

见 P-7。要点复述：**PG-only + 按名字匹配 + 依赖未实证唯一约束** ⇒ 87 改用 migration_218 / 221 的 `NOT EXISTS` + parent_id 精确授权。

### F-4 · 既有测试基建（无需新建配置）

| 基建 | 位置 | 用法 |
|---|---|---|
| sqlite 内存库 | `github.com/glebarez/sqlite`（已在 go.mod，零新依赖） | `gorm.Open(sqlite.Open("file:"+t.Name()+"?mode=memory&cache=shared&_busy_timeout=5000"), &gorm.Config{})` `[VERIFIED: migration_218_test.go:59-60]` |
| 前端渲染 harness | `src/test/utils/renderWithProviders.tsx` | `renderWithProviders(<RoleManagement />, { queryClient: qc })` `[VERIFIED: :55-78]` |
| API mock harness | `src/test/utils/createApiMock.ts` | `mockApiBatch` / `setGenericFallback` / `createApiTestingModule` `[VERIFIED: :118-128 / :147-152 / :164-207]` |
| 权限三态构造 | `useMenuStore.setState({ permissions, permissionsLoaded, permissionsError })` | `[VERIFIED: permission.gate.test.tsx:43-52]` |

---

## Open Questions

> 阻塞 planner 的疑点**按严重度排序**。OQ-1..OQ-7 来自 `87-RESEARCH.md:947-956`（原文不改）；OQ-8 之后是本 mapper 新增。

| # | 疑点 | 影响 | 建议处置 | 状态 |
|---|---|---|---|---|
| **OQ-13**（新增，**最高优先**） | **`ops:floor-plan:edit` 的连字符陷阱**：`PERM_LITERAL`（`check-permission-codes.mjs:44`）与 Go 侧扫描正则（`:376`）的段内字符类 `[A-Za-z0-9_]` **不含 `-`** ⇒ 该码既进不了 `untyped`（c 类）也进不了 `fType`（d 类）。本会话实测：72 码中原正则漏网 **1 个**（正是它），放宽后 72/72 全中，且整个 migrations 目录**新接受字面量 = 0 个** | **清空白名单后 c 类必然 exit 1**（无论 migration 里是否写该字面量） | **必须**把两处（+ 新增 F 块 `innerRe`）的段内字符类改为 `[A-Za-z0-9_-]`（Files to Modify §3(d)）。本会话已用脚本副本模拟全组合 ⇒ `EXIT=0`、漂移 0 项 | **已定，必做**（研究文档未覆盖此点） |
| **OQ-8**（新增，**高优先**） | **`visible` 口径与 GORM 落库冲突**：AGENTS.md 要求播种 F 行 `visible=0`，但 `models.Menu.Visible` 的 tag 是 `gorm:"default:1"`，GORM 对「零值 + default tag」字段在 INSERT 时**省略该列** ⇒ `db.Create`（migration_218/202/221 同款）落 `visible=1`。本会话实测三条路径：`db.Create` → 1；`db.Select("*").Create` → 1；原生 INSERT → 0 | 87 的 F 行可见性口径与 AGENTS.md P1 冲突 | **按 P-1 走 `db.Create` + `Update("visible", 0)` 归零**，并写 `TestMigrate226_VisibleIsZero` 锁死。该项对权限判定无功能影响（F 行不进导航树），但 AGENTS.md 是硬约束 ⇒ 必须归零 | **建议按此执行**（planner 若要改口径须回写 AGENTS.md 或在 87-VERIFICATION 显式登记） |
| **OQ-9**（新增） | **F 型声明块的内容边界**：块内每个码都会进 `menuFType`（d 类）。若误放非 72 码（如 `ops:floor-plan:list` —— 该码 TS 未声明 `[CITED: 87-RESEARCH.md:955]`）⇒ d 类立即 exit 1 | 校验脚本红 | **F_TYPE_SEED 块内只放 72 条动作码**；父 perms 可放块外（scope 过滤保证 `system:role` 不会被 d 类命中，因 d 类作用域是 `system:role:` 前缀） | 已定（P-1 + P-4） |
| **OQ-10**（新增） | **两份清单的一致性**（F 型声明块 72 条 vs 种子表 72 条）会漂移 | 脚本认为已播种、库里没有 | migrate 入口加硬断言（条数 + 逐条相等），漂移即返回 error（非阻断降级） | 已定（P-1） |
| **OQ-11**（新增） | **`_test.go` 里的字面量也会被 c 类扫到**（`collectMigrationFiles` 收所有 `.go`）`[VERIFIED: check-permission-codes.mjs:351]` | c 类可能因测试文件而假绿 | **72 码字面量只写在 `migration_226_action_permission_f_type_seed.go`（非测试文件）里**；测试文件可另用 `perm226Expect` 变量间接引用，避免重复字面量 | 建议照办 |
| **OQ-12**（新增） | **`ops:manager:{rule,org}:*` 6 码的处置**（OQ-2 的延伸） | 若把这 6 码放进 F 型声明块 ⇒ `ops:manager:rule:list` 是 **C 型页面本体码**（不是 F 型按钮），语义错误 | **87 不播种、不入 F 型块**；白名单条目按 OQ-2 移除并在 `87-STATE.md` 注明「migration_221 已覆盖」 | 待 Lead 确认（与 OQ-2 同步） |
| **OQ-1** | `sys_role_menu` 在**生产 PG** 上是否真有 `(role_id, menu_id)` 唯一约束？模型侧只是普通索引 `[VERIFIED: internal/models/role.go:25-26]`，helper 注释却声称依赖唯一约束 `[VERIFIED: menu_grant_helpers.go:22-23]` | 决定 `ON CONFLICT DO NOTHING` 是否可用 | **87 一律用 `NOT EXISTS`**，不依赖唯一约束；`ON CONFLICT` 一律不用 | 待用户在 §Manual-Only 表复核 `\d sys_role_menu` |
| **OQ-2** | 是否要一并播种 6 条 `ops:manager:*`？ | 白名单 6 项处置口径 | **87 不播种**（避免同 perms 重复行），白名单条目移除并注明 | 待 Lead 确认 |
| **OQ-3** | `ops:building:import/export` 的父是「楼宇管理」还是也要给「楼宇空间」一份？ | 树展示 / Step 2 继承的父 | 不重复挂 —— 父 = 楼宇管理 | 待 Lead 确认 |
| **OQ-4** | `system:role:export` 无后端路由 | 白名单完整性 | 仍然播种（白名单是权威清单），在 `87-VERIFICATION.md` 标注「无路由，仅作授权来源占位」 | 待 Lead 确认 |
| **OQ-5** | 前端菜单缓存 5min（内存）+ 30min（sessionStorage）叠加后端 30s 权限缓存 | UI 实机验收可复现性 | 验收清单显式写「清 sessionStorage `menu:cache` 或重新登录后再断言按钮可见性」 | 已定（写入验收清单） |
| **OQ-6** | `ops:walls:*` / `ops:doors:*` / `ops:floor-plan:edit` 挂「楼层管理」是推断；CAD 平面图编辑器的实际使用页需人工确认 | 父菜单选择 ⇒ 影响 Step 2 继承与树展示 | **实地确认前不改这三个码的父** | 待用户/Lead 确认（唯一实质性待确认项） |
| **OQ-7** | `ops:floor-plan:{list,query,save}` / `ops:walls:list` / `ops:doors:list` 不在 TS 也不在白名单，但读路由引用了它们 ⇒ 读路径对非 admin 恒 403 | 读路径可用性 | **87 不处理**（不在白名单、不在 TS）；登记为独立后续项 | 待 Lead 确认（建议独立立项） |

---

*Pattern map produced by gsd-pattern-mapper, 2026-10-30。全部 in-repo 离散值均已本会话用 Read 工具逐字读取并标注 `[VERIFIED: 路径:行号]`；本会话另新增两项实证（visible 落库行为、`IN ?` 切片展开），探针测试文件已删除，仓库零残留。*
