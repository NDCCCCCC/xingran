---
phase: quick-261019-mmc
plan: 01
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/services/addomain/ldap_client.go
  - internal/services/addomain/ldap_client_rfc4514_test.go
  - internal/services/addomain/ldap_client_78_07_test.go
autonomous: true
requirements:
  - AD-RDN-RFC4514
must_haves:
  truths:
    - "extractRDNFromDN 对 `CN=Zhang\\,San,OU=IT,DC=corp,DC=com` 返回完整的 `CN=Zhang\\,San`，而不是截断的 `CN=Zhang\\`"
    - "逗号仅在前面是**偶数**个连续反斜杠（即未转义）时才作为分隔符；奇数个反斜杠后的逗号是值的一部分"
    - "对不含转义逗号的输入（普通多段 DN、单 RDN 无逗号、空串、含空格 DN）行为与修改前逐字节一致"
    - "手写拆分器与 go-ldap 自带的 RFC4514 解析器 ldap.ParseDN 在整个测试表上判定一致（第一个 RDN 语义相等，且截断输出恰好只含 1 个 RDN）"
    - "MoveUser 传给 ldap.NewModifyDNRequest 的 newRDN 是完整 RDN，不再是截断串"
    - "Phase 58 / Phase 57 回归测试全绿，go build ./... 与 go vet ./internal/services/addomain/ 无错"
  artifacts:
    - "internal/services/addomain/ldap_client.go（修正后的 extractRDNFromDN，行为未转义逗号时不变）"
    - "internal/services/addomain/ldap_client_rfc4514_test.go（新测试，含 ldap.ParseDN 交叉校验）"
    - "internal/services/addomain/ldap_client_78_07_test.go（TestLC78_ExtractRDNFromDN_Edge 的转义逗号期望值更新）"
  key_links:
    - "ldap_client.go:353 MoveUser → ldap.NewModifyDNRequest(userDN, rdn, true, newOUDN)（deleteOldRDN=true，数据破坏点；不改调用点语义）"
    - "ldap_client.go:429 extractRDNFromDN（被 ad_push_queue_drain_service.go:523、user_ad_sync_service.go:139/:332 复用；改函数即改四处，调用点一行不动）"
    - "dn_util.go:63 BuildOUDN 用 ldap.EscapeDN 做 OU 侧转义（Phase 58 已有的先例；本次只补 RDN 侧，NormalizeDN 刻意不用 ParseDN 的理由不适用于本函数）"
---

<objective>
修复 `extractRDNFromDN` 在 RFC4514 转义逗号处截断的缺陷（Phase 58 遗留 MISSED-04，数据破坏级），并用与 go-ldap 官方解析器交叉校验的测试表锁定语义。

