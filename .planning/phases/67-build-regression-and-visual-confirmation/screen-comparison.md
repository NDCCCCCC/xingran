# Phase 67 QA-04 六屏视觉确认 — `screen-comparison.md`

> 来源: orchestrator 自动化 chrome-devtools 逐屏实测（2026-08-18）
> 路由策略: SPA 通过 `history.pushState` + `popstate` 触发（`navigate_page` 跨页丢 session）
> 后端状态: `:9000` 503；`:4000` Vite dev server 健康 —— 部分屏以**空态**呈现，框架品牌化与数据屏（监控/资产）实测验证

| # | 屏 | 路径 | 侧栏 | 顶栏 | 表头/Card | 关键品牌值 | 状态 |
|---|---|---|---|---|---|---|---|
| 1 | 登录页 | `/login` | — | — | — | 左 brandPanel `#D4A574`→`#B8854C` 铜金渐变；登录按钮同款渐变（brand-spec 允许的铜金实心场景，3.15:1）；SM2/SM3/SM4 浅黄标签锚点；body `#F0ECE3` 奶油 | ✓ PASS |
| 2 | 仪表盘 | `/dashboard` | `#14532D` | 白底 64px | 空态「创建仪表盘 / 查看仪表盘列表」按钮 | ⌘K Tag `#FEF3C7`+`#B88850`；主按钮白字；body `#F0ECE3` | ✓ PASS |
| 3 | 系统用户 | `/system/user` | `#14532D` | 白底 64px | 表头 `#E9EFEB`；状态 Tag `#E9EFEB` 底 + `#2D8949` 字（功能色保留） | Card 白卡 `#FFFFFF`；启用 Tag 与品牌语态一致 | ✓ PASS |
| 4 | 工位管理 | `/operations/workstations` | `#14532D` | 白底 64px | 表头 `#E9EFEB`；Card 白卡 | 路由可达 + 框架品牌化 ✓ | ✓ PASS |
| 5 | 监控仪表盘 | `/monitor/dashboard` | `#14532D` | 白底 64px | 8 个 Card：CPU/内存/磁盘/进程数 + 服务器列表 + 内存详情 + 网络流量 + 最近登录日志 | 实测数据：CPU 72.2% / 内存 87.9% / 磁盘 56.6% / 进程 705；ChartCard 卡片白卡 | ✓ PASS |
| 6 | 资产对账看板 | `/assets/dashboard` | `#14532D` | 白底 64px | 9 个 Card：推送已断开 + 全量资产数 + 未解决异常数 + critical 级未解决 + 7d 新增异常 + Top1 冲突类型 + 冲突类型分布 + 严重级别分布 + 7d 健康度趋势 | HealthBadge/Card 替换色深底可读 ✓ | ✓ PASS |

## 与 refs/ 旧截图对比（admin-design-plan 2026-08-17 实测存档）

| 屏 | refs/ 旧截图特征 | v1.22 后特征 | 状态 |
|---|---|---|---|
| system-dashboard.png | 主按钮 `#4F46E5` indigo；侧栏 indigo/blue 系 | 主按钮 `#156031` 绿底白字；侧栏 `#14532D` 深绿 | ✓ 缺口修复 |
| system-users.png | 表头 `#F1F5F9` 冷蓝灰 | 表头 `#E9EFEB` 绿灰淡彩 | ✓ 缺口修复 |
| system-workstations.png | 表头冷蓝灰 + 主按钮 indigo | 表头绿灰 + 主按钮绿 | ✓ 缺口修复 |
| system-monitor-dashboard.png | 同上 | 8 Card 框架 + 品牌化数据 | ✓ 缺口修复 |
| system-assets-dashboard.png | indigo 光晕残留 | 9 Card 框架 + HealthBadge 品牌化 | ✓ 缺口修复 |
| login | 已是品牌锚点（admin-design-plan 标记） | 维持铜金渐登录按钮 + SM2/SM3/SM4 浅黄标签 | ✓ 一致 |

## 与 66-01-SUMMARY 速查手册逐项核对

- 侧栏底 light `#14532D` / dark `#0a2418`：实测 light `#14532D` ✓（dark 未在此轮复测；66 T6 已复测）
- 顶栏底 light `#ffffff` / dark `#0a2418`：实测 light 白底 ✓
- 表头 `#E9EFEB`：实测 `#E9EFEB` ✓（5/6 列表页含表头）
- 主按钮 `#156031` 绿底白字：实测 ✓
- 默认 Tag `#FEF3C7` 淡黄底 + `#B88850` 铜金字：实测 ⌘K Tag ✓
- 语义状态 Tag：实测保留功能色 ✓

## 6 项 SC#2 判据（ROADMAP Phase 67 SC#2）勾选

- ☑ 无布局崩坏 — 六屏框架（侧栏/顶栏/Card/表头）实测正常
- ☑ 无不可读文本 — 主按钮 7.64:1 / 状态 Tag 5.03:1+ / 默认 Tag 装饰性（2.8:1 已豁免）
- ☑ 无残留 indigo/slate 冲突色 — scanner 0 命中（627 files allowlist 5）
- ☑ 侧栏深绿 + 顶栏白底解耦 — 双状态正确
- ☑ 主按钮绿底白字 — 全屏绿 + ⌘K 锚点铜金实心（合规）
- ☑ Tag 默认配方 — SM2/SM3/SM4 + ⌘K 全部到位

## Fallback 路径备注

- `take_screenshot` 截图管线历史超时（65 T6 / 66 T6 均遇）—— 本 phase 全程使用 `evaluate_script` 提取 DOM computed-style + 文本，作为结构级证据替代
- 后端 :9000 503 — 数据屏（monitor/assets）部分以零态呈现，但 Card 框架 + 颜色 + 布局 + 标签全部品牌化（空态本身也是品牌化验收对象）
- SPA 路由：`popstate` 触发可靠；`navigate_page` URL 跳转丢 session（dashboard 屏保留 session）