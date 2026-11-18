# 机构层级数据权限 —— 独立核验报告

> **核验者**：verifier（只读审计员，独立于开发）
> **核验日期**：2026-11-17
> **契约基准**：`.planning/notes/261118-anchor-scope-contract.md`（§一 C1~C7、§九 验收标准）
> **核验方式**：不采信开发交付时的自述，全部结论均由**亲自读代码 + 跑命令 + 补测试**得出。
> **是否修改实现**：**否**。本次仅新增/修改 `*_test.go`（测试）与本报告。

---

## 一、结论

# **PASS-with-issues**

**一句话结论**：契约 §一 的 7 条强制约束（C1~C7）与 §九 的 6 条验收用例**全部通过**，无一条被违背；编译、测试、前端检查全绿。
但发现 **1 个阻塞级安全漏洞**（岗位锚点可被低权限操作者自助挂载 → 权限提升）与 **2 条有风险的静默失败/降权路径**，必须在上线前处理或由 owner 明确接受风险。

| 维度 | 结论 |
|---|---|
| `go build ./...` | ✅ 0 error |
| `go test ./...` | ✅ 0 FAIL 包（82 个 ok） |
| `npm run type-check` / `npm run lint` | ✅ 均 exit 0 |
| 契约 §九 六条用例 | ✅ 6/6 通过（含我补的用例 B 精确语义测试） |
| 静态审查 9 项 | ✅ 9/9 合规（2 项附带观察） |
| 边界审查 7 项 | ⚠️ 1 漏洞 / 3 有风险 / 3 安全 |
| 测试覆盖缺口 | ⚠️ 原测试**未证明中间件真的并入了锚点**（已由我补齐 4 条端到端测试） |

---

## 二、编译与测试输出摘要（实际命令与结果）

```bash
# 1) 全仓编译
$ cd D:/CODE/ClaudeCode/ruoyi-go-backend && go build ./...
BUILD_EXIT=0                      # 0 error

# 2) 契约指定三包
$ go test ./pkg/datascope/ ./pkg/middleware/ ./internal/services/system/
ok  github.com/xingran-next/xingran-go-backend/pkg/datascope               (cached)
ok  github.com/xingran-next/xingran-go-backend/pkg/middleware              0.593s
ok  github.com/xingran-next/xingran-go-backend/internal/services/system    8.329s
TEST_EXIT=0

# 3) 全仓测试（我在补齐测试后重跑）
$ go test ./...
ok 包数量 = 82 ; FAIL 计数 = 0

# 4) 前端（前端目录下）
$ npm run type-check    # tsc --noEmit -p tsconfig.app.json
TC_EXIT=0
$ npm run lint          # eslint . --max-warnings 0
LINT_EXIT=0
```

**新增测试（verifier 补，全部 PASS）**：

```bash
$ go test ./pkg/middleware/ -run "TestDataScopePermission_AnchorWiring" -v
--- PASS: TestDataScopePermission_AnchorWiring_Level2IncludesBranch (0.08s)
--- PASS: TestDataScopePermission_AnchorWiring_AllSurvives (0.07s)
--- PASS: TestDataScopePermission_AnchorWiring_SelfOverridden (0.07s)
--- PASS: TestDataScopePermission_AnchorWiring_NoAnchorIsIdentity (0.06s)

$ go test ./pkg/datascope/ -run "TestContractCaseB|TestMerge_Rule|TestResolveAnchor_LevelParamToleratesWhitespace|TestValidateAnchor_DeptModeAcceptsAnyExistingDeptID" -v
--- PASS: 6/6
```

---

## 三、契约 §九 六条用例 —— 逐条结果

> 判定标准：必须找到**具体测试函数名**且该测试**确实断言了契约语义**，不接受「看起来没问题」。
> 「原测试」= 开发交付时已有的；「补测」= 本次核验新增。

