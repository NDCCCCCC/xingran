# 前端覆盖率基线台账（frontend coverage ratchet ledger）

> 本文件是 `.coverage-fe-floors` 的配套台账。`.coverage-fe-floors` 文件头注释
> 要求「每次上调登记到 `.planning/frontend-coverage-baseline.md`」，本文件即该登记处。
> 单一真相源：门禁脚本 `.github/scripts/check-frontend-coverage.sh`，口径 =
> vitest v8 全量 `include: ["src/**/*.{ts,tsx}"]`（`coverage.exclude` 见
> `xingran-react-frontend/vitest.config.ts`），按 statements 加权，目录粒度
> `src/pages/<二级目录>` / `src/components/<二级目录>` / 其它一级目录。

## 1. 当前各目录实测覆盖率基线

数据采集命令（仓库根，MSYS bash）：

```bash
cd xingran-react-frontend && npm run test:coverage
# 汇总 All files = 加权 statements；逐目录数值以门禁脚本输出为准：
bash .github/scripts/check-frontend-coverage.sh \
  xingran-react-frontend/coverage/coverage-final.json .coverage-fe-floors
```

**基线轮（2026-11-12，本台账建立时）**：`Test Files 593 passed (593)`、
`Tests 4161 passed (4161)`、`All files 63.01%`（TOTAL 14710/23345 stmts）。

| 目录 | floors 值（基线轮 → 本轮生效值） | 门禁实测%（covered/stmts） | 备注 |
| --- | --- | --- | --- |
| GLOBAL（加权） | 3.8 → 3.8（保持） | 63.01（14710/23345） | D-14 把 GLOBAL 行钳制到 `min(measured, 3.8)`，故 3.8 是**阈值上限**而非实测；见 §4 |
| pages/ad-domain | 33.4 → **36.2**（本轮上调，见 §5） | 36.73（479/1304） | 目录实名 `ad-domain`；任务书里的「pages/ad 0.0」不是门禁 key（见 §3） |
| pages/vdi | 31.1 → 31.1（保持） | 31.62（160/506） | 实测 -0.5pp = 31.12 → 31.1，与现值相同 |
| pages/system | 45.4 → 45.4（保持） | 49.72（1401/2818） | |
| pages/monitor | 63.0 | 64.54（395/612） | |
| pages/workorder | 59.8 | 61.95（368/594） | |
| pages/login | 64.8 | 65.31（64/98） | |
| 其余 40 个目录 | 见 `.coverage-fe-floors` | 见门禁 PASS 行 | 基线轮 45/45 目录全部 `>= floor` |

完整逐目录 PASS 行见本轮门禁输出（45/45 directories >= floor）。所有数值均来自
`check-frontend-coverage.sh` 的 `% -50s %8d %8d %6.2f%%` 表与 per-dir PASS 行，
不是手抄 vitest text reporter 的 `All files`。

## 2. Ratchet 规则（只升不降）

1. **只升不降**：`.coverage-fe-floors` 的任何一行数值只能上调，禁止下调或删除行
   （门禁方向 b 会校验「profile 里每个目录都已登记」，删行会让新目录绕过门禁）。
2. **初始值口径**：`floor = round1(max(0, measured% - 0.5pp))`（D-06 噪声余量），
   **绝不贴实测值**。0.5pp 用于吸收工具链 patch 漂移 / 环境分支差异 / 异步计时抖动。
3. **上调幅度**：一次上调后的值必须 `<= 实测% - 0.5pp`（保留噪声余量）；
   若 `实测 - 0.5 <= 现值` 则该行本轮**无上调空间**，保持现值并登记原因
   （本轮 `pages/vdi` 即此情形）。
4. **同 commit 证据**：每次上调必须与「回归证据」在同一个 commit 里落地：
   - 全量 `npm run test:coverage` 通过（Test Files / Tests 全绿）；
   - `npm run lint:ci` 通过；
   - 门禁脚本 `check-frontend-coverage.sh ... .coverage-fe-floors` 退出码 0；
   - 本台账追加一行（日期 / 目录 / 旧值 / 新值 / 实测 / 证据命令与结果 / 变更原因）。
   证据不足时**不上调**——宁可让 floor 保守，也不为让数字好看而破门禁。
5. **改数据不改脚本**：ratchet 只编辑 `.coverage-fe-floors` 的数值行，
   不修改 `check-frontend-coverage.sh`（D-07）。

## 3. 与任务书措辞的事实对齐（重要，勿误改）

