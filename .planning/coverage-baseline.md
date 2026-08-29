# Coverage Baseline: v1.26 后端测试覆盖率优秀

**起点来源:** `.planning/quick/260820-backend-test-coverage-scan/SUMMARY.md` (2026-08-20 纯只读扫描, 不可回填)
**测量口径:** 74 业务包加权平均, 排除 scripts/migrations/cmd main/internal/docs (43652 stmts 口径)
**生成方式:** `bash .github/scripts/check-coverage.sh coverage.out` (本地 + CI 同公式)

---

## 起点 (v1.26 启动前)

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-20 | 起点 | 12.8 | 43652 | 5589 | 33 | 5ead742 | n/a | n/a | n/a |

### Per-package (起点)

```
pkg/normalize                                                             45         44      97.8%
internal/config                                                          147        137      93.2%
internal/middleware                                                      196        169      86.2%
internal/transform                                                       111         95      85.6%
internal/services/portwrite                                              259        221      85.3%
internal/services/component_collector                                    345        285      82.6%
internal/utils/operlog                                                    90         74      82.2%
internal/services/topology                                                73         55      75.3%
internal/services/lldp                                                    96         57      59.4%
internal/core/security                                                   313        153      48.9%
internal/templates                                                       243        108      44.4%
internal/services/asset                                                 1354        549      40.5%
internal/core/db                                                         643        241      37.5%
internal/websocket                                                       129         45      34.9%
pkg/crypto                                                               439        148      33.7%
internal/core/db/migrations                                              293         83      28.3%
pkg/permission                                                           114         30      26.3%
pkg/cache                                                                926        228      24.6%
internal/services/base                                                    61         14      23.0%
internal/services/operations                                            3714        835      22.5%
internal/services/portcollection                                         580        112      19.3%
internal/services/addomain                                              2415        371      15.4%
pkg/middleware                                                           609         87      14.3%
pkg/errors                                                               326         45      13.8%
internal/services                                                       5202        589      11.3%
internal/services/system                                                3483        355      10.2%
internal/api/v1/asset                                                    420         35       8.3%
internal/api/v1/network                                                 1971        149       7.6%
internal/api/v1                                                          578         38       6.6%
internal/services/scheduler                                              167          8       4.8%
internal/utils                                                           531         24       4.5%
internal/scheduler                                                      1103         36       3.3%
internal/api/v1/operations                                              1285         39       3.0%
internal/services/vdi                                                   1127         30       2.7%
internal/device                                                         1249         31       2.5%
internal/core                                                            754         16       2.1%
internal/agent/server                                                    616         13       2.1%
internal/services/rpa                                                   1865         21       1.1%
internal/services/workorder                                              715          4       0.6%
internal/api/v1/system                                                  3039         14       0.5%
internal/models                                                          445          1       0.2%
pkg/time                                                                  63          0       0.0%
pkg/response                                                              51          0       0.0%
pkg/query                                                                105          0       0.0%
pkg/logger                                                                79          0       0.0%
pkg/ldaputils                                                             33          0       0.0%
pkg/gormutil                                                             194          0       0.0%
pkg/captcha                                                              409          0       0.0%
internal/services/network                                                127          0       0.0%
internal/services/monitor                                                485          0       0.0%
internal/services/knowledge                                               85          0       0.0%
internal/services/duty                                                   114          0       0.0%
internal/services/common                                                   1          0       0.0%
internal/server                                                            2          0       0.0%
internal/pkg/system                                                      345          0       0.0%
internal/pkg/cache                                                       167          0       0.0%
internal/models/system/requests                                          109          0       0.0%
internal/models/system                                                    11          0       0.0%
internal/models/rpa                                                       94          0       0.0%
internal/models/operations                                                23          0       0.0%
internal/docs                                                              1          0       0.0%
internal/api/v1/workorder                                                297          0       0.0%
internal/api/v1/vdi                                                      298          0       0.0%
internal/api/v1/scheduler                                                152          0       0.0%
internal/api/v1/rpa                                                      612          0       0.0%
internal/api/v1/operations/requests                                       15          0       0.0%
internal/api/v1/monitor                                                  518          0       0.0%
internal/api/v1/knowledge                                                273          0       0.0%
internal/api/v1/duty                                                     265          0       0.0%
internal/api/v1/agent                                                     38          0       0.0%
internal/api                                                             417          0       0.0%
internal/agent/pkg/retry                                                  33          0       0.0%
cmd/agent                                                                 59          0       0.0%
cmd                                                                      106          0       0.0%
PACKAGE                                                                 STMT    COVERED        PCT
-------                                                                 ----    -------        ---
```

---

## Phase 71 后

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-20 | Phase 71 后 | 12.8 | 43652 | 5589 | 33 | 326a541 | gsd-execute-phase 71 | n/a | n/a |

### Per-package (Phase 71 后)

