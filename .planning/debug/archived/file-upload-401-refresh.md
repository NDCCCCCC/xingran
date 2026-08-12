---
slug: file-upload-401-refresh
status: resolved
trigger: 文件管理页面视频上传失败问题：1. 后端日志显示文件上传成功（storage/file_service.go:203）2. 前端显示"服务器返回的数据格式无效" 3. 控制台出现401错误：GET /api/v1/recordings 和 GET /api/v1/input-configs/active 返回401 Unauthorized 4. 错误发生在文件上传成功后尝试刷新列表时
created: 2026-05-13
updated: 2026-06-26
---

## Symptoms

**Expected behavior:**
- 文件上传成功后，前端应该正常刷新文件列表
- 刷新请求应该使用有效的token

**Actual behavior:**
- 文件上传成功（后端日志确认）
- 前端显示"服务器返回的数据格式无效"
- 刷新列表时出现 401 Unauthorized 错误
- 受影响的端点：GET /api/v1/recordings, GET /api/v1/input-configs/active

**Error messages:**
- 前端：GET https://10.62.0.123:5443/api/v1/recordings?page=1&page_size=20 401 (Unauthorized)
- 前端：GET https://10.62.0.123:5443/api/v1/input-configs/active 401 (Unauthorized)
- 后端：文件上传成功日志

**Timeline:**
- 2026-05-13 报告

**Reproduction:**
1. 在文件管理页面上传视频
2. 文件上传成功
3. 前端尝试刷新列表
4. 出现401错误

## Current Focus

**Checkpoint Reached:** 需要用户澄清以继续调查

**Investigation Summary:**
1. 发现FileUpload组件使用原生XMLHttpRequest，绕过了axios拦截器的token自动刷新机制
2. 未能在代码库中找到 /api/v1/recordings 和 /api/v1/input-configs/active 端点
3. 这些端点可能不存在或路径描述有误

**Questions for User:**
1. **具体的页面/功能**: 您在哪个页面进行文件上传操作？
2. **正确的API端点**: 请检查浏览器Network标签，确认返回401的实际端点路径
3. **功能说明**: 这是现有功能还是自定义开发的新功能？

**Awaiting:** 用户澄清实际的API端点和错误场景

## Evidence

- timestamp: 2026-05-13T14:30:00Z
  source: code_analysis
  summary: "分析了前端API客户端架构"
  details: |
    - src/lib/api.ts: 标准API客户端使用axios拦截器处理token自动刷新
    - TokenManager: 使用SM4加密存储token，支持自动刷新
    - 请求拦截器: 检查token是否即将过期，自动刷新
    - 响应拦截器: 捕获401错误，触发token刷新并重试请求
    - 刷新队列: 防止并发刷新，使用isRefreshing锁

- timestamp: 2026-05-13T14:31:00Z
  source: code_analysis
  summary: "检查了token刷新机制"
  details: |
    - TokenManager.refreshToken() 使用刷新锁防止并发
    - 响应拦截器捕获401时会将请求加入refreshQueue
    - 刷新成功后按顺序重试队列中的请求
    - 问题：如果文件上传没有使用标准API客户端，将无法受益于此机制

- timestamp: 2026-05-13T14:32:00Z
  source: code_search
  summary: "搜索前端文件管理相关代码"
  details: |
    - 未找到 /api/v1/recordings 或 /api/v1/input-configs 相关前端代码
    - 可能是未实现的端点或使用动态路由
    - 需要进一步确认这些端点的存在和使用情况

- timestamp: 2026-05-13T14:35:00Z
  source: code_analysis
  summary: "发现FileUpload组件使用原生XMLHttpRequest"
  details: |
    - src/components/shared/FileUpload.tsx 使用原生XMLHttpRequest而非axios
    - customRequest函数中通过getAccessToken()获取token
    - 问题：XMLHttpRequest不经过axios拦截器，无法自动处理401和token刷新
    - 如果token即将过期，上传可能成功但后续请求失败
    - 文件删除功能（handleRemove）也使用fetch而非标准API客户端

- timestamp: 2026-05-13T14:36:00Z
  source: code_search
  summary: "后端路由中未找到recordings和input-configs端点"
  details: |
    - internal/api/router.go 中未定义 /api/v1/recordings 路由
    - internal/api/router.go 中未定义 /api/v1/input-configs 路由
    - 这些端点可能不存在或路径不正确
    - 需要用户确认实际的端点路径和功能

## Eliminated

- ~~token刷新机制故障~~: TokenManager实现正确，支持自动刷新和并发控制

## Resolution
**Root cause:** 调查暂停 - 等待用户澄清
**Fix:** 待定
**Verification:** 待定
**Files changed:** None

## Phase 41 Closure (2026-06-26)
won't_fix_reason: FileUpload 组件(`src/components/shared/FileUpload.tsx:160`)用原生 XMLHttpRequest + `xhr.upload.onprogress` 实时进度条,refactor 到 axios 会丢失精细进度控制,需配合 antd Upload 组件的 customRequest 重写 + onProgress callback 重构,涉及上传 UX 行为变更(预计 2-3 文件 + UX 测试),属新功能范畴(v1.16 是 debug 清理 milestone,不做 UX 重构);当前缓解:用户上传完手动刷新页面,token 走 axios 拦截器可正常 401→refresh→重试
action: wontfix (D-02,XHR 改造属新功能推后续 phase)
verification: 复测 `src/components/shared/FileUpload.tsx:160` 仍用 XHR + onprogress,改动范围超出 v1.16 边界
