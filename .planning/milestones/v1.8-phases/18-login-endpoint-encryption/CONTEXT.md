# Phase 18: 登录端点请求体加密 - Context

## Milestone
**v1.8 - 登录端点加密增强**

## Goal
为登录端点 `/system/auth/login` 启用 SM2+SM4 混合请求体加密，实现三层加密保护。

## Current State

### 登录加密现状
```
Layer 1: HTTPS (TLS) - 传输层加密 ✅
Layer 2: SM2 密码加密 - 字段级加密 ✅
Layer 3: SM2+SM4 请求体加密 - 当前禁用 ❌
```

### 当前配置

**后端 `config.yaml`:**
```yaml
exclude_paths:
  - "/api/v1/system/auth/login"       # 当前排除登录端点
```

**前端 `api.ts`:**
```typescript
const ENCRYPTION_BLACKLIST = [
    '/system/auth/login',  // 当前在黑名单
    '/system/auth/public-key',
    '/system/auth/captcha',
    '/system/auth/encryption-config',
    '/upload',
];
```

### 中间件执行顺序
```
RequestID → RequestDecryption → ResponseEncryption → Router
                                              ↓
                                        /system/auth/login (无JWT认证)
```

**关键发现：登录端点在JWT认证之前，无"鸡生蛋"问题！**

## Research Questions

1. **安全分析**
   - 登录端点启用请求体加密的安全收益是什么？
   - 是否存在新的安全风险？

2. **性能影响**
   - SM2+SM4加密对登录性能的影响有多大？
   - 需要考虑哪些优化措施？

3. **实现范围**
   - 需要修改哪些配置文件？
   - 需要修改哪些代码文件？
   - 后端登录处理器需要调整吗？

4. **兼容性**
   - 如何向后兼容旧的客户端？
   - 是否需要配置开关？

5. **测试策略**
   - 需要哪些测试用例？
   - 如何验证三层加密都正常工作？

## Dependencies
- Phase 17 (前后端加密配置同步) - ✅ 已完成
- SM2/SM4 国密算法库 - ✅ 已集成

## Success Criteria
1. 登录请求经过SM2+SM4请求体加密
2. 密码仍经过SM2字段级加密（双层加密）
3. 性能影响可接受（< 100ms 增加延迟）
4. 向后兼容旧客户端
5. 测试覆盖率 > 80%
