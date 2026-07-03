/**
 * 资产管理页面
 */

import dayjs from "dayjs";
import { useState, useCallback, useEffect, useMemo } from "react";
import type { FC } from "react";
import {
  App,
  Table,
  Button,
  Space,
  Form,
  Input,
  Select,
  Tag,
  Modal,
  Card,
  Tooltip,
} from "antd";
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  ImportOutlined,
  ExportOutlined,
  DownloadOutlined,
  SettingOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import type { Asset } from "@/types/operations";
import { useTableManager } from "@/hooks/useTableManager";
import { usePagination } from "@/hooks/usePagination";
import { assetApi } from "@/lib/opsApi";
import ExcelImport from "@/components/shared/ExcelImport";
import { getAuthHeaders } from "@/utils/authHelpers";
import { useColumnConfig } from "@/hooks/useColumnConfig";
import { ColumnConfigModal } from "@/components/shared/ColumnConfigModal";
import { AssetRow } from "@/components/table/AssetRow";
import { createSorterMeta } from "@/utils/tableHelpers";
import {
  ReconciliationDrawer,
  useReconciliationVisibility,
  type DrawerTabKey,
} from "@/components/reconciliation";
import { defaultAssetColumns, type AssetColumnConfig } from "./columnsSchema";

const { Option } = Select;

// 统计数据类型
interface Statistics {
  total: number;
  normal: number;
  stopped: number;
  nbf: number;
}

// 设备类型选项
interface DeviceTypeOption {
  value: string;
  count: number;
}

// 导出类型供其他文件复用(避免 columnsSchema.ts 直接被外引,保持单向依赖)
export type { AssetColumnConfig };