| 用例 | 契约要求 | 证据（文件:行号 / 测试名） | 结论 |
|---|---|---|---|
| **A** | 刘洋（dept=综合管理部，ancestors 含武汉中支）持 `anchor_mode=5/param="2"` → 可见集**必须包含** level=3 的硚口支公司 | 原测试 `pkg/datascope/anchor_test.go:100` `TestResolveAnchor_Level2_HitsSubBangAndDescendants`（断言 `D-qk` 在子树内，`:114`）<br>**补测（中间件级，更关键）** `pkg/middleware/anchor_mount_test.go:163` `TestDataScopePermission_AnchorWiring_Level2IncludesBranch` —— 真实 `DataScopePermission` + 真实 SQLite 夹具，断言注入的 `ScopeResult.DeptIDs` 含 `D-qk`（`:206`） | ✅ **PASS** |
| **B** | 同上但 `param="3"` → 未命中 → FailClosed 空集 | 原测试 `anchor_test.go:121` `TestResolveAnchor_Level3_NotMatched_FailClosed`（断言 `!resolved`、`AnchorReasonLevelMiss`、`anchorID != D-zh`）<br>**补测（精确语义）** `anchor_test.go:364` `TestContractCaseB_Level3Miss_AnchorEmptyAndNoDeptFallback`：① 锚点维度空集且不回退 `D-zh`（C3）；② 角色维度为空 → `FailClosed=true`；③ 角色维度非空 → **保留角色维度且不 FailClosed**（C2 纯增量） | ✅ **PASS**（附契约措辞偏差，见问题 P-3） |
| **C** | `All(1)` 角色 + 有效锚点 → 仍 `All=true`（**C1 静默降权回归**） | 原测试 `anchor_test.go:167` `TestMerge_AllSurvivesAnchorDimension`（纯函数层）<br>**补测（中间件级）** `anchor_mount_test.go:212` `TestDataScopePermission_AnchorWiring_AllSurvives`：断言 `All=true`、`DeptIDs` 为空、且 `c.Set("data_scope")` 仍为 `models.DataScope(1)` 未被改动 | ✅ **PASS**（最关键回归已闭环） |
| **D** | `Self(5)` 角色 + 有效锚点 → `Hook=None` 且 `DeptIDs` 非空 | 原测试 `anchor_test.go:176` `TestMerge_SelfOverriddenByAnchor`（纯函数层）<br>**补测（中间件级）** `anchor_mount_test.go:254` `TestDataScopePermission_AnchorWiring_SelfOverridden`：断言 `HookNone`、`DeptIDs` 非空、含 `D-qk` | ✅ **PASS**（但该语义本身有安全争议，见 C-1） |
| **E** | 无任何锚点来源 → 与原 `Resolve` 输出**逐字段相等**（C2 零回归） | 原测试 `anchor_test.go:186` `TestMerge_NoAnchor_IsIdentity`（4 种 ScopeResult 全字段 `assert.Equal`）<br>**补测（中间件级）** `anchor_mount_test.go:295` `TestDataScopePermission_AnchorWiring_NoAnchorIsIdentity`：中间件跑完后取 `FromContext` 结果，与独立调用 `datascope.Resolve(...)` **整体 struct 相等** | ✅ **PASS** |
| **F** | `ValidateAnchor(1)`/`(4)` 报错；`mode=5` 的 `""`/`"0"`/`"abc"` 报错 | 原测试 `anchor_test.go:200` `TestValidateAnchor_Rejects`（表驱动 10 例：1/4/9/空/"0"/"5"/"abc"/层级不存在/dept 空/dept 不存在），逐例 `require.Error` + 错误文案匹配 | ✅ **PASS** |
| F-额外 | `mode=3` 但库里 `is_external_org=1` 计数为 0 → 报错 | 原测试 `anchor_test.go:248` `TestValidateAnchor_ExternalOrg_NoCandidates`（先 `UPDATE ... is_external_org = 0` 再断言报错） | ✅ **PASS** |

### 关于「原测试不足以签字」的判断

契约 §九 要求的是**用户可见的最终效果**（"可见集含硚口支公司"）。原测试只覆盖到 `ResolveAnchor` / `Merge` 的**纯函数**层，没有任何一条验证 `DataScopePermission` 是否真的调用了 `Merge` 并把结果注入 context。也就是说：如果开发的接线写错了（例如写成 `anchorDeptIDs, _ := ...` 后面忘了 Merge），原测试**全绿但功能不生效**。

