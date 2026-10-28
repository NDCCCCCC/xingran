/**
 * NetworkDevice 设备管理页面
 *
 * Phase 67-T3: 迁移 66 契约组件（PageTable 单实例 + 子组件拆分）
 * PR 差异记录（对照迁移前 1282 行版本）：
 * - Table+useTableManager 双实例化骨架 → PageTable（request 直连
 *   /network/devices/list，搜索值/extraFilters/排序由 PageTable 合参）；
 *   usePagination+setTotal 外部分页链路删除（PageTable 内建 total 承接，SC2）。
 * - 单删 Modal.confirm（cancelButton 防抖 confirmInstance.update 写法）→
 *   createDeleteAction → ConfirmDelete（65 模板文案，B 类对齐）。
 * - 批删 confirmDelete 手工按钮 → batchDelete={{onDelete,target:"设备"}}
 *   （INV-8 逐字文案：确认批量删除/将删除选中的 {N} 项，删除后不可恢复，
 *   是否继续？/删除/取消/danger）。
 * - 搜索卡 5 字段 → searchFields（查询→搜索 B 类；重置/刷新内建）；
 *   deviceType options 保留 useDict 字典回退静态语义。
 * - 左侧部门树 → extraFilters={deptFilters}（useMemo 引用稳定）联动，
 *   handleDeptSelect 仅 setState，刷新由 PageTable extraFilters 触发。
 * - virtual + scroll={{x:1600,y:600}} 经 rest 透传（PAGE_TABLE 契约不吞）。
 * - 统计刷新：原 request 内 loadStatistics() 保留——任何列表刷新均联动统计。
 * - 已选择 Alert → children ctx.selectedRowKeys/clearSelection。
 * - 三弹窗纯搬迁至 components/DeviceModals.tsx（Modal 原样，未改 FormModal——
 *   原「提交中禁用取消按钮」防抖是契约外增强，保持原状）。
 */

import { useState, useCallback, useEffect, useMemo } from "react";
import { useLocation } from "react-router-dom";
import { useNavigate } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import { Layout, App, Form, Button, Alert } from "antd";
import { PlusOutlined, ThunderboltOutlined } from "@ant-design/icons";
import type { NetworkDevice } from "@/types";
import type { PageResponse } from "@/types";
import { post } from "@/lib/api";
import NetworkExport from "@/components/shared/NetworkExport";
import { PageTable } from "@/components/shared";
import { BatchExportModal } from "@/components/shared";
import type {  } from "@/utils/tableHelpers";
import { createSorterMeta } from "@/utils/tableHelpers";
import { useMenuStore } from "@/store/menuStore";
import { useDict } from "@/hooks/useDict";
import DeptTreePanel from "./components/DeptTreePanel";
import DeviceStatsCards from "./components/DeviceStatsCards";
import DeviceModals from "./components/DeviceModals";
import { getDeviceTableColumns } from "./components/DeviceTableColumns";
import { DEVICE_TYPE_OPTIONS } from "./constants";
import { getOptionLabel } from "./utils";
import { buildDeviceSearchFields } from "./searchFields";
import { useDeviceData } from "./hooks/useDeviceData";
import { useDeviceModals } from "./hooks/useDeviceModals";
import { useDeviceActions } from "./hooks/useDeviceActions";

// ==================== 主组件 ====================

