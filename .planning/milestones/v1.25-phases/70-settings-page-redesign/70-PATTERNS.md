# Phase 70: settings-page-redesign - Pattern Map

**Mapped:** 2026-08-19
**Files analyzed:** 16 组（5 新建 + 8 改造/追加 + 3 删除/吸收处置）
**Analogs found:** 13 / 13 代码施工项有可抄先例（10 项 exact；3 项「自身即范本」的改写类）；2 个视觉子形态仅有 partial 先例（见 No Analog Found）

> 文件清单来源：`70-CONTEXT.md` D-01~D-12 + `70-RESEARCH.md` 现状盘点/Wave 0 Gaps + `70-UI-SPEC.md` L-1~L-4 / IC-1~IC-4（已 approved，分类注册表与 `.xr-*` 新类清单以 UI-SPEC 为准）。

## File Classification

| New/Modified File | Role | Data Flow | Closest Analog | Match Quality |
|-------------------|------|-----------|----------------|---------------|
| `xingran-react-frontend/src/design-system/components/SettingsShell.tsx`（新） | component | request-response（URL 参数驱动） | `src/pages/system/user/index.tsx:554-565`（Sider 分栏）+ `src/pages/network/mac/index.tsx:47-51`（断点+searchParams）+ `src/design-system/components/DensitySwitcher.tsx`（目录惯例） | exact（三范本组合） |
| `xingran-react-frontend/src/pages/system/settings/index.tsx`（改写：barrel → 设置页入口） | component（路由入口） | request-response | `src/pages/system/settings-page/index.tsx:15-61`（分类注册表迁移源） | exact |
| `xingran-react-frontend/src/pages/system/settings/email-config.tsx`（改造） | component | CRUD | `src/pages/system/user/index.tsx:531-660`（v16 三段式） | exact |
| `xingran-react-frontend/src/pages/system/settings/api-config.tsx`（改造） | component | CRUD | 同上 + 自身 Modal 内 Tabs 保留段（:351-436） | exact |
| `xingran-react-frontend/src/pages/system/settings/captcha-background.tsx`（改造为图片网格墙） | component | CRUD + file-I/O（上传） | `user/index.tsx:531-552`（stat-cards）+ 自身 Image 预览（:259-267）/上传约束（:236-250） | exact |
| `xingran-react-frontend/src/pages/settings/index.tsx`（改造：行式设置项） | component | request-response（即改即存 PUT） | 自身 store 接线（:25-36）+ `settingsStore.ts:94-157` + `DensitySwitcher.tsx`（选项常量结构） | role-match（行式形态新） |
| `xingran-react-frontend/src/index.css`（追加 `.xr-settings-*` / `.xr-setting-row` / `.xr-appearance-*` / `.xr-captcha-*` 段） | config（样式） | — | 自身 v16 段 `:5099-5266` | exact |
| `xingran-react-frontend/src/store/settingsStore.ts`（删死 actions） | store | — | 自身（删除段 :45-46/:181-197；保留段 :94-134） | exact |
| `internal/core/db/migrations/migration_208_update_settings_menu_component.go`（新） | migration | batch（幂等 UPDATE） | `migration_205_rpa_worker_id_default.go`（注释+applogger 风格）+ `migration_207_menu_catalog_seed.go`（幂等/断言/双方言） | exact |
| `internal/core/db/migrations/migration_208_*_test.go`（新） | test | batch | `migration_207_menu_catalog_seed_test.go` | exact |
| `internal/core/db/database.go`（注册 208 双分支） | config | batch | 同文件 `:837-847`（207 的挂法） | exact |
| `xingran-react-frontend/src/pages/system/settings/__tests__/SettingsShell.test.tsx`（新） | test | — | `src/pages/network/ports/__tests__/index.test.tsx` | exact |
| `xingran-react-frontend/src/pages/settings/__tests__/index.test.tsx`（新） | test | — | 同上 | exact |
| 删除 `src/pages/system/settings-page/index.tsx`（D-11） | — | — | —（与 Migrate208 配套执行） | — |
| 删除 `src/pages/system/captcha-background/`（7 文件死目录，D-12） | — | — | —（RESEARCH 死目录裁定 + 删除前防御 SQL） | — |
| D-10 首任务原子提交（吸收 13 文件未提交改动） | — | — | —（git 操作；`go build ./...` + `tsc --noEmit` 已双绿验证） | — |

