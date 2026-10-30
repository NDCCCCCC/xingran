/**
 * 机房设备管理页面 — Phase 68.1 T6b / Phase 93 模块化解耦重构（SPLIT-01）
 *
 * 拆分说明：
 * - 类型与常量映射抽离到 constants.ts
 * - Table 列定义工厂抽离到 columns.tsx
 * - 搜索区字段构建抽离到 searchFields.ts
 * - 卡片网格视图渲染抽离到 components/RoomDeviceCardView.tsx
 * - 数据交互与弹窗操作抽离到 hooks/useRoomDeviceActions.ts
 * - 本入口文件严格保持在 ≤ 350 行，负责布局粘合与顶层渲染。
 */

import { useCallback, useEffect, useRef, useMemo } from "react";
import type { FC, Key } from "react";
import type { FormInstance } from "antd";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import { Button, Space, Card, Alert, Radio, Layout } from "antd";
import {
  PlusOutlined,
  CheckCircleOutlined,
  StopOutlined,
  WarningOutlined,
  AppstoreOutlined,
  TableOutlined,
  ImportOutlined,
  ExportOutlined,
  AppstoreAddOutlined,
} from "@ant-design/icons";
import type { RoomDevice } from "@/types";
import { useSidebarDeptFilter } from "@/hooks/useSidebarDeptFilter";
import { createSorterMeta } from "@/utils/tableHelpers";
import { PageTable, useActionPermission } from "@/components/shared";
import type { PageTableRenderContext } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import ExcelImportLazy from "@/components/shared/ExcelImportLazy";
import ExcelExport from "@/components/shared/ExcelExport";
import { DeptSidebar } from "@/components/operations/DeptSidebar";
import { StatisticsCards } from "@/components/operations/StatisticsCards";

import type { ViewMode } from "./constants";
import { getRoomDeviceColumns, getRoomDeviceRowActions } from "./columns";
import { getRoomDeviceSearchFields } from "./searchFields";
import { RoomDeviceCardView } from "./components/RoomDeviceCardView";
import { RoomDeviceModal } from "./components/RoomDeviceModal";
import { useRoomDeviceActions } from "./hooks/useRoomDeviceActions";

const { Content } = Layout;

