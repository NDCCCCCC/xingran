import { useState, useEffect, useCallback } from "react";
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
  Descriptions,
  Tooltip,
} from "antd";
import {
  ReloadOutlined,
  PlusOutlined,
  MinusCircleOutlined,
  SyncOutlined,
  InfoCircleOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import {
  getADGroupList,
  getADGroupMembers,
  addADGroupMember,
  removeADGroupMember,
  updateADGroup,
  getADUserList,
  getADOUTree,
  syncADGroups,
  getADGroupSyncStatus,
  type ADGroup,
  type ADUser,
  type ADOUNode,
  type ADGroupSyncStatus,
} from "@/lib/adDomainApi";
import { useADConfigs } from "@/hooks/useADConfigs";

import type { FC } from "react";
import { usePagination } from "@/hooks/usePagination";

const ADGroupPage: FC = () => {
  const [form] = Form.useForm();

  // 使用共享的 AD 配置 Hook
  const { configs, selectedConfig, setSelectedConfig } = useADConfigs({
    enabledOnly: true,
    autoSelectFirst: true,
  });

  const [groups, setGroups] = useState<ADGroup[]>([]);
  const [groupLoading, setGroupLoading] = useState(false);

  // 排序状态
  const [orderByColumn, setOrderByColumn] = useState<string>("");
  const [isAsc, setIsAsc] = useState<boolean>(true);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  const [selectedGroup, setSelectedGroup] = useState<ADGroup | null>(null);
  const [members, setMembers] = useState<ADUser[]>([]);
  const [memberTotal, setMemberTotal] = useState(0);
  const [memberLoading, setMemberLoading] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);

  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingGroup, setEditingGroup] = useState<ADGroup | null>(null);

  const [addMemberVisible, setAddMemberVisible] = useState(false);
  const [availableUsers, setAvailableUsers] = useState<ADUser[]>([]);

  const [searchGroupName, setSearchGroupName] = useState<string>();
  const [ouList, setOUList] = useState<Array<{ dn: string; name: string }>>([]);
  const [selectedOUDN, setSelectedOUDN] = useState<string | undefined>();

  // Group sync state
  const [syncLoading, setSyncLoading] = useState(false);
  const [syncStatus, setSyncStatus] = useState<ADGroupSyncStatus | null>(null);

  // 扁平化OU树，提取所有OU
  const flattenOUTree = (nodes: ADOUNode[]): Array<{ dn: string; name: string }> => {
    const result: Array<{ dn: string; name: string }> = [];
    const traverse = (node: ADOUNode) => {
      result.push({ dn: node.dn, name: node.name });
      if (node.children) {
        node.children.forEach(traverse);
      }
    };
    nodes.forEach(traverse);
    return result;
  };

  // 获取OU列表
  useEffect(() => {
    if (!selectedConfig) return;

    const fetchOUs = async () => {
      try {
        const res = await getADOUTree(selectedConfig);
        if (res.code === 0 && res.data) {
          const flattened = flattenOUTree(res.data);
          setOUList(flattened);

          // 默认选择"本部部门分组"
          const defaultOU = flattened.find(ou => ou.name === "本部部门分组");
          if (defaultOU) {
            setSelectedOUDN(defaultOU.dn);
          }
        }
      } catch {
        // 忽略错误
      }
    };

    fetchOUs();
  }, [selectedConfig]);

  const fetchGroups = useCallback(async (groupName?: string, sortCol?: string, sortAsc?: boolean) => {
    if (!selectedConfig) return;

    setGroupLoading(true);
    try {
      // 接受 sortCol/sortAsc 参数（handleTableChange 同步传新值，规避 React 18 setState 异步时序）
      const orderCol = sortCol ?? orderByColumn;
      const asc = sortAsc ?? isAsc;
      const res = await getADGroupList({
        configId: selectedConfig,
        ouDn: selectedOUDN,
        groupName: groupName,
        current: paginationProps.current,
        pageSize: paginationProps.pageSize,
        orderByColumn: orderCol,
        isAsc: asc,
      });
      if (res.code === 0) {
        setGroups(res.data?.list ?? []);
        setTotal(res.data?.total ?? 0);
        setSearchGroupName(groupName);
      }
    } catch {
      message.error("获取用户组列表失败");
    } finally {
      setGroupLoading(false);
    }
  }, [selectedConfig, selectedOUDN, paginationProps.current, paginationProps.pageSize]);

  useEffect(() => {
    if (selectedConfig) {
      fetchGroups(searchGroupName);
    }
  }, [selectedConfig, selectedOUDN, paginationProps.current, paginationProps.pageSize, searchGroupName, fetchGroups]);

  // Fetch sync status when config changes
  const fetchSyncStatus = useCallback(async () => {
    if (!selectedConfig) return;
    try {
      const res = await getADGroupSyncStatus(selectedConfig);
      if (res.code === 0 && res.data) {
        setSyncStatus(res.data);
      }
    } catch {
      // ignore
    }
  }, [selectedConfig]);

  useEffect(() => {
    fetchSyncStatus();
  }, [fetchSyncStatus]);

  const handleSyncGroups = async () => {
    if (!selectedConfig) return;
    setSyncLoading(true);
    try {
      const res = await syncADGroups(selectedConfig);
      if (res.code === 0 && res.data) {
        const r = res.data;
        message.success(
          `组同步完成: 总数=${r.totalGroups}, 创建=${r.createdGroups}, 更新=${r.updatedGroups}, 删除=${r.deletedGroups}, 耗时=${r.duration}ms`
        );
        fetchGroups(searchGroupName);
        fetchSyncStatus();
      }
    } catch {
      message.error("组同步失败");
    } finally {
      setSyncLoading(false);
    }
  };

  const handleSearch = () => {
    const groupName = form.getFieldValue("groupName");
    setCurrent(1);
    fetchGroups(groupName);
  };

  const handleOUChange = (ouDn: string) => {
    setSelectedOUDN(ouDn || undefined);
    setCurrent(1);
  };

  // 表格排序双向关联
  const handleTableChange = (pagination: any, _filters: any, sorter: any) => {
    // 用 local const 持有新值传 fetchGroups，规避 React 18 setState 异步时序
    const newCol = sorter?.field || "";
    const newAsc = sorter?.order === "ascend";
    setOrderByColumn(newCol);
    setIsAsc(newAsc);
    fetchGroups(searchGroupName, newCol, newAsc);
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
      message.error("获取组成员失败");
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
      message.success("更新成功");
      setEditModalVisible(false);
      fetchGroups();
    } catch (error) {
      message.error((error as Error).message || "更新失败");
    }
  };

  const handleAddMember = async () => {
    if (!selectedConfig) return;

    try {
      const res = await getADUserList({ configId: selectedConfig, current: 1, pageSize: 50 });
      if (res.code === 0) {
        setAvailableUsers(res.data?.list ?? []);
      }
      setAddMemberVisible(true);
    } catch {
      message.error("获取用户列表失败");
    }
  };

  const handleAddMemberConfirm = async (userDn: string) => {
    if (!selectedGroup || !selectedConfig) return;

    try {
      await addADGroupMember(selectedGroup.id, selectedConfig, userDn);
      message.success("添加成员成功");
      setAddMemberVisible(false);
      fetchMembers(selectedGroup.groupDn);
    } catch (error) {
      message.error((error as Error).message || "添加成员失败");
    }
  };

  const handleRemoveMember = async (userDn: string) => {
    if (!selectedGroup || !selectedConfig) return;

    try {
      await removeADGroupMember(selectedGroup.id, selectedConfig, userDn);
      message.success("移除成员成功");
      fetchMembers(selectedGroup.groupDn);
    } catch (error) {
      message.error((error as Error).message || "移除成员失败");
    }
  };

  const groupColumns: ColumnsType<ADGroup> = [
    {
      title: "组名称",
      dataIndex: "groupName",
      key: "groupName",
      sorter: true,
    },
    {
      title: "描述",
      dataIndex: "description",
      key: "description",
      ellipsis: true,
      sorter: true,
    },
    {
      title: "成员数",
      dataIndex: "memberCount",
      key: "memberCount",
      width: 100,
      sorter: true,
      render: (count: number) => <Tag color="blue">{count}</Tag>,
    },
    {
      title: "作用域",
      dataIndex: "groupScope",
      key: "groupScope",
      width: 120,
      render: (scope: string) => scope ? <Tag>{scope}</Tag> : "-",
    },
    {
      title: "类型",
      dataIndex: "groupType",
      key: "groupType",
      width: 100,
      render: (type: number) => {
        if (type === 1) return <Tag color="blue">安全组</Tag>;
        if (type === 2) return <Tag color="default">分发组</Tag>;
        return "-";
      },
    },
    {
      title: "最后同步",
      dataIndex: "lastSyncAt",
      key: "lastSyncAt",
      width: 170,
      render: (val: string) => val ? new Date(val).toLocaleString() : <Tag color="warning">未同步</Tag>,
    },
    {
      title: "操作",
      key: "action",
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
      title: "用户名",
      dataIndex: "username",
      key: "username",
    },
    {
      title: "显示名",
      dataIndex: "displayName",
      key: "displayName",
    },
    {
      title: "邮箱",
      dataIndex: "email",
      key: "email",
    },
    {
      title: "部门",
      dataIndex: "department",
      key: "department",
    },
    {
      title: "操作",
      key: "action",
      render: (_: unknown, record: ADUser) => (
        <Button
          type="link"
          size="small"
          danger
          icon={<MinusCircleOutlined />}
          onClick={() => {
            Modal.confirm({
              title: "确定移除该成员吗？",
              okText: "确定",
              cancelText: "取消",
              okButtonProps: { danger: true },
              onOk: () => handleRemoveMember(record.userDn),
            });
          }}
        >
          移除
        </Button>
      ),
    },
  ];

  return (
    <Card>
      <Space orientation="vertical" style={{ width: "100%" }} size="large">
        {/* 搜索栏 */}
        <Form form={form} layout="inline">
          <Form.Item label="AD配置">
            <Select
              style={{ width: 200 }}
              value={selectedConfig}
              onChange={setSelectedConfig}
              options={configs.map(c =>    ({ label: c.configName, value: c.id }))}
             onSearch={() => {}}/>
          </Form.Item>
          <Form.Item label="所属OU">
            <Select
              style={{ width: 250 }}
              value={selectedOUDN}
              onChange={handleOUChange}
              allowClear
              placeholder="选择OU（默认：本部部门分组）"
              showSearch
              optionFilterProp="children"
             onSearch={() => {}}>
              {ouList.map(ou => (
                <Select.Option key={ou.dn} value={ou.dn}>
                  {ou.name}
                </Select.Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="groupName">
            <Input placeholder="组名称" allowClear />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" onClick={handleSearch}>
                查询
              </Button>
              <Button icon={<ReloadOutlined />} onClick={() => fetchGroups(searchGroupName)}>
                刷新
              </Button>
              <Button
                type="primary"
                icon={<SyncOutlined spin={syncLoading} />}
                loading={syncLoading}
                onClick={handleSyncGroups}
                disabled={!selectedConfig}
              >
                同步组
              </Button>
              {syncStatus && (
                <Tooltip title={
                  `总组数: ${syncStatus.totalGroups} | ` +
                  `已同步: ${syncStatus.recentlySynced} | ` +
                  `未同步: ${syncStatus.neverSynced} | ` +
                  `成员关系: ${syncStatus.totalMemberRelations}`
                }>
                  <Tag color={syncStatus.neverSynced > 0 ? "orange" : "green"} style={{ cursor: "pointer" }}>
                    <InfoCircleOutlined /> 同步状态
                  </Tag>
                </Tooltip>
              )}
            </Space>
          </Form.Item>
        </Form>

        {/* Sync Status Summary */}
        {syncStatus && (
          <Descriptions size="small" bordered column={4}>
            <Descriptions.Item label="总组数">{syncStatus.totalGroups}</Descriptions.Item>
            <Descriptions.Item label="最近同步">{syncStatus.recentlySynced}</Descriptions.Item>
            <Descriptions.Item label="未同步">
              <Tag color={syncStatus.neverSynced > 0 ? "warning" : "success"}>
                {syncStatus.neverSynced}
              </Tag>
            </Descriptions.Item>
            <Descriptions.Item label="成员关系数">{syncStatus.totalMemberRelations}</Descriptions.Item>
          </Descriptions>
        )}

        {/* 用户组列表 */}
        <Table
          columns={groupColumns}
          dataSource={groups}
          loading={groupLoading}
          rowKey="id"
          pagination={paginationProps}
          onChange={handleTableChange}
        />
      </Space>

      {/* 成员抽屉 */}
      <Drawer
        title={`组成员 - ${selectedGroup?.groupName}`}
        size="large"
        open={drawerVisible}
        onClose={() => setDrawerVisible(false)}
      >
        <Space orientation="vertical" style={{ width: "100%" }} size="middle">
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAddMember}>
              添加成员
            </Button>
          </Space>
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
              onChange: (page) => {
                fetchMembers(selectedGroup?.groupDn || "", page);
              },
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
            { title: "用户名", dataIndex: "username", key: "username" },
            { title: "显示名", dataIndex: "displayName", key: "displayName" },
            {
              title: "操作",
              key: "action",
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
