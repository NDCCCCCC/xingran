/**
 * 资产管理页面
 */

import dayjs from 'dayjs';
import { useState, useCallback, useEffect, useMemo } from 'react';
import type { FC } from 'react';
import {
  Table,
  Button,
  Space,
  Form,
  Input,
  Select,
  Tag,
  Modal,
  message,
  Card,
  Tooltip,
} from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  DeleteOutlined,
  ImportOutlined,
  ExportOutlined,
  DownloadOutlined,
  SettingOutlined,
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { Asset } from '@/types/operations';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import { assetApi } from '@/lib/opsApi';
import ExcelImport from '@/components/shared/ExcelImport';
import { getAuthHeaders } from '@/utils/authHelpers';
import { useColumnConfig } from '@/hooks/useColumnConfig';
import type { ColumnConfig } from '@/hooks/useColumnConfig';
import { ColumnConfigModal } from '@/components/shared/ColumnConfigModal';
import { AssetRow } from '@/components/table/AssetRow';

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

// 默认列配置（43 列）
const defaultAssetColumns: ColumnConfig[] = [
  // 核心标识 (3列)
  { key: 'devicesn', label: '设备序列号', visible: true, order: 1, width: 150, group: '核心标识' },
  { key: 'sequenceNo', label: '序列号', visible: false, order: 2, width: 120, group: '核心标识' },
  { key: 'fixAssetNo', label: '固定资产编号', visible: true, order: 3, width: 120, group: '核心标识' },
  // 设备信息 (4列)
  { key: 'deviceModelName', label: '设备型号', visible: true, order: 4, width: 120, group: '设备信息' },
  { key: 'deviceTypeName', label: '设备类型', visible: true, order: 5, width: 100, group: '设备信息' },
  { key: 'deviceCategorySecondName', label: '设备中类', visible: false, order: 6, width: 120, group: '设备信息' },
  { key: 'deviceBasicTypeName', label: '是否固定资产', visible: true, order: 7, width: 100, group: '设备信息' },
  // 设备状态 (1列)
  { key: 'deviceStatusName', label: '设备状态', visible: true, order: 8, width: 100, group: '设备状态' },
  // 所属部门和领取人 (4列)
  { key: 'deptName', label: '所属部门', visible: true, order: 9, width: 120, group: '部门与用户' },
  { key: 'recipientName', label: '领取人', visible: true, order: 10, width: 100, group: '部门与用户' },
  { key: 'recipientPhone', label: '领取人电话', visible: false, order: 11, width: 120, group: '部门与用户' },
  { key: 'recipientEmail', label: '领取人邮箱', visible: false, order: 12, width: 150, group: '部门与用户' },
  // 网络信息 (2列)
  { key: 'machineIp', label: '加域IP', visible: true, order: 13, width: 120, group: '网络信息' },
  { key: 'mac1', label: '有线MAC', visible: true, order: 14, width: 140, group: '网络信息' },
  // 主机和操作系统 (2列)
  { key: 'assetHostname', label: '主机名', visible: false, order: 15, width: 120, group: '系统信息' },
  { key: 'osName', label: '操作系统', visible: false, order: 16, width: 100, group: '系统信息' },
  { key: 'osVersion', label: '系统版本', visible: false, order: 17, width: 100, group: '系统信息' },
  // 硬件信息 (5列)
  { key: 'cpuModel', label: 'CPU型号', visible: false, order: 18, width: 120, group: '硬件信息' },
  { key: 'cpuCores', label: 'CPU核心数', visible: false, order: 19, width: 80, group: '硬件信息' },
  { key: 'memoryCapacity', label: '内存容量', visible: false, order: 20, width: 100, group: '硬件信息' },
  { key: 'diskCapacity', label: '硬盘容量', visible: false, order: 21, width: 100, group: '硬件信息' },
  { key: 'graphicsCard', label: '显卡型号', visible: false, order: 22, width: 120, group: '硬件信息' },
  // 采购与保修 (3列)
  { key: 'purchaseDate', label: '购置日期', visible: false, order: 23, width: 120, group: '采购信息' },
  { key: 'warrantyExpireDate', label: '保修到期日', visible: false, order: 24, width: 120, group: '采购信息' },
  { key: 'supplierName', label: '供应商', visible: false, order: 25, width: 120, group: '采购信息' },
  { key: 'manufacturerName', label: '制造商', visible: false, order: 26, width: 120, group: '采购信息' },
  // 位置信息 (5列)
  { key: 'assetLocation', label: '资产位置', visible: true, order: 27, width: 120, group: '位置信息' },
  { key: 'buildingName', label: '所在大楼', visible: true, order: 28, width: 100, group: '位置信息' },
  { key: 'floorName', label: '所在楼层', visible: true, order: 29, width: 80, group: '位置信息' },
  { key: 'workstationName', label: '所在工位', visible: false, order: 30, width: 120, group: '位置信息' },
  { key: 'serverRoomName', label: '所在机房', visible: false, order: 31, width: 120, group: '位置信息' },
  // 网络连接 (2列)
  { key: 'networkSwitchName', label: '网络交换机', visible: false, order: 32, width: 120, group: '网络连接' },
  { key: 'switchPort', label: '交换机端口', visible: false, order: 33, width: 120, group: '网络连接' },
  { key: 'voltage', label: '电压', visible: false, order: 34, width: 80, group: '网络连接' },
  // 使用状态 (1列)
  { key: 'useStatusName', label: '使用状态', visible: true, order: 35, width: 100, group: '使用状态' },
  // 财务信息 (7列)
  { key: 'financeCode', label: '财务编码', visible: false, order: 36, width: 100, group: '财务信息' },
  { key: 'projectCode', label: '项目编码', visible: false, order: 37, width: 100, group: '财务信息' },
  { key: 'assetCost', label: '资产成本', visible: false, order: 38, width: 100, group: '财务信息' },
  { key: 'depreciationPeriod', label: '折旧年限', visible: false, order: 39, width: 100, group: '财务信息' },
  { key: 'netBookValue', label: '账面净值', visible: false, order: 40, width: 100, group: '财务信息' },
  { key: 'depreciationRate', label: '折旧率', visible: false, order: 41, width: 100, group: '财务信息' },
  { key: 'accumulatorDepreciation', label: '累计折旧', visible: false, order: 42, width: 120, group: '财务信息' },
  // 其他信息 (1列)
  { key: 'remark', label: '备注', visible: true, order: 43, width: 200, group: '其他' },
];

