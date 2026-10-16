# Phase 17: 前后端加解密配置同步

## 触发背景

在参数管理中将请求加密开关设置为 false 后，出现加解密不一致问题：

1. **请求加密不匹配**：前端仍发送加密请求，后端不解析，导致 JSON 绑定失败（400 错误）
2. **响应加密缺失**：后端响应未加密，但前端可能尝试解密，或前端未解密但后端实际已加密

**根本原因：**
- 前端使用**构建时**环境变量 `VITE_ENABLE_REQUEST_ENCRYPTION` 控制加解密
- 后端使用**运行时**数据库配置 `sys.request.encryption.enabled` 控制加解密
- 响应加密中间件未启用（`ResponseEncryption` 的 `config.Enabled` 为 false）
- 配置不一致导致前后端加解密行为不同步

## 目标

实现前后端加解密配置的统一同步机制：
1. **请求加密同步**：前端动态读取后端加密配置，决定是否加密请求
2. **响应加密同步**：后端根据数据库配置决定是否加密响应，前端根据配置决定是否解密
3. **统一配置源**：由参数管理页面的"请求加密开关"统一控制前后端加解密行为

## 约束条件

- 必须向后兼容现有构建时配置（`VITE_ENABLE_REQUEST_ENCRYPTION`）
- 不应影响已部署的生产环境
- 需要处理配置变更时的状态同步
- 必须保证 token 刷新等关键端点的可靠性

## 技术栈

- Backend: Go 1.24, Gin, GORM, PostgreSQL
- Frontend: React 19.2, TypeScript, Vite, Zustand
- Security: SM2/SM3/SM4 国密算法
