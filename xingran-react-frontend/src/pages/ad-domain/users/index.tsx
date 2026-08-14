import { useState, useEffect, useMemo } from "react";
import {
  Card,
  Table,
  Select,
  TreeSelect,
  Input,
  Space,
  Button,
  Tag,
  Form,
  Modal,
  message,
  Tooltip,
} from "antd";
import {
  ReloadOutlined,
  CheckCircleOutlined,
  StopOutlined,
  SwapOutlined,
  EditOutlined,
  SearchOutlined,
  FolderOutlined,
  SyncOutlined,
  CheckSquareOutlined,
  BorderOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import {
  getADUserList,
  getADUserIds,
  updateADUser,
  moveADUser,
  enableADUser,
  disableADUser,
  getADOUTree,
  batchSyncADUsersDirect,
  type ADUser,
  type ADOUNode,
} from "@/lib/adDomainApi";
import type { UnknownError } from "@/types/common";
import ActionButtons from "@/components/shared/ActionButtons";
import { useADConfigs } from "@/hooks/useADConfigs";

import type { FC } from "react";
import { usePagination } from "@/hooks/usePagination";

const ADUserPage: FC = () => {
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  // 使用共享的 AD 配置 Hook
  const { configs, selectedConfig, setSelectedConfig } = useADConfigs({
    enabledOnly: true,
    autoSelectFirst: true,
  });

  const [users, setUsers] = useState<ADUser[]>([]);
  const [loading, setLoading] = useState(false);

  // 排序状态
  const [orderByColumn, setOrderByColumn] = useState<string>("");
  const [isAsc, setIsAsc] = useState<boolean>(true);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  const [ouTree, setOUTree] = useState<ADOUNode[]>([]);
  const [ouSelectVisible, setOUSelectVisible] = useState(false);

  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingUser, setEditingUser] = useState<ADUser | null>(null);

  const [moveModalVisible, setMoveModalVisible] = useState(false);
  const [movingUser, setMovingUser] = useState<ADUser | null>(null);

  // Batch sync state
  const [selectedUsers, setSelectedUsers] = useState<ADUser[]>([]);
  const [selectedUserIds, setSelectedUserIds] = useState<string[]>([]);
  const [selectAllMode, setSelectAllMode] = useState(false);
  const [selectAllTotal, setSelectAllTotal] = useState(0);
  const [batchSyncLoading, setBatchSyncLoading] = useState(false);
  const [loadingAllUserIds, setLoadingAllUserIds] = useState(false);

  const fetchOUTree = async () => {
    if (!selectedConfig) return;
    try {
      const res = await getADOUTree(selectedConfig);
      if (res.code === 0 && res.data) {
        // 过滤掉不需要的OU：Computer, None, Servers, mgtgrp
        const excludedOUNames = ["Computer", "None", "Servers", "mgtgrp"];
        const filterOUTree = (nodes: ADOUNode[]): ADOUNode[] => {
          return nodes
            .filter(node => !excludedOUNames.includes(node.name))
            .map(node => ({
              ...node,
              children: node.children ? filterOUTree(node.children) : undefined,
            }));
        };
        setOUTree(filterOUTree(res.data));
      }
    } catch (error) {
      console.error("获取OU树失败", error);
    }
  };

  // 常用OU快捷列表（可配置）
  const commonOUs = useMemo(() => {
    if (!ouTree.length) return [];
    const findOUByName = (name: string, nodes: ADOUNode[] = ouTree): ADOUNode | null => {
      for (const node of nodes) {
        if (node.name === name) return node;
        if (node.children) {
          const found = findOUByName(name, node.children);
          if (found) return found;
        }
      }
      return null;
    };

    const commonNames = ["湖北分公司", "失效终端", "本部部门分组"];
    return commonNames
      .map(name => findOUByName(name))
      .filter(Boolean) as ADOUNode[];
  }, [ouTree]);

  useEffect(() => {
    if (selectedConfig) {
      fetchOUTree();
      fetchUsers();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- fetchOUTree/fetchUsers recreated each render; disable to avoid loop
  }, [selectedConfig, paginationProps.current, paginationProps.pageSize]);

  const fetchUsers = async (sortCol?: string, sortAsc?: boolean) => {
    if (!selectedConfig) return;

    setLoading(true);
    try {
      const values = form.getFieldsValue();
      // 接受 sortCol/sortAsc 参数（handleTableChange 同步传新值，规避 React 18 setState 异步时序
      // 导致 fetchUsers 读到旧 state——config/notice 同类坑加深一层）
      const orderCol = sortCol ?? orderByColumn;
      const asc = sortAsc ?? isAsc;
      const res = await getADUserList({
        configId: selectedConfig,
        ouDn: values.ouDn,
        username: values.username,
        isEnabled: values.isEnabled,
        current: paginationProps.current,
        pageSize: paginationProps.pageSize,
        orderByColumn: orderCol,
        isAsc: asc,
      });
      if (res.code === 0) {
        setUsers(res.data?.list ?? []);
        setTotal(res.data?.total ?? 0);
      }
    } catch {
      message.error("获取用户列表失败");
    } finally {
      setLoading(false);
    }
  };

  const handleSearch = () => {
    setCurrent(1);
    fetchUsers();
  };

  // 表格排序双向关联
  const handleTableChange = (pagination: any, _filters: any, sorter: any) => {
    // 用 local const 持有新值传 fetchUsers，规避 React 18 setState 异步时序
    // （setState 后立即读 state 仍为旧值——加 deep 一层避坑）
    const newCol = sorter?.field || "";
    const newAsc = sorter?.order === "ascend";
    setOrderByColumn(newCol);
    setIsAsc(newAsc);
    fetchUsers(newCol, newAsc);
  };

  const handleEdit = (user: ADUser) => {
    setEditingUser(user);
    editForm.setFieldsValue({
      displayName: user.displayName,
      email: user.email,
      phone: user.phone,
      mobile: user.mobile,
      title: user.title,
      department: user.department,
      description: user.description,
    });
    setEditModalVisible(true);
  };

  const handleUpdateUser = async () => {
    if (!editingUser || !selectedConfig) return;

    try {
      const values = await editForm.validateFields();
      await updateADUser(editingUser.id, selectedConfig, values);
      message.success("更新成功");
      setEditModalVisible(false);
      fetchUsers();
    } catch (error: unknown) {
      const err = error as UnknownError;
      message.error(err.message || "更新失败");
    }
  };

  const handleMove = (user: ADUser) => {
    setMovingUser(user);
    setMoveModalVisible(true);
  };

  const handleMoveConfirm = async (ouDn: string) => {
    if (!movingUser || !selectedConfig) return;

    try {
      await moveADUser(movingUser.id, selectedConfig, ouDn);
      message.success("移动成功");
      setMoveModalVisible(false);
      fetchUsers();
    } catch (error: unknown) {
      const err = error as UnknownError;
      message.error(err.message || "移动失败");
    }
  };

  const handleEnable = async (user: ADUser) => {
    if (!selectedConfig) return;

    try {
      if (user.isEnabled) {
        await disableADUser(user.id, selectedConfig);
        message.success("禁用成功");
      } else {
        await enableADUser(user.id, selectedConfig);
        message.success("启用成功");
      }
      fetchUsers();
    } catch (error: unknown) {
      const err = error as UnknownError;
      message.error(err.message || "操作失败");
    }
  };

  // 全选所有功能
  const handleSelectAll = async () => {
    if (!selectedConfig) return;

    // 如果已经是全选模式，则取消全选
    if (selectAllMode) {
      setSelectAllMode(false);
      setSelectedUserIds([]);
      setSelectedUsers([]);
      setSelectAllTotal(0);
      return;
    }

    // 获取当前筛选条件
    const values = form.getFieldsValue();

    setLoadingAllUserIds(true);
    try {
      const res = await getADUserIds({
        configId: selectedConfig,
        ouDn: values.ouDn,
        username: values.username,
        isEnabled: values.isEnabled,
      });

      if (res.code === 0 && res.data) {
        const allUserIds = res.data;
        setSelectAllMode(true);
        setSelectedUserIds(allUserIds);
        setSelectAllTotal(allUserIds.length);
        message.success(`已选择所有 ${allUserIds.length} 个用户`);
      }
    } catch (error) {
      message.error("获取所有用户失败");
    } finally {
      setLoadingAllUserIds(false);
    }
  };

  const handleBatchSync = async () => {
    if (!selectedConfig) return;

    // 确定要同步的用户数量
    const syncCount = selectAllMode ? selectAllTotal : selectedUsers.length;
    if (syncCount === 0) return;

    Modal.confirm({
      title: `确定同步选中的 ${syncCount} 个用户到系统用户表？`,
      content: selectAllMode
        ? `当前选择的是"全选所有"模式，将同步所有符合筛选条件的 ${selectAllTotal} 个用户`
        : "同步后用户可以使用AD账号登录系统",
      okText: "确定",
      cancelText: "取消",
      onOk: async () => {
        setBatchSyncLoading(true);
        try {
          let userDns: string[] = [];

          if (selectAllMode) {
            // 全选模式：使用所有用户ID获取用户信息
            const res = await getADUserList({
              configId: selectedConfig,
              ouDn: form.getFieldValue("ouDn"),
              username: form.getFieldValue("username"),
              isEnabled: form.getFieldValue("isEnabled"),
              current: 1,
              pageSize: selectAllTotal,
            });

            if (res.code === 0 && res.data) {
              userDns = res.data.list.map(u => u.userDn);
            }
          } else {
            // 普通模式：使用当前选中的用户
            userDns = selectedUsers.map(u => u.userDn);
          }

          const res = await batchSyncADUsersDirect({
            configId: selectedConfig,
            userDns,
          });

          if (res.code === 0 && res.data) {
            const data = res.data;
            const { success, failed, skipped } = data;
            message.success(`同步完成: 成功${success}个, 失败${failed}个, 跳过${skipped}个`);

            // 显示错误详情
            if (data.errors && data.errors.length > 0) {
              Modal.warning({
                title: "部分用户同步失败",
                width: 600,
                content: (
                  <div style={{ maxHeight: 400, overflow: "auto" }}>
                    {data.errors.map((err, i) => (
                      <div key={i} style={{ marginBottom: 4 }}>
                        <strong>{err.username}:</strong> {err.error}
                      </div>
                    ))}
                  </div>
                ),
              });
            }

            // 清空选择
            setSelectedUsers([]);
            setSelectedUserIds([]);
            setSelectAllMode(false);
            setSelectAllTotal(0);
            // 刷新用户列表
            fetchUsers();
          }
        } catch (error) {
          message.error("批量同步失败");
        } finally {
          setBatchSyncLoading(false);
        }
      },
    });
  };

  // 构建TreeSelect所需的数据结构
  const buildTreeSelectData = (nodes: ADOUNode[]): Array<{ value: string; title: string; children?: Array<{ value: string; title: string }> }> => {
    return nodes.map(node => ({
      value: node.dn,
      title: node.name,
      children: node.children && node.children.length > 0 ? buildTreeSelectData(node.children) : undefined,
    }));
  };

  // eslint-disable-next-line react-hooks/exhaustive-deps -- buildTreeSelectData recreated each render; disable to avoid loop
  const treeSelectData = useMemo(() => buildTreeSelectData(ouTree), [ouTree]);

  const handleQuickOUSelect = (ouDn: string) => {
    form.setFieldValue("ouDn", ouDn);
    handleSearch();
  };

  const columns: ColumnsType<ADUser> = [
    {
      title: "用户名",
      dataIndex: "username",
      key: "username",
      fixed: "left",
      width: 150,
      sorter: true,
    },
    {
      title: "显示名",
      dataIndex: "displayName",
      key: "displayName",
      width: 150,
      sorter: true,
    },
    {
      title: "邮箱",
      dataIndex: "email",
      key: "email",
      width: 200,
      ellipsis: true,
      sorter: true,
    },
    {
      title: "部门",
      dataIndex: "department",
      key: "department",
      width: 150,
      sorter: true,
    },
    {
      title: "OU",
      dataIndex: "ouDn",
      key: "ouDn",
      width: 300,
      ellipsis: true,
      render: (ouDn: string, record: ADUser) => {
        if (!ouDn) return "-";
        // 获取当前配置的BaseDN
        const currentConfig = configs.find(c => c.id === selectedConfig);
        if (!currentConfig) return ouDn;

        // 去除BaseDN，获取所有OU部分
        const baseDN = currentConfig.baseDn;
        let dnToProcess = ouDn;

        // 如果DN以BaseDN结尾，去除BaseDN部分
        if (ouDn.endsWith(baseDN)) {
          dnToProcess = ouDn.substring(0, ouDn.length - baseDN.length - 1);
        }

        // 提取所有OU并格式化显示
        const parts = dnToProcess.split(",");
        const ous: string[] = [];
        for (const part of parts) {
          const trimmed = part.trim();
          if (trimmed.startsWith("OU=")) {
            ous.push(trimmed.replace("OU=", ""));
          }
        }

        // 用斜杠显示多个OU层级
        return ous.length > 0 ? ous.join(" / ") : "-";
      },
    },
    {
      title: "职位",
      dataIndex: "title",
      key: "title",
      width: 120,
    },
    {
      title: "办公电话",
      dataIndex: "phone",
      key: "phone",
      width: 130,
    },
    {
      title: "手机",
      dataIndex: "mobile",
      key: "mobile",
      width: 130,
    },
    {
      title: "状态",
      key: "status",
      width: 200,
      render: (_: unknown, user: ADUser) => (
        <Space size="small">
          <Tag color={user.isEnabled ? "success" : "default"}>
            {user.isEnabled ? "启用" : "禁用"}
          </Tag>
          {user.isLocked && <Tag color="error">锁定</Tag>}
          {user.passwordExpired && <Tag color="warning">密码过期</Tag>}
        </Space>
      ),
    },
    {
      title: "操作",
      key: "action",
      fixed: "right",
      width: 100,
      render: (_: unknown, user: ADUser) => {
        const actions = [
          {
            key: "toggle-status",
            label: user.isEnabled ? "禁用" : "启用",
            icon: user.isEnabled ? <StopOutlined /> : <CheckCircleOutlined />,
            onClick: () => handleEnable(user),
          },
          {
            key: "move",
            label: "移动",
            icon: <SwapOutlined />,
            onClick: () => handleMove(user),
          },
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => handleEdit(user),
          },
        ];

        return <ActionButtons actions={actions} />;
      },
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
          <Form.Item name="ouDn" label="所属OU">
            <Space.Compact style={{ width: "100%" }}>
              <TreeSelect
                placeholder="请选择OU"
                allowClear
                showSearch
                treeDefaultExpandAll={false}
                treeIcon={<FolderOutlined />}
                treeNodeFilterProp="title"
                style={{ width: 300 }}
                styles={{ popup: { root: { maxHeight: 400, overflow: "auto" } } }}
                treeData={treeSelectData}
                onChange={(value) => {
                  form.setFieldValue("ouDn", value);
                }}
              />
              {commonOUs.length > 0 && (
                <Tooltip title="快捷选择常用OU">
                  <Button
                    icon={<SearchOutlined />}
                    onClick={() => setOUSelectVisible(true)}
                  >
                    快捷
                  </Button>
                </Tooltip>
              )}
            </Space.Compact>
          </Form.Item>
          <Form.Item name="username">
            <Input placeholder="用户名" allowClear style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="isEnabled">
            <Select placeholder="状态" allowClear style={{ width: 120 }} onSearch={() => {}}>
              <Select.Option value={true}>启用</Select.Option>
              <Select.Option value={false}>禁用</Select.Option>
            </Select>
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" onClick={handleSearch}>
                查询
              </Button>
              <Button icon={<ReloadOutlined />} onClick={() => fetchUsers()}>
                刷新
              </Button>
              <Button
                icon={selectAllMode ? <CheckSquareOutlined /> : <BorderOutlined />}
                onClick={handleSelectAll}
                loading={loadingAllUserIds}
                type={selectAllMode ? "primary" : "default"}
              >
                {selectAllMode ? "取消全选" : "全选所有"}
                {selectAllMode && ` (${selectAllTotal})`}
              </Button>
              <Button
                type="primary"
                icon={<SyncOutlined />}
                disabled={selectedUsers.length === 0 && !selectAllMode}
                onClick={handleBatchSync}
                loading={batchSyncLoading}
              >
                批量同步 ({selectAllMode ? selectAllTotal : selectedUsers.length})
              </Button>
            </Space>
          </Form.Item>
        </Form>

        {/* 用户列表 */}
        <Table
          columns={columns}
          dataSource={users}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1500 }}
          pagination={paginationProps}
          onChange={handleTableChange}
          rowSelection={
            selectAllMode
              ? undefined // 全选模式下禁用表格行选择
              : {
                  selectedRowKeys: selectedUsers.map(u => u.id),
                  onChange: (selectedKeys, selectedRows) => {
                    setSelectedUsers(selectedRows);
                  },
                }
          }
        />
      </Space>

      {/* 编辑用户弹窗 */}
      <Modal
        title="编辑用户"
        open={editModalVisible}
        onOk={handleUpdateUser}
        onCancel={() => setEditModalVisible(false)}
        width={600}
      >
        <Form form={editForm} layout="vertical">
          <Form.Item label="显示名" name="displayName">
            <Input />
          </Form.Item>
          <Form.Item label="邮箱" name="email">
            <Input />
          </Form.Item>
          <Form.Item label="办公电话" name="phone">
            <Input />
          </Form.Item>
          <Form.Item label="手机" name="mobile">
            <Input />
          </Form.Item>
          <Form.Item label="职位" name="title">
            <Input />
          </Form.Item>
          <Form.Item label="部门" name="department">
            <Input />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>

      {/* 移动用户弹窗 */}
      <Modal
        title={`移动用户 - ${movingUser?.username}`}
        open={moveModalVisible}
        onCancel={() => setMoveModalVisible(false)}
        footer={null}
        width={500}
      >
        <Space orientation="vertical" style={{ width: "100%" }}>
          <div>选择目标OU：</div>
          <TreeSelect
            style={{ width: "100%" }}
            placeholder="请选择目标OU"
            allowClear
            showSearch
            treeDefaultExpandAll={false}
            treeIcon={<FolderOutlined />}
            treeNodeFilterProp="title"
            styles={{ popup: { root: { maxHeight: 400, overflow: "auto" } } }}
            treeData={treeSelectData}
            onChange={(value) => {
              handleMoveConfirm(value);
            }}
          />
        </Space>
      </Modal>

      {/* OU快捷选择弹窗 */}
      <Modal
        title="快捷选择常用OU"
        open={ouSelectVisible}
        onCancel={() => setOUSelectVisible(false)}
        footer={[
          <Button key="close" onClick={() => setOUSelectVisible(false)}>
            关闭
          </Button>,
        ]}
        width={600}
      >
        <Space orientation="vertical" style={{ width: "100%" }} size="middle">
          <div style={{ color: "#666", fontSize: "14px" }}>
            点击下方OU标签快速筛选该OU下的用户
          </div>
          <div style={{ maxHeight: 400, overflowY: "auto" }}>
            <Space wrap>
              {commonOUs.map((ou) => (
                <Tag
                  key={ou.dn}
                  style={{
                    cursor: "pointer",
                    padding: "8px 16px",
                    fontSize: "14px",
                    marginBottom: "8px",
                  }}
                  icon={<FolderOutlined />}
                  onClick={() => {
                    handleQuickOUSelect(ou.dn);
                    setOUSelectVisible(false);
                  }}
                >
                  {ou.name}
                </Tag>
              ))}
            </Space>
          </div>
          {commonOUs.length === 0 && (
            <div style={{ textAlign: "center", color: "#999", padding: "40px" }}>
              暂无常用OU配置
            </div>
          )}
        </Space>
      </Modal>
    </Card>
  );
};

export default ADUserPage;
