---
slug: kb-tag-table-stats-400
status: awaiting_human_verify
trigger: 前端知识库页报 "查询标签列表失败: SQL logic error: no such table: sys_knowledge_tag"（/knowledge/tags/all 500），运维楼宇页报 "加载统计数据 Error: 解密失败"（/ops/building/statistics 400 + 前端 SM2 解密失败清除公钥缓存）；另有 useForm 未连接 / antd Drawer width / Alert message 弃用警告。
created: "2026-08-18T01:00:00Z"
updated: "2026-08-18T01:18:00Z"
---

# Debug Session: kb-tag-table-stats-400

## Symptoms

1. **Expected behavior**: SQLite 模式（`database.type: "sqlite"`，延续上一会话环境）下，知识库文章页标签列表正常加载（`/api/v1/knowledge/tags/all` 200）；运维楼宇页统计数据正常加载（`/api/v1/ops/building/statistics` 200）。
2. **Actual behavior**:
   - `/api/v1/knowledge/tags/all` → 500，GORM: `SQL logic error: no such table: sys_knowledge_tag (1)`，SQL 为 `SELECT * FROM sys_knowledge_tag ORDER BY use_count DESC, created_at ASC`。前端两处消费方报错（useArticleData.ts:154、index.tsx:87），页面每次加载触发两次。
   - `/api/v1/ops/building/statistics` → 400 Bad Request；前端 errorHandler.ts:206 "SM2 解密失败，清除公钥缓存" → errorHandler.ts:219 抛 "解密失败"（调用栈 opsApi.ts:79 → index.tsx:146）。
   - 同一时刻 `/knowledge/categories/list` 200（含 L2 缓存写入 kb:category:tree）、`/knowledge/articles/search` 200 —— 知识库其余表存在且正常。
   - 次要（可能不在本次范围）：React 警告 useForm 未连接 Form（index.tsx:791）、antd Drawer `width` 弃用（index.tsx:525）、antd Alert `message` 弃用（index.tsx:136）。
3. **Error messages**（逐字摘录）:
   - 后端: `ERRO ... [GORM错误] SELECT * FROM `sys_knowledge_tag` ORDER BY use_count DESC, created_at ASC | 耗时: 0s | 错误: SQL logic error: no such table: sys_knowledge_tag (1)`（00:55:39 与 00:55:43 两次）
   - 后端: `ERRO ... Internal server error ... path=/api/v1/knowledge/tags/all ... status_code=500`
   - 前端: `POST http://127.0.0.1:4000/api/v1/ops/building/statistics 400 (Bad Request)`
   - 前端: `[ErrorHandler] SM2 解密失败，清除公钥缓存` / `[加载统计数据] Error: 解密失败`
4. **Timeline**: 2026-08-18 00:55:39-43 持续发生（每次页面加载必现）。关键背景：紧接 resolved 会话 `ops-sqlite-tables-uuid-cast`（00:48 完成，**修复尚未提交**，即当前工作区未提交改动）之后；后端已重启且上一会话的 ops 列表批量 500/400 已全部消失 —— 本症是重启后的剩余问题，非上一会话回归。
5. **Reproduction**: sqlite 模式后端 + 前端 dev server（127.0.0.1:4000 代理），打开知识库文章页（自动触发 tags/all×2）与运维楼宇页（自动触发 statistics）即复现。

## Evidence

