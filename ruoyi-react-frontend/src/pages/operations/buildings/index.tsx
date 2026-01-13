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
  Radio,
  Grid,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  BuildingOutlined,
} from '@ant-design/icons';
import type { Key } from 'antd/es/table/interface';
import { post } from '@/lib/api';
import { buildingApi } from '@/services/operations/buildings';
import { useTableManager } from '@/hooks/useTableManager';
import type { Building } from '@/types/operations';

const { Option } = Select;

const BuildingManagement: FC = () => {
  const [viewMode, setViewMode] = useState<'table' | 'card'>('table');
  const [departments, setDepartments] = useState<any[]>([]);

  const fetchBuildings = useCallback(async (params: any) => {
    const result = await buildingApi.list(params);
    return { list: result.data.list, total: result.data.total };
  }, []);

  const {
    loading,
    data: buildings,
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
    loadData: loadBuildings,
    handleAdd,
    handleEdit,
    resetSelection,
  } = useTableManager<Building>(fetchBuildings, { pageSize: 10 });

  // 加载部门列表
  const loadDepartments = useCallback(async () => {
    try {
      const result = await post('/system/departments/tree-select');
      setDepartments(result.data || []);
    } catch (error) {
      console.error('加载部门列表失败:', error);
    }
  }, []);

  useEffect(() => {
    loadBuildings();
    loadDepartments();
  }, [loadBuildings, loadDepartments]);

  const handleSearch = () => {
    const values = searchForm.getFieldsValue();
    const searchParams: any = {};
    Object.keys(values).forEach(key => {
      const value = values[key];
      if (value !== undefined && value !== null && value !== '') {
        if (key === 'status') {
          searchParams[key] = Number(value);
        } else {
          searchParams[key] = value;
        }
      }
    });
    loadBuildings(searchParams);
  };

  const handleReset = () => {
    searchForm.resetFields();
    loadBuildings();
  };

  const handleRefresh = () => {
    loadBuildings();
  };

  const handleAddBuilding = () => {
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
      await buildingApi.delete(id);
      Modal.success({
        title: '删除成功',
        content: '楼宇已成功删除',
      });
      loadBuildings();
    } catch (error) {
      console.error('删除失败:', error);
    }
  };

  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) return;

    try {
      await buildingApi.batchDelete(selectedRowKeys as string[]);
      Modal.success({
        title: '批量删除成功',
        content: `已成功删除 ${selectedRowKeys.length} 个楼宇`,
      });
      resetSelection();
      loadBuildings();
    } catch (error) {
      console.error('批量删除失败:', error);
    }
  };

  const handleSave = async () => {
    try {
      const values = await editForm.validateFields();
      const url = editingItem
        ? `/ops/buildings/${editingItem.id}/update`
        : '/ops/buildings';

      await post(url, values);
      Modal.success({
        title: editingItem ? '更新成功' : '创建成功',
        content: editingItem ? '楼宇已成功更新' : '楼宇已成功创建',
      });
      setEditModalVisible(false);
      loadBuildings();
    } catch (error) {
      console.error('操作失败:', error);
    }
  };

  const columns: ColumnsType<Building> = [
    { title: '楼宇名称', dataIndex: 'name', key: 'name' },
    { title: '楼宇编码', dataIndex: 'code', key: 'code' },
    { title: '地址', dataIndex: 'address', key: 'address' },
    { title: '所属机构', dataIndex: 'orgName', key: 'orgName' },
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
            title="确定要删除这个楼宇吗？"
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

  // 渲染部门树形选项
  const renderDeptTreeOptions = (departments: any[], level = 0) => {
    const options: any[] = [];
    departments.forEach((node) => {
      options.push(
        <Option key={node.id} value={node.id}>
          {'　'.repeat(level * 2)}{node.deptName || node.name}
        </Option>,
      );
      if (node.children && node.children.length > 0) {
        options.push(...renderDeptTreeOptions(node.children, level + 1));
      }
    });
    return options;
  };

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Statistic
              title="总楼宇数"
              value={buildings.length}
              prefix={<BuildingOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="正常"
              value={buildings.filter((b) => b.status === 0).length}
              valueStyle={{ color: '#3f8600' }}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="停用"
              value={buildings.filter((b) => b.status === 1).length}
              valueStyle={{ color: '#cf1322' }}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="name" label="楼宇名称">
              <Input placeholder="请输入楼宇名称" />
            </Form.Item>
            <Form.Item name="code" label="楼宇编码">
              <Input placeholder="请输入楼宇编码" />
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
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAddBuilding}>
              新增楼宇
            </Button>
          </Space>
        </div>
      </Card>

      {/* 视图切换 */}
      <Radio.Group
        value={viewMode}
        onChange={(e) => setViewMode(e.target.value)}
        style={{ marginBottom: 16 }}
      >
        <Radio.Button value="table">表格视图</Radio.Button>
        <Radio.Button value="card">卡片视图</Radio.Button>
      </Radio.Group>

      {/* 内容区 */}
      {viewMode === 'table' ? (
        <Card>
          <Table
            columns={columns}
            dataSource={buildings}
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
                loadBuildings({ current: page, pageSize: size });
              },
            }}
            rowSelection={{
              selectedRowKeys,
              onChange: setSelectedRowKeys,
            }}
          />
        </Card>
      ) : (
        <Grid gutter={16}>
          {buildings.map((building) => (
            <Col span={8} key={building.id}>
              <Card
                title={building.name}
                extra={
                  <Space>
                    <Button
                      size="small"
                      icon={<EditOutlined />}
                      onClick={() => handleEdit(building)}
                    >
                      编辑
                    </Button>
                    <Popconfirm
                      title="确定要删除这个楼宇吗？"
                      onConfirm={() => handleDelete(building.id)}
                      okText="确定"
                      cancelText="取消"
                    >
                      <Button size="small" danger icon={<DeleteOutlined />}>
                        删除
                      </Button>
                    </Popconfirm>
                  </Space>
                }
              >
                <p>
                  <strong>编码：</strong>
                  {building.code}
                </p>
                <p>
                  <strong>地址：</strong>
                  {building.address || '-'}
                </p>
                <p>
                  <strong>状态：</strong>
                  <Tag color={building.status === 0 ? 'success' : 'error'}>
                    {building.status === 0 ? '正常' : '停用'}
                  </Tag>
                </p>
              </Card>
            </Col>
          ))}
        </Grid>
      )}

      {/* 编辑弹窗 */}
      <Modal
        title={editingItem ? '编辑楼宇' : '新增楼宇'}
        open={editModalVisible}
        onOk={handleSave}
        afterOpenChange={handleModalOpenChange}
        onCancel={() => setEditModalVisible(false)}
        width={600}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item
            name="name"
            label="楼宇名称"
            rules={[{ required: true, message: '请输入楼宇名称' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            name="code"
            label="楼宇编码"
            rules={[{ required: true, message: '请输入楼宇编码' }]}
          >
            <Input />
          </Form.Item>
          <Form.Item name="address" label="地址">
            <Input />
          </Form.Item>
          <Form.Item
            name="orgId"
            label="所属机构"
            rules={[{ required: true, message: '请选择所属机构' }]}
          >
            <Select placeholder="请选择机构" allowClear showSearch>
              {renderDeptTreeOptions(departments)}
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

export default BuildingManagement;
