import { useState, useEffect } from "react";
import { App, Tree, Card, Select, Spin, Tag, Space, Button, Table, Col, Row, Modal } from "antd";
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
  getOUDeptMapping,
  updateOUDeptMapping,
  getADGroupList,
  type ADOUNode,
  type ADUser,
  type OUDeptMappingResponse,
  type ADGroup,
} from "@/lib/adDomainApi";
import { useADConfigs } from "@/hooks/useADConfigs";
import { getDeptTree } from "@/lib/dutyApi";
import { createSorter } from "@/utils/tableHelpers";

import type { FC } from "react";

interface Department {
  id: string;
  deptName: string;
  children?: Department[];
}

const ADOUPage: FC = () => {
  const { message } = App.useApp();
  const { configs, selectedConfig, setSelectedConfig } = useADConfigs({
    enabledOnly: true,
    autoSelectFirst: true,
  });

  const [ouTree, setOUTree] = useState<ADOUNode[]>([]);
  const [selectedOU, setSelectedOU] = useState<string>("");
  const [users, setUsers] = useState<ADUser[]>([]);
  const [treeLoading, setTreeLoading] = useState(false);
  const [usersLoading, setUsersLoading] = useState(false);

  // 部门映射相关状态
  const [deptMapping, setDeptMapping] = useState<OUDeptMappingResponse | null>(null);
  const [mappingLoading, setMappingLoading] = useState(false);
  const [deptModalVisible, setDeptModalVisible] = useState(false);
  const [deptTree, setDeptTree] = useState<Department[]>([]);
  const [selectedDeptId, setSelectedDeptId] = useState<string>("");

  // 用户组相关状态
  const [groups, setGroups] = useState<ADGroup[]>([]);
  const [groupsLoading, setGroupsLoading] = useState(false);

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
    // eslint-disable-next-line react-hooks/exhaustive-deps -- fetchOUTree is render-defined; re-run on selectedConfig change
  }, [selectedConfig]);

  // 当选择OU时获取该OU下的用户和部门映射信息
  const handleOUSelect = async (selectedKeys: React.Key[]) => {
    if (selectedKeys.length > 0) {
      const ouDn = selectedKeys[0] as string;
      setSelectedOU(ouDn);
      fetchUsers(ouDn);
      fetchDeptMapping(ouDn);
      fetchGroups(ouDn);
    } else {
      setSelectedOU("");
      setUsers([]);
      setDeptMapping(null);
      setGroups([]);
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
      message.error("获取用户列表失败");
    } finally {
      setUsersLoading(false);
    }
  };

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

  const fetchGroups = async (ouDn: string) => {
    if (!selectedConfig) return;

    setGroupsLoading(true);
    try {
      const res = await getADGroupList({
        configId: selectedConfig,
        ouDn: ouDn,
        current: 1,
        pageSize: 100,
      });
      if (res.code === 0) {
        setGroups(res.data?.list ?? []);
      }
    } catch {
      message.error("获取用户组列表失败");
    } finally {
      setGroupsLoading(false);
    }
  };

  const fetchDeptTree = async () => {
    try {
      const res = await getDeptTree();
      if (res.code === 0) {
        setDeptTree(res.data || []);
      }
    } catch {
      message.error("获取部门树失败");
    }
  };

  const openDeptModal = () => {
    setDeptModalVisible(true);
    if (deptTree.length === 0) {
      fetchDeptTree();
    }
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

  const buildDeptTreeData = (depts: Department[]): DataNode[] => {
    return depts.map((dept) => ({
      title: dept.deptName,
      key: dept.id,
      children:
        dept.children && dept.children.length > 0 ? buildDeptTreeData(dept.children) : undefined,
    }));
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

  const groupColumns: ColumnsType<ADGroup> = [
    {
      title: "组名称",
      dataIndex: "groupName",
      key: "groupName",
      sorter: createSorter<ADGroup>("groupName", "string"),
    },
    {
      title: "描述",
      dataIndex: "description",
      key: "description",
      ellipsis: true,
      sorter: createSorter<ADGroup>("description", "string"),
    },
    {
      title: "成员数",
      dataIndex: "memberCount",
      key: "memberCount",
      width: 100,
      sorter: createSorter<ADGroup>("memberCount", "number"),
      render: (count: number) => <Tag color="blue">{count}</Tag>,
    },
    {
      title: "类型",
      dataIndex: "groupType",
      key: "groupType",
      width: 100,
      sorter: createSorter<ADGroup>("groupType", "number"),
      render: (type: number) => {
        if (type === 1) return <Tag color="blue">安全组</Tag>;
        if (type === 2) return <Tag color="default">分发组</Tag>;
        return "-";
      },
    },
  ];

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

  // 辅助函数：递归查找部门名称
  const findDeptName = (depts: Department[], id: string): string => {
    for (const dept of depts) {
      if (dept.id === id) {
        return dept.deptName;
      }
      if (dept.children && dept.children.length > 0) {
        const found = findDeptName(dept.children, id);
        if (found) return found;
      }
    }
    return "";
  };

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
            bodyStyle={{ maxHeight: "calc(100vh - 280px)", overflow: "auto" }}
          >
            <Space orientation="vertical" style={{ width: "100%" }} size="small">
              <Select
                style={{ width: "100%" }}
                placeholder="选择AD配置"
                value={selectedConfig}
                onChange={setSelectedConfig}
                options={configs.map((c) => ({ label: c.configName, value: c.id }))}
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
                  <div style={{ textAlign: "center", color: "#999", padding: "20px" }}>
                    {selectedConfig ? "暂无OU数据，请先同步" : "请选择AD配置"}
                  </div>
                )}
              </Spin>
            </Space>
          </Card>
        </Col>

        {/* 右侧：用户列表和部门映射 */}
        <Col span={18} style={{ height: "100%", overflow: "auto" }}>
          <Space direction="vertical" style={{ width: "100%" }} size="small">
            {/* 用户列表卡片 */}
            <Card
              title={
                <Space>
                  <TeamOutlined />
                  <span>
                    用户列表 -{" "}
                    {selectedOU
                      ? ouTree.find((n) => n.dn === selectedOU)?.name || selectedOU
                      : "全部"}
                  </span>
                </Space>
              }
              size="small"
              bodyStyle={{ maxHeight: "calc(100vh - 450px)", overflow: "auto" }}
            >
              <Table
                columns={userColumns}
                dataSource={users}
                loading={usersLoading}
                rowKey="id"
                pagination={{ pageSize: 50, size: "small" }}
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
                <Space direction="vertical" style={{ width: "100%" }}>
                  <div>
                    <span style={{ color: "#999" }}>关联部门：</span>
                    <Tag color="blue" style={{ marginLeft: 8 }}>
                      {deptMapping.mapping?.deptName}
                    </Tag>
                  </div>
                  <div>
                    <span style={{ color: "#999" }}>同步状态：</span>
                    <Tag
                      color={getSyncStatusColor(deptMapping.mapping?.syncStatus || "")}
                      style={{ marginLeft: 8 }}
                    >
                      {getSyncStatusText(deptMapping.mapping?.syncStatus || "")}
                    </Tag>
                  </div>
                </Space>
              ) : (
                <div style={{ color: "#999", textAlign: "center", padding: "20px" }}>
                  {selectedOU ? "该 OU 尚未关联部门" : "请选择 OU"}
                </div>
              )}
            </Card>

            {/* 用户组卡片 */}
            <Card
              title={
                <Space>
                  <TeamOutlined />
                  <span>关联用户组</span>
                </Space>
              }
              size="small"
              loading={groupsLoading}
            >
              {groups.length > 0 ? (
                <Table
                  columns={groupColumns}
                  dataSource={groups}
                  rowKey="id"
                  pagination={false}
                  size="small"
                />
              ) : (
                <div style={{ color: "#999", textAlign: "center", padding: "20px" }}>
                  {selectedOU ? "该 OU 下暂无用户组" : "请选择 OU"}
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
        onOk={handleUpdateMapping}
        onCancel={() => {
          setDeptModalVisible(false);
          setSelectedDeptId("");
        }}
        width={600}
        okText="确认关联"
        cancelText="取消"
      >
        <Spin spinning={deptTree.length === 0}>
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
        {selectedDeptId && (
          <div style={{ marginTop: 16 }}>
            已选择：<Tag color="blue">{findDeptName(deptTree, selectedDeptId)}</Tag>
          </div>
        )}
      </Modal>
    </div>
  );
};

export default ADOUPage;