我已补齐 4 条中间件级端到端测试（`anchor_mount_test.go:163/212/254/295`），用真实 `core.Core` + 真实 SQLite 夹具 + 真实中间件跑通 A/C/D/E 四条。这是本次核验**对测试资产最大的增量**。

---

## 四、静态审查 9 项 —— 逐条结论

| # | 审查项 | 证据 | 结论 |
|---|---|---|---|
| **B1** | `getUserMaxDataScope` 与 `datascope.Resolve(...)` 调用**未被改动**（C1） | `git diff pkg/middleware/permission.go` 只新增 16 行（`@@ -534,6 +534,22 @@`），全部位于 `datascope.Resolve` 调用**之后**；`getUserMaxDataScope` 位于 `permission.go:560`，diff 中**无任何行触及**；`Resolve` 调用位于 `:531`，同样未变 | ✅ **PASS** |
| **B2** | `pkg/datascope/datascope.go` **无改动**（`Resolve` 原样） | `git diff pkg/datascope/datascope.go` 输出**完全为空**（命令已跑，无输出） | ✅ **PASS** |
| **B3** | `ScopeResult` 仍是 5 字段（C4） | `pkg/datascope/datascope.go:43-49`：`All bool / DeptIDs []string / ResourceHook ResourceHook / FailClosed bool / UserID string` —— 恰好 5 个字段，且该文件零改动 | ✅ **PASS** |
| **B4** | 无第二套 `ancestors LIKE`（C5） | 全仓 `grep "ancestors LIKE"`：命中仅 `pkg/datascope/datascope.go:237-239`（`queryDeptChildIDs` 唯一实现）+ 既有 ops/system 实现（`building_service_typesafe.go:133`、`dept_filter.go:52`、`department_service.go:168`、`user_service.go:444`）。**`anchor.go` 内 `ancestors LIKE` 只出现在注释（`:20`、`:184`，均为"禁止第二套"的说明文字），无实际 SQL** —— anchor.go 复用了 `queryDeptChildIDs`（`:512`） | ✅ **PASS** |
| **B5** | `anchor.go` 所有 SQL 都有 `CAST(... AS TEXT)`（C7） | 逐条核对（`:92/93`、`:106/107`、`:125/126`、`:140/141`、`:156`、`:427-434`、`:451-458`）**全部带 CAST**。<br>两处例外均为**正确例外**：`:364` `is_external_org = 1`（int 常量比较，非 uuid/varchar）、`:380` `org_level = ?`（int 列与 int 参数，非 uuid/varchar）。无遗漏 | ✅ **PASS** |
| **B6** | 前端 `role/constants.ts` 的 `DATA_SCOPE_OPTIONS` 仍是原 5 项、未混入机构选项 | `xingran-react-frontend/src/pages/system/role/constants.ts:9-15`：全部数据(1)/自定义(2)/本部门(3)/本部门及子部门(4)/仅本人(5) —— **5 项，与 HEAD 一致**（`git status` 中该文件未被修改）。机构选项在**独立文件** `src/constants/anchor.ts:17`，且有显式禁令注释 `:6-9` | ✅ **PASS** |
| **B7** | `anchor_mode=1/4` 在 service 层被真实拒绝，且 Create 与 Update 都调用 | `role_service.go:110`（Create）与 `:169`（Update）均调用 `validateAnchorForWrite`；`post_service.go:90`（Create）与 `:125`（Update）同样。该函数 `role_service.go:549` 内部委托 `datascope.ValidateAnchor`，失败转 `apperrors.BadRequest`（→ 400）。底层 `anchor.go:335-337` 对 `!IsValidAnchorMode` 直接报错，`base.go` 的 `IsValidAnchorMode` 只认 0/2/3/5 | ✅ **PASS** |
| **B8** | operlog 在 role/post/department 写路径被调用 | `grep operlog.Record` 命中 14 处：`department_handler.go:142/175/200/228/262`（Create/Update/Delete/Batch/Status）、`post_handler.go:65/197/225/255`、`role_handler.go:56/159/185/213/247`。逐一核对均在 `response.Success(...)` **之前**的 success path（例：`post_handler.go:64-65`、`role_handler.go:158-159`）。预览端点为只读，正确地**不**记日志 | ✅ **PASS** |
| **B9** | `migration_252` 幂等（`HasColumn` 守卫）+ 非阻断（`runAndRecord` + 只告警） | 迁移文件 `:41/52/58/70/76` 共 5 处 `IF !HasColumn` 守卫，且 `:39/49/67` 有 `HasTable` 前置守卫（缺表时只 `log.Println` 跳过）；`database.go:1518-1521` `runAnchorScopeMigration` 用 `d.runAndRecord(252, ...)`，`err != nil` 时**只** `applogger.Errorf("...非阻断,留待下次启动")`，不返回 error、不中断启动；注册点 `database.go:1492` 在 `runMenuConsolidationMigrations` 内（PG/SQLite 两分支共用）。另有 lead 补充的 `migration_252_*_test.go` 覆盖幂等/缺表/列存在性/默认值 | ✅ **PASS** |