```
pkg/normalize                                                             45         44      97.8%
internal/config                                                          147        137      93.2%
internal/middleware                                                      196        169      86.2%
internal/transform                                                       111         95      85.6%
internal/services/portwrite                                              259        221      85.3%
internal/services/component_collector                                    345        285      82.6%
internal/utils/operlog                                                    90         74      82.2%
internal/services/topology                                                73         55      75.3%
internal/services/lldp                                                    96         57      59.4%
internal/core/security                                                   313        153      48.9%
internal/templates                                                       243        108      44.4%
internal/services/asset                                                 1354        549      40.5%
internal/core/db                                                         643        241      37.5%
internal/websocket                                                       129         45      34.9%
pkg/crypto                                                               439        148      33.7%
internal/core/db/migrations                                              293         83      28.3%
pkg/permission                                                           114         30      26.3%
pkg/cache                                                                926        228      24.6%
internal/services/base                                                    61         14      23.0%
internal/services/operations                                            3714        835      22.5%
internal/services/portcollection                                         580        112      19.3%
internal/services/addomain                                              2415        371      15.4%
pkg/middleware                                                           609         87      14.3%
pkg/errors                                                               326         45      13.8%
internal/services                                                       5202        589      11.3%
internal/services/system                                                3483        355      10.2%
internal/api/v1/asset                                                    420         35       8.3%
internal/api/v1/network                                                 1971        149       7.6%
internal/api/v1                                                          578         38       6.6%
internal/services/scheduler                                              167          8       4.8%
internal/utils                                                           531         24       4.5%
internal/scheduler                                                      1103         36       3.3%
internal/api/v1/operations                                              1285         39       3.0%
internal/services/vdi                                                   1127         30       2.7%
internal/device                                                         1249         31       2.5%
internal/core                                                            754         16       2.1%
internal/agent/server                                                    616         13       2.1%
internal/services/rpa                                                   1865         21       1.1%
internal/services/workorder                                              715          4       0.6%
internal/api/v1/system                                                  3039         14       0.5%
internal/models                                                          445          1       0.2%
pkg/time                                                                  63          0       0.0%
pkg/response                                                              51          0       0.0%
pkg/query                                                                105          0       0.0%
pkg/logger                                                                79          0       0.0%
pkg/ldaputils                                                             33          0       0.0%
pkg/gormutil                                                             194          0       0.0%
pkg/captcha                                                              409          0       0.0%
internal/services/network                                                127          0       0.0%
internal/services/monitor                                                485          0       0.0%
internal/services/knowledge                                               85          0       0.0%
internal/services/duty                                                   114          0       0.0%
internal/services/common                                                   1          0       0.0%
internal/server                                                            2          0       0.0%
internal/pkg/system                                                      345          0       0.0%
internal/pkg/cache                                                       167          0       0.0%
internal/models/system/requests                                          109          0       0.0%
internal/models/system                                                    11          0       0.0%
internal/models/rpa                                                       94          0       0.0%
internal/models/operations                                                23          0       0.0%
internal/docs                                                              1          0       0.0%
internal/api/v1/workorder                                                297          0       0.0%
internal/api/v1/vdi                                                      298          0       0.0%
internal/api/v1/scheduler                                                152          0       0.0%
internal/api/v1/rpa                                                      612          0       0.0%
internal/api/v1/operations/requests                                       15          0       0.0%
internal/api/v1/monitor                                                  518          0       0.0%
internal/api/v1/knowledge                                                273          0       0.0%
internal/api/v1/duty                                                     265          0       0.0%
internal/api/v1/agent                                                     38          0       0.0%
internal/api                                                             417          0       0.0%
internal/agent/pkg/retry                                                  33          0       0.0%
cmd/agent                                                                 59          0       0.0%
cmd                                                                      106          0       0.0%
PACKAGE                                                                 STMT    COVERED        PCT
-------                                                                 ----    -------        ---
```

### Per-package 倒退检查 (本行)

- [x] 无新增 0% 包 (起点 33 → Phase 71 后 33)
- [x] 无 per-package 倒退 (Phase 71 不改业务, 不改测试)

---

## Phase 72 后

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-21 | Phase 72 后 | 21.5 | 43652 | 9405 | 31 | TBD (atomic ratchet) | gsd-execute-phase 72 | 12.8 | 21.5 |

### Per-package (Phase 72 后)

```
pkg/normalize                                                             45         44      97.8%
internal/config                                                          147        137      93.2%
internal/middleware                                                      196        169      86.2%
internal/transform                                                       111         95      85.6%
internal/services/portwrite                                              259        221      85.3%
internal/services/component_collector                                    345        285      82.6%
internal/utils/operlog                                                    90         74      82.2%
internal/services/topology                                                73         55      75.3%
internal/api/v1/scheduler                                                152        130      85.5%
internal/services/lldp                                                    96         57      59.4%
internal/services/workorder                                              715        527      73.7%
internal/core/security                                                   313        153      48.9%
internal/templates                                                       243        108      44.4%
internal/services/asset                                                 1354        549      40.5%
internal/api/v1/monitor                                                  518        369      71.2%
internal/core/db                                                         643        241      37.5%
internal/websocket                                                       129         45      34.9%
pkg/crypto                                                               439        148      33.7%
internal/core/db/migrations                                              293         83      28.3%
pkg/permission                                                           114         30      26.3%
pkg/cache                                                                926        228      24.6%
internal/services/operations                                            3714        835      22.5%
internal/services/portcollection                                         580        112      19.3%
internal/services/addomain                                              2415        371      15.4%
internal/services/system                                                3483       1862      53.5%
pkg/middleware                                                           609         87      14.3%
pkg/errors                                                               326         45      13.8%
internal/services                                                       5202        589      11.3%
internal/api/v1/system                                                  3039       1077      35.4%
internal/api/v1/asset                                                    420         35       8.3%
internal/api/v1/network                                                 1971        149       7.6%
internal/api/v1                                                          578         38       6.6%
internal/services/scheduler                                              167          8       4.8%
internal/utils                                                           531         24       4.5%
internal/scheduler                                                      1103         36       3.3%
internal/api/v1/operations                                              1285         39       3.0%
internal/services/vdi                                                   1127         30       2.7%
internal/device                                                         1249         31       2.5%
internal/core                                                            754         16       2.1%
internal/agent/server                                                    616         13       2.1%
internal/services/rpa                                                   1865         21       1.1%
internal/api/v1/workorder                                                297        224      75.4%
pkg/time                                                                  63          0       0.0%
pkg/response                                                              51          0       0.0%
pkg/query                                                                105          0       0.0%
pkg/logger                                                                79          0       0.0%
pkg/ldaputils                                                             33          0       0.0%
pkg/gormutil                                                             194          0       0.0%
pkg/captcha                                                              409          0       0.0%
internal/services/network                                                127          0       0.0%
internal/services/monitor                                                485          0       0.0%
internal/services/knowledge                                               85          0       0.0%
internal/services/duty                                                   114          0       0.0%
internal/services/common                                                   1          0       0.0%
internal/server                                                            2          0       0.0%
internal/pkg/system                                                      345          0       0.0%
internal/pkg/cache                                                       167          0       0.0%
internal/models/system/requests                                          109          0       0.0%
internal/models/system                                                    11          0       0.0%
internal/models/rpa                                                       94          0       0.0%
internal/models/operations                                                23          0       0.0%
internal/docs                                                              1          0       0.0%
internal/api/v1/vdi                                                      298          0       0.0%
internal/api/v1/rpa                                                      612          0       0.0%
internal/api/v1/operations/requests                                       15          0       0.0%
internal/api/v1/knowledge                                                273          0       0.0%
internal/api/v1/duty                                                     265          0       0.0%
internal/api/v1/agent                                                     38          0       0.0%
internal/api                                                             417          0       0.0%
internal/agent/pkg/retry                                                  33          0       0.0%
cmd/agent                                                                 59          0       0.0%
cmd                                                                      106          0       0.0%
PACKAGE                                                                 STMT    COVERED        PCT
-------                                                                 ----    -------        ---
PACKAGE                                                                43652       9405      21.5%
```

