---
phase: quick-261019-mmc
plan: 01
status: complete
subsystem: addomain
tags: [ldap, rfc4514, modifydn, phase58-missed-04, bugfix]
requires: [AD-RDN-RFC4514]
provides: [RFC4514-correct-extractRDNFromDN, go-ldap-cross-checked-test-table]
affects: [addomain, phase58-drain-path]
tech-stack:
  added: []
  patterns: [escaping-state-machine-byte-scan, go-ldap-ParseDN-cross-check]
key-files:
  created:
    - internal/services/addomain/ldap_client_rfc4514_test.go
  modified:
    - internal/services/addomain/ldap_client.go
    - internal/services/addomain/ldap_client_78_07_test.go
decisions:
  - "转义状态位而非显式计反斜杠个数：escaping bool 在 `\\` 上翻转，天然实现奇偶语义，两个连续反斜杠让状态回到 false，其后逗号判为未转义分隔符"
  - "按 byte 而非 rune 遍历：RFC4514 转义字符与分隔符均为 ASCII，中文等多字节值的续字节不可能等于 ',' 或 '\\'；用 range 会引入下标错位"
  - "空串与 `CN=a\\\\,b` 两条走 t.Skip 而非删用例：前者 ParseDN 返回 0 个 RDN 无从比较，后者整串本身不是合法 DN（已用探针实测确认）"
  - "Task 1/2/3 合并为单个原子提交：实现翻转会使存量锁定缺陷的用例变红，三文件互为编译/语义依赖，分开提交会留下红的中间态"
metrics:
  duration: 约 25 分钟
  completed_date: 2026-10-19
  commits: 1
  files_changed: 3
commits: 1
plan_head_before: f8efa7dbf71b223a72fc64257303fe2cc4d3a932
plan_head_after: 264aaf3301584a6c46a1a721b46c3ed45d6ae1d9
---

# Quick 261019-mmc Summary: extractRDNFromDN 按首个未转义逗号切分