### B 组附带观察（非阻塞）

- **O-1** `internal/models/{dept,role,post}.go` 与 `pkg/middleware/permission.go` 未通过 `gofmt`（`gofmt -l` 会列出）。其中三个 model 文件的**结构体字段对齐被新增字段打乱**（`dept.go` 的 `OrgLevel` 字段注释导致整块 tabwriter 重排），`permission.go` 仅文件末尾多一个空行。**这是格式问题，不影响功能，但会污染后续每次 diff**。建议在收尾时统一跑一次 `gofmt -w`（owner：lead）。
- **O-2** 契约 §五 第 3 条要求 `mode=2` 的 param「非空且为**合法 UUID**」，实现（`anchor.go:344-356`）只校验「非空 + `sys_dept` 中存在」**没有 UUID 正则**。PG 下 `sys_dept.id` 是 uuid 类型，非 UUID 串查不到行 → 行为与契约等价；**SQLite（嵌入/测试）下 `id` 是 TEXT，非 UUID 形状的部门 ID 可以通过**。已由 `anchor_test.go:458` `TestValidateAnchor_DeptModeAcceptsAnyExistingDeptID` 钉死现状。裁定方式见问题 P-4。

---

## 五、边界与漏洞审查 7 项 —— 逐项判定

> 每项给出**明确判定**：安全 / 有风险 / 漏洞。

### C-1 `sanitizeAnchorParam` 与 `ValidateAnchor` 的顺序 → **安全**

**结论：顺序正确，不存在「先 sanitize 把脏参数洗成空串从而通过校验」的问题。**

- `role_service.go`：:**110** 行 `validateAnchorForWrite(...)`（validate）→ **:122** 行 `sanitizeAnchorParam(...)`（sanitize）。**validate 在前**。
- `post_service.go`：**:90** validate → **:99** sanitize。**同序**。
- `Update` 路径同序（`role_service.go:169` → `:181`；`post_service.go:125` → `:133`）。

因此 `mode=3` 传了脏 `param`（如 `"2"`）时，`ValidateAnchor`（`anchor.go:358-361`）会先命中 `strings.TrimSpace(param) != ""` 分支并直接报错「锚点模式为「外部机构」时不需要参数，请清空锚点参数」，**不会**被 sanitize 掩盖。契约 §五 第 4 条「mode=3 时 param 必须为空」被真实执行。

sanitize 只在**校验通过之后**做归一化（mode=0/3 恒置空串），其作用是把「用户从 mode=2 切到 mode=3 后前端残留的旧 param」清掉，属于防脏数据，不承担校验职责。**职责分离清晰，判定安全。**

### C-2 `queryAnchorGrants` 只看 `status=0`，预览看全部 → **安全（合规，但需知会管理员）**