- 任务书写的「pages/ad = 0.0」**不存在于 `.coverage-fe-floors`**。门禁按
  `src/pages/<二级目录>` 聚合，仓库里 AD 相关页面实名目录是 `src/pages/ad-domain/`，
  对应 floors 行是 `pages/ad-domain 33.4`。凭空新增一行 `pages/ad 0.0` 会触发
  门禁方向 a 的 `FAIL: pages/ad not found in profile`，直接破门禁。
- 任务书写的「GLOBAL 3.8」是 D-14 钳制上限（`min(measured, 3.8)`），
  **不是实测覆盖率**；本轮实测加权 63.01%。GLOBAL 行的语义是「全局阈值」，
  强抬到实测附近会让门禁在噪声下变红，故本轮不动。
- 因此本轮「先 pages/ad、pages/vdi」按真实门禁 key 落地为
  「先 `pages/ad-domain`、再 `pages/vdi`、然后 `pages/system`」。

## 4. 分阶段提升计划

| 阶段 | 目标目录 | 前置条件 | 目标 floor | 状态 |
| --- | --- | --- | --- | --- |
| 阶段 1（本轮） | `pages/ad-domain` | 已有门禁实测 36.73%；全量覆盖率 + lint:ci + 门禁脚本全绿 | 33.4 → 36.2（= 36.73 - 0.5，一位） | ✅ 本轮上调，见 §5 |
| 阶段 1（本轮） | `pages/vdi` | 实测 31.62% - 0.5 = 31.12 → 31.1 = 现值 | 保持 31.1 | ✅ 登记「无上调空间」，不抬 |
| 阶段 2 | `pages/system` | 新增 `system/backup` RestoreModal 冒烟；待下轮全量实测稳定后按 `实测-0.5pp` 上调（现值 45.4，实测 49.72 → 目标 49.2） | 45.4 → 49.2（候选） | ⏳ 本轮不动，留待有独立证据的 commit |
| 阶段 3 | `pages/ad-domain` / `pages/vdi` 再抬 | 需为这两个目录补**真实渲染/行为测试**把实测继续推高，再按 §2.3 上调 | 依实测 | ⏳ 待补测试 |
| GLOBAL | `GLOBAL 3.8` | **本轮不强抬**（任务书明确，避免门禁红）。前置条件：CI 分片合并产物稳定、多轮本地/CI 实测一致、且 D-14 钳制语义被重新裁决；否则 3.8 只是安全上限 | 待定 | ⏳ 只写目标与前置条件，不动数值 |

GLOBAL 目标（仅记录，不执行）：若未来要抬，必须先证明 4 个 shard 合并后的
`coverage-final.json` 与本地全量一致、且连续多轮加权实测波动 < 0.1pp，
再按「新阈值 = 保守实测 - 噪声余量」重新裁定，并同步本台账与 `.coverage-fe-floors`。

## 5. 上调登记（append-only，逐行追加）

| 日期 | 目录 | 旧值 | 新值 | 当轮实测 | 证据 | 原因 |
| --- | --- | --- | --- | --- | --- | --- |
| 2026-11-12 | `pages/ad-domain` | 33.4 | 36.2 | 36.73%（479/1304） | 基线轮 `npm run test:coverage` → 593 files / 4161 tests 全绿；`check-frontend-coverage.sh` → GATE_EXIT=0（45/45 目录 >= floor）；`npm run lint:ci` → 通过 | 基线轮实测高出 floor 3.33pp，按 §2.3 上调到 `实测-0.5pp` 的一位（36.2），仍留 0.53pp 噪声余量 |
| 2026-11-12 | `pages/vdi` | 31.1 | 31.1（保持） | 31.62%（160/506） | 同上 | `31.62 - 0.5 = 31.12 → 31.1` = 现值，按 §2.3 本轮无上调空间，登记为「已复核、不抬」而非静默跳过 |
| 2026-11-12 | `GLOBAL` | 3.8 | 3.8（保持） | 加权 63.01%（14710/23345） | 同上 | 本轮按要求不强抬；目标/前置条件见 §4 |

> 注：本轮新增 4 组冒烟测试落在 `pages/monitor`（日志清空 login 分支）、
> `pages/workorder`（周期工单 4 端点）、`pages/system`（RestoreModal 二次确认）、
> `pages/login`（登录加密契约锁，e2e 孤儿价值迁移）。它们**不改变**
> `pages/ad-domain` / `pages/vdi` 的实测数值，故这两行以基线轮实测为证据。
> 新增测试同时让 `pages/monitor` / `pages/workorder` / `pages/system` / `pages/login`
> 的现有 floor 更安全，但按 §2.3 本轮不为它们上调（避免一次改动抬太多行）。
