/**
 * System User 用户管理页面
 */

import { useState, useEffect, useCallback, useMemo, type FC, type Key } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Tag,
  Avatar,
  Card,
  Row,
  Col,
  Statistic,
  Layout,
  Alert,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  UserOutlined,
  TeamOutlined,
  LockOutlined,
  UnlockOutlined,
  KeyOutlined,
  ImportOutlined,
} from "@ant-design/icons";
import type { User } from "@/types";
import { post } from "@/lib/api";
import type { PageResponse } from "@/types";
import DeptTree from "@/components/DeptTree";
import { useTableManager } from "@/hooks/useTableManager";
import { usePagination } from "@/hooks/usePagination";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import ActionButtons from "@/components/shared/ActionButtons";
import { DepartmentTreeSelect } from "@/components/shared";
import ExcelImport from "@/components/shared/ExcelImport";
import { formatDateTime } from "@/utils/datetime";
import { createSorterMeta } from "@/utils/tableHelpers";
import type { SortOrder } from "@/hooks/useServerSort";

// 导入提取的文件
import { GENDER_OPTIONS, STATUS_OPTIONS } from "./constants";
import { formatGender, formatStatus } from "./utils.tsx";
import { useUserData } from "./hooks/useUserData";
import { useUserModals } from "./hooks/useUserModals";

const { Option } = Select;
const { Sider, Content } = Layout;

// 模块级稳定空数组 — 给 DeptTree.selectedKeys 在"未选部门"时复用，
// 避免每次 render 新数组引用（为 DeptTree 将来包 React.memo 铺路）。
const EMPTY_SELECTED_KEYS: Key[] = [];

// ==================== 表格列定义 ====================

interface UserTableColumnsProps {
  handleEdit: (record: User) => void;
  handleUpdateStatus: (id: string, status: number) => void;
  handleResetPassword: (user: User) => void;
  handleDelete: (id: string) => void;
  getColumnSortOrder: (field: string) => SortOrder | undefined;
}

function getUserTableColumns(props: UserTableColumnsProps): ColumnsType<User> {
  const { handleEdit, handleUpdateStatus, handleResetPassword, handleDelete, getColumnSortOrder } = props;

  return [
    {
      title: "用户名",
      dataIndex: "username",
      key: "username",
      width: 140,
      minWidth: 120,
      sorter: true,
      sortOrder: getColumnSortOrder("username"),
      render: (text, record) => (
        <Space>
          <Avatar src={record.avatar} icon={<UserOutlined />} />
          {text}
        </Space>
      ),
    },
    {
      title: "昵称",
      dataIndex: "nickname",
      key: "nickname",
      width: 100,
      minWidth: 80,
      sorter: true,
      sortOrder: getColumnSortOrder("nickname"),
    },
    {
      title: "工号",
      dataIndex: "employeeNo",
      key: "employeeNo",
      width: 100,
      minWidth: 80,
      sorter: true,
      sortOrder: getColumnSortOrder("employeeNo"),
      render: (text) => text || "-",
    },
    {
      title: "邮箱",
      dataIndex: "email",
      key: "email",
      width: 180,
      minWidth: 150,
      ellipsis: true,
      sorter: true,
      sortOrder: getColumnSortOrder("email"),
    },
    {
      title: "手机号",
      dataIndex: "phone",
      key: "phone",
      width: 130,
      minWidth: 120,
      sorter: true,
      sortOrder: getColumnSortOrder("phone"),
    },
    {
      title: "性别",
      dataIndex: "gender",
      key: "gender",
      width: 70,
      minWidth: 60,
      align: "center" as const,
      sorter: true,
      sortOrder: getColumnSortOrder("gender"),
      render: (gender) => formatGender(gender),
    },
    {
      title: "部门",
      dataIndex: "deptName",
      key: "deptName",
      width: 200,
      minWidth: 150,
      ellipsis: true,
      sorter: true,
      sortOrder: getColumnSortOrder("deptName"),
      render: (_, record) => record.deptFullName || record.deptName || "-",
    },
    {
      title: "角色",
      dataIndex: "roles",
      key: "roles",
      width: 160,
      minWidth: 120,
      render: (roles: string[] | Array<{ roleName?: string; roleKey?: string; id?: string }>) => {
        if (!roles || roles.length === 0) {
          return <span style={{ color: "var(--theme-text-tertiary, #999)" }}>无角色</span>;
        }
        return (
          <Space size="small" wrap>
            {roles.map((role, index) => {
              const roleName = typeof role === "string" ? role : role.roleName || role.roleKey || "-";
              return (
                <Tag key={index} color="blue">
                  {roleName}
                </Tag>
              );
            })}
          </Space>
        );
      },
    },
    {
      title: "状态",
      dataIndex: "status",
      key: "status",
      width: 80,
      minWidth: 70,
      align: "center" as const,
      sorter: true,
      sortOrder: getColumnSortOrder("status"),
      render: (status) => {
        const config = formatStatus(status);
        return <Tag color={config.color}>{config.text}</Tag>;
      },
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 170,
      minWidth: 160,
      sorter: true,
      sortOrder: getColumnSortOrder("createdAt"),
      render: (text) => formatDateTime(text),
    },
    {
      title: "操作",
      key: "action",
      width: 220,
      minWidth: 200,
      fixed: "right" as const,
      render: (_, record) => {
        const actions = [
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => handleEdit(record),
          },
          {
            key: "status",
            label: record.status === 0 ? "禁用" : "启用",
            icon: record.status === 1 ? <LockOutlined /> : <UnlockOutlined />,
            onClick: () => handleUpdateStatus(record.id, record.status === 1 ? 0 : 1),
          },
          {
            key: "reset-password",
            label: "重置密码",
            icon: <KeyOutlined />,
            onClick: () => handleResetPassword(record),
          },
          {
            key: "delete",
            label: "删除",
            icon: <DeleteOutlined />,
            danger: true,
            onClick: () => {
              Modal.confirm({
                title: "确定要删除这个用户吗？",
                okText: "确定",
                cancelText: "取消",
                okButtonProps: { danger: true },
                onOk: () => handleDelete(record.id),
              });
            },
          },
        ];

        return <ActionButtons actions={actions} />;
      },
    },
  ];
}