## Pattern Assignments

### `src/design-system/components/SettingsShell.tsx`（新，component / URL 参数驱动）

**Analog:** `src/pages/system/user/index.tsx`（分栏先例）+ `src/pages/network/mac/index.tsx`（断点先例）+ `src/design-system/components/DensitySwitcher.tsx`（目录惯例）

**分栏骨架先例**（user/index.tsx:554-565 —— 注意 `width={250}` 是源码值，CSS `.xr-dept-panel` 实际覆写为 300（index.css:5243-5247）；本 phase 按 UI-SPEC L-1 裁定取 **220**，勿照抄 250/300）：

```tsx
const { Sider, Content } = Layout;   // :49

<Layout style={{ display: "flex", alignItems: "stretch", background: "transparent" }}>
  <Sider
    width={250}
    className="dept-list-sider xr-dept-panel"
    style={{ background: "transparent", padding: "0 14px 16px 0" }}
  >
    {/* 左面板内容 */}
  </Sider>
  <Content style={{ padding: 0, background: "transparent" }}>
    {/* 右侧内容 */}
  </Content>
</Layout>
```

关键：`background: "transparent"` 让 Sider 透出奶油画布（`.dept-list-sider { background: transparent !important; }` index.css:5249-5251 是配套覆写——SettingsShell 的 Sider 同样需要 `background: transparent !important` 覆写，否则会被全局侧栏深绿样式污染）。user 页 `:528` 注释「顶部 TabBar 已显示标题，页面内不再重复 PageTitle」——设置页同理不加 PageTitle。

**断点 + searchParams 组合先例**（mac/index.tsx:46-52）：

```tsx
const { useBreakpoint } = Grid;

const MACAddressPage: FC = () => {
  const breakpoint = useBreakpoint();
  const isMobile = !!breakpoint.xs;
  const [searchParams, setSearchParams] = useSearchParams();
```

SettingsShell 按 UI-SPEC 用 `screens.lg`（非 xs）判定降级。Segmented 消费先例见 `DensitySwitcher.tsx:44-69`（options.map + value/onChange + 令牌 style）。**useEffect 纪律**：categories 注册表必须模块级常量或 useMemo（CLAUDE.md 强制条款；user 页 `:322-328` 的 init effect + eslint-disable 注释是既有惯例形态）。

**目录归属**：`src/design-system/components/`（UI-SPEC L-1 已裁定）——该目录现有 AntdThemeBridge / DensitySwitcher / LayoutProvider / LayoutSwitcher / PageTitle / ThemeProvider，均为「跨页共用 + 纯令牌消费」组件，SettingsShell 同性质。

---

### `src/pages/system/settings/index.tsx`（改写：barrel → 系统设置页入口）

**Analog:** `src/pages/system/settings-page/index.tsx`（现壳，分类注册表迁移源）

**注册表结构**（settings-page/index.tsx:23-54 —— key/label/icon/children 四元组直接改造成 UI-SPEC `SettingsCategory`，cat 值按 UI-SPEC 从 `captcha-background` 收敛为 `captcha`）：

```tsx
const tabItems = [
  { key: "email", label: <span><MailOutlined />邮箱配置</span>, children: <EmailConfigPage /> },
  { key: "api", label: <span><ApiOutlined />API配置</span>, children: <APIConfigPage /> },
  { key: "captcha-background", label: <span><PictureOutlined />验证码背景图</span>, children: <CaptchaBackgroundSettingsPage /> },
];
return <div><Tabs activeKey={activeTab} onChange={setActiveTab} items={tabItems} /></div>;
```

改造后 = `<SettingsShell categories={[...]} defaultCat="email" />`；`usePersistedStateController` activeTab 段（:16-21）**整体删除**（D-03/D-12），替换为 SettingsShell 内部的 `useSearchParams`。

**路由解析硬约束（Pitfall 4 的代码依据）**：`src/router/componentLoader.tsx:34-42` 的 glob 模式为 `"/src/pages/**/{index,detail}.tsx"` —— 合并目录入口**必须**命名 `index.tsx`；DB component 值不带 `pages/` 前缀、不带 `.tsx` 后缀（Migrate208 写 `system/settings/index`）。tab key = `location.pathname`（`src/components/layout/shared/useRouteTabs.ts:97-99` 已复核，**实际路径不在 hooks/ 下**），searchParams 变化不产生新 RouteTab。