Phase 58 遗留 MISSED-04（数据破坏级）：`extractRDNFromDN` 用 `strings.Index(dn, ",")` 无条件切第一个字面逗号，CN 值内含 RFC4514 转义逗号（`CN=Zhang\,San`）时被截断成 `CN=Zhang\`。`MoveUser` 把截断串交给 `ldap.NewModifyDNRequest(userDN, rdn, true, newOUDN)`，`deleteOldRDN=true` 会让 AD 把对象改名成截断串并删掉旧 RDN —— 逗号后缀永久丢失。

---

## Tasks

### Task 1 — `extractRDNFromDN` 改为"首个未转义逗号"切分

`internal/services/addomain/ldap_client.go:429`（改前 7 行 → 改后 41 行，含注释）。

```go
func extractRDNFromDN(dn string) string {
	escaping := false
	for i := 0; i < len(dn); i++ {
		switch dn[i] {
		case '\\':
			escaping = !escaping
		case ',':
			if !escaping {
				return dn[:i]
			}
			escaping = false
		default:
			escaping = false
		}
	}
	return dn
}
```

- **为什么用 `escaping` 翻转而不是计反斜杠个数**：`\\` 连续两次翻转回 `false`，故其后的逗号被判为未转义分隔符 —— 奇偶语义由状态机天然给出，少一个易错的计数分支。
- **为什么按 byte 遍历**：`for i := 0; i < len(dn); i++` 用下标，`dn[i]` 是字节。多字节 UTF-8 的续字节取值域是 `0x80-0xBF`，不可能等于 `,`(0x2C) 或 `\`(0x5C)，中文 CN 天然安全。反之若 `for range` 按 rune 迭代再拿 rune 当下标会偏移错位（ threat T-Q261019-03）。
- **函数注释**补记了：RFC4514 分隔符语义、`ModifyDN(deleteOldRDN=true)` 的数据破坏链路、Phase 58 只给 OU 侧加 `ldap.EscapeDN`（`dn_util.go` `BuildOUDN`）而 RDN 侧此前未加、以及"不要顺手与 `NormalizeDN` 统一口径"的显式禁令。
- **未新增 import**（`strings` 仍在文件内被 `extractNetBIOSName` 等使用，`go build` 验证）。
- **未改 `NormalizeDN`**（它刻意不用 `ldap.ParseDN`，存量 `sys_dept_ou_mapping.ou_dn` 含未转义特殊字符）。

### Task 2 — 新增 `ldap_client_rfc4514_test.go`

新文件，无 build tag（同 `dn_util_test.go`），`package addomain`，命名 `TestPhase58_*` 以便 `-run TestPhase58` 捞起。

- **A 表 `TestPhase58_ExtractRDNFromDN_RFC4514Table`**：17 条共用数据（`rfc4514Cases`），字段 `{name, dn, want, crossCheck}`，`assert.Equal(t, tt.want, extractRDNFromDN(tt.dn))`。覆盖转义逗号（中文/英文/多处/末段/十六进制/多值 RDN 带转义）、多值 RDN、字面反斜杠奇偶三条、8 条回归不变（普通多段 / 多层 OU / 单 RDN / 空串 / 含空格 / 中文 CN）。
- **额外 `..._NotTruncatedAtEscapedComma`**：`assert.NotEqual(t, "CN=Zhang\\", got)` 单独钉死"不再被截断"，并断言结果是原串的真前缀且短于原串（切分确实发生）。
- **B 表 `..._AgreesWithGoLDAP`**：与 go-ldap `ldap.ParseDN` 交叉校验，每条 `crossCheck=true` 验证 (1) 整串可解析、(2) 截断输出可解析且 **`RDNs` 恰为 1**、(3) `parsed.RDNs[0].Equal(gotParsed.RDNs[0])` —— 用 `Equal` 而非字符串比较，因为 go-ldap 会小写 attr 名并对多值 RDN 排序。

### Task 3 — 翻转锁定旧（错误）行为的存量用例 + 全量回归

`ldap_client_78_07_test.go` 的 `TestLC78_ExtractRDNFromDN_Edge` 仅改"转义逗号"一条：`want` 由 `CN=John\` 翻为 `CN=John\,Doe`。注释重写为记录**期望为何翻转**（函数变为 RFC4514 正确；截断串会被 `ModifyDN(deleteOldRDN=true)` 当新 RDN），避免下一个读者读成"有人把通过的测试改坏了"。该文件其余三条、文件顶部 `!skip_db_tests` tag 与头部注释块均未动；`ldap_client_test.go` 零改动。

---

## RED → GREEN 证据

**RED**（新增测试文件就位、实现未改）：

```
--- FAIL: TestPhase58_ExtractRDNFromDN_RFC4514Table (0.00s)
    --- FAIL: .../转义逗号-中文姓名       expected: "CN=Zhang\\,San"  actual: "CN=Zhang\\"
    --- FAIL: .../转义逗号-英文姓         expected: "CN=John\\,Doe"    actual: "CN=John\\"
    --- FAIL: .../转义逗号后仍有多段      expected: "CN=a\\,b\\,c"    actual: "CN=a\\"
    --- FAIL: .../转义逗号出现在末段      expected: "CN=x\\,y"        actual: "CN=x\\"
    --- FAIL: .../十六进制转义值          expected: "CN=\\41\\,B"     actual: "CN=\\41\\"
    --- FAIL: .../多值 RDN 带转义逗号     expected: "CN=Zhang\\,San+SN=Z"  actual: "CN=Zhang\\"
--- FAIL: TestPhase58_ExtractRDNFromDN_NotTruncatedAtEscapedComma (0.00s)
--- FAIL: TestPhase58_ExtractRDNFromDN_AgreesWithGoLDAP (0.00s)
    （同样 6 条转义逗号子用例 FAIL）
FAIL	github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain	1.796s
```

恰好 6 条转义逗号用例红；8 条回归用例**在 RED 阶段就已是绿的**（证明它们对旧实现同样成立，即"行为不变"这一 must_have 被真实验证，而非事后宣称）。

**GREEN**（实现改后，同一命令）：

```
--- PASS: TestPhase58_ExtractRDNFromDN_RFC4514Table (0.00s)     [17/17 PASS]
--- PASS: TestPhase58_ExtractRDNFromDN_NotTruncatedAtEscapedComma (0.00s)
--- PASS: TestPhase58_ExtractRDNFromDN_AgreesWithGoLDAP (0.00s)  [15 PASS / 2 SKIP]
    --- SKIP: .../字面反斜杠且后半段无 =   "CN=a\\\\,b"
    --- SKIP: .../空串
