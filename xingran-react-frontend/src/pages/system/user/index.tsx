/**
 * System User 用户管理页面
 */

import { useState, useEffect, useCallback, useMemo, type FC, type Key } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import { Table, Button, Space, Modal, Form, Input, Switch, Layout, Alert, Card, App } from "antd";
// Phase 71 (W2)：裸 Input/Select → 契约组件。
// 跳过项：<Input.Password>（成员表达式）不在本波 selector 命中面，保留 antd Input 导入。
// Select 形态：状态/性别为固定枚举（children → options + 禁搜）；角色多选为 children 带
// label（→ options 形态）；职位下拉 options 无 label 字段且显式 optionFilterProp="value"，
// 该显式值经 AppSelect 的 {...rest} 覆盖默认 "label"，行为不变。
import { AppInput, AppSelect } from "@/components/ui";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  ImportOutlined,
} from "@ant-design/icons";
import type { User } from "@/types";
import { post } from "@/lib/api";
import type { PageResponse } from "@/types";
import { previewManagers, type ManagerPreviewItem } from "@/lib/managerApi";
import DeptTree from "@/components/DeptTree";
import { useTableManager } from "@/hooks/useTableManager";
import { usePagination } from "@/hooks/usePagination";
import { useDict, type DictItem } from "@/hooks/useDict";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import { DepartmentTreeSelect, confirmDelete, useActionPermission } from "@/components/shared";
import type { ActionPermissionDecision } from "@/components/shared";
import ExcelImportLazy from "@/components/shared/ExcelImportLazy";
import { formatDateTime } from "@/utils/datetime";
import { createSorterMeta } from "@/utils/tableHelpers";
import type { SortOrder } from "@/hooks/useServerSort";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

// 导入提取的文件
import { GENDER_OPTIONS, STATUS_OPTIONS } from "./constants";
import { formatGender } from "./utils.tsx";
import { useUserData } from "./hooks/useUserData";
import { useUserModals } from "./hooks/useUserModals";

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
  /** Phase 69 DICT-03: sys_user_sex 字典数据（空数组时 formatGender 回退静态映射） */
  genderDict: DictItem[];
  /** 经理预览映射（key=用户 id；纯计算不落库，规则变更后由后端缓存失效保证准实时） */
  managerMap: Record<string, ManagerPreviewItem>;
  /**
   * v1.24 #16（PERM-02/03）：操作列门控决策。
   * 三态语义：visible=false → 不渲染；disabled=true → 渲染但禁用（hint 作提示）。
   */
  permEdit: ActionPermissionDecision;
  permResetPwd: ActionPermissionDecision;
  permRemove: ActionPermissionDecision;
}

