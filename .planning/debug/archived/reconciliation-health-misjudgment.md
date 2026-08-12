# 资产对账健康判断逻辑错误

**Slug**: `reconciliation-health-misjudgment`
**Created**: 2026-07-02
**Reporter**: 用户
**Status**: 🟢 all_three_fixes_applied
**Module**: 资产对账 (v1.17)

## 现象 (Symptoms)

### 现象 1: 4F001 工位健康状态全部错误显示红色

工位 4F001 (浙商大厦 4楼 / 运营服务部 / 程步启 / 序列号 1CZ151008W) 下:
- 域控设备 (1台): CXHUB-151008W / 1CZ151008W / HP 400 G7 / B0:22:7A:2E:4A:4F / 责任人程步启 / 状态:正常 / **对账健康:🔴 红点**
- 资产设备 (1台): CXHUB-151008W / 1CZ151008W / HP 400 G7 / B0:22:7A:2E:4A:4F / 责任人程步启 / 状态:正常 / **对账健康:🔴 红点**
- 物理链路设备 (1台): CXHUB-151008W / 1CZ151008W / HP 400 G7 / B0:22:7A:2E:4A:4F / 10.62.8.1 / 责任人程步启 / 状态:正常 / 实测 / **对账健康:🔴 红点**

三条路径都是同一台设备,所有元数据一致,设备名/MAC/序列号/型号/责任人完全相同,但健康状态全红。
用户报告健康详情是: **"物理有/责任人不一致(高危)"**

### 现象 2: 异常标签判定错位

- 截图 9 (2026-07-02 12:01:57): 标签 "物理有责任/无人" 高危, 数据是 wangwenye-001 / yangfan-131 → 实际应该是"物理使用人和责任人不一致"(两边都有人)
- 截图 10 (缺数据 中): MAC 4CE9160ZJZ / IP 10.62.8.55 / 设备名 xiaoshan / 责任人 xiaoshan → 实际设备名=责任人,应该是**正常**,无冲突
- 截图 11 (物理无责 高): 序列号 1CZ033400V1 / IP 10.62.6.3 / 设备名 luowei-020 / 责任人 `-` → 应该是"物理使用人有,责任人无"

## 调查进度 (Phase 1-4)

### Phase 1: 代码范围

**关键文件:**
- `internal/services/asset/reconciliation_detection.go` - Layer 3 检测引擎(ClassifySignals / ClassifyType / ComputeSeverity)
- `internal/services/asset/reconciliation_service.go` - GetByWorkstation 聚合 (R4 / D-A4-02)
- `internal/services/asset/reconciliation_exception_matcher.go` - 例外规则匹配
- `internal/core/db/migrations/migration_169_reconciliation_dicts_configs.go` - dict_label seed
- `internal/core/db/migrations/migration_181_reconciliation_normalized_workstation_id.go` - MV reconciliation_normalized 最新定义
- `xingran-react-frontend/src/components/reconciliation/HealthBadge.tsx` - 字典 → 中文 tooltip 映射 (硬编码)
- `xingran-react-frontend/src/pages/asset/reconciliation/exceptions/index.tsx` - 异常列表 (dictLabel 直接渲染)

### Phase 2-4: 关键发现 — 三套冲突的 A-F 语义

#### 发现 1: 检测引擎 (reconciliation_detection.go) 对 A-F 的定义
| Type | 含义 | 信号(HasPhysical, HasDeclared, Match) |
|------|------|---------------------------------------|
| A | physical + declared 匹配(健康) | P=Y, D=Y, P==D, AD可不一致 |
| B | physical 有, declared 无 | P=Y, D=N |
| C | physical + declared 都有但不匹配 | P=Y, D=Y, P≠D |
| D | physical 无, declared 有 | P=N, D=Y |
| E | physical + declared 都没有 | P=N, D=N |
| F | physical/declared 匹配但 AD 不一致 | A 的 AD 分支 |

#### 发现 2: dict seed (migration_169:83-90) 对 A-F 的标签
| dictValue | dictLabel(seed) | listClass |
|-----------|------------------|-----------|
| A | A类-完全一致 | success |
| B | B类-物理无责 | warning |
| C | C类-物理有责无 | error |
| D | D类-物理与责任人不一致 | warning |
| E | E类-三方不一致 | default |
| F | F类-缺数据 | info |

#### 发现 3: HealthBadge.tsx tooltip (硬编码,frontend/src/components/reconciliation/HealthBadge.tsx:25-32) 对 A-F 的定义
| Type | 硬编码中文 |
|------|----------|
| A | 物理有/责任人有且一致 |
| B | 物理有/责任人无 |
| C | 物理有/责任人不一致(高危) |
| D | 物理无/责任人有 |
| E | 三路数据均未检测到该资产(疑似幽灵资产) |
| F | 仅 AD managed_by 与系统登记不一致(AD 已知不可靠) |

#### 发现 4: sys_workorder_category (migration_169:222-227) 对 A-F 的描述
| Type | 描述 |
|------|------|
| A | 物理有/责任人有且一致 健康无需动作 |
| B | 物理无(未采集)/责任人有 |
| C | 物理有/责任人无 |
| D | 物理有/责任人有但不一致 |
| E | 三方(物理/责任人/AD)互不一致 |
| F | 缺数据 |

