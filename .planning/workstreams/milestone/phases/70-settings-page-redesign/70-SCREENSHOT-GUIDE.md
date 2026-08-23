# Phase 70 截图矩阵操作指引（70-07 Task 4 · checkpoint:human-verify）

> 本文件由 70-07 executor 生成（executor 无浏览器访问权）。主会话编排器请按本指引
> 用 chrome-devtools 完成截图与运行时确认，归档后由用户回复 "approved" 收口，
> 或列出需修正的视觉/行为问题。
>
> 目标归档目录：`.planning/phases/70-settings-page-redesign/screenshots/`（需先创建）

## 前置准备

1. **后端启动（sqlite 开发库，`configs/config.yaml` type=sqlite）**：
   ```bash
   go run ./cmd/main.go    # 或 .\xingran-backend.exe
   ```
   - 首启日志确认两点：
     - Migrate209 执行行（`system/settings-page/index` → `system/settings/index` 数据修正，changed=true）
     - 「菜单缓存已失效」行（迁移后菜单缓存失效日志）
   - **二启**（重启进程）：Migrate209 应为 no-op（changed=false），无缓存失效日志
   - 后端端口 `:9000`
2. **前端 dev server**：
   ```bash
   cd xingran-react-frontend && npm run dev    # http://localhost:4000
   ```
3. **登录**管理员账号。明暗切换用「用户设置 → 界面设置」分段卡（或 header 主题切换），
   断点用 DevTools 响应式模拟（`<lg` = <992px，建议 768×1024；`≥lg` 用桌面视口 ≥1200px）。

## 截图矩阵（10 张）

### A 组 — ≥lg 桌面断点 × light/dark（6 张）

| # | URL | 模式 | 文件名 | 期望视觉要点 |
|---|-----|------|--------|--------------|
| 1 | `/system/settings-page?cat=email` | light | `01-email-lg-light.png` | 左侧竖排分类白卡 3 项（邮箱/API/验证码背景图，邮箱 active 品牌绿）；右侧表格页撑满（无 maxWidth）；统计卡行 3 卡（总配置数/启用/停用）；工具栏卡 = **状态筛选 Select + 搜索/重置按钮 + 深绿主按钮「新增配置」，无名称输入框**（70-07 已移除）；双层纸感表格（绿灰表头/斑马纹/分割线 `#DBD7CE`） |
| 2 | `/system/settings-page?cat=email` | dark | `02-email-lg-dark.png` | 同上结构；统计卡/表头/导航卡前景背景可读（WCAG AA 抽查）；无灰底黑字不可读项 |
| 3 | `/system/settings-page?cat=captcha` | light | `03-captcha-lg-light.png` | 统计卡 4 卡（总数量/启用数量/禁用数量/总使用次数）；紧凑工具栏（文件名/形状/难度/状态筛选）；图片网格墙（4:3 缩略图卡，卡脚文件名 + 状态徽标：启用=xr-tag-green）；内容撑满 |
| 4 | `/system/settings-page?cat=captcha` | dark | `04-captcha-lg-dark.png` | 同上网格墙暗色形态；徽标/统计数值对比度可读 |
| 5 | `/user/settings?cat=appearance` | light | `05-appearance-lg-light.png` | 左导航 3 项（界面/布局/数据，界面 active）；右侧内容**限宽 760px**（明显窄于表格页，居左或居中按实现）；「界面设置」卡片内行式设置项（label+描述+右侧控件）；明暗模式分段卡片选择器（浅色/深色预览块，当前选中项品牌绿描边） |
| 6 | `/user/settings?cat=appearance` | dark | `06-appearance-lg-dark.png` | 分段预览块双形态正确（深色选中态描边）；行式项分割线与描述文字暗色可读 |

### B 组 — <lg 窄屏（DevTools 响应式模拟 768×1024）（4 张）

| # | URL | 模式 | 文件名 | 期望视觉要点 |
|---|-----|------|--------|--------------|
| 7 | `/system/settings-page?cat=email` | light | `07-system-narrow-light.png` | 左侧竖排导航**降级为顶部横向 Segmented block**（3 分类）；内容区全宽；表格横向滚动正常 |
| 8 | `/system/settings-page?cat=email` | dark | `08-system-narrow-dark.png` | Segmented 暗色形态正确、选中项可读 |
| 9 | `/user/settings?cat=appearance` | light | `09-user-narrow-light.png` | Segmented 降级 + 内容全宽（限宽在窄屏不再生效）；分段卡片选择器两卡并排不溢出 |
| 10 | `/user/settings?cat=appearance` | dark | `10-user-narrow-dark.png` | 同上暗色形态 |

## 运行时行为确认（不截图，记录通过/失败）

1. **迁移链路**：见前置准备 1 的双启验证（首启 changed / 二启 no-op）。
2. **侧边栏入口**：点击侧边栏「系统设置」**不白屏**（迁移后 `system/settings/index` 组件解析正常）；Console 无 `[createLazyComponent] Module not found`。
3. **D-03 URL 还原**：访问 `/system/settings-page?cat=captcha` 后**刷新**页面 → 停留在 captcha 分类（网格墙）。
4. **后退语义**：在设置页内依次切 email → api → captcha，然后点浏览器**后退** → 应**离开设置页**（回到上一页面），而非逐分类回退（`replace: true` 语义）。
5. **非法 cat 值回退**：访问 `/system/settings-page?cat=nonexistent` → 应回退到 email 分类（defaultCat）。

## 基准对照（QA-04 风格）

对照基准：
- `.planning/phases/64-brand-token-layer-and-contrast-verification/test1-system-user-page.png`（用户管理页实测）
- `.planning/sketch/real-final-v16.webp`（品牌视觉基准）

核对语汇一致性：统计卡行（图标圆角块 + 标题数值）、工具栏（筛选 + 深绿主按钮）、双层纸感卡片（白卡衬奶油 `#F0ECE3` 画布）、绿灰表头表格、深绿主按钮 `#156031`、铜金点缀克制（每屏 ≤2 处）。

## 通过标准

- 10 张截图全部归档，命名与上表一致
- 运行时确认 5 项全部通过
- 基准对照无视觉语汇偏离（发现偏离逐条列出，回复中说明修正建议）

## 完成信号

回复 **"approved"** 收口（70-07 续作将补写 70-07-SUMMARY.md），或列出需修正的问题清单。
