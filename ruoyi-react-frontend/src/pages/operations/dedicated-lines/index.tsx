import { useEffect, useCallback } from 'react';
import type { FC } from 'react';
import { post } from '@/lib/api';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Popconfirm,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  DatePicker,
  InputNumber,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  LineChartOutlined,
} from '@ant-design/icons';
import { dedicatedLineApi } from '@/services/operations/dedicated-lines';
import { useTableManager } from '@/hooks/useTableManager';
import type { DedicatedLine } from '@/types/operations';

const { Option } = Select;
const { RangePicker } = DatePicker;

const DedicatedLineManagement: FC = () => {
  const fetchDedicatedLines = useCallback(async (params: any) => {
    const result = await dedicatedLineApi.list(params);
    return { list: result.data.list, total: result.data.total };
  }, []);

  const {
    loading,
    data: dedicatedLines,
    total,
    current,
    pageSize,
    selectedRowKeys,
    searchForm,
    editForm,
    editModalVisible,
    editingItem,
    setCurrent,
    setPageSize,
    setSelectedRowKeys,
    setEditModalVisible,
    loadData: loadDedicatedLines,
    handleAdd,
    handleEdit,
    resetSelection,
  } = useTableManager<DedicatedLine>(fetchDedicatedLines, { pageSize: 10 });

  useEffect(() => {
    loadDedicatedLines();
  }, [loadDedicatedLines]);

  const handleSearch = () => {
    const values = searchForm.getFieldsValue();
    const searchParams: any = {};
    Object.keys(values).forEach((key) => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== '') {
        if (key === 'status') {
          searchParams[key] = Number(value);
        } else if (key === 'dateRange' && value) {
          searchParams['activationDateStart'] = value[0];
          searchParams['activationDateEnd'] = value[1];
        } else {
          searchParams[key] = value;
        }
      }
    });
    loadDedicatedLines(searchParams);
  };

  const handleReset = () => {
    searchForm.resetFields();
    loadDedicatedLines();
  };

  const handleRefresh = () => {
    loadDedicatedLines();
  };

  const handleAddDedicatedLine = () => {
    handleAdd();
    editForm.setFieldsValue({
      status: 0,
      lineType: 'internet',
      monthlyFee: 0,
    });
  };

  const handleModalOpenChange = (open: boolean) => {
    if (open && editingItem) {
      const formValues = {
        ...editingItem,
        activationDate: editingItem.activationDate ? new Date(editingItem.activationDate) : null,
        expiryDate: editingItem.expiryDate ? new Date(editingItem.expiryDate) : null,
      };
      editForm.setFieldsValue(formValues);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await dedicatedLineApi.delete(id);
      Modal.success({
        title: '删除成功',
        content: '专线已成功删除',
      });
      loadDedicatedLines();
    } catch (error) {
      console.error('删除失败:', error);
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;

    try {
      await dedicatedLineApi.batchDelete(selectedRowKeys as string[]);
      Modal.success({
        title: '批量删除成功',
        content: `已成功删除 ${selectedRowKeys.length} 条专线`,
      });
      resetSelection();
      loadDedicatedLines();
    } catch (error) {
      console.error('批量删除失败:', error);
    }
  };

  const handleSave = async () => {
    try {
      const values = await editForm.validateFields();
      const url = editingItem
        ? `/ops/dedicated-lines/${editingItem.id}/update`
        : '/ops/dedicated-lines';

      await post(url, values);
      Modal.success({
        title: editingItem ? '更新成功' : '创建成功',
        content: editingItem ? '专线已成功更新' : '专线已成功创建',
      });
      setEditModalVisible(false);
      loadDedicatedLines();
    } catch (error) {
      console.error('操作失败:', error);
    }
  };

  const columns: ColumnsType<DedicatedLine> = [
    { title: '专线名称', dataIndex: 'name', key: 'name' },
    { title: '专线编码', dataIndex: 'lineCode', key: 'lineCode' },
    {
      title: '专线类型',
      dataIndex: 'lineType',
      key: 'lineType',
      render: (type) => {
        const typeMap: Record<string, { text: string; color: string }> = {
          internet: { text: '互联网专线', color: 'blue' },
          mpls: { text: 'MPLS VPN', color: 'green' },
          leased_line: { text: '租用专线', color: 'orange' },
          fiber: { text: '光纤专线', color: 'purple' },
        };
        const info = typeMap[type] || { text: type, color: 'default' };
        return <Tag color={info.color}>{info.text}</Tag>;
      },
    },
    { title: '带宽', dataIndex: 'bandwidth', key: 'bandwidth' },
    { title: '运营商', dataIndex: 'isp', key: 'isp' },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => {
        const statusMap: Record<number, { text: string; color: string }> = {
          0: { text: '正常', color: 'success' },
          1: { text: '故障', color: 'error' },
          2: { text: '停用', color: 'default' },
        };
        const info = statusMap[status] || { text: String(status), color: 'default' };
        return <Tag color={info.color}>{info.text}</Tag>;
      },
    },
    {
      title: '月租',
      dataIndex: 'monthlyFee',
      key: 'monthlyFee',
      render: (fee) => `¥${fee}`,
    },
    {
      title: '操作',
      key: 'action',
      render: (_, record) => (
        <Space>
          <Button
            type="link"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定要删除这条专线吗？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button type="link" danger size="small" icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic
              title="总专线数"
              value={dedicatedLines.length}
              prefix={<LineChartOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="正常"
              value={dedicatedLines.filter((l) => l.status === 0).length}
              valueStyle={{ color: '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="故障"
              value={dedicatedLines.filter((l) => l.status === 1).length}
              valueStyle={{ color: '#cf1322' }}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="停用"
              value={dedicatedLines.filter((l) => l.status === 2).length}
              valueStyle={{ color: '#8c8c8c' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            flexWrap: 'wrap',
            gap: '16px',
          }}
        >
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="name" label="专线名称">
              <Input placeholder="请输入专线名称" />
            </Form.Item>
            <Form.Item name="lineCode" label="专线编码">
              <Input placeholder="请输入专线编码" />
            </Form.Item>
            <Form.Item name="lineType" label="专线类型">
              <Select placeholder="请选择类型" style={{ width: 150 }} allowClear>
                <Option value="internet">互联网专线</Option>
                <Option value="mpls">MPLS VPN</Option>
                <Option value="leased_line">租用专线</Option>
                <Option value="fiber">光纤专线</Option>
              </Select>
            </Form.Item>
            <Form.Item name="isp" label="运营商">
              <Input placeholder="请输入运营商" />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" style={{ width: 120 }} allowClear>
                <Option value={0}>正常</Option>
                <Option value={1}>故障</Option>
                <Option value={2}>停用</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  搜索
                </Button>
                <Button onClick={handleReset}>重置</Button>
                <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                  刷新
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            {selectedRowKeys.length > 0 && (
              <Button
                danger
                icon={<DeleteOutlined />}
                onClick={handleBatchDelete}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAddDedicatedLine}>
              新增专线
            </Button>
          </Space>
        </div>
      </Card>

      {/* 表格 */}
      <Card>
        <Table
          columns={columns}
          dataSource={dedicatedLines}
          rowKey="id"
          loading={loading}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (total) => `共 ${total} 条`,
            onChange: (page, size) => {
              setCurrent(page);
              setPageSize(size);
              loadDedicatedLines({ current: page, pageSize: size });
            },
          }}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
        />
      </Card>

      {/* 编辑弹窗 */}
      <Modal
        title={editingItem ? '编辑专线' : '新增专线'}
        open={editModalVisible}
        onOk={handleSave}
        afterOpenChange={handleModalOpenChange}
        onCancel={() => setEditModalVisible(false)}
        width={800}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="name"
                label="专线名称"
                rules={[{ required: true, message: '请输入专线名称' }]}
              >
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="lineCode"
                label="专线编码"
                rules={[{ required: true, message: '请输入专线编码' }]}
              >
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="lineType"
                label="专线类型"
                rules={[{ required: true, message: '请选择专线类型' }]}
                initialValue="internet"
              >
                <Select>
                  <Option value="internet">互联网专线</Option>
                  <Option value="mpls">MPLS VPN</Option>
                  <Option value="leased_line">租用专线</Option>
                  <Option value="fiber">光纤专线</Option>
                </Select>
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="bandwidth"
                label="带宽"
                rules={[{ required: true, message: '请输入带宽' }]}
              >
                <Input placeholder="如：100M, 1G" />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="isp"
                label="运营商"
                rules={[{ required: true, message: '请输入运营商' }]}
              >
                <Input placeholder="电信/移动/联通等" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="contractNo" label="合同编号">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="sourceDeviceId" label="源设备ID">
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="sourcePort" label="源端口">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="destDeviceId" label="目的设备ID">
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="destPort" label="目的端口">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="ipAddress" label="IP地址">
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="subnetMask" label="子网掩码">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="gateway" label="网关">
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="vlan" label="VLAN">
                <Input />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="activationDate" label="启用日期">
                <DatePicker style={{ width: '100%' }} />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="expiryDate" label="到期日期">
                <DatePicker style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="responsibleId" label="负责人ID">
                <Input />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="monthlyFee"
                label="月租费用"
                rules={[{ required: true, message: '请输入月租费用' }]}
                initialValue={0}
              >
                <InputNumber min={0} precision={2} style={{ width: '100%' }} />
              </Form.Item>
            </Col>
          </Row>
          <Form.Item name="status" label="状态" initialValue={0}>
            <Select>
              <Option value={0}>正常</Option>
              <Option value={1}>故障</Option>
              <Option value={2}>停用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default DedicatedLineManagement;
