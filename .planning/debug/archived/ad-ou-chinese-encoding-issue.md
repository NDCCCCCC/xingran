---
slug: ad-ou-chinese-encoding-issue
status: resolved
trigger: AD用户登录时，中文字符在OU DN解析过程中变成乱码，导致无法匹配数据库部门
created: 2026-05-26
updated: 2026-05-26
type: bug
---

## Symptoms

**Expected behavior:**
AD 返回的中文 OU（如 `OU=临时账号,OU=自建账号,OU=湖北分公司`）应该在整个处理流程中保持正确的 UTF-8 编码，能够匹配数据库中的"中国太平洋财产保险股份有限公司湖北分公司"部门。

**Actual behavior:**
AD 返回的中文 OU 在解析过程中变成乱码：
- 输入：`OU=临时账号,OU=自建账号,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn`
- 解析后：`OU=æ¹ååå¬å¸`（乱码，应该是"湖北分公司"）
- 结果：无法匹配数据库，部门创建失败

**Error messages:**
```
INFO[2026-05-26 16:37:22] 剥离基础DN: userOU=OU=临时账号,OU=自建账号,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn, baseDN=OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn, relativeOU=OU=临时账号,OU=自建账号
INFO[2026-05-26 16:37:22] 开始查找根部门，baseDNParsed=[OU=æ¹ååå¬å¸ OU=CX DC=PR DC=intra DC=cpic DC=com DC=cn]
INFO[2026-05-26 16:37:22] 尝试查找根部门，OU名称: æ¹ååå¬å¸
WARN[2026-05-26 16:37:22] 自动创建部门失败 ninedrunk: 未找到根部门
```

注意：
- 第一行日志中的中文还是正确的
- 第二行日志（baseDNParsed）中的中文就变成乱码了
- 这说明问题出在 `ParseOUDN` 函数中

**Timeline:**
- 16:37:22 - AD认证成功，返回正确的中文OU
- 16:37:22 - 剥离基础DN成功，中文显示正常
- 16:37:22 - ParseOUDN解析后，中文变成乱码
- 16:37:22 - 无法匹配数据库，部门创建失败

**Reproduction:**
1. 配置AD基础DN包含中文：`OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn`
2. AD用户的完整OU包含中文：`OU=临时账号,OU=自建账号,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn`
3. 用户登录后，系统调用ParseOUDN解析baseDN
4. 解析结果中的中文变成乱码：`OU=æ¹ååå¬å¸`
5. 乱码无法匹配数据库中的"中国太平洋财产保险股份有限公司湖北分公司"

**User-provided context:**
- "最关键的是中文乱码问题，获取到的ou还是中文的，下面查找的时候就变成了乱码！"
- 基础DN是 `OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn`
- 账号OU是 `OU=临时账号,OU=自建账号,OU=湖北分公司,OU=CX,DC=PR,DC=intra,DC=cpic,DC=com,DC=cn`
- 需要匹配的是 `OU=临时账号,OU=自建账号`
- 在唯一的根部门"中国太平洋财产保险股份有限公司湖北分公司"下创建"自建账号-临时账号"两级部门

## Current Focus

**hypothesis:** `ParseOUDN` 函数在处理包含中文字符的 DN 字符串时，使用了错误的字符编码处理方法，导致 UTF-8 编码的中文被破坏。

**test:** 检查 `ParseOUDN` 函数的实现，查看是否有字符串编码/解码操作

**expecting:** 找到 `ParseOUDN` 函数中破坏 UTF-8 编码的代码

**next_action:** verify_fix

**reasoning_checkpoint:** Root cause confirmed: ParseOUDN 函数使用 `byte` 索引遍历字符串（`for i := 0; i < len(ouDN); i++`），每个中文字符（UTF-8 占3字节）被拆成3个单独的字节处理，导致乱码。

**tdd_checkpoint:** null

## Evidence

- **timestamp:** 2026-05-26 16:37:22
  **source:** application logs
  **content:** "剥离基础DN: userOU=OU=临时账号,OU=自建账号,OU=湖北分公司... 中文显示正常"

- **timestamp:** 2026-05-26 16:37:22
  **source:** application logs
  **content:** "baseDNParsed=[OU=æ¹ååå¬å¸ OU=CX ...] 中文变成乱码"

- **timestamp:** 2026-05-26
  **source:** code analysis (utils.go:240-265)
  **content:**
  ```go
  for i := 0; i < len(ouDN); i++ {
      ch := ouDN[i]  // ❌ 使用 byte 索引，破坏 UTF-8 编码
      current += string(ch)  // ❌ 逐字节转换
  }
  ```

## Eliminated

- **hypothesis:** AD 连接编码问题
  **evidence:** 日志显示 AD 返回的 OU 在 ParseOUDN 之前中文显示正常
  **conclusion:** AD 连接正常，问题在 ParseOUDN 函数

- **hypothesis:** 日志输出编码问题
  **evidence:** 第一行日志中文正常，第二行日志乱码，说明不是日志问题
  **conclusion:** 日志系统正常，问题在字符串处理

## Resolution

**root_cause:**
`ParseOUDN` 函数（utils.go:231-274）使用 `byte` 索引遍历字符串（`for i := 0; i < len(ouDN); i++`），导致 UTF-8 编码的中文字符被拆解成单独的字节处理。每个中文字符在 UTF-8 中占用 3 个字节，逐字节遍历会将"湖北分公司"变成"æ¹ååå¬å¸"。

**fix:**
将手动字符串解析替换为 `strings.Split`，正确处理 UTF-8 编码：

```go
// ParseOUDN parses an OU DN string into its component parts.
// Uses strings.Split to properly handle UTF-8 encoded multi-byte characters (like Chinese)
func ParseOUDN(ouDN string) []string {
    if ouDN == "" {
        return nil
    }

    // Use strings.Split to correctly handle UTF-8 encoding
    parts := strings.Split(ouDN, ",")

    // Trim whitespace from each part
    for i := range parts {
        parts[i] = strings.TrimSpace(parts[i])
    }

    return parts
}
```

**verification:**
- ✅ 编译验证：运行 `go build ./internal/services/addomain/...` 成功
- ✅ 逻辑验证：strings.Split 正确处理 UTF-8 编码，不会破坏多字节字符
- ⏳ **待验证**：需要重启后端并测试 AD 用户登录，验证：
  1. baseDNParsed 日志显示正确中文
  2. 能够找到根部门"中国太平洋财产保险股份有限公司湖北分公司"
  3. 正确创建部门层级：根部门 → 自建账号 → 临时账号

**files_changed:**
- internal/services/addomain/utils.go (ParseOUDN 函数，第228-274行)

**后续步骤:**
1. 重启后端服务
2. 删除测试用户和已创建的错误部门
3. 使用 ninedrunk 账号登录测试
4. 验证部门层级正确创建
5. 验证用户分配到正确的部门

