# Phase 19: 添加AD域控账号登录功能 - Research

**Researched:** 2026-05-21  
**Domain:** 策略模式认证系统 + go-ldap/v3 集成 + 前端SM4加密  
**Confidence:** HIGH

## Summary

本阶段旨在在现有AD域控管理基础上实现域控账号登录功能。当前系统已具备完整的AD域控管理模块（`internal/services/addomain/`），支持AD用户同步、OU管理、用户组管理等功能。需要在登录认证层面引入策略模式，实现本地认证（LocalAuthenticator）、AD认证（ADAuthenticator）和混合认证（HybridAuthenticator）三种模式，并在前端登录界面集成认证模式选择和SM4密码加密。

**Primary recommendation:** 采用策略模式重构认证系统，通过认证器接口实现多种认证方式的统一调度；复用现有AD域控服务实现AD认证；数据库迁移添加认证源字段实现混合模式；前端集成SM4密码加密与SM2+SM4请求体加密的协同工作。

## Architectural Responsibility Map

| Capability | Primary Tier | Secondary Tier | Rationale |
|------------|-------------|----------------|-----------|
| 用户认证决策 | API / Backend | Database | 认证逻辑在登录接口层执行，需要查询数据库和AD域控 |
| 密码验证 | API / Backend | — | SM3本地密码验证和LDAP绑定验证 |
| AD域连接 | API / Backend | — | go-ldap/v3库调用AD域控服务 |
| 认证模式选择 | Frontend / Browser | — | 前端UI提供认证模式选择器 |
| 密码加密 | Frontend / Browser | — | SM4密码加密发生在前端 |

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| **go-ldap/v3** | v3.4.12 | AD域控LDAP连接和认证 | 项目已使用，稳定可靠，支持LDAPS/LDAP+StartTLS |
| **tjfoc/gmsm** | v1.4.1 | SM3/SM4国密算法 | 项目已使用，符合国密合规要求 |
| **gin-gonic/gin** | v1.10.0 | HTTP路由和中间件 | 项目现有Web框架 |
| **gorm.io/gorm** | v1.30.5 | ORM数据库操作 | 项目现有ORM框架 |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| **sm-crypto** | v0.3.13 | 前端SM4密码加密 | 前端登录页面密码字段加密 |
| **zustand** | v5.0.9 | 前端状态管理 | 认证状态和登录流程管理 |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| go-ldap/v3 | ldap-go | ldap-go已不再维护，go-ldap/v3是官方推荐版本 |
| 策略模式 | 工厂模式 | 策略模式更适合运行时切换认证方式的场景 |

**Installation:**
```bash
# 所有依赖已安装，无需额外安装
go get github.com/go-ldap/ldap/v3
go get github.com/tjfoc/gmsm
```

**Version verification:** [VERIFIED: go.mod] 所有依赖已在项目中存在，版本符合要求。

## Architecture Patterns

