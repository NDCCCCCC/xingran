import React, { useState, useEffect } from 'react';
import { Button, Form, Input, Select, Table, Modal, InputNumber, Space, Tag, message, Popconfirm, Card, Statistic, Row, Col, TreeSelect } from 'antd';
import { PlusOutlined, EditOutlined, DeleteOutlined, SearchOutlined, ReloadOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons';
import type { ColumnsType, TablePaginationConfig } from 'antd/es/table';
import dayjs from 'dayjs';
import {
  getDutyPoolList,
  createDutyPool,
  updateDutyPool,
  deleteDutyPool,
  getUserList,
  getDeptList,
  type DutyPool,
  type DutyPoolCreateRequest,
  type DutyPoolUpdateRequest,
  type SimpleUser,
  type SimpleDept,
} from '@/lib/dutyApi';

const { Option } = Select;

// 将部门数据转换为 TreeSelect 需要的树形结构
const buildDeptTree = (depts: SimpleDept[]): any[] => {
  return depts.map((dept) => ({
    title: dept.deptName,
    value: dept.id,
    key: dept.id,
    children: dept.children && dept.children.length > 0 ? buildDeptTree(dept.children) : undefined,
  }));
};

const DutyPoolPage: React.FC = () => {
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  const [loading, setLoading] = useState(false);
  const [dataSource, setDataSource] = useState<DutyPool[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const [modalVisible, setModalVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<DutyPool | null>(null);

  // 统计数据
  const [stats, setStats] = useState({ total: 0, enabled: 0, disabled: 0, totalMembers: 0 });

  // 用户和部门数据
  const [users, setUsers] = useState<SimpleUser[]>([]);
  const [depts, setDepts] = useState<SimpleDept[]>([]);
  // 筛选后的用户列表（用于弹窗中的值班成员选择）
  const [filteredUsers, setFilteredUsers] = useState<SimpleUser[]>([]);

  // 获取部门及其所有子部门的ID列表
  const getDeptAndChildrenIds = (deptId: string | undefined, deptList: SimpleDept[]): string[] => {
    if (!deptId) return [];
    const ids: string[] = [deptId];

    const findChildren = (parentId: string) => {
      for (const dept of deptList) {
        if (dept.id === parentId && dept.children) {
          for (const child of dept.children) {
            ids.push(child.id);
            findChildren(child.id);
          }
          break;
        }
        // 递归在子部门中查找
        if (dept.children) {
          findChildren(parentId);
        }
      }
    };

    // 更高效的递归方法
    const collectChildrenIds = (dept: SimpleDept): string[] => {
      const result: string[] = [];
      if (dept.children && dept.children.length > 0) {
        for (const child of dept.children) {
          result.push(child.id);
          result.push(...collectChildrenIds(child));
        }
      }
      return result;
    };

    const findAndCollect = (depts: SimpleDept[], targetId: string): string[] => {
      for (const dept of depts) {
        if (dept.id === targetId) {
          return collectChildrenIds(dept);
        }
        if (dept.children && dept.children.length > 0) {
          const result = findAndCollect(dept.children, targetId);
          if (result.length > 0) {
            return result;
          }
        }
      }
      return [];
    };

    ids.push(...findAndCollect(deptList, deptId));
    return ids;
  };

  const fetchList = async (page?: number, pageSize?: number) => {
    setLoading(true);
    try {
      const values = form.getFieldsValue();
      const result = await getDutyPoolList({
        current: page ?? current,
        pageSize: pageSize ?? pageSize,
        poolName: values.poolName,
        status: values.status,
        deptId: values.deptId,
      });
      setDataSource(result.data?.list ?? []);
      setTotal(result.data?.total ?? 0);
      setCurrent(result.data?.current ?? 1);
      setPageSize(result.data?.page_size ?? 10);

      // 计算统计数据
      const list = result.data?.list ?? [];
      setStats({
        total: list.length,
        enabled: list.filter((p) => p.status === 0).length,
        disabled: list.filter((p) => p.status === 1).length,
        totalMembers: list.reduce((sum, p) => sum + (p.members?.length || 0), 0),
      });
    } catch (error) {
      message.error('获取值班池列表失败');
    } finally {
      setLoading(false);
    }
  };

  // 获取用户列表（只获取启用的用户）
  const fetchUsers = async () => {
    try {
      const result = await getUserList({ status: 0 }); // 只获取启用状态的用户
      setUsers(result.data?.list || []);
    } catch (error) {
      console.error('获取用户列表失败:', error);
    }
  };

  // 获取部门列表
  const fetchDepts = async () => {
    try {
      const result = await getDeptList();
      setDepts(result.data || []);
    } catch (error) {
      console.error('获取部门列表失败:', error);
    }
  };

  useEffect(() => {
    fetchList(1, 10);
    fetchUsers();
    fetchDepts();
  }, []);

  const handleTableChange = (pagination: TablePaginationConfig) => {
    fetchList(pagination.current, pagination.pageSize);
  };

  const handleSearch = () => {
    fetchList(1, 10);
  };

  const handleReset = () => {
    form.resetFields();
    fetchList(1, 10);
  };

  const handleAdd = () => {
    setEditingRecord(null);
    editForm.resetFields();
    editForm.setFieldsValue({ dailyCount: 1, memberIds: [] });
    setFilteredUsers([]); // 重置筛选后的用户列表
    setModalVisible(true);
  };

  const handleEdit = (record: DutyPool) => {
    setEditingRecord(record);
    setModalVisible(true);
    // 根据记录的部门筛选用户
    if (record.deptId) {
      const deptIds = getDeptAndChildrenIds(record.deptId, depts);
      const filtered = users.filter(user => user.deptId && deptIds.includes(user.deptId));
      setFilteredUsers(filtered);
    } else {
      setFilteredUsers(users);
    }
  };

  const handleModalOpenChange = (open: boolean) => {
    if (open && editingRecord) {
      // Modal 打开时设置表单值
      editForm.setFieldsValue({
        poolName: editingRecord.poolName,
        deptId: editingRecord.deptId,
        description: editingRecord.description,
        dailyCount: editingRecord.dailyCount,
        memberIds: editingRecord.members?.map((m) => m.userId) || [],
      });
      // 根据记录的部门筛选用户
      if (editingRecord.deptId) {
        const deptIds = getDeptAndChildrenIds(editingRecord.deptId, depts);
        const filtered = users.filter(user => user.deptId && deptIds.includes(user.deptId));
        setFilteredUsers(filtered);
      }
    } else if (open && !editingRecord) {
      // 新增时重置筛选
      setFilteredUsers([]);
    }
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteDutyPool(id);
      message.success('删除成功');
      fetchList();
    } catch (error) {
      message.error('删除失败');
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await editForm.validateFields();
      if (editingRecord) {
        const updateData: DutyPoolUpdateRequest = {
          poolName: values.poolName,
          deptId: values.deptId,
          description: values.description,
          dailyCount: values.dailyCount,
          memberIds: values.memberIds,
        };
        await updateDutyPool(editingRecord.id, updateData);
        message.success('更新成功');
      } else {
        const createData: DutyPoolCreateRequest = {
          poolName: values.poolName,
          deptId: values.deptId,
          description: values.description,
          dailyCount: values.dailyCount,
          memberIds: values.memberIds,
        };
        await createDutyPool(createData);
        message.success('创建成功');
      }
      setModalVisible(false);
      fetchList();
    } catch (error: any) {
      if (error.errorFields) {
        // 表单验证错误
        return;
      }
      message.error(editingRecord ? '更新失败' : '创建失败');
    }
  };

  const columns: ColumnsType<DutyPool> = [
    {
      title: '序号',
      key: 'index',
      width: 60,
      render: (_: unknown, __: unknown, index: number) => (current - 1) * pageSize + index + 1,
    },
    {
      title: '值班池名称',
      dataIndex: 'poolName',
      key: 'poolName',
      width: 150,
    },
    {
      title: '所属部门',
      dataIndex: 'deptId',
      key: 'deptId',
      width: 120,
      render: (deptId: string, record: DutyPool) => {
        // TODO: 从后端API获取部门信息，这里暂用记录中的数据
        return record.dept?.deptName || '-';
      },
    },
    {
      title: '成员数量',
      key: 'memberCount',
      width: 100,
      render: (_: unknown, record: DutyPool) => record.members?.length || 0,
    },
    {
      title: '每日值班人数',
      dataIndex: 'dailyCount',
      key: 'dailyCount',
      width: 120,
      render: (count: number) => `${count} 人/天`,
    },
    {
      title: '成员列表',
      key: 'members',
      width: 300,
      render: (_: unknown, record: DutyPool) => {
        const members = record.members || [];
        if (members.length === 0) return '-';
        return (
          <Space size={[0, 4]} wrap>
            {members.map((m, i) => (
              <Tag key={m.id} color={i < record.dailyCount ? 'blue' : 'default'}>
                {m.user?.nickName || m.user?.username}
              </Tag>
            ))}
          </Space>
        );
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      width: 80,
      render: (status: number) => (
        <Tag color={status === 0 ? 'green' : 'red'}>{status === 0 ? '启用' : '停用'}</Tag>
      ),
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      width: 200,
      ellipsis: true,
    },
    {
      title: '创建时间',
      dataIndex: 'createdAt',
      key: 'createdAt',
      width: 160,
      render: (date: string) => dayjs(date).format('YYYY-MM-DD HH:mm'),
    },
    {
      title: '操作',
      key: 'action',
      width: 150,
      fixed: 'right',
      render: (_: unknown, record: DutyPool) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Popconfirm title="确定要删除吗？" onConfirm={() => handleDelete(record.id)} okText="确定" cancelText="取消">
            <Button type="link" size="small" danger icon={<DeleteOutlined />}>
              删除
            </Button>
          </Popconfirm>
        </Space>
      ),
    },
  ];

  return (
    <div className="p-6">
      <Card title={null} className="mb-4">
        <Row gutter={16}>
          <Col span={6}>
            <Statistic title="总值班池" value={stats.total} prefix={<TeamOutlined />} />
          </Col>
          <Col span={6}>
            <Statistic title="启用中" value={stats.enabled} styles={{ content: { color: '#3f8600' } }} />
          </Col>
          <Col span={6}>
            <Statistic title="已停用" value={stats.disabled} styles={{ content: { color: '#cf1322' } }} />
          </Col>
          <Col span={6}>
            <Statistic title="总成员数" value={stats.totalMembers} prefix={<UserOutlined />} />
          </Col>
        </Row>
      </Card>

      <Card title="值班池管理">
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={form} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="poolName" label="值班池名称">
              <Input placeholder="请输入值班池名称" allowClear style={{ width: 150 }} />
            </Form.Item>
            <Form.Item name="deptId" label="所属部门">
              <TreeSelect
                placeholder="请选择部门"
                allowClear
                style={{ width: 200 }}
                treeData={buildDeptTree(depts)}
                showSearch
                treeNodeFilterProp="title"
              />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" allowClear style={{ width: 100 }}>
                <Option value={0}>启用</Option>
                <Option value={1}>停用</Option>
              </Select>
            </Form.Item>
            <Form.Item>
              <Space>
                <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                  查询
                </Button>
                <Button icon={<ReloadOutlined />} onClick={handleReset}>
                  重置
                </Button>
              </Space>
            </Form.Item>
          </Form>
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              新增值班池
            </Button>
          </Space>
        </div>

        <Table
          rowKey="id"
          columns={columns}
          dataSource={dataSource}
          loading={loading}
          scroll={{ x: 1200 }}
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (t) => `共 ${t} 条`,
          }}
          onChange={handleTableChange}
        />
      </Card>

      <Modal
        title={editingRecord ? '编辑值班池' : '新增值班池'}
        open={modalVisible}
        onOk={handleModalOk}
        afterOpenChange={handleModalOpenChange}
        onCancel={() => setModalVisible(false)}
        width={600}
        destroyOnHidden
      >
        <Form form={editForm} layout="vertical" preserve={false}>
          <Form.Item
            name="poolName"
            label="值班池名称"
            rules={[{ required: true, message: '请输入值班池名称' }]}
          >
            <Input placeholder="请输入值班池名称" />
          </Form.Item>

          <Form.Item name="deptId" label="所属部门">
            <TreeSelect
              placeholder="请选择部门"
              allowClear
              treeData={buildDeptTree(depts)}
              showSearch
              treeNodeFilterProp="title"
              onChange={(value) => {
                // 根据选择的部门筛选用户
                const deptIds = getDeptAndChildrenIds(value, depts);
                if (deptIds.length > 0) {
                  const filtered = users.filter(user => user.deptId && deptIds.includes(user.deptId));
                  setFilteredUsers(filtered);
                } else {
                  setFilteredUsers(users);
                }
                // 清空已选择的成员（因为部门已改变）
                editForm.setFieldValue('memberIds', []);
              }}
            />
          </Form.Item>

          <Form.Item
            name="dailyCount"
            label="每日值班人数"
            rules={[{ required: true, message: '请输入每日值班人数' }]}
          >
            <InputNumber min={1} max={10} placeholder="请输入每日值班人数" style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item
            name="memberIds"
            label="值班成员"
            rules={[{ required: true, message: '请选择值班成员' }]}
          >
            <Select
              mode="multiple"
              placeholder={filteredUsers.length === 0 && users.length > 0 ? '请先选择所属部门' : '请选择值班成员'}
              optionLabelProp="label"
              disabled={filteredUsers.length === 0 && users.length > 0}
            >
              {(filteredUsers.length > 0 ? filteredUsers : users).map((user) => (
                <Option key={user.id} value={user.id} label={user.nickName || user.username}>
                  {user.username} - {user.nickName} {user.deptName ? `(${user.deptName})` : ''}
                </Option>
              ))}
            </Select>
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="请输入描述" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default DutyPoolPage;
