/**
 * 岗位管理 — Phase 66 试点页（PageTable + FormModal 契约样板，UI-SPEC §1/§3/§5）
 *
 * 迁移自 480 行双 hook（useTableManager+usePagination）实现；数据层/搜索区/工具栏/
 * 分页/三态/删除确认全部由 PageTable 内建，本页只保留业务块（统计卡片、已选择
 * Alert、新增/编辑弹窗表单）。
 */
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { FC } from "react";
import { Button, Form, InputNumber, Alert, Divider } from "antd";
import { PlusOutlined, EditOutlined } from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import type { Key } from "react";
import type { Post } from "@/types";
import { post } from "@/lib/api";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import { createStatusColumn, createDateTimeColumn, createSorterMeta } from "@/utils/tableHelpers";
import { PageTable, useActionPermission, StatCards } from "@/components/shared";
import type { PageTableRenderContext } from "@/components/shared";
import { createDeleteAction } from "@/components/shared/ActionButtons";
import { FormModal } from "@/components/shared/FormModal";
import { useFormModalForm } from "@/components/shared/hooks/useFormModalForm";
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import { AnchorScopeFields } from "@/components/system/AnchorScopeFields";
import { AnchorPreviewPanel } from "@/components/system/AnchorPreviewPanel";
import {
  anchorModeLabel,
  normalizeAnchorMode,
  normalizeAnchorParam,
  retargetAnchorParam,
} from "@/constants/anchor";
import type { SimpleDept } from "@/lib/dutyApi";
import type { AnchorMode } from "@/types/system";
// Phase 86 W1（RBAC-02）：操作级权限门控 —— 单一真相源 + 三态门控 hook。
import { PERMISSION_CODES } from "@/constants/permissionCodes";

const STATUS_OPTIONS = [
  { label: "正常", value: 0 },
  { label: "停用", value: 1 },
];

/** 部门树节点（后端 /system/departments/tree-select 返回形态，兼容两种字段名） */
interface DeptTreeNode {
  id?: string;
  key?: string;
  deptName?: string;
  title?: string;
  value?: string;
  children?: unknown;
}

