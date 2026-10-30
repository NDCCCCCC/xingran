/**
 * 岗位管理 — Phase 66 试点页（PageTable + FormModal 契约样板，UI-SPEC §1/§3/§5）
 *
 * 迁移自 480 行双 hook（useTableManager+usePagination）实现；数据层/搜索区/工具栏/
 * 分页/三态/删除确认全部由 PageTable 内建，本页只保留业务块（统计卡片、已选择
 * Alert、新增/编辑弹窗表单）。
 */
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { FC } from "react";
import { Button, Card, Col, Form, InputNumber, Row, Statistic, Alert } from "antd";
import {
  PlusOutlined,
  EditOutlined,
  CheckCircleOutlined,
  StopOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import type { Key } from "react";
import type { Post } from "@/types";
import { post } from "@/lib/api";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import {
  createStatusColumn,
  createDateTimeColumn,
  createSorterMeta,
} from "@/utils/tableHelpers";
import { PageTable, useActionPermission } from "@/components/shared";
import type { PageTableRenderContext } from "@/components/shared";
import { createDeleteAction } from "@/components/shared/ActionButtons";
import { FormModal } from "@/components/shared/FormModal";
import { useFormModalForm } from "@/components/shared/hooks/useFormModalForm";
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
// Phase 86 W1（RBAC-02）：操作级权限门控 —— 单一真相源 + 三态门控 hook。
import { PERMISSION_CODES } from "@/constants/permissionCodes";

const STATUS_OPTIONS = [
  { label: "正常", value: 0 },
  { label: "停用", value: 1 },
];

const PostManagement: FC = () => {
  const [statistics, setStatistics] = useState({ total: 0, active: 0, inactive: 0 });
  const [modalOpen, setModalOpen] = useState(false);
  const [editingPost, setEditingPost] = useState<Post | null>(null);
  const form = useFormModalForm();
  // PageTable ctx.loadData 提升到提交回调（弹窗 onFinish 成功后刷新列表）
  const reloadRef = useRef<() => void>(() => {});

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
    // 与迁前一致：mount 统计首载（异步 setStatistics 在 promise 回调内，非 effect 同步 setState）
    // eslint-disable-next-line react-hooks/set-state-in-effect
    loadStatistics();
  }, [loadStatistics]);

  const openCreate = () => {
    setEditingPost(null);
    setModalOpen(true);
  };
  const openEdit = (record: Post) => {
    setEditingPost(record);
    setModalOpen(true);
  };
  const closeModal = () => {
    setModalOpen(false);
    setEditingPost(null);
  };

  // 行删（经 createDeleteAction→ConfirmDelete；失败 confirmDelete 内 handleApiError 后 rethrow）
  const handleDelete = async (id: string) => {
    await post(`/system/posts/${id}/delete`, {});
    handleSuccess("删除");
    loadStatistics();
  };

  // FormModal 提交（防重/脏检查/失败保持打开由 FormModal 内建；isFormValidationError 静默分流亦然）
  const handleFinish = async (values: Record<string, unknown>) => {
    if (editingPost) {
      await post(`/system/posts/${editingPost.id}/update`, values);
      handleSuccess("更新");
    } else {
      await post("/system/posts", values);
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
        {total > 10 && (
          <Row gutter={16} style={{ marginBottom: 16 }}>
            <Col span={8}>
              <Card>
                <Statistic title="总岗位数" value={statistics.total} prefix={<CheckCircleOutlined />} />
              </Card>
            </Col>
            <Col span={8}>
              <Card>
                <Statistic
                  title="正常岗位"
                  value={statistics.active}
                  styles={{ content: { color: "var(--theme-success)" } }}
                  prefix={<CheckCircleOutlined />}
                />
              </Card>
            </Col>
            <Col span={8}>
              <Card>
                <Statistic
                  title="停用岗位"
                  value={statistics.inactive}
                  styles={{ content: { color: "var(--theme-error)" } }}
                  prefix={<StopOutlined />}
                />
              </Card>
            </Col>
          </Row>
        )}
        {selectedRowKeys.length > 0 && (
          <Alert
            title={
              <span>
                已选择 <strong>{selectedRowKeys.length}</strong> 个岗位，
                <Button
                  type="link"
                  size="small"
                  onClick={clearSelection}
                  style={{ padding: 0 }}
                >
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
          { name: "status", label: "状态", type: "select", options: STATUS_OPTIONS, placeholder: "请选择状态" },
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
        initialValues={editingPost ? { ...editingPost } : { postSort: 0, status: 0 }}
      >
        <Form.Item
          name="postCode"
          label="岗位编码"
          rules={[{ required: true, message: "请输入岗位编码" }]}
        >
          <AppInput placeholder="请输入岗位编码" className="user-form-input" disabled={!!editingPost} />
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
          <AppSelect placeholder="请选择状态" className="user-form-input" options={STATUS_OPTIONS} />
        </Form.Item>
        <Form.Item name="remark" label="备注">
          <AppTextArea rows={3} placeholder="请输入备注" />
        </Form.Item>
      </FormModal>
    </div>
  );
};

export default PostManagement;