const DeviceManagement = () => {
  const { message } = App.useApp();
  const navigate = useNavigate();

  // 权限(14-05b:查看 MAC 历史联动入口)— 与 history 页主权限点 network:mac:list 保持一致
  const menuPermissions = useMenuStore((s) => s.permissions);
  const canViewMACHistory = menuPermissions.includes("network:mac:list");

  // 表单实例
  const [quickCreateForm] = Form.useForm();
  const [editForm] = Form.useForm();

  // 自定义状态
  const location = useLocation();
  const [selectedDeptId, setSelectedDeptId] = usePersistedStateController<string>({
    keyPrefix: location.pathname,
    keySuffix: "selectedDeptId",
    defaultValue: "",
  });
  const [collectingDeviceId, setCollectingDeviceId] = useState<string | null>(null);
  const [batchModalVisible, setBatchModalVisible] = useState(false);
  const [batchExporting, setBatchExporting] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  // NetworkExport current/pageSize（原 paginationProps 承接；request 回填）
  const [exportPage, setExportPage] = useState({ current: 1, pageSize: 10 });

  // Phase 69 DICT-03: 设备类型下拉迁 useDict("network_device_type")——字典管理页改 label
  // 后搜索/编辑下拉与表格 Tag 随之变化；字典空态/接口异常时回退静态 DEVICE_TYPE_OPTIONS。
  // VENDOR_OPTIONS 无对应字典组，保持静态现状（明确不迁）。
  const { data: deviceTypeDict = [] } = useDict("network_device_type");

  // 设备类型 label 渲染：优先字典 dictLabel，空态/未命中回退静态映射（dedicated-lines 三件套）
  const getDeviceTypeText = useCallback(
    (type: string) => {
      const dictItem = deviceTypeDict.find((d) => d.dictValue === type);
      return dictItem?.dictLabel || getOptionLabel(DEVICE_TYPE_OPTIONS, type);
    },
    [deviceTypeDict]
  );

  // 搜索/编辑下拉共用 options：字典优先，回退静态
  const deviceTypeOptions = useMemo(
    () =>
      deviceTypeDict.length > 0
        ? deviceTypeDict.map((d) => ({ label: d.dictLabel, value: d.dictValue }))
        : DEVICE_TYPE_OPTIONS.map((opt) => ({ label: opt.label, value: opt.value })),
    [deviceTypeDict]
  );

  const {
    departments,
    credentials,
    statistics,
    loadCredentials,
    loadStatistics,
    ensureCredential,
  } = useDeviceData();

  const {
    quickCreateModalVisible,
    detailModalVisible,
    viewingDevice,
    probeResult,
    probing,
    setQuickCreateModalVisible,
    setDetailModalVisible,
    setProbeResult,
    setProbing,
    openDetailModal,
    closeQuickCreateModal,
  } = useDeviceModals();

  // 服务端排序:field 对应后端 networkDeviceAllowedSortFields 白名单 key
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<NetworkDevice>("deviceName"),
      createSorterMeta<NetworkDevice>("deviceType"),
      createSorterMeta<NetworkDevice>("vendor"),
      createSorterMeta<NetworkDevice>("model"),
      createSorterMeta<NetworkDevice>("ipAddress", "string"),
      createSorterMeta<NetworkDevice>("port", "number"),
      createSorterMeta<NetworkDevice>("status", "number"),
      createSorterMeta<NetworkDevice>("lastSeenAt", "date"),
      createSorterMeta<NetworkDevice>("createdAt", "date"),
    ],
    []
  );

  // 列表请求：搜索值/extraFilters/排序/分页由 PageTable 合参直连
  const request = useCallback(
    async (params: Record<string, unknown>) => {
      const result = (await post<PageResponse<NetworkDevice>>(
        "/network/devices/list",
        params
      )) as { data: { list: NetworkDevice[]; total: number } };
      // 统计联动保留原语义：任何列表刷新均带动统计刷新
      void loadStatistics();
      setExportPage({ current: Number(params.current) || 1, pageSize: Number(params.pageSize) || 10 });
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    [loadStatistics]
  );

  // 部门树联动：引用稳定对象，变化 → PageTable applyFilters 回第 1 页
  const deptFilters = useMemo(
    () => (selectedDeptId ? { deptId: selectedDeptId } : {}),
    [selectedDeptId]
  );

  // ==================== 数据操作 ====================

  // 编辑弹窗开关（原 useTableManager editModalVisible 承接：本地 state 桥接）
  const [modalVisible, setModalVisibleState] = useState(false);
  const [editingDevice, setEditingDevice] = useState<NetworkDevice | null>(null);

  // 批量导出弹窗触发按钮（原 toolbar 位）
  // ⚠️ reloadRef 是**外部注入的 ref**（不是 state）：PageTable 的 children 渲染函数
  //    在渲染期把 ctx.loadData 回填进它（见 components/DeviceTablePanel.tsx），
  //    提交/删除成功后由 handlers 读它来刷新列表。不得改成 state（会引入渲染时序差）。
  const [reloadRef] = useState<{ current: (() => void) | null }>({ current: null });

  // v1.24 HYG-04 D3：数据操作 + 行操作抽到 hooks/useDeviceActions.ts
  //（行操作的权限门控随之接入 Phase 78 的 useActionPermission 三态门控）
  const {
    handleProbe,
    handleQuickCreate,
    handleCreate,
    handleBatchExport,
    handleCollectPorts,
    openModal,
    openQuickCreateModal,
    buildRowActions,
  } = useDeviceActions({
    quickCreateForm,
    editForm,
    editingDevice,
    collectingDeviceId,
    canViewMACHistory,
    navigate,
    message,
    loadCredentials,
    loadStatistics,
    ensureCredential,
    setQuickCreateModalVisible,
    setProbeResult,
    setProbing,
    closeQuickCreateModal,
    openDetailModal,
    setCollectingDeviceId,
    setModalVisible: setModalVisibleState,
    setEditingDevice,
    setBatchModalVisible,
    setBatchExporting,
  });

  // mount 统计首载（列表首载由 PageTable extraFilters effect 承接）
  useEffect(() => {
    void loadStatistics();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const columns = useMemo(
    () =>
      getDeviceTableColumns({
        navigate,
        handleCollectPorts,
        openDetailModal,
        openModal,
        collectingDeviceId,
        canViewMACHistory,
        getDeviceTypeText,
      }),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [collectingDeviceId, canViewMACHistory, deviceTypeDict]
  );

  const searchFields = buildDeviceSearchFields(deviceTypeOptions);

  return (
    <Layout style={{ background: "var(--theme-bg-secondary)", minHeight: "calc(100vh - 64px)" }}>
      {/* 左侧部门树 */}
      <DeptTreePanel
        selectedDeptId={selectedDeptId}
        onSelect={(selectedKeys) => {
          if (selectedKeys.length > 0) {
            setSelectedDeptId(selectedKeys[0] as string);
          } else {
            setSelectedDeptId("");
          }
        }}
      />

      <PageTable<NetworkDevice>
        request={request}
        columns={columns}
        rowKey={(r) => r.id}
        searchFields={searchFields}
        sorterMetas={sorterMetas}
        extraFilters={deptFilters}
        virtual
        scroll={{ x: 1600, y: 600 }}
        batchDelete={{
          target: "设备",
          onDelete: async (keys) => {
            await post("/network/devices/batch-delete", { ids: keys });
            void loadStatistics();
          },
        }}
        rowActions={(record) => buildRowActions(record, reloadRef.current as () => void)}
        toolbar={
          <>
            <NetworkExport
              entityType="devices"
              entityName="网络设备"
              current={exportPage.current}
              pageSize={exportPage.pageSize}
            />
            <Button
              aria-label="快速创建设备"
              title="快速创建设备"
              type="primary"
              icon={<ThunderboltOutlined />}
              onClick={openQuickCreateModal}
            >
              快速创建
            </Button>
            <Button
              type="default"
              icon={<PlusOutlined />}
              onClick={() => openModal()}
              aria-label="新增设备"
              title="新增设备"
            >
              手动新增
            </Button>
          </>
        }
      >
        {(ctx) => {
          // loadData / searchForm 转发（提交后刷新 + children 内接线）
          reloadRef.current = ctx.loadData;
          return (
            <>
              <DeviceStatsCards statistics={statistics} />
              {ctx.selectedRowKeys.length > 0 && (
                <Alert
                  message={
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
        visible={batchModalVisible}
        onConfirm={handleBatchExport}
        onCancel={() => setBatchModalVisible(false)}
        loading={batchExporting}
      />

      {/* 三弹窗（快速创建/编辑/详情） */}
      <DeviceModals
        quickCreateModalVisible={quickCreateModalVisible}
        quickCreateForm={quickCreateForm}
        credentials={credentials}
        probeResult={probeResult}
        probing={probing}
        submitting={submitting}
        departments={departments}
        onProbe={handleProbe}
        onQuickCreateOk={async () => {
          setSubmitting(true);
          try {
            await handleQuickCreate(reloadRef.current as () => void);
          } finally {
            setSubmitting(false);
          }
        }}
        onQuickCreateCancel={() => {
          closeQuickCreateModal();
          quickCreateForm.resetFields();
        }}
        onClearProbeResult={() => setProbeResult(null)}
        modalVisible={modalVisible}
        editingDevice={editingDevice}
        editForm={editForm}
        deviceTypeOptions={deviceTypeOptions}
        onEditOk={async () => {
          setSubmitting(true);
          try {
            await handleCreate(reloadRef.current as () => void);
          } finally {
            setSubmitting(false);
          }
        }}
        onEditCancel={() => {
          setModalVisibleState(false);
          editForm.resetFields();
        }}
        detailModalVisible={detailModalVisible}
        viewingDevice={viewingDevice}
        canViewMACHistory={canViewMACHistory}
        navigate={navigate}
        getDeviceTypeText={getDeviceTypeText}
        onCloseDetail={() => setDetailModalVisible(false)}
      />
    </Layout>
  );
};

export default DeviceManagement;