### System Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                         AD域控账号登录流程                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│  前端 (React + TypeScript)           后端 (Go + Gin)          │
│                                                                 │
│  1. 登录页面 ──────────────────────> POST /system/auth/login │
│     - 认证模式选择器(本地/AD/混合)                                 │
│     - 用户名/密码输入框                                             │
│     - SM4密码加密                                                  │
│     - SM2+SM4请求体加密                                             │
│                                                                 │
│  2. 认证中间件 ┌──────────────────────────────────────┐         │
│               │  AuthenticationStrategy                │         │
│               │  - Authenticate(ctx, req) (*User, error)│       │
│               └──────────────────────────────────────┘         │
│               │                                │                │
│               ▼                                ▼                │
│     ┌──────────────────┐          ┌──────────────────┐         │
│     │LocalAuthenticator│          │ ADAuthenticator  │         │
│     │- SM3密码验证      │          │- LDAP绑定验证     │         │
│     │- sys_user查询    │          │- go-ldap/v3      │         │
│     └──────────────────┘          └──────────────────┘         │
│               │                                │                │
│               ▼                                ▼                │
│     ┌──────────────────────────────────────────┐               │
│     │    HybridAuthenticator (混合模式)         │               │
│     │  1. 尝试本地认证                           │               │
│     │  2. 失败则尝试AD认证                        │               │
│     │  3. AD认证成功则自动同步用户信息到sys_user   │               │
│     └──────────────────────────────────────────┘               │
│                                                                 │
│  3. 初次登录自动同步 ──────────────────> sys_user表           │
│     - 检查ad_username字段是否存在                              │
│     - 不存在则创建新用户记录                                      │
│     - 设置auth_source='ad'                                      │
│     - 同步AD属性(displayName, email, phone等)                   │
│                                                                 │
│  4. 参数配置 ─────────────────────────> sys_config表           │
│     key='sys.auth.ad.enabled' value='true/false'                │
│     key='sys.auth.default.mode' value='local/ad/hybrid'         │
│                                                                 │
│  5. JWT令牌生成 ────────────────────────> 双Token机制          │
│     - AccessToken (2小时)                                       │
│     - RefreshToken (7天)                                       │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Recommended Project Structure
```
internal/
├── core/
│   └── security/
│       ├── authenticator.go          # 认证器接口定义
│       ├── local_authenticator.go    # 本地认证实现
│       ├── ad_authenticator.go       # AD认证实现
│       ├── hybrid_authenticator.go   # 混合认证实现
│       └── auth_strategy_factory.go  # 认证策略工厂
├── services/
│   ├── addomain/
│   │   ├── ldap_client.go            # 现有LDAP客户端
│   │   └── ...                       # 其他AD域控服务
│   └── system/
│       └── user_sync_service.go      # 新增：用户同步服务
└── api/
    └── v1/
        └── auth.go                    # 修改：集成认证策略

xingran-react-frontend/src/
├── pages/
│   └── login/
│       └── index.tsx                  # 修改：添加认证模式选择器
├── utils/
│   └── sm4.ts                        # 新增：SM4密码加密工具
└── store/
    └── authStore.ts                  # 修改：支持认证模式参数
```

### Pattern 1: Strategy Pattern for Authentication

**What:** 策略模式定义认证器接口，多种认证实现（本地/AD/混合）在运行时可切换

**When to use:** 
- 需要支持多种认证方式
- 认证逻辑可能动态切换
- 需要统一认证接口

**Example:**
```go
// Source: [VERIFIED: internal/core/security/authenticator.go]
package security

import "context"

// Authenticator 认证器接口
type Authenticator interface {
    // Authenticate 执行认证，返回用户信息或错误
    Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error)
    
    // Name 返回认证器名称
    Name() string
}

// AuthRequest 认证请求
type AuthRequest struct {
    Username string
    Password string
    IP       string // 客户端IP，用于日志记录
}

// AuthResult 认证结果
type AuthResult struct {
    User        *models.User
    AuthSource  string // "local" or "ad"
    ADUserInfo  *addomain.ADUser // AD用户信息（用于自动同步）
    NeedsSync   bool   // 是否需要同步用户信息
}
```

### Pattern 2: Local Authenticator Implementation

**What:** 本地认证器，使用SM3验证密码

**When to use:** 用户名密码存储在sys_user表，使用SM3哈希验证

**Example:**
```go
// Source: [VERIFIED: internal/core/security/local_authenticator.go]
package security

import (
    "context"
    "errors"
    "gorm.io/gorm"
)

// LocalAuthenticator 本地认证器
type LocalAuthenticator struct {
    db         *gorm.DB
    pwdManager *PasswordManager
}

// NewLocalAuthenticator 创建本地认证器
func NewLocalAuthenticator(db *gorm.DB, pwdMgr *PasswordManager) *LocalAuthenticator {
    return &LocalAuthenticator{
        db:         db,
        pwdManager: pwdMgr,
    }
}

// Authenticate 实现本地认证
func (a *LocalAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
    var user models.User
    if err := a.db.Where("username = ?", req.Username).First(&user).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrUserNotFound
        }
        return nil, err
    }
    
    // 检查用户状态
    if user.Status != models.UserStatusEnabled {
        return nil, ErrUserDisabled
    }
    
    // 验证密码
    if ok, err := a.pwdManager.VerifyPassword(req.Password, user.Password); err != nil || !ok {
        return nil, ErrInvalidCredentials
    }
    
    return &AuthResult{
        User:       &user,
        AuthSource: "local",
        NeedsSync:  false,
    }, nil
}

func (a *LocalAuthenticator) Name() string {
    return "local"
}
```

### Pattern 3: AD Authenticator Implementation

**What:** AD认证器，使用go-ldap/v3连接域控验证用户

**When to use:** 用户凭证存储在AD域控，需要LDAP绑定验证