const AssetList: FC = () => {
  const { message } = App.useApp();
  const [statistics, setStatistics] = useState<Statistics>({
    total: 0,
    normal: 0,
    stopped: 0,
    nbf: 0,
  });
  const [importVisible, setImportVisible] = useState(false);
  const [exporting, setExporting] = useState(false);
  const [deviceTypeOptions, setDeviceTypeOptions] = useState<DeviceTypeOption[]>([]);
  const [deviceCategoryOptions, setDeviceCategoryOptions] = useState<DeviceTypeOption[]>([]);
  const [showColumnConfig, setShowColumnConfig] = useState(false);

  // R4 (Phase 45) — ReconciliationDrawer state (lifts to page level per UI-SPEC)
  const [drawerState, setDrawerState] = useState<{
    open: boolean;
    assetId: string | null;
    workstationId: string | null;
    assetCode?: string;
    activeTab: DrawerTabKey;
  }>({ open: false, assetId: null, workstationId: null, activeTab: "summary" });

  const _reconciliationVisible = useReconciliationVisibility();
  void _reconciliationVisible;

  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 服务端排序:field 对应后端 assetAllowedSortFields 白名单 key
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<Asset>("devicesn"),
      createSorterMeta<Asset>("deviceTypeName"),
      createSorterMeta<Asset>("lastInventoryDate", "date"),
    ],
    []
  );

  const {
    loading,
    data: assets,
    total,
    selectedRowKeys,
    searchForm,
    setSelectedRowKeys,
    getColumnSortOrder,
    handleTableChange,
    handleSearch,
    handleReset,
    handleAdd,
    loadData: loadAssets,
    resetSelection,
  } = useTableManager<Asset>(
    async (params) => {
      const result = await assetApi.list(params) as { data?: { list: Asset[]; total: number } };
      return { list: result.data?.list || [], total: result.data?.total || 0 };
    },
    {
      externalPagination: {
        current: paginationProps.current ?? 1,
        pageSize: paginationProps.pageSize ?? 10,
        setCurrent,
        setPageSize,
        setTotal,
      },
      sorterMetas,
    }
  );

  // 列配置 Hook
  const {
    config,
    visibleColumns,
    loading: configLoading,
    saving: configSaving,
    saveConfig,
    resetConfig,
  } = useColumnConfig({
    pageKey: "asset.list",
    defaultColumns: defaultAssetColumns,
    enableCache: true,
  });

  // 加载统计数据(专用 COUNT 端点,真实 status/nbf_status 计数,不再用伪造比例)
  const loadStatistics = useCallback(async () => {
    try {
      const result = await assetApi.statistics();
      setStatistics(result ?? { total: 0, normal: 0, stopped: 0, nbf: 0 });
    } catch (error) {
      console.error("加载统计数据失败:", error);
    }
  }, []);

  // 加载设备类型选项
  const loadDeviceTypes = useCallback(async () => {
    try {
      const result = await assetApi.getDeviceTypes();
      if (result.code === 0 && result.data) {
        setDeviceTypeOptions(result.data);
      }
    } catch (error) {
      console.error("加载设备类型失败:", error);
      // 降级到固定选项
      setDeviceTypeOptions([
        { value: "台式机", count: 0 },
        { value: "笔记本", count: 0 },
        { value: "服务器", count: 0 },
        { value: "打印机", count: 0 },
        { value: "交换机", count: 0 },
        { value: "路由器", count: 0 },
        { value: "其他", count: 0 },
      ]);
    }
  }, []);

  // 加载设备种类选项
  const loadDeviceCategories = useCallback(async () => {
    try {
      const result = await assetApi.getDeviceCategories();
      if (result.code === 0 && result.data) {
        setDeviceCategoryOptions(result.data);
      }
    } catch (error) {
      console.error("加载设备种类失败:", error);
      // 降级到固定选项
      setDeviceCategoryOptions([
        { value: "计算机设备", count: 0 },
        { value: "网络设备", count: 0 },
        { value: "其他设备", count: 0 },
      ]);
    }
  }, []);

  // WR-04 审计 useEffect deps: useTableManager.loadData 内部已用 useCallback([]) 稳定
  // (loadAssets 来自 useTableManager 的 destructure),同样 loadStatistics / loadDeviceTypes /
  // loadDeviceCategories 均为本页内 useCallback([]) 稳定引用。整个 deps 数组都是稳定引用,
  // 不会触发额外 re-render / re-fetch。R4 在本文件新增 useReconciliationVisibility() 与
  // ReconciliationDrawer 状态未引入新依赖 — 上述 deps 保持完整且稳定。
  useEffect(() => {
    loadStatistics();
    loadDeviceTypes();
    loadDeviceCategories();
    loadAssets(); // 页面加载时自动获取资产列表
  }, [loadStatistics, loadDeviceTypes, loadDeviceCategories, loadAssets]);

  const handleDelete = useCallback(async (id: string) => {
    Modal.confirm({
      title: "确认删除",
      content: "确定要删除该资产吗？",
      onOk: async () => {
        try {
          await assetApi.delete(id);
          message.success("删除成功");
          loadAssets();
          loadStatistics();
        } catch (error) {
          message.error("删除失败");
        }
      },
    });
  }, [loadAssets, loadStatistics]);

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning("请先选择要删除的资产");
      return;
    }

    Modal.confirm({
      title: "确认批量删除",
      content: `确定要删除选中的 ${selectedRowKeys.length} 条资产吗？`,
      onOk: async () => {
        try {
          await assetApi.batch("delete", { ids: selectedRowKeys });
          message.success("批量删除成功");
          resetSelection();
          loadAssets();
          loadStatistics();
        } catch (error) {
          message.error("批量删除失败");
        }
      },
    });
  };

  const handleExport = async () => {
    setExporting(true);
    try {
      const searchValues = searchForm.getFieldsValue();
      const params = {
        current: 1,
        pageSize: 10000,
        ...(searchValues as Record<string, unknown>),
      };

      await assetApi.excel.export(params as any);
      message.success("导出成功");
    } catch (error) {
      message.error("导出失败");
    } finally {
      setExporting(false);
    }
  };

  const columns: ColumnsType<Asset> = [
    // order=1
    {
      title: "设备序列号", dataIndex: "devicesn", key: "devicesn", width: 150,
      fixed: "left", sorter: true, sortOrder: getColumnSortOrder("devicesn"),
      render: (text) => (
        <Tooltip title={text}><span style={{ cursor: "pointer" }}>{text}</span></Tooltip>
      ),
    },
    // order=2
    { key: "sequenceNo", title: "序列号", dataIndex: "sequenceNo", width: 120, ellipsis: true },
    // order=3
    { key: "fixAssetNo", title: "固定资产编号", dataIndex: "fixAssetNo", width: 120, ellipsis: true },
    // order=4
    { key: "deviceModelName", title: "设备型号", dataIndex: "deviceModelName", width: 120, ellipsis: true },
    // order=5
    {
      key: "deviceTypeName", title: "设备类型", dataIndex: "deviceTypeName", width: 100,
      ellipsis: true, sorter: true, sortOrder: getColumnSortOrder("deviceTypeName"),
    },
    // order=6
    { key: "deviceCategorySecondName", title: "设备中类", dataIndex: "deviceCategorySecondName", width: 120, ellipsis: true },
    // order=7
    { key: "deviceBasicTypeName", title: "是否固定资产", dataIndex: "deviceBasicTypeName", width: 100, ellipsis: true },
    // order=8
    { key: "deviceStatusName", title: "设备状态", dataIndex: "deviceStatusName", width: 100, ellipsis: true },
    // order=9
    { key: "deptName", title: "所属部门", dataIndex: "deptName", width: 120, ellipsis: true },
    // order=10 已删除(recipientName 后端不存在)
    // order=11
    { key: "recipientPhone", title: "领取人电话", dataIndex: "recipientPhone", width: 120, ellipsis: true },
    // order=12
    { key: "recipientEmail", title: "领取人邮箱", dataIndex: "recipientEmail", width: 150, ellipsis: true },
    // order=13
    { key: "machineIp", title: "加域IP", dataIndex: "machineIp", width: 120, ellipsis: true },
    // order=14
    { key: "mac1", title: "有线MAC", dataIndex: "mac1", width: 140, ellipsis: true,
      render: (text) => (text ? String(text).toUpperCase() : "-") },
    // order=15
    { key: "assetHostname", title: "主机名", dataIndex: "assetHostname", width: 120, ellipsis: true },
    // order=16
    { key: "osName", title: "操作系统", dataIndex: "osName", width: 100, ellipsis: true },
    // order=17
    { key: "osVersion", title: "系统版本", dataIndex: "osVersion", width: 100, ellipsis: true },
    // order=18
    { key: "cpuModel", title: "CPU型号", dataIndex: "cpuModel", width: 120, ellipsis: true },
    // order=19
    { key: "cpuCores", title: "CPU核心数", dataIndex: "cpuCores", width: 80, ellipsis: true },
    // order=20
    { key: "memoryCapacity", title: "内存容量", dataIndex: "memoryCapacity", width: 100, ellipsis: true },
    // order=21
    { key: "diskCapacity", title: "硬盘容量", dataIndex: "diskCapacity", width: 100, ellipsis: true },
    // order=22
    { key: "graphicsCard", title: "显卡型号", dataIndex: "graphicsCard", width: 120, ellipsis: true },
    // order=23
    { key: "purchaseDate", title: "购置日期", dataIndex: "purchaseDate", width: 120,
      render: (date) => (date ? dayjs(date).format("YYYY-MM-DD") : "-"), ellipsis: true },
    // order=24
    { key: "warrantyExpireDate", title: "保修到期日", dataIndex: "warrantyExpireDate", width: 120,
      render: (date) => (date ? dayjs(date).format("YYYY-MM-DD") : "-"), ellipsis: true },
    // order=25
    { key: "supplierName", title: "供应商", dataIndex: "supplierName", width: 120, ellipsis: true },
    // order=26
    { key: "manufacturerName", title: "制造商", dataIndex: "manufacturerName", width: 120, ellipsis: true },
    // order=27
    { key: "assetLocation", title: "资产位置", dataIndex: "assetLocation", width: 120, ellipsis: true },
    // order=28
    { key: "buildingName", title: "所在大楼", dataIndex: "buildingName", width: 100, ellipsis: true },
    // order=29
    { key: "floorName", title: "所在楼层", dataIndex: "floorName", width: 80, ellipsis: true },
    // order=30
    { key: "workstationName", title: "所在工位", dataIndex: "workstationName", width: 120, ellipsis: true },
    // order=31
    { key: "serverRoomName", title: "所在机房", dataIndex: "serverRoomName", width: 120, ellipsis: true },
    // order=32
    { key: "networkSwitchName", title: "网络交换机", dataIndex: "networkSwitchName", width: 120, ellipsis: true },
    // order=33
    { key: "switchPort", title: "交换机端口", dataIndex: "switchPort", width: 120, ellipsis: true },
    // order=34
    { key: "voltage", title: "电压", dataIndex: "voltage", width: 80, ellipsis: true },
    // order=35
    { key: "useStatusName", title: "使用状态", dataIndex: "useStatusName", width: 100, ellipsis: true },
    // order=36
    { key: "financeCode", title: "财务编码", dataIndex: "financeCode", width: 100, ellipsis: true },
    // order=37
    { key: "projectCode", title: "项目编码", dataIndex: "projectCode", width: 100, ellipsis: true },
    // order=38
    { key: "assetCost", title: "资产成本", dataIndex: "assetCost", width: 100, ellipsis: true },
    // order=39
    { key: "depreciationPeriod", title: "折旧年限", dataIndex: "depreciationPeriod", width: 100, ellipsis: true },
    // order=40
    { key: "netBookValue", title: "账面净值", dataIndex: "netBookValue", width: 100, ellipsis: true },
    // order=41
    { key: "depreciationRate", title: "折旧率", dataIndex: "depreciationRate", width: 100, ellipsis: true },
    // order=42
    { key: "accumulatorDepreciation", title: "累计折旧", dataIndex: "accumulatorDepreciation", width: 120, ellipsis: true },
    // order=43
    { key: "remark", title: "备注", dataIndex: "remark", width: 200, ellipsis: true },
    // order=44
    { key: "signOrgnoName", title: "归属机构", dataIndex: "signOrgnoName", width: 150, ellipsis: true },
    // order=45
    { key: "nowUserName", title: "责任人", dataIndex: "nowUserName", width: 100, ellipsis: true },
    // order=46
    { key: "nowUserDeptCode", title: "部门编码", dataIndex: "nowUserDeptCode", width: 120, ellipsis: true },
    // order=47 deviceUserName 取代 recipientName(真领取人字段)
    { key: "deviceUserName", title: "领取人", dataIndex: "deviceUserName", width: 100, ellipsis: true },
    // order=48
    {
      key: "status", title: "状态", dataIndex: "status", width: 80,
      render: (s: number) => (
        <Tag color={s === 0 ? "green" : "red"}>{s === 0 ? "正常" : "停用"}</Tag>
      ),
    },
    // order=49
    {
      key: "nbfStatus", title: "拟报废", dataIndex: "nbfStatus", width: 80,
      render: (s: number) => (
        <Tag color={s === 1 ? "orange" : "default"}>{s === 1 ? "是" : "否"}</Tag>
      ),
    },
    // order=50
    { key: "drawingDate", title: "接收日期", dataIndex: "drawingDate", width: 120,
      render: (date) => (date ? dayjs(date).format("YYYY-MM-DD") : "-"), ellipsis: true },
    // order=51
    { key: "machineUptime", title: "最后上线", dataIndex: "machineUptime", width: 150,
      render: (date) => (date ? dayjs(date).format("YYYY-MM-DD HH:mm:ss") : "-"), ellipsis: true },
    // order=52
    {
      key: "lastInventoryDate", title: "盘点日期", dataIndex: "lastInventoryDate", width: 120,
      ellipsis: true, sorter: true, sortOrder: getColumnSortOrder("lastInventoryDate"),
      render: (date) => (date ? dayjs(date).format("YYYY-MM-DD") : "-"),
    },
    // 末尾特殊列:action + reconciliation(不参与 defaultAssetColumns 排序)
    {
      title: "操作", key: "action", width: 120, fixed: "right",
      render: (_, record) => (
        <AssetRow record={record} onEdit={handleEdit} onDelete={handleDelete} />
      ),
    },
    // R4 (Phase 45) — 对账健康列(行内徽标,assets 列表)
    // CR-01 修复:ops_asset 工作站绑定未在 /ops/asset/list API 响应中暴露(workstation_id
    // 不在 SELECT 列表,无法 lift useWorkstationHealth 到页面层)。在此后端限制解除前,
    // 渲染 "-" 占位以避免显示错误的 healthy 绿色徽标(原 conflictType={null} 会让
    // HealthBadge 默认渲染绿色圆点,误导用户以为所有资产均健康)。后续若需要真实
    // 对账状态需扩展 Asset DTO 增加 workstationId 字段,然后才能复用 workstations 页面的
    // lift-up 模式(assetConflictMap)。
    {
      title: "对账健康", key: "reconciliation", width: 96,
      render: (_: unknown, _record: Asset) => <>-</>,
    },
  ];

  // 根据列配置过滤和排序列
  const tableColumns = useMemo(() => {
    const allColumnsMap = new Map<string, any>();
    columns.forEach(col => allColumnsMap.set(col.key as string, col));

    const visibleCols = visibleColumns
      .map(colConfig => {
        const col = allColumnsMap.get(colConfig.key);
        return {
          ...col,
          width: colConfig.width || col?.width,
        };
      })
      .filter(col => col !== undefined);

    return visibleCols;
  }, [columns, visibleColumns]);

  const handleEdit = useCallback((record: Asset) => {
    // TODO: 实现编辑功能
    message.info("编辑功能待实现");
  }, []);

  return (
    <div className="p-6">
      {/* 统计卡片 */}
      <div className="grid grid-cols-4 gap-4 mb-6">
        <Card>
          <div className="text-2xl font-bold">{statistics.total}</div>
          <div className="text-gray-500">总资产数</div>
        </Card>
        <Card>
          <div className="text-2xl font-bold text-green-600">{statistics.normal}</div>
          <div className="text-gray-500">正常状态</div>
        </Card>
        <Card>
          <div className="text-2xl font-bold text-red-600">{statistics.stopped}</div>
          <div className="text-gray-500">停用状态</div>
        </Card>
        <Card>
          <div className="text-2xl font-bold text-orange-600">{statistics.nbf}</div>
          <div className="text-gray-500">拟报废</div>
        </Card>
      </div>

      {/* 搜索表单 */}
      <Card className="mb-4">
        <Form form={searchForm} layout="inline">
          <Form.Item name="devicesn" label="设备序列号">
            <Input placeholder="请输入设备序列号" allowClear />
          </Form.Item>
          <Form.Item name="deviceModelName" label="型号">
            <Input placeholder="请输入型号" allowClear />
          </Form.Item>
          <Form.Item name="deviceTypeName" label="设备类型">
            <Select
              placeholder="请选择设备类型"
              allowClear
              style={{ width: 200 }}
              loading={deviceTypeOptions.length === 0}
             onSearch={() => {}}>
              {deviceTypeOptions.map((option) => (
                <Option key={option.value} value={option.value}>
                  {option.value} ({option.count})
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="deviceCategorySecondName" label="设备种类">
            <Select
              placeholder="请选择设备种类"
              allowClear
              style={{ width: 200 }}
              loading={deviceCategoryOptions.length === 0}
             onSearch={() => {}}>
              {deviceCategoryOptions.map((option) => (
                <Option key={option.value} value={option.value}>
                  {option.value} ({option.count})
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select placeholder="请选择状态" allowClear style={{ width: 120 }} onSearch={() => {}}>
              <Option value={0}>正常</Option>
              <Option value={1}>停用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="nbfStatus" label="拟报废">
            <Select placeholder="请选择" allowClear style={{ width: 120 }} onSearch={() => {}}>
              <Option value={0}>否</Option>
              <Option value={1}>是</Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                搜索
              </Button>
              <Button icon={<ReloadOutlined />} onClick={handleReset}>
                重置
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      {/* 工具栏 */}
      <div className="mt-4 mb-4 flex items-center justify-between">
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
            新增资产
          </Button>
          <Button icon={<DownloadOutlined />} onClick={() => setImportVisible(true)}>
            下载模板
          </Button>
          <Button icon={<ImportOutlined />} onClick={() => setImportVisible(true)}>
            导入
          </Button>
          <Button icon={<ExportOutlined />} loading={exporting} onClick={handleExport}>
            导出
          </Button>
          <Button icon={<SettingOutlined />} onClick={() => setShowColumnConfig(true)}>
            列设置
          </Button>
        </Space>
        <Button
          danger
          icon={<DeleteOutlined />}
          disabled={selectedRowKeys.length === 0}
          onClick={handleBatchDelete}
        >
          批量删除 ({selectedRowKeys.length})
        </Button>
      </div>

      {/* 数据表格 */}
      <Table
        rowKey="id"
        columns={tableColumns}
        dataSource={assets}
        loading={loading || configLoading}
        virtual
        pagination={{
          current: paginationProps.current,
          pageSize: paginationProps.pageSize,
          total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 条`,
        }}
        onChange={handleTableChange}
        scroll={{ x: 4200, y: 600 }}
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
          columnWidth: 50,
        }}
      />

      {/* 导入弹窗 */}
      <ExcelImport
        entityType="asset"
        entityName="资产"
        visible={importVisible}
        onClose={() => setImportVisible(false)}
        onImportSuccess={() => {
          setImportVisible(false);
          loadAssets();
          loadStatistics();
        }}
      />

      {/* 列配置模态框 */}
      <ColumnConfigModal
        visible={showColumnConfig}
        config={config}
        defaultConfig={defaultAssetColumns}
        onSave={saveConfig}
        onReset={resetConfig}
        onClose={() => setShowColumnConfig(false)}
        saving={configSaving}
      />

      {/* R4 (Phase 45) — ReconciliationDrawer at page level
          Plan 02 / SC9: onApplyException 省略让抽屉自带的内联 navigate 接管,自动携带
          workstationId + assetId + assetIp + conflictType 四个 query 参数。
      */}
      <ReconciliationDrawer
        open={drawerState.open}
        onClose={() =>
          setDrawerState((s) => ({ ...s, open: false, activeTab: "summary" }))
        }
        selectedAssetId={drawerState.assetId}
        workstationId={drawerState.workstationId}
        assetCode={drawerState.assetCode}
        activeTab={drawerState.activeTab}
        onTabChange={(k) => setDrawerState((s) => ({ ...s, activeTab: k }))}
      />
    </div>
  );
};

export default AssetList;
