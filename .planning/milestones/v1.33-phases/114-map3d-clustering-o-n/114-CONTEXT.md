# Phase 114: map3d-clustering（地图聚类 O(n²) 消除） - Context

**Gathered:** 2026-09-12
**Status:** Ready for planning
**Mode:** Auto-generated (infrastructure phase — pure performance refactor, all-technical success criteria, no user-facing design decisions)

<domain>
## Phase Boundary

湖北地图（HubeiMap/HubeiMapGL）在千级楼宇点位与缩放/倾斜切换时不再出现秒级主线程卡死——聚类算法单遍化 + 像素网格分桶，两份复制实现合并为单一共享函数并有单元测试守护（H-2 全库最重 JS 热点）。范围 = MAP3D-01~06（聚类单遍化 / 40px 网格分桶 / 共享工具函数 + 单元测试 / 5 道 filter 收敛 useMemo / 事件监听 cleanup / 死组件删除）。不含 @uiw/react-baidu-map 依赖移除（Phase 120 DEAD-02，依赖本 phase 的 MAP3D-06 解锁）。

</domain>

<decisions>
## Implementation Decisions

### Claude's Discretion
All implementation choices are at Claude's discretion — pure infrastructure/performance phase. Use REQUIREMENTS.md MAP3D-01~06 条目、ROADMAP success criteria 与 D-05 复用范本（info-points:603 Map 索引风格）指导实现。

</decisions>

<code_context>
## Existing Code Insights

### Reusable Assets
- `info-points:603` Map 索引范本（D-05 锁定的既有正确模式）
- 前端测试基建已就绪（Vitest，`npx vitest run`），聚类共享函数按 MAP3D-03 要求附单元测试（聚类结果一致性，D-04 口径）

### Established Patterns
- O(n²) 双循环 + 内层 `pointToOverlayPixel` 地图 API 调用：`HubeiMap.tsx:264-318` 与 `HubeiMapGL.tsx:279-321` 为两份近似复制实现（差异仅在 BMap vs BMapGL namespace）
- HubeiMap.tsx:223 已有 zoomend 监听但无 cleanup；HubeiMapGL.tsx:150-151 zoomend/tiltend 均无 cleanup
- 死组件 `BuildingMarkers.tsx` / `CityMarkers.tsx` 位于同目录 `building-spaces-3d/components/`

### Integration Points
- 聚类共享函数建议落位 `building-spaces-3d/` 下 utils（与既有 utils.ts 同目录）
- 两页聚类视觉结果必须一致（success criteria 2）——共享函数是唯一聚类实现，渲染层各自消费

</code_context>

<specifics>
## Specific Ideas

No specific requirements — open to standard approaches. REQUIREMENTS.md 已量化关键参数：40px 像素网格分桶、单遍 `Map<id,pixel>` 预计算、useMemo deps `[buildings]`。

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope。

</deferred>