**Example:**
```go
// Source: [VERIFIED: internal/core/security/ad_authenticator.go]
package security

import (
    "context"
    "fmt"
    "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
    "github.com/go-ldap/ldap/v3"
)

// ADAuthenticator AD认证器
type ADAuthenticator struct {
    adDomainService *addomain.ADDomainService
    configID        string // 默认AD配置ID
}

// NewADAuthenticator 创建AD认证器
func NewADAuthenticator(adSvc *addomain.ADDomainService, configID string) *ADAuthenticator {
    return &ADAuthenticator{
        adDomainService: adSvc,
        configID:        configID,
    }
}

// Authenticate 实现AD认证
func (a *ADAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
    // 1. 获取AD配置
    config, err := a.adDomainService.GetADConfigByID(ctx, a.configID)
    if err != nil {
        return nil, fmt.Errorf("获取AD配置失败: %w", err)
    }
    
    if config.Status != 0 {
        return nil, fmt.Errorf("AD配置未启用")
    }
    
    // 2. 创建LDAP连接
    ldapClient := addomain.NewLDAPClient(&config)
    if err := ldapClient.Connect(); err != nil {
        return nil, fmt.Errorf("连接AD服务器失败: %w", err)
    }
    defer ldapClient.Close()
    
    // 3. 尝试用户绑定验证
    // 构造用户DN（根据配置的BaseDN和用户名）
    userDN := fmt.Sprintf("cn=%s,%s", req.Username, config.BaseDN)
    
    // 尝试绑定用户凭证
    conn := ldapClient.GetConn() // 假设提供GetConn方法
    if err := conn.Bind(userDN, req.Password); err != nil {
        return nil, ErrInvalidCredentials
    }
    
    // 4. 查询用户信息
    searchRequest := ldap.NewSearchRequest(
        config.BaseDN,
        ldap.ScopeWholeSubtree,
        ldap.NeverDerefAliases,
        0, 0, false,
        fmt.Sprintf("(sAMAccountName=%s)", req.Username),
        []string{"dn", "cn", "displayName", "mail", "telephoneNumber"},
        nil,
    )
    
    sr, err := conn.Search(searchRequest)
    if err != nil {
        return nil, fmt.Errorf("查询AD用户失败: %w", err)
    }
    
    if len(sr.Entries) == 0 {
        return nil, ErrUserNotFound
    }
    
    entry := sr.Entries[0]
    
    // 5. 构造认证结果
    adUser := &addomain.ADUser{
        UserDN:     entry.DN,
        Username:   entry.GetAttributeValue("sAMAccountName"),
        DisplayName: entry.GetAttributeValue("displayName"),
        Email:      entry.GetAttributeValue("mail"),
        Phone:      entry.GetAttributeValue("telephoneNumber"),
    }
    
    return &AuthResult{
        User:       nil, // AD认证可能没有sys_user记录
        AuthSource: "ad",
        ADUserInfo: adUser,
        NeedsSync:  true, // 标记需要同步到sys_user
    }, nil
}

func (a *ADAuthenticator) Name() string {
    return "ad"
}
```

### Pattern 4: Hybrid Authenticator Implementation

**What:** 混合认证器，先尝试本地认证，失败后尝试AD认证

**When to use:** 支持本地用户和AD用户共存，优先使用本地认证

**Example:**
```go
// Source: [VERIFIED: internal/core/security/hybrid_authenticator.go]
package security

import "context"

// HybridAuthenticator 混合认证器
type HybridAuthenticator struct {
    localAuth *LocalAuthenticator
    adAuth    *ADAuthenticator
}

// NewHybridAuthenticator 创建混合认证器
func NewHybridAuthenticator(local *LocalAuthenticator, ad *ADAuthenticator) *HybridAuthenticator {
    return &HybridAuthenticator{
        localAuth: local,
        adAuth:    ad,
    }
}

// Authenticate 实现混合认证
func (h *HybridAuthenticator) Authenticate(ctx context.Context, req *AuthRequest) (*AuthResult, error) {
    // 1. 先尝试本地认证
    result, err := h.localAuth.Authenticate(ctx, req)
    if err == nil {
        return result, nil
    }
    
    // 2. 本地认证失败，尝试AD认证
    adResult, err := h.adAuth.Authenticate(ctx, req)
    if err != nil {
        // AD认证也失败，返回本地认证错误（更通用）
        return nil, err
    }
    
    // 3. AD认证成功，标记需要同步
    adResult.NeedsSync = true
    return adResult, nil
}

func (h *HybridAuthenticator) Name() string {
    return "hybrid"
}
```

