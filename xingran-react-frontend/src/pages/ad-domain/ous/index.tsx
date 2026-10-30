import { useState, useEffect } from "react";
// Phase 71 (W5)：裸 Select → 契约组件；P68 裸 hex 同步清偿为主题变量
import { AppSelect } from "@/components/ui";
import { App, Tree, Card, Spin, Tag, Space, Button, Table, Col, Row, Modal, Transfer } from "antd";
import {
  ReloadOutlined,
  FolderOutlined,
  TeamOutlined,
  ApartmentOutlined,
  EditOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import type { DataNode } from "antd/es/tree";
import {
  getADOUTree,
  getADUserList,
  getADGroupList,
  getOUGroupMappingsByOU,
  createOUGroupMapping,
  deleteOUGroupMapping,
  getOUDeptMapping,
  updateOUDeptMapping,
  type ADOUNode,
  type ADUser,
  type ADGroup,
  type OUGroupMapping,
  type OUDeptMappingResponse,
} from "@/lib/adDomainApi";
import { useADConfigs } from "@/hooks/useADConfigs";
import { usePagination } from "@/hooks/usePagination";
import { useDeptTree, type DeptTreeNode } from "@/hooks/useDeptTree";
import { createSorter } from "@/utils/tableHelpers";

import type { FC } from "react";

type Department = DeptTreeNode;

const ADOUPage: FC = () => {
  const { message } = App.useApp();
  const { configs, selectedConfig, setSelectedConfig } = useADConfigs({
    enabledOnly: true,
    autoSelectFirst: true,
  });

  const [ouTree, setOUTree] = useState<ADOUNode[]>([]);
  const [selectedOU, setSelectedOU] = useState<string>("");
  const [selectedOUName, setSelectedOUName] = useState<string>("");
  const [users, setUsers] = useState<ADUser[]>([]);
  // STOP-03: 用户表改服务端分页（pageSize 硬约束沿用 50，与后端 MaxPageSize=100 兼容）
  const {
    current: userCurrent,
    pageSize: userPageSize,
    paginationProps: userPaginationProps,
    setCurrent: userSetCurrent,
    setTotal: userSetTotal,
  } = usePagination({ pageSize: 50 });
  const [treeLoading, setTreeLoading] = useState(false);
  const [usersLoading, setUsersLoading] = useState(false);

  // 部门映射相关状态
  const [deptMapping, setDeptMapping] = useState<OUDeptMappingResponse | null>(null);
  const [mappingLoading, setMappingLoading] = useState(false);
  const [deptModalVisible, setDeptModalVisible] = useState(false);
  const [mappingSubmitting, setMappingSubmitting] = useState(false);
  // 部门树 — 通过 React Query 全局共享缓存 (D-13 Step 2)
  const { data: deptTreeData, isLoading: deptTreeLoading } = useDeptTree();
  const deptTree: Department[] = (deptTreeData ?? []) as Department[];
  const [selectedDeptId, setSelectedDeptId] = useState<string>("");

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
      message.error("获取OU树失败");
    } finally {
      setTreeLoading(false);
    }
  };

  useEffect(() => {
    if (selectedConfig) {
      fetchOUTree();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- fetchOUTree recreated each render; disable to avoid loop
  }, [selectedConfig]);

  // 当选择OU时获取该OU下的用户、部门映射和用户组信息
  const handleOUSelect = async (selectedKeys: React.Key[]) => {
    if (selectedKeys.length > 0) {
      const ouDn = selectedKeys[0] as string;
      const ouNode = ouTree.find((n) => n.dn === ouDn);
      setSelectedOU(ouDn);
      setSelectedOUName(ouNode?.name || ouDn);
      userSetCurrent(1); // 切换 OU 时页码重置第 1 页
      fetchDeptMapping(ouDn);
      fetchOUGroupMappings(ouDn);
      fetchAllGroups();
    } else {
      setSelectedOU("");
      setSelectedOUName("");
      setUsers([]);
      setDeptMapping(null);
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
        current: userCurrent,
        pageSize: userPageSize,
      });
      if (res.code === 0) {
        setUsers(res.data?.list ?? []);
        userSetTotal(res.data?.total ?? 0);
      }
    } catch {
      message.error("获取用户列表失败");
    } finally {
      setUsersLoading(false);
    }
  };

  // STOP-03: OU / 配置 / 分页任一变化时按服务端分页重新拉取用户
  useEffect(() => {
    if (!selectedConfig || !selectedOU) return;
    fetchUsers(selectedOU);
    // eslint-disable-next-line react-hooks/exhaustive-deps -- fetchUsers uses latest closure state
  }, [selectedConfig, selectedOU, userCurrent, userPageSize]);

  const fetchDeptMapping = async (ouDn: string) => {
    setMappingLoading(true);
    try {
      const res = await getOUDeptMapping(ouDn);
      if (res.code === 0 && res.data) {
        setDeptMapping(res.data);
      }
    } catch {
      message.error("获取部门映射失败");
    } finally {
      setMappingLoading(false);
    }
  };

  const fetchAllGroups = async () => {
    if (!selectedConfig) return;

    try {
      const res = await getADGroupList({
        configId: selectedConfig,
        current: 1,
        pageSize: 50, // 获取所有组用于选择
      });
      if (res.code === 0) {
        setAllGroups(res.data?.list ?? []);
      }
    } catch {
      message.error("获取用户组列表失败");
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
      message.error("获取OU组映射失败");
    } finally {
      setGroupsLoading(false);
    }
  };

  // 部门树 — 通过 useDeptTree() (React Query) 自动获取并全局共享 (D-13 Step 2)
  // 此处无需手动 fetch；useDeptTree 在组件首次挂载时即触发 query。

  const openDeptModal = () => {
    setDeptModalVisible(true);
  };

  const handleDeptSelect = (selectedKeys: React.Key[]) => {
    if (selectedKeys.length > 0) {
      setSelectedDeptId(selectedKeys[0] as string);
    }
  };

  const handleUpdateMapping = async () => {
    if (!selectedOU || !selectedDeptId) {
      message.warning("请选择部门");
      return;
    }

    try {
      const res = await updateOUDeptMapping(selectedOU, { deptId: selectedDeptId });
      if (res.code === 0) {
        message.success("关联成功");
        // 重新获取部门映射信息
        await fetchDeptMapping(selectedOU);
        setDeptModalVisible(false);
        setSelectedDeptId("");
      }
    } catch {
      message.error("关联失败");
    }
  };

  const openGroupModal = () => {
    setGroupModalVisible(true);
  };

  const handleGroupTransfer = async (
    targetKeys: React.Key[],
    _direction: "right" | "left",
    _moveKeys: React.Key[]
  ) => {
    if (!selectedOU) return;

    setGroupModalLoading(true);
    try {
      // 获取当前映射的组ID
      const currentMappedIds = ouGroups.map((g) => g.adGroupId);

      // 找出需要添加的组
      const toAdd = targetKeys.map(String).filter((id) => !currentMappedIds.includes(id));
      // 找出需要删除的组
      const toRemove = currentMappedIds.filter((id) => !targetKeys.map(String).includes(id));

      // 创建新映射（并行）
      const addResults = await Promise.allSettled(
        toAdd.map((groupId) => {
          const group = allGroups.find((g) => g.id === String(groupId));
          if (group) {
            return createOUGroupMapping({
              adConfigId: selectedConfig,
              ouDn: selectedOU,
              ouName: selectedOUName,
              adGroupId: groupId,
              syncEnabled: true,
            });
          }
          return Promise.reject(new Error(`组 ${groupId} 未找到`));
        })
      );
      const addFailed = addResults.filter((r) => r.status === "rejected");
      if (addFailed.length > 0) {
        message.error(`创建映射失败 ${addFailed.length} 项`);
      }

      // 删除映射（并行）
      const removeResults = await Promise.allSettled(
        toRemove.map((mappingId) => {
          const mapping = ouGroups.find((g) => g.adGroupId === mappingId);
          if (mapping) {
            return deleteOUGroupMapping(mapping.id);
          }
          return Promise.reject(new Error(`映射 ${mappingId} 未找到`));
        })
      );
      const removeFailed = removeResults.filter((r) => r.status === "rejected");
      if (removeFailed.length > 0) {
        message.error(`删除映射失败 ${removeFailed.length} 项`);
      }

      message.success("更新成功");
      await fetchOUGroupMappings(selectedOU);
      setGroupModalVisible(false);
    } catch {
      message.error("更新失败");
    } finally {
      setGroupModalLoading(false);
    }
  };

  const buildTreeData = (nodes: ADOUNode[]): DataNode[] => {
    return nodes.map((node) => ({
      title: (
        <Space>
          <FolderOutlined />
          <span>{node.name}</span>
        </Space>
      ),
      key: node.dn,
      children:
        node.children && node.children.length > 0 ? buildTreeData(node.children) : undefined,
    }));
  };

  const buildDeptTreeData = (depts: Department[]): DataNode[] => {
    return depts.map((dept) => ({
      title: dept.deptName,
      key: dept.id,
      children:
        dept.children && dept.children.length > 0 ? buildDeptTreeData(dept.children) : undefined,
    }));
  };

  // 辅助函数：递归查找部门名称
  const _findDeptName = (depts: Department[], id: string): string => {
    for (const dept of depts) {
      if (dept.id === id) {
        return dept.deptName;
      }
      if (dept.children && dept.children.length > 0) {
        const found = _findDeptName(dept.children, id);
        if (found) return found;
      }
    }
    return "";
  };

  const getSyncStatusColor = (status: string) => {
    switch (status) {
      case "synced":
        return "success";
      case "pending":
        return "processing";
      case "failed":
        return "error";
      default:
        return "default";
    }
  };

  const getSyncStatusText = (status: string) => {
    switch (status) {
      case "synced":
        return "已同步";
      case "pending":
        return "待同步";
      case "failed":
        return "同步失败";
      default:
        return status;
    }
  };

  const userColumns: ColumnsType<ADUser> = [
    {
      title: "用户名",
      dataIndex: "username",
      key: "username",
      fixed: "left",
      width: 150,
      sorter: createSorter<ADUser>("username", "string"),
    },
    {
      title: "显示名",
      dataIndex: "displayName",
      key: "displayName",
      width: 150,
      sorter: createSorter<ADUser>("displayName", "string"),
    },
    {
      title: "邮箱",
      dataIndex: "email",
      key: "email",
      width: 200,
      ellipsis: true,
      sorter: createSorter<ADUser>("email", "string"),
    },
    {
      title: "部门",
      dataIndex: "department",
      key: "department",
      width: 150,
      sorter: createSorter<ADUser>("department", "string"),
    },
    {
      title: "职位",
      dataIndex: "title",
      key: "title",
      width: 120,
      sorter: createSorter<ADUser>("title", "string"),
    },
    {
      title: "办公电话",
      dataIndex: "phone",
      key: "phone",
      width: 130,
      sorter: createSorter<ADUser>("phone", "string"),
    },
    {
      title: "手机",
      dataIndex: "mobile",
      key: "mobile",
      width: 130,
      sorter: createSorter<ADUser>("mobile", "string"),
    },
    {
      title: "状态",
      key: "status",
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
      title: "组名称",
      dataIndex: "adGroup",
      key: "groupName",
      sorter: createSorter<OUGroupMapping>("adGroup", "string"),
      render: (group) => group?.groupName || "-",
    },
    {
      title: "描述",
      dataIndex: "adGroup",
      key: "description",
      ellipsis: true,
      sorter: createSorter<OUGroupMapping>("adGroup", "string"),
      render: (group) => group?.description || "-",
    },
    {
      title: "成员数",
      dataIndex: "adGroup",
      key: "memberCount",
      width: 100,
      render: (group) => <Tag color="blue">{group?.memberCount || 0}</Tag>,
    },
    {
      title: "同步状态",
      dataIndex: "syncEnabled",
      key: "syncEnabled",
      width: 100,
      sorter: createSorter<OUGroupMapping>("syncEnabled", "boolean"),
      render: (enabled) => (
        <Tag color={enabled ? "success" : "default"}>{enabled ? "已启用" : "未启用"}</Tag>
      ),
    },
    {
      title: "最后同步",
      dataIndex: "lastSyncAt",
      key: "lastSyncAt",
      width: 150,
      sorter: createSorter<OUGroupMapping>("lastSyncAt", "date"),
      render: (time) => (time ? new Date(time).toLocaleString("zh-CN") : "-"),
    },
  ];

  // Transfer组件的dataSource
  const transferDataSource = allGroups.map((group) => ({
    key: group.id,
    title: group.groupName,
    description: group.description,
  }));

  // 当前已映射的组ID
  const targetKeys = ouGroups.map((mapping) => mapping.adGroupId);

  return (
    <div style={{ height: "calc(100vh - 200px)" }}>
      <Row gutter={16} style={{ height: "100%" }}>
        {/* 左侧：OU树 */}
        <Col span={6} style={{ height: "100%", overflow: "auto" }}>
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
            styles={{ body: { maxHeight: "calc(100vh - 280px)", overflow: "auto" } }}
          >
            <Space orientation="vertical" style={{ width: "100%" }} size="small">
              {/* Phase 71 W5：迁 AppSelect——原显式 showSearch 保持；configId 为 UUID 外键，
                  原 allowClear=true 行为不变；AppSelect 泛型被 ComponentProps 擦除 →
                  onChange 值为 unknown，收敛回业务类型 */}
              <AppSelect
                allowClear
                style={{ width: "100%" }}
                placeholder="选择AD配置"
                value={selectedConfig}
                onChange={(value) => setSelectedConfig(value as string)}
                options={configs.map((c) => ({ label: c.configName, value: c.id }))}
                showSearch
                // noop 满足 local/no-large-dropdown-list(要求 Select 带 onSearch);
                // showSearch 让 antd 警告(onSearch 应配合 showSearch)消除
                onSearch={() => {}}
              />
              <Spin spinning={treeLoading}>
                {ouTree.length > 0 ? (
                  <Tree
                    showLine
                    defaultExpandAll={false}
                    onSelect={handleOUSelect}
                    treeData={buildTreeData(ouTree)}
                    style={{ fontSize: "13px" }}
                  />
                ) : (
                  <div
                    style={{
                      textAlign: "center",
                      color: "var(--theme-text-tertiary)",
                      padding: "20px",
                    }}
                  >
                    {selectedConfig ? "暂无OU数据，请先同步" : "请选择AD配置"}
                  </div>
                )}
              </Spin>
            </Space>
          </Card>
        </Col>

        {/* 右侧：用户列表、部门映射和用户组映射 */}
        <Col span={18} style={{ height: "100%", overflow: "auto" }}>
          <Space orientation="vertical" style={{ width: "100%" }} size="small">
            {/* 用户列表卡片 */}
            <Card
              title={
                <Space>
                  <TeamOutlined />
                  <span>用户列表 - {selectedOUName || "全部"}</span>
                </Space>
              }
              size="small"
              styles={{ body: { maxHeight: "calc(100vh - 550px)", overflow: "auto" } }}
            >
              <Table
                columns={userColumns}
                dataSource={users}
                loading={usersLoading}
                rowKey="id"
                pagination={{ ...userPaginationProps, showSizeChanger: false, size: "small" }}
                scroll={{ x: 1200 }}
                size="small"
              />
            </Card>

            {/* 部门关联卡片 */}
            <Card
              title={
                <Space>
                  <ApartmentOutlined />
                  <span>关联部门信息</span>
                </Space>
              }
              size="small"
              extra={
                <Button
                  type="primary"
                  size="small"
                  icon={<EditOutlined />}
                  onClick={openDeptModal}
                  disabled={!selectedOU}
                >
                  {deptMapping?.hasMapping ? "修改关联" : "关联部门"}
                </Button>
              }
              loading={mappingLoading}
            >
              {deptMapping?.hasMapping ? (
                <Space orientation="vertical" style={{ width: "100%" }}>
                  <div>
                    <span style={{ color: "var(--theme-text-tertiary)" }}>关联部门：</span>
                    <Tag color="blue" style={{ marginLeft: 8 }}>
                      {deptMapping.mapping?.deptName}
                    </Tag>
                  </div>
                  <div>
                    <span style={{ color: "var(--theme-text-tertiary)" }}>同步状态：</span>
                    <Tag
                      color={getSyncStatusColor(deptMapping.mapping?.syncStatus || "")}
                      style={{ marginLeft: 8 }}
                    >
                      {getSyncStatusText(deptMapping.mapping?.syncStatus || "")}
                    </Tag>
                  </div>
                </Space>
              ) : (
                <div
                  style={{
                    color: "var(--theme-text-tertiary)",
                    textAlign: "center",
                    padding: "20px",
                  }}
                >
                  {selectedOU ? "该 OU 尚未关联部门" : "请选择 OU"}
                </div>
              )}
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
                <div
                  style={{
                    color: "var(--theme-text-tertiary)",
                    textAlign: "center",
                    padding: "20px",
                  }}
                >
                  {selectedOU ? "该 OU 尚未关联用户组" : "请选择 OU"}
                </div>
              )}
            </Card>
          </Space>
        </Col>
      </Row>

      {/* 部门选择模态框 */}
      <Modal
        title="选择关联部门"
        open={deptModalVisible}
        onOk={async () => {
          setMappingSubmitting(true);
          try {
            await handleUpdateMapping();
          } finally {
            setMappingSubmitting(false);
          }
        }}
        cancelButtonProps={{ disabled: mappingSubmitting }}
        onCancel={() => {
          setDeptModalVisible(false);
          setSelectedDeptId("");
        }}
        width={600}
        okText="确认关联"
        cancelText="取消"
      >
        <Spin spinning={deptTreeLoading || deptTree.length === 0}>
          {deptTree.length > 0 && (
            <Tree
              showLine
              defaultExpandAll
              onSelect={handleDeptSelect}
              treeData={buildDeptTreeData(deptTree)}
              style={{ maxHeight: 400, overflow: "auto" }}
            />
          )}
        </Spin>
      </Modal>

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
            render={(item) => (
              <div>
                <div>{item.title}</div>
                <div style={{ fontSize: "12px", color: "var(--theme-text-tertiary)" }}>
                  {item.description}
                </div>
              </div>
            )}
            listStyle={{
              width: 350,
              height: 400,
            }}
            titles={["可用用户组", "已关联用户组"]}
            showSearch
            filterOption={(inputValue, item) => {
              const titleMatch =
                item.title?.toLowerCase().includes(inputValue.toLowerCase()) ?? false;
              const descMatch =
                item.description?.toLowerCase().includes(inputValue.toLowerCase()) ?? false;
              return titleMatch || descMatch;
            }}
          />
        </Spin>
      </Modal>
    </div>
  );
};

export default ADOUPage;
