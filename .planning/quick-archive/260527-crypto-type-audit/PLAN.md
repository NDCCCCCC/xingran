---
title: 加解密类型一致性审查
status: pending
created: 2026-05-27
updated: 2026-05-27
quick_id: 260527-mln
slug: crypto-type-audit
---

# 加解密类型一致性审查

## 目标
排查所有加解密相关的逻辑，检查是否存在类似刚才发现的类型不匹配问题。

## 背景
刚才修复了 `connection_pool.go` 中的类型不匹配问题：
- Core.SM4Cipher 是 `addomain.PasswordCipher` 接口类型
- ConnectionPool 期望 `*crypto.SM4Cipher` 具体类型
- 导致初始化时传入 `nil`，无法解密密码

## 审查范围
1. **SM2/SM3/SM4 加密模块** (`pkg/crypto/`)
2. **密码管理器** (`internal/core/security/password.go`)
3. **请求加密中间件** (`pkg/middleware/encryption.go`)
4. **AD 域密码加解密** (`internal/services/addomain/`)
5. **所有调用加解密的地方**

## 审查重点
- [ ] 接口类型 vs 具体类型的一致性
- [ ] nil 传递风险
- [ ] 初始化顺序依赖
- [ ] 依赖注入中的类型匹配

## 执行计划
1. 搜索所有使用 SM4/SM2/SM3 的地方
2. 检查每个调用点的类型定义
3. 验证初始化链路
4. 记录发现的问题
5. 如发现严重问题，立即修复

## 输出
- 问题清单（如有）
- 修复建议（如有）
- 审查报告
