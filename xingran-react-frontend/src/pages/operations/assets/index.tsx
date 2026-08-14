/**
 * 资产管理页面
 */

import dayjs from "dayjs";
import { useState, useCallback, useEffect, useMemo } from "react";
import type { FC } from "react";
import { App, Table, Button, Space, Form, Input, Select, Tag, Modal, Card, Tooltip } from "antd";
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
import ComponentListTab from "./components/ComponentListTab";

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
      const result = (await assetApi.list(params)) as { data?: { list: Asset[]; total: number } };
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

  const handleDelete = useCallback(
    async (id: string) => {
      Modal.confirm({
        title: "确认删除",
        content: "确定要删除该资产吗？",
        onOk: async () => {
          try {
            await assetApi.delete(id);
            message.success("删除成功");
            loadAssets();
            loadStatistics();
          } catch (_error) {
            message.error("删除失败");
          }
        },
      });
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
    [loadAssets, loadStatistics]
  );

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
        } catch (_error) {
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
    } catch (_error) {
      message.error("导出失败");
    } finally {
      setExporting(false);
    }
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps -- columns array recreated each render; dependency array at tableColumns useMemo intentionally references it
  const columns: ColumnsType<Asset> = [
    // order=1
    {
      title: "设备序列号",
      dataIndex: "devicesn",
      key: "devicesn",
      width: 150,
      fixed: "left",
      sorter: true,
      sortOrder: getColumnSortOrder("devicesn"),
      render: (text) => (
        <Tooltip title={text}>
          <span style={{ cursor: "pointer" }}>{text}</span>
        </Tooltip>
      ),
    },
    // order=2
    { key: "sequenceNo", title: "序列号", dataIndex: "sequenceNo", width: 120, ellipsis: true },
    // order=3
    {
      key: "fixAssetNo",
      title: "固定资产编号",
      dataIndex: "fixAssetNo",
      width: 120,
      ellipsis: true,
    },
    // order=4
    {
      key: "deviceModelName",
      title: "设备型号",
      dataIndex: "deviceModelName",
      width: 120,
      ellipsis: true,
    },
    // order=5
    {
      key: "deviceTypeName",
      title: "设备类型",
      dataIndex: "deviceTypeName",
      width: 100,
      ellipsis: true,
      sorter: true,
      sortOrder: getColumnSortOrder("deviceTypeName"),
    },
    // order=6
    {
      key: "deviceCategorySecondName",
      title: "设备中类",
      dataIndex: "deviceCategorySecondName",
      width: 120,
      ellipsis: true,
    },
    // order=7
    {
      key: "deviceBasicTypeName",
      title: "是否固定资产",
      dataIndex: "deviceBasicTypeName",
      width: 100,
      ellipsis: true,
    },
    // order=9
    {
      key: "usefulDeptName",
      title: "所属部门",
      dataIndex: "usefulDeptName",
      width: 120,
      ellipsis: true,
    },
    // order=13
    { key: "machineIp", title: "加域IP", dataIndex: "machineIp", width: 120, ellipsis: true },
    // order=14
    {
      key: "mac1",
      title: "有线MAC",
      dataIndex: "mac1",
      width: 140,
      ellipsis: true,
      render: (text) => (text ? String(text).toUpperCase() : "-"),
    },
    // order=35
    {
      key: "useStatusLabel",
      title: "使用状态",
      dataIndex: "useStatusLabel",
      width: 100,
      ellipsis: true,
    },
    // order=43
    { key: "remark", title: "备注", dataIndex: "remark", width: 200, ellipsis: true },
    // order=44
    {
      key: "signOrgnoName",
      title: "归属机构",
      dataIndex: "signOrgnoName",
      width: 150,
      ellipsis: true,
    },
    // order=45
    { key: "nowUserName", title: "责任人", dataIndex: "nowUserName", width: 100, ellipsis: true },
    // order=46
    {
      key: "nowUserDeptCode",
      title: "部门编码",
      dataIndex: "nowUserDeptCode",
      width: 120,
      ellipsis: true,
    },
    // order=47
    {
      key: "deviceUserName",
      title: "领取人",
      dataIndex: "deviceUserName",
      width: 100,
      ellipsis: true,
    },
    // order=48
    {
      key: "status",
      title: "状态",
      dataIndex: "status",
      width: 80,
      render: (s: number) => (
        <Tag color={s === 0 ? "green" : "red"}>{s === 0 ? "正常" : "停用"}</Tag>
      ),
    },
    // order=49
    {
      key: "nbfStatus",
      title: "拟报废",
      dataIndex: "nbfStatus",
      width: 80,
      render: (s: number) => (
        <Tag color={s === 1 ? "orange" : "default"}>{s === 1 ? "是" : "否"}</Tag>
      ),
    },
    // order=50
    {
      key: "drawingDate",
      title: "接收日期",
      dataIndex: "drawingDate",
      width: 120,
      render: (date) => (date ? dayjs(date).format("YYYY-MM-DD") : "-"),
      ellipsis: true,
    },
    // order=51
    {
      key: "machineUptime",
      title: "最后上线",
      dataIndex: "machineUptime",
      width: 150,
      render: (date) => (date ? dayjs(date).format("YYYY-MM-DD HH:mm:ss") : "-"),
      ellipsis: true,
    },
    // order=52
    {
      key: "lastInventoryDate",
      title: "盘点日期",
      dataIndex: "lastInventoryDate",
      width: 120,
      ellipsis: true,
      sorter: true,
      sortOrder: getColumnSortOrder("lastInventoryDate"),
      render: (date) => (date ? dayjs(date).format("YYYY-MM-DD") : "-"),
    },
    // 设备信息扩展 (order=54 设备渠道, order=55 设备属性)
    { key: "qudaoName", title: "设备渠道", dataIndex: "qudaoName", width: 100, ellipsis: true },
    {
      key: "attributeValue",
      title: "设备属性",
      dataIndex: "attributeValue",
      width: 120,
      ellipsis: true,
    },
    // 部门与用户扩展 (order=53 受益部门)
    { key: "deptName", title: "受益部门", dataIndex: "deptName", width: 120, ellipsis: true },
    // 网络信息扩展 (order=63 加域标识, order=64 无线MAC)
    { key: "machineBs", title: "加域标识", dataIndex: "machineBs", width: 100, ellipsis: true },
    {
      key: "mac2",
      title: "无线MAC",
      dataIndex: "mac2",
      width: 140,
      ellipsis: true,
      render: (text) => (text ? String(text).toUpperCase() : "-"),
    },
    // 归属与责任扩展 (order=58 使用机构, 59 使用人, 60 责任人岗位, 61 用途, 62 子用途, 66 APP扫码账号, 68 APP扫码地理位置)
    { key: "orgnoName", title: "使用机构", dataIndex: "orgnoName", width: 150, ellipsis: true },
    { key: "outerUser", title: "使用人", dataIndex: "outerUser", width: 100, ellipsis: true },
    {
      key: "nowUserJobName",
      title: "责任人岗位",
      dataIndex: "nowUserJobName",
      width: 120,
      ellipsis: true,
    },
    { key: "usingTypeName", title: "用途", dataIndex: "usingTypeName", width: 100, ellipsis: true },
    {
      key: "subUsingTypeName",
      title: "子用途",
      dataIndex: "subUsingTypeName",
      width: 100,
      ellipsis: true,
    },
    { key: "userName", title: "APP扫码账号", dataIndex: "userName", width: 100, ellipsis: true },
    {
      key: "scanSite",
      title: "APP扫码地理位置",
      dataIndex: "scanSite",
      width: 200,
      ellipsis: true,
    },
    // 状态与日期扩展 (order=56 发放日期, 57 入库日期, 65 最后上线账号, 67 APP扫码时间, 69 Y07更新时间)
    {
      key: "useDate",
      title: "发放日期",
      dataIndex: "useDate",
      width: 120,
      render: (date) => (date ? dayjs(date).format("YYYY-MM-DD") : "-"),
      ellipsis: true,
    },
    {
      key: "storageDatetime",
      title: "入库日期",
      dataIndex: "storageDatetime",
      width: 120,
      render: (date) => (date ? dayjs(date).format("YYYY-MM-DD") : "-"),
      ellipsis: true,
    },
    {
      key: "machineUserId",
      title: "最后上线账号",
      dataIndex: "machineUserId",
      width: 120,
      ellipsis: true,
    },
    {
      key: "lastUpdateDate",
      title: "APP扫码时间",
      dataIndex: "lastUpdateDate",
      width: 150,
      render: (date) => (date ? dayjs(date).format("YYYY-MM-DD HH:mm:ss") : "-"),
      ellipsis: true,
    },
    {
      key: "y07UpdateTime",
      title: "Y07更新时间",
      dataIndex: "y07UpdateTime",
      width: 150,
      render: (date) => (date ? dayjs(date).format("YYYY-MM-DD HH:mm:ss") : "-"),
      ellipsis: true,
    },
    // 标识与盘点 (order=70 新设备标识, 71 异常标识, 72 盘点结果)
    {
      key: "newFlagLabel",
      title: "新设备标识",
      dataIndex: "newFlagLabel",
      width: 100,
      ellipsis: true,
    },
    {
      key: "errorFlagName",
      title: "异常标识",
      dataIndex: "errorFlagName",
      width: 100,
      ellipsis: true,
    },
    {
      key: "inventoryResult",
      title: "盘点结果",
      dataIndex: "inventoryResult",
      width: 100,
      ellipsis: true,
    },
    // 末尾特殊列:action + reconciliation(不参与 defaultAssetColumns 排序)
    {
      title: "操作",
      key: "action",
      width: 120,
      fixed: "right",
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
      title: "对账健康",
      key: "reconciliation",
      width: 96,
      render: (_: unknown, _record: Asset) => <>-</>,
    },
  ];

  // 根据列配置过滤和排序列
  const tableColumns = useMemo(() => {
    const allColumnsMap = new Map<string, any>();
    columns.forEach((col) => allColumnsMap.set(col.key as string, col));

    const visibleCols = visibleColumns
      .map((colConfig) => {
        const col = allColumnsMap.get(colConfig.key);
        return {
          ...col,
          width: colConfig.width || col?.width,
        };
      })
      .filter((col) => col !== undefined);

    return visibleCols;
  }, [columns, visibleColumns]);

  const handleEdit = useCallback((_record: Asset) => {
    // TODO: 实现编辑功能
    message.info("编辑功能待实现");
    // eslint-disable-next-line react-hooks/exhaustive-deps -- message from App.useApp() is stable
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
              onSearch={() => {}}
            >
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
              onSearch={() => {}}
            >
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
        expandable={{
          // Phase 48 Wave 3 (D-07): 行展开渲染「从属组件清单」Tab。
          // 仅主设备行可展开(componentType 为空即主设备);组件行本身不会
          // 出现在此列表(后端默认 component_type IS NULL 过滤已生效)。
          expandedRowRender: (record) => <ComponentListTab parentAssetId={record.id} />,
          rowExpandable: (record) => !record.componentType,
        }}
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
        onClose={() => setDrawerState((s) => ({ ...s, open: false, activeTab: "summary" }))}
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