Purpose: `MoveUser` 把截断的 RDN 传给 `ldap.NewModifyDNRequest(userDN, rdn, true, newOUDN)`，`deleteOldRDN=true` 会让 AD 把对象重命名成 `CN=Zhang\` 再删掉旧 RDN —— 名字里的逗号后缀永久丢失。Phase 58 已给 OU 侧加了 `ldap.EscapeDN`，RDN 侧却仍在按字面逗号切，且被挂上了 30 秒自动排水路径。
Output: 一个行为正确、有交叉校验测试覆盖的 `extractRDNFromDN`；四个调用点一行未改。
</objective>

<execution_context>
@~/.claude/gsd-core/workflows/execute-plan.md
@~/.claude/gsd-core/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md

# 只读参考（不改）
internal/services/addomain/dn_util.go            # BuildOUDN 的 ldap.EscapeDN 先例
internal/services/addomain/ad_push_queue_drain_service.go:523   # 调用点 2（Phase 58）
internal/services/addomain/user_ad_sync_service.go:139,332       # 调用点 3/4
internal/services/addomain/ldap_client_78_07_test.go:200-236     # 锁定了旧（错误）行为的用例

# 缺陷来源
.planning/phases/58-compare-and-move/58-GOAL-VERIFICATION.md   §6 MISSED-04
.planning/phases/58-compare-and-move/58-REVIEW.md              "RDN-side asymmetry"
</context>

<tasks>

<task type="auto" tdd="true">
  <name>Task 1: 修正 extractRDNFromDN 为"首个未转义逗号"切分</name>
  <files>internal/services/addomain/ldap_client.go</files>
  <behavior>
    - `extractRDNFromDN("CN=Zhang\\,San,OU=IT,DC=corp,DC=com")` → `CN=Zhang\\,San`
    - `extractRDNFromDN("CN=John,OU=Users,DC=example,DC=com")` → `CN=John`（不变）
    - `extractRDNFromDN("CN=John")` → `CN=John`（不变）
    - `extractRDNFromDN("")` → `""`（不变）
    - `extractRDNFromDN("CN=a\\\\,b")` → `CN=a\\\\`（两反斜杠 = 转义的反斜杠，后面的逗号是**未转义**分隔符）
    - `extractRDNFromDN("CN=ends-with-backslash\\\\")` → 原样返回（无未转义逗号）
    - 反斜杠扫描必须按字节（RFC4514 转义字符全是 ASCII），中文等多字节值不得被误判
  </behavior>
  <action>
重写 `internal/services/addomain/ldap_client.go` 中的 `extractRDNFromDN`（当前 :429-436 用 `strings.Index(dn, ",")` 无条件切第一个逗号）。

实现要求 —— 单趟字节扫描，**不新增 import**（`strings` 已在用，保持现状即可）：

1. 从左到右遍历 `dn` 的每个字节索引 `i`。
2. 维护一个 `escaping bool` 状态位：遇 `\`（0x5C）置 `true` 并 `continue`；任何其它字符若 `escaping` 为 `true` 则清 `false` 并 `continue`（即被转义的字符一律不是分隔符）。
3. 当 `escaping` 为 `false` 且字节是 `,`（0x2C）时，返回 `dn[:i]`。
4. 遍历结束没遇到未转义逗号则返回 `dn` 本身（保留 `Index == -1` 的旧行为）。

⚠ 关于"奇偶反斜杠"：上面的 `escaping` 状态机**天然**实现奇偶语义 —— 两个连续反斜杠是 `\` + "被转义的反斜杠"，状态位回到 false，所以其后的逗号被视为未转义。不需要单独数反斜杠个数。

⚠ 严格按字节（byte）遍历，**不要**用 `range dn`（那会按 rune 迭代、对多字节 UTF-8 续字节做判断）或任何 `unicode`/`utf8` 逻辑 —— RFC4514 的转义字符与分隔符都是 ASCII，AD 里的中文 CN 值靠"非 ASCII 字节 ≠ ','"自动安全。

⚠ 函数签名、注释位置、包名保持不变。调用点（ldap_client.go:353、ad_push_queue_drain_service.go:523、user_ad_sync_service.go:139/:332）**一行都不许改** —— 它们的语义本意就是要完整 RDN，是被这个函数截断的。

在函数注释里补一段说明：逗号仅在**未转义**（前面是偶数个连续反斜杠）时才是 RDN 分隔符；Phase 58 只给 OU 侧加了 `ldap.EscapeDN`（见 `dn_util.go` `BuildOUDN`），RDN 侧此前按字面逗号切会截断 `CN=Zhang\,San` 这类值，进而让 `MoveUser` 的 `ModifyDN(deleteOldRDN=true)` 把对象改名成截断串。

**不要**改 `NormalizeDN`（它刻意不用 `ldap.ParseDN`，因为存量 `sys_dept_ou_mapping.ou_dn` 含未转义特殊字符，`ParseDN` 会对这些失败）。`extractRDNFromDN` 的输入是**已在库里的 AD 侧 DN**，本就是合法转义形态，所以这里的期望语义与 `NormalizeDN` 不同，不要顺手"统一"。
  </action>
  <verify>
    <automated>go build ./... &amp;&amp; go vet ./internal/services/addomain/</automated>
  </verify>
  <done>extractRDNFromDN 按首个未转义逗号切分；四个调用点源码零改动（`grep -c extractRDNFromDN` 结果与改前一致：ldap_client.go 2 处含注释、drain 2 处含注释、user_ad_sync 3 处含注释）；go build ./... 与 go vet 均通过。</done>
</task>

<task type="auto">
  <name>Task 2: 新增 RFC4514 测试表 + ldap.ParseDN 交叉校验</name>
  <files>internal/services/addomain/ldap_client_rfc4514_test.go</files>
  <action>
新建 `internal/services/addomain/ldap_client_rfc4514_test.go`（**不要**加 build tag —— 这是纯函数测试，与 `dn_util_test.go` 同款：不连 DB、不引用 fixture）。包名 `package addomain`（测未导出函数）。imports：`testing`、`github.com/go-ldap/ldap/v3`、`github.com/stretchr/testify/assert`。

**命名**：所有用例前缀 `TestPhase58_`，这样 `go test ./internal/services/addomain/ -run TestPhase58 -count=1` 一把捞起（本 quick 任务的修复面在 Phase 58 遗留，沿用同一前缀便于以后一并在 Phase 58 的回归命令里守住）。

建议拆成两个测试函数：

**A. `TestPhase58_ExtractRDNFromDN_RFC4514Table`** —— 表驱动，字段 `{name, dn, want}`，用 `assert.Equal(t, tt.want, extractRDNFromDN(tt.dn))` 断言。必须覆盖的用例（Go 字面量用反引号 raw string，`\\` 表示**一个**反斜杠字节）：

| name | dn (raw) | want (raw) |
|---|---|---|
| 转义逗号-中文姓名 | `CN=Zhang\,San,OU=IT,DC=corp,DC=com` | `CN=Zhang\,San` |
| 转义逗号-英文姓 | `CN=John\,Doe,OU=Users,DC=example,DC=com` | `CN=John\,Doe` |
| 转义逗号后仍有多段 | `CN=a\,b\,c,OU=IT,DC=corp,DC=com` | `CN=a\,b\,c` |
| 多值 RDN（+ 连接） | `CN=x+SN=y,OU=IT,DC=corp,DC=com` | `CN=x+SN=y` |
| 多值 RDN 带转义逗号 | `CN=Zhang\,San+SN=Z,OU=IT,DC=corp,DC=com` | `CN=Zhang\,San+SN=Z` |
| 普通多段（回归不变） | `CN=John,OU=Users,DC=example,DC=com` | `CN=John` |
| 多层 OU（回归不变） | `CN=lisi,OU=SubDept,OU=ParentDept,DC=company,DC=com` | `CN=lisi` |
| 单 RDN 无逗号（回归不变） | `CN=John` | `CN=John` |
| 空串（回归不变） | `` | `` |
| 含空格（回归不变） | `CN=zhang san,OU=Test Dept,DC=company,DC=com` | `CN=zhang san` |
| 中文 CN（回归不变） | `CN=张三,OU=研发部,DC=corp,DC=com` | `CN=张三` |
| 字面反斜杠 + 未转义逗号 | `CN=a\\,OU=IT,DC=corp` | `CN=a\\` |
| 字面反斜杠且后半段无 `=` | `CN=a\\,b` | `CN=a\\` |
| 值以反斜杠结尾（无逗号） | `CN=ends-with-backslash\\` | 原样 `CN=ends-with-backslash\\` |
| 值以**转义反斜杠**结尾且后接分隔逗号 | `CN=trailing\\,OU=IT,DC=corp` | `CN=trailing\\` |
| 转义逗号出现在末段 | `CN=x\,y,DC=corp,DC=com` | `CN=x\,y` |
| 十六进制转义值 | `CN=\41\,B,OU=IT,DC=corp` | `CN=\41\,B` |

表下再补一条显式断言：转义逗号用例下 `extractRDNFromDN(dn)` 的结果**以 dn 自身为前缀**且长度**大于**旧实现会返回的长度（即不再被截断）—— 或更直接：断言 `assert.NotEqual(t, "CN=Zhang\\", got)`。

**B. `TestPhase58_ExtractRDNFromDN_AgreesWithGoLDAP`** —— 交叉校验（**本任务的核心价值，不要只写硬编码期望**）。遍历 A 表里的全部 `dn`：

1. `parsed, err := ldap.ParseDN(tt.dn)`；`require.NoError(t, err)`。
2. `got := extractRDNFromDN(tt.dn)`。
3. `gotParsed, err := ldap.ParseDN(got)`；`require.NoError(t, err)`。
4. `require.Len(t, gotParsed.RDNs, 1, "截断输出必须恰好只含 1 个 RDN")` —— 这一条钉死"不能多带后面的段"。
5. `assert.True(t, parsed.RDNs[0].Equal(gotParsed.RDNs[0]), "手写拆分器的第一个 RDN 必须与 go-ldap RFC4514 解析结果语义相等")`（用 `Equal` 而不是比字符串：go-ldap 会把 attr 名小写、多值 RDN 排序，无法直接比字面量）。

⚠ **已知例外，必须显式处理**（不要让 `require.NoError` 直接红）：`CN=a\\,b` 这一条 `ldap.ParseDN` 会对**整串**返回 `DN ended with incomplete type, value pair` —— 因为按 RFC4514 逗号确为未转义分隔符，后半段 `b` 是个没有 `=` 的残段，是非法 DN。这不是我们实现的问题，是输入本身非法。所以给表加一个 `parseWholeDN bool` 字段（或 `skipCrossCheck bool`）：默认 `true`，仅 `CN=a\\,b` 置 `false`。该条仍由 A 表的硬编码期望守住；B 表对它 `t.Skip` 并附一句注释说明为什么。

同理，空串一条 `ldap.ParseDN("")` 返回 `len(RDNs)==0`（不报错），第 4/5 步无从比较 —— 空串也要走 `skipCrossCheck`（或 B 表里显式 `if tt.dn == "" { t.Skip(...) }`），只由 A 表的 `""` 期望守住。

这样一来：任何**手写拆分器与 go-ldap 判定分歧**的未来改动（多漏一个转义、反斜杠奇偶搞错、按 rune 迭代引入偏移错位）都会让 B 表红，而不只是让 A 表某个字面量红。
  </action>
  <verify>
    <automated>go test ./internal/services/addomain/ -run TestPhase58_ExtractRDNFromDN -count=1 -v</automated>
  </verify>
  <done>新测试文件存在且无 build tag；A 表 16 条全部 PASS（含转义逗号、多值 RDN、字面反斜杠奇偶、以反斜杠结尾、十六进制转义、中文）；B 表对每条 `skipCrossCheck=false` 的用例都验证了"截断输出恰好 1 个 RDN"+"与 go-ldap 第一个 RDN 语义相等"；2 条已知例外（`CN=a\\,b`、空串）以带注释的 skip 显式豁免而非静默通过。</done>
</task>

<task type="auto">
  <name>Task 3: 更新锁定旧（错误）行为的存量用例 + 全量回归</name>
  <files>internal/services/addomain/ldap_client_78_07_test.go</files>
  <action>
`internal/services/addomain/ldap_client_78_07_test.go` 的 `TestLC78_ExtractRDNFromDN_Edge`（约 :202-236）里有一条用例把**缺陷行为**写成了期望，必须改：

```go
{
    name: "转义逗号", // extractRDNFromDN 不处理转义,只找第一个逗号字面值
    dn:   "CN=John\\,Doe,OU=Users,DC=example,DC=com",
    want: "CN=John\\", // 第一个逗号在 "Doe" 前,返回 "CN=John\\"
},
```

改为期望完整 RDN，并把注释从"记录缺陷"翻成"锁定修复"：

```go
{
    // Phase 58 MISSED-04 / 本 quick 261019-mmc：逗号仅在**未转义**时才是分隔符。
    // 旧实现返回 "CN=John\"（截断），会让 MoveUser 的 ModifyDN(deleteOldRDN=true)
    // 把 AD 对象改名成截断串并删除旧 RDN。
    name: "转义逗号",
    dn:   "CN=John\\,Doe,OU=Users,DC=example,DC=com",
    want: "CN=John\\,Doe",
},
```

同文件其余三条（普通 DN / 无逗号单段 / 空串）期望值不变，保持原样。文件顶部的 `//go:build !skip_db_tests` tag 与文件头注释块保持不动（只改这一条用例；如果顶部注释块里列了"extractRDNFromDN 边界覆盖(已有 100%,补边界用例)"这类描述，无需改）。