// ==================== 主组件 ====================

const UserManagement: FC = () => {
  // 自定义状态
  const location = useLocation();
  const [selectedDeptId, setSelectedDeptId] = usePersistedStateController<string>({
    keyPrefix: location.pathname,
    keySuffix: "selectedDeptId",
    defaultValue: "",
  });

  // 使用自定义 Hooks
  // departments 来自 useDeptTree 共享缓存(Phase 37 收敛),无需显式 load
  const { statistics, roles, departments, loadStatistics, loadRoles, ensureRoles } = useUserData();
  const {
    resetPasswordModalVisible,
    resettingUser,
    resetPasswordForm,
    openResetPasswordModal,
    closeResetPasswordModal,
  } = useUserModals();

  // 用户导入模态框
  const [importModalVisible, setImportModalVisible] = useState(false);

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 服务端排序：field 对应后端 userAllowedSortFields 白名单 key
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<User>("username"),
      createSorterMeta<User>("nickname"),
      createSorterMeta<User>("employeeNo"),
      createSorterMeta<User>("email"),
      createSorterMeta<User>("phone"),
      createSorterMeta<User>("gender", "number"),
      createSorterMeta<User>("deptName"),
      createSorterMeta<User>("status"),
      createSorterMeta<User>("createdAt", "date"),
    ],
    []
  );

  // 数据加载函数（status 统一转 Number 以适配后端）
  const fetchUsers = useCallback(async (params: Record<string, unknown>) => {
    if (params.status !== undefined && params.status !== null && params.status !== "") {
      params = { ...params, status: Number(params.status) };
    }
    const result = await post<PageResponse<User>>("/system/users/list", params);
    return { list: result.data?.list || [], total: result.data?.total || 0 };
  }, []);

  // 使用 useTableManager 管理表格状态（含服务端排序 + filters 持久化）
  const {
    loading,
    data: users,
    selectedRowKeys,
    searchForm,
    editForm,
    editModalVisible,
    editingItem: editingUser,
    setSelectedRowKeys,
    setEditModalVisible,
    loadData: loadUsers,
    handleAdd,
    handleEdit,
    resetSelection,
    getColumnSortOrder,
    handleTableChange,
    applyFilters,
    handleReset: tableHandleReset,
  } = useTableManager<User>(
    fetchUsers,
    {
      externalPagination: {
        current: paginationProps.current ?? 1,
        pageSize: paginationProps.pageSize ?? 10,
        setCurrent,
        setPageSize,
        setTotal,
      },
      sorterMetas,
    }
  );

  // ==================== 初始化 ====================

  useEffect(() => {
    loadUsers();
    loadStatistics();
    loadRoles();
    // departments 由 useDeptTree 自动拉取,无需在此手动触发
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ==================== 事件处理 ====================

  // 搜索：applyFilters 读 searchForm 并合并 recursiveDeptId（部门+子部门递归），自动带上当前排序
  // （status 在 fetchUsers 转 Number）。原来用 deptId 单值语义，选父部门时漏子部门用户，
  // 见 debug-session user-list-only-current-dept-no-children。
  const handleSearch = useCallback(() => {
    applyFilters(selectedDeptId ? { recursiveDeptId: selectedDeptId } : undefined);
  }, [applyFilters, selectedDeptId]);

  // 重置：清部门 + 清表单 + 清排序，回到第 1 页
  const handleReset = useCallback(() => {
    setSelectedDeptId("");
    tableHandleReset();
  }, [setSelectedDeptId, tableHandleReset]);

  const handleRefresh = useCallback(() => {
    loadUsers();
    loadStatistics();
  }, [loadUsers, loadStatistics]);

  const handleAddUser = useCallback(() => {
    handleAdd();
    const defaultValues: Record<string, unknown> = {
      gender: 0,
      status: 0,
    };
    if (departments.length > 0) {
      defaultValues.deptId = departments[0].id;
    }
    if (roles.length > 0) {
      defaultValues.roleIds = [roles[0].id];
    }
    editForm.setFieldsValue(defaultValues);
  }, [handleAdd, departments, roles, editForm]);

  const handleModalOpenChange = useCallback((open: boolean) => {
    if (open && editingUser) {
      // 角色兜底注入(2026-06-30,同 info-points):loadRoles 是 init 阶段异步调用,
      // 编辑打开模态框时若 roles 列表未到位 → 多选 Select 显示 raw UUID。
      // 主动重触发 loadRoles,并用 record.roles(若带 roleName)注入兜底。
      loadRoles().catch(() => {/* hook 内部已 handleApiError */});
      if (Array.isArray(editingUser.roles)) {
        // 兼容两种形态:string[] (UUID) 或 Array<{id, roleName, roleKey}>
        // 用 unknown 显式声明谓词参数,绕开 TS2677(predicate 必须可赋给参数类型)。
        const asObjects = (editingUser.roles as unknown[])
          .filter((r): r is { id: string; roleName?: string; roleKey?: string } => typeof r === "object" && r !== null && "id" in r)
          .map(r => ({ id: (r as { id: string }).id, roleName: r.roleName, roleKey: r.roleKey }));
        if (asObjects.length > 0) {
          ensureRoles(asObjects);
        }
      }
      editForm.setFieldsValue(editingUser);
    }
  }, [editingUser, editForm, loadRoles, ensureRoles]);

  const handleDelete = useCallback(async (id: string) => {
    try {
      await post(`/system/users/${id}/delete`);
      handleSuccess("删除");
      loadUsers();
      loadStatistics();
    } catch (error) {
      handleApiError(error, "删除");
    }
  }, [loadUsers, loadStatistics]);

  const handleBatchDelete = useCallback(async () => {
    if (selectedRowKeys.length === 0) {
      return;
    }

    try {
      await post("/system/users/batch-delete", { ids: selectedRowKeys });
      handleSuccess("批量删除");
      resetSelection();
      loadUsers();
      loadStatistics();
    } catch (error) {
      handleApiError(error, "批量删除");
    }
  }, [selectedRowKeys, resetSelection, loadUsers, loadStatistics]);

  const handleUpdateStatus = useCallback(async (id: string, status: number) => {
    try {
      await post(`/system/users/${id}/status`, { status });
      handleSuccess(status === 0 ? "启用" : "禁用");
      loadUsers();
      loadStatistics();
    } catch (error) {
      handleApiError(error, "操作");
    }
  }, [loadUsers, loadStatistics]);

  const handleSave = useCallback(async () => {
    try {
      const values = await editForm.validateFields() as Record<string, unknown>;

      if (values.gender !== undefined) {
        values.gender = Number(values.gender);
      }
      if (values.status !== undefined) {
        values.status = Number(values.status);
      }

      const url = editingUser
        ? `/system/users/${editingUser.id}/update`
        : "/system/users";

      await post(url, values);
      handleSuccess(editingUser ? "更新" : "创建");
      setEditModalVisible(false);
      loadUsers();
      loadStatistics();
      // 注:部门写操作(create/update/delete user 关联 deptId)不需要在此主动刷新部门列表
      // 因为部门树本身(sys_dept)未变;若后续有部门 CRUD,应调用 useInvalidateDept()
      // 失效共享缓存(Phase 37 收敛后部门刷新由 hook 自动管理)
    } catch (error) {
      handleApiError(error, "操作");
    }
  }, [editingUser, editForm, loadUsers, loadStatistics, setEditModalVisible]);

  const handleSaveResetPassword = useCallback(async () => {
    try {
      const values = await resetPasswordForm.validateFields() as { password: string };

      await post(`/system/users/${resettingUser!.id}/reset-password`, {
        password: values.password,
      });

      handleSuccess("密码重置");
      closeResetPasswordModal();
    } catch (error) {
      handleApiError(error, "重置密码");
    }
  }, [resetPasswordForm, resettingUser, closeResetPasswordModal]);

  // 部门联动：applyFilters 合并 recursiveDeptId（部门+子部门递归），保留搜索框值与当前排序，回第 1 页
  // 用 recursiveDeptId 而非 deptId：左侧点父部门时要看到父+所有子部门的用户（与用户预期一致）。
  const handleDeptSelect = useCallback((selectedKeys: Key[]) => {
    if (selectedKeys.length > 0) {
      setSelectedDeptId(selectedKeys[0] as string);
      applyFilters({ recursiveDeptId: selectedKeys[0] });
    } else {
      setSelectedDeptId("");
      applyFilters();
    }
  }, [applyFilters, setSelectedDeptId]);

  // 表格列
  const columns = getUserTableColumns({
    handleEdit,
    handleUpdateStatus,
    handleResetPassword: openResetPasswordModal,
    handleDelete,
    getColumnSortOrder,
  });

  // 给 DeptTree 喂稳定引用 — selectedKeys 在 selectedDeptId 不变时复用同一数组，
  // 配合 EMPTY_SELECTED_KEYS 避免空数组每次 render 新引用。
  // （Vercel rerender-defer-reads / rerender-memo-with-default-value）
  const deptTreeSelectedKeys = useMemo<Key[]>(
    () => (selectedDeptId ? [selectedDeptId] : EMPTY_SELECTED_KEYS),
    [selectedDeptId]
  );

  return (
    <Layout style={{ display: "flex", alignItems: "stretch" }}>
      {/* 左侧部门树 */}
      <Sider width={360} className="dept-list-sider" style={{ background: "#fff", padding: "0 16px 16px 0" }}>
        <DeptTree
          onSelect={handleDeptSelect}
          selectedKeys={deptTreeSelectedKeys}
        />
      </Sider>

      {/* 右侧内容区 */}
      <Content style={{ padding: 0, background: "#fff" }}>
        <div>
          {/* 统计卡片 */}
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={8}>
              <Card>
                <Statistic
                  title="总用户数"
                  value={statistics.total}
                  prefix={<TeamOutlined />}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card>
                <Statistic
                  title="正常用户"
                  value={statistics.active}
                  styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
                  prefix={<UnlockOutlined />}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card>
                <Statistic
                  title="禁用用户"
                  value={statistics.inactive}
                  styles={{ content: { color: "var(--theme-error, #cf1322)" } }}
                  prefix={<LockOutlined />}
                />
              </Card>
            </Col>
          </Row>

          {/* 搜索表单和操作按钮 */}
          <Card style={{ marginBottom: 16 }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: "16px" }}>
              <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
                <Form.Item name="username" label="用户名">
                  <Input placeholder="请输入用户名" />
                </Form.Item>
                <Form.Item name="nickname" label="昵称">
                  <Input placeholder="请输入昵称" />
                </Form.Item>
                <Form.Item name="status" label="状态">
                  <Select placeholder="请选择状态" style={{ width: 120 }} allowClear onSearch={() => {}}>
                    {STATUS_OPTIONS.map(opt => (
                      <Option key={opt.value} value={String(opt.value)}>{opt.label}</Option>
                    ))}
                  </Select>
                </Form.Item>
                <Form.Item>
                  <Space>
                    <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                      搜索
                    </Button>
                    <Button onClick={handleReset}>重置</Button>
                    <Button icon={<ReloadOutlined />} onClick={handleRefresh}>
                      刷新
                    </Button>
                  </Space>
                </Form.Item>
              </Form>
              <Space>
                {selectedRowKeys.length > 0 && (
                  <Button
                    icon={<DeleteOutlined />}
                    onClick={handleBatchDelete}
                    style={{
                      color: "var(--theme-error, #ff4d4f)",
                    }}
                  >
                    批量删除 ({selectedRowKeys.length})
                  </Button>
                )}
                <Button type="primary" icon={<PlusOutlined />} onClick={handleAddUser}>
                  新增用户
                </Button>
                <Button icon={<ImportOutlined />} onClick={() => setImportModalVisible(true)}>
                  导入用户
                </Button>
              </Space>
            </div>
            {selectedRowKeys.length > 0 && (
              <Alert
                message={
                  <span>
                    已选择 <strong>{selectedRowKeys.length}</strong> 个用户，
                    <Button
                      type="link"
                      size="small"
                      onClick={() => setSelectedRowKeys([])}
                      style={{ padding: 0 }}
                    >
                      取消选择
                    </Button>
                  </span>
                }
                type="info"
                showIcon
                style={{ marginTop: 12 }}
              />
            )}
          </Card>

          {/* 用户表格 */}
          <Card>
            <Table
              columns={columns}
              dataSource={users}
              rowKey="id"
              loading={loading}
              pagination={paginationProps}
              onChange={handleTableChange}
              rowSelection={{
                selectedRowKeys,
                onChange: setSelectedRowKeys,
              }}
            />
          </Card>

          {/* 用户导入模态框：复用通用 ExcelImport 组件 */}
          <ExcelImport
            entityType="user"
            entityName="用户"
            importUrl="/api/v1/system/users/import"
            templateUrl="/api/v1/system/users/import/template"
            visible={importModalVisible}
            onClose={() => setImportModalVisible(false)}
            onImportSuccess={() => {
              loadUsers();
              loadStatistics();
            }}
          />
        </div>
      </Content>

      {/* 编辑弹窗 */}
      <Modal
        title={editingUser ? "编辑用户" : "新增用户"}
        open={editModalVisible}
        onOk={handleSave}
        afterOpenChange={handleModalOpenChange}
        onCancel={() => setEditModalVisible(false)}
        width={600}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 4 }} wrapperCol={{ span: 20 }}>
          <Form.Item
            name="username"
            label="用户名"
            rules={[{ required: true, message: "请输入用户名" }]}
          >
            <Input disabled={!!editingUser} className="user-form-input" />
          </Form.Item>
          {!editingUser && (
            <Form.Item
              name="password"
              label="密码"
              rules={[{ required: true, message: "请输入密码" }]}
            >
              <Input.Password className="user-form-input" />
            </Form.Item>
          )}
          <Form.Item
            name="nickname"
            label="昵称"
            rules={[{ required: true, message: "请输入昵称" }]}
          >
            <Input className="user-form-input" />
          </Form.Item>
          <Form.Item name="employeeNo" label="工号">
            <Input placeholder="请输入工号" className="user-form-input" />
          </Form.Item>
          <Form.Item name="email" label="邮箱">
            <Input className="user-form-input" />
          </Form.Item>
          <Form.Item name="phone" label="手机号">
            <Input className="user-form-input" />
          </Form.Item>
          <Form.Item name="gender" label="性别" initialValue={2}>
            <Select className="user-form-input" onSearch={() => {}}>
              {GENDER_OPTIONS.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="deptId" label="部门">
            <DepartmentTreeSelect
              departments={departments}
              className="user-form-input"
            />
          </Form.Item>
          <Form.Item name="roleIds" label="角色">
            <Select
              mode="multiple"
              placeholder="请选择角色"
              allowClear
              optionFilterProp="label"
              showSearch
              className="user-form-input"
             onSearch={() => {}}>
              {roles.map(role => (
                <Option key={role.id} value={role.id} label={role.roleName}>
                  {role.roleName}
                </Option>
              ))}
            </Select>
          </Form.Item>
          <Form.Item name="status" label="状态" initialValue={0}>
            <Select className="user-form-input" onSearch={() => {}}>
              {STATUS_OPTIONS.map(opt => (
                <Option key={opt.value} value={opt.value}>{opt.label}</Option>
              ))}
            </Select>
          </Form.Item>
        </Form>
      </Modal>

      {/* 重置密码弹窗 */}
      <Modal
        title={`重置密码 - ${resettingUser?.username || ""}`}
        open={resetPasswordModalVisible}
        onOk={handleSaveResetPassword}
        onCancel={closeResetPasswordModal}
        width={400}
      >
        <Form form={resetPasswordForm} layout="vertical">
          <Form.Item
            name="password"
            label="新密码"
            rules={[
              { required: true, message: "请输入新密码" },
              { min: 6, message: "密码长度不能少于6位" },
              { max: 20, message: "密码长度不能超过20位" },
            ]}
          >
            <Input.Password placeholder="请输入新密码" />
          </Form.Item>
          <Form.Item
            name="confirmPassword"
            label="确认密码"
            dependencies={["password"]}
            rules={[
              { required: true, message: "请确认密码" },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue("password") === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error("两次输入的密码不一致"));
                },
              }),
            ]}
          >
            <Input.Password placeholder="请再次输入密码" />
          </Form.Item>
        </Form>
      </Modal>
    </Layout>
  );
};

export default UserManagement;
