/**
 * NetworkDevice 设备管理页面（v1.24 HYG-04 拆分后的页面装配层）。
 *
 * 已拆出：列定义工厂 → components/DeviceTableColumns.tsx；搜索区 schema → searchFields.ts；
 * 数据操作 → hooks/useDeviceActions.tsx；行操作 + 行操作权限门控 → hooks/useDeviceRowActions.tsx；
 * 页面 JSX 外壳 → components/DeviceTablePanel.tsx。
 * components/DeviceModals.tsx **保持不拆**（v1.22 刚拆，68.1 验收基线）。
 *
 * 三条必须保持的引用/时序纪律（v1.23 FAIL-1 同源风险）：
 *   1. `deptFilters` 与 `columns` 的 useMemo **留在本组件**——引用不稳 → 页码被打回第 1 页；
 *   2. `columns` 的依赖数组**故意不完整**（见下方 eslint-disable）：`collectingDeviceId` 变化会
 *      重建整列定义，这是已知的性能债，**拆分时不得顺手"修正"**（会引入无限渲染）；
 *   3. `reloadRef` 是**外部注入的 ref**（不是 state）：由页面持有、由 DeviceTablePanel 的
 *      PageTable children 渲染函数回填 `ctx.loadData`，提交/删除后据此刷新列表。
 */

import { useState, useCallback, useEffect, useMemo } from "react";
import type { Key } from "react";
import { useLocation } from "react-router-dom";
import { useNavigate } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import { App, Form } from "antd";
import type { NetworkDevice } from "@/types";
import type { PageResponse } from "@/types";
import { post } from "@/lib/api";
import type {  } from "@/utils/tableHelpers";
import { createSorterMeta } from "@/utils/tableHelpers";
import { useMenuStore } from "@/store/menuStore";
import { useDict } from "@/hooks/useDict";
import { getDeviceTableColumns } from "./components/DeviceTableColumns";
import DeviceTablePanel from "./components/DeviceTablePanel";
import type { DeviceModalsProps } from "./components/DeviceModals";
import { DEVICE_PERMISSION_CODES, DEVICE_TYPE_OPTIONS } from "./constants";
import { resolveDeviceActionPerm } from "./hooks/useDeviceRowActions";
import { useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
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
  // ⚠️ Phase 86 W3：这处直读是**只读入口**（MAC 历史跳转）的可见性判断，
  // 不是写入口门控（写入口的门控判定由下方 useActionPermission 承担）。
  const menuPermissions = useMenuStore((s) => s.permissions);
  const canViewMACHistory = menuPermissions.includes("network:mac:list");

  // Phase 86 W3（RBAC-02）：工具栏写入口的三态门控（批量删除 / 批量导出 / 快速创建+手动新增）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  // ⚠️ 门控键沿用**降级解析**（resolveDeviceActionPerm，87 播种前多数角色只有页面级码）：
  //    环境已播种动作级码 → 用动作码；否则 → 退回页面级码，与拆分前行为一致。
  //    不得在此改为硬绑动作码（会让现役角色看不到入口，比后端 fail-closed 更严）。
  const permGate = useActionPermission();
  const readyPermissions = permGate.state === "ready" ? menuPermissions : [];
  const batchDeletePerm = permGate.decide(
    resolveDeviceActionPerm(readyPermissions, DEVICE_PERMISSION_CODES.remove)
  );
  const addPerm = permGate.decide(
    resolveDeviceActionPerm(readyPermissions, DEVICE_PERMISSION_CODES.add)
  );
  // 导出：`networkDevice.export` 是 85-03 新增码（87 播种前**无授权来源**）→
  // 不走降级解析，直接按码门控：ready 无码 → 不渲染，与后端 fail-closed 同向
  // （若也降级到页面级码，窗口期用户会看到导出按钮但点了必 403 —— 反向不一致）。
  const exportPerm = permGate.decide(PERMISSION_CODES.networkDevice.export);

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

  // v1.24 HYG-04 D3/D5：数据操作抽到 hooks/useDeviceActions.tsx；
  // 行操作 + 行操作权限门控抽到 hooks/useDeviceRowActions.tsx
  //（行操作接入 Phase 78 的 useActionPermission 三态门控）
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

  // 左侧部门树选择（引用每次渲染新建，与拆分前的内联箭头函数语义一致）
  const handleDeptSelect = (selectedKeys: Key[]) => {
    if (selectedKeys.length > 0) {
      setSelectedDeptId(selectedKeys[0] as string);
    } else {
      setSelectedDeptId("");
    }
  };

  // 三弹窗 props（DeviceModals 本 phase 不拆，props 组装留在页面侧）
  const modalsProps: DeviceModalsProps = {
    quickCreateModalVisible,
    quickCreateForm,
    credentials,
    probeResult,
    probing,
    submitting,
    departments,
    onProbe: handleProbe,
    onQuickCreateOk: async () => {
      setSubmitting(true);
      try {
        await handleQuickCreate(reloadRef.current as () => void);
      } finally {
        setSubmitting(false);
      }
    },
    onQuickCreateCancel: () => {
      closeQuickCreateModal();
      quickCreateForm.resetFields();
    },
    onClearProbeResult: () => setProbeResult(null),
    modalVisible,
    editingDevice,
    editForm,
    deviceTypeOptions,
    onEditOk: async () => {
      setSubmitting(true);
      try {
        await handleCreate(reloadRef.current as () => void);
      } finally {
        setSubmitting(false);
      }
    },
    onEditCancel: () => {
      setModalVisibleState(false);
      editForm.resetFields();
    },
    detailModalVisible,
    viewingDevice,
    canViewMACHistory,
    navigate,
    getDeviceTypeText,
    onCloseDetail: () => setDetailModalVisible(false),
  };

  return (
    <DeviceTablePanel
      selectedDeptId={selectedDeptId}
      onDeptSelect={handleDeptSelect}
      request={request}
      columns={columns}
      searchFields={searchFields}
      sorterMetas={sorterMetas}
      extraFilters={deptFilters}
      onBatchDelete={async (keys) => {
        await post("/network/devices/batch-delete", { ids: keys });
        void loadStatistics();
      }}
      rowActions={buildRowActions}
      reloadRef={reloadRef}
      statistics={statistics}
      exportPage={exportPage}
      onQuickCreate={openQuickCreateModal}
      onManualCreate={() => openModal()}
      permission={{
        batchDelete: batchDeletePerm,
        export: exportPerm,
        add: addPerm,
      }}
      batchExportModal={{
        visible: batchModalVisible,
        loading: batchExporting,
        onConfirm: handleBatchExport,
        onCancel: () => setBatchModalVisible(false),
      }}
      modals={modalsProps}
    />
  );
};

export default DeviceManagement;