同时检查 `internal/services/addomain/ldap_client_test.go` 的 `TestExtractRDN` / `TestExtractRDN_EdgeCases` —— 已核对过，它们的用例全都不含转义逗号（标准 DN / 多层 OU / 单段 / 空串 / 含空格），期望值不受影响，**不要动**。

最后跑全量回归（见 `<verification>`）。确认以下都绿：
- `go build ./...`
- `go vet ./internal/services/addomain/`
- `go test ./internal/services/addomain/ -run TestPhase58 -count=1`（Phase 58 的 drain / dn_util / config_resolver 用例 + 本次新增用例）
- `go test ./internal/services/system/ -run TestPhase57 -count=1`（outbox-lite 队列模型 / partial unique index）
- `go test ./internal/services/addomain/ -run 'TestLC78_ExtractRDNFromDN_Edge|TestExtractRDN' -count=1`
- `go test ./internal/services/addomain/ -count=1`（包内全量）
  </action>
  <verify>
    <automated>go build ./... &amp;&amp; go vet ./internal/services/addomain/ &amp;&amp; go test ./internal/services/addomain/ -count=1 &amp;&amp; go test ./internal/services/system/ -run TestPhase57 -count=1</automated>
  </verify>
  <done>`TestLC78_ExtractRDNFromDN_Edge` 的转义逗号期望值为 `CN=John\\,Doe`；该文件除此一条外零改动；ldap_client_test.go 零改动；上述全部回归命令通过（addomain 包全量 + system 的 TestPhase57 + build + vet 全绿，无 FAIL、无 build 错误）。</done>