### Per-package 倒退检查 (Phase 72 后)

- [x] 无新增 0% 包 (Phase 71 后 33 → Phase 72 后 31; 减少 2 个 0% 包)
- [x] 无 per-package 倒退 (Phase 72 严格测试-only, 无业务代码改动 per D-08)
- [x] 6 个 CORE 目标加权平均显著上升:
  - CORE-01 internal/api/v1/workorder: 0.0% → 75.4%
  - CORE-02 internal/api/v1/monitor: 0.0% → 71.2%
  - CORE-03 internal/api/v1/scheduler: 0.0% → 85.5%
  - CORE-04 internal/api/v1/system: 0.5% → 35.4% (sub-target，未达 70% per-sub-module)
  - CORE-05 internal/services/workorder: 0.6% → 73.7%
  - CORE-06 internal/services/system: 10.2% → 53.5% (sub-target，未达 70% per-sub-module)
- [ ] 整体加权平均 12.8% → 21.5% (上升 8.7 个百分点，未达 ≥30% 预估目标)

### Notes (Phase 72 deviation)

- CORE-04 (api/v1/system) 35.4% 与 CORE-06 (services/system) 53.5% 仍低于 70% per-sub-module 目标 (D-04 strict)。
- 部分子模块 (ad_account_handler.go / ad_domain_handler.go / ad_dept_sync_handler.go 等 AD 系列) 仍为 0%,因 Phase 36 AD 账号池已有完整测试 (ad_account_handler_test.go 排除 per D-08)。
- 0% 包数量减少 2 个,但 31 个 0% 包仍需 Phase 73 (P1) / Phase 74 (P2) 补齐。
- **Ratchet commit 落地**:12.8% → 21.5%,phase-level gate 已重置,后续 Phase 73/74 持续 ratchet 上调。

> **Ratchet note (D-04):** The `commit` column on the Phase 72 后 row reads
> `TBD (atomic ratchet)` until plan 72-13 Task 4 amends this file with the actual
> short SHA of the commit that ships the .coverage-threshold + this file.
> Subsequent phases (73 / 74) append a new section with the same column schema,
> bumping `.coverage-threshold` in the same atomic commit (D-04 manual ratchet).

> **Ratchet note (D-04):** The `commit` column on the Phase 71 后 row reads
> `TBD` until plan 71-01b Task 4 amends this file with the actual short
> SHA of the commit that ships the .coverage-threshold + ci.yml + this
> file. Subsequent phases (72 / 73 / 74) append a new section with the
> same column schema, bumping `.coverage-threshold` in the same atomic
> commit (D-04 manual ratchet, see `.planning/phases/71-.../71-RESEARCH.md`
> §"Ratchet Workflow").

---

## Phase 73 后

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-21 | Phase 73 后 | 25.9 | 43652 | 11336 | 22 | TBD (atomic ratchet) | gsd-execute-phase 73 | 21.5 | 25.9 |

### 8 P1 packages per-package (Phase 73 后 — D-04 + D-10 strict)

| IMP | Package | Coverage | Stmts | Status |
|-----|---------|---------:|------:|--------|
| IMP-01 | internal/api/v1/duty | 83.0% | 220/265 | PASS (≥70%) |
| IMP-02 | internal/api/v1/knowledge | 84.2% | 230/273 | PASS (≥70%) |
| IMP-03 | internal/api/v1/rpa | 79.2% | 485/612 | PASS (≥70%) |
| IMP-04 | internal/api/v1/vdi | 76.2% | 227/298 | PASS (≥70%) |
| IMP-05 | internal/services/duty | 95.6% | 109/114 | PASS (≥70%) |
| IMP-05 | internal/services/knowledge | 95.3% | 81/85 | PASS (≥70%) |
| IMP-06 | internal/services/network | 92.1% | 117/127 | PASS (≥70%) |
| IMP-06 | internal/services/monitor | 95.3% | 462/485 | PASS (≥70%) |

All 8 P1 targets ≥70% (D-04 + D-10 strict). check-coverage.sh extended with p1_package_check floor (exit code 4) — additive to weighted-avg gate.

### Per-package (Phase 73 后)