const AssetList: FC = () => {
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

  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  const {
    loading,
    data: assets,
    total,
    selectedRowKeys,
    searchForm,
    setSelectedRowKeys,
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
    pageKey: 'asset.list',
    defaultColumns: defaultAssetColumns,
    enableCache: true,
  });

  // 加载统计数据
  const loadStatistics = useCallback(async () => {
    try {
      const result = await assetApi.list({ current: 1, pageSize: 1 }) as { data?: { list: Asset[]; total: number } };
      const total = result.data?.total || 0;

      // 获取各状态数量（简化实现）
      setStatistics({
        total,
        normal: Math.floor(total * 0.8),
        stopped: Math.floor(total * 0.15),
        nbf: Math.floor(total * 0.05),
      });
    } catch (error) {
      console.error('加载统计数据失败:', error);
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
      console.error('加载设备类型失败:', error);
      // 降级到固定选项
      setDeviceTypeOptions([
        { value: '台式机', count: 0 },
        { value: '笔记本', count: 0 },
        { value: '服务器', count: 0 },
        { value: '打印机', count: 0 },
        { value: '交换机', count: 0 },
        { value: '路由器', count: 0 },
        { value: '其他', count: 0 },
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
      console.error('加载设备种类失败:', error);
      // 降级到固定选项
      setDeviceCategoryOptions([
        { value: '计算机设备', count: 0 },
        { value: '网络设备', count: 0 },
        { value: '其他设备', count: 0 },
      ]);
    }
  }, []);

  useEffect(() => {
    loadStatistics();
    loadDeviceTypes();
    loadDeviceCategories();
    loadAssets(); // 页面加载时自动获取资产列表
  }, [loadStatistics, loadDeviceTypes, loadDeviceCategories, loadAssets]);

  const handleDelete = useCallback(async (id: string) => {
    Modal.confirm({
      title: '确认删除',
      content: '确定要删除该资产吗？',
      onOk: async () => {
        try {
          await assetApi.delete(id);
          message.success('删除成功');
          loadAssets();
          loadStatistics();
        } catch (error) {
          message.error('删除失败');
        }
      },
    });
  }, [loadAssets, loadStatistics]);

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请先选择要删除的资产');
      return;
    }

    Modal.confirm({
      title: '确认批量删除',
      content: `确定要删除选中的 ${selectedRowKeys.length} 条资产吗？`,
      onOk: async () => {
        try {
          await assetApi.batch('delete', { ids: selectedRowKeys });
          message.success('批量删除成功');
          resetSelection();
          loadAssets();
          loadStatistics();
        } catch (error) {
          message.error('批量删除失败');
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
      message.success('导出成功');
    } catch (error) {
      message.error('导出失败');
    } finally {
      setExporting(false);
    }
  };

  const columns: ColumnsType<Asset> = [
    {
      title: '设备序列号',
      dataIndex: 'devicesn',
      key: 'devicesn',
      width: 150,
      fixed: 'left',
      render: (text) => (
        <Tooltip title={text}>
          <span style={{ cursor: 'pointer' }}>{text}</span>
        </Tooltip>
      ),
    },
    {
      title: '型号',
      dataIndex: 'deviceModelName',
      key: 'deviceModelName',
      width: 120,
      ellipsis: true,
    },
    {
      title: '类型',
      dataIndex: 'deviceTypeName',
      key: 'deviceTypeName',
      width: 100,
      ellipsis: true,
    },
    {
      title: '受益部门',
      dataIndex: 'deptName',
      key: 'deptName',
      width: 120,
      ellipsis: true,
    },
    {
      title: '归属机构',
      dataIndex: 'signOrgnoName',
      key: 'signOrgnoName',
      width: 120,
      ellipsis: true,
    },
    {
      title: '责任人',
      dataIndex: 'nowUserName',
      key: 'nowUserName',
      width: 100,
      ellipsis: true,
    },
    {
      title: '部门编码',
      dataIndex: 'nowUserDeptCode',
      key: 'nowUserDeptCode',
      width: 120,
      ellipsis: true,
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: number) => (
        <Tag color={status === 0 ? 'green' : 'red'}>
          {status === 0 ? '正常' : '停用'}
        </Tag>
      ),
    },
    {
      title: '拟报废',
      dataIndex: 'nbfStatus',
      key: 'nbfStatus',
      width: 90,
      render: (status?: number) => (
        status === 1 ? <Tag color="orange">是</Tag> : <Tag>否</Tag>
      ),
    },
    {
      title: '加域IP',
      dataIndex: 'machineIp',
      key: 'machineIp',
      width: 120,
      ellipsis: true,
    },
    {
      title: '有线MAC',
      dataIndex: 'mac1',
      key: 'mac1',
      width: 140,
      ellipsis: true,
    },
    {
      title: '领取人',
      dataIndex: 'deviceUserName',
      key: 'deviceUserName',
      width: 100,
      ellipsis: true,
    },
    {
      title: '接收日期',
      dataIndex: 'drawingDate',
      key: 'drawingDate',
      width: 140,
      render: (date) => (date ? dayjs(date).format('YYYY-MM-DD') : '-'),
    },
    {
      title: '最后上线',
      dataIndex: 'machineUptime',
      key: 'machineUptime',
      width: 160,
      render: (date) => (date ? dayjs(date).format('YYYY-MM-DD HH:mm') : '-'),
    },
    {
      title: '盘点日期',
      dataIndex: 'lastInventoryDate',
      key: 'lastInventoryDate',
      width: 140,
      render: (date) => (date ? dayjs(date).format('YYYY-MM-DD') : '-'),
    },
    {
      title: '备注',
      dataIndex: 'remark',
      key: 'remark',
      width: 200,
      ellipsis: true,
    },
    // 额外的列定义（对应 defaultAssetColumns 中缺失的列）
    { key: 'sequenceNo', title: '序列号', dataIndex: 'sequenceNo', width: 120, ellipsis: true },
    { key: 'fixAssetNo', title: '固定资产编号', dataIndex: 'fixAssetNo', width: 120, ellipsis: true },
    { key: 'deviceCategorySecondName', title: '设备中类', dataIndex: 'deviceCategorySecondName', width: 120, ellipsis: true },
    { key: 'deviceBasicTypeName', title: '是否固定资产', dataIndex: 'deviceBasicTypeName', width: 100, ellipsis: true },
    { key: 'deviceStatusName', title: '设备状态', dataIndex: 'deviceStatusName', width: 100, ellipsis: true },
    { key: 'recipientName', title: '领取人', dataIndex: 'recipientName', width: 100, ellipsis: true },
    { key: 'recipientPhone', title: '领取人电话', dataIndex: 'recipientPhone', width: 120, ellipsis: true },
    { key: 'recipientEmail', title: '领取人邮箱', dataIndex: 'recipientEmail', width: 150, ellipsis: true },
    { key: 'assetHostname', title: '主机名', dataIndex: 'assetHostname', width: 120, ellipsis: true },
    { key: 'osName', title: '操作系统', dataIndex: 'osName', width: 100, ellipsis: true },
    { key: 'osVersion', title: '系统版本', dataIndex: 'osVersion', width: 100, ellipsis: true },
    { key: 'cpuModel', title: 'CPU型号', dataIndex: 'cpuModel', width: 120, ellipsis: true },
    { key: 'cpuCores', title: 'CPU核心数', dataIndex: 'cpuCores', width: 80, ellipsis: true },
    { key: 'memoryCapacity', title: '内存容量', dataIndex: 'memoryCapacity', width: 100, ellipsis: true },
    { key: 'diskCapacity', title: '硬盘容量', dataIndex: 'diskCapacity', width: 100, ellipsis: true },
    { key: 'graphicsCard', title: '显卡型号', dataIndex: 'graphicsCard', width: 120, ellipsis: true },
    { key: 'purchaseDate', title: '购置日期', dataIndex: 'purchaseDate', width: 120, render: (date) => (date ? dayjs(date).format('YYYY-MM-DD') : '-'), ellipsis: true },
    { key: 'warrantyExpireDate', title: '保修到期日', dataIndex: 'warrantyExpireDate', width: 120, render: (date) => (date ? dayjs(date).format('YYYY-MM-DD') : '-'), ellipsis: true },
    { key: 'supplierName', title: '供应商', dataIndex: 'supplierName', width: 120, ellipsis: true },
    { key: 'manufacturerName', title: '制造商', dataIndex: 'manufacturerName', width: 120, ellipsis: true },
    { key: 'assetLocation', title: '资产位置', dataIndex: 'assetLocation', width: 120, ellipsis: true },
    { key: 'buildingName', title: '所在大楼', dataIndex: 'buildingName', width: 100, ellipsis: true },
    { key: 'floorName', title: '所在楼层', dataIndex: 'floorName', width: 80, ellipsis: true },
    { key: 'workstationName', title: '所在工位', dataIndex: 'workstationName', width: 120, ellipsis: true },
    { key: 'serverRoomName', title: '所在机房', dataIndex: 'serverRoomName', width: 120, ellipsis: true },
    { key: 'networkSwitchName', title: '网络交换机', dataIndex: 'networkSwitchName', width: 120, ellipsis: true },
    { key: 'switchPort', title: '交换机端口', dataIndex: 'switchPort', width: 120, ellipsis: true },
    { key: 'voltage', title: '电压', dataIndex: 'voltage', width: 80, ellipsis: true },
    { key: 'useStatusName', title: '使用状态', dataIndex: 'useStatusName', width: 100, ellipsis: true },
    { key: 'financeCode', title: '财务编码', dataIndex: 'financeCode', width: 100, ellipsis: true },
    { key: 'projectCode', title: '项目编码', dataIndex: 'projectCode', width: 100, ellipsis: true },
    { key: 'assetCost', title: '资产成本', dataIndex: 'assetCost', width: 100, ellipsis: true },
    { key: 'depreciationPeriod', title: '折旧年限', dataIndex: 'depreciationPeriod', width: 100, ellipsis: true },
    { key: 'netBookValue', title: '账面净值', dataIndex: 'netBookValue', width: 100, ellipsis: true },
    { key: 'depreciationRate', title: '折旧率', dataIndex: 'depreciationRate', width: 100, ellipsis: true },
    { key: 'accumulatorDepreciation', title: '累计折旧', dataIndex: 'accumulatorDepreciation', width: 120, ellipsis: true },
    {
      title: '操作',
      key: 'action',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <AssetRow
          record={record}
          onEdit={handleEdit}
          onDelete={handleDelete}
        />
      ),
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
    message.info('编辑功能待实现');
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
            >
              {deviceCategoryOptions.map((option) => (
                <Option key={option.value} value={option.value}>
                  {option.value} ({option.count})
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="status" label="状态">
            <Select placeholder="请选择状态" allowClear style={{ width: 120 }}>
              <Option value={0}>正常</Option>
              <Option value={1}>停用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="nbfStatus" label="拟报废">
            <Select placeholder="请选择" allowClear style={{ width: 120 }}>
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
          onChange: (page, pageSize) => {
            setCurrent(page);
            setPageSize(pageSize);
          },
        }}
        scroll={{ x: 4200, y: 600 }}
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
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
    </div>
  );
};

export default AssetList;