</task>

</tasks>

<threat_model>
## Trust Boundaries

| Boundary | Description |
|---|---|
| AD 目录 → `extractRDNFromDN` | 输入是域控返回/存储的 DN 字符串。RFC4514 允许值内含 `\,` `\+` `\\` 等转义序列；不可信（可含用户可编辑的姓名，如 `CN=Zhang\,San`）。 |
| `extractRDNFromDN` → `ldap.NewModifyDNRequest(deleteOldRDN=true)` | 本次修复的数据完整性红线：错误的 RDN 会让 AD 重命名对象并删除旧 RDN，不可逆。 |

## STRIDE Threat Register

| Threat ID | Category | Component | Severity | Disposition | Mitigation Plan |
|---|---|---|---|---|---|
| T-Q261019-01 | Tampering | `extractRDNFromDN` 截断 RDN → AD 对象被改名/旧 RDN 被删 | critical | mitigate | Task 1 按首个**未转义**逗号切分（转义状态机，奇偶反斜杠天然正确）；Task 2 用 `ldap.ParseDN` 交叉校验表钉死语义（输出恰 1 个 RDN + 与库解析结果 `Equal`），任何回归立即红。 |
| T-Q261019-02 | Tampering | 值内含字面反斜杠时误判逗号转义（偶/奇混淆） | medium | mitigate | Task 2 表内含 `CN=a\\,OU=IT,...`（偶数→未转义分隔）与 `CN=a\\` / `CN=trailing\\,OU=IT,...`（边界）用例，并全部走 `ldap.ParseDN` 交叉校验。 |
| T-Q261019-03 | Tampering | 按 rune 而非 byte 遍历致多字节 UTF-8 值（中文 CN）切分错位 | medium | mitigate | Task 1 强制字节扫描（禁 `range`/`utf8`）；Task 2 表含 `CN=张三,OU=研发部,...` 且走交叉校验（`ParseDN` 对错位输出会报错或 RDN 不等）。 |
| T-Q261019-04 | Information Disclosure | 无 —— 本次纯本地字符串切分，不引入外部输入面/网络调用/凭据 | low | accept | 不新增任何 IO、依赖或配置面；`ldap.EscapeDN`（`dn_util.go`）已在本包，`ldap.ParseDN` 同库，go.mod 无变化。 |