function getUserTableColumns(props: UserTableColumnsProps): ColumnsType<User> {
  const {
    handleEdit,
    handleUpdateStatus,
    handleResetPassword,
    handleDelete,
    getColumnSortOrder,
    genderDict,
    managerMap,
    permEdit,
    permResetPwd,
    permRemove,
  } = props;

  return [
    {
      title: "用户名",
      dataIndex: "username",
      key: "username",
      width: 140,
      minWidth: 120,
      sorter: true,
      sortOrder: getColumnSortOrder("username"),
      // 原型复刻：用户名纯 mono 文本（demo .cell-id，无头像）
      render: (text) => <span className="xr-cell-id">{text}</span>,
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
      render: (text) => <span className="xr-cell-id">{text || "—"}</span>,
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
      // phone 为后端「联系电话」字段；mobile 为后端新增的「手机号」字段（AD 同步写入）
      title: "联系电话",
      dataIndex: "phone",
      key: "phone",
      width: 130,
      minWidth: 120,
      sorter: true,
      sortOrder: getColumnSortOrder("phone"),
      render: (text) => <span className="xr-cell-id">{text || "—"}</span>,
    },
    {
      title: "手机号",
      dataIndex: "mobile",
      key: "mobile",
      width: 130,
      minWidth: 120,
      ellipsis: true,
      render: (text) => <span className="xr-cell-id">{text || "—"}</span>,
    },
    {
      title: "职位",
      dataIndex: "title",
      key: "title",
      width: 140,
      minWidth: 120,
      ellipsis: true,
      render: (text) => <span className="xr-cell-id">{text || "—"}</span>,
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
      render: (gender) => formatGender(gender, genderDict),
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
      // 经理预览：按当前生效的经理同步规则纯内存计算（不落库），
      // 显示「姓名(账号)」；未配置规则或未解析出时显示 "—"，
      // hover 提示给出原因（如“未命中任何机构定义”）。
      title: "经理",
      dataIndex: "manager",
      key: "manager",
      width: 140,
      minWidth: 120,
      ellipsis: true,
      render: (_, record) => {
        const preview = managerMap[record.id];
        if (!preview) return <span className="xr-cell-id">—</span>;
        const label =
          preview.managerName || preview.managerUsername
            ? `${preview.managerName || preview.managerUsername}${
                preview.managerUsername && preview.managerName ? `(${preview.managerUsername})` : ""
              }`
            : "—";
        const tip =
          preview.reason ||
          (preview.warnings && preview.warnings.length > 0
            ? preview.warnings.join("；")
            : preview.ruleName
              ? `命中规则：${preview.ruleName}`
              : "");
        return (
          <span className="xr-cell-id" title={tip}>
            {label}
          </span>
        );
      },
    },
    {
      title: "角色",
      dataIndex: "roles",
      key: "roles",
      width: 160,
      minWidth: 120,
      // 原型复刻：品牌圆点 tag（demo .tag-green / .tag-gold，超级管理员=金）
      render: (roles: string[] | Array<{ roleName?: string; roleKey?: string; id?: string }>) => {
        if (!roles || roles.length === 0) {
          return <span style={{ color: "var(--theme-text-tertiary)" }}>无角色</span>;
        }
        return (
          <Space size="small" wrap>
            {roles.map((role, index) => {
              const roleName =
                typeof role === "string" ? role : role.roleName || role.roleKey || "-";
              const isSuperAdmin =
                roleName === "超级管理员" || roleName.toLowerCase().includes("admin");
              return (
                <span
                  key={index}
                  className={`xr-tag ${isSuperAdmin ? "xr-tag-gold" : "xr-tag-green"}`}
                >
                  {roleName}
                </span>
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
      // 原型复刻：状态用 Switch 直改（0=启用 / 1=禁用），替代旧 Tag + 操作列「禁用/启用」
      // v1.24 #16：状态变更走 POST /system/users/:id/status（后端绑定 system:user:edit），
      // 前端同步门控——无 edit 权限时 Switch 不可交互（禁用而非隐藏：状态是只读信息）。
      render: (status, record) => (
        <Switch
          checked={status === 0}
          disabled={permEdit.disabled || !permEdit.visible}
          onChange={(checked) => handleUpdateStatus(record.id, checked ? 0 : 1)}
        />
      ),
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      width: 170,
      minWidth: 160,
      sorter: true,
      sortOrder: getColumnSortOrder("createdAt"),
      render: (text) => <span className="xr-cell-time">{formatDateTime(text)}</span>,
    },
    {
      title: "操作",
      key: "action",
      width: 200,
      minWidth: 180,
      fixed: "right" as const,
      // 原型复刻：纯文字链接（demo .row-ops，无图标不折叠）
      // v1.24 #16：编辑 / 重置密码 / 删除 分别按 system:user:edit / resetPwd / remove 门控
      render: (_, record) => (
        <div className="xr-row-ops">
          {permEdit.visible && (
            <Button
              type="link"
              size="small"
              disabled={permEdit.disabled}
              title={permEdit.hint}
              onClick={() => handleEdit(record)}
            >
              编辑
            </Button>
          )}
          {permResetPwd.visible && (
            <Button
              type="link"
              size="small"
              disabled={permResetPwd.disabled}
              title={permResetPwd.hint}
              onClick={() => handleResetPassword(record)}
            >
              重置密码
            </Button>
          )}
          {permRemove.visible && (
            <Button
              type="link"
              size="small"
              className="xr-op-danger"
              disabled={permRemove.disabled}
              title={permRemove.hint}
              onClick={() => {
                confirmDelete({
                  target: `用户 ${record.username}`,
                  onDelete: async () => {
                    await handleDelete(record.id);
                  },
                });
              }}
            >
              删除
            </Button>
          )}
        </div>
      ),
    },
  ];
}

// ==================== 主组件 ====================

const UserManagement: FC = () => {
  const { message: _message } = App.useApp();

  // v1.24 #16（PERM-02/03）：操作级权限门控（三态 ready/loading/error）。
  // 权限码来自单一真相源 src/constants/permissionCodes.ts。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  const permEdit = permGate.decide(PERMISSION_CODES.systemUser.edit);
  const permResetPwd = permGate.decide(PERMISSION_CODES.systemUser.resetPwd);
  const permRemove = permGate.decide(PERMISSION_CODES.systemUser.remove);
  const permAdd = permGate.decide(PERMISSION_CODES.systemUser.add);
  const permImport = permGate.decide(PERMISSION_CODES.systemUser.import);
  // 注：system:user:export 已登记在权限码表，但用户模块**未注册导出路由**、
  // 前端也无导出按钮（xtgl 缺 export 权限也不会 403）——待导出功能落地时接入。
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
  const [submitting, setSubmitting] = useState(false);

  // Phase 69 DICT-03: 性别下拉迁 useDict("sys_user_sex")——字典管理页改 label 后
  // 搜索/编辑下拉与表格渲染随之变化；字典空态/接口异常时回退静态 GENDER_OPTIONS
  const { data: genderDict = [] } = useDict("sys_user_sex");

  // 新增用户默认性别取字典 isDefault 项（seed 为 "2" 保密，对齐后端 User.Gender gorm
  // default:2）；字典空态回退静态 0（迁移前行为）
  const defaultGender = useMemo(() => {
    const item = genderDict.find((d) => d.isDefault);
    return item ? Number(item.dictValue) : 0;
  }, [genderDict]);

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
  } = useTableManager<User>(fetchUsers, {
    externalPagination: {
      current: paginationProps.current ?? 1,
      pageSize: paginationProps.pageSize ?? 10,
      setCurrent,
      setPageSize,
      setTotal,
    },
    sorterMetas,
  });

  // ==================== 经理预览（当前页） ====================

  // 经理预览映射（key=用户 id）。监听当前页数据变化批量查询；
  // previewManagers 内部静默降级，失败不阻塞用户列表展示。
  const [managerMap, setManagerMap] = useState<Record<string, ManagerPreviewItem>>({});

  const userIdsKey = useMemo(() => (users ?? []).map((u) => u.id).join(","), [users]);

  useEffect(() => {
    const ids = userIdsKey ? userIdsKey.split(",") : [];
    if (ids.length === 0) {
      setManagerMap({});
      return;
    }
    let cancelled = false;
    previewManagers(ids).then((items) => {
      if (cancelled) return;
      const next: Record<string, ManagerPreviewItem> = {};
      for (const item of items) next[item.userId] = item;
      setManagerMap(next);
    });
    return () => {
      cancelled = true;
    };
  }, [userIdsKey]);

  // ==================== 职位下拉（数据源：岗位管理 sys_post） ====================

  // 职位下拉选项 = 启用中的岗位名称。title 列本身是自由文本（AD 同步会写入
  // AD 侧 title 原文），值不在岗位列表时 Select 直接显示原文本，不丢数据。
  const [postTitleOptions, setPostTitleOptions] = useState<{ value: string }[]>([]);

  useEffect(() => {
    let cancelled = false;
    post<{ list: { postName: string }[]; total: number }>("/system/posts/list", {
      current: 1,
      pageSize: 200,
      status: 0,
    })
      .then((res) => {
        if (cancelled) return;
        const names = Array.from(new Set((res.data?.list ?? []).map((p) => p.postName)));
        setPostTitleOptions(names.map((n) => ({ value: n })));
      })
      .catch(() => {
        /* 拉取失败降级为无选项的可输入下拉，不阻塞表单 */
      });
    return () => {
      cancelled = true;
    };
  }, []);

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
      gender: defaultGender,
      status: 0,
    };
    if (departments.length > 0) {
      defaultValues.deptId = departments[0].id;
    }
    if (roles.length > 0) {
      defaultValues.roleIds = [roles[0].id];
    }
    editForm.setFieldsValue(defaultValues);
  }, [handleAdd, departments, roles, editForm, defaultGender]);

  const handleModalOpenChange = useCallback(
    (open: boolean) => {
      if (open && editingUser) {
        // 角色兜底注入(2026-06-30,同 info-points):loadRoles 是 init 阶段异步调用,
        // 编辑打开模态框时若 roles 列表未到位 → 多选 Select 显示 raw UUID。
        // 主动重触发 loadRoles,并用 record.roles(若带 roleName)注入兜底。
        loadRoles().catch(() => {
          /* hook 内部已 handleApiError */
        });
        if (Array.isArray(editingUser.roles)) {
          // 兼容两种形态:string[] (UUID) 或 Array<{id, roleName, roleKey}>
          // 用 unknown 显式声明谓词参数,绕开 TS2677(predicate 必须可赋给参数类型)。
          const asObjects = (editingUser.roles as unknown[])
            .filter(
              (r): r is { id: string; roleName?: string; roleKey?: string } =>
                typeof r === "object" && r !== null && "id" in r
            )
            .map((r) => ({
              id: (r as { id: string }).id,
              roleName: r.roleName,
              roleKey: r.roleKey,
            }));
          if (asObjects.length > 0) {
            ensureRoles(asObjects);
          }
        }
        editForm.setFieldsValue(editingUser);
      }
    },
    [editingUser, editForm, loadRoles, ensureRoles]
  );

  const handleDelete = useCallback(
    async (id: string) => {
      try {
        await post(`/system/users/${id}/delete`);
        handleSuccess("删除");
        loadUsers();
        loadStatistics();
      } catch (error) {
        handleApiError(error, "删除");
      }
    },
    [loadUsers, loadStatistics]
  );

  const handleBatchDelete = useCallback(() => {
    if (selectedRowKeys.length === 0) {
      return;
    }

    // STOP-05: 批删前二次确认（Phase 65 统一走 confirmDelete helper）
    confirmDelete({
      title: "确认批量删除",
      target: `选中的 ${selectedRowKeys.length} 项`,
      onDelete: async () => {
        await post("/system/users/batch-delete", { ids: selectedRowKeys });
        resetSelection();
        loadUsers();
        loadStatistics();
      },
    });
  }, [selectedRowKeys, resetSelection, loadUsers, loadStatistics]);

  const handleUpdateStatus = useCallback(
    async (id: string, status: number) => {
      try {
        await post(`/system/users/${id}/status`, { status });
        handleSuccess(status === 0 ? "启用" : "禁用");
        loadUsers();
        loadStatistics();
      } catch (error) {
        handleApiError(error, "操作");
      }
    },
    [loadUsers, loadStatistics]
  );

  const handleSave = useCallback(async () => {
    try {
      const values = (await editForm.validateFields()) as Record<string, unknown>;

      if (values.gender !== undefined) {
        values.gender = Number(values.gender);
      }
      if (values.status !== undefined) {
        values.status = Number(values.status);
      }

      const url = editingUser ? `/system/users/${editingUser.id}/update` : "/system/users";

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
      const values = (await resetPasswordForm.validateFields()) as { password: string };

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
  const handleDeptSelect = useCallback(
    (selectedKeys: Key[]) => {
      if (selectedKeys.length > 0) {
        setSelectedDeptId(selectedKeys[0] as string);
        applyFilters({ recursiveDeptId: selectedKeys[0] });
      } else {
        setSelectedDeptId("");
        applyFilters();
      }
    },
    [applyFilters, setSelectedDeptId]
  );

  // 表格列
  const columns = getUserTableColumns({
    handleEdit,
    handleUpdateStatus,
    handleResetPassword: openResetPasswordModal,
    handleDelete,
    getColumnSortOrder,
    genderDict,
    managerMap,
    permEdit,
    permResetPwd,
    permRemove,
  });

  // 给 DeptTree 喂稳定引用 — selectedKeys 在 selectedDeptId 不变时复用同一数组，
  // 配合 EMPTY_SELECTED_KEYS 避免空数组每次 render 新引用。
  // （Vercel rerender-defer-reads / rerender-memo-with-default-value）
  const deptTreeSelectedKeys = useMemo<Key[]>(
    () => (selectedDeptId ? [selectedDeptId] : EMPTY_SELECTED_KEYS),
    [selectedDeptId]
  );

  // 统计卡占比（禁用/正常 占总数的百分比，total 为 0 时兜底 0）
  const activePct = statistics.total
    ? Math.round((statistics.active / statistics.total) * 1000) / 10
    : 0;

  // 部门数（全树节点数，含根节点 — 原型「部门数」卡）
  const deptCount = useMemo(() => {
    const count = (nodes: typeof departments): number =>
      nodes.reduce((sum, node) => sum + 1 + count(node.children ?? []), 0);
    return count(departments);
  }, [departments]);

  return (
    <>
      {/* 顶部 TabBar 已显示标题，页面内不再重复 PageTitle */}

      {/* 统计卡组（theme.css .stat-card：左 3px 色条 + label/value/trend） */}
      <div className="stat-cards">
        <div className="stat-card">
          <div className="stat-label">用户总数</div>
          <div className="stat-value">{statistics.total}</div>
          <div className="stat-trend">全部门合计</div>
        </div>
        <div className="stat-card sc-green">
          <div className="stat-label">正常用户</div>
          <div className="stat-value">{statistics.active}</div>
          <div className="stat-trend">占比 {activePct}%</div>
        </div>
        <div className="stat-card sc-gray">
          <div className="stat-label">禁用用户</div>
          <div className="stat-value">{statistics.inactive}</div>
          <div className="stat-trend">可随时恢复</div>
        </div>
        <div className="stat-card sc-gold">
          <div className="stat-label">部门数</div>
          <div className="stat-value">{deptCount}</div>
          <div className="stat-trend">左侧部门列表</div>
        </div>
      </div>

      <Layout style={{ display: "flex", alignItems: "stretch", background: "transparent" }}>
        {/* 左侧部门树（原型 250px 白卡） */}
        <Sider
          width={250}
          className="dept-list-sider xr-dept-panel"
          style={{ background: "transparent", padding: "0 14px 16px 0" }}
        >
          <DeptTree onSelect={handleDeptSelect} selectedKeys={deptTreeSelectedKeys} />
        </Sider>

        {/* 右侧内容区 */}
        <Content style={{ padding: 0, background: "transparent" }}>
          <div>
            {/* 搜索表单（原型 .filter-bar：label + 输入框一排；导入/新增按钮在右侧） */}
            <Card style={{ marginBottom: 14 }}>
              <div
                style={{
                  display: "flex",
                  justifyContent: "space-between",
                  alignItems: "flex-start",
                  flexWrap: "wrap",
                  gap: "16px",
                }}
              >
                <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
                  <Form.Item name="username" label="用户名">
                    <AppInput placeholder="用户名 / 昵称 / 手机号" />
                  </Form.Item>
                  <Form.Item name="status" label="状态">
                    {/* 形态分类器②④：用户状态枚举（业务禁搜）。注意原值经 String() 转换 */}
                    <AppSelect
                      placeholder="全部状态"
                      style={{ width: 140 }}
                      allowClear
                      onSearch={() => {}}
                      showSearch={false}
                      options={STATUS_OPTIONS.map((opt) => ({
                        value: String(opt.value),
                        label: opt.label,
                      }))}
                    />
                  </Form.Item>
                  <Form.Item>
                    <Space>
                      <Button onClick={handleReset}>重置</Button>
                      <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                        搜索
                      </Button>
                      <Button icon={<ReloadOutlined />} onClick={handleRefresh} aria-label="刷新" />
                    </Space>
                  </Form.Item>
                </Form>
                <Space>
                  {selectedRowKeys.length > 0 && permRemove.visible && (
                    <Button
                      icon={<DeleteOutlined />}
                      onClick={handleBatchDelete}
                      disabled={permRemove.disabled}
                      title={permRemove.hint}
                      style={{
                        color: "var(--theme-error)",
                      }}
                    >
                      批量删除 ({selectedRowKeys.length})
                    </Button>
                  )}
                  {permImport.visible && (
                    <Button
                      icon={<ImportOutlined />}
                      onClick={() => setImportModalVisible(true)}
                      disabled={permImport.disabled}
                      title={permImport.hint}
                    >
                      导入用户
                    </Button>
                  )}
                  {permAdd.visible && (
                    <Button
                      type="primary"
                      icon={<PlusOutlined />}
                      onClick={handleAddUser}
                      disabled={permAdd.disabled}
                      title={permAdd.hint}
                    >
                      新增用户
                    </Button>
                  )}
                </Space>
              </div>
              {selectedRowKeys.length > 0 && (
                <Alert
                  title={
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

            {/* 用户表格（原型 .grid：绿灰表头 + 斑马纹） */}
            <Card>
              <Table
                columns={columns}
                dataSource={users}
                rowKey="id"
                loading={loading}
                pagination={paginationProps}
                onChange={handleTableChange}
                scroll={{ x: "max-content" }}
                className="xr-table-zebra"
                size="medium"
              />
            </Card>

            {/* 用户导入模态框：复用通用 ExcelImport 组件 */}
            <ExcelImportLazy
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
          onOk={async () => {
            setSubmitting(true);
            try {
              await handleSave();
            } finally {
              setSubmitting(false);
            }
          }}
          cancelButtonProps={{ disabled: submitting }}
          afterOpenChange={handleModalOpenChange}
          onCancel={() => setEditModalVisible(false)}
          width={600}
        >
          <Form
            form={editForm}
            layout="horizontal"
            labelCol={{ span: 4 }}
            wrapperCol={{ span: 20 }}
          >
            <Form.Item
              name="username"
              label="用户名"
              rules={[{ required: true, message: "请输入用户名" }]}
            >
              <AppInput disabled={!!editingUser} className="user-form-input" />
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
              <AppInput className="user-form-input" />
            </Form.Item>
            <Form.Item name="employeeNo" label="工号">
              <AppInput placeholder="请输入工号" className="user-form-input" />
            </Form.Item>
            <Form.Item name="email" label="邮箱">
              <AppInput className="user-form-input" />
            </Form.Item>
            <Form.Item name="phone" label="联系电话">
              <AppInput className="user-form-input" />
            </Form.Item>
            {/* mobile / title 为后端新增字段：手机号与职位（AD 同步写入） */}
            <Form.Item name="mobile" label="手机号">
              <AppInput placeholder="请输入手机号" className="user-form-input" />
            </Form.Item>
            <Form.Item name="title" label="职位">
              {/* 形态分类器③：options 仅有 value 无 label → 显式 optionFilterProp="value"
                  （AppSelect 的 {...rest} 展开在后，覆盖其默认 "label"，过滤行为与迁移前一致） */}
              <AppSelect
                className="user-form-input"
                showSearch
                allowClear
                placeholder="请选择职位"
                options={postTitleOptions}
                optionFilterProp="value"
              />
            </Form.Item>
            <Form.Item name="gender" label="性别" initialValue={2}>
              {/* 形态分类器②③④：性别枚举（业务禁搜）。字典优先、常量兜底的取值逻辑原样保留 */}
              <AppSelect
                allowClear
                className="user-form-input"
                onSearch={() => {}}
                showSearch={false}
                options={
                  genderDict.length > 0
                    ? genderDict.map((d) => ({
                        value: Number(d.dictValue),
                        label: d.dictLabel,
                      }))
                    : GENDER_OPTIONS
                }
              />
            </Form.Item>
            <Form.Item name="deptId" label="部门">
              <DepartmentTreeSelect departments={departments} className="user-form-input" />
            </Form.Item>
            <Form.Item name="roleIds" label="角色">
              {/* 形态分类器③：children 带 label → options 形态（label 真实存在，
                  与原显式 optionFilterProp="label" 同构）；多选需搜索，显式 showSearch */}
              <AppSelect
                mode="multiple"
                placeholder="请选择角色"
                allowClear
                optionFilterProp="label"
                showSearch
                className="user-form-input"
                onSearch={() => {}}
                options={roles.map((role) => ({ value: role.id, label: role.roleName }))}
              />
            </Form.Item>
            <Form.Item name="status" label="状态" initialValue={0}>
              <AppSelect
                allowClear
                className="user-form-input"
                onSearch={() => {}}
                showSearch={false}
                options={STATUS_OPTIONS}
              />
            </Form.Item>
          </Form>
        </Modal>

        {/* 重置密码弹窗 */}
        <Modal
          title={`重置密码 - ${resettingUser?.username || ""}`}
          open={resetPasswordModalVisible}
          onOk={async () => {
            setSubmitting(true);
            try {
              await handleSaveResetPassword();
            } finally {
              setSubmitting(false);
            }
          }}
          cancelButtonProps={{ disabled: submitting }}
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
    </>
  );
};

export default UserManagement;
