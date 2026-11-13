# 豁免治理策略（EXEMPTIONS-POLICY）

> 适用范围：本仓 `scripts/` 下所有「豁免/白名单/跳过」类清单。
>
> - `scripts/frontend-url-exemptions.json` —— 前端 URL 与后端路由不匹配豁免（由 `check-frontend-api-urls.mjs` 消费）
> - `xingran-react-frontend/scripts/permission-drift-exemptions.json` —— 权限码存量漂移豁免（由 `check-permission-codes.mjs` 消费）
> - `scripts/check-frontend-api-urls.mjs` 的 `PENDING_DERIVED` —— 「确有需求但后端暂未实现」的工厂派生端点待实现台账
>
> 本文件只定义规则，不改任何门禁脚本的行为；规则由各门禁的 schema 校验与 CI 执行落地。

## 一、三要素（缺一不可）

每一条豁免 / 待实现登记，必须同时具备：

| 要素 | 含义 | 落地字段 |
| --- | --- | --- |
| **reason** | 为什么必须豁免：根因、影响面、为何不能当次修复 | `reason`（非空字符串） |
| **清偿窗口** | 明确的 phase / 日期 / 版本，不是「后续」「将来」「尽快」 | `window` / `deadline` / `phase`（非空字符串） |
| **到期复核** | 窗口到期后由谁、多久内、复跑什么命令、未清偿如何升级 | `review` / 清单顶层 `clearancePolicy.review` |

**无理由保留一律禁止。** 只有「 grandfathered 存量」可进豁免；新代码、新端点、新漂移不允许登记豁免，必须当次补齐声明或改实现。

## 二、优先顺序（能修就别豁免）

1. **删死代码 / 删死派生**：确认无后端实现且前端未调用 → 从派生列表或导出对象中移除该函数本身，而不是加豁免。
   - 例：`createResourceApi` 会机械生成 8 个方法，若前端只显式 pick 了 3 个，其余 5 个是「工厂生成但从未调用」的幽灵端点。正确做法是让门禁按实际消费方法精确派生（见 `check-frontend-api-urls.mjs` 第 5 点），而不是为这 5 条 WARN 加豁免。
2. **补声明 / 补实现**：能在本任务文件所有权内补齐 TS 声明、后端常量、菜单播种的，直接补齐并删除对应豁免。
3. **登记台账**：确需改他人所有权文件（如 `pkg/permission/config.go`、`internal/core/db/migrations/`、`src/constants/permissionCodes.ts`）才能清偿的，不改越界文件，改为在台账登记「需后续 phase 补 X」+ 明确窗口。
4. **加豁免**：仅当以上三步都不可行、且确属历史存量漂移时，才写入豁免清单，并满足第一节三要素。

## 三、两类清单的具体规则

### 3.1 前端 URL 豁免（`frontend-url-exemptions.json`）

- 结构：顶层数组，条目 `{ url, file?, reason }`；`reason` 必填，门禁逐条强校验，缺失即 `exit 1`。
- **当前状态：空数组 `[]`，且不得新增条目。** 存量断链已全部通过「精确派生 + 删死函数」清零。
- 新增不匹配的正确处置：改 URL / 下线调用 / 补后端路由 / 删死派生方法，四选一；不许回退到加豁免。
- 工厂派生端点若「确有需求但后端暂未实现」，登记在 `check-frontend-api-urls.mjs` 的 `PENDING_DERIVED` 台账（`reason` + `window` 双必填门禁校验），**不写进本 JSON、不计入豁免命中、不产生 WARN**；到期复核清除。
- 严格模式：任何工厂派生 WARN 或不匹配都使 `exit=1`，不再静默放行。

### 3.2 权限码存量漂移豁免（`permission-drift-exemptions.json`）

- 结构：顶层对象，`exemptions` 数组条目 `{ code, reason, grandfathered: true }`；门禁强校验三字段，且 `code` 必须形态合法、不重复。
- **当前状态：104 条**（整改前 126 条）。删除了 20 条 `code` 被污染（带中文尾巴、脚本 `byCode` 从未匹配）的无效条目与 2 条声明已补齐的过期条目。
- 剩余 104 条按「缺失方向」分组为 A/B/C/D 四类，每条 `reason` 内写明：
  - 分组与缺失声明方向；
  - 需后续 phase 补 X（具体文件：`config.go` / `permissionCodes.ts` / `migrations/`）；
  - 清偿窗口（phase + 日期）；
  - 到期复核动作（复跑 `node scripts/check-permission-codes.mjs`，未清偿升级为阻塞项）。
- 清单顶层 `groups` / `windows` / `clearancePolicy` / `dropped` 为分组台账与审计轨迹，`check-permission-codes.mjs` 只读 `exemptions`，额外字段不影响门禁。
- **所有权边界**：补齐这些声明需要改 `pkg/permission/config.go`、`src/constants/permissionCodes.ts`、`internal/core/db/migrations/`，均不在本任务文件所有权内；按第二节第 3 点登记台账、不越界修改。
- **清偿目标**：104 → 0。窗口到期后必须逐条复核并删除已清偿条目；`✓ 双向零漂移` 之外的「按存量豁免跳过」提示由门禁脚本硬编码打印，随豁免归零自然消失。

## 四、到期复核流程

1. 每个清偿窗口到期后 **5 个工作日内**，清单 owner 复跑对应门禁：
   - `node scripts/check-frontend-api-urls.mjs`
   - `cd xingran-react-frontend && node scripts/check-permission-codes.mjs`
2. 逐条核对：已补齐声明 / 已删死代码 → 从清单删除该条；仍未清偿 → 收窄窗口并升级为阻塞项，在台账写明阻塞原因。
3. 复核结果写回清单顶层 `clearancePolicy` 与 `dropped` 审计字段，保留历史轨迹。
4. 门禁输出出现「提示：豁免条目已无对应不匹配项，可清理」时，必须当次清理，不许留腐烂条目。

## 五、反模式（禁止）

- ❌ 用豁免掩盖「工厂生成但从未调用」的幽灵端点（应删死派生）。
- ❌ `reason` 写「历史遗留」「后续处理」「待整改」而无窗口与复核动作。
- ❌ 为让门禁变绿而新增豁免条目（应改实现或登记待实现台账）。
- ❌ 越界修改他人所有权文件来「顺手」清偿（应登记台账并给出窗口）。
- ❌ `code` / `url` 字段混入中文说明、括号尾巴等噪声（会导致门禁永不匹配，形成假豁免）。
- ❌ 保留已过期 / 已失效条目不清理。

## 六、变更记录

| 日期 | 变更 | 结果 |
| --- | --- | --- |
| 2026-11-12 | 前端 URL 门禁改「精确派生」+ 严格模式（WARN 即 `exit=1`）；16 条工厂派生 WARN 清零；`frontend-url-exemptions.json` 保持空数组 | WARN 0 / 豁免命中 0 / `exit=0` |
| 2026-11-12 | 删除 `rpaApi.ts` 死代码（生产零导入）并同步 invariants 基线 | 全仓仅测试引用，删后测试 10→6 仍绿 |
| 2026-11-12 | 权限漂移豁免 126→104：删 20 污染 + 2 过期，104 条分组台账化 + 清偿窗口 | 门禁 `exit=0`、双向零漂移（含台账） |