**不在本次范围**（Phase 58 已裁决，显式不动）：时间戳仲裁、新旧值比较、排水器注册 `sys_job`。四个调用点的语义、行号、逻辑一律不改。
</threat_model>

<verification>
```
go build ./...
go vet ./internal/services/addomain/
go test ./internal/services/addomain/ -run TestPhase58 -count=1
go test ./internal/services/system/  -run TestPhase57 -count=1
go test ./internal/services/addomain/ -run 'TestLC78_ExtractRDNFromDN_Edge|TestExtractRDN' -count=1
go test ./internal/services/addomain/ -count=1
```

附加人工检查（一条命令）：`git diff --stat` 的改动文件必须**恰好**是 frontmatter `files_modified` 里的三个（`ldap_client.go` / `ldap_client_rfc4514_test.go` / `ldap_client_78_07_test.go`）。任何第四个文件出现都意味着发生了越界重构。
</verification>

<success_criteria>
1. `extractRDNFromDN` 在首个未转义逗号处切分；转义逗号输入返回完整 RDN。
2. 新测试文件含 16 条硬编码表用例 + 对 go-ldap `ldap.ParseDN` 的交叉校验（输出恰 1 个 RDN 且首 RDN 语义相等），2 条已知非法输入显式带注释豁免。
3. 四个调用点（ldap_client.go:353、ad_push_queue_drain_service.go:523、user_ad_sync_service.go:139/:332）零改动。
4. `go build ./...`、`go vet ./internal/services/addomain/`、`addomain` 包全量测试、`TestPhase58`、`TestPhase57` 全绿。
5. `go.mod` 无变化；未触及 Phase 58 三条审计否决项。
</success_criteria>

<output>
完成后在 `.planning/quick/261019-mmc-extractrdnfromdn-rfc4514-modifydn-ad-pha/` 下创建 `261019-mmc-SUMMARY.md`。
</output>