- timestamp: 2026-08-18T00:58:00Z — orchestrator 预查: `internal/core/db/database.go:541` 注释 `// KnowledgeCategory, KnowledgeTag 已通过 SQL 迁移创建，不需要 AutoMigrate`，542-543 行仅注册 `&models.KnowledgeArticle{}` / `&models.KnowledgeArticleTag{}`；该"SQL 迁移建表"假设只在 PG 成立，sqlite 分支不执行归档迁移 → `sys_knowledge_tag` 永远缺表（与 resolved/ops-sqlite-tables-uuid-cast 根因 A 同类：AutoMigrate 漏注册）
- timestamp: 2026-08-18T00:58:00Z — KnowledgeTag 模型定义存在于 `internal/models/knowledge.go`（grep 命中）
- timestamp: 2026-08-18T00:58:00Z — 矛盾点待查: categories/list 200 说明 `sys_knowledge_category` 在 sqlite 库**存在**，与 541 注释"已通过 SQL 迁移创建"矛盾 —— 需查明 category 有表而 tag 没有的确切原因（可能 category 在别处注册，或 542 行列表实际作用于 sqlite 分支而注释失效）
- timestamp: 2026-08-18T00:58:00Z — building Statistics handler 位于 `internal/api/v1/operations/building_handler.go:35`（`h.service.Statistics(ctx, params)`）；400 = 参数绑定/校验/中间件层错误（SQL 错应为 500）；且后端日志片段中**未见**该请求记录，前端却在响应处理阶段报 SM2 解密失败（errorHandler.ts:206/219）—— 需确认 400 响应体是否加密与前端解密期望不匹配
- timestamp: 2026-08-18T00:58:00Z — 环境限制（上一会话同样命中）: Grep/Glob 专用工具在本环境报 "Executable not found"（claude.exe 路径失效），需用 bash grep/rfind 替代；gsd-sdk 不可用（exit 127）
- timestamp: 2026-08-18T00:58:00Z — 工作区状态: 上一会话修复（database.go + database_test.go + operations 6 个 service）未提交，本会话任何 fix 不得回滚/覆盖这些改动；internal/services/operations 既有 8 个 excel_service 预存在失败（见上一会话 verification #4），非回归基线
- timestamp: 2026-08-18T01:02:00Z — sqlite 实库确认（sqlite3 CLI）: data/xingran.db 中 `%knowledge%` 表仅有 sys_knowledge_category + sys_knowledge_article;另有 sys_kb_article_tags（join 表）。`sys_workorder` / `sys_workorder_category` 也存在 —— 而这两者在 MigrateModelList 中同样未注册（532 行注释同款"SQL 迁移创建"）。备份 xingran.db.bak-260817-menus 同状态 → 非近期回归，DB 生成以来一直如此
- timestamp: 2026-08-18T01:05:00Z — 机制定位（GORM v1.30.5 migrator.go 源码 + 实证）: AutoMigrate→ReorderModels(values, autoAdd=true) 的 parseDependence 中，**belongs-to 依赖**（KnowledgeArticle.Category→KnowledgeCategory、SourceWorkOrder→WorkOrder）在 `valuesMap[dep.Schema.Table] = dep` **之前**直接 append 进 dep.Depends → 级联建表成功（category/workorder/workorder_category 全部由此而来）；而 **many2many 远端**（KnowledgeArticle.Tags→KnowledgeTag）的 append 位于 defer 闭包，在 map 存值**之后**才执行，slice 增长后 map 中的旧副本看不到 → KnowledgeTag 被静默丢弃，永不建表。join 表本身（sys_kb_article_tags）经 parseDependence(joinValue, autoAdd) 直接注册故存在
- timestamp: 2026-08-18T01:06:00Z — **全新 sqlite 库实证复现**（临时探针测试 TestTmpProbeCascadeTables, 用后即删）: 当前代码 AutoMigrate 后 HasTable: sys_knowledge_article=true / sys_kb_article_tags=true / sys_knowledge_category=true(未注册却存在) / **sys_knowledge_tag=false(BUG 复现)** / sys_workorder=true(未注册) / sys_workorder_category=true(未注册) —— 与生产库状态完全一致，根因(1)确认

- timestamp: 2026-08-18T01:10:00Z — **症状(2)后端日志实证**（logs/app.log，orchestrator"无请求记录"结论有误——日志有两条记录）:
  - line 117384: `{"data_length":24,"error":"解密失败","level":"warning","msg":"解密请求体失败","path":"/api/v1/ops/building/statistics","time":"2026-08-18 00:55:09"}` + line 117385 同请求 status_code=400（"Client error"）——400 来自 `pkg/middleware/request_decryption.go:160`（`DecryptRequestWithKeyInfo` 失败 → `response.Error(c, ErrBadRequest, "解密失败")`），非参数绑定非路由
  - 同秒 `/ops/building/list` 解密成功（117386-117389）；statistics 全日志 5 次调用 4 成功 1 失败 → **非确定性、非路径专属**
  - 全日志仅 2 次"解密请求体失败": statistics@00:55:09 与 `/ad-domain/configs/list`@01:05:47；后者 01:06:15 同端点成功
  - 后端启动标记: 00:37:27 / 00:49:29 / 01:05:38 —— 两次解密失败均紧跟重启（01:05:47 失败距 01:05:38 重启 9 秒）