- 生效面：`anchor.go:413-474` `queryAnchorGrants` 两条 SQL 均带 `r.status = ? / p.status = ?`（`RoleStatusEnabled`/`PostStatusEnabled`）+ `deleted_at IS NULL`，与 `getUserMaxDataScope`（`permission.go:563-568` 只计 `r.status = 0`）口径**一致**。
- 预览面：`role_service.go:626` `listRoleAssigneeIDs` 与 `post_service.go` `listPostAssigneeIDs` **不筛 status**。
- **契约 §六 明文要求**：「取该角色/岗位的**全部任职人**（`sys_user_role`/`sys_user_post`，不看 status）」。预览是「配置面体检」，回答的是「我这个锚点配置对挂在它下面的所有人是否都能算出来」，而非「当前谁真的生效」。两个口径**各自符合其契约条文**，实现也是按条文写的。
- **残余风险（判定为"安全但需知会"，非缺陷）**：管理员看预览看到「12 人全部已解析」，其中若有 3 人的角色已停用，这 3 人实际不产生任何范围。预览面板的 `Alert description` 已写「这些人不会获得机构范围，将按原有角色范围生效」，但**未提示"停用角色的任职人不在生效范围内"**。建议前端在预览面板加一行静态说明（前端 owner）。不影响安全，只影响管理员判断精度。

### C-3 `isMissingTableError` 的降级 → **有风险（静默失败，零日志）**

**结论：降级本身是合理的（C2 要求新增维度绝不收窄），但「吞错且不记日志」在生产不可接受。**

- `pkg/datascope/anchor.go:459-465`：
  ```go
  if err := db...Error; err != nil {
      if !isMissingTableError(err) { return nil, fmt.Errorf("查询岗位锚点来源失败: %w", err) }
      else { postRows = nil }        // ← 静默吞掉，无任何日志
  }
  ```
- 中间件的降级日志（`permission.go:547-550`）只在 `ResolveAnchorGrants` **返回 error** 时打印。而缺表分支**永不返回 error**，所以这条路径**永远不会有任何日志**。
- 后果：生产上若 `sys_user_post` 因历史原因缺失，所有岗位维度的机构授权**全量失效**，用户「配了但没有效果」，运维**没有任何线索**。契约 §四 明确要求「锚点解析失败…**必须告警**」，这条路径违背了该要求的精神。
- 对照证据：`internal/services/system/user_service.go` 的同名判断 `isMissingTableErr`（`:647-655`）至少有 `applogger.Warnf("回填用户岗位失败...")`。**同一类降级，锚点侧（安全相关）反而没有告警。**
- **建议**：在 `anchor.go:462` 分支内加 `applogger.Warnf("sys_user_post 缺失，岗位维度机构授权已静默失效: %v", err)`。改动 2 行，owner：lead。

### C-4 「Self 用户 + 锚点 → 变成机构范围」是否越权 → **有风险（规则合规，风险在授权环节）**

**结论：`Merge` 规则 3 的行为是契约明文授权的，本身不构成越权；真正的风险在于「谁能给谁挂带锚点的岗位」，而这与下面的 C-5 是同一根链条。合并判定见 C-5。**

- 契约 §三 3.4 规则 3 明文：「⚠️ 落此分支时，若 `roleScope.Hook == HookSelfUser`，**以机构维度胜出**（显式授予的更大范围覆盖"仅本人"）」。实现 `anchor.go:548-558` 严格按此执行，且有测试锁定（`TestMerge_SelfOverriddenByAnchor`）。
- 语义上这是**有意设计**：一个「仅本人」的普通员工被赋予「中支管理员」岗位后，应当获得中支范围，否则该岗位毫无意义。
- **但**：该规则的成立前提是「岗位的授予是可信的、经过授权的」。若任何人都能自行把岗位挂到自己身上（`user_service.go` 零校验，见 C-5），则「Self → 机构范围」就从"设计特性"变成了"自助提权的一条跳板"。
- 另附一条我补充锁定的分支（开发/契约均未覆盖）：**`FailClosed` 用户 + 有效锚点 → 也会走规则 3 被放宽为 `FailClosed=false` + 锚点子树**（`anchor_test.go:409` `TestMerge_Rule2_FailClosedWithAnchorWidens`，当前 PASS）。即「脏 `data_scope` 枚举」原本的兜底全空**被锚点覆盖掉了**。这是 C2「宽者胜」的必然推论，但在安全评审上属于「fail-closed 被降级」，请 lead 显式裁定接受或调整规则优先级。
- **前置修复依赖**：C-5 的 P0 措施（岗位挂载需 `system:post:edit`）落地后，本条风险降为「可控」。**判定：有风险，非独立漏洞。**

