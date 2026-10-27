# Phase 68 执行 SUMMARY — P2 一致性债务（颜色/日期/类型收敛）

执行：GSD executor（glm），基于 HEAD 378e9bd1（67 已收口）。每任务独立 commit（父指示覆盖 PLAN「全程不 commit」条款）。

## Commits

| 任务 | Commit | 内容 |
| --- | --- | --- |
| T1 | `2500e4d3` | 新建 `src/components/shared/statusHelpers.tsx`（StatusDomain 6 域 + statusLabel/statusColor/StatusTag，13 单测）；tableHelpers.createStatusColumn / ModernTag.renderStatusTag 内部改调 helper，签名不变 |
| T2 | `f409565d` | datetime.ts 增 FORMAT_DATETIME_MINUTE / FORMAT_DATE_SHORT / FORMAT_TIME_MINUTE 三常量；展示语境收敛 20 文件（含 Dayjs `.toDate()` 适配） |
| T5 | `3520847e` | monitor/logs 16 处 any 清零（SearchFormState/LogFetchParams/processTimeRangeParams 类型化）、ExecutionDetailModal/ad-domain/duty/apikeys/AccountPoolTab/HubeiMap×2 类型化 |
| T3 | `3ca703ef` | hex 批①②③：`var(--theme-x,#hex)` 双写兜底清偿 + 裸 hex 值映射主题 token（97 文件，含批①重叠），豁免登记 `hex-exemptions.md` |
| T4 | `d7ae6d79` | eslint DEBT-01 裸 hex 拦截：selector 并入 P65 契约块数组（flat config 语义），三层豁免，fixture 验证拦截/豁免均生效 |
| T6 | （无 commit） | 总闸验证 + 本 SUMMARY |

## 基线终值表（硬门槛全部达标）

| 门槛 | 基线 | 终值 | 判定 |
| --- | --- | --- | --- |
| type-check | 0 | **0** | ✅ |
| vitest | 3906 passed / 0 failed | **3919 passed / 0 failed**（+13 为 T1 新增 statusHelpers 单测） | ✅ |
| lint errors | 83 / 0 新增 | **83 errors（0 新增）/ 1380 warnings（起点 1416，净降 36）** | ✅ |
| vendor-react gzip | ≤ 794,624 B | **648,481 B**（build 成功） | ✅ |

注：warnings 起点实测 1416（任务书 1411 系过时口径），仅降不升。

## SC1–4 Verdict

- **SC1（statusHelpers 唯一落点）**：PASS。唯一落点 `src/components/shared/statusHelpers.tsx`；6 域映射 + 未知值「未知/default」；两个锚点（createStatusColumn/renderStatusTag）内部改造、调用方零改动；enable 文案连带「正常→启用」已同步两处测试断言。
- **SC2（日期收敛）**：PASS。三常量入 datetime.ts；非基准展示字面量清零，余量 32 处全部为 payload 构建/常量定义/豁免项（`date-exemptions.md`：年月日 4 处=中文周标题展示、notice/index.tsx:257,280 的 T 格式实测为 ISO 8601 API payload 非 UI 展示 bug——与 SPEC 判定不符，未改，此为对 SPEC 的事实性修订上报）。
- **SC3（hex 收敛+守卫）**：PASS。基线 526 处 → 非豁免域残留 32 处全部登记 `hex-exemptions.md`（A 语义 diff/徽章色、B 导航卡分类色、C echarts 绘图行、D 结构性目录）；`--theme-purple` 为未定义变量（伪双写），清偿兜底会致样式失效，故保留并登记。DEBT-01 lint 守卫上线，新代码裸 hex 即 error。
- **SC4（any B 类）**：PASS。宽口径 52 → 21 行残留，全部 A/C/D 类登记 `any-exemptions.md`；B 类（可安全类型化）全部清偿。

## 双主题抽查记录

- token 来源唯一：`src/index.css`（light :29-104 / dark :208 起）`--theme-*` 变量，收敛后组件仅引用变量名 → 深浅主题由变量切换自动生效。
- 抽查点：①monitor/job 统计卡 content 色 `var(--theme-success/--theme-error)`；②backups diff 面板（豁免保留原色，双主题下 diff 红/绿语义不变）；③CaptchaModal `var(--theme-primary)`；④workers 状态底色 `var(--theme-bg-secondary)/var(--theme-error-bg)`。theme-styles.css 双写按 E5 dismissed 不清偿。

## T 格式 bug 修复证据

SPEC 声称 2 处 `YYYY-MM-DDTHH:mm:ss` 为展示 bug；实测 `pages/system/notice/index.tsx:257,280` 均为提交给后端的 payload 构建（ISO 8601 带时区偏移），非展示层。按「发现疑似语义反转停下上报」纪律未改，证据见 `date-exemptions.md`。**此类唯一真实展示修复**为 rpa/workers/columns.tsx（原 `date.format(...)` 局部拼接 → `formatDateTime(date.toDate())`，且与 datetime.ts 全局口径统一了时区处理）。

## 遗留豁免清单指针

- `hex-exemptions.md` — hex 残留 32 处分类 + DEBT-01 lint 豁免链说明
- `any-exemptions.md` — any 残留 21 行 A9/C7/D 类理由
- `date-exemptions.md` — 年月日/T 格式豁免与 payload 判定
- rpa/workers/columns.tsx:11 局部 formatDateTime（多剥 +HH:MM）与 datetime.ts 口径不一致，保留待后续 phase 统一
- InnovativeLayout/QuickNav 彩色导航卡（B 类）与 backups diff 色（A 类）为最大宗 hex 豁免，建议后续 phase 引入专用 token 后清偿

## 偏差记录

1. 每任务 commit（父指示）替代 PLAN「全程不 commit」。
2. warnings 起点 1416 非 1411。
3. SPEC T 格式「展示 bug」判定被实测推翻（SC2 上报项）。
4. T3 批①脚本误改 design-system/store.test/utils-three，已 `git stash push`（stash@{0}）留档回滚；后续批次脚本排除该路径。
5. T4 首版「独立 config 块」方案与 flat config 覆盖语义冲突（会静默失效既有 65/66 拦截），改为 selector 并入既有数组 + 豁免 override 前置，行为经 fixture 与 83/0 终值验证。