## 根因 (Root Cause)

**三处独立的 A-F 语义定义互相矛盾,导致同一资产在列表/详情/dict 呈现中走错标签。**

具体冲突点:
1. **dict seed (admin 实际展示)** 把 **B 类** 定义成 "物理无责"(即 D 行为),把 **D 类** 定义成 "物理与责任人不一致"(即 C 行为)。
2. **detection engine (后端真分类逻辑)** 才是判定分类的权威源,完全以 signals 为准。
3. **HealthBadge.tsx (tooltip 文本)** 又是第三方独立定义。

**这就是 "截图 9 (物理有责任/无人)" 的根本原因 — dict 标签 "物理有责任/无人" 来自 `dictLabel="C类-物理有责无"` + tooltip 字符串拼接,但按 detection.go,C 应该是 "物理使用人与责任人冲突",即 "物理有/责任人不一致",但 seed 写的 `C类-物理有责无`,若前端渲染时只 show dictLabel 会出现标签与实际语义错位。**

**4F001 红点场景的根因:** 既然三路径设备元数据完全一致,detection.go 应当按 Type A (物理+声明匹配)归类 → 健康。但截图显示全红,说明 detection 在该场景下走入了非 A 分支。最可能原因 (待验证):
- `reconciliation_normalized.MV` 的 `physical_user_id` 和 `asset_user_id` 中至少有一个为 NULL 或不匹配,导致 ClassifySignals 误判。
- 现象 1 中 "责任人程步启" 和设备名匹配,但 physical_user_id 链路 `reconciliation_physical_chain.pc.physical_user_id` 可能因 workstation→user 关联异常而填入的 user_id 与 device 责任人 sys_user.id 不一致(类型/范围不匹配) → 走 C 分支 → 红点。

## 假设空间 (下一步验证)

1. 假设 `physical_user_id` 与 `asset_user_id` 在 ws+asset 关联处的 JOIN 生成空或不匹配
2. 假设 `reconciliation_physical_chain` 物理链路 view 的 user_id 取错表 (sys_workstation.user_id vs sys_user)
3. 假设 B/D 标签文案文字顺序与 A/C 标签暗示顺序相反(微观层面误导)

## 修复进展 (Fix Applied)

### Phase 5: 修复实施 (2026-07-02 16:xx)

**用户决策:** 先修 dict seed (低风险), 4F001 红点另起一轮 DB 验证.

**实施:** 新增 `migration_196_reconciliation_dict_labels_align.go`, 在 `internal/core/db/database.go` 注册.

**修复范围 (两轮合并):**

1. **dict_label 文字对齐 detection (5 处 UPDATE):**
   - A: 完全一致 (不变)
   - B: 物理无责 → 物理有/责任人无
   - C: 物理有责无 → 物理有/责任人不一致(高危)
   - D: 物理与责任人不一致 → 物理无/责任人有
   - E: 三方不一致 → 双方都无用户关联
   - F: 缺数据 → 物理与责任人一致但 AD 不一致

2. **list_class 颜色档次对齐 ComputeSeverity (2 处 UPDATE):**
   - B (high): warning → error
   - F (medium): info → warning

**用户反馈 (第二轮触发):** "ABCDEF 不是按严重度排序的吗" → 我澄清字母按 signals 组合分配 (非严重度), 但 seed list_class 颜色档次也错 2 处, 同步修正.

**不动:**
- detection engine logic (用户要求"只改显示, 不动检测")
- HealthBadge.tsx (已是权威文案)
- sys_workorder_category 描述 (同根因, 留待后续)

**验证:**
- ✅ `go build ./...` EXIT=0
- ✅ `go vet ./internal/core/db/migrations/...` EXIT=0
- ✅ `internal/services/asset/...` 测试全部通过
- 失败的 integration tests (login_encryption 等) 预先存在, 与本改动无关

## 任务清单
- [x] 编写 migration_196 修 dict_label + list_class
- [x] 注册 migration_196 到 database.go
- [x] go build / go vet / go test 验证
- [x] commit M196 (0c202ad5)
- [x] **4F001 红点深挖**: scripts/diag/red_4f001/ 一次性诊断脚本, 跑出真相
- [x] **修复 4F001 红点**: reconciliation_detection.go:235-263 加 auto-resolve-on-healthy 逻辑
- [x] **同步修 workorder_category 描述**: migration_197 (新建, 在 database.go 注册)
- [ ] commit 4F001 修复 + M197

## 完整修复清单 (3 个修复, 同根因)

### 修复 1 (M196): dict_label + list_class 颜色
文件: `internal/core/db/migrations/migration_196_reconciliation_dict_labels_align.go`
状态: ✓ commit 0c202ad5

### 修复 2: detection 引擎 Type A 时 auto-resolve 历史 open 冲突
文件: `internal/services/asset/reconciliation_detection.go:235-263`
影响: 4F001 红点 + 所有"已自愈但 open 记录未清理"的资产
部署后: 5-6 分钟 cron 自动清, 无需手动 SQL

### 修复 3 (M197): sys_workorder_category 描述同步对齐
文件: `internal/core/db/migrations/migration_197_reconciliation_workorder_categories_align.go`
状态: ✓ 已注册, 待 commit
