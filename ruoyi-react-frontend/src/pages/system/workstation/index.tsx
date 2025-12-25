import React, { useState, useEffect } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  InputNumber,
  message,
  Popconfirm,
  Tag,
} from 'antd';
import type { ColumnsType } from 'antd/es/table';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  UserOutlined,
  UserDeleteOutlined,
} from '@ant-design/icons';
import type { Workstation, Department, User } from '@/types';
import { post } from '@/lib/api';

const { Option } = Select;
const { TextArea } = Input;

const WorkstationManagement: React.FC = () => {
  const [workstations, setWorkstations] = useState<Workstation[]>([]);
  const [loading, setLoading] = useState(false);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [searchForm] = Form.useForm();
  const [modalVisible, setModalVisible] = useState(false);
  const [assignModalVisible, setAssignModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [assignForm] = Form.useForm();
  const [editingWorkstation, setEditingWorkstation] = useState<Workstation | null>(null);
  const [assigningWorkstation, setAssigningWorkstation] = useState<Workstation | null>(null);
  const [selectedRowKeys, setSelectedRowKeys] = useState<React.Key[]>([]);
  const [departments, setDepartments] = useState<Department[]>([]);
  const [availableUsers, setAvailableUsers] = useState<User[]>([]);

  // 加载工位列表
  const loadWorkstations = async (params: any = {}) => {
    setLoading(true);
    try {
      const values = searchForm.getFieldsValue();
      const result = await post('/system/workstations/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...values,
      });
      setWorkstations(result.data.list);
      setTotal(result.data.total);
    } catch (error) {
      console.error('加载工位列表失败:', error);
    } finally {
      setLoading(false);
    }
  };

  // 加载部门列表
  const loadDepartments = async () => {
    try {
      const result = await post('/system/departments/list', { current: 1, pageSize: 1000 });
      setDepartments(result.data.list || []);
    } catch (error) {
      console.error('加载部门列表失败:', error);
    }
  };

  // 加载可用用户列表
  const loadAvailableUsers = async () => {
    try {
      const result = await post('/system/users/list', { current: 1, pageSize: 1000, status: 0 });
      setAvailableUsers(result.data.list || []);
    } catch (error) {
      console.error('加载用户列表失败:', error);
    }
  };

  // 创建工位
  const handleCreate = async () => {
    try {
      const values = await editForm.validateFields();
      if (editingWorkstation) {
        await post(`/system/workstations/${editingWorkstation.id}/update`, values);
        message.success('更新成功');
      } else {
        await post('/system/workstations', values);
        message.success('创建成功');
      }
      setModalVisible(false);
      editForm.resetFields();
      setEditingWorkstation(null);
      loadWorkstations();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error('操作失败');
    }
  };

  // 删除工位
  const handleDelete = async (id: string) => {
    try {
      await post(`/system/workstations/${id}/delete`, {});
      message.success('删除成功');
      loadWorkstations();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 批量删除工位
  const handleBatchDelete = async () => {
    if (selectedRowKeys.length === 0) {
      message.warning('请选择要删除的数据');
      return;
    }
    try {
      await post('/system/workstations/batch-delete', { ids: selectedRowKeys });
      message.success('批量删除成功');
      setSelectedRowKeys([]);
      loadWorkstations();
    } catch (error) {
      message.error('批量删除失败');
    }
  };

  // 打开编辑模态框
  const openModal = (record?: Workstation) => {
    if (record) {
      setEditingWorkstation(record);
      editForm.setFieldsValue({
        ...record,
        deptId: record.deptId || undefined,
      });
    } else {
      setEditingWorkstation(null);
      editForm.resetFields();
      editForm.setFieldsValue({ workstationType: 0, status: 0, capacity: 1 });
    }
    setModalVisible(true);
    loadDepartments();
  };

  // 打开分配用户模态框
  const openAssignModal = (record: Workstation) => {
    if (record.userId) {
      message.warning('该工位已分配用户，请先取消分配');
      return;
    }
    setAssigningWorkstation(record);
    assignForm.resetFields();
    setAssignModalVisible(true);
    loadAvailableUsers();
  };

  // 分配用户
  const handleAssignUser = async () => {
    try {
      const values = await assignForm.validateFields();
      await post(`/system/workstations/${assigningWorkstation?.id}/assign`, { userId: values.userId });
      message.success('分配成功');
      setAssignModalVisible(false);
      loadWorkstations();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      message.error('分配失败');
    }
  };

  // 取消分配用户
  const handleUnassignUser = async (record: Workstation) => {
    if (!record.userId) {
      return;
    }
    try {
      await post(`/system/workstations/${record.id}/unassign`, {});
      message.success('取消分配成功');
      loadWorkstations();
    } catch (error) {
      message.error('取消分配失败');
    }
  };

  // 工位类型选项
  const workstationTypeOptions = [
    { label: '固定工位', value: 0 },
    { label: '灵活工位', value: 1 },
    { label: '管理工位', value: 2 },
  ];

  // 表格列
  const columns: ColumnsType<Workstation> = [
    { title: '工位编码', dataIndex: 'workstationCode', key: 'workstationCode', width: 130 },
    { title: '工位名称', dataIndex: 'workstationName', key: 'workstationName', width: 150 },
    { title: '部门', dataIndex: 'deptName', key: 'deptName', width: 120 },
    { title: '位置', dataIndex: 'location', key: 'location', width: 150, ellipsis: true },
    { title: '楼层', dataIndex: 'floor', key: 'floor', width: 100 },
    {
      title: '工位类型',
      dataIndex: 'workstationType',
      key: 'workstationType',
      width: 100,
      render: (workstationType: number) => {
        const option = workstationTypeOptions.find(o => o.value === workstationType);
        return <Tag color="blue">{option?.label}</Tag>;
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: number) => (
        <Tag color={status === 0 ? 'success' : 'error'}>{status === 0 ? '启用' : '禁用'}</Tag>
      ),
    },
    { title: '容量', dataIndex: 'capacity', key: 'capacity', width: 80 },
    {
      title: '使用人',
      dataIndex: 'userName',
      key: 'userName',
      width: 100,
      render: (userName: string, record) => (
        <Space>
          <span>{userName || '-'}</span>
          {userName && (
            <Popconfirm title="确认取消分配?" onConfirm={() => handleUnassignUser(record)}>
              <Button type="link" size="small" danger icon={<UserDeleteOutlined />} />
            </Popconfirm>
          )}
        </Space>
      ),
    },
    { title: '备注', dataIndex: 'description', key: 'description', ellipsis: true },
    { title: '创建时间', dataIndex: 'createdAt', key: 'createdAt', width: 180 },
    {
      title: '操作',
      key: 'action',
      fixed: 'right',
      width: 200,
      render: (_, record) => (
        <Space size="small">
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(record)}>编辑</Button>
          {!record.userId && (
            <Button type="link" size="small" icon={<UserOutlined />} onClick={() => openAssignModal(record)}>分配</Button>
          )}
          <Popconfirm title="确认删除?" onConfirm={() => handleDelete(record.id)}>
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>删除</Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  useEffect(() => {
    loadWorkstations();
  }, [current, pageSize]);

  return (
    <div>
      <Form form={searchForm} layout="inline" style={{ marginBottom: 16 }}>
        <Form.Item name="workstationCode" label="工位编码">
          <Input placeholder="请输入工位编码" allowClear style={{ width: 150 }} />
        </Form.Item>
        <Form.Item name="workstationName" label="工位名称">
          <Input placeholder="请输入工位名称" allowClear style={{ width: 150 }} />
        </Form.Item>
        <Form.Item name="deptId" label="部门">
          <Select placeholder="请选择部门" allowClear style={{ width: 150 }}>
            {departments.map(dept => (
              <Option key={dept.id} value={dept.id}>{dept.deptName}</Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="workstationType" label="工位类型">
          <Select placeholder="请选择工位类型" allowClear style={{ width: 130 }}>
            {workstationTypeOptions.map(opt => (
              <Option key={opt.value} value={opt.value}>{opt.label}</Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="status" label="状态">
          <Select placeholder="请选择状态" allowClear style={{ width: 100 }}>
            <Option value={0}>启用</Option>
            <Option value={1}>禁用</Option>
          </Select>
        </Form.Item>
        <Form.Item>
          <Space>
            <Button type="primary" icon={<SearchOutlined />} onClick={() => { loadWorkstations(); loadDepartments(); }}>查询</Button>
            <Button icon={<ReloadOutlined />} onClick={() => { searchForm.resetFields(); loadWorkstations(); }}>重置</Button>
          </Space>
        </Form.Item>
      </Form>

      <div style={{ marginBottom: 16 }}>
        <Space>
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>新增</Button>
          <Button danger icon={<DeleteOutlined />} onClick={handleBatchDelete} disabled={selectedRowKeys.length === 0}>批量删除</Button>
        </Space>
      </div>

      <Table
        rowSelection={{
          selectedRowKeys,
          onChange: setSelectedRowKeys,
        }}
        columns={columns}
        dataSource={workstations}
        loading={loading}
        rowKey="id"
        scroll={{ x: 1400 }}
        pagination={{
          current,
          pageSize,
          total,
          showSizeChanger: true,
          showQuickJumper: true,
          showTotal: (total) => `共 ${total} 条`,
          onChange: (page, size) => { setCurrent(page); setPageSize(size || 10); },
        }}
      />

      {/* 编辑模态框 */}
      <Modal
        title={editingWorkstation ? '编辑工位' : '新增工位'}
        open={modalVisible}
        onOk={handleCreate}
        onCancel={() => { setModalVisible(false); editForm.resetFields(); setEditingWorkstation(null); }}
        width={600}
      >
        <Form form={editForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item name="workstationCode" label="工位编码" rules={[{ required: true, message: '请输入工位编码' }]}>
            <Input placeholder="请输入工位编码" disabled={!!editingWorkstation} />
          </Form.Item>
          <Form.Item name="workstationName" label="工位名称" rules={[{ required: true, message: '请输入工位名称' }]}>
            <Input placeholder="请输入工位名称" />
          </Form.Item>
          <Form.Item name="deptId" label="所属部门">
            <Select placeholder="请选择部门" allowClear>
              {departments.map(dept => (
                <Option key={dept.id} value={dept.id}>{dept.deptName}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="location" label="位置">
            <Input placeholder="请输入位置" />
          </Form.Item>
          <Form.Item name="floor" label="楼层">
            <Input placeholder="请输入楼层" />
          </Form.Item>
          <Form.Item name="workstationType" label="工位类型" rules={[{ required: true }]}>
            <Select>
              {workstationTypeOptions.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select>
              <Option value={0}>启用</Option>
              <Option value={1}>禁用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="capacity" label="容量" rules={[{ required: true, message: '请输入容量' }]}>
            <InputNumber min={1} max={100} style={{ width: '100%' }} />
          </Form.Item>
          <Form.Item name="description" label="备注">
            <TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 分配用户模态框 */}
      <Modal
        title="分配用户"
        open={assignModalVisible}
        onOk={handleAssignUser}
        onCancel={() => setAssignModalVisible(false)}
        width={500}
      >
        <Form form={assignForm} labelCol={{ span: 6 }} wrapperCol={{ span: 16 }}>
          <Form.Item label="工位名称">
            <Input value={assigningWorkstation?.workstationName} disabled />
          </Form.Item>
          <Form.Item name="userId" label="选择用户" rules={[{ required: true, message: '请选择用户' }]}>
            <Select placeholder="请选择用户" showSearch optionFilterProp="nickname">
              {availableUsers.map(user => (
                <Option key={user.id} value={user.id}>
                  {user.nickname || user.username} ({user.deptName || '无部门'})
                </Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default WorkstationManagement;
