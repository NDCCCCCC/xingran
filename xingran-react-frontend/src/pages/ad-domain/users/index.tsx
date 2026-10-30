import { useState, useEffect, useMemo } from "react";
// Phase 71 (W5)：裸 Input/Select/TextArea → 契约组件（children 形态改写 options）
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import { Card, Table, TreeSelect, Space, Button, Tag, Form, Modal, message, Tooltip } from "antd";
import type { TablePaginationConfig } from "antd/es/table";
import type { SorterResult } from "antd/es/table/interface";
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
// ⚠️ 直接引模块路径（不走 @/components/shared 桶文件）：桶文件 re-export 了
// FloorPlanEditor → @react-three/fiber → three.js，引入它会让本页的动态 import
// （既有测试的 `await import(...)`）显著变慢并触发超时（实测）。
import { useActionPermission } from "@/components/shared/useActionPermission";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { useADConfigs } from "@/hooks/useADConfigs";

import type { FC } from "react";
import { usePagination } from "@/hooks/usePagination";

const ADUserPage: FC = () => {
  const [form] = Form.useForm();
  const [editForm] = Form.useForm();

  // Phase 86 W4（RBAC-02）：操作级权限门控（三态 ready / loading / error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  /**
   * 本页**全部**写操作归 `ops:ad:user:edit` 单一码 —— 该码组在
   * `src/constants/permissionCodes.ts` 里**只有 `edit` 一个写键**
   * （无独立 `enable` / `move` / `sync` 键）。
   *
   * 归码的后端依据（`internal/api/v1/system/ad_domain_router.go:137-146`）：
   * ```
   * // 85-03 RBAC-01：读组保留 ANY-of，用户写操作走 strict 兄弟组（move/enable/
   * // disable/batch-sync 均为编辑语义，归 ops:ad:user:edit）。
   * userEdit.Use(middleware.RequireStrictPermissions([]string{string(permission.OpsADUserEdit)}, core))
   * ```
   * 即后端把 move / enable / disable / batch-sync **四条写路由挂在同一个 strict 单码**下
   * → 前端四条写入口必须同码，否则会出现「前端放行 / 后端 403」的错配。
   */
  const permUserEdit = permGate.decide(PERMISSION_CODES.opsADUser.edit);

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
  const { paginationProps, setCurrent, setTotal } = usePagination();

  const [ouTree, setOUTree] = useState<ADOUNode[]>([]);
  const [ouSelectVisible, setOUSelectVisible] = useState(false);

  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editSubmitting, setEditSubmitting] = useState(false);
  const [editingUser, setEditingUser] = useState<ADUser | null>(null);

  const [moveModalVisible, setMoveModalVisible] = useState(false);
  const [movingUser, setMovingUser] = useState<ADUser | null>(null);

  // Batch sync state
  const [selectedUsers, setSelectedUsers] = useState<ADUser[]>([]);
  const [_selectedUserIds, setSelectedUserIds] = useState<string[]>([]);
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
            .filter((node) => !excludedOUNames.includes(node.name))
            .map((node) => ({
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
    return commonNames.map((name) => findOUByName(name)).filter(Boolean) as ADOUNode[];
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
  const handleTableChange = (
    _pagination: TablePaginationConfig,
    _filters: Record<string, unknown>,
    sorter: SorterResult<ADUser> | SorterResult<ADUser>[]
  ) => {
    // 用 local const 持有新值传 fetchUsers，规避 React 18 setState 异步时序
    // （setState 后立即读 state 仍为旧值——加 deep 一层避坑）
    const s = Array.isArray(sorter) ? sorter[0] : sorter;
    const newCol = s?.field != null ? String(s.field) : "";
    const newAsc = s?.order === "ascend";
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
    } catch (_error) {
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

    // eslint-disable-next-line no-restricted-syntax -- 非删除语义：同步用户到系统用户表为幂等补写，不删除任何数据，UI-SPEC §1.1 判定规则排除
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
              userDns = res.data.list.map((u) => u.userDn);
            }
          } else {
            // 普通模式：使用当前选中的用户
            userDns = selectedUsers.map((u) => u.userDn);
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
        } catch (_error) {
          message.error("批量同步失败");
        } finally {
          setBatchSyncLoading(false);
        }
      },
    });
  };

  // 构建TreeSelect所需的数据结构
  const buildTreeSelectData = (
    nodes: ADOUNode[]
  ): Array<{
    value: string;
    title: string;
    children?: Array<{ value: string; title: string }>;
  }> => {
    return nodes.map((node) => ({
      value: node.dn,
      title: node.name,
      children:
        node.children && node.children.length > 0 ? buildTreeSelectData(node.children) : undefined,
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
      render: (ouDn: string, _record: ADUser) => {
        if (!ouDn) return "-";
        // 获取当前配置的BaseDN
        const currentConfig = configs.find((c) => c.id === selectedConfig);
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
      // 域控 manager 属性（DN 字符串），为空时显示 "-"
      title: "域控经理",
      dataIndex: "manager",
      key: "manager",
      width: 260,
      ellipsis: true,
      render: (manager?: string) =>
        manager ? (
          <span title={manager} style={{ cursor: "help" }}>
            {manager}
          </span>
        ) : (
          "-"
        ),
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
            // Phase 86 W4：启用/禁用是**状态写**（POST /users/:id/enable|disable）
            // → 归 ops:ad:user:edit（该码组无独立 enable 键；后端同挂此码）。
            permission: PERMISSION_CODES.opsADUser.edit,
          },
          {
            key: "move",
            label: "移动",
            icon: <SwapOutlined />,
            onClick: () => handleMove(user),
            // Phase 86 W4：移动是**位置写**（POST /users/:id/move，改 OU）
            // → 归 ops:ad:user:edit（该码组无独立 move 键；后端同挂此码）。
            permission: PERMISSION_CODES.opsADUser.edit,
          },
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => handleEdit(user),
            // Phase 86 W4：编辑用户资料 → ops:ad:user:edit
            permission: PERMISSION_CODES.opsADUser.edit,
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
            {/* Phase 71 W5：迁 AppSelect——原显式 showSearch 保持；configId 为 UUID 外键，
                原 allowClear=true 行为不变；AppSelect 泛型被 ComponentProps 擦除 →
                onChange 值为 unknown，收敛回业务类型 */}
            <AppSelect
              allowClear
              style={{ width: 200 }}
              value={selectedConfig}
              onChange={(value) => setSelectedConfig(value as string)}
              options={configs.map((c) => ({ label: c.configName, value: c.id }))}
              showSearch
              // noop 满足 local/no-large-dropdown-list(要求 Select 带 onSearch);
              // showSearch 让 antd 警告(onSearch 应配合 showSearch)消除
              onSearch={() => {}}
            />
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
                  <Button icon={<SearchOutlined />} onClick={() => setOUSelectVisible(true)}>
                    快捷
                  </Button>
                </Tooltip>
              )}
            </Space.Compact>
          </Form.Item>
          <Form.Item name="username">
            <AppInput placeholder="用户名" allowClear style={{ width: 150 }} />
          </Form.Item>
          <Form.Item name="isEnabled">
            {/* Phase 71 W5：children 写法改写 options；原显式 showSearch（boolean 值下拉，
                契约过滤走 label）保持 */}
            <AppSelect
              placeholder="状态"
              allowClear
              style={{ width: 120 }}
              showSearch
              onSearch={() => {}}
              options={[
                { value: true, label: "启用" },
                { value: false, label: "禁用" },
              ]}
            />
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
              {/*
                Phase 86 W4 —— **施工图（86-04 PLAN）未列的写入口，本波按后端硬证据补齐**：
                「批量同步」调 `batchSyncADUsersDirect` → POST /users/batch-sync，
                把 AD 用户**写进系统用户表**（落库）→ 是业务写入口，不是只读。
                后端 `ad_domain_router.go:137-146` 已把它与 move/enable/disable
                挂在同一个 strict 单码 `ops:ad:user:edit` 下（注释原文：
                「move/enable/disable/batch-sync 均为编辑语义，归 ops:ad:user:edit」）
                → 前端归 `opsADUser.edit`，与本页其余三条行操作同码。
                （施工图遗漏已登记在 86-STATE.md 的「W4 施工图外发现」节。）
              */}
              {permUserEdit.visible && (
                <Button
                  type="primary"
                  icon={<SyncOutlined />}
                  disabled={permUserEdit.disabled || (selectedUsers.length === 0 && !selectAllMode)}
                  title={permUserEdit.hint}
                  onClick={handleBatchSync}
                  loading={batchSyncLoading}
                >
                  批量同步 ({selectAllMode ? selectAllTotal : selectedUsers.length})
                </Button>
              )}
            </Space>
          </Form.Item>
        </Form>

        {/* 用户列表 */}
        <Table
          columns={columns}
          dataSource={users}
          loading={loading}
          rowKey="id"
          scroll={{ x: 1900 }}
          pagination={paginationProps}
          onChange={handleTableChange}
          rowSelection={
            selectAllMode
              ? undefined // 全选模式下禁用表格行选择
              : {
                  selectedRowKeys: selectedUsers.map((u) => u.id),
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
        onOk={async () => {
          setEditSubmitting(true);
          try {
            await handleUpdateUser();
          } finally {
            setEditSubmitting(false);
          }
        }}
        cancelButtonProps={{ disabled: editSubmitting }}
        onCancel={() => setEditModalVisible(false)}
        width={600}
      >
        <Form form={editForm} layout="vertical">
          <Form.Item label="显示名" name="displayName">
            <AppInput allowClear />
          </Form.Item>
          <Form.Item label="邮箱" name="email">
            <AppInput allowClear />
          </Form.Item>
          <Form.Item label="办公电话" name="phone">
            <AppInput allowClear />
          </Form.Item>
          <Form.Item label="手机" name="mobile">
            <AppInput allowClear />
          </Form.Item>
          <Form.Item label="职位" name="title">
            <AppInput allowClear />
          </Form.Item>
          {/* 域控 manager 属性（DN）为只读展示，不参与提交 */}
          <Form.Item label="域控经理">
            <AppInput allowClear readOnly value={editingUser?.manager || ""} placeholder="未设置" />
          </Form.Item>
          <Form.Item label="部门" name="department">
            <AppInput allowClear />
          </Form.Item>
          <Form.Item label="描述" name="description">
            <AppTextArea rows={3} />
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
        <Space orientation="vertical" style={{ width: "100%" }} size="medium">
          <div style={{ color: "var(--theme-text-secondary)", fontSize: "14px" }}>
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
            <div
              style={{ textAlign: "center", color: "var(--theme-text-tertiary)", padding: "40px" }}
            >
              暂无常用OU配置
            </div>
          )}
        </Space>
      </Modal>
    </Card>
  );
};

export default ADUserPage;
