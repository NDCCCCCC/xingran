import { useState, useEffect } from 'react';
import {
  Card,
  Table,
  Select,
  Input,
  Space,
  Button,
  Tag,
  Modal,
  Form,
  message,
  Drawer,
  Popconfirm
} from 'antd';
import {
  ReloadOutlined,
  PlusOutlined,
  MinusCircleOutlined
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import {
  getADGroupList,
  getADGroupMembers,
  addADGroupMember,
  removeADGroupMember,
  updateADGroup,
  getADConfigList,
  getADUserList,
  type ADGroup,
  type ADUser,
  type ADConfig
} from '@/lib/adDomainApi';

import type { FC } from 'react';

const ADGroupPage: FC = () => {
  const [form] = Form.useForm();

  const [configs, setConfigs] = useState<ADConfig[]>([]);
  const [selectedConfig, setSelectedConfig] = useState<string>('');

  const [groups, setGroups] = useState<ADGroup[]>([]);
  const [groupTotal, setGroupTotal] = useState(0);
  const [groupLoading, setGroupLoading] = useState(false);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);

  const [selectedGroup, setSelectedGroup] = useState<ADGroup | null>(null);
  const [members, setMembers] = useState<ADUser[]>([]);
  const [memberTotal, setMemberTotal] = useState(0);
  const [memberLoading, setMemberLoading] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);

  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingGroup, setEditingGroup] = useState<ADGroup | null>(null);

  const [addMemberVisible, setAddMemberVisible] = useState(false);
  const [availableUsers, setAvailableUsers] = useState<ADUser[]>([]);

  useEffect(() => {
    fetchConfigs();
  }, []);

  const fetchConfigs = async () => {
    try {
      const res = await getADConfigList({ current: 1, pageSize: 100, status: 0 });
      if (res.code === 0 && res.data && res.data.list && res.data.list.length > 0) {
        setConfigs(res.data.list);
        if (!selectedConfig) {
          setSelectedConfig(res.data.list[0].id);
        }
      }
    } catch {
      message.error('获取AD配置失败');
    }
  };

  const fetchGroups = async () => {
    if (!selectedConfig) return;

    setGroupLoading(true);
    try {
      const res = await getADGroupList({
        configId: selectedConfig,
        current,
        pageSize,
      });
      if (res.code === 0) {
        setGroups(res.data?.list ?? []);
        setGroupTotal(res.data?.total ?? 0);
      }
    } catch {
      message.error('获取用户组列表失败');
    } finally {
      setGroupLoading(false);
    }
  };

  useEffect(() => {
    if (selectedConfig) {
      fetchGroups();
    }
  }, [selectedConfig, current, pageSize]);

  const handleSearch = () => {
    setCurrent(1);
    fetchGroups();
  };

  const handleViewMembers = async (group: ADGroup) => {
    setSelectedGroup(group);
    setDrawerVisible(true);
    await fetchMembers(group.groupDn);
  };

  const fetchMembers = async (groupDn: string, page = 1) => {
    if (!selectedConfig) return;

    setMemberLoading(true);
    try {
      const res = await getADGroupMembers(groupDn, selectedConfig, {
        current: page,
        pageSize: 50,
      });
      if (res.code === 0) {
        setMembers(res.data?.list ?? []);
        setMemberTotal(res.data?.total ?? 0);
      }
    } catch {
      message.error('获取组成员失败');
    } finally {
      setMemberLoading(false);
    }
  };

  const handleEditGroup = (group: ADGroup) => {
    setEditingGroup(group);
    form.setFieldsValue({
      groupName: group.groupName,
      description: group.description,
    });
    setEditModalVisible(true);
  };

  const handleUpdateGroup = async () => {
    if (!editingGroup || !selectedConfig) return;

    try {
      const values = await form.validateFields();
      await updateADGroup(editingGroup.id, selectedConfig, values);
      message.success('更新成功');
      setEditModalVisible(false);
      fetchGroups();
    } catch (error) {
      message.error((error as Error).message || '更新失败');
    }
  };

  const handleAddMember = async () => {
    if (!selectedConfig) return;

    try {
      const res = await getADUserList({ configId: selectedConfig, current: 1, pageSize: 1000 });
      if (res.code === 0) {
        setAvailableUsers(res.data?.list ?? []);
      }
      setAddMemberVisible(true);
    } catch {
      message.error('获取用户列表失败');
    }
  };

  const handleAddMemberConfirm = async (userDn: string) => {
    if (!selectedGroup || !selectedConfig) return;

    try {
      await addADGroupMember(selectedGroup.id, selectedConfig, userDn);
      message.success('添加成员成功');
      setAddMemberVisible(false);
      fetchMembers(selectedGroup.groupDn);
    } catch (error) {
      message.error((error as Error).message || '添加成员失败');
    }
  };

  const handleRemoveMember = async (userDn: string) => {
    if (!selectedGroup || !selectedConfig) return;

    try {
      await removeADGroupMember(selectedGroup.id, selectedConfig, userDn);
      message.success('移除成员成功');
      fetchMembers(selectedGroup.groupDn);
    } catch (error) {
      message.error((error as Error).message || '移除成员失败');
    }
  };

  const groupColumns: ColumnsType<ADGroup> = [
    {
      title: '组名称',
      dataIndex: 'groupName',
      key: 'groupName',
    },
    {
      title: '描述',
      dataIndex: 'description',
      key: 'description',
      ellipsis: true,
    },
    {
      title: '成员数',
      dataIndex: 'memberCount',
      key: 'memberCount',
      width: 100,
      render: (count: number) => <Tag color="blue">{count}</Tag>,
    },
    {
      title: '作用域',
      dataIndex: 'groupScope',
      key: 'groupScope',
      width: 120,
      render: (scope: string) => scope ? <Tag>{scope}</Tag> : '-',
    },
    {
      title: '操作',
      key: 'action',
      width: 180,
      render: (_: unknown, record: ADGroup) => (
        <Space>
          <Button type="link" size="small" onClick={() => handleViewMembers(record)}>
            查看成员
          </Button>
          <Button type="link" size="small" onClick={() => handleEditGroup(record)}>
            编辑
          </Button>
        </Space>
      ),
    },
  ];

  const memberColumns: ColumnsType<ADUser> = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
    },
    {
      title: '显示名',
      dataIndex: 'displayName',
      key: 'displayName',
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
    },
    {
      title: '部门',
      dataIndex: 'department',
      key: 'department',
    },
    {
      title: '操作',
      key: 'action',
      render: (_: unknown, record: ADUser) => (
        <Popconfirm
          title="确定移除该成员吗？"
          onConfirm={() => handleRemoveMember(record.userDn)}
          okText="确定"
          cancelText="取消"
        >
          <Button type="link" size="small" danger icon={<MinusCircleOutlined />}>
            移除
          </Button>
        </Popconfirm>
      ),
    },
  ];

  return (
    <Card>
      <Space direction="vertical" style={{ width: '100%' }} size="large">
        {/* 搜索栏 */}
        <Form form={form} layout="inline">
          <Form.Item label="AD配置">
            <Select
              style={{ width: 200 }}
              value={selectedConfig}
              onChange={setSelectedConfig}
              options={configs.map(c => ({ label: c.configName, value: c.id }))}
            />
          </Form.Item>
          <Form.Item name="groupName">
            <Input placeholder="组名称" allowClear />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" onClick={handleSearch}>
                查询
              </Button>
              <Button icon={<ReloadOutlined />} onClick={fetchGroups}>
                刷新
              </Button>
            </Space>
          </Form.Item>
        </Form>

        {/* 用户组列表 */}
        <Table
          columns={groupColumns}
          dataSource={groups}
          loading={groupLoading}
          rowKey="id"
          pagination={{
            current,
            pageSize,
            total: groupTotal,
            showSizeChanger: true,
            showQuickJumper: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (page, size) => {
              setCurrent(page);
              setPageSize(size || 10);
            },
          }}
        />
      </Space>

      {/* 成员抽屉 */}
      <Drawer
        title={`组成员 - ${selectedGroup?.groupName}`}
        width={800}
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
      >
        <Space direction="vertical" style={{ width: '100%' }} size="middle">
          <Button type="primary" icon={<PlusOutlined />} onClick={handleAddMember}>
            添加成员
          </Button>
          <Table
            columns={memberColumns}
            dataSource={members}
            loading={memberLoading}
            rowKey="id"
            pagination={{
              current: 1,
              pageSize: 50,
              total: memberTotal,
              showTotal: (t) => `共 ${t} 条`,
              onChange: (page) => fetchMembers(selectedGroup?.groupDn || '', page),
            }}
          />
        </Space>
      </Drawer>

      {/* 编辑用户组弹窗 */}
      <Modal
        title="编辑用户组"
        open={editModalVisible}
        onOk={handleUpdateGroup}
        onCancel={() => setEditModalVisible(false)}
      >
        <Form layout="vertical">
          <Form.Item label="组名称" name="groupName" rules={[{ required: true }]}>
            <Input />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 添加成员弹窗 */}
      <Modal
        title="添加成员"
        open={addMemberVisible}
        onCancel={() => setAddMemberVisible(false)}
        footer={null}
        width={600}
      >
        <Table
          columns={[
            { title: '用户名', dataIndex: 'username', key: 'username' },
            { title: '显示名', dataIndex: 'displayName', key: 'displayName' },
            {
              title: '操作',
              key: 'action',
              render: (_: unknown, record: ADUser) => (
                <Button
                  type="link"
                  icon={<PlusOutlined />}
                  onClick={() => handleAddMemberConfirm(record.userDn)}
                >
                  添加
                </Button>
              ),
            },
          ]}
          dataSource={availableUsers}
          rowKey="id"
          pagination={{ pageSize: 10 }}
          size="small"
        />
      </Modal>
    </Card>
  );
};

export default ADGroupPage;
