# 技术债务全量审计 findings（2026-09-07，gsd-audit-fix）

> 审计来源：`/gsd-progress --do 完整检查项目完成情况/硬编码/TODO/重复实现`
> 已修复：F-01~F-05（commit 2154cd5 / de1baf0 / 8d27ebe / 64ee1ff / cf3d488，全量 go test 73 包 ok + 前端 type-check 过）。
> 本文件 = 未修复项的完整清单，供下个 milestone 规划（候选 phase：tech-debt 治理）。

## Not-attempted（auto-fixable 但超出本次 --max 5 预算）

| # | Finding | Severity | 位置 |
|---|---------|----------|------|
| F-06 | captcha.go storageKey 内联 6 处重复（`captcha:data:%s` / `login:fail:%s`） | high | internal/core/captcha.go:297,326,361,367,419,426,503,529 |
| F-07 | 其余 status 字面量 12 处（job/vdi/duty/workorder：`[]int{0,1}`、`.Update("status", 0)`、`Status: 0 // 成功` 等） | medium | internal/services/workorder/base.go:183、internal/scheduler/cron.go:43,62,235,407,435,832、internal/scheduler/vdi_sync_tasks.go:48,85、internal/services/scheduler/job_service.go:331、workorder_tasks.go:195,328,451、reconciliation_tasks.go:196、mac_history_tasks.go:127、mac_history_matview_tasks.go:56 |
| F-08 | cache key 内联 ~35 处（notice/settings/duty/workorder/kb/network/api_endpoint/widget/excel/dropdown 等，多数文件模块级未注册 cache_keys.go） | medium | 见 Agent 扫描清单 C-03~C-12 |
| F-09 | 分页双 normalize：internal/utils/pagination.go ParsePagination（cap=MaxListPageSize）与 pkg/query.NormalizePagination 口径分叉 | medium | internal/utils/pagination.go:15-29 |

## Manual-only（需决策/重构）

- **F-10 (high) 缓存闭包残留**：mac_history_query_service.go 4 处 legacy `GetOrSet(func() (interface{}, error){})` + 手写 cache-aside（:259-281,:307,:432,:835；heatmap_service.go:118）；asset/reconciliation_service.go:799-820 GetByWorkstation 手写读穿透；rpa/selector_learner.go:169-174,226 手写 JSON cache-aside。均应收敛 base.GetOrSetJSON（在 invariants test 扫描口径外，零守护）。
- **F-11 (high) response helper wire 契约分叉**：operations/base_handler.go:10-25 本地 handleJSONBinding/handleServiceError 与 pkg/response/handler_helpers.go 并存且错误码语义不同（CodeParamError/CodeServerError vs http 400/500+BusinessError 409）。收敛需定 wire 契约方向。
- **F-12 (medium) handler CRUD 样板复制**：operations 14 个 handler 同构（wall vs server_room 逐行同构；server_room List/Statistics 已漂移手写错误路径 :101-103,:37）；monitor oper_log_handler.go:54-160 vs login_log_handler.go:49-157 五方法复制（login 全手写 response.Error）。
- **F-13 (medium) 前端 19 处手写 CRUD** 绕过 apiFactory（adDomainApi×3/knowledgeApi×5/dutyApi×5/workorderApi×6，D-14 单参 delete 契约白名单已登记）。
- **F-14 (medium) 前端选项/颜色映射重复**：server-rooms/index.tsx:653,417 + MACHistoryPage.tsx:654 内联正常/停用；fix-suggestion fixStatusColor/Label 两份已漂移（index.tsx:68-75 vs FixSuggestionDetailDrawer.tsx:22-29）；11 处内联 `status === 0 ? "success" : "error"` 三元 Tag（DashboardList:215、assets:387、FloorCardView:67、menu:110、email-config:268、api-config:252、FloorView3D:90、BuildingView3D:183、ad-domain/configs:253,474、exception-rules:285），颜色 token success/green 混用需视觉决策统一。
- **F-15 (medium) TODO 功能缺失占位**：Go ~16 处（workorder 评价功能 workorder_router.go:65、rpa 扩缩容配置 worker_handler.go:260,279、ListSessions credential_handler.go:154、解锁用户 login_log_handler.go:194、回滚逻辑 error_handling.go:311、selector_learner.go:235,367、agent 动态更新 handlers.go:304、config 缓存刷新 config_service.go:257、device_discovery_service.go:662、task_service.go:296、core.go:911、init_data.go:638 未使用函数、oper_log_handler.go:213、reconciliation_exception.go:578）；前端 6 处（LayoutToolbar:132,139、useGeocoding:131、WorkstationView:73、assets 编辑:582、AIScriptEditor:97）。
- **F-16 (medium) 测试 skip**：core/security 15 处 t.Skip（等真实 LDAP+DB 环境 / HybridAuthenticator interface refactor：ad_authenticator_test ×5、hybrid_authenticator_test ×5、authenticator_test、user_sync_service_test 等）。
- **F-17 (low~medium) 前端类型卫生**：`as any` 11 处/10 文件（HubeiMap:490、HubeiMapGL:481、login:28、ExcelImport:261、FileUpload:259、WidgetRenderer:41、assets:255、EditModal:182 等）；72 处无理由 eslint-disable（VirtualMachineList ×5、useRoleActions ×3、buildings/info-points/mac ×6 等约 31 文件）。@ts-ignore/@ts-nocheck 全域 0 处。

## Skip / 观察项（无需处理或有意设计）

- **captcha-background 1=启用非 bug**：models.CaptchaBgEnabled=1（gorm default:1，QUIRK-80-03-D 锁定），前端 value:1+启用 与后端一致。禁止套用 0=启用共享常量修它。
- **超时字面量 ~120 处**：绝大多数为模块私有命名常量（约定允许）；pkg/constants 已收编值无重复违规。
- **agent/server 裸 c.JSON**（handlers.go 9 处，疑为 agent 协议有意）；**三层 adapter 并存**（cache_adapter/adapter/data_cache_service，架构性）；**协议默认值**（agent/config.go:25、cmd/agent/main.go:29、rpa-worker config.go:211、baidu API URL、cors TrimPrefix）；**geocoding baiduResp.Status != 0**（百度 API 返回码白名单）。
- **rpa/data_mapper.go:332 nilness "impossible condition: non-nil == nil"**：LSP 新报的预存问题，建议下轮 debug 排查。

## 已修复（本轮，供追溯）

| # | Commit | 内容 |
|---|--------|------|
| F-01 | 2154cd5 | 分页字面量 53 处/26 文件 → constants.DefaultCurrent/DefaultPageSize/MaxListPageSize |
| F-02 | de1baf0 | AD/user status SQL 字面量 7 处/5 文件 → models.ADConfigStatusEnabled/UserStatusEnabled |
| F-03 | 8d27ebe | config cache key 5 处 → cache_keys.go 注册表（新注册 CacheKeyConfigAll） |
| F-04 | 64ee1ff | docker URL "http://" → constants.HTTPProto |
| F-05 | cf3d488 | buildings 卡片停用态 label "1" → "停用"（用户可见 bug） |
