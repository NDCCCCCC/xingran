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
import { Layout, App, Form, Button, Alert, Tag } from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  ThunderboltOutlined,
  CloudUploadOutlined,
  EditOutlined,
  EyeOutlined,
  HistoryOutlined,
} from "@ant-design/icons";
import type { NetworkDevice } from "@/types";
import type { PageResponse } from "@/types";
import { post } from "@/lib/api";
import { batchExport } from "@/lib/api/networkApi";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import { createDeleteAction } from "@/components/shared/ActionButtons";
import NetworkExport from "@/components/shared/NetworkExport";
import { PageTable } from "@/components/shared";
import { BatchExportModal } from "@/components/shared";
import type { ActionButton } from "@/components/shared/ActionButtons";
import type { ColumnsType } from "antd/es/table";
import type { SorterMeta } from "@/utils/tableHelpers";
import { createSorterMeta } from "@/utils/tableHelpers";
import { useMenuStore } from "@/store/menuStore";
import { useDict } from "@/hooks/useDict";
import DeptTreePanel from "./components/DeptTreePanel";
import DeviceStatsCards from "./components/DeviceStatsCards";
import DeviceModals from "./components/DeviceModals";
import {
  VENDOR_OPTIONS,
  DEVICE_TYPE_OPTIONS,
  STATUS_OPTIONS,
  DEVICE_TYPE_TAG_COLOR,
  VENDOR_TAG_COLOR,
} from "./constants";
import { formatDateTime, getOptionLabel, getStatusColor } from "./utils";
import { useDeviceData } from "./hooks/useDeviceData";
import { useDeviceModals, type ProbeResult } from "./hooks/useDeviceModals";

// ==================== 表格列定义 ====================

interface DeviceTableColumnsProps {
  navigate: (path: string) => void;
  handleCollectPorts: (device: NetworkDevice) => void;
  openDetailModal: (device: NetworkDevice) => void;
  openModal: (record?: NetworkDevice) => void;
  collectingDeviceId: string | null;
  canViewMACHistory: boolean;
  /** Phase 69 DICT-03: 设备类型 label 渲染（优先字典，回退静态 DEVICE_TYPE_OPTIONS） */
  getDeviceTypeText: (type: string) => string | undefined;
}