### C-5 `anchor_mode=5, target=1`（分公司管理员 = 全省可见）是否有约束 → **漏洞（权限提升）**

**结论：这是一个可被利用的权限提升漏洞，不是"建议"。**

**根因有两层，叠加后闭环：**

**① 授权面零约束** —— `internal/services/system/user_service.go:132-143`（Create）与 `:252-263`（Update）：
```go
if len(req.PostIds) > 0 {
    userPosts := make([]models.UserPost, len(req.PostIds))
    for i, postID := range req.PostIds { userPosts[i] = models.UserPost{UserID: ..., PostID: postID} }
    if err := tx.Create(&userPosts).Error; ...     // ← 零校验直接落库
}
```
**不校验岗位是否存在、不校验 `post.status`、不校验该岗位 `anchor_mode != 0`、无二次确认、无告警**。任何持有 `system:user:edit` 的人（路由 `internal/api/router.go:331`，`DataScopePermission` 只约束数据面**不约束岗位字典面**）都可以在自己的用户编辑表单里提交任意 `postIds`。

**② 解析面自动放大** —— `pkg/datascope/anchor.go:451-471`：岗位维度的锚点按**任职人自己的部门链**解析。`anchor_mode=5, param="1"` 的岗位，对任何能上溯到分公司的人都会解析成「湖北分公司」→ `queryDeptChildIDs` 展开**全子树**（全省所有部门）。

**完整攻击链（全部在代码上可验证，非理论）：**
1. 省管理员按本功能的**正常业务用法**创建岗位「分公司管理员」`anchor_mode=5, anchor_param="1"`（这正是契约 §二 2.2 的目标场景，生产中必然存在）；
2. 任一持 `system:user:edit` 的中支/部门管理员，在自己的用户编辑表单里把自己挂到该岗位；
3. 下一次请求，`DataScopePermission` → `ResolveAnchorGrants` 命中「湖北分公司」→ 展开全子树 → `ScopeResult.DeptIDs` = 全省；
4. 该用户对 ops / vdi / users 全域**从「只看本部门」跃升为「全省可见」**，且**没有任何审计痕迹指向这次提权**（operlog 只记「用户管理-修改」）。

**这同时坐实了 C-4 的实际危险性**：C-4 的「Self → 机构范围」正是这条链的第 3 步。

**建议修复（owner：backend-crud + lead，我不改实现）：**
- **P0（必须，最小改动）**：`user_service.go` 两处 PostIds 写入前逐个校验：`sys_post` 存在且 `status=0`；若目标岗位 `anchor_mode <> 0`，要求操作者具备 `system:post:edit`（否则 400 拒绝），并额外写一条 operlog（或至少 `applogger.Warnf` 记录"谁把哪个锚点岗位授给了谁"）。
- **P1**：`post_service.Create/Update` 对 `anchor_mode=5/param="1"` 这类最高层级授权落审计记录（谁在何时创建了"分公司级"授权岗位）。
- **P2**：用户编辑页对被分配岗位展示「此岗位携带机构授权：湖北分公司（含全部下级）」并二次确认（可复用已有的 `AnchorScopeFields` / `anchorModeLabel`）。
- **P3（可选加固）**：`Merge` 规则 2 改为「`FailClosed` 优先于锚点」，避免脏枚举被锚点放宽（对应 C-4 的补充发现）。

### C-6 预览接口 `unresolved` 与 `items` 双字段是否一致 → **安全**

**后端一致**：`role_service.go:640` `buildAnchorPreview` 中，`Items` 直接取 `preview.Items`（全量），`Unresolved` 由**同一份 `preview.Items` 过滤 `!item.Resolved`** 生成（`role_service.go` 内 `for _, item := range preview.Items { if !item.Resolved { resp.Unresolved = append(...) } }`）。**同源同生，不可能不一致**。`mode=0` 时先改写 `preview.Items[i].Reason`，再生成 `resp` 与 `Unresolved`，故两处文案也一致。