---

### `src/pages/system/settings/email-config.tsx`（改造，CRUD）

**Analog:** `src/pages/system/user/index.tsx:531-660`（v16 三段式，直接对照改写）

**v16 三段式**（user/index.tsx:531-552 统计卡 + 568-646 工具栏卡 + 649-660 表格卡）：

```tsx
<div className="stat-cards">
  <div className="stat-card">
    <div className="stat-label">用户总数</div>
    <div className="stat-value">{statistics.total}</div>
    <div className="stat-trend">全部门合计</div>
  </div>
  <div className="stat-card sc-green">
    <div className="stat-label">正常用户</div>
    <div className="stat-value">{statistics.active}</div>
    <div className="stat-trend">占比 {activePct}%</div>
  </div>
  <div className="stat-card sc-gray">
    <div className="stat-label">禁用用户</div>
    <div className="stat-value">{statistics.inactive}</div>
    <div className="stat-trend">可随时恢复</div>
  </div>
</div>

<Card style={{ marginBottom: 14 }}>           {/* 工具栏卡 */}
  <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: "16px" }}>
    <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
      {/* 搜索项 + 状态 Select(allowClear) + 重置(default)/搜索(primary) 按钮 */}
    </Form>
    <Space>
      <Button type="primary" icon={<PlusOutlined />}>新增用户</Button>
    </Space>
  </div>
</Card>

<Card>
  <Table className="xr-table-zebra" size="middle" ... />
</Card>
```

统计卡数据口径（Pitfall 5 已验证可行）：`getEmailConfigList` 的参数类型 `EmailConfigListParams` **已含 `status?: number`**（`src/lib/notificationConfigApi.ts:21-26`）——主列表 `total` + 启用/停用各一次 `pageSize=1, status=N` 轻请求取 total，**零 API 封装改动**。email/api 遵循全局惯例 `0=正常 / 1=停用`（IC-4）。

**业务保留段（重构时逐字保留，勿动逻辑）**：
- 密码脱敏（email-config.tsx:107-113）：`if (!values.password || values.password === "******") { delete updateData.password; }`
- 双请求收敛点（:278-281 useEffect 驱动 + :305-309 `Table onChange` 里又手动 `loadConfigs()`）：删掉 onChange 内的手动加载，仅保留 `setCurrent/setPageSize`，由 useEffect 单一路径触发——现文件为双请求风险，user 页 `useTableManager` 同样不在 onChange 里手动加载
- 编辑/测试双 Modal（:314-421）：结构与宽度 700 不变（IC-3），仅 Tag 迁移
- 错误处理惯例（:125-129）：`isFormValidationError(error)` 早退 → `message.error(...)`

**Tag 迁移**（D-12 第 6 项）：`<Tag color="blue">默认</Tag>`（:185）→ `xr-tag-gold`；`<Tag color="green">SSL</Tag>`（:197）→ `xr-tag`；状态 Tag（:223）→ `status === 0 ? xr-tag-green : xr-tag`。先例 = user/index.tsx:161-167 的 `className={`xr-tag ${isSuperAdmin ? "xr-tag-gold" : "xr-tag-green"}`}`。
**列语汇迁移**：时间列 `.xr-cell-time`（user:198）、名称列 `.xr-cell-id`（user:79）、操作列 `.xr-row-ops` + 删除 `.xr-op-danger`（user:208-231）。

---

### `src/pages/system/settings/api-config.tsx`（改造，CRUD）

**Analog:** 同 email-config（v16 三段式）+ 自身保留段

与 email 的差异点：
- 工具栏筛选多一档：`configTypeFilter` 现状（:52 状态 + :67-277 Select onChange）并入工具栏 inline Form；`APINotificationConfigListParams` 同样已含 `status?: number`（notificationConfigApi.ts:160 附近）
- **Modal 内 Tabs 保留**（:351-436，headers/template/auth 三段）与 authType 条件表单（:392-431 `Form.Item noStyle shouldUpdate` + getFieldValue 分支）——IC-3 明确只有页面级 Tabs 被替换，diff 里不应出现 Modal 内 Tabs 删除
- Tag 迁移：类型 Tag（:163-168，SMS=cyan/WEBHOOK=purple/PUSH=orange）→ `xr-tag` 中性；`默认`（:154）→ `xr-tag-gold`；状态（:204）→ green/中性二值
- 统计卡 3 卡口径按 UI-SPEC L-2（不加第 4 卡）

