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
  BuildOutlined,
} from '@ant-design/icons';
import { floorApi } from '@/services/operations/floors';
import { buildingApi } from '@/services/operations/buildings';
import { useTableManager } from '@/hooks/useTableManager';
import type { Floor, Building } from '@/types/operations';

const { Option } = Select;

const FloorManagement: FC = () => {
  const [buildings, setBuildings] = useState<Building[]>([]);

  const fetchFloors = useCallback(async (params: any) => {
    const result = await floorApi.list(params);
    return { list: result.data.list, total: result.data.total };
  }, []);

  const {
    loading,
    data: floors,
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
    loadData: loadFloors,
    handleAdd,
    handleEdit,
    resetSelection,
  } = useTableManager<Floor>(fetchFloors, { pageSize: 10 });

  // 加载楼宇列表
  const loadBuildings = useCallback(async () => {
    try {
      const result = await buildingApi.list({ current: 1, pageSize: 1000 });
      setBuildings(result.data.list);
    } catch (error) {
      console.error('加载楼宇列表失败:', error);
    }
  }, []);

  useEffect(() => {
    loadFloors();
    loadBuildings();
  }, [loadFloors, loadBuildings]);

  const handleSearch = () => {
    const values = searchForm.getFieldsValue();
    const searchParams: any = {};
    Object.keys(values).forEach((key) => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== '') {
        if (key === 'status') {
          searchParams[key] = Number(value);
        } else {
          searchParams[key] = value;
        }
      }
    });
    loadFloors(searchParams);
  };

  const handleReset = () => {
    searchForm.resetFields();
    loadFloors();
  };

  const handleRefresh = () => {
    loadFloors();
  };

  const handleAddFloor = () => {
    handleAdd();
    editForm.setFieldsValue({
      status: 0,
      orderNum: 0,
    });
  };

  const handleModalOpenChange = (open: boolean) => {
    if (open && editingItem) {
      editForm.setFieldsValue(editingItem);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await floorApi.delete(id);
      Modal.success({
        title: '删除成功',
        content: '楼层已成功删除',
      });
      loadFloors();
    } catch (error) {
      console.error('删除失败:', error);
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;

    try {
      await floorApi.batchDelete(selectedRowKeys as string[]);
      Modal.success({
        title: '批量删除成功',
        content: `已成功删除 ${selectedRowKeys.length} 个楼层`,
      });
      resetSelection();
      loadFloors();
    } catch (error) {
      console.error('批量删除失败:', error);
    }
  };

  const handleSave = async () => {
    try {
      const values = await editForm.validateFields();
      const url = editingItem ? `/ops/floors/${editingItem.id}/update` : '/ops/floors';

      await post(url, values);
      Modal.success({
        title: editingItem ? '更新成功' : '创建成功',
        content: editingItem ? '楼层已成功更新' : '楼层已成功创建',
      });
      setEditModalVisible(false);
      loadFloors();
    } catch (error) {
      console.error('操作失败:', error);
    }
  };

  const columns: ColumnsType<Floor> = [
    {
      title: '楼层名称',
      dataIndex: 'name',
      key: 'name',
    },
    {
      title: '楼层编号',
      dataIndex: 'floorNo',
      key: 'floorNo',
    },
    {
      title: '所属楼宇',
      dataIndex: 'buildingId',
      key: 'buildingId',
      render: (buildingId) => {
        const building = buildings.find((b) => b.id === buildingId);
        return building ? building.name : '-';
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status) => (
        <Tag color={status === 0 ? 'success' : 'error'}>
          {status === 0 ? '正常' : '停用'}
        </Tag>
      ),
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
            title="确定要删除这个楼层吗？"
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
              title="总楼层数"
              value={floors.length}
              prefix={<BuildOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="正常"
              value={floors.filter((f) => f.status === 0).length}
              valueStyle={{ color: '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="停用"
              value={floors.filter((f) => f.status === 1).length}
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
            <Form.Item name="name" label="楼层名称">
              <Input placeholder="请输入楼层名称" />
            </Form.Item>
            <Form.Item name="buildingId" label="所属楼宇">
              <Select placeholder="请选择楼宇" style={{ width: 200 }} allowClear showSearch>
                {buildings.map((building) => (
                  <Option key={building.id} value={building.id}>
                    {building.name}
                  </Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" style={{ width: 120 }} allowClear>
                <Option value={0}>正常</Option>
                <Option value={1}>停用</Option>
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
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAddFloor}>
              新增楼层
            </Button>
          </Space>
        </div>
      </Card>

      {/* 表格 */}
      <Card>
        <Table
          columns={columns}
          dataSource={floors}
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
              loadFloors({ current: page, pageSize: size });
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
        title={editingItem ? '编辑楼层' : '新增楼层'}
        open={editModalVisible}
        onOk={handleSave}
        afterOpenChange={handleModalOpenChange}
        onCancel={() => setEditModalVisible(false)}
        width={600}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item
            name="name"
            label="楼层名称"
            rules={[{ required: true, message: '请输入楼层名称' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="floorNo"
            label="楼层编号"
            rules={[{ required: true, message: '请输入楼层编号' }]}
          >
            <Input placeholder="如：1F, 2F, B1" />
          </Form.Item>
          <Form.Item
            name="buildingId"
            label="所属楼宇"
            rules={[{ required: true, message: '请选择所属楼宇' }]}
          >
            <Select placeholder="请选择楼宇" showSearch>
              {buildings.map((building) => (
                <Option key={building.id} value={building.id}>
                  {building.name}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue={0}>
            <Select>
              <Option value={0}>正常</Option>
              <Option value={1}>停用</Option>
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

export default FloorManagement;
