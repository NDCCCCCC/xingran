/**
 * 网络设备页的页面外壳（v1.24 HYG-04 / D4）。
 *
 * 从 index.tsx 原样搬迁（零行为变化）：Layout / 部门树 / PageTable / 工具栏 /
 * children 业务区（统计卡片 + 已选择 Alert）/ 批量导出 Modal / 三弹窗 的
 * 结构、props、文案、样式逐字保留。
 *
 * ⚠️ reloadRef 回填链路（本刀最关键的一条）：
 *   PageTable 的 children 是**渲染函数**，它在渲染期把 `ctx.loadData` 写回
 *   `reloadRef.current`（见下方 children 实现）。reloadRef 由**页面持有**
 *   （index.tsx）并以 props 注入本组件 —— 即「外部注入 ref」语义：
 *   页面侧的提交/删除 handler 通过 `reloadRef.current` 触发刷新，
 *   本组件只负责**回填**。⚠️ 不得改成 state（会引入渲染时序差：
 *   渲染期写入 → 提交回调读取之间存在中间态，且会多一次渲染）。
 *
 * 本组件是纯展示层：不持有业务 state、不发请求、不做权限判定
 * （行操作的权限门控在 hooks/useDeviceRowActions.tsx 内完成）。
 *
 * Phase 86 W3（RBAC-02）：工具栏的**写入口**（批量删除 / 批量导出 / 快速创建 + 手动新增）
 * 也需要三态门控。为守住「纯展示层不做判定」的纪律，判定结果由**页面层**
 * （`index.tsx` 的 `useActionPermission` + 降级解析）算好后经 `permission` prop 注入；
 * 本组件只按 `visible` 决定渲染与否、按 `disabled`/`hint` 透传禁用态与提示。
 */

import type { Key } from "react";
import { Alert, Button, Layout } from "antd";
import { PlusOutlined, ThunderboltOutlined } from "@ant-design/icons";
import type { NetworkDevice } from "@/types";
import type { ColumnsType } from "antd/es/table";
import { BatchExportModal, PageTable } from "@/components/shared";
import type { PageTableRenderContext, PageTableRequest, SearchField } from "@/components/shared";
import type { ActionButton } from "@/components/shared/ActionButtons";
import type { ActionPermissionDecision } from "@/components/shared";
import NetworkExport from "@/components/shared/NetworkExport";
import type { SorterMeta } from "@/utils/tableHelpers";
import DeptTreePanel from "./DeptTreePanel";
import DeviceStatsCards from "./DeviceStatsCards";
import DeviceModals from "./DeviceModals";
import type { DeviceModalsProps } from "./DeviceModals";
import type { DeviceStatistics } from "../hooks/useDeviceData";

export interface DeviceTablePanelProps {
  // —— 左侧部门树 ——
  /** 当前选中的部门 id */
  selectedDeptId: string;
  /** 部门树选择回调（selectedKeys 为空表示取消选择） */
  onDeptSelect: (selectedKeys: Key[]) => void;
  // —— 列表 ——
  /** 列表请求（PageTable 合参直连后端） */
  request: PageTableRequest<NetworkDevice>;
  /** 列定义（引用稳定的 useMemo 结果） */
  columns: ColumnsType<NetworkDevice>;
  /** 搜索区 schema */
  searchFields: SearchField[];
  /** 服务端排序元数据 */
  sorterMetas: Array<SorterMeta<NetworkDevice> | undefined>;
  /** 额外固定筛选（部门树联动；引用变化 → 回第 1 页重载） */
  extraFilters: Record<string, unknown>;
  /** 批量删除回调 */
  onBatchDelete: (keys: Key[]) => Promise<void>;
  /** 行操作（PageTable ctx 注入 loadData 后组装） */
  rowActions: (record: NetworkDevice, loadData: () => void) => ActionButton[];
  /** reloadRef：外部（页面）持有的 ref，本组件在 children 渲染期回填 ctx.loadData */
  reloadRef: { current: (() => void) | null };
  // —— 统计与导出 ——
  /** 统计卡片数据 */
  statistics: DeviceStatistics;
  /** NetworkExport 的 current/pageSize（request 回填） */
  exportPage: { current: number; pageSize: number };
  // —— 工具栏 ——
  /** 打开快速创建弹窗 */
  onQuickCreate: () => void;
  /** 打开手动新增弹窗 */
  onManualCreate: () => void;
  /**
   * Phase 86 W3（RBAC-02）：工具栏写入口的三态门控决策（由页面层注入）。
   * - batchDelete → 批量删除（`networkDevice.delete`）
   * - export → 批量导出（`networkDevice.export`，门控点 = `<NetworkExport>` 元素本身）
   * - add → 快速创建 + 手动新增（`networkDevice.add`）
   */
  permission: {
    batchDelete: ActionPermissionDecision;
    export: ActionPermissionDecision;
    add: ActionPermissionDecision;
  };
  // —— 弹窗 ——
  /** 批量导出弹窗 */
  batchExportModal: {
    visible: boolean;
    loading: boolean;
    onConfirm: (entityTypes: string[]) => Promise<void>;
    onCancel: () => void;
  };
  /** 三弹窗（快速创建/编辑/详情）props */
  modals: DeviceModalsProps;
}

