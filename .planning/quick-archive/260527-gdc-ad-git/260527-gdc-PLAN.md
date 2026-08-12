---
phase: quick
plan: 260527-gdc
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/core/core.go
  - internal/core/security/ad_authenticator.go
  - internal/core/security/auth_strategy_factory.go
  - internal/services/addomain/config.go
autonomous: true
requirements: [bugfix-ad-password-encryption]
must_haves:
  truths:
    - "AD密码加解密使用统一的SM4加密器，initSM4Cipher不再返回nil"
    - "ADAuthenticator通过PasswordCipher接口解密密码，含SM4+AES-legacy回退链"
    - "代码编译通过，无语法错误"
  artifacts:
    - path: "internal/core/core.go"
      provides: "修复后的initSM4Cipher函数，删除CryptoSM4Cipher死代码"
    - path: "internal/core/security/auth_strategy_factory.go"
      provides: "SM4 cipher注入到ADAuthenticator"
    - path: "internal/core/security/ad_authenticator.go"
      provides: "PasswordCipher接口解密，统一回退链"
    - path: "internal/services/addomain/config.go"
      provides: "连接测试调试日志"
  key_links:
    - from: "internal/core/core.go"
      to: "internal/core/security/auth_strategy_factory.go"
      via: "NewAuthStrategyFactory(db, pwdMgr, c.SM4Cipher)"
      pattern: "NewAuthStrategyFactory.*SM4Cipher"
    - from: "auth_strategy_factory.go"
      to: "ad_authenticator.go"
      via: "ad.SetPasswordCipher(f.sm4Cipher)"
      pattern: "SetPasswordCipher"
---

<objective>
提交 AD 密码加解密修复代码到 git 仓库。

Purpose: 修复 initSM4Cipher() 硬编码返回 nil 导致 AD 域控密码加解密完全失效的问题。修复已验证通过编译，需要提交到版本控制。

Output: git commit 包含 4 个修复文件。
</objective>

<execution_context>
@$HOME/.claude/get-shit-done/workflows/execute-plan.md
@$HOME/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/debug/ad-password-encryption-decryption-errors.md
</context>

<tasks>

<task type="auto">
  <name>Task 1: 验证编译并提交 AD 密码加解密修复</name>
  <files>internal/core/core.go, internal/core/security/ad_authenticator.go, internal/core/security/auth_strategy_factory.go, internal/services/addomain/config.go</files>
  <action>
    1. 确认 go build ./... 编译通过（已验证，再次确认）
    2. 使用 git add 暂存以下 4 个文件：
       - internal/core/core.go
       - internal/core/security/ad_authenticator.go
       - internal/core/security/auth_strategy_factory.go
       - internal/services/addomain/config.go
    3. 创建 commit，消息遵循仓库 conventional commits 风格：
       fix(ad): 修复 AD 域控密码加解密逻辑，统一 SM4 加密器初始化

       修复内容:
       - core.go: initSM4Cipher() 使用 crypto.NewSM4Cipher() 替代硬编码 nil
       - core.go: 删除未使用的 CryptoSM4Cipher 死代码字段
       - core.go: initAuthFactory() 将 SM4Cipher 传递给 AuthStrategyFactory
       - auth_strategy_factory.go: 接收并注入 SM4 cipher 到 ADAuthenticator
       - ad_authenticator.go: sm4Cipher 类型改为 PasswordCipher 接口
       - ad_authenticator.go: decryptPassword 含 SM4 + AES-legacy 完整回退链
       - config.go: 添加连接测试调试日志
  </action>
  <verify>
    <automated>cd D:/CODE/ClaudeCode/xingran-go-backend && go build ./... && git log --oneline -1</automated>
  </verify>
  <done>4 个文件已提交，编译通过，commit 消息符合 conventional commits 风格</done>
</task>

</tasks>

<verification>
go build ./... 编译通过
git log --oneline -1 显示新提交
git diff HEAD~1 --stat 显示恰好 4 个文件变更
</verification>

<success_criteria>
- 4 个 AD 密码加解密相关文件已提交到 git
- 编译无错误
- commit 消息清晰描述修复内容
</success_criteria>

<output>
After completion, create `.planning/quick/260527-gdc-ad-git/260527-gdc-SUMMARY.md`
</output>