- timestamp: 2026-08-18T01:12:00Z — 症状(2)机制链确认: `configs/config.yaml` jwt.use_sm2=true 且 sm2_private_key/sm2_public_key 为空 → `internal/core/security/jwt.go:86` 每次启动 `crypto.GenerateKeyPair()` 动态生成 → **后端重启即更换公钥**；前端公钥缓存为模块内存变量（sm2.ts:24 `cachedPublicKeyHex`），SPA 长驻页签跨重启持旧公钥 → 旧公钥加密的 SM4 key 后端 `DecryptWithSM2` 解不开 → 400"解密失败"。errorHandler.ts:205-208 收到 400+message含"解密" → 清公钥缓存（下次请求取新 key 自愈）但**本请求已失败上抛页面**
- timestamp: 2026-08-18T01:12:00Z — 症状(2)"前端 SM2 解密失败"定性: 前端**并未尝试解密该错误响应**——errorHandler.ts:206 的 console 文案 "SM2 解密失败，清除公钥缓存" 是对后端 message 的启发式响应（清缓存自愈），抛出的 Error("解密失败")(219 行) 文本直接来自后端响应体 message 字段。用户感知的"解密失败"= 后端请求解密拒绝，非前端密码学故障
- timestamp: 2026-08-18T01:12:00Z — 401 token 刷新重试先例: api.ts:397-464 错误拦截器已有"401→刷新→重放原请求"模式；症状(2)的 400"解密失败"仅有 errorHandler 清缓存（sm2.ts:116），无请求级重放 → 页面仍报错，需手动刷新

## Eliminated

- hypothesis: sys_knowledge_category 存在说明 sqlite 分支曾有建表路径（如 service 层 AutoMigrate / init_data / SQL runner）
  evidence: grep 全仓: knowledge 服务/init_data/迁移 Go 文件均无该表 CREATE;归档 .sql(legacy-2026-06-15/012)无任何运行期执行方;dbprovision 注册的是 KnowledgeTag 而非 Category（但 dbprovision 仅服务 PG 新部署,且若跑过 sys_knowledge_tag 应存在）
  timestamp: 2026-08-18T01:05:00Z

## Current Focus

reasoning_checkpoint:
  hypothesis: |
    (1) sys_knowledge_tag 缺表: MigrateModelList 漏注册 KnowledgeTag。541 行注释假设"SQL 迁移已建表"仅 PG 成立（归档 SQL 启动期不执行）；GORM v1.30.5 ReorderModels 对 belongs-to 依赖（Article.Category→Category）在 valuesMap 存值前 append → 级联建表成功（故 category/workorder 表存在）；many2many 远端（Article.Tags→KnowledgeTag）append 在 defer 闭包、map 存值后执行，slice 扩容致旧副本丢失 → KnowledgeTag 永不建表（探针实证 HasTable=false）。
    (2) statistics 400: 后端 SM2 密钥对每启动动态生成（config 密钥为空），重启后前端长驻页签用旧公钥缓存加密 → request_decryption.go:160 拒绝 400"解密失败"；前端只清缓存不重放请求 → 页面报错。
  confirming_evidence:
    - "探针测试（全新 sqlite 库）: sys_knowledge_tag=false, 其余 5 表=true，与生产库完全一致"
    - "GORM v1.30.5 migrator.go:939-965 源码: belongs-to 直接 append vs many2many defer 闭包 append"
    - "app.log: 两次解密失败均紧跟后端重启; statistics 5 调用 4 成功; 同秒 list 成功"
    - "jwt.go:84-93: 配置密钥空 → GenerateKeyPair() 每次启动生成新密钥对"
  falsification_test: "注册 KnowledgeTag 后探针/清单测试 HasTable(sys_knowledge_tag)=true 且 tags/all 200; 前端重放修复后 400 解密失败被自动重试吞掉（用户无感）"
  fix_rationale: |
    (1) sqlite 分支显式注册 KnowledgeCategory+KnowledgeTag —— 直接补上缺的建表事实源，不依赖 GORM cascade 的未定义行为；沿用 ops-sqlite-tables-uuid-cast 的 6 表先例（sqlite-only 注册,PG 零改动）。
    (2) api.ts 错误拦截器对"400+解密+本请求已加密+未重试过"清缓存并重放一次（恢复明文 data 由请求拦截器用新公钥重加密）—— 对齐 401 刷新重试模式，根因（旧公钥）被直接消除，非掩盖症状。
  blind_spots: |
    (1) 未验证 PG 分支对两模型注册的漂移影响——故只动 sqlite 分支; KnowledgeCategory 与 BaseModel 同 KnowledgeArticle（主列表已有）,sqlite 净化需求一致，无新增风险。
    (2) 前端重试后若仍 400（如双重启窗口）→ 落回原错误路径（可接受）; 未跑浏览器 e2e 实测重放链路（依赖 type-check+lint+代码审查）。