ok  	github.com/ruoyi-next/ruoyi-go-backend/internal/services/addomain
```

---

## `ldap.ParseDN` 交叉校验结果

15 条通过、2 条跳过。每条跳过都单独核实过理由（不是照抄 plan）：

| 用例 | 结果 | 理由 |
|---|---|---|
| `CN=a\\,b`（2 反斜杠） | **SKIP** | 用探针实测（`cmd/_probe4514`，已删除）：`ldap.ParseDN` 返回 `err="DN ended with incomplete type, value pair"`, `nRDNs=1`。该逗号按 RFC4514 **确为**未转义分隔符，其后 `b` 是无 `=` 的残段 ⇒ **输入本身不是合法 DN**，不是实现的问题。该条由 A 表硬编码 `want="CN=a\\"` 守住。 |
| 空串 | **SKIP** | 实测 `ldap.ParseDN("")` → `err=<nil>`, `nRDNs=0`。第 3/4 步的 `parsed.RDNs[0]` 无从比较 ⇒ 由 A 表 `want=""` 守住。 |

其余 15 条全部走完 4 步判定（含"截断输出恰含 1 个 RDN"）。**`Equal` 的判别力经反向验证**：RED 阶段 6 条转义用例正是靠这一步变红的，说明它不是恒真断言。

探针另确认（供后续参考，均 PASS 侧）：`ParseDN("CN=a\\,OU=IT,DC=corp")` → 3 个 RDN，RDN[0] 值 `a\`；`ParseDN("CN=trailing\\,OU=IT,DC=corp")` → 3 个 RDN；`ParseDN("CN=ends-with-backslash\\")` → 1 个 RDN。

---

## 回归命令实际输出

| 命令 | 结果 |
|---|---|
| `go build ./...` | 退出 0，无输出 |
| `go vet ./internal/services/addomain/` | 退出 0，无输出 |
| `go test ./internal/services/addomain/ -count=1` | `ok ... 7.895s` |
| `go test ./internal/services/addomain/ -run TestPhase58 -count=1` | `ok ... 1.932s` |
| `go test ./internal/services/addomain/ -run TestPhase58 -count=10` | **FAIL** —— 见下方归因 |
| `go test ./internal/services/system/ -run TestPhase57 -count=1` | `ok ... 2.371s` |
| `go test ./internal/services/addomain/ -run 'TestLC78_ExtractRDNFromDN_Edge\|TestExtractRDN' -count=1` | 全部 PASS（4 + 3 + 3 子用例） |
| `go test ./internal/services/addomain/ -run 'TestPhase58_ExtractRDNFromDN\|TestLC78_ExtractRDNFromDN_Edge\|TestExtractRDN' -count=10` | `ok ... 1.691s`（本次新增/翻转的用例在 10 轮下确定性通过） |

### `-count=10` 失败的归因（非本次引入，已证伪）

失败者是 `TestPhase58_LoadDrainIntervalSeconds_FallsBackOnGarbage` 的 4 个子用例（non-numeric / zero / negative / valid-45），与 `extractRDNFromDN` 无调用关系。

用 `git worktree add /tmp/gsd-baseline-f8efa7db HEAD`（**未使用 git stash**）在基线 commit `f8efa7db` 上跑同一命令，得到**逐字相同的失败签名**：

```
--- FAIL: TestPhase58_LoadDrainIntervalSeconds_FallsBackOnGarbage (0.00s)
    --- FAIL: .../non-numeric    --- FAIL: .../zero
    --- FAIL: .../negative       --- FAIL: .../valid-45
