import { useState, useEffect, useCallback } from "react";
// Phase 71 (W5)：裸 Input/Select/TextArea → 契约组件（children 形态改写 options）
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import {
  Card,
  Table,
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
import type { TablePaginationConfig } from "antd/es/table";
import type { SorterResult } from "antd/es/table/interface";
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
import { confirmDelete } from "@/components/shared/confirmDelete";
// ⚠️ 直接引模块路径（不走 @/components/shared 桶文件）：桶文件 re-export 了
// FloorPlanEditor → @react-three/fiber → three.js，引入它会让本页的动态 import
// （既有测试的 `await import(...)`）显著变慢并触发超时（实测）。
import { useActionPermission } from "@/components/shared/useActionPermission";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

import type { FC } from "react";
import { usePagination } from "@/hooks/usePagination";

const ADGroupPage: FC = () => {
  const [form] = Form.useForm();

  // Phase 86 W4（RBAC-02）：操作级权限门控（三态 ready / loading / error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  // 组编辑 / 成员增删 → ops:ad:group:edit（后端 POST /groups/:id/update 同一码组）
  const permGroupEdit = permGate.decide(PERMISSION_CODES.opsADGroup.edit);
  // 工具栏「同步组」→ ops:ad:group:mapping:sync（独立的映射同步码）
  const permMappingSync = permGate.decide(PERMISSION_CODES.opsADGroup.mappingSync);

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
  const { paginationProps, setCurrent, setPageSize: _setPageSize, setTotal } = usePagination();

  const [selectedGroup, setSelectedGroup] = useState<ADGroup | null>(null);
  const [members, setMembers] = useState<ADUser[]>([]);
  const [memberTotal, setMemberTotal] = useState(0);
  const [memberLoading, setMemberLoading] = useState(false);
  const [drawerVisible, setDrawerVisible] = useState(false);

  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editSubmitting, setEditSubmitting] = useState(false);
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
          const defaultOU = flattened.find((ou) => ou.name === "本部部门分组");
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

  const fetchGroups = useCallback(
    async (groupName?: string, sortCol?: string, sortAsc?: boolean) => {
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
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps -- stable setters and pagination object
    [selectedConfig, selectedOUDN, paginationProps.current, paginationProps.pageSize]
  );

  useEffect(() => {
    if (selectedConfig) {
      fetchGroups(searchGroupName);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- paginationProps.current is intentional
  }, [
    selectedConfig,
    selectedOUDN,
    paginationProps.current,
    paginationProps.pageSize,
    searchGroupName,
    fetchGroups,
  ]);

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
  const handleTableChange = (
    _pagination: TablePaginationConfig,
    _filters: Record<string, unknown>,
    sorter: SorterResult<ADGroup> | SorterResult<ADGroup>[]
  ) => {
    // 用 local const 持有新值传 fetchGroups，规避 React 18 setState 异步时序
    const s = Array.isArray(sorter) ? sorter[0] : sorter;
    const newCol = s?.field != null ? String(s.field) : "";
    const newAsc = s?.order === "ascend";
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
      render: (scope: string) => (scope ? <Tag>{scope}</Tag> : "-"),
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
      render: (val: string) =>
        val ? new Date(val).toLocaleString() : <Tag color="warning">未同步</Tag>,
    },
    {
      title: "操作",
      key: "action",
      width: 180,
      render: (_: unknown, record: ADGroup) => (
        <Space>
          {/* 查看成员是读入口 → 不门控 */}
          <Button type="link" size="small" onClick={() => handleViewMembers(record)}>
            查看成员
          </Button>
          {/* Phase 86 W4：编辑组是写入口 → ops:ad:group:edit */}
          {permGroupEdit.visible && (
            <Button
              type="link"
              size="small"
              disabled={permGroupEdit.disabled}
              title={permGroupEdit.hint}
              onClick={() => handleEditGroup(record)}
            >
              编辑
            </Button>
          )}
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
      render: (_: unknown, record: ADUser) =>
        // Phase 86 W4：移除成员是写入口（改组成员关系）→ ops:ad:group:edit；
        // 触发器是裸 Button + confirmDelete → 门控点落在触发按钮本身
        // （条件渲染 + disabled + 提示），confirmDelete.ts 零改动。
        permGroupEdit.visible && (
          <Button
            type="link"
            size="small"
            danger
            icon={<MinusCircleOutlined />}
            disabled={permGroupEdit.disabled}
            title={permGroupEdit.hint}
            onClick={() => {
              confirmDelete({
                target: `AD 成员 ${record.userDn}`,
                onDelete: () => handleRemoveMember(record.userDn),
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
            {/* Phase 71 W5：迁 AppSelect——原 antd 单选默认禁搜，显式 showSearch={false}；
                configId 为 UUID 外键，原 allowClear=true 行为不变；
                AppSelect 泛型被 ComponentProps 擦除 → onChange 值为 unknown，收敛回业务类型 */}
            <AppSelect
              allowClear
              style={{ width: 200 }}
              value={selectedConfig}
              onChange={(value) => setSelectedConfig(value as string)}
              options={configs.map((c) => ({ label: c.configName, value: c.id }))}
              showSearch={false}
              onSearch={() => {}}
            />
          </Form.Item>
          <Form.Item label="所属OU">
            {/* Phase 71 W5：children 写法改写 options；optionFilterProp 由 children 改 label；
                showSearch 保持显式开启。AppSelect 泛型被 ComponentProps 擦除 → onChange 值为
                unknown，收敛回业务类型 */}
            <AppSelect
              style={{ width: 250 }}
              value={selectedOUDN}
              onChange={(value) => handleOUChange(value as string)}
              allowClear
              placeholder="选择OU（默认：本部部门分组）"
              showSearch
              optionFilterProp="label"
              onSearch={() => {}}
              options={ouList.map((ou) => ({ value: ou.dn, label: ou.name }))}
            />
          </Form.Item>
          <Form.Item name="groupName">
            <AppInput placeholder="组名称" allowClear />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" onClick={handleSearch}>
                查询
              </Button>
              <Button icon={<ReloadOutlined />} onClick={() => fetchGroups(searchGroupName)}>
                刷新
              </Button>
              {/* Phase 86 W4：同步组是写入口 → ops:ad:group:mapping:sync（独立码，不归 edit） */}
              {permMappingSync.visible && (
                <Button
                  type="primary"
                  icon={<SyncOutlined spin={syncLoading} />}
                  loading={syncLoading}
                  onClick={handleSyncGroups}
                  disabled={permMappingSync.disabled || !selectedConfig}
                  title={permMappingSync.hint}
                >
                  同步组
                </Button>
              )}
              {syncStatus && (
                <Tooltip
                  title={
                    `总组数: ${syncStatus.totalGroups} | ` +
                    `已同步: ${syncStatus.recentlySynced} | ` +
                    `未同步: ${syncStatus.neverSynced} | ` +
                    `成员关系: ${syncStatus.totalMemberRelations}`
                  }
                >
                  <Tag
                    color={syncStatus.neverSynced > 0 ? "orange" : "green"}
                    style={{ cursor: "pointer" }}
                  >
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
            <Descriptions.Item label="成员关系数">
              {syncStatus.totalMemberRelations}
            </Descriptions.Item>
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
        <Space orientation="vertical" style={{ width: "100%" }} size="medium">
          <Space>
            {/* Phase 86 W4：添加成员是写入口 → ops:ad:group:edit */}
            {permGroupEdit.visible && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                disabled={permGroupEdit.disabled}
                title={permGroupEdit.hint}
                onClick={handleAddMember}
              >
                添加成员
              </Button>
            )}
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
        onOk={async () => {
          setEditSubmitting(true);
          try {
            await handleUpdateGroup();
          } finally {
            setEditSubmitting(false);
          }
        }}
        cancelButtonProps={{ disabled: editSubmitting }}
        onCancel={() => setEditModalVisible(false)}
      >
        <Form layout="vertical">
          <Form.Item label="组名称" name="groupName" rules={[{ required: true }]}>
            <AppInput allowClear />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <AppTextArea rows={4} />
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
              render: (_: unknown, record: ADUser) =>
                // Phase 86 W4：弹窗内「添加」是真正的写触发点（调 addADGroupMember）
                // → ops:ad:group:edit。抽屉里的「添加成员」入口已门控，此处再门控一次
                // （纵深防御：弹窗内的确认点不能只依赖触发按钮）。
                permGroupEdit.visible && (
                  <Button
                    type="link"
                    icon={<PlusOutlined />}
                    disabled={permGroupEdit.disabled}
                    title={permGroupEdit.hint}
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
