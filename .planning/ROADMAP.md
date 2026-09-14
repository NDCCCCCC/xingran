# Roadmap: XingRan-Next

## Milestones

- v1.0 – v1.32 — 已交付里程碑见 `.planning/MILESTONES.md` 与 `.planning/milestones/`
- **v1.33 前端性能治理 (Frontend Performance Remediation)** — Phases 114-120 — **SHIPPED 2026-09-14**

**Active milestone:** None — awaiting next milestone. Start with `/gsd:new-milestone`.

## Phase Numbering

- Integer phases: Sequential across milestones (114-120 used by v1.33, next milestone starts at 121)
- Decimal phases: Reserved for emergency insertions (`/gsd:phase insert`)

## Backlog

- Next milestone: 未定义 — start with `/gsd:new-milestone`
- Candidate inputs: State of `.planning/STATE.md` Deferred Items section

## v1.33 SHIPPED — 前端性能治理 (Frontend Performance Remediation)

**Shipped:** 2026-09-14 | **Phases:** 114-120 | **Plans:** 23 | **Requirements:** 38/38

### Accomplishments

| Phase | Domain | Key Deliverables |
|-------|--------|-----------------|
| 114 | MAP3D | 地图聚类 O(n²)→单遍 Map 预计算 + 40px 网格分桶；cluster.ts 共享纯函数；HubeiMap/HubeiMapGL 合并；5 filter useMemo；zoomend/tiltend cleanup；死组件删除解锁 @uiw |
| 115 | SELECTOR | useTabs 14 字段 + useLayout 10 字段 hook 内 selector 化；路由层 2 文件 + 3D 页 5 文件共 17 selector；页面级 6 处 + NotificationBell 7 项；全库无参 `useXxxStore()` 归零 |
| 116 | DASH | Dashboard N² 重渲染级联消除（H-1）：useWidgetData selector 化 + L1 缓存移出 state + dashboard 9 处订阅收敛 + DashboardGrid 稳定化 |
| 117 | RENDER+BUGFIX | 7 处 columns 工厂 useMemo + 5+ 大数据页 Table virtual + MACHeatmapChart mobile memo + DoorElement snapCoord；BUGFIX-01/02/03 回归测试 |
| 118 | DATA | VDI react-query 去重（4→1 请求）+ 菜单 hydrate-then-revalidate 消除门控 + useColumnConfig 缓存短路 + useHolidayData Promise.all |
| 119 | MISC | Map 索引（useWorkstationView/Tar getSelector）+ 惰性 sessionStorage（useTableManager）+ scroll 短路（TabBar）+ expandedRowRender useCallback |
| 120 | BUNDLE+DEAD | ExcelImport 懒加载统一（9 点）+ iconUtils 假动态导入删除 + 路由 glob phantom chunk 清零 + 4 僵尸依赖移除 + 5 死代码删除 |

### Key Decisions

- **D-01** 全量 33 findings + 8 死代码清理，不分批 defer
- **D-02** 七 gate 不倒退（go build / go test / 后端 coverage ≥78.33 / 前端 45 dirs / lint / type-check / diff coverage）
- **D-03** bundle 基线不倒退（size-limit 门禁 entry gzip 1MB / 全量 2.5MB）
- **D-04** 行为变更（BUGFIX-01..03 / BUNDLE-02 / DATA-02）附回归测试；纯性能重构现有测试零回归
- **D-05** 复用范本：info-points:603 Map 索引 / executions:112 columns useMemo / useRouteTabs:45 selector 风格 / noticeStore P1-M4 缓存出 state
- **D-06** Phase 编号从 114 续编

### Deferred Items (at close)

- ci-lint-hardcoded-ip [awaiting_human_verify] — 修复已应用，等待 CI 确认
- knowledge-base [reference] — 参考项
- Phase 114 HUMAN-UAT [partial] — 1 pending scenario（MAP3D-02 浏览器性能验证）
- Phase 116/117/118/119 verification [human_needed] — 多 phase 人工验证项

**Archive:** `.planning/milestones/v1.33-ROADMAP.md` + `.planning/milestones/v1.33-REQUIREMENTS.md`

---

## v1.32 SHIPPED — 审计驱动的安全与可靠性收尾

**Shipped:** 2026-09-09 | **Phases:** 109-113 | **Plans:** 12 | **Requirements:** 18

TLS 环境变量化 + 裸 goroutine 守护 + 8 项回归守护前置。

---

## v1.31 SHIPPED — 技术债清偿 (Tech Debt Retirement)

**Shipped:** 2026-09-08 | **Phases:** 102-108 | **Plans:** 25

详见 `.planning/milestones/v1.31-ROADMAP.md` + `.planning/MILESTONES.md`

---

*Last updated: 2026-09-14 after v1.33 milestone close*