/** 列 sorter:true 由 PageTable/useTableManager 排序管线承接；原手工 sortOrder 视觉字段随契约删除 */
function getDeviceTableColumns(props: DeviceTableColumnsProps): ColumnsType<NetworkDevice> {
  const { navigate, getDeviceTypeText } = props;

  return [
    {
      title: "设备名称",
      dataIndex: "deviceName",
      key: "deviceName",
      width: 200,
      sorter: true,
      render: (deviceName: string, record: NetworkDevice) => (
        <a
          onClick={() => navigate(`/network/ports?deviceId=${record.id}`)}
          style={{ color: "var(--theme-info)", cursor: "pointer" }}
        >
          {deviceName}
        </a>
      ),
    },
    {
      title: "设备类型",
      dataIndex: "deviceType",
      key: "deviceType",
      width: 120,
      sorter: true,
      render: (deviceType: string) => (
        <Tag color={DEVICE_TYPE_TAG_COLOR}>{getDeviceTypeText(deviceType)}</Tag>
      ),
    },
    {
      title: "厂商",
      dataIndex: "vendor",
      key: "vendor",
      width: 100,
      sorter: true,
      render: (vendor: string) => (
        <Tag color={VENDOR_TAG_COLOR}>{getOptionLabel(VENDOR_OPTIONS, vendor)}</Tag>
      ),
    },
    {
      title: "型号",
      dataIndex: "model",
      key: "model",
      width: 150,
      sorter: true,
    },
    {
      title: "IP地址",
      dataIndex: "ipAddress",
      key: "ipAddress",
      width: 140,
      sorter: true,
    },
    {
      title: "端口",
      dataIndex: "port",
      key: "port",
      width: 80,
      sorter: true,
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 90,
      sorter: true,
      render: (status: number) => (
        <Tag color={getStatusColor(status)}>{getOptionLabel(STATUS_OPTIONS, status)}</Tag>
      ),
    },
    { title: "部门", dataIndex: "deptName", key: "deptName", width: 180 },
    {
      title: "最后连接",
      dataIndex: "lastSeenAt",
      key: "lastSeenAt",
      width: 180,
      sorter: true,
      render: (lastSeenAt: string) => formatDateTime(lastSeenAt),
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 180,
      sorter: true,
      render: (createdAt: string) => formatDateTime(createdAt),
    },
  ];
}

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

  // 探测设备
  const handleProbe = async () => {
    try {
      const values = await quickCreateForm.validateFields(["ipAddress", "credentialId"]);
      setProbing(true);
      setProbeResult(null);

      const result = await post<ProbeResult>("/network/devices/discover", {
        ipAddress: values.ipAddress,
        credentialId: values.credentialId,
      });

      if (result.data?.success) {
        setProbeResult(result.data);
        handleSuccess("设备探测");
      } else {
        handleApiError(result.data?.message || "探测失败", "探测");
      }
    } catch (error) {
      if (error && typeof error === "object" && "errorFields" in error) {
        handleApiError(error, "请先填写 IP 地址和授权凭证", false);
      } else {
        handleApiError(error, "探测");
      }
    } finally {
      setProbing(false);
    }
  };

  // 快速创建设备（reloadRef 承接：提交成功后由 ctx 转发的 loadData 刷新）
  const handleQuickCreate = async (loadData: () => void) => {
    try {
      const values = await quickCreateForm.validateFields();

      await post("/network/devices/quick-create", {
        ipAddress: values.ipAddress,
        credentialId: values.credentialId,
        snmpPort: values.snmpPort || 161,
        communities: values.communities || [],
        deptId: values.deptId || undefined,
        location: values.location || "",
        description: values.description || "",
      });

      handleSuccess("设备创建");
      closeQuickCreateModal();
      quickCreateForm.resetFields();
      setProbeResult(null);

      loadData();
      void loadStatistics();
    } catch (error) {
      if (error && typeof error === "object" && "errorFields" in error) {
        return;
      }
      handleApiError(error, "创建");
    }
  };

  // 创建/更新设备
  const handleCreate = async (loadData: () => void) => {
    try {
      const values = await editForm.validateFields();
      if (editingDevice) {
        await post(`/network/devices/${editingDevice.id}/update`, values);
        handleSuccess("更新");
      } else {
        await post("/network/devices", values);
        handleSuccess("创建");
      }
      setModalVisibleState(false);
      editForm.resetFields();
      loadData();
      void loadStatistics();
    } catch (error) {
      if (error && typeof error === "object" && "errorFields" in error) {
        return;
      }
      handleApiError(error, "操作");
    }
  };

  // 编辑弹窗开关（原 useTableManager editModalVisible 承接：本地 state 桥接）
  const [modalVisible, setModalVisibleState] = useState(false);
  const [editingDevice, setEditingDevice] = useState<NetworkDevice | null>(null);

  // 删除设备（createDeleteAction → ConfirmDelete 契约：失败 helper 承接并保持弹窗）
  const handleDelete = async (id: string, loadData: () => void) => {
    await post(`/network/devices/${id}/delete`, {});
    handleSuccess("删除");
    loadData();
    void loadStatistics();
  };

  const handleBatchExport = async (entityTypes: string[]) => {
    setBatchExporting(true);
    try {
      const filename = await batchExport(entityTypes, {}); // 可以根据需要添加筛选条件
      message.success(`批量导出成功，文件: ${filename}`);
      setBatchModalVisible(false);
    } catch (error: any) {
      message.error(`批量导出失败：${error.message}`);
    } finally {
      setBatchExporting(false);
    }
  };

  // 采集设备端口
  const handleCollectPorts = async (device: NetworkDevice) => {
    try {
      setCollectingDeviceId(device.id);
      const result = await post<{
        errorMessage?: string;
        successCount?: number;
        failedCount?: number;
      }>("/network/ports/collect", { deviceId: device.id });

      const data = result.data;
      if (!data) {
        handleApiError("采集失败: 未获取到端口数据", "采集");
      } else if (data.errorMessage) {
        handleApiError(data.errorMessage, "采集", false);
      } else if (data.successCount && data.successCount > 0) {
        handleSuccess(
          `采集成功: 成功 ${data.successCount} 个端口${data.failedCount && data.failedCount > 0 ? `，失败 ${data.failedCount} 个` : ""}`
        );
      } else {
        handleApiError("采集失败: 未获取到端口数据", "采集");
      }
    } catch (error) {
      handleApiError(error, "采集");
    } finally {
      setCollectingDeviceId(null);
    }
  };

  // 打开编辑模态框
  const openModal = (record?: NetworkDevice) => {
    if (record) {
      setEditingDevice(record);
      // 凭证兜底注入(2026-06-30,同 info-points):loadCredentials 异步且 pageSize:50
      // 可能不覆盖当前凭证 → 编辑回填 Select 显示 raw UUID。用 record.credentialName 注入。
      if (record.credentialId) {
        ensureCredential({ id: record.credentialId, credentialName: record.credentialName });
      }
      editForm.setFieldsValue({
        ...record,
        deptId: record.deptId || undefined,
        credentialId: record.credentialId || undefined,
      });
    } else {
      setEditingDevice(null);
      editForm.setFieldsValue({ port: 22, snmpPort: 161, status: 0 });
    }
    setModalVisibleState(true);
    void loadCredentials();
  };

  // 打开快速创建模态框
  const openQuickCreateModal = () => {
    quickCreateForm.resetFields();
    setProbeResult(null);
    setQuickCreateModalVisible(true);
    void loadCredentials();
  };

  // 行操作（PageTable ctx 注入 loadData 后组装；操作列由 PageTable 追加）
  const buildRowActions = useCallback(
    (record: NetworkDevice, loadData: () => void): ActionButton[] => [
      {
        key: "collect-ports",
        label: "采集端口",
        icon: <CloudUploadOutlined />,
        onClick: () => handleCollectPorts(record),
        // 契约 ActionButton 无 loading 位：采集进行中降级为 disabled（记 PR 偏差）
        disabled: collectingDeviceId === record.id,
      },
      {
        key: "detail",
        label: "详情",
        icon: <EyeOutlined />,
        onClick: () => openDetailModal(record),
      },
      // 14-05b:网络设备 → MAC 历史联动入口(D-16 锁定)
      ...(canViewMACHistory
        ? [
            {
              key: "mac-history",
              label: "查看 MAC 历史",
              icon: <HistoryOutlined />,
              onClick: () => navigate(`/network/mac/history?deviceId=${record.id}`),
            },
          ]
        : []),
      {
        key: "edit",
        label: "编辑",
        icon: <EditOutlined />,
        onClick: () => openModal(record),
      },
      createDeleteAction(record.id, () => handleDelete(record.id, loadData), {
        label: record.deviceName,
      }),
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 采集/详情/编辑 handler 均为稳定闭包
    [collectingDeviceId, canViewMACHistory, navigate]
  );

  // 批量导出弹窗触发按钮（原 toolbar 位）
  const [reloadRef] = useState<{ current: (() => void) | null }>({ current: null });

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

  const searchFields = [
    { name: "deviceName", label: "设备名称", type: "input" as const, placeholder: "请输入设备名称" },
    { name: "ipAddress", label: "IP地址", type: "input" as const, placeholder: "请输入IP地址" },
    {
      name: "vendor",
      label: "厂商",
      type: "select" as const,
      placeholder: "请选择厂商",
      options: VENDOR_OPTIONS.map((opt) => ({ label: opt.label, value: opt.value })),
    },
    {
      name: "deviceType",
      label: "设备类型",
      type: "select" as const,
      placeholder: "请选择设备类型",
      options: deviceTypeOptions,
    },
    {
      name: "status",
      label: "状态",
      type: "select" as const,
      placeholder: "请选择状态",
      options: STATUS_OPTIONS.map((opt) => ({ label: opt.label, value: opt.value })),
    },
  ];

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
