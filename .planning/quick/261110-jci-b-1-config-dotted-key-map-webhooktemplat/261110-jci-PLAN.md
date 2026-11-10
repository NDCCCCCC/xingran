---
quick_id: 261110-jci
status: complete
created: 2026-11-10
scope: B-1 遗留的 config 契约漂移（前端适配后端 dotted-key map / 删 webhookTemplate 死控件 / 修 updateConfig 400）
commit: false  # 待全量闸门后由用户授权提交
decision_source: 用户 2026-11-10 选择「1 修 B-1 config 契约漂移」；沿用 v1.56 草案「Phase 152 后续项」口径
---

# Quick 261110-jci · B-1 config 契约漂移修复

## 0. 根因（实证，非推测）

| 面 | 现状 | 证据 |
|---|---|---|
| 后端 `GET /config` | `DriftAlertConfigView = map[string]string`，9 个 `ad_sync.drift_alert.*` **dotted key**，值全为 string | `ad_sync_drift_alert_query.go:61,206-244` |
| 后端 `POST /config/update` | `ShouldBindJSON(&map[string]string)` + `adDriftConfigKeys` 白名单，非白名单 key 静默忽略 | `ad_drift_alert_handler.go:145-157`、`ad_sync_drift_alert_query.go:249-269` |
| 前端 `ADDriftAlertConfig` | camelCase 类型化对象（boolean/number），字段名与后端 key 完全不同 | `adDriftAlertApi.ts:79-98` |
| `webhookTemplate` | 后端**不存在**（全仓 Go 对 `webhook_template` / `WebhookTemplate` 0 命中，migration 238/239 无该 key），纯前端死控件 | grep 实证 + `ConfigForm.tsx:241-246` |
| 后果 | ① `getConfig()` 返回 `{...}` 但前端按 `.enabled` 读 → 全 undefined、表单回填全落空<br>② `updateConfig(payload)` 发 camelCase 对象 → 后端 `ShouldBindJSON` 反序列化失败或白名单忽略 → **400 / 配置永不生效** | 契约逐字段比对 |

用户 261110-f5g 已把 logs 契约对齐、已读 UI 做完，唯独 config 表单按「调试开始于具体报告问题、不擅自扩大范围」留作 Phase 152 后续项（`261110-f5g-SUMMARY.md:47`）。本 quick 只清这一项。

## 1. 方案（前端适配后端，零后端改动）

**不改后端**：后端 9-key dotted map + 白名单是既有稳定契约（有 `TestPhase144_GetConfig_DefaultsAndOverride` / `TestPhase144_UpdateConfig_WhitelistAndIdempotent` 守护），改后端响应形状会破坏 `/config` 消费方并扩出 migration/权限面，违反最小改动。

**前端做适配层**（`adDriftAlertApi.ts` 内部）：
- `ADDriftAlertConfigView = Record<string,string>`（后端原文形状，供 `getConfig` 返回）
- `ADDriftAlertConfig`（表单/页面用的 camelCase 类型化对象）保留，但**删 `webhookTemplate`**、字段与后端 8 个可写 key 一一对应
- `getConfig()`：`post<ADDriftAlertConfigView>` → `parseConfigView()` 转 camelCase（含 string→boolean/number 安全解析）
- `updateConfig(config)`：`toConfigPayload()` 转 9 个 dotted key 的 `Record<string,string>`（含 `baseline` 只读回显也带值，与后端白名单一致；`webhookConfigId` 等空串照发）→ `post<{message:string}>`
- `parseConfigView` / `toConfigPayload` 导出，供测试直接覆盖「string↔类型化」边界（空串/undefined/非法数值回落默认）

**ConfigForm.tsx**：
- `ConfigFormValues` 删 `webhookTemplate`；删 `Webhook 模板（可选）` Form.Item 与 `AppTextArea` import
- 回填依赖/initialValues 删对应字段；`handleSubmit` payload 删 `webhookTemplate`
- `baseline` 仍从 `config` 只读展示，**提交时仍带上**（后端白名单含 `ad_sync.drift_alert.baseline`，与 `defaultADDriftConfigView` 9 行口径一致）；但控件 `readOnly` 不变
- UI 其余（webhook 下拉 / 三态门控 / 唯一错误出口 / 稳定依赖）一字不动

**明确不做**：不改后端、不加 migration、不动 `TriggerLogTable`/`StatusCards`、不动 79 条跟踪表、不代执行 UAT。

## 2. 任务

### Task 1 · api 契约适配（文件：`xingran-react-frontend/src/lib/adDriftAlertApi.ts`）
- 新增 `ADDriftAlertConfigView = Record<string,string>` + 9 个 dotted key 常量（与后端 `ad_sync.drift_alert.*` 逐字一致）
- 重写 `getConfig`/`updateConfig`/`ADDriftAlertConfig`：删 `webhookTemplate`，补 `parseConfigView`/`toConfigPayload`
- `verify`：`npm run type-check` exit 0

### Task 2 · ConfigForm 去死控件 + 适配（文件：`.../components/ConfigForm.tsx`）
- 删 `webhookTemplate` 全链路（values / 回填 / 提交 / JSX / import）
- `verify`：`npm run type-check` exit 0

### Task 3 · 测试同步（3 文件）
- `src/lib/__tests__/adDriftAlertApi.test.ts`：断言 `getConfig` 返回 camelCase 转换结果、`updateConfig` 发 **dotted-key 全 string** payload（不再发 camelCase 对象）、`webhookTemplate` 不再出现、补 `parseConfigView`/`toConfigPayload` 边界用例
- `.../__tests__/components.test.tsx`：configFixture 删 `webhookTemplate`，去掉「Webhook 模板」文本断言，保存断言改判 dotted-key payload
- `.../__tests__/index.render.test.tsx`：getConfigMock 返回值改 dotted-key view（真实形状），供编排测试
- `verify`：`npx vitest run` 3 文件全绿

### Task 4 · 全量闸门 + 文档
- `npm run type-check` / `npx vitest run`(3 文件) / `npm run lint:ci` / `go build ./...`
- 写 `261110-jci-SUMMARY.md`；更新 `STATE.md` Quick Tasks 表（用 `gsd_run quick-tasks-append`）+ v1.54 ① 行清偿注记；同步 `MILESTONES.md` / `PROJECT.md` 的 11-10 过时口径（上一轮发现的文档滞后，顺手清）

## 3. 硬约束
- 沿用 Handler-Service / wrapped `post` / 唯一错误出口 / useEffect 稳定依赖
- 零新依赖、零 migration、零后端改动、零生产数据
- C 类 79 条保持 0 勾选、不代执行、不代勾选
- 未执行 `git add` / `git commit`，提交前待用户授权

## 4. 完成判据
- `updateConfig` 发出的 payload 是 9 个 `ad_sync.drift_alert.*` key 的 `Record<string,string>`，后端 `ShouldBindJSON(&map[string]string)` + 白名单可原样消费
- `getConfig` 能把后端 map 正确还原成表单值（string→boolean/number）
- `webhookTemplate` 在前端源码/测试中 0 命中
- 全量闸门全绿；工作区可提交（待用户授权）