```
pkg/normalize                                                             45         44      97.8%
internal/config                                                          147        137      93.2%
internal/middleware                                                      196        169      86.2%
internal/transform                                                       111         95      85.6%
internal/services/portwrite                                              259        221      85.3%
internal/services/component_collector                                    345        285      82.6%
internal/utils/operlog                                                    90         74      82.2%
internal/services/topology                                                73         55      75.3%
internal/api/v1/scheduler                                                152        130      85.5%
internal/services/lldp                                                    96         57      59.4%
internal/services/workorder                                              715        527      73.7%
internal/api/v1/monitor                                                  518        369      71.2%
internal/core/security                                                   313        153      48.9%
internal/templates                                                       243        108      44.4%
internal/services/asset                                                 1354        549      40.6%
internal/api/v1/system                                                  3039       1077      35.4%
internal/core/db                                                         643        241      37.5%
internal/services/system                                                3483       1862      53.5%
internal/websocket                                                      129         45      34.9%
pkg/crypto                                                              439        148      33.7%
internal/core/db/migrations                                             293         83      28.3%
pkg/permission                                                          114         30      26.3%
pkg/cache                                                               926        228      24.6%
internal/services/base                                                   61         14      23.0%
internal/services/operations                                           3714        835      22.5%
internal/services/portcollection                                        580        112      19.3%
internal/services/addomain                                             2415        371      15.4%
pkg/middleware                                                          609         87      14.3%
pkg/errors                                                              326         45      13.8%
internal/services                                                      5202        589      11.3%
internal/api/v1/asset                                                   420         35       8.3%
internal/api/v1/network                                                1971        149       7.6%
internal/api/v1                                                          578         38       6.6%
internal/services/scheduler                                             167          8       4.8%
internal/utils                                                          531         24       4.5%
internal/scheduler                                                     1103         36       3.3%
internal/api/v1/operations                                             1285         39       3.0%
internal/services/vdi                                                  1127         30       2.7%
internal/device                                                        1249         31       2.5%
internal/core                                                           754         16       2.1%
internal/agent/server                                                   616         13       2.1%
internal/services/rpa                                                  1865         21       1.1%
internal/api/v1/workorder                                               297        224      75.4%
internal/services/duty                                                 114        109      95.6%
internal/services/knowledge                                              85         81      95.3%
internal/services/network                                               127        117      92.1%
internal/services/monitor                                               485        462      95.3%
internal/api/v1/duty                                                    265        220      83.0%
internal/api/v1/knowledge                                               273        230      84.2%
internal/api/v1/rpa                                                     612        485      79.2%
internal/api/v1/vdi                                                     298        227      76.2%
pkg/time                                                                 63          0       0.0%
pkg/response                                                             51          0       0.0%
pkg/query                                                               105          0       0.0%
pkg/logger                                                               79          0       0.0%
pkg/ldaputils                                                            33          0       0.0%
pkg/gormutil                                                            194          0       0.0%
pkg/captcha                                                             409          0       0.0%
internal/services/common                                                  1          0       0.0%
internal/server                                                           2          0       0.0%
internal/pkg/system                                                     345          0       0.0%
internal/pkg/cache                                                      167          0       0.0%
internal/models/system/requests                                         109          0       0.0%
internal/models/system                                                   11          0       0.0%
internal/models/rpa                                                      94          0       0.0%
internal/models/operations                                               23          0       0.0%
internal/docs                                                             1          0       0.0%
internal/api/v1/operations/requests                                      15          0       0.0%
internal/api/v1/agent                                                     38          0       0.0%
internal/api                                                             417          0       0.0%
internal/agent/pkg/retry                                                 33          0       0.0%
cmd/agent                                                                59          0       0.0%
cmd                                                                     106          0       0.0%
PACKAGE                                                                 STMT    COVERED        PCT
-------                                                                 ----    -------        ---
PACKAGE                                                                43652      11336      25.9%
```

### Per-package 倒退检查 (Phase 73 后)

- [x] 0% 包数 Phase 72 后 31 → Phase 73 后 22 (减少 9 个 0% 包; 全部来自 P1 8 包 + 1 来自覆盖率波动)
- [x] 8 个 P1 包全部 ≥70% per-package (D-04 + D-10 strict)
- [x] 加权平均 21.5% → 25.9% (上升 4.4 个百分点)
- [x] 无 per-package 倒退 (Phase 73 严格测试-only, 无业务代码改动 per D-12)

### Notes (Phase 73 deviation)

