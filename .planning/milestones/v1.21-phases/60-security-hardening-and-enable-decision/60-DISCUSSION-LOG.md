# Phase 60: 安全加固与启用决策 - Discussion Log

**Gathered:** 2026-08-13
**For:** Human reference (audits, retrospectives). NOT consumed by downstream agents.

## Areas Discussed

### Area 1: AUTH-03 MultiAuth 启用决策（4 问）

| Q | Question | Options | Selected |
|---|----------|---------|----------|
| 1 | 顶层启用决策 | 启用全量挂载 / 推迟 / 子集启用 | **启用 — 全量挂载** |
| 2 | 挂载范围 | 仅 apikeys / 所有 JWT 模块 / apikeys+monitor | **仅 /system/apikeys/* 管理面** |
| 3 | InheritPerms / 优先级 | X-API-Key 优先+JWT 回退 / JWT 优先+API Key 备选 / 都处理以 API Key 为准 | **X-API-Key 优先+JWT 回退** |
| 4 | IP 白名单语义 | 严格拒绝 / 仅记录告警 / 禁用 | **启用 — 严格拒绝** |

**Follow-up notes:** D-01 决策直接触发 Phase 61（资源级权限矩阵 + 限流调优）立即执行；Phase 61 不再 conditional。

---

### Area 2: SEC-01 API Key 哈希存储决策（4 问）

| Q | Question | Options | Selected |
|---|----------|---------|----------|
| 1 | 顶层方案（首问被质疑） | SM3-PBKDF2 / SM3+salt / 保留明文 | 用户质疑「为什么 SM3」→ 重新澄清国密栈用法 |
| 1' | 重新澄清后 | SM4 对称加密 / SM3 单向哈希 / 保留明文 | **迁移到 SM3 单向哈希** |
| 2 | List 搜索能力保留 | KeyHash+Name 双字段 / 只按 Name / 碰撞检测列 | **保留 — KeyHash+Name 双字段** |
| 3 | 平滑过渡 | 双读期+回填 / 一次性迁移 / 旧 key 保留明文 | **不需要迁移（用户确认无活跃 key）** |
| 4 | 创建/轮换流程 | 一次性返回明文 / 列表仍可查 | **创建一次性返回明文** |

**Follow-up notes:** 用户偏离推荐——我推荐 SM4 对称加密，用户选 SM3 单向哈希。理由：① API Key 高熵无需拉伸；② DB + salt 泄漏后无法还原；③ 不依赖 SM4_KEY 保护。「现在没有使用中的 api key」是关键事实，简化了迁移路径（无双读期、无回填）。

---

### Area 3: SEC-02 冗余索引移除实施（2 问）

| Q | Question | Options | Selected |
|---|----------|---------|----------|
| 1 | migration 路径 | 新建 migration_086 / 手动 SQL / 修改 085 | **手动运维 SQL**（用户偏离推荐） |
| 2 | 验证/交付形式 | SQL+文档+验证查询 / 仅 SQL+验证 / 仅 CHANGELOG | **SQL+文档+验证查询** |

**Follow-up notes:** 用户偏离推荐——我推荐新建 migration_086，用户选手动 SQL。理由：决策快路径 + 运维可控。delivery = `docs/operations/sql/2026-08-13-drop-idx-api-keys-key.sql` + `.planning/notes/260813-sec02-redundant-index-removal.md`。

---

### Area 4: QUAL-01 限流头编码修复范围（2 问）

| Q | Question | Options | Selected |
|---|----------|---------|----------|
| 1 | 修复范围 | 仅 strconv.Itoa / 同时修 QUAL-03 | **只修 QUAL-01 限流头编码**（严格 Phase 60 scope） |
| 2 | 实现方式 | strconv.Itoa+单测+集成测试 / 仅单测 / FormatInt | **strconv.Itoa+单测+集成测试** |

**Follow-up notes:** `getScopeFromContext` 多 scope 选择（line 285-304）属 QUAL-03 范畴严格留 Phase 61，避免与 Phase 61 资源权限设计冲突。

---

## Deferred Ideas

- 资源级权限矩阵 → Phase 61 / AUTH-04（unconditional since AUTH-03=启用）
- 限流生产调优 + getScopeFromContext 多 scope 选择 → Phase 61 / QUAL-03
- username 语义修正 → Phase 61 资源权限领域
- 密钥轮换/吊销、配额告警 → FUTURE-APIKEY-03/04（仍 v2 Future）

## Claude's Discretion Items

- SEC-01 SM3 哈希格式是否带版本前缀（建议保持简单，仅 hex 无前缀）
- Salt 长度（建议 16 字节）
- 新 schema 列在 models/api_key.go 中的位置
- 验证查询的 SQL 形式（PG vs SQLite）