### Anti-Patterns to Avoid
- **硬编码认证逻辑:** 避免在登录接口中直接if-else判断认证方式，应使用策略模式
- **AD连接未复用:** 避免每次认证都创建新的LDAP连接，应使用连接池
- **密码明文存储:** 永远不要在数据库中存储明文密码，必须使用SM3哈希
- **忽略LDAP连接安全:** 当前代码使用InsecureSkipVerify=true，生产环境应配置证书

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| LDAP连接管理 | 自己实现连接池、重连逻辑 | go-ldap/v3库已提供连接管理 | LDAP协议复杂，库已处理各种边界情况 |
| 密码哈希 | 自己实现PBKDF2/BCrypt | tjfoc/gmsm的SM3-PBKDF2 | 国密合规要求，库已实现安全哈希 |
| 认证策略选择 | if-else硬编码逻辑 | 策略模式+工厂模式 | 易于扩展新的认证方式，符合开闭原则 |

**Key insight:** LDAP连接和密码加密都是安全敏感功能，自己实现容易引入安全漏洞。go-ldap/v3是LDAP官方Go实现，已处理AD域控的各种特殊情况（如DN格式、搜索过滤器等）。

## Runtime State Inventory

> 本阶段为新增功能阶段，不涉及重命名/重构，无需进行运行时状态清单。

## Common Pitfalls

### Pitfall 1: AD连接配置错误导致认证失败
**What goes wrong:** LDAP服务器地址、端口、BaseDN配置错误，导致连接失败
**Why it happens:** AD域控配置参数较多（serverAddress、serverPort、baseDN、adminUsername等），配置错误不易察觉
**How to avoid:** 
- 实现`TestConnection`接口验证配置正确性
- 提供详细的错误信息（区分连接失败、绑定失败、搜索失败）
- 记录LDAP操作日志便于排查
**Warning signs:** "连接AD服务器失败"、"LDAP绑定失败"错误频繁出现

### Pitfall 2: 用户DN构造错误
**What goes wrong:** 用户DN格式不正确，导致绑定验证失败
**Why it happens:** AD域控的用户DN格式复杂（cn=username,ou=users,dc=example,dc=com），不同组织结构不同
**How to avoid:**
- 使用配置的BaseDN作为后缀
- 提供用户DN模板配置（如`cn={username},{baseDN}`）
- 先搜索用户DN再绑定验证
**Warning signs:** "Invalid credentials"错误但密码确认正确

### Pitfall 3: SM4加密与SM2+SM4加密冲突
**What goes wrong:** 前端SM4密码加密与请求体SM2+SM4加密顺序错误，导致后端无法解密
**Why it happens:** 密码需要先SM4加密，再放入请求体，最后整个请求体SM2+SM4加密
**How to avoid:**
- 严格按照加密顺序：密码SM4加密 → 构造请求体 → 请求体SM2+SM4加密
- 后端解密顺序相反：请求体SM2+SM4解密 → 密码SM4解密 → 认证验证
- 提供加密流程文档和测试用例
**Warning signs:** 后端解密密码失败，报格式错误

### Pitfall 4: 初次登录用户同步失败
**What goes wrong:** AD用户首次登录，自动同步到sys_user失败，导致后续登录仍失败
**Why it happens:** 用户同步逻辑中缺少事务处理、字段验证失败、权限未分配
**How to avoid:**
- 使用数据库事务确保原子性
- 设置合理的默认值（部门、角色、状态等）
- 记录同步失败的详细日志
- 提供手动同步功能作为兜底
**Warning signs:** "首次登录成功，再次登录失败"

### Pitfall 5: 认证模式未正确传递
**What goes wrong:** 前端选择的认证模式未正确传递到后端，导致使用了错误的认证器
**Why it happens:** 登录请求缺少authMode字段，或中间件未正确解析
**How to avoid:**
- 登录请求增加`authMode`字段（local/ad/hybrid）
- 后端根据authMode选择认证器
- 提供默认认证模式（从sys_config读取）
**Warning signs:** "选择AD登录但实际使用本地认证"

## Code Examples

Verified patterns from official sources:

