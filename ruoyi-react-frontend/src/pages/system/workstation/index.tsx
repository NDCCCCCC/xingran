import type { FC } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  InputNumber,
  Popconfirm,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  Alert,
} from 'antd';
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  UserOutlined,
  UserDeleteOutlined,
  CheckCircleOutlined,
  StopOutlined,
} from '@ant-design/icons';
import type { Workstation, Department, User } from '@/types';
import { post } from '@/lib/api';
import { useTableManager } from '@/hooks/useTableManager';
import { handleApiError, handleSuccess } from '@/utils/errorHandler';
import { useState, useEffect, useCallback } from 'react';
import ActionButtons from '@/components/shared/ActionButtons';

const { Option } = Select;
const { TextArea } = Input;

const WorkstationManagement: FC = () => {
  const [departments, setDepartments] = useState<Department[]>([]);
  const [availableUsers, setAvailableUsers] = useState<User[]>([]);
  const [assignModalVisible, setAssignModalVisible] = useState(false);
  const [assignForm] = Form.useForm();
  const [assigningWorkstation, setAssigningWorkstation] = useState<Workstation | null>(null);
  const [statistics, setStatistics] = useState({
    total: 0,
    active: 0,
    inactive: 0,
  });

  const {
    loading,
    data: workstations,
    total,
    current,
    pageSize,
    selectedRowKeys,
    setSelectedRowKeys,
    searchForm,
    editForm,
    editModalVisible,
    editingItem: editingWorkstation,
    setEditModalVisible,
    setEditingItem,
    loadData: loadWorkstations,
    handleSearch,
    handleReset,
    handleAdd,
    handleEdit,
    handleModalClose,
    setCurrent,
    setPageSize,
  } = useTableManager<Workstation>(
    async (params) => {
      const values = searchForm.getFieldsValue();
      const result = await post('/system/workstations/list', {
        current: params.current || current,
        pageSize: params.pageSize || pageSize,
        ...values,
      });
      return result.data;
    }
  );

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

  // 加载统计数据
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post('/system/workstations/list', { current: 1, pageSize: 1000 });
      const allWorkstations = result.data.list || [];
      const activeCount = allWorkstations.filter((w: any) => w.status === 0).length;
      const inactiveCount = allWorkstations.filter((w: any) => w.status === 1).length;
      setStatistics({
        total: allWorkstations.length,
        active: activeCount,
        inactive: inactiveCount,
      });
    } catch (error) {
      handleApiError(error, '加载统计数据', false);
    }
  }, []);

  useEffect(() => {
    loadDepartments();
    loadStatistics();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 打开编辑模态框
  const openModal = (record?: Workstation) => {
    if (record) {
      setEditingItem(record);
      editForm.setFieldsValue({
        ...record,
        deptId: record.deptId || undefined,
      });
    } else {
      setEditingItem(null);
      editForm.resetFields();
      editForm.setFieldsValue({ workstationType: 0, status: 0, capacity: 1 });
    }
    setEditModalVisible(true);
  };

  // 创建工位
  const handleCreate = async () => {
    try {
      const values = await editForm.validateFields();
      if (editingWorkstation) {
        await post(`/system/workstations/${editingWorkstation.id}/update`, values);
        handleSuccess('更新');
      } else {
        await post('/system/workstations', values);
        handleSuccess('创建');
      }
      handleModalClose();
      loadWorkstations();
      loadStatistics();
    } catch (error: any) {
      if (error.errorFields) {
        return;
      }
      handleApiError(error, '操作');
    }
  };

  // 删除工位
  const handleDelete = async (id: string) => {
    try {
      await post(`/system/workstations/${id}/delete`, {});
      handleSuccess('删除');
      loadWorkstations();
      loadStatistics();
    } catch (error) {
      handleApiError(error, '删除');
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
      handleSuccess('批量删除');
      setSelectedRowKeys([]);
      loadWorkstations();
      loadStatistics();
    } catch (error) {
      handleApiError(error, '批量删除');
    }
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
    await withErrorHandling(
      async () => {
        const values = await assignForm.validateFields();
        await post(`/system/workstations/${assigningWorkstation?.id}/assign`, { userId: values.userId });
        return '分配成功';
      },
      {
        onSuccess: () => {
          setAssignModalVisible(false);
          loadWorkstations();
        },
      }
    );
  };

  // 取消分配用户
  const handleUnassignUser = async (record: Workstation) => {
    if (!record.userId) {
      return;
    }
    await withErrorHandling(
      async () => {
        await post(`/system/workstations/${record.id}/unassign`, {});
        return '取消分配成功';
      },
      {
        onSuccess: () => loadWorkstations(),
      }
    );
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
      width: 100,
      render: (_, record) => {
        const actions = [
          {
            key: 'edit',
            label: '编辑',
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
          ...(!record.userId ? [{
            key: 'assign',
            label: '分配',
            icon: <UserOutlined />,
            onClick: () => openAssignModal(record),
          }] : []),
          {
            key: 'delete',
            label: '删除',
            icon: <DeleteOutlined />,
            danger: true,
            render: () => (
              <Popconfirm title="确认删除?" onConfirm={() => handleDelete(record.id)}>
                <Button type="link" danger icon={<DeleteOutlined />} size="small">
                  删除
                </Button>
              </Popconfirm>
            ),
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];

  return (
    <div>
      {/* 统计卡片 - 只在总数大于10时显示 */}
      {statistics.total > 10 && (
        <Row gutter={16} style={{ marginBottom: 16 }}>
          <Col span={8}>
            <Card>
              <Statistic
                title="总工位数"
                value={statistics.total}
                prefix={<CheckCircleOutlined />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="启用工位"
                value={statistics.active}
                styles={{ content: { color: '#3f8600' } }}
                prefix={<CheckCircleOutlined />}
              />
            </Card>
          </Col>
          <Col span={8}>
            <Card>
              <Statistic
                title="禁用工位"
                value={statistics.inactive}
                styles={{ content: { color: '#cf1322' } }}
                prefix={<StopOutlined />}
              />
            </Card>
          </Col>
        </Row>
      )}

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="workstationCode" label="工位编码">
              <Input placeholder="请输入工位编码" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="workstationName" label="工位名称">
              <Input placeholder="请输入工位名称" allowClear className="user-form-input" style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="deptId" label="部门">
              <Select placeholder="请选择部门" allowClear className="user-form-input" style={{ width: 150 }}>
                {departments.map(dept => (
                  <Option key={dept.id} value={dept.id}>{dept.deptName}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="workstationType" label="工位类型">
              <Select placeholder="请选择工位类型" allowClear className="user-form-input" style={{ width: 130 }}>
                {workstationTypeOptions.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                ))}
              </Select>
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear className="user-form-input" style={{ width: 100 }}>
                <Option value={0}>启用</Option>
                <Option value={1}>禁用</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  搜索
                </Button>
                <Button icon={<ReloadOutlined />} onClick={() => { handleReset(); loadWorkstations(); loadStatistics(); }}>
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
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              新增工位
            </Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && (
          <Alert
            message={
              <span>
                已选择 <strong>{selectedRowKeys.length}</strong> 个工位，
                <Button
                  type="link"
                  size="small"
                  onClick={() => setSelectedRowKeys([])}
                  style={{ padding: 0 }}
                >
                  取消选择
                </Button>
              </span>
            }
            type="info"
            showIcon
            style={{ marginTop: 12 }}
          />
        )}
      </Card>

      {/* 数据表格 */}
      <Card>
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
            onChange: (page, size) => {
              setCurrent(page);
              setPageSize(size || 10);
              loadWorkstations({ current: page, pageSize: size || 10 });
            },
          }}
        />
      </Card>

      {/* 编辑模态框 */}
      <Modal
        title={editingWorkstation ? '编辑工位' : '新增工位'}
        open={editModalVisible}
        onOk={handleCreate}
        onCancel={handleModalClose}
        width={600}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item name="workstationCode" label="工位编码" rules={[{ required: true, message: '请输入工位编码' }]}>
            <Input placeholder="请输入工位编码" disabled={!!editingWorkstation} className="user-form-input" />
          </Form.Item>
          <Form.Item name="workstationName" label="工位名称" rules={[{ required: true, message: '请输入工位名称' }]}>
            <Input placeholder="请输入工位名称" className="user-form-input" />
          </Form.Item>
          <Form.Item name="deptId" label="所属部门">
            <Select placeholder="请选择部门" allowClear className="user-form-input">
              {departments.map(dept => (
                <Option key={dept.id} value={dept.id}>{dept.deptName}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="location" label="位置">
            <Input placeholder="请输入位置" className="user-form-input" />
          </Form.Item>
          <Form.Item name="floor" label="楼层">
            <Input placeholder="请输入楼层" className="user-form-input" />
          </Form.Item>
          <Form.Item name="workstationType" label="工位类型" rules={[{ required: true }]}>
            <Select className="user-form-input">
              {workstationTypeOptions.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="status" label="状态" rules={[{ required: true }]}>
            <Select className="user-form-input">
              <Option value={0}>启用</Option>
              <Option value={1}>禁用</Option>
            </Select>
          </Form.Item>
          <Form.Item name="capacity" label="容量" rules={[{ required: true, message: '请输入容量' }]}>
            <InputNumber min={1} max={100} style={{ width: '100%' }} placeholder="请输入容量" />
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
        <Form form={assignForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item label="工位名称">
            <Input value={assigningWorkstation?.workstationName} disabled className="user-form-input" />
          </Form.Item>
          <Form.Item name="userId" label="选择用户" rules={[{ required: true, message: '请选择用户' }]}>
            <Select placeholder="请选择用户" showSearch optionFilterProp="nickname" className="user-form-input">
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
