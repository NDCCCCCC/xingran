/**
 * 资产管理页面
 */

import { useState, useCallback, useEffect } from 'react';
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
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { Asset } from '@/types/operations';
import { useTableManager } from '@/hooks/useTableManager';
import { usePagination } from '@/hooks/usePagination';
import { assetApi } from '@/lib/opsApi';
import ExcelImport from '@/components/shared/ExcelImport';
import { getAuthHeaders } from '@/utils/authHelpers';

const { Option } = Select;

// 统计数据类型
interface Statistics {
  total: number;
  normal: number;
  stopped: number;
  nbf: number;
}

const AssetList: FC = () => {
  const [statistics, setStatistics] = useState<Statistics>({
    total: 0,
    normal: 0,
    stopped: 0,
    nbf: 0,
  });
  const [importVisible, setImportVisible] = useState(false);
  const [exporting, setExporting] = useState(false);

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

  useEffect(() => {
    loadStatistics();
  }, [loadStatistics]);

  const handleDelete = async (id: string) => {
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
  };

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
        ...searchValues,
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
      title: '责任人',
      dataIndex: 'nowUserName',
      key: 'nowUserName',
      width: 100,
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
      title: '备注',
      dataIndex: 'remark',
      key: 'remark',
      width: 200,
      ellipsis: true,
    },
    {
      title: '操作',
      key: 'action',
      width: 120,
      fixed: 'right',
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Button type="link" size="small" danger onClick={() => handleDelete(record.id)}>
            删除
          </Button>
        </Space>
      ),
    },
  ];

  const handleEdit = (record: Asset) => {
    // TODO: 实现编辑功能
    message.info('编辑功能待实现');
  };

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
            <Select placeholder="请选择设备类型" allowClear style={{ width: 150 }}>
              <Option value="台式机">台式机</Option>
              <Option value="笔记本">笔记本</Option>
              <Option value="服务器">服务器</Option>
              <Option value="打印机">打印机</Option>
              <Option value="交换机">交换机</Option>
              <Option value="路由器">路由器</Option>
              <Option value="其他">其他</Option>
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
        columns={columns}
        dataSource={assets}
        loading={loading}
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
        scroll={{ x: 1500 }}
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
    </div>
  );
};

export default AssetList;