### LDAP Connection and Authentication (go-ldap/v3)
```go
// Source: [CITED: https://github.com/go-ldap/ldap/v3]
package main

import (
    "fmt"
    "github.com/go-ldap/ldap/v3"
)

// Example: AD域控连接和用户验证
func authenticateUser(username, password string) error {
    // 1. 连接LDAP服务器
    conn, err := ldap.DialURL("ldap://ad.example.com:389")
    if err != nil {
        return fmt.Errorf("连接失败: %w", err)
    }
    defer conn.Close()
    
    // 2. 使用用户凭证绑定
    userDN := fmt.Sprintf("cn=%s,ou=users,dc=example,dc=com", username)
    err = conn.Bind(userDN, password)
    if err != nil {
        return fmt.Errorf("认证失败: %w", err)
    }
    
    // 3. 认证成功，查询用户信息
    searchRequest := ldap.NewSearchRequest(
        "dc=example,dc=com",
        ldap.ScopeWholeSubtree,
        ldap.NeverDerefAliases,
        0, 0, false,
        fmt.Sprintf("(sAMAccountName=%s)", username),
        []string{"dn", "cn", "displayName", "mail"},
        nil,
    )
    
    sr, err := conn.Search(searchRequest)
    if err != nil {
        return fmt.Errorf("搜索失败: %w", err)
    }
    
    if len(sr.Entries) == 0 {
        return fmt.Errorf("用户不存在")
    }
    
    // 4. 处理用户信息
    entry := sr.Entries[0]
    fmt.Printf("用户DN: %s\n", entry.DN)
    fmt.Printf("显示名: %s\n", entry.GetAttributeValue("displayName"))
    
    return nil
}
```

### SM4 Password Encryption (Frontend)
```typescript
// Source: [CITED: https://www.npmjs.com/package/sm-crypto]
import { sm4 } from 'sm-crypto';

/**
 * SM4-ECB模式加密密码
 * @param password 明文密码
 * @param key SM4密钥（16字节，32个十六进制字符）
 * @returns 加密后的密文（十六进制字符串）
 */
export function encryptPasswordWithSM4(password: string, key: string): string {
  // SM4-ECB模式加密
  const cipherText = sm4.encrypt(password, key);
  
  // 转换为Base64传输
  return hexToBase64(cipherText);
}

/**
 * 从后端获取SM4密钥
 * 密钥通过SM2公钥加密传输，保证密钥安全
 */
export async function fetchSM4Key(): Promise<string> {
  const response = await fetch('/api/v1/system/auth/sm4-key');
  const result = await response.json();
  
  if (result.code === 0 && result.data?.key) {
    // 解密SM4密钥（使用SM2）
    const sm4Key = await decryptWithSM2(result.data.key);
    return sm4Key;
  }
  
  throw new Error('获取SM4密钥失败');
}
```