**前端一致**：`src/lib/systemAnchorApi.ts` 的 `normalizeAnchorPreview` 优先用 `items` 过滤 `!resolved`，`unresolved` 仅当 `items` 缺失时兜底；`resolvedCount`/`unresolvedCount`/`totalCount` 若后端已给则直接采用（后端恒给）。`AnchorPreviewPanel.tsx` 渲染表格用 `result?.unresolved ?? []`，`hasUnresolved` 同时看 `unresolvedCount` 与 `unresolved.length`（双条件，不会因字段缺失而漏提示）。

**唯一观察**：`AnchorPreviewResponse` 同时给两个字段是**冗余**（`items` 已足够推导 `unresolved`），契约 §六 原文举例只写 `unresolved[]`。冗余不影响正确性，属设计取舍，已在 `role_service.go` 注释中说明。**判定：安全。**

### C-7 前端 `AnchorScopeFields` 切模式是否清 param（2→5 与 5→2 双向）→ **安全**

**结论：双向都会正确清理，且是单元级可验证的纯函数。**

- `AnchorScopeFields.tsx:62-65` `handleModeChange` 调 `onChange({ mode: nextMode, param: retargetAnchorParam(nextMode, currentParam) })` —— **成对抛出**，父级不会看到「mode 已变、param 未清」的中间态。
- `src/constants/anchor.ts:126-130` `retargetAnchorParam`：
  ```ts
  if (nextMode === 2) return isAnchorDeptParam(prevParam) ? prevParam : "";   // 非 UUID → 清空
  if (nextMode === 5) return isAnchorLevelParam(prevParam) ? prevParam : "";  // 非 "1"~"4" → 清空
  return "";                                                                  // mode 0 / 3 → 恒清空
  ```
- **2→5**：残留的是部门 UUID → `isAnchorLevelParam(UUID)` 为 false → **清空**。✅
- **5→2**：残留的是 `"2"` 之类 → `isAnchorDeptParam("2")` 为 false（UUID 正则不匹配）→ **清空**。✅
- **→0 / →3**：`return ""` → **恒清空**。✅
- 持久化侧二次兜底：`role_service.go:562` `sanitizeAnchorParam` 对 mode 0/3 恒返回空串，即使前端漏清也不会入库脏参数。
- 父级三处调用点均复用同一函数：`role/index.tsx` `handleAnchorChange`、`post/index.tsx:159-166` `handleAnchorChange`、`useRoleActions.ts` 提交时。**无一处绕过**。
- 提交侧再兜底：`useRoleActions.ts` 提交前 `retargetAnchorParam(normalizeAnchorMode(anchorMode ?? values.anchorMode), ...)`。

**判定：安全。**

---

## 六、问题清单

### 阻塞（上线前必须处理或由 owner 书面接受）

| ID | 级别 | 位置 | 问题 | Owner |
|---|---|---|---|---|
| **P-1** | 🔴 **阻塞（漏洞）** | `internal/services/system/user_service.go:132-143`（Create）、`:252-263`（Update） | `PostIds` 写入零校验；叠加锚点岗位的动态解析 → 低权限者可自助提权至全省可见（详见 C-5） | backend-crud + lead |
| **P-2** | 🟠 **阻塞（静默失败）** | `pkg/datascope/anchor.go:459-465` | `sys_user_post` 缺表时静默吞错、零日志；岗位维度授权全量失效且运维无感知，违背契约 §四「必须告警」 | lead（2 行改动） |

### 建议

