---
quick_id: 261110-jci
status: complete
created: 2026-11-10
completed: 2026-11-10
scope: B-1 遗留的 config 契约漂移（前端适配后端 dotted-key map / 删 webhookTemplate 死控件 / 修 updateConfig 400）
commit: false  # 待用户授权提交（用户此前已授权提交，但本轮按 CLAUDE.md 提交前确认纪律留白）
decision_source: 用户 2026-11-10 选择「1 修 B-1 config 契约漂移」；沿用 v1.56 草案「Phase 152 后续项」口径
---

# Quick 261110-jci · SUMMARY

用户裁定「修 B-1 config 契约漂移」。据 `261110-jci-PLAN.md`，本轮**只清这一项**，零后端改动、零 migration、零生产数据、C 类 79 条保持 0 勾选。

## 根因（实证）

后端 `GET /config` 返回 `DriftAlertConfigView = map[string]string`（9 个 `ad_sync.drift_alert.*` dotted key，值全 string），`POST /config/update` 用 `ShouldBindJSON(&map[string]string)` + `adDriftConfigKeys` 白名单消费同一形状（`ad_sync_drift_alert_query.go:61,206-269`、`ad_drift_alert_handler.go:145-157`）。前端却是 camelCase 类型化对象 + 虚构的 `webhookTemplate`（后端全仓对该旋钮 0 命中）→ 读取全落空、保存必 400。

## 改动（5 个前端文件，零后端）

### `src/lib/adDriftAlertApi.ts`
- 新增 `AD_DRIFT_CONFIG_KEYS`（9 个 dotted key，与后端 `adDriftConfigKeys` 逐字一致，前端唯一映射入口）+ `ADDriftAlertConfigView = Record<string,string>`
- 新增 `parseConfigView`（string→boolean/number 安全回落）与 `toConfigPayload`（camelCase→dotted-key 全 string，只发白名单 9 key）
- `getConfig()` 改 `post<ADDriftAlertConfigView>` 再转换；`updateConfig()` 先转 payload 再 `post<{message}>`，返回合并后的表单对象供页面刷新
- `ADDriftAlertConfig` 删 `webhookTemplate`；`baseline` 保留（只读回显，随提交原样带回，白名单含该 key）

### `src/pages/system/ad-drift-alert/components/ConfigForm.tsx`
- 删 `webhookTemplate` 全链路（ConfigFormValues / 回填依赖 / initialValues / 提交 payload / JSX Form.Item / `AppTextArea` import）
- 头注释改为「8 可写旋钮 + baseline 只读」，声明适配统一在 api 层
- 其余（webhook 下拉 / 三态门控 / 唯一错误出口 / useEffect 稳定依赖）一字不动

### 3 份测试
- `src/lib/__tests__/adDriftAlertApi.test.ts`：mock 的 `data` 改真实 dotted-key view；断言 `getConfig` 返回 camelCase 类型化对象、`updateConfig` 发 9 个 dotted-key 全 string（不发 camelCase/不发 400）、死字段已删；新增 5 个 `parseConfigView`/`toConfigPayload`/往返一致/边界用例
- `.../__tests__/components.test.tsx`：configFixture 删 `webhookTemplate`，「Webhook 模板」文本断言改为 `queryByText` 反向断言，保存 payload 补 `not.toHaveProperty("webhookTemplate")`
- `.../__tests__/index.render.test.tsx`：getConfigMock 删 `webhookTemplate`

## 全量闸门（全绿）

| 命令 | 结果 |
|---|---|
| `npm run type-check` | ✅ exit 0 |
| `npx vitest run` 3 文件 | ✅ 3 files / 33 tests passed |
| `npm run lint:ci` | ✅ **ALL PASS**（C13 见下方隔离说明） |
| `npx prettier --check` 5 改动文件 | ✅ all matched（首轮 `adDriftAlertApi.ts` 已 `--write` 修复） |
| `go build ./...` | ✅ exit 0 |
| `go test -run TestPhase144_(GetConfig\|UpdateConfig) ./internal/services/addomain/` | ✅ ok（后端契约单测零回归） |

### lint:ci 的 C13 隔离说明（重要，非本任务缺陷）
工作区存在**另一个并发任务**的未提交改动：`src/index.css`（+12 行 Excel 导入提示样式）与 `src/components/shared/ExcelImport.tsx`（+1 行），使 `index.css` 行数 3219→3231，触发 `css-contract-check.js` 的 C13 ratchet（M9=3232>3220）。**这两个文件与本 quick 无关，未回滚/未修改/未纳入提交**。用 `git stash push -m ... -- <两文件>` 暂存后重跑 `lint:ci` = ALL PASS，随后 `git stash pop stash@{0}` 完整恢复（index.css 回到 3231 行、ExcelImport.tsx 改动仍在）。C13 是该并发任务自己需要处理的预算校准，不伪造本轮已覆盖。

## 硬约束复核
- ✅ Handler-Service / wrapped `post` / 唯一错误出口 / useEffect 稳定依赖
- ✅ 零后端改动、零 migration、零新依赖、零生产数据
- ✅ C 类 79 条保持 0 勾选、不代执行、不代勾选
- ✅ 未执行 `git add` / `git commit`（提交前待用户授权；工作区另有并发 WIP，须按文件面精确提交）

## 清偿结论
`STATE.md` v1.54 ① 行「config 形状漂移留 152 后续」→ **本 quick 已清偿**：B-1 前端契约与已读 UI（261110-f5g）+ config 双向适配（本 quick）全部闭环，`webhookTemplate` 死控件删除，`updateConfig` 不再 400。B-2 仍按用户裁定暂缓、B-4 自动回滚本体按推荐不做（诚实性短板 261110-161 已补）。