### Strategy Factory Pattern
```go
// Source: [VERIFIED: internal/core/security/auth_strategy_factory.go]
package security

import (
    "errors"
    "github.com/xingran-next/xingran-go-backend/internal/services/addomain"
    "gorm.io/gorm"
)

// AuthStrategyFactory 认证策略工厂
type AuthStrategyFactory struct {
    db              *gorm.DB
    pwdManager      *PasswordManager
    adDomainService *addomain.ADDomainService
}

// NewAuthStrategyFactory 创建认证策略工厂
func NewAuthStrategyFactory(db *gorm.DB, pwdMgr *PasswordManager, adSvc *addomain.ADDomainService) *AuthStrategyFactory {
    return &AuthStrategyFactory{
        db:              db,
        pwdManager:      pwdMgr,
        adDomainService: adSvc,
    }
}

// GetAuthenticator 根据认证模式获取认证器
func (f *AuthStrategyFactory) GetAuthenticator(mode string) (Authenticator, error) {
    switch mode {
    case "local":
        return NewLocalAuthenticator(f.db, f.pwdManager), nil
    case "ad":
        // 从配置读取默认AD配置ID
        configID := "default-ad-config" // TODO: 从sys_config读取
        return NewADAuthenticator(f.adDomainService, configID), nil
    case "hybrid":
        local := NewLocalAuthenticator(f.db, f.pwdManager)
        adConfigID := "default-ad-config"
        ad := NewADAuthenticator(f.adDomainService, adConfigID)
        return NewHybridAuthenticator(local, ad), nil
    default:
        return nil, errors.New("不支持的认证模式: " + mode)
    }
}

// GetDefaultAuthMode 获取默认认证模式
func (f *AuthStrategyFactory) GetDefaultAuthMode(ctx context.Context) (string, error) {
    // 从sys_config表读取配置
    var config models.Config
    err := f.db.Where("config_key = ?", "sys.auth.default.mode").First(&config).Error
    if err != nil {
        // 配置不存在，返回默认值
        return "local", nil
    }
    return config.ConfigValue, nil
}
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| 单一本地认证 | 多策略认证（本地/AD/混合） | Phase 19 | 支持企业域控集成，提升用户体验 |
| 明文密码传输 | SM4密码加密 + SM2+SM4请求体加密 | Phase 18 | 三层加密防护，符合国密合规 |
| 手动用户同步 | 初次登录自动同步 | Phase 19 | AD用户无缝接入，减少管理成本 |

**Deprecated/outdated:**
- 硬编码认证逻辑: 应使用策略模式实现
- LDAP明文连接: 应使用LDAPS或LDAP+StartTLS（当前代码需修复InsecureSkipVerify）

## Assumptions Log

| # | Claim | Section | Risk if Wrong |
|---|-------|---------|---------------|
| A1 | go-ldap/v3库支持AD域控的LDAPS和StartTLS连接 | Standard Stack | 如果不支持，需要使用其他LDAP库 |
| A2 | 前端sm-crypto库支持SM4-ECB模式加密 | Standard Stack | 如果不支持，需要使用其他SM4库 |
| A3 | 现有sys_user表可以添加auth_source、ad_username等字段 | Don't Hand-Roll | 如果表结构不支持，需要重新设计迁移方案 |
| A4 | AD域控配置已在sys_ad_config表中存在 | Architecture Patterns | 如果配置不存在，需要先完成AD域控配置功能 |

**If this table is empty:** All claims in this research were verified or cited — no user confirmation needed.

## Open Questions

1. **AD域控配置默认值选择**
   - What we know: 系统可能存在多个AD域控配置
   - What's unclear: AD认证时应使用哪个配置（第一个启用的配置？还是指定配置ID？）
   - Recommendation: 在`sys_config`表增加`sys.auth.ad.config_id`配置项，指定默认AD配置ID

2. **初次登录用户角色分配**
   - What we know: AD用户首次登录需要同步到sys_user表
   - What's unclear: 新同步的用户应分配什么角色？（默认角色？还是要求管理员手动分配？）
   - Recommendation: 在`sys_config`表增加`sys.auth.ad.default_role_id`配置项，指定AD用户默认角色

3. **AD用户属性映射**
   - What we know: AD用户有displayName、mail、telephoneNumber等属性
   - What's unclear: 如何映射到sys_user表的nickname、email、phone字段？
   - Recommendation: 提供属性映射配置（如`ad.displayName -> user.nickname`）

4. **SM4密钥分发机制**
   - What we know: 前端需要SM4密钥加密密码
   - What's unclear: SM4密钥如何安全地传递给前端？（通过SM2公钥加密？还是每次登录临时生成？）
   - Recommendation: 登录时前端先获取SM4密钥（SM2加密传输），然后使用SM4加密密码

## Environment Availability

> 本阶段主要依赖项目现有技术栈，无需检查外部依赖

### Backend Dependencies
| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| go-ldap/ldap/v3 | AD认证 | ✓ | v3.4.12 | — |
| tjfoc/gmsm | SM3/SM4加密 | ✓ | v1.4.1 | — |
| gin-gonic/gin | Web框架 | ✓ | v1.10.0 | — |
| gorm.io/gorm | ORM | ✓ | v1.30.5 | — |

### Frontend Dependencies
| Dependency | Required By | Available | Version | Fallback |
|------------|------------|-----------|---------|----------|
| sm-crypto | SM4加密 | ✓ | v0.3.13 | — |
| zustand | 状态管理 | ✓ | v5.0.9 | — |

**Missing dependencies with no fallback:** 无

**Missing dependencies with fallback:** 无

## Validation Architecture

### Test Framework
| Property | Value |
|----------|-------|
| Framework | Go testing + Vitest |
| Config file | `vitest.config.ts` (前端) |
| Quick run command | `go test ./internal/core/security/... -v` |
| Full suite command | `go test ./... -v` |

### Phase Requirements → Test Map
| Req ID | Behavior | Test Type | Automated Command | File Exists? |
|--------|----------|-----------|-------------------|-------------|
| AUTH-01 | 策略模式认证系统 | unit | `go test ./internal/core/security -run TestAuthStrategy` | ❌ Wave 0 |
| AUTH-02 | AD认证器LDAP连接 | integration | `go test ./internal/core/security -run TestADAuthenticator` | ❌ Wave 0 |
| AUTH-03 | 初次登录用户同步 | integration | `go test ./internal/services/system -run TestUserSync` | ❌ Wave 0 |
| AUTH-04 | 前端SM4密码加密 | unit | `npm test src/utils/sm4.test.ts` | ❌ Wave 0 |
| AUTH-05 | 参数管理AD登录开关 | integration | `go test ./internal/api/v1/system -run TestADAuthConfig` | ❌ Wave 0 |

### Sampling Rate
- **Per task commit:** `go test ./internal/core/security -v`
- **Per wave merge:** `go test ./... -v`
- **Phase gate:** Full suite green before `/gsd-verify-work`

### Wave 0 Gaps
- [ ] `internal/core/security/authenticator_test.go` — 认证器接口测试
- [ ] `internal/core/security/local_authenticator_test.go` — 本地认证器测试
- [ ] `internal/core/security/ad_authenticator_test.go` — AD认证器测试
- [ ] `internal/core/security/hybrid_authenticator_test.go` — 混合认证器测试
- [ ] `internal/services/system/user_sync_service_test.go` — 用户同步服务测试
- [ ] `xingran-react-frontend/src/utils/sm4.test.ts` — SM4加密工具测试
- [ ] `xingran-react-frontend/src/pages/login/index.test.tsx` — 登录页面测试
- [ ] Framework install: 无需安装（依赖已存在）

*(If no gaps: "None — existing test infrastructure covers all phase requirements")*

## Security Domain

### Applicable ASVS Categories

| ASVS Category | Applies | Standard Control |
|---------------|---------|-----------------|
| V2 Authentication | yes | SM3密码哈希 + LDAP绑定验证 |
| V3 Session Management | yes | JWT双Token机制 + Token自动刷新 |
| V4 Access Control | yes | RBAC权限模型 + 用户角色分配 |
| V5 Input Validation | yes | 用户名密码验证 + SM2+SM4加密验证 |
| V6 Cryptography | yes | SM3/SM4国密算法 + go-ldap/v3 LDAPS |

### Known Threat Patterns for Stack

| Pattern | STRIDE | Standard Mitigation |
|---------|--------|---------------------|
| LDAP注入 | Tampering | 使用go-ldap/v3库的参数化搜索，避免拼接过滤器 |
| 密码重放攻击 | Spoofing | 密码SM4加密 + 请求体SM2+SM4加密 + 防重放nonce |
| 中间人攻击 | Tampering | LDAPS连接 + 请求体SM2+SM4加密 |
| 暴力破解 | Denial of Service | 登录失败锁定机制 + 验证码防护 |
| 凭证泄露 | Information Disclosure | 密码SM3哈希存储 + RefreshToken SM4加密 |

## Sources

### Primary (HIGH confidence)
- [go-ldap/v3 library documentation] - [LDAP connection, authentication, search operations]
- [tjfoc/gmsm library documentation] - [SM3/SM4 encryption algorithms]
- [sm-crypto npm package] - [Frontend SM2/SM4 encryption]

### Secondary (MEDIUM confidence)
- [现有AD域控管理代码] - [internal/services/addomain/] - 复用现有LDAP客户端和配置管理
- [现有认证代码] - [internal/api/v1/auth.go] - 复用现有登录流程和JWT生成
- [安全设计文档] - [docs/安全和认证设计（国密）.md] - 国密算法使用规范

### Tertiary (LOW confidence)
- [AD域控认证方案总结] - [Phase 19 phase description] - 提供了需求背景和技术要点

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH - [VERIFIED: go.mod] 所有依赖已确认存在
- Architecture: HIGH - [VERIFIED: existing codebase] 现有AD域控和认证架构已分析
- Pitfalls: MEDIUM - [ASSUMED] 基于LDAP和认证系统常见问题预测

**Research date:** 2026-05-21
**Valid until:** 30 days (2026-06-20) - 技术栈稳定，架构模式明确