const PostManagement: FC = () => {
  const [statistics, setStatistics] = useState({ total: 0, active: 0, inactive: 0 });
  const [modalOpen, setModalOpen] = useState(false);
  const [editingPost, setEditingPost] = useState<Post | null>(null);
  const form = useFormModalForm();
  // PageTable ctx.loadData 提升到提交回调（弹窗 onFinish 成功后刷新列表）
  const reloadRef = useRef<() => void>(() => {});

  // 机构锚点（与 dataScope 无关的独立维度；岗位没有 dataScope，只有锚点）
  const [anchorMode, setAnchorMode] = useState<AnchorMode>(0);
  const [anchorParam, setAnchorParam] = useState("");
  /**
   * 部门树数据源（仅 mode=2 的部门选择使用）。
   *
   * 刻意**不**用 `useDeptTree()` hook：那是 React Query hook，会让本页强依赖
   * QueryClientProvider（本页既有渲染测试不注入 provider）。改为直连同一后端端点
   * `/system/departments/tree-select`，失败时静默降级为空树（不影响表单其余部分）。
   */
  const [anchorDepartments, setAnchorDepartments] = useState<SimpleDept[]>([]);
  const [anchorDepartmentsLoading, setAnchorDepartmentsLoading] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setAnchorDepartmentsLoading(true);
    void (async () => {
      try {
        const res = await post<DeptTreeNode[]>("/system/departments/tree-select", {});
        if (cancelled) return;
        const toSimpleDept = (nodes: DeptTreeNode[]): SimpleDept[] =>
          (nodes ?? []).map((node) => ({
            id: node.key || node.id || node.value || "",
            deptName: node.title || node.deptName || "",
            children: node.children ? toSimpleDept(node.children as DeptTreeNode[]) : undefined,
          }));
        setAnchorDepartments(toSimpleDept(Array.isArray(res?.data) ? res.data : []));
      } catch {
        /* 部门树拉取失败：锚点部门下拉为空，不影响保存/列表（后端会做 400 校验兜底） */
      } finally {
        if (!cancelled) setAnchorDepartmentsLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  // Phase 86 W1（RBAC-02）：操作级权限门控（三态 ready/loading/error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  // 工具栏「新增岗位」→ system:post:add；批量删除 → system:post:remove（与行删除同键）。
  const permAdd = permGate.decide(PERMISSION_CODES.systemPost.add);
  const permRemove = permGate.decide(PERMISSION_CODES.systemPost.remove);

  // 服务端排序:field 对应后端 postAllowedSortFields 白名单 key
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<Post>("postCode"),
      createSorterMeta<Post>("postName"),
      createSorterMeta<Post>("postSort"),
      createSorterMeta<Post>("status"),
      createSorterMeta<Post>("remark"),
      createSorterMeta<Post>("createdAt", "date"),
    ],
    []
  );

  const request = useCallback(async (params: Record<string, unknown>) => {
    const result = (await post("/system/posts/list", params)) as {
      data: { list: Post[]; total: number };
    };
    return { list: result.data.list || [], total: result.data.total || 0 };
  }, []);

  // 统计数据(专用 COUNT 端点,不受 MaxPageSize=100 钳制)
  const loadStatistics = useCallback(async () => {
    try {
      const result = await post<{ total: number; active: number; inactive: number }>(
        "/system/posts/statistics"
      );
      setStatistics(result.data ?? { total: 0, active: 0, inactive: 0 });
    } catch (error) {
      handleApiError(error, "加载统计数据", false);
    }
  }, []);

  useEffect(() => {
    // 与迁前一致：mount 统计首载（setStatistics 发生在 promise 回调内，非 effect 同步 setState）
    loadStatistics();
  }, [loadStatistics]);

  const openCreate = () => {
    setEditingPost(null);
    // 新增：锚点归零（不参与），避免沿用上一条编辑记录的残留值
    setAnchorMode(0);
    setAnchorParam("");
    setModalOpen(true);
  };
  const openEdit = (record: Post) => {
    setEditingPost(record);
    // 编辑回显：后端字段为 anchorMode/anchorParam（缺省视为 0/空）
    const mode = normalizeAnchorMode(record.anchorMode);
    setAnchorMode(mode);
    setAnchorParam(normalizeAnchorParam(mode, record.anchorParam));
    setModalOpen(true);
  };
  const closeModal = () => {
    setModalOpen(false);
    setEditingPost(null);
  };

  // 锚点受控变更（成对抛值；切模式时参数已由 AnchorScopeFields 按形状清空）
  const handleAnchorChange = useCallback(({ mode, param }: { mode: AnchorMode; param: string }) => {
    setAnchorMode(mode);
    setAnchorParam(retargetAnchorParam(mode, param));
  }, []);

  // 行删（经 createDeleteAction→ConfirmDelete；失败 confirmDelete 内 handleApiError 后 rethrow）
  const handleDelete = async (id: string) => {
    await post(`/system/posts/${id}/delete`, {});
    handleSuccess("删除");
    loadStatistics();
  };

  // FormModal 提交（防重/脏检查/失败保持打开由 FormModal 内建；isFormValidationError 静默分流亦然）
  const handleFinish = async (values: Record<string, unknown>) => {
    // 机构锚点成对写入（与 dataScope 无关的独立维度；模式 0/3 强制空串参数）
    const payload = {
      ...values,
      anchorMode,
      anchorParam: retargetAnchorParam(anchorMode, anchorParam),
    };
    if (editingPost) {
      await post(`/system/posts/${editingPost.id}/update`, payload);
      handleSuccess("更新");
    } else {
      await post("/system/posts", payload);
      handleSuccess("创建");
    }
    closeModal();
    loadStatistics();
    reloadRef.current();
  };

  const columns = useMemo<ColumnsType<Post>>(
    () => [
      { title: "岗位编码", dataIndex: "postCode", key: "postCode", width: 150, sorter: true },
      { title: "岗位名称", dataIndex: "postName", key: "postName", width: 150, sorter: true },
      { title: "显示排序", dataIndex: "postSort", key: "postSort", width: 100, sorter: true },
      {
        title: "机构授权",
        dataIndex: "anchorMode",
        key: "anchorMode",
        width: 180,
        // 与「数据范围」无关的独立维度：只展示岗位自身的锚点模式（0=不参与）
        render: (_: unknown, record: Post) => {
          const mode = normalizeAnchorMode(record.anchorMode);
          if (mode === 0) {
            return <span style={{ color: "var(--theme-text-tertiary)" }}>不参与</span>;
          }
          return <span className="xr-tag xr-tag-green">{anchorModeLabel(mode)}</span>;
        },
      },
      createStatusColumn("status", { width: 100, sorter: true }),
      { title: "备注", dataIndex: "remark", key: "remark", ellipsis: true, sorter: true },
      createDateTimeColumn("createdAt", { width: 180, sorter: true }),
    ],
    []
  );

  // 业务区：统计卡片（total>10 才显示）+ 已选择 Alert（行为与迁前一致）
  const renderBusinessArea = ({
    selectedRowKeys,
    clearSelection,
    total,
    loadData,
  }: PageTableRenderContext<Post>) => {
    reloadRef.current = loadData;
    return (
      <>
        <StatCards
          show={total > 10}
          items={[
            { label: "总岗位数", value: statistics.total, tone: "default" },
            { label: "正常岗位", value: statistics.active, tone: "green" },
            { label: "停用岗位", value: statistics.inactive, tone: "danger" },
          ]}
        />
        {selectedRowKeys.length > 0 && (
          <Alert
            title={
              <span>
                已选择 <strong>{selectedRowKeys.length}</strong> 个岗位，
                <Button type="link" size="small" onClick={clearSelection} style={{ padding: 0 }}>
                  取消选择
                </Button>
              </span>
            }
            type="info"
            showIcon
            style={{ marginBottom: 16 }}
          />
        )}
      </>
    );
  };

  return (
    <div>
      <PageTable<Post>
        request={request}
        columns={columns}
        rowKey={(record) => record.id}
        searchFields={[
          { name: "postCode", label: "岗位编码", type: "input", placeholder: "请输入岗位编码" },
          { name: "postName", label: "岗位名称", type: "input", placeholder: "请输入岗位名称" },
          {
            name: "status",
            label: "状态",
            type: "select",
            options: STATUS_OPTIONS,
            placeholder: "请选择状态",
          },
        ]}
        sorterMetas={sorterMetas}
        rowActions={(record) => [
          {
            key: "edit",
            label: "编辑",
            icon: <EditOutlined />,
            onClick: () => openEdit(record),
            // Phase 86 W1（RBAC-02）：行编辑 → system:post:edit（ActionButtons 内建三态门控）。
            permission: PERMISSION_CODES.systemPost.edit,
          },
          createDeleteAction(record.id, handleDelete, {
            target: record.postName,
            // 行删除 → system:post:remove。
            permission: PERMISSION_CODES.systemPost.remove,
          }),
        ]}
        // Phase 86 W1：批量删除走页面层条件传参（PageTable.batchDelete 契约无 permission 位，
        // 不改 shared 组件契约）—— ready 无权限 → 不传 batchDelete（按钮不入 DOM）。
        batchDelete={
          permRemove.visible
            ? {
                onDelete: async (keys: Key[]) => {
                  await post("/system/posts/batch-delete", { ids: keys });
                  loadStatistics();
                },
                target: "岗位",
              }
            : undefined
        }
        toolbar={
          // Phase 86 W1：工具栏新增按权限码条件渲染（ready 无权限 → 不渲染；未就绪 → 渲染但 disabled + 提示）。
          permAdd.visible ? (
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={openCreate}
              disabled={permAdd.disabled}
              title={permAdd.hint}
            >
              新增岗位
            </Button>
          ) : undefined
        }
      >
        {renderBusinessArea}
      </PageTable>

      <FormModal
        open={modalOpen}
        title={editingPost ? "编辑岗位" : "新增岗位"}
        onFinish={handleFinish}
        onCancel={closeModal}
        form={form}
        width={600}
        initialValues={
          editingPost
            ? { ...editingPost, anchorMode: normalizeAnchorMode(editingPost.anchorMode) }
            : { postSort: 0, status: 0, anchorMode: 0 }
        }
      >
        <Form.Item
          name="postCode"
          label="岗位编码"
          rules={[{ required: true, message: "请输入岗位编码" }]}
        >
          <AppInput
            placeholder="请输入岗位编码"
            className="user-form-input"
            disabled={!!editingPost}
          />
        </Form.Item>
        <Form.Item
          name="postName"
          label="岗位名称"
          rules={[{ required: true, message: "请输入岗位名称" }]}
        >
          <AppInput placeholder="请输入岗位名称" className="user-form-input" />
        </Form.Item>
        <Form.Item
          name="postSort"
          label="显示排序"
          rules={[{ required: true, message: "请输入显示排序" }]}
        >
          <InputNumber min={0} style={{ width: "100%" }} placeholder="请输入显示排序" />
        </Form.Item>
        <Form.Item name="status" label="状态" rules={[{ required: true, message: "请选择状态" }]}>
          <AppSelect
            placeholder="请选择状态"
            className="user-form-input"
            options={STATUS_OPTIONS}
          />
        </Form.Item>
        <Form.Item name="remark" label="备注">
          <AppTextArea rows={3} placeholder="请输入备注" />
        </Form.Item>

        {/* 机构授权（可选）：与「数据范围」无关的独立维度，岗位不参与 MAX(data_scope) 排序 */}
        <Divider titlePlacement="start" plain>
          机构授权（可选）
        </Divider>
        <AnchorScopeFields
          mode={anchorMode}
          param={anchorParam}
          onChange={handleAnchorChange}
          departments={anchorDepartments}
          departmentsLoading={anchorDepartmentsLoading}
        />
        <AnchorPreviewPanel
          kind="post"
          targetId={editingPost?.id}
          mode={anchorMode}
          param={anchorParam}
        />
      </FormModal>
    </div>
  );
};

export default PostManagement;
