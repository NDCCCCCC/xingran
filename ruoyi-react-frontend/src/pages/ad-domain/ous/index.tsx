import { useState, useEffect } from 'react';
import {
  Tree,
  Card,
  Select,
  Spin,
  Tag,
  Space,
  Button,
  Table,
  message,
  Col,
  Row,
  Modal,
  Transfer
} from 'antd';
import {
  ReloadOutlined,
  FolderOutlined,
  TeamOutlined,
  EditOutlined,
  PlusOutlined,
  DeleteOutlined
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { DataNode } from 'antd/es/tree';
import type { TransferProps } from 'antd/es/transfer';
import {
  getADOUTree,
  getADUserList,
  getADGroupList,
  getOUGroupMappingsByOU,
  createOUGroupMapping,
  deleteOUGroupMapping,
  type ADOUNode,
  type ADUser,
  type ADGroup,
  type OUGroupMapping
} from '@/lib/adDomainApi';
import { useADConfigs } from '@/hooks/useADConfigs';
import type { ADConfig } from '@/lib/adDomainApi';

import type { FC } from 'react';

const ADOUPage: FC = () => {
  const { configs, selectedConfig, setSelectedConfig } = useADConfigs({
    enabledOnly: true,
    autoSelectFirst: true,
  });

  const [ouTree, setOUTree] = useState<ADOUNode[]>([]);
  const [selectedOU, setSelectedOU] = useState<string>('');
  const [selectedOUName, setSelectedOUName] = useState<string>('');
  const [users, setUsers] = useState<ADUser[]>([]);
  const [treeLoading, setTreeLoading] = useState(false);
  const [usersLoading, setUsersLoading] = useState(false);

  // 用户组相关状态
  const [allGroups, setAllGroups] = useState<ADGroup[]>([]);
  const [ouGroups, setOuGroups] = useState<OUGroupMapping[]>([]);
  const [groupsLoading, setGroupsLoading] = useState(false);
  const [groupModalVisible, setGroupModalVisible] = useState(false);
  const [groupModalLoading, setGroupModalLoading] = useState(false);

  const fetchOUTree = async () => {
    if (!selectedConfig) return;

    setTreeLoading(true);
    try {
      const res = await getADOUTree(selectedConfig);
      if (res.code === 0) {
        setOUTree(res.data ?? []);
      }
    } catch {
      message.error('获取OU树失败');
    } finally {
      setTreeLoading(false);
    }
  };

  useEffect(() => {
    if (selectedConfig) {
      fetchOUTree();
    }
  }, [selectedConfig]);

  // 当选择OU时获取该OU下的用户和用户组信息
  const handleOUSelect = async (selectedKeys: React.Key[]) => {
    if (selectedKeys.length > 0) {
      const ouDn = selectedKeys[0] as string;
      const ouNode = ouTree.find(n => n.dn === ouDn);
      setSelectedOU(ouDn);
      setSelectedOUName(ouNode?.name || ouDn);
      fetchUsers(ouDn);
      fetchOUGroupMappings(ouDn);
      fetchAllGroups();
    } else {
      setSelectedOU('');
      setSelectedOUName('');
      setUsers([]);
      setOuGroups([]);
    }
  };

  const fetchUsers = async (ouDn?: string) => {
    if (!selectedConfig) return;

    setUsersLoading(true);
    try {
      const res = await getADUserList({
        configId: selectedConfig,
        ouDn: ouDn || selectedOU,
        current: 1,
        pageSize: 100,
      });
      if (res.code === 0) {
        setUsers(res.data?.list ?? []);
      }
    } catch {
      message.error('获取用户列表失败');
    } finally {
      setUsersLoading(false);
    }
  };

  const fetchAllGroups = async () => {
    if (!selectedConfig) return;

    try {
      const res = await getADGroupList({
        configId: selectedConfig,
        current: 1,
        pageSize: 1000, // 获取所有组用于选择
      });
      if (res.code === 0) {
        setAllGroups(res.data?.list ?? []);
      }
    } catch {
      message.error('获取用户组列表失败');
    }
  };

  const fetchOUGroupMappings = async (ouDn: string) => {
    setGroupsLoading(true);
    try {
      const res = await getOUGroupMappingsByOU(ouDn);
      if (res.code === 0 && res.data) {
        setOuGroups(res.data);
      }
    } catch {
      message.error('获取OU组映射失败');
    } finally {
      setGroupsLoading(false);
    }
  };

  const openGroupModal = () => {
    setGroupModalVisible(true);
  };

  const handleGroupTransfer = async (targetKeys: string[]) => {
    if (!selectedOU) return;

    setGroupModalLoading(true);
    try {
      // 获取当前映射的组ID
      const currentMappedIds = ouGroups.map(g => g.adGroupId);

      // 找出需要添加的组
      const toAdd = targetKeys.filter(id => !currentMappedIds.includes(id));
      // 找出需要删除的组
      const toRemove = currentMappedIds.filter(id => !targetKeys.includes(id));

      // 创建新映射
      for (const groupId of toAdd) {
        const group = allGroups.find(g => g.id === groupId);
        if (group) {
          await createOUGroupMapping({
            adConfigId: selectedConfig,
            ouDn: selectedOU,
            ouName: selectedOUName,
            adGroupId: groupId,
            syncEnabled: true,
          });
        }
      }

      // 删除映射
      for (const mappingId of toRemove) {
        const mapping = ouGroups.find(g => g.adGroupId === mappingId);
        if (mapping) {
          await deleteOUGroupMapping(mapping.id);
        }
      }

      message.success('更新成功');
      await fetchOUGroupMappings(selectedOU);
      setGroupModalVisible(false);
    } catch {
      message.error('更新失败');
    } finally {
      setGroupModalLoading(false);
    }
  };

  const buildTreeData = (nodes: ADOUNode[]): DataNode[] => {
    return nodes.map(node => ({
      title: (
        <Space>
          <FolderOutlined />
          <span>{node.name}</span>
        </Space>
      ),
      key: node.dn,
      children: node.children && node.children.length > 0 ? buildTreeData(node.children) : undefined,
    }));
  };

  const userColumns: ColumnsType<ADUser> = [
    {
      title: '用户名',
      dataIndex: 'username',
      key: 'username',
      fixed: 'left',
      width: 150,
    },
    {
      title: '显示名',
      dataIndex: 'displayName',
      key: 'displayName',
      width: 150,
    },
    {
      title: '邮箱',
      dataIndex: 'email',
      key: 'email',
      width: 200,
      ellipsis: true,
    },
    {
      title: '部门',
      dataIndex: 'department',
      key: 'department',
      width: 150,
    },
    {
      title: '职位',
      dataIndex: 'title',
      key: 'title',
      width: 120,
    },
    {
      title: '办公电话',
      dataIndex: 'phone',
      key: 'phone',
      width: 130,
    },
    {
      title: '手机',
      dataIndex: 'mobile',
      key: 'mobile',
      width: 130,
    },
    {
      title: '状态',
      key: 'status',
      width: 180,
      render: (_: unknown, user: ADUser) => (
        <Space size="small">
          {user.isEnabled && <Tag color="success">启用</Tag>}
          {!user.isEnabled && <Tag color="default">禁用</Tag>}
          {user.isLocked && <Tag color="error">锁定</Tag>}
          {user.passwordExpired && <Tag color="warning">密码过期</Tag>}
        </Space>
      ),
    },
  ];

  const ouGroupColumns: ColumnsType<OUGroupMapping> = [
    {
      title: '组名称',
      dataIndex: 'adGroup',
      key: 'groupName',
      render: (group) => group?.groupName || '-',
    },
    {
      title: '描述',
      dataIndex: 'adGroup',
      key: 'description',
      ellipsis: true,
      render: (group) => group?.description || '-',
    },
    {
      title: '成员数',
      dataIndex: 'adGroup',
      key: 'memberCount',
      width: 100,
      render: (group) => <Tag color="blue">{group?.memberCount || 0}</Tag>,
    },
    {
      title: '同步状态',
      dataIndex: 'syncEnabled',
      key: 'syncEnabled',
      width: 100,
      render: (enabled) => (
        <Tag color={enabled ? 'success' : 'default'}>
          {enabled ? '已启用' : '未启用'}
        </Tag>
      ),
    },
    {
      title: '最后同步',
      dataIndex: 'lastSyncAt',
      key: 'lastSyncAt',
      width: 150,
      render: (time) => time ? new Date(time).toLocaleString('zh-CN') : '-',
    },
  ];

  // Transfer组件的dataSource
  const transferDataSource = allGroups.map(group => ({
    key: group.id,
    title: group.groupName,
    description: group.description,
  }));

  // 当前已映射的组ID
  const targetKeys = ouGroups.map(mapping => mapping.adGroupId);

  return (
    <div style={{ height: 'calc(100vh - 200px)' }}>
      <Row gutter={16} style={{ height: '100%' }}>
        {/* 左侧：OU树 */}
        <Col span={6} style={{ height: '100%', overflow: 'auto' }}>
          <Card
            title="OU组织单位"
            size="small"
            extra={
              <Button
                type="text"
                icon={<ReloadOutlined />}
                onClick={fetchOUTree}
                loading={treeLoading}
              />
            }
            bodyStyle={{ maxHeight: 'calc(100vh - 280px)', overflow: 'auto' }}
          >
            <Space orientation="vertical" style={{ width: '100%' }} size="small">
              <Select
                style={{ width: '100%' }}
                placeholder="选择AD配置"
                value={selectedConfig}
                onChange={setSelectedConfig}
                options={configs.map(c => ({ label: c.configName, value: c.id }))}
              />
              <Spin spinning={treeLoading}>
                {ouTree.length > 0 ? (
                  <Tree
                    showLine
                    defaultExpandAll={false}
                    onSelect={handleOUSelect}
                    treeData={buildTreeData(ouTree)}
                    style={{ fontSize: '13px' }}
                  />
                ) : (
                  <div style={{ textAlign: 'center', color: '#999', padding: '20px' }}>
                    {selectedConfig ? '暂无OU数据，请先同步' : '请选择AD配置'}
                  </div>
                )}
              </Spin>
            </Space>
          </Card>
        </Col>

        {/* 右侧：用户列表和用户组映射 */}
        <Col span={18} style={{ height: '100%', overflow: 'auto' }}>
          <Space direction="vertical" style={{ width: '100%' }} size="small">
            {/* 用户列表卡片 */}
            <Card
              title={
                <Space>
                  <TeamOutlined />
                  <span>
                    用户列表 - {selectedOUName || '全部'}
                  </span>
                </Space>
              }
              size="small"
              bodyStyle={{ maxHeight: 'calc(100vh - 450px)', overflow: 'auto' }}
            >
              <Table
                columns={userColumns}
                dataSource={users}
                loading={usersLoading}
                rowKey="id"
                pagination={{ pageSize: 50, size: 'small' }}
                scroll={{ x: 1200 }}
                size="small"
              />
            </Card>

            {/* 用户组映射卡片 */}
            <Card
              title={
                <Space>
                  <TeamOutlined />
                  <span>关联用户组</span>
                  <Tag color="blue">{ouGroups.length} 个组</Tag>
                </Space>
              }
              size="small"
              extra={
                <Button
                  type="primary"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={openGroupModal}
                  disabled={!selectedOU}
                >
                  修改关联
                </Button>
              }
              loading={groupsLoading}
            >
              {ouGroups.length > 0 ? (
                <Table
                  columns={ouGroupColumns}
                  dataSource={ouGroups}
                  rowKey="id"
                  pagination={false}
                  size="small"
                />
              ) : (
                <div style={{ color: '#999', textAlign: 'center', padding: '20px' }}>
                  {selectedOU ? '该 OU 尚未关联用户组' : '请选择 OU'}
                </div>
              )}
            </Card>
          </Space>
        </Col>
      </Row>

      {/* 用户组选择模态框 */}
      <Modal
        title={`选择关联用户组 - ${selectedOUName}`}
        open={groupModalVisible}
        onCancel={() => setGroupModalVisible(false)}
        width={800}
        footer={null}
      >
        <Spin spinning={groupModalLoading}>
          <Transfer
            dataSource={transferDataSource}
            targetKeys={targetKeys}
            onChange={handleGroupTransfer}
            render={item => (
              <div>
                <div>{item.title}</div>
                <div style={{ fontSize: '12px', color: '#999' }}>{item.description}</div>
              </div>
            )}
            listStyle={{
              width: 350,
              height: 400,
            }}
            titles={['可用用户组', '已关联用户组']}
            showSearch
            filterOption={(inputValue, item) =>
              item.title?.toLowerCase().includes(inputValue.toLowerCase()) ||
              item.description?.toLowerCase().includes(inputValue.toLowerCase())
            }
          />
        </Spin>
      </Modal>
    </div>
  );
};

export default ADOUPage;