---

### `src/pages/system/settings/captcha-background.tsx`（改造为图片网格墙，CRUD + file-I/O）

**Analog:** `user/index.tsx:531-552`（stat-cards）+ 自身保留段

**统计卡 v16 化**：现 Row/Col/Statistic（:364-407）→ `.stat-cards` 四卡（总数默认绿条 / 启用 sc-green / 禁用 sc-gray / 总使用次数 sc-gold）。**D-12 第 5 项同点位修复**：:376/:385 的 `var(--theme-success, #3f8600)` / `var(--theme-error, #cf1322)` fallback 旧写法改纯 `var(--theme-success)`（v1.22 令牌已保证存在）。

**网格墙卡片保留的现成 API 消费**：
- Image 预览（:259-267，网格墙直接沿用同 API，不自建灯箱）：

```tsx
<Image
  width={50} height={30}
  src={record.previewUrl}
  preview={{ src: record.previewUrl }}
  style={{ objectFit: "cover", cursor: "pointer" }}
/>
```

- 上传约束（:236-250）：`beforeUpload` 校验 image/* + <2MB + `return false` 手动上传 + `maxCount: 1` —— 原样保留
- 统计端点（:110-117）：`captchaService.getCaptchaBackgroundStatistics()` 返回 `{totalCount, enabledCount, disabledCount, totalUsage?}`，网格墙统计卡数据源现成
- **status 语义反转（IC-4 / Pitfall 1）**：`:311` `status === 1 ? "启用" : "禁用"`、`:331` 启停按钮文案取反 —— 网格墙沿用 `1=启用`，**勿"纠正"**

**随重构消除的死状态**（D-12 第 4 项）：`:74-75` `previewVisible/_setPreviewImage` + `:638-650` 隐藏 Image 元素，整体不搬运。
**保留功能**：预加载缓存按钮（:206-213, :486-488）、筛选（文件名/形状/难度/状态，降为工具栏紧凑形态）、`usePagination` 分页。
**网格容器**：新类 `.xr-captcha-grid`（`repeat(auto-fill, minmax(200px, 1fr)); gap: 12px`——对齐 `.stat-cards` 的 auto-fill 思路 index.css:5100-5105）。

---

### `src/pages/settings/index.tsx`（改造：行式设置项 + 即改即存）

**Analog:** 自身 store 接线 + `settingsStore.ts` 单字段 actions + `DensitySwitcher.tsx` 选项结构

**store 接线保留段**（settings/index.tsx:25-36 —— initialize 懒加载模式保留）：

```tsx
const { preferences, loading, initialized, updatePreferences } = useSettingsStore();
useEffect(() => {
  if (!initialized) {
    useSettingsStore.getState().initialize();
  }
}, [initialized]);
```

**即改即存的 store 能力**（settingsStore.ts:94-134 —— `updateTheme` / `updateLayout` / `updateDataPageSize` 已具备单字段更新，IC-1 直接消费）：

```typescript
updateLayout: async (layoutUpdate) => {
  const { preferences } = get();
  const updatedPreferences: UserPreferences = {
    ...preferences,
    layout: { ...preferences.layout, ...layoutUpdate },
  };
  await get().updatePreferences(updatedPreferences);
},
```

**失败回滚的实现事实**（settingsStore.ts:137-157）：`updatePreferences` 是「先 `await configService.updateUserPreferences(...)` 服务器成功、再 `set({ preferences })` 本地提交」——失败时 throw 且本地 state 未变；行式控件若为受控（值取自 store），**控件态天然回滚**，无需额外回滚代码。页面侧只需 `try/catch + message.error("保存设置失败，请重试")`。

**删除段**：Form 批量提交结构（:62-131 的 Form/Divider/Radio.Button + :39-53 handleSave/handleReset + :125-130 保存/重置按钮）整体替换为行式；密度选项值 `compact/comfortable/spacious`（:91-95）与分页选项（:117-122）**原值保留**（与 settingsStore 权威字段一致，THEME-03 不回归）。
**明暗分段卡片选择器**：无完整仓内范本（见 No Analog Found），结构参照 DensitySwitcher 的模块级 options 常量（:19-38）+ 视觉参照 `.xr-dept-panel` selected 语汇。

---

### `src/index.css`（追加 `.xr-*` 新类段）

**Analog:** 自身 v16 段 `:5099-5266`（节注释 + 令牌引用 + `::before` 色条的既有写法范式）

**白卡双层纸感样板**（:5106-5114 —— `.xr-settings-nav` / `.xr-settings-content` / `.xr-captcha-card` / `.xr-appearance-card` 全部按此语汇新写）：

```css
.stat-card {
  background: var(--theme-bg-surface);
  border: 1px solid var(--theme-border-primary);
  border-radius: var(--theme-radius-xl);
  padding: 16px 18px;
  box-shadow: 0 1px 2px rgba(120, 100, 70, 0.06);
  position: relative;
  overflow: hidden;
}
```

**active 态品牌绿样板**（:5259-5266 —— 分类项 active 语汇直接同款；checker 提示：底色优先引用令牌 `var(--theme-brand-alpha-10)`（:41 已定义 `rgba(21, 96, 49, 0.1)`）而非手写 rgba）：

```css
.xr-dept-panel .ant-tree .ant-tree-node-content-wrapper:hover {
  background: var(--theme-primary-50);
}
.xr-dept-panel .ant-tree .ant-tree-node-content-wrapper.ant-tree-node-selected {
  background: rgba(21, 96, 49, 0.1);
  color: var(--theme-primary);
  font-weight: 600;
}
```

**分段预览块可用令牌**（IC-2）：`--sidebar-bg: #14532d`（:62，明暗两模式均为深绿）、`--sidebar-text-active: #e0e0b0`（:68，浅黄强调）、`--sidebar-accent: #c09058`（:69，**铜金**——注意 UI-SPEC IC-2 把深色预览内容条写作「`var(--sidebar-accent)`（#E0E0B0 系强调浅黄）」，**该注释与令牌实际值不符**：`#e0e0b0` 对应的是 `--sidebar-text-active`，`--sidebar-accent` 是铜金 `#c09058`。planner 按 UI-SPEC 文字意图（浅黄内容条）应取 `--sidebar-text-active`；若取铜金则与「深色预览块内铜金点缀」预算冲突）。
间距/圆角令牌：`:107-113`（`--theme-spacing-xs..3xl`，全部 4 的倍数）。**新段落位置建议**：追加在 v16 段（:5266 `.xr-dept-panel` 之后、分页器段之前）保持同类相邻；全部类只允许引用 `var(--theme-*)` / `var(--sidebar-*)`，禁手写 hex（QA-02 `check-hardcoded-colors.mjs` lint 门拦截）。

---

### `src/store/settingsStore.ts`（删死 actions）

**Analog:** 自身。删除 `exportPreferences` / `importPreferences` 两处：接口声明 `:45-46` + 实现 `:181-197`。保留段 `:94-134`（单字段 actions）与 `:209-215`（类型选择器）。验证：`npm run lint` + `npm run deadcode`（knip，RESEARCH D-12 行已有门）。

---

### `internal/core/db/migrations/migration_208_update_settings_menu_component.go`（新，migration / batch）

**Analog:** `migration_205_rpa_worker_id_default.go`（GORM 幂等写 + applogger 风格）+ `migration_207_menu_catalog_seed.go`（幂等注释/断言/双方言语义）

**结构模式**（migration_205:31-51 —— 函数签名、日志、错误包装风格照抄；UPDATE 本身天然幂等无需 205 的方言守卫，207 的双方言结论（`migration_207:37-39` 注释）适用——无 PG 专有语法则不写 `isPostgreSQL` 分支）：

```go
func Migrate205RpaWorkerIdDefault(db *gorm.DB) error {
    log.Println("Running migration 205: ...")
    if err := db.Exec(`...`).Error; err != nil {
        return fmt.Errorf("... 失败: %w", err)
    }
    applogger.Infof("[迁移] 205 ... 已设置")
    return nil
}
```

Migrate208 主体（RESEARCH 骨架，UPDATE by id + 旧值双条件 = 天然幂等）：

```go
res := db.Model(&models.Menu{}).
    Where("id = ?", "308d89be-e516-4556-b949-bc22bf6ab759").
    Where("component = ?", "system/settings-page/index").
    Update("component", "system/settings/index")
if res.Error != nil { return fmt.Errorf("migration 208: %w", res.Error) }
if res.RowsAffected > 0 { applogger.Infof("[迁移] 208 系统设置菜单组件路径已更新 (rows=%d)", res.RowsAffected) }
```

**★ 缓存失效的时序修正（本 mapping 新事实，推翻 RESEARCH A4 的「迁移函数内 DeleteByPattern」）**：迁移在 `db.NewDatabase`（core.go:258-261 `initDBAndData` 内，:206 触发）执行，而 `c.Cache` 直到 core.go:343 才创建——**迁移函数内拿不到 cache 实例**。可行方案二选一（planner 定）：
- **方案 A（推荐）**：Migrate208 返回 `(changed bool, err error)`（`RowsAffected > 0` 即 changed）；core.go 在 :343 cache 就绪后、菜单服务构建点附近，changed 时按下方「菜单缓存失效」shared pattern 清 6 个 menu key 前缀
- **方案 B（更简）**：core.go cache 就绪后**无条件**失效 menu keys（每次启动多一次菜单缓存重建，成本可忽略）——一行调用换实现简单

失效工具与 key 清单（menu_cache_impl.go:161-171）：`InvalidateCacheByPattern(ctx, cache, []string{...}, "MENU")`（定义于 `internal/services/system/cache_utils.go:64`）；patterns = `menu:tree*` / `menu:router*` / `menu:all*` / `menu:user:menus:*` / `menu:user:all-menus:*` / `menu:user:permissions:*`（常量定义 `internal/services/system/cache_keys.go:99-103`；key 不带 `xingran:` 前缀，Redis 前缀自动添加）。

---

### `internal/core/db/migrations/migration_208_*_test.go`（新，test）

**Analog:** `migration_207_menu_catalog_seed_test.go`

**内存 sqlite 夹具**（207 test:12-28）：

```go
func freshSQLiteDBForMigrate207(t *testing.T) *gorm.DB {
    t.Helper()
    dsn := "file:" + t.Name() + "?mode=memory&cache=shared&_busy_timeout=5000"
    db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})  // glebarez/sqlite 纯 Go 驱动
    if err != nil { t.Fatalf("open sqlite: %v", err) }
    if err := db.AutoMigrate(&models.Menu{}, &models.Role{}, &models.RoleMenu{}); err != nil {
        t.Fatalf("AutoMigrate: %v", err)
    }
    return db
}
```

208 版：AutoMigrate 最小表集只留 `&models.Menu{}`。**两个必写用例**（对照 207 test:42-91 幂等段）：
1. 预插一行 `{ID: "308d89be-…", Component: "system/settings-page/index"}` → 跑迁移 → 断言 component == `system/settings/index` → **再跑一遍断言值不变**（幂等）
2. 组件路径已是不含旧值的库（或 component 为其他值）→ 跑迁移 → 断言 RowsAffected==0 / 值未被误改（双条件 WHERE 的守护验证）

---

### `internal/core/db/database.go`（注册 208 双分支）

**Analog:** 同文件 207 的挂法（唯一逐字照抄的位置模式）

**PG 分支（advisory-lock 块内，:837-842）**：

```go
// 规范菜单目录种子 (239 条, 幂等; debug admin-role-incomplete-menus 修复)
if err := migrations.Migrate207SeedCanonicalMenuCatalog(d.DB); err != nil {
    applogger.Errorf("规范菜单目录种子失败 (非阻断,留待下次启动): %v", err)
}
```

**SQLite else 分支（:843-847）**：

```go
} else {
    // sqlite 分支: 规范菜单目录种子 (双方言迁移; PG 分支在上方 advisory-lock 块内执行)
    if err := migrations.Migrate207SeedCanonicalMenuCatalog(d.DB); err != nil {
        applogger.Errorf("规范菜单目录种子失败 (非阻断,留待下次启动): %v", err)
    }
```

208 挂法：PG 分支追加在 207 调用之后（块尾）、SQLite 分支追加在 207 之后（或 `ensureSQLiteReconciliationViews` 之前）；错误处理统一 `applogger.Errorf(... 非阻断,留待下次启动 ...)`。**数据 UPDATE 双方言通用，两个分支都要调**。

**⚠️ 编号冲突警示（本 mapping 新事实）**：Phase 69（`69-PATTERNS.md`，planning 中）同样规划 `migration_208_dict_seed.go`。执行时以 `internal/core/db/migrations/` 目录实际最大编号为准（本 mapping 实测现最大 **207**）——若 69 先落地占用 208，本 phase 顺延 209 并同步改函数名/注册点/测试名。

---

### `SettingsShell.test.tsx` + `pages/settings/__tests__/index.test.tsx`（新，test）

**Analog:** `src/pages/network/ports/__tests__/index.test.tsx`（项目内页面级 vitest 标准形态）

**基础设施三件套**（ports test:21-29 / :155-161 / :33-37）：

```tsx
// (1) ResizeObserver polyfill（antd v6 依赖，jsdom 缺失）
class ResizeObserverStub { observe() {} unobserve() {} disconnect() {} }
if (typeof globalThis.ResizeObserver === "undefined") { ... }

// (2) Wrapper：MemoryRouter + antd App（message 上下文）
function Wrapper({ children }: { children: ReactNode }) {
  return <MemoryRouter><App>{children}</App></MemoryRouter>;
}
render(<PortStatusPage />, { wrapper: Wrapper });

// (3) store mock（selector 直通可控状态）
let mockPermissions: string[] = [];
vi.mock("@/store/menuStore", () => ({
  useMenuStore: (selector: (s: { permissions: string[] }) => unknown) =>
    selector({ permissions: mockPermissions }),
}));
```

**断点 mock 写法**（ports test 无先例，用其 :145-151 的 `vi.mock + importActual` 部分替换模式扩展）：

```tsx
vi.mock("antd", async (importOriginal) => {
  const actual = await importOriginal<typeof import("antd")>();
  return { ...actual, Grid: { ...actual.Grid, useBreakpoint: () => mockBreakpoint } };
});
// 用例内切换 mockBreakpoint = { lg: true } / { lg: false } 驱动 Sider/Segmented 分支
```

SettingsShell.test 断言集（RESEARCH Wave 0）：`?cat=` 驱动 active 分类、非法值回退默认、`replace: true` 语义（mock setSearchParams 断言调用参数）、`<lg` 渲染 Segmented / ≥lg 渲染 Sider。MemoryRouter 的 `initialEntries={["/system/settings-page?cat=api"]}` 可注入初始 cat。
用户设置 index.test：mock `settingsStore`（selector 直通 preferences + spy updateTheme/updateLayout/updateDataPageSize）→ 断言行式控件 onChange 调用对应 action（D-06）。
分类注册表完整性测试：对两页各自导出的 categories 做纯数据断言（key/label/icon 非空、3 项、key 唯一）——纯常量测试参照 `src/design-system/tokens/colors.test.ts` 同目录惯例。

---

## Shared Patterns

### v16 列表页三段式（stat-cards + 工具栏卡 + 表格卡）
**Source:** `user/index.tsx:531-660` + `index.css:5099-5184`
**Apply to:** email-config.tsx、api-config.tsx（完整三段）；captcha-background.tsx（前两段 + 网格墙替代表格段）

### 白卡双层纸感语汇（surface 底 + 1px border-primary + radius-xl + 弱阴影）
**Source:** `index.css:5106-5114`（`.stat-card`）
**Apply to:** `.xr-settings-nav` / `.xr-settings-nav-item` / `.xr-settings-content` / `.xr-captcha-card` / `.xr-appearance-card` / 空状态卡全部新类

### active 态品牌绿（hover primary-50 → active brand-alpha-10 + primary 字 + 600）
**Source:** `index.css:5259-5266`（`.xr-dept-panel` selected）+ 令牌 `--theme-brand-alpha-10`（index.css:41）
**Apply to:** SettingsShell 分类项 active、明暗分段卡选中态（IC-2 的 2px primary 描边 + focus 环同源）

### Tag 品牌圆点迁移（Antd preset → xr-tag 家族）
**Source:** `user/index.tsx:161-167` + `index.css:5187-5217`
**Apply to:** email（默认→gold / SSL→中性 / 状态→green 二值）、api（类型→中性 / 默认→gold / 状态→green）、captcha（形状→中性 / 难度→gold / 启用→green）

### 迁移双分支注册 + 非阻断错误处理
**Source:** `database.go:837-847`
**Apply to:** Migrate208 注册（仅一处修改点，两分支各一次）

### 菜单缓存失效（6 个 menu key 前缀）
**Source:** `menu_cache_impl.go:161-171`（InvalidateMenuCache 的 pattern 列表）+ `cache_utils.go:64`（InvalidateCacheByPattern 签名）+ `cache_keys.go:99-103`（常量）
**Apply to:** Migrate208 改库后的配套失效——因 core.go 时序（迁移 :206 早于 cache :343）必须在 cache 就绪后执行（方案 A changed 标志 / 方案 B 无条件，见 migration_208 节）

### 页面错误/加载惯例
**Source:** email-config.tsx:63-78（loadConfigs setLoading/try/finally + message.error）、:125-129（isFormValidationError 早退）
**Apply to:** 全部分类子页的重构后加载函数（结构不变，仅文案按 UI-SPEC Copywriting 表升级）

## No Analog Found

| File / 子形态 | Role | Data Flow | Reason |
|------|------|-----------|--------|
| 明暗模式分段卡片选择器（`.xr-appearance-cards/-card/-preview-light/-dark`） | component（视觉子形态） | — | 仓内无「带预览块的卡片式单选」先例；组合参照：DensitySwitcher.tsx:19-38（模块级选项常量）+ `.xr-dept-panel` selected 语汇 + IC-2 完整规格（160px 双卡、预览区 90px、`role="radio"` + `aria-checked`、←/→ 键盘切换）。全部色值引用既有令牌，无新色值决策 |
| 行式设置项（`.xr-setting-row`） | component（视觉子形态） | — | 无现成 CSS 类；结构简单（flex space-between + 分割线），按 UI-SPEC L-4 规格新写（padding 16px 0、1px border-primary 分割、desc 12px secondary） |

两者均有完整 UI-SPEC 契约 + 组合先例，风险低；planner 直接按 IC-1/IC-2/L-4 规格施工即可。

## Metadata

**Analog search scope:** `xingran-react-frontend/src/{design-system,pages/system/{user,settings,settings-page},pages/settings,pages/network/{mac,ports},store,lib,router,components/layout/shared,index.css}`、`internal/core/db/{database.go,migrations/}`、`internal/services/system/`、`internal/core/core.go`
**Files scanned:** 18 个实读 + 6 组 grep 精确验证（Glob/Grep 专用工具在本环境不可用，bash grep 等效——与 69-PATTERNS 记录一致）
**Pattern extraction date:** 2026-08-19

**给 planner 的五个本 mapping 新增事实（RESEARCH 未记或需修正）：**
1. **缓存失效时序**：迁移在 `db.NewDatabase`（core.go:258，由 :206 `initDBAndData` 触发）内执行，`c.Cache` 在 core.go:343 才创建——RESEARCH A4 的「迁移函数内 DeleteByPattern」不可行，须改为 changed 标志 + cache 就绪后失效（方案 A/B 见 migration_208 节）
2. **统计卡轻请求零改动确认**：`EmailConfigListParams.status?: number`（notificationConfigApi.ts:21-26）与 `APINotificationConfigListParams.status`（:160 附近）均已存在——A5 假设实证通过，前端直接传 status 即可
3. **useRouteTabs 实际路径**为 `src/components/layout/shared/useRouteTabs.ts:97-99`（RESEARCH 写 `hooks/useRouteTabs.ts:98`），`key: location.pathname` 已复核
4. **migration 编号冲突**：Phase 69（planning 中）同样规划 `migration_208_dict_seed.go`；本 phase 实测目录现最大 207，执行时以实际目录状态定编号，先落者得 208
5. **两处引用修正**：(a) user 页 Sider 源码 `width={250}` 但 CSS `.xr-dept-panel` 覆写为 300（index.css:5243-5247），SettingsShell 按 UI-SPEC 取 220，勿照抄；(b) UI-SPEC IC-2 深色预览内容条注释「`var(--sidebar-accent)`（#E0E0B0 系）」与令牌实际值不符——`#e0e0b0` 是 `--sidebar-text-active`（index.css:68），`--sidebar-accent` 是铜金 `#c09058`（:69），按文字意图应取前者
