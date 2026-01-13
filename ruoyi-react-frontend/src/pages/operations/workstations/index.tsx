import { useState, useEffect, useCallback } from 'react';
import type { FC } from 'react';
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
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  DesktopOutlined,
} from '@ant-design/icons';
import { workstationApi } from '@/services/operations/workstations';
import { floorApi } from '@/services/operations/floors';
import { useTableManager } from '@/hooks/useTableManager';
import type { Workstation, Floor } from '@/types/operations';

const { Option } = Select;

const WorkstationManagement: FC = () => {
  const [floors, setFloors] = useState<Floor[]>([]);

  const fetchWorkstations = useCallback(async (params: any) => {
    const result = await workstationApi.list(params);
    return { list: result.data.list, total: result.data.total };
  }, []);

  const {
    loading,
    data: workstations,
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
    loadData: loadWorkstations,
    handleAdd,
    handleEdit,
    resetSelection,
  } = useTableManager<Workstation>(fetchWorkstations, { pageSize: 10 });

  // 加载楼层列表
  const loadFloors = useCallback(async () => {
    try {
      const result = await floorApi.list({ current: 1, pageSize: 1000 });
      setFloors(result.data.list);
    } catch (error) {
      console.error('加载楼层列表失败:', error);
    }
  }, []);

  useEffect(() => {
    loadWorkstations();
    loadFloors();
  }, [loadWorkstations, loadFloors]);

  const handleSearch = () => {
    const values = searchForm.getFieldsValue();
    const searchParams: any = {};
    Object.keys(values).forEach((key) => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== '') {
        if (key === 'status' || key === 'type') {
          searchParams[key] = Number(value);
        } else {
          searchParams[key] = value;
        }
      }
    });
    loadWorkstations(searchParams);
  };

  const handleReset = () => {
    searchForm.resetFields();
    loadWorkstations();
  };

  const handleRefresh = () => {
    loadWorkstations();
  };

  const handleAddWorkstation = () => {
    handleAdd();
    editForm.setFieldsValue({
      status: 0,
      type: 0,
    });
  };

  const handleModalOpenChange = (open: boolean) => {
    if (open && editingItem) {
      editForm.setFieldsValue(editingItem);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await workstationApi.delete(id);
      Modal.success({
        title: '删除成功',
        content: '工位已成功删除',
      });
      loadWorkstations();
    } catch (error) {
      console.error('删除失败:', error);
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;

    try {
      await workstationApi.batchDelete(selectedRowKeys as string[]);
      Modal.success({
        title: '批量删除成功',
        content: `已成功删除 ${selectedRowKeys.length} 个工位`,
      });
      resetSelection();
      loadWorkstations();
    } catch (error) {
      console.error('批量删除失败:', error);
    }
  };

  const handleSave = async () => {
    try {
      const values = await editForm.validateFields();
      const url = editingItem
        ? `/ops/workstations/${editingItem.id}/update`
        : '/ops/workstations';

      await post(url, values);
      Modal.success({
        title: editingItem ? '更新成功' : '创建成功',
        content: editingItem ? '工位已成功更新' : '工位已成功创建',
      });
      setEditModalVisible(false);
      loadWorkstations();
    } catch (error) {
      console.error('操作失败:', error);
    }
  };

  const columns: ColumnsType<Workstation> = [
    { title: '工位编号', dataIndex: 'code', key: 'code' },
    { title: '工位名称', dataIndex: 'name', key: 'name' },
    {
      title: '所属楼层',
      dataIndex: 'floorId',
      key: 'floorId',
      render: (floorId) => {
        const floor = floors.find((f) => f.id === floorId);
        return floor ? floor.name : '-';
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={status === 0 ? 'success' : 'error'}>
          {status === 0 ? '空闲' : '占用'}
        </Tag>
      ),
    },
    {
      title: '类型',
      dataIndex: 'type',
      key: 'type',
      render: (type) => (
        <Tag color={type === 0 ? 'blue' : 'green'}>
          {type === 0 ? '固定' : '共享'}
        </Tag>
      ),
    },
    { title: '电话信息点', dataIndex: 'phonePoint', key: 'phonePoint', render: (text) => text || '-' },
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
            title="确定要删除这个工位吗？"
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
        <Col span={8}>
          <Card>
            <Statistic
              title="总工位数"
              value={workstations.length}
              prefix={<DesktopOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="空闲"
              value={workstations.filter((w) => w.status === 0).length}
              valueStyle={{ color: '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="占用"
              value={workstations.filter((w) => w.status === 1).length}
              valueStyle={{ color: '#cf1322' }}
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
            <Form.Item name="name" label="工位名称">
              <Input placeholder="请输入工位名称" />
            </Form.Item>
            <Form.Item name="code" label="工位编号">
              <Input placeholder="请输入工位编号" />
            </Form.Item>
            <Form.Item name="floorId" label="所属楼层">
              <Select placeholder="请选择楼层" style={{ width: 200 }} allowClear showSearch>
                {floors.map((floor) => (
                  <Option key={floor.id} value={floor.id}>
                    {floor.name}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" style={{ width: 120 }} allowClear>
                <Option value={0}>空闲</Option>
                <Option value={1}>占用</Option>
              </Select>
            </Form.Item>
            <Form.Item name="type" label="类型">
              <Select placeholder="请选择类型" style={{ width: 120 }} allowClear>
                <Option value={0}>固定</Option>
                <Option value={1}>共享</Option>
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
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAddWorkstation}>
              新增工位
            </Button>
          </Space>
        </div>
      </Card>

      {/* 表格 */}
      <Card>
        <Table
          columns={columns}
          dataSource={workstations}
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
              loadWorkstations({ current: page, pageSize: size });
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
        title={editingItem ? '编辑工位' : '新增工位'}
        open={editModalVisible}
        onOk={handleSave}
        afterOpenChange={handleModalOpenChange}
        onCancel={() => setEditModalVisible(false)}
        width={600}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item
            name="code"
            label="工位编号"
            rules={[{ required: true, message: '请输入工位编号' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="name"
            label="工位名称"
            rules={[{ required: true, message: '请输入工位名称' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="floorId"
            label="所属楼层"
            rules={[{ required: true, message: '请选择所属楼层' }]}
          >
            <Select placeholder="请选择楼层" showSearch>
              {floors.map((floor) => (
                <Option key={floor.id} value={floor.id}>
                  {floor.name}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="phonePoint" label="电话信息点">
            <Input />
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue={0}>
            <Select>
              <Option value={0}>空闲</Option>
              <Option value={1}>占用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="type" label="类型" initialValue={0}>
            <Select>
              <Option value={0}>固定</Option>
              <Option value={1}>共享</Option>
            </Select>
          </Form.Item>
          <Form.Item name="equipment" label="设备配置">
            <Input.TextArea rows={2} placeholder="JSON格式" />
          </Form.Item>
          <Form.Item name="networkInfo" label="网络信息">
            <Input />
          </Form.Item>
          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default WorkstationManagement;