const DeviceTablePanel: React.FC<DeviceTablePanelProps> = ({
  selectedDeptId,
  onDeptSelect,
  request,
  columns,
  searchFields,
  sorterMetas,
  extraFilters,
  onBatchDelete,
  rowActions,
  reloadRef,
  statistics,
  exportPage,
  onQuickCreate,
  onManualCreate,
  permission,
  batchExportModal,
  modals,
}) => (
  <Layout style={{ background: "var(--theme-bg-secondary)", minHeight: "calc(100vh - 64px)" }}>
    {/* 左侧部门树 */}
    <DeptTreePanel selectedDeptId={selectedDeptId} onSelect={onDeptSelect} />

    <PageTable<NetworkDevice>
      request={request}
      columns={columns}
      rowKey={(r) => r.id}
      searchFields={searchFields}
      sorterMetas={sorterMetas}
      extraFilters={extraFilters}
      virtual
      scroll={{ x: 1600, y: 600 }}
      batchDelete={
        // Phase 86 W3：批量删除 = 写入口（networkDevice.delete）。
        // PageTable 的 batchDelete 是「传/不传」二值 → ready 无码时整块 rowSelection 不渲染；
        // 未就绪时仍传（入口在 DOM），但禁用的语义由 PageTable 内部选择框承担
        // （此处至少保证不因未就绪而消失）。
        permission.batchDelete.visible
          ? {
              target: "设备",
              onDelete: onBatchDelete,
            }
          : undefined
      }
      rowActions={(record) => rowActions(record, reloadRef.current as () => void)}
      toolbar={
        <>
          {/* Phase 86 W3：批量导出 —— 门控点 = `<NetworkExport>` 组件元素本身
              （「导出」按钮在 `NetworkExport.tsx:120` 组件内部，页面上无独立触发按钮）。
              ready 无 networkDevice.export → 整个组件（含其内部「导出」按钮）不在 DOM。 */}
          {permission.export.visible && (
            <span
              title={permission.export.hint}
              aria-disabled={permission.export.disabled || undefined}
            >
              <NetworkExport
                entityType="devices"
                entityName="网络设备"
                current={exportPage.current}
                pageSize={exportPage.pageSize}
              />
            </span>
          )}
          {/* Phase 86 W3：快速创建 + 手动新增 —— 写入口（networkDevice.add） */}
          {permission.add.visible && (
            <>
              <Button
                aria-label="快速创建设备"
                title={permission.add.hint ?? "快速创建设备"}
                type="primary"
                icon={<ThunderboltOutlined />}
                onClick={onQuickCreate}
                disabled={permission.add.disabled}
              >
                快速创建
              </Button>
              <Button
                type="default"
                icon={<PlusOutlined />}
                onClick={onManualCreate}
                aria-label="新增设备"
                title={permission.add.hint ?? "新增设备"}
                disabled={permission.add.disabled}
              >
                手动新增
              </Button>
            </>
          )}
        </>
      }
    >
      {(ctx: PageTableRenderContext<NetworkDevice>) => {
        // loadData / searchForm 转发（提交后刷新 + children 内接线）
        reloadRef.current = ctx.loadData;
        return (
          <>
            <DeviceStatsCards statistics={statistics} />
            {ctx.selectedRowKeys.length > 0 && (
              <Alert
                title={
                  <span>
                    已选择 <strong>{ctx.selectedRowKeys.length}</strong> 个设备，
                    <Button
                      type="link"
                      size="small"
                      onClick={ctx.clearSelection}
                      style={{ padding: 0 }}
                    >
                      取消选择
                    </Button>
                  </span>
                }
                type="info"
                showIcon
                style={{ marginBottom: 16 }}
              />
            )}
          </>
        );
      }}
    </PageTable>

    {/* 批量导出 Modal */}
    <BatchExportModal
      visible={batchExportModal.visible}
      onConfirm={batchExportModal.onConfirm}
      onCancel={batchExportModal.onCancel}
      loading={batchExportModal.loading}
    />

    {/* 三弹窗（快速创建/编辑/详情） */}
    <DeviceModals {...modals} />
  </Layout>
);

export default DeviceTablePanel;
