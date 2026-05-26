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
  Tree as DepartmentTree
} from 'antd';
import {
  ReloadOutlined,
  FolderOutlined,
  TeamOutlined,
  ApartmentOutlined,
  EditOutlined
} from '@ant-design/icons';
import type { ColumnsType } from 'antd/es/table';
import type { DataNode } from 'antd/es/tree';
import {
  getADOUTree,
  getADUserList,
  getOUDeptMapping,
  updateOUDeptMapping,
  type ADOUNode,
  type ADUser,
  type OUDeptMappingResponse
} from '@/lib/adDomainApi';
import { useADConfigs } from '@/hooks/useADConfigs';
import type { ADConfig } from '@/lib/adDomainApi';
import { getDeptTree } from '@/lib/api';

import type { FC } from 'react';

const ADOUPage: FC = () => {
  // 使用共享的 AD 配置 Hook
  const { configs, selectedConfig, setSelectedConfig } = useADConfigs({
    enabledOnly: true,
    autoSelectFirst: true,
  });

  const [ouTree, setOUTree] = useState<ADOUNode[]>([]);
  const [selectedOU, setSelectedOU] = useState<string>('');
  const [users, setUsers] = useState<ADUser[]>([]);
  const [treeLoading, setTreeLoading] = useState(false);
  const [usersLoading, setUsersLoading] = useState(false);

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

  // 当选择OU时获取该OU下的用户
  const handleOUSelect = (selectedKeys: React.Key[]) => {
    if (selectedKeys.length > 0) {
      const ouDn = selectedKeys[0] as string;
      setSelectedOU(ouDn);
      fetchUsers(ouDn);
    } else {
      setSelectedOU('');
      setUsers([]);
    }
  };

  // 构建树形数据
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

        {/* 右侧：用户列表 */}
        <Col span={18} style={{ height: '100%', overflow: 'auto' }}>
          <Card
            title={
              <Space>
                <TeamOutlined />
                <span>
                  用户列表 - {selectedOU ? (ouTree.find(n => n.dn === selectedOU)?.name || selectedOU) : '全部'}
                </span>
              </Space>
            }
            size="small"
            bodyStyle={{ maxHeight: 'calc(100vh - 280px)', overflow: 'auto' }}
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
        </Col>
      </Row>
    </div>
  );
};

export default ADOUPage;