## Resolution

root_cause: |
  (1) sys_knowledge_tag 缺表: MigrateModelList 主列表依 541 行注释排除 KnowledgeCategory/
  KnowledgeTag("已通过 SQL 迁移创建")——该假设仅 PG 成立,归档 SQL 启动期不执行。sqlite 下
  sys_knowledge_category 之所以存在,是 GORM v1.30.5 ReorderModels 的 belongs-to 依赖级联
  (Article.Category 在 valuesMap 存值前 append → 建表);而 many2many 远端(Article.Tags→
  KnowledgeTag)的 append 在 defer 闭包、map 存值后执行,slice 扩容致丢失 → sys_knowledge_tag
  永不建表(全新库探针实证复现,与生产库状态完全一致)。
  (2) /ops/building/statistics 400: 产生于 pkg/middleware/request_decryption.go:160 ——
  后端 SM2 私钥解不开请求携带的 SM4 密钥。因 config jwt.use_sm2=true 且密钥对为空,
  jwt.go:86 每次启动 GenerateKeyPair() 动态生成 → 后端重启即换公钥;SPA 长驻页签持有旧
  公钥内存缓存(sm2.ts:24),重启后首个加密请求即被 400"解密失败"拒绝(app.log 全程仅 2 次
  失败,均紧跟重启;statistics 5 调用 4 成功)。前端 errorHandler.ts:205 仅清缓存自愈后续
  请求,本请求仍失败上抛页面;"前端 SM2 解密失败"文案是前端对后端 message 的启发式标注,
  前端并未解密错误响应。
fix: |
  (1) internal/core/db/database.go: sqlite 分支显式注册 &models.KnowledgeCategory{} +
  &models.KnowledgeTag{}(不再依赖 cascade 未定义行为;PG 分支零改动);修正 541 行误导性
  注释;database_test.go TestNewDatabaseSQLite 清单追加 sys_knowledge_category +
  sys_knowledge_tag 回归守护。
  (2) xingran-react-frontend/src/lib/api.ts: 请求拦截器加密前暂存明文请求体
  (__originalPlainData);错误拦截器新增 400"解密失败"单次自动重放(条件: 400+message含
  "解密"+本请求已加密+未重试过)——清公钥缓存、清理旧 X-Request-ID 密钥存储条目、恢复
  明文后重放,请求拦截器用新公钥重新加密。对齐既有 401 token 刷新重试模式;axios 类型经
  module augmentation 扩展两个下划线字段。
verification: |
  go build ./... 通过;TestNewDatabaseSQLite PASS(全新库两表均建出);升级路径探针(生产
  data/xingran.db 副本,前置确认缺表)AutoMigrate 后 sys_knowledge_tag=true → 用户重启后端
  即自愈;internal/core/db 全套测试 PASS;go vet 干净;前端 npm run type-check 通过、
  eslint 0 errors。临时探针文件已删除。待人工验证: 重启后端(激活建表)→ 打开知识库文章页
  tags/all 应 200;后端重启后不刷新前端页签直接操作,首次加密请求应自动重试成功无感。
files_changed:
  - internal/core/db/database.go
  - internal/core/db/database_test.go
  - xingran-react-frontend/src/lib/api.ts