const RoomDeviceManagement: FC = () => {
  // Phase 86 W2（RBAC-02）：操作级权限门控（三态）
  const permGate = useActionPermission();
  const permEdit = permGate.decide(PERMISSION_CODES.opsRoomDevice.edit);
  const permDelete = permGate.decide(PERMISSION_CODES.opsRoomDevice.delete);
  const permAdd = permGate.decide(PERMISSION_CODES.opsRoomDevice.add);
  const permImport = permGate.decide(PERMISSION_CODES.opsRoomDevice.import);
  const permExport = permGate.decide(PERMISSION_CODES.opsRoomDevice.export);

  const location = useLocation();
  const [viewMode, setViewMode] = usePersistedStateController<ViewMode>({
    keyPrefix: location.pathname,
    keySuffix: "viewMode",
    defaultValue: "table",
  });

  const searchFormRef = useRef<FormInstance | null>(null);

  // 业务逻辑 hook
  const {
    roomOptions,
    loadRoomOptions,
    statistics,
    setStatistics,
    loadStatistics,
    refreshData,
    cardDevices,
    modalVisible,
    editingDevice,
    deviceForm,
    closeModal,
    openModal,
    handleSave,
    handleDelete,
    handleBatchDelete,
    importVisible,
    setImportVisible,
    submitting,
    exportVisible,
    setExportVisible,
    exportFilters,
    setExportFilters,
    reloadRef,
    request,
  } = useRoomDeviceActions();

  // 部门筛选管理
  const { selectedDeptId, handleDeptSelect } = useSidebarDeptFilter();

  // 服务端排序元数据
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<RoomDevice>("name"),
      createSorterMeta<RoomDevice>("deviceCode"),
      createSorterMeta<RoomDevice>("roomName"),
      createSorterMeta<RoomDevice>("deviceType"),
      createSorterMeta<RoomDevice>("status"),
      createSorterMeta<RoomDevice>("createdAt", "date"),
    ],
    []
  );

  // 初始化加载统计
  useEffect(() => {
    void loadStatistics().then(setStatistics);
  }, [loadStatistics, setStatistics]);

  // 部门变化：重载机房选项
  useEffect(() => {
    void loadRoomOptions(selectedDeptId || undefined);
  }, [selectedDeptId, loadRoomOptions]);

  const handleImportSuccess = () => {
    refreshData();
    setImportVisible(false);
  };

  const handleOpenExport = () => {
    const values = (searchFormRef.current?.getFieldsValue() ?? {}) as Record<string, unknown>;
    const currentFilters: Record<string, unknown> = {};
    Object.keys(values).forEach((key) => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== "") {
        currentFilters[key] = value;
      }
    });
    setExportFilters(currentFilters);
    setExportVisible(true);
  };

  const roomSelectOptions = useMemo(
    () => roomOptions.map((r) => ({ label: r.name, value: r.id })),
    [roomOptions]
  );

  const searchFields = useMemo(
    () => getRoomDeviceSearchFields({ selectedDeptId, roomSelectOptions }),
    [selectedDeptId, roomSelectOptions]
  );

  const extraFilters = useMemo(
    () => (selectedDeptId ? { orgId: selectedDeptId } : {}),
    [selectedDeptId]
  );

  const handleDeptSelectAndReset = useCallback(
    (selectedKeys: Key[], info: Parameters<typeof handleDeptSelect>[1]) => {
      searchFormRef.current?.resetFields();
      handleDeptSelect(selectedKeys, info);
    },
    [handleDeptSelect]
  );

  const columns = useMemo(() => getRoomDeviceColumns(), []);

  const rowActions = useMemo(
    () => getRoomDeviceRowActions(openModal, handleDelete),
    [openModal, handleDelete]
  );

  // 业务区：统计卡片 + 选中提示
  const renderBusinessArea = ({
    selectedRowKeys,
    clearSelection,
    total,
    loadData,
    searchForm,
  }: PageTableRenderContext<RoomDevice>) => {
    // 协议保留：reloadRef 提升到业务回调
    reloadRef.current = loadData;
    searchFormRef.current = searchForm;
    return (
      <>
        <StatisticsCards
          show={total > 10}
          items={[
            { title: "总设备数", value: statistics.total, prefix: <AppstoreAddOutlined /> },
            {
              title: "正常",
              value: statistics.normal,
              styles: { content: { color: "var(--theme-success)" } },
              prefix: <CheckCircleOutlined />,
            },
            {
              title: "故障",
              value: statistics.fault,
              styles: { content: { color: "var(--theme-error)" } },
              prefix: <WarningOutlined />,
            },
            {
              title: "报废",
              value: statistics.scrapped,
              styles: { content: { color: "var(--theme-text-tertiary)" } },
              prefix: <StopOutlined />,
            },
          ]}
        />
        {selectedRowKeys.length > 0 && (
          <Alert
            title={
              <span>
                已选择 <strong>{selectedRowKeys.length}</strong> 个设备，
                <Button type="link" size="small" onClick={clearSelection} style={{ padding: 0 }}>
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
  };

  return (
    <Layout style={{ background: "var(--theme-bg-secondary)", minHeight: "calc(100vh - 64px)" }}>
      <DeptSidebar selectedDeptId={selectedDeptId} onSelect={handleDeptSelectAndReset} />
      <Content style={{ background: "var(--theme-bg-surface)", padding: 16 }}>
        <PageTable<RoomDevice>
          request={request}
          columns={columns}
          rowKey={(record) => record.id}
          searchFields={searchFields}
          sorterMetas={sorterMetas}
          extraFilters={extraFilters}
          rowActions={rowActions}
          batchDelete={
            permDelete.visible ? { onDelete: handleBatchDelete, target: "机房设备" } : undefined
          }
          toolbar={
            <Space>
              <Radio.Group
                value={viewMode}
                onChange={(e) => setViewMode(e.target.value)}
                buttonStyle="solid"
              >
                <Radio.Button value="table">
                  <TableOutlined /> 表格
                </Radio.Button>
                <Radio.Button value="card">
                  <AppstoreOutlined /> 卡片
                </Radio.Button>
              </Radio.Group>
              {permImport.visible && (
                <Button
                  icon={<ImportOutlined />}
                  onClick={() => setImportVisible(true)}
                  disabled={permImport.disabled}
                  title={permImport.hint}
                >
                  导入
                </Button>
              )}
              {permExport.visible && (
                <Button
                  icon={<ExportOutlined />}
                  onClick={handleOpenExport}
                  disabled={permExport.disabled}
                  title={permExport.hint}
                >
                  导出
                </Button>
              )}
              {permAdd.visible && (
                <Button
                  type="primary"
                  icon={<PlusOutlined />}
                  onClick={() => openModal()}
                  disabled={permAdd.disabled}
                  title={permAdd.hint}
                >
                  新增设备
                </Button>
              )}
            </Space>
          }
          style={viewMode === "card" ? { display: "none" } : undefined}
        >
          {renderBusinessArea}
        </PageTable>

        {viewMode === "card" && (
          <Card style={{ marginTop: 16 }}>
            <RoomDeviceCardView
              devices={cardDevices}
              permEdit={permEdit}
              permDelete={permDelete}
              openModal={openModal}
              handleDelete={handleDelete}
            />
          </Card>
        )}

        <RoomDeviceModal
          visible={modalVisible}
          editingDevice={editingDevice}
          form={deviceForm}
          submitting={submitting}
          roomSelectOptions={roomSelectOptions}
          onOk={handleSave}
          onCancel={closeModal}
        />
        <ExcelImportLazy
          entityType="roomDevice"
          entityName="机房设备"
          visible={importVisible}
          onClose={() => setImportVisible(false)}
          onImportSuccess={handleImportSuccess}
        />
        <ExcelExport
          entityType="roomDevice"
          entityName="机房设备"
          visible={exportVisible}
          onClose={() => setExportVisible(false)}
          filters={exportFilters}
        />
      </Content>
    </Layout>
  );
};

export default RoomDeviceManagement;