| ID | 级别 | 位置 | 问题 | Owner |
|---|---|---|---|---|
| **P-3** | 🟡 建议（契约措辞） | 契约 §九 用例 B | 「param="3" → 未命中 → FailClosed 空集」表述不精确：锚点未命中只让**锚点维度**为空；若用户还有 Dept/Custom 角色，`Merge` 保留角色维度**且不 FailClosed**（否则违反 C2 纯增量）。真值表已由 `anchor_test.go:364` 锁定 | lead（改契约文案） |
| **P-4** | 🟡 建议 | 契约 §五 第 3 条 vs `anchor.go:344-356` | 契约要求 mode=2 的 param 是「合法 UUID」，实现未做 UUID 正则，只校验部门存在。PG 等价、SQLite 不等价。需裁定「补正则」或「改契约」 | lead |
| **P-5** | 🟡 建议 | `internal/models/{dept,role,post}.go`、`pkg/middleware/permission.go` | 未 `gofmt`（新增字段打乱 tabwriter 对齐）；不影响功能但持续污染后续 diff | lead |
| **P-6** | 🟡 建议 | `pkg/datascope/anchor.go:537-567` `Merge` 规则 2 | `FailClosed`（脏 `data_scope`）+ 有效锚点 → 被放宽为 `FailClosed=false` + 锚点子树。属 C2 推论，但「fail-closed 被降级」值得显式裁定 | lead |
| **P-7** | 🟡 建议 | `internal/services/system/user_service.go:351` `GetByID` | 只 `fillUserRoles`，**不** `fillUserPosts`；当前前端编辑走列表行数据（列表已 `fillUserPosts`，`user_service.go:554`）故不影响，但任何走详情接口编辑的路径会丢岗位。建议 GetByID 也调用 `fillUserPosts` | backend-crud |
| **P-8** | 🟡 建议 | `xingran-react-frontend/src/components/system/AnchorPreviewPanel.tsx` | 预览面板未说明「停用角色的任职人不在生效范围内」。预览口径（全量任职人）与生效口径（仅 status=0）的差异需静态提示（判定见 C-2） | frontend |

---

## 七、未解决的问题 + 建议后续动作

1. **P-1 的修复粒度需 lead 决策**：是「阻止非 `system:post:edit` 者挂载锚点岗位」（最小），还是「岗位挂载一律需二次确认 + 审计」（更彻底）。前者改动 2 个函数即可，后者涉及前端。建议**先落 P0 再上线**，P1/P2 可排后续迭代。
2. **P-2 与 P-1 应同一批次修**：两者都属「锚点维度静默失效/静默放大」，一起修便于一次性回归。
3. **契约 §九 用例 B 的措辞（P-3）建议本次就改**：否则下一个审计者会重复遇到「按契约字面写测试会失败」的困惑。我已把真值表以测试形式固化。
4. **契约 §五 UUID 校验（P-4）**：若生产只有 PG，可判定为「契约文本过严」，改契约即可；若存在 SQLite 部署（本项目 `internal/core/db` 支持双驱动），则建议补正则（`anchor.go:345` 后加一处 UUID 校验，5 行）。
5. **本报告未覆盖的项**（透明声明）：
   - 未做真实 PG 环境的集成/E2E 验证（仅 SQLite 夹具 + 代码审查）。
   - 未做前端浏览器实机点击验证（仅 type-check / lint / 代码审查）。若需 UAT 级确认，建议由 lead 或用户走一次真实页面流程。
   - 未审计 `anchor_param` 在 `sys_role`/`sys_post` 上的**历史脏数据**（如存量 `anchor_mode=1/4` 的行）。运行期 `ResolveAnchor` 对非法 mode 返回 `AnchorReasonBadMode` 且不解析（`anchor.go:244-246`），**不会**导致越权或崩溃，但会在预览里显示"锚点模式非法"。建议上线后跑一次 `SELECT * FROM sys_role WHERE anchor_mode NOT IN (0,2,3,5)` 巡检。

---

## 八、核验者签收

- 我**没有**修改任何实现代码（`git status` 中我触碰的文件仅 `pkg/datascope/anchor_test.go`、`pkg/middleware/anchor_mount_test.go`、本报告）。
- 本报告每条结论均附「文件:行号」或「测试函数名」级证据，无「看起来没问题」式结论。
- **签名含义**：C1~C7 全部合规、六条用例全部通过 = 可以进入验收；但 **P-1 / P-2 未关闭前不建议上线**。