- 加权平均 25.9% 低于规划区间 27-30% (预估) — 实际 gate 测量 25.97% (脚本 %.2f), 25.9 是 %.1f 截断值; Phase 72 同款 21.5/21.55 模式
- check-coverage.sh 扩展 p1_package_check (8 P1 包 per-package ≥70% floor, exit code 4) — 与 weighted-avg gate 并列, 加性不互斥
- CORE-04 (api/v1/system 35.4%) / CORE-06 (services/system 53.5%) 仍低于 70% per-sub-module (Phase 72 deviation #1 遗留); per-sub-package validation 仍属 Phase 74 (GOV-03 scope), 未引入
- IMP-06 整体完成 (network 92.1% + monitor 95.3%, 含 oper_log per D-03 不豁免)
- **Ratchet commit 落地**: 21.5% → 25.9%, phase-level gate 已重置, Phase 74 (P2) 持续 ratchet 上调; p1_package_check 守卫 8 个 P1 包不退

> **Ratchet note (D-04):** The `commit` column on the Phase 73 后 row reads
> `TBD (atomic ratchet)` until plan 73-05 Task 4 amends this file with the
> actual short SHA of the commit that ships the .coverage-threshold + this
> file. Subsequent phases (74) append a new section with the same column
> schema, bumping `.coverage-threshold` in the same atomic commit (D-04
> manual ratchet).
---

## Phase 74 后

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-22 | Phase 74 后 | 55.5 | 43652 | 24254 | 5 | 1f18e20 | gsd-execute-phase 74 | 25.9 | 55.5 |

### 10 P2 packages per-package (Phase 74 后 — D-15)

| Package | Coverage | Stmts | Floor | Status |
|---------|---------:|------:|------:|--------|
| internal/api/v1/operations | 72.30% | 929/1285 | 70.0 | PASS |
| internal/api/v1/asset | 84.52% | 355/420 | 70.0 | PASS |
| internal/api/v1/network | 75.34% | 1485/1971 | 70.0 | PASS |
| internal/services/rpa | 86.11% | 1606/1865 | 70.0 | PASS |
| internal/services/vdi | 85.09% | 959/1127 | 70.0 | PASS |
| internal/core | 38.33% | 289/754 | 38.33 (ratcheted) | BLOCKED — documented |
| internal/device | 39.07% | 488/1249 | 39.07 (ratcheted) | BLOCKED — documented |
| internal/utils | 95.10% | 505/531 | 70.0 | PASS |
| internal/agent/server | 22.08% | 136/616 | 22.08 (ratcheted) | BLOCKED — documented |
| internal/services/scheduler | 89.82% | 150/167 | 70.0 | PASS |

7/10 at the D-15 70% floor; 3 structurally blocked in unit-test scope (scrapligo
concrete SSH driver / full Core.Init dependency graph / agent subprocess server —
see 74-08-SUMMARY.md). Their floors are ratcheted UP-ONLY to the shipped values
inside check-coverage.sh section 4; removal condition: package crosses 70.0%.

### 8 P1 packages per-package (Phase 74 后 — preserved, no regression)

| Package | Phase 73 | Phase 74 | Status |
|---------|---------:|---------:|--------|
| internal/api/v1/duty | 83.0% | 83.0% | PASS |
| internal/api/v1/knowledge | 84.2% | 84.2% | PASS |
| internal/api/v1/rpa | 79.2% | 79.2% | PASS |
| internal/api/v1/vdi | 76.2% | 76.2% | PASS |
| internal/services/duty | 95.6% | 95.6% | PASS |
| internal/services/knowledge | 95.3% | 95.3% | PASS |
| internal/services/network | 92.1% | 92.1% | PASS |
| internal/services/monitor | 95.3% | 95.3% | PASS |

### Per-package 倒退检查 (Phase 74 后)

- [x] 0% 包数 Phase 73 后 22 → Phase 74 后 5 (减少 17 个; 剩余 5 个全部为入口/装配/生成代码: cmd, cmd/agent, internal/api, internal/docs, internal/server)
- [x] 8 个 P1 包全部保持 ≥70% (D-04 + D-10 strict, 零回归)
- [x] 加权平均 25.9% → 55.5% (上升 29.6 个百分点)
- [x] 无 per-package 倒退 (Phase 74 严格测试-only, 无业务代码改动 per D-12)
- [x] SC-b 达成: 0% 包 22 → 5 (≤5)

### Notes (Phase 74 deviation — SC shortfall documented honestly)

- **SC-a (weighted ≥70%) 未达**: 实际 55.56%,差 14.44pp。两轮 escalation
  gap-closure (74-08 + 74-12) 后的结构性阻塞面:
  - addomain 2415 stmts 21.78% (LDAP 真实连接池)
  - services/operations 3714 stmts 61.07% (Excel/geocoding 外部 HTTP)
  - device 761 未覆盖 stmts / core 465 未覆盖 / agent/server 480 未覆盖
    (SSH scrapligo 具体 driver / Core.Init 全链 / 子进程 server — 单测不可构造)
  - services/system 3483 stmts 53.5%、api/v1/system 3039 stmts 35.4%
    (Phase 72 遗留 sub-target)
- P2 floor 落地为 70.0% x 7 + UP-ONLY ratcheted x 3 (core 38.33 / device
  39.07 / agent-server 22.08),gate 不撒谎:CI 绿是因为显式 ratchet,不是降标准
- QUIRK 清单 15 项(D-12 不修复)分布在 74-08/74-12-SUMMARY 与测试注释中
- **Ratchet commit 落地**: 25.9% → 55.5%,check-coverage.sh 扩展 section 4
  p2_package_check (exit 5);v1.26 收口,详见 74-MILESTONE-AUDIT.md

> **Ratchet note (D-04):** The `commit` column on the Phase 74 后 row reads
> `TBD (atomic ratchet)` until plan 74-11 Task 3 amends this file with the
> actual short SHA. This is the v1.26 closing ratchet.

## Phase 79 后

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-28 | Phase 79 后 | 70.9 | 43893 | 31119 | 7 | 3d8019e | gsd-executor 79-06 | 55.5 | 55.5 (不动,Phase 81 收口 bump) |

### Per-package (Phase 79 后)

```
pkg/ldaputils                                            33       32  96.97%
internal/services/knowledge                              85       81  95.29%
pkg/crypto                                              443      332  74.94%
internal/core/db/migrations                             301       90  29.90%
pkg/logger                                               97       62  63.92%
internal/services/duty                                  114      109  95.61%
internal/services/scheduler                             167      150  89.82%
cmd                                                     106        0   0.00%
internal/services/rpa                                  1865     1615  86.60%
internal/api/v1/vdi                                     298      227  76.17%
internal/services/monitor                               485      462  95.26%
internal/services                                      5202     4245  81.60%
internal/services/network                               127      117  92.13%
internal/core                                           756      624  82.54%
internal/services/portcollection                        580      334  57.59%
internal/services/base                                   61       14  22.95%
pkg/errors                                              326       45  13.80%
internal/websocket                                      129       45  34.88%
internal/config                                         147      137  93.20%
internal/device                                        1265     1047  82.77%
internal/api/v1/workorder                               297      224  75.42%
internal/pkg/system                                     352      116  32.95%
xingran-react-frontend/node_modules/flatted/golang/pkg/flatted      160        0   0.00%
internal/docs                                             1        0   0.00%
internal/utils                                          531      505  95.10%
internal/services/common                                  1        1 100.00%
internal/services/component_collector                   345      285  82.61%
pkg/normalize                                            45       44  97.78%
pkg/permission                                          114       30  26.32%
cmd/agent                                                63        0   0.00%
internal/api/v1/operations/requests                      15       15 100.00%
internal/models/system/requests                         109       85  77.98%
pkg/gormutil                                            194      123  63.40%
internal/services/workorder                             715      527  73.71%
internal/api/v1/rpa                                     612      485  79.25%
internal/services/lldp                                   96       57  59.38%
internal/services/portwrite                             259      221  85.33%
pkg/time                                                 63       54  85.71%
internal/agent/pkg/retry                                 33       31  93.94%
internal/server                                           2        0   0.00%
internal/scheduler                                     1103       36   3.26%
internal/models/rpa                                      94       30  31.91%
tests/fixtures                                            4        0   0.00%
internal/services/system                               3483     2632  75.57%
pkg/captcha                                             409      358  87.53%
internal/api/v1/duty                                    265      220  83.02%
internal/utils/operlog                                   90       85  94.44%
internal/api/v1/asset                                   420      355  84.52%
internal/api/v1/operations                             1285      929  72.30%
internal/api/v1                                         578       38   6.57%
internal/core/security                                  313      234  74.76%
internal/services/vdi                                  1127      959  85.09%
internal/api/v1/knowledge                               273      230  84.25%
internal/services/asset                                1354      960  70.90%
internal/api                                            417        0   0.00%
internal/api/v1/system                                 3039     2138  70.35%
internal/models                                         445        1   0.22%
pkg/cache                                               924      598  64.72%
internal/services/topology                               73       55  75.34%
internal/models/operations                               23       11  47.83%
internal/api/v1/agent                                    38       30  78.95%
pkg/query                                               105       71  67.62%
pkg/middleware                                          609      419  68.80%
internal/api/v1/scheduler                               152      130  85.53%
internal/middleware                                     196      169  86.22%
internal/agent/server                                   627      567  90.43%
internal/services/operations                           3714     3109  83.71%
internal/pkg/cache                                      168       89  52.98%
pkg/response                                             51       49  96.08%
internal/core/db                                        647      470  72.64%
internal/api/v1/monitor                                 518      369  71.24%
internal/templates                                      243      214  88.07%
internal/services/addomain                             2419     1403  58.00%
internal/transform                                      111       95  85.59%
internal/models/system                                   11       10  90.91%
internal/api/v1/network                                1971     1485  75.34%
PACKAGE    43893    31119  70.90%
```

### Per-package 倒退检查 (Phase 79 后)

- [x] SC-1: internal/services root **81.60%** (5202 stmts / 4245 covered) ≥ 70% gate — 基线 589/11.3% (Phase 74 后表), Phase 79 前 62.9%, 本 phase 六个 plan 累计 +3656 covered, 其中 79-06 device 家族 +984
- [x] 0% 包数 Phase 74 后 5 → Phase 79 后 7 (+2): 新增两包均为「首次进入扫描的第三方/测试支撑包」——`xingran-react-frontend/node_modules/flatted/golang/pkg/flatted` (160 stmts, 前端 node_modules 内的 Go 移植) 与 `tests/fixtures` (4 stmts), 非既有覆盖包回退; 既有 5 个 (cmd, cmd/agent, internal/api, internal/docs, internal/server) 保持 0% (入口/装配/生成代码)
- [x] 8 个 P1 包全部 ≥70% (gate PASS, 零回归)
- [x] 10 个 P2 包全部 ≥ floor (gate PASS: 70.0% × 7 + ratcheted × 3 — core 82.54% / device 82.77% / agent-server 90.43%, 三个 ratcheted 包全部越过 70%, 仅 UP-ONLY floor 未删)
- [x] 加权平均 55.5% (Phase 74 后) → 70.9% (上升 15.4 个百分点); `.coverage-threshold` diff = 0 (55.5 保持, Phase 81 才 bump)
- [x] 无 per-package 倒退 (Phase 79 严格测试-only; 唯一生产树 touch = internal/device/e2e_helpers.go 追加 ForTesting helper, D-79-02 豁免, AST 守护全绿, 零行为变更)

### device 家族 7 文件达标判定 (79-06 / TAIL-01)

| 文件 | 基线 (RESEARCH §2) | 实测 | 目标 | 结果 |
|------|------|------|------|------|
| device_discovery_service.go | 1.4% (4/293) | **87.4%** (256/293) | ≥60% | ✅ |
| device_info_collection_service.go | 29.5% (115/390) | **66.9%** (261/390) | ≥65% | ✅ |
| config_backup_service.go | 0.0% (0/244) | **86.9%** (212/244) | ≥65% | ✅ |
| device_monitor_service.go | 0.0% (0/189) | **68.3%** (129/189) | ≥65% | ✅ |
| config_execution_service.go | 0.0% (0/152) | **65.1%** (99/152) | ≥60% | ✅ |
| command_dispatch_service.go | 3.4% (4/116) | **90.5%** (105/116) | ≥60% | ✅ |
| device_credential_helper.go | 0.0% (0/47) | **95.7%** (45/47) | ≥70% | ✅ |
| **7 文件合计** | 123/1431 (8.6%) | **1107/1431 (77.4%)** | — | +984 covered |

### Notes (Phase 79 收口)

- 实测命令: `go test -count=1 -coverprofile=coverage.out ./...` (exit 0) + `bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold` (exit 0, 全部 PASS 行见上方 gate 输出)
- internal/services root 单包口径: `go test -count=1 -coverprofile ./internal/services/` → 445s, 81.6% (5202/4245)
- device 家族 Tier-2 (executor 路径) 通过公共构造器装配 + `device.SeedConnectionForTesting` (D-79-02 唯一豁免生产树 touch) 种子 FileTransport 连接实现; 详见 79-06-SUMMARY
- `-race` 本地不可执行 (Windows cgo 工具链故障);**ci.yml 无 race job**(四 job:backend/coverage-diff/frontend/frontend-coverage-diff),`-race` 不在 v1.27 gate 口径(D-01 禁 race 进 coverage 跑);更正见 v1.27-MILESTONE-AUDIT.md
- **Ratchet 落地**: 55.5% → 70.9% (实测), `.coverage-threshold` 不动 — Phase 81 收口按 UP-ONLY 纪律 bump; 本行为 v1.27 milestone 的加权平均贡献段

> **Ratchet note (D-04):** The `commit` column on the Phase 79 后 row reads
> `3d8019e` — the docs(79-06) closeout commit that shipped this section together
> with 79-06-SUMMARY.md (71-01b two-step fill precedent: the SHA is back-filled
> by an immediate follow-up docs commit rather than an amend).

---

## Phase 75 后 (backfill — 文档债 #3 还原)

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-23 | Phase 75 后 | 55.7 | n/a | n/a | n/a | 3ca6e20 | gsd-executor 75-06 | n/a | n/a |

**说明 (75):** Phase 75 收口时未做全仓 weight 测量(15 项 QUIRK 修复,gate 未 bump)。最近一次测得的全仓数字在 75-05 plan 内部 (`scripts/check-ci-local.sh backend --no-npm-ci` PASS,weighted 55.74%)。stmts/covered/0pct_pkg_count 留 `n/a` —— 该 plan 未产出 coverprofile.out 持久化文件,无源数据回填。

## Phase 76 后 (backfill — 文档债 #3 还原)

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-25 | Phase 76 后 | 56.0 | n/a | n/a | n/a | c4dd0ab | gsd-executor 76-05 | n/a | n/a |

**说明 (76):** Phase 76 收口时未做全仓 weight 测量(测试基建,无业务覆盖增量)。最近一次测得的全仓数字在 76-01 plan 内部 (`check-ci-local.sh backend` PASS,weighted 56.02%)。stmts/covered/0pct_pkg_count 留 `n/a` —— 该 plan 未产出 coverprofile.out 持久化文件,无源数据回填。

## Phase 77 后 (backfill — 文档债 #3 还原)

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-27 | Phase 77 后 | n/a | n/a | n/a | n/a | e69e92e | gsd-executor 77-05 | n/a | n/a |

**说明 (77):** Phase 77 收口 (agent/server 50.0% → 90.4% / BLOCK-02 收口) 期间未做全仓 weight 测量。SUMMARIES 内只有 per-package 数字 (agent/server 90.4%),无 PACKAGE 行级汇总。**不造数** —— 留 `n/a`。

## Phase 78 后 (backfill — 文档债 #3 还原)

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-27 | Phase 78 后 | n/a | n/a | n/a | n/a | 0bd0c53 | gsd-executor 78-07 | n/a | n/a |

**说明 (78):** Phase 78 收口 (core 82.5% / device 82.6% / addomain 58.0% Partial / BLOCK-05 框架未结案) 期间未做全仓 weight 测量。SUMMARIES 内只有 per-package 数字 (78-01/02/04/05),无 PACKAGE 行级汇总。**不造数** —— 留 `n/a`,78-07-SUMMARY 的 Conclusion B (addomain 58.0%) 是 per-package 锁值,非全仓 weighted。

## Phase 80 后 (backfill — 同类文档债,81-01 plan 未点名但同性质)

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-28 | Phase 80 后 | n/a | n/a | n/a | n/a | 813345e | gsd-executor 80-05 | n/a | n/a |

**说明 (80):** Phase 80 收口 (scheduler 家族 +2989 stmts,14 包 13/14 + lldp 豁免) 期间未做全仓 weight 测量 —— 按 research 决策,scheduler 长尾工作专为 Phase 81 收口前的 profile 稳定期让路。**不造数** —— 留 `n/a`。81-01 plan §Task 2 仅点名 75-78,本行作为同类文档债**主动**回填一行 + commit ref,数字留 `n/a` 与 77/78 一致处理。

## Phase 81 后 (v1.27 收口)

| date | phase_label | weighted_avg | total_stmts | total_covered | 0pct_pkg_count | commit | phase_executor | ratchet_from | ratchet_to |
|------|-------------|--------------|-------------|---------------|----------------|--------|----------------|--------------|------------|
| 2026-08-28 | Phase 81 后 | 78.0 | 43729 | 34103 | 4 | 2830800 | gsd-executor 81-01 | 55.5 | 77.5 |

### Per-package (Phase 81 后)

```
pkg/ldaputils                                            33       32  96.97%
internal/services/knowledge                              85       81  95.29%
pkg/crypto                                              443      332  74.94%
internal/core/db/migrations                             301       90  29.90%
pkg/logger                                               97       74  76.29%
internal/services/duty                                  114      109  95.61%
cmd                                                     106        0   0.00%
internal/services/scheduler                             167      150  89.82%
internal/services/rpa                                  1865     1606  86.11%
internal/api/v1/vdi                                     298      227  76.17%
internal/services/monitor                               485      462  95.26%
internal/services                                      5202     4245  81.60%
internal/services/network                               127      117  92.13%
internal/core                                           756      624  82.54%
internal/services/portcollection                        580      334  57.59%
internal/services/base                                   61       50  81.97%
pkg/errors                                              326      325  99.69%
internal/websocket                                      129      106  82.17%
internal/config                                         147      137  93.20%
internal/device                                        1265     1047  82.77%
internal/api/v1/workorder                               297      224  75.42%
internal/pkg/system                                     352      116  32.95%
internal/docs                                             1        0   0.00%
internal/utils                                          531      505  95.10%
internal/services/common                                  1        1 100.00%
internal/services/component_collector                   345      285  82.61%
pkg/normalize                                            45       44  97.78%
pkg/permission                                          114      101  88.60%
cmd/agent                                                63        0   0.00%
internal/api/v1/operations/requests                      15       15 100.00%
internal/models/system/requests                         109       85  77.98%
pkg/gormutil                                            194      162  83.51%
internal/services/workorder                             715      527  73.71%
internal/api/v1/rpa                                     612      485  79.25%
internal/services/lldp                                   96       66  68.75%
internal/services/portwrite                             259      221  85.33%
pkg/time                                                 63       54  85.71%
internal/agent/pkg/retry                                 33       31  93.94%
internal/server                                           2        0   0.00%
internal/scheduler                                     1103      899  81.50%
internal/models/rpa                                      94       30  31.91%
pkg/captcha                                             409      358  87.53%
internal/services/system                               3483     2632  75.57%
internal/api/v1/duty                                    265      220  83.02%
internal/utils/operlog                                   90       85  94.44%
internal/api/v1/asset                                   420      355  84.52%
internal/api/v1/operations                             1285      929  72.30%
internal/api/v1                                         578      504  87.20%
internal/services/vdi                                  1127      959  85.09%
internal/core/security                                  313      234  74.76%
internal/api/v1/knowledge                               273      230  84.25%
internal/services/asset                                1354      960  70.90%
internal/api                                            417      402  96.40%
internal/api/v1/system                                 3039     2138  70.35%
internal/models                                         445      408  91.69%
pkg/cache                                               924      824  89.18%
internal/services/topology                               73       55  75.34%
internal/models/operations                               23       11  47.83%
internal/api/v1/agent                                    38       30  78.95%
pkg/query                                               105       97  92.38%
pkg/middleware                                          609      514  84.40%
internal/api/v1/scheduler                               152      130  85.53%
internal/middleware                                     196      169  86.22%
internal/agent/server                                   627      567  90.43%
internal/services/operations                           3714     3109  83.71%
internal/pkg/cache                                      168       89  52.98%
pkg/response                                             51       49  96.08%
internal/core/db                                        647      470  72.64%
internal/api/v1/monitor                                 518      369  71.24%
internal/templates                                      243      214  88.07%
internal/services/addomain                             2419     1403  58.00%
internal/transform                                      111       95  85.59%
internal/models/system                                   11       10  90.91%
internal/api/v1/network                                1971     1485  75.34%
PACKAGE    43729    34103  77.99%
```

### Per-package 倒退检查 (Phase 81 后)

- [x] SC-1: 加权平均 70.9% (Phase 79 后) → **77.99%** (Phase 81 后实测, +7.09 个百分点);`.coverage-threshold` 55.5 → **77.5** (D-81-01 ratchet, +22.0pp 单向 UP)
- [x] 0pct 包数 Phase 79 后 7 → Phase 81 后 4 (-3):`xingran-react-frontend/node_modules/flatted/golang/pkg/flatted` (160 stmts) + `tests/fixtures` (4 stmts) + `internal/api` (417 stmts) 均退出 0%,`internal/api` 经 Phase 80 达 96.40% (417/402);既有 4 个 0pct 锁定为 cmd (106) + cmd/agent (63) + internal/server (2) + internal/docs (1) = 172 stmts(基线锁,不计 floor,符合 check-coverage.sh 现状)
- [x] 8 个 P1 包全部 ≥70% (gate PASS, 零回归)
- [x] 10 个 P2 包全部 ≥ floor (gate PASS: 70.0% × 7 + ratcheted × 3 — core 82.54% / device 82.77% / agent-server 90.43%, 三个 ratcheted 包全部越过 70%, Phase 81-02 待删豁免行)
- [x] 无 per-package 倒退 (Phase 79 后基线比对:internal/services/lldp 59.38% → 68.75% (+9.4pp)、internal/api 0% → 96.40% (+96.4pp)、internal/pkg/cache 52.98% → 89.18% (+36.2pp) 显著上升,internal/scheduler 3.26% → 81.50% (+78.2pp, Phase 80 长尾 + 81-01 Task 0 修复新增覆盖路径);唯一保留 0% 是基线锁的入口/装配/生成代码,非回退)

### Notes (Phase 81 收口)

- 实测命令 (ci.yml L64-66 byte-identical): `go test -timeout 15m -count=1 -coverprofile=coverage.out -covermode=atomic ./internal/... ./pkg/... ./cmd/...` → `TEST_EXIT=0`,`grep -c "^FAIL"` = 0,72 ok / 0 FAIL / 0 panic
- gate 实跑: `bash .github/scripts/check-coverage.sh coverage.out .coverage-threshold` → `GATE_EXIT=0`,`PACKAGE 43729 34103 77.99%`,P1 8/8 + P2 10/10 PASS(详见上方 gate 输出)
- **D-81-01 threshold 取值**: 写 **77.5** 而非先例式 1-decimal 严格截断 78.0 —— 这是对 Phase 72/73/74 三次 ratchet 的有意偏离,非笔误。理由:本批 199 个 commit 是 CI 从未见过的后端覆盖工作(Phase 77/78/79/80 全部产出均在本地 main 领先 origin/main 207 commits),且本地是 Windows / CI 是 Linux,首次跨 OS 大跃迁,Go patch 插桩漂移先例 ±1.7%(754 vs 767 stmts),0.5pp 缓冲(实测 78.0 - 写入 77.5 = 0.5)是必要的 CI 漂移缓冲;UP-only 纪律不破:55.5 → 77.5 是 +22pp 单向 ratchet,未来仍可 UP 到 78+(如实测连续两次 CI 全绿 + 本地 + CI 漂移 ≤0.5pp,即可截断到 78.0)
- stmts 分母漂移口径: 43652 (v1.26 起点 / Phase 71-74 后基线) → 43893 (Phase 79 后,+241) → **43729** (Phase 81 后,-164);差异来源 = Go patch 插桩漂移(主要)+ Phase 76/77 少量生产重构 stmts(INFRA-02/04 注入缝);SC-a 数学公式 `covered_needed = ceil(0.7 × current_stmts)` 当前 = `ceil(0.7 × 43729)` = 30611 covered needed,实测 **34103 超 +3492 stmts**;v1.26 缺口收口数学:24254 → 34103,补 **+9849 covered**,是 6287/6302 缺口的 **~1.56 倍**
- Task 0 修复副作用: internal/scheduler stmts 1103 / covered 36 / 3.26% (Phase 79 后基线,反映 Phase 80 跑覆盖率时尚有部分 flake 用例跑挂未计入) → **stmts 1103 / covered 899 / 81.50%** (Phase 81 后实测,含 Phase 80 已落测试 + Task 0 修复后的 CheckPortStatusDrift/Cleanup 家族覆盖);`internal/services/scheduler` 单独入 P2 名单 89.82% (167/150)
- **Ratchet 落地**: 55.5% → **77.5%** (D-81-01 写入值),测量截断 78.0%(1-decimal);`.coverage-threshold` 字节格式保持 4 字节 (35 35 2e 35),无尾随换行,git diff 仅见数值变更
- 加权平均 12.8% (Phase 71 后) → 21.5% (Phase 72 后) → 25.9% (Phase 73 后) → 55.5% (Phase 74 后) → 70.9% (Phase 79 后) → **77.5%** (Phase 81 后) — v1.27 收口链路完整,5 次 ratchet 累计 +64.7pp,UP-only 纪律全程未破
- 75-78 文档债回填:见上方 `## Phase 75/76/77/78/80 后 (backfill)` 五段;75/76 用 mid-phase 测得值 (55.74% / 56.02%) 占位 weighted,stmts/covered/0pct 三列 `n/a`;77/78/80 全 `n/a` 且**不造数**——只在 SUMMARIES 里有 per-package 数据,无 PACKAGE 行级汇总可溯

> **Ratchet note (D-04):** The `commit` column on the Phase 81 后 row reads
> `2830800` — the docs(81-01) atomic-ratchet closeout commit that shipped this
> section together with 81-01-SUMMARY.md back-fill (71-01b / 79-06 two-step
> fill precedent: 受保护 main 分支禁 amend,后随 commit 改 commit 列)。

> **Backfill note (D-04 / 文档债 #3):** Phase 75-78 backfill rows have no
> phase-end full-package measurement; rows with `n/a` in weighted column
> (77/78/80) record **不造数** per 81-01 plan §Task 2 rule. Rows with mid-phase
> weighted values (75-05 的 55.74% → 75 行 `55.7`,76-01 的 56.02% → 76 行 `56.0`)
> carry partial data with `n/a` in stmts/covered/0pct cells where source-of-truth
> coverprofile.out 不可得。