```

⇒ 既有 flake（STATE.md 58-07 已如实登记："仅在 `-count=5` 失败，Plan 05 的 `cache=shared` DSN 跨轮次存活"），`-count=1` 全绿。**本次不修**（超出 scope）。worktree 已 `git worktree remove --force` 清理，`git worktree list` 只剩主工作树。

---

## 与 Plan 的偏差

1. **Task 1/2/3 合并为单个原子提交 `264aaf33`**（而非三个）。理由：三者互为依赖 —— 单独提交 Task 1 会让仍在断言 `CN=John\` 的存量用例变红；单独提交 Task 2 会让新增的 7 条用例变红。任何两两切分都会留下一个红的中间态，不可 bisect。
2. **A 表 17 条（原 plan 建议 16 条）**：拆细了两条以把"多值 RDN"与"多值 RDN 带转义逗号"、"字面反斜杠"与"值以转义反斜杠结尾"分列，使奇偶语义两向各有独立用例。断言强度提高，无一条是新增的宽松豁免。
3. **提交信息 body 不含双引号**：本机 `gsd-validate-commit.sh` 用 `-m[[:space:]]+\"([^\"]+)\"` 提取消息，body 内的 `"` 会截断提取并触发相邻守卫，导致所有含 `"` 的 heredoc 提交被 block。已改写为无 `"` 文本，语义不变。
4. **额外跑了一组隔离回归**（`-run 'TestPhase58_ExtractRDNFromDN|TestLC78_ExtractRDNFromDN_Edge|TestExtractRDN' -count=10`），用于在已知既有 flake 的包级 `-count=10` 之外，证明本次触及的用例本身 10 轮确定性通过。

无功能性偏差：四个调用点零改动、`go.mod`/`go.sum` 零变化、Phase 58 三条审计否决项（时间戳仲裁 / 新旧值比较 / `sys_job` 注册）均未触及。

**改动面核对**：`git show --stat HEAD` 恰好 3 个文件，与 frontmatter `files_modified` 一致；无越界重构。工作区 688 条 `.planning/phases/*` 未暂存删除未被触碰，全程未 `git stash`。

---

## 后续发现（**本次刻意未修**）

1. **存量脏数据不会被本次修复自愈（最高优先级）**。三处回写路径都是 `ad_dn = extractRDNFromDN(*user.AdDn) + "," + targetOUDN`，输入是**数据库里已有的 `sys_user.ad_dn`**。若某用户在修复部署前已经被排水器走过一次，其 `ad_dn` 已被写成截断串，那么本修复只会忠实地把那个截断前缀再传播一次，**不会还原**；AD 侧对象名也已是被改名后的截断值。建议后续 phase 评估一次性数据核查/修复（判据：RDN 末尾是孤立反斜杠 ⇒ 疑似截断），并与人工核对 AD 侧实际 CN。
2. **`ldap.ParseDN` 对 `CN=a\\,b` 的宽松解析**：go-ldap v3.4.12 在报错的同时仍返回了可用的 `RDNs[0]`（值 `a\`）。也就是说若将来放宽本测试的 `require.NoError`，第 3/4/5 步实际是能通过的 —— 但那是**建立在 go-ldap 宽松行为上**（其内部把 `\\,` 整体当作转义序列，与 RFC4514 严格解释不一致）。本次保持 SKIP 是正确的：整串解析已失败，不应把它当权威。
3. **`extractNetBIOSName`（同文件 :439）用 `strings.Split` + `ToUpper` 处理 `short.domain.com` 这类值，会产生 `SHORT` 而 `MoveUser` 场景不需要它** —— 与本次修复无关，仅记录位置，未触碰。

---

## Self-Check: PASSED

- 三个改动文件均存在：`ldap_client.go` / `ldap_client_rfc4514_test.go`（新） / `ldap_client_78_07_test.go`
- 提交存在且在 HEAD 祖先链上：`264aaf33`（`git merge-base --is-ancestor` 通过）
- 提交计数**实测**（ledger `gsd-plan-head-before-261019-mmc-01` → HEAD）：`git rev-list --count f8efa7db..264aaf33` = **1**，与 frontmatter `commits: 1` 一致，非叙述值
- `plan_head_before` / `plan_head_after` 均取自实测值
- SUMMARY frontmatter 含 `status: complete`
- 提交内无文件删除（`git show --first-parent --diff-filter=D` 输出为空）
- 工作区 688 条 `.planning/phases/*` 未暂存删除未被触碰；全程未使用 `git stash`
