/**
 * ManagedOrgDrawer — 条线部门管理归属机构映射 Drawer（权限数据范围域）
 *
 * 后端复用 sys_dept_location_alias 表 scope='permission' 行，API 契约：
 * - POST /ops/dept-managed-org/list {pageNum, pageSize}
 * - POST /ops/dept-managed-org {deptId, orgId, remark}
 * - POST /ops/dept-managed-org/:id/delete
 *
 * 挂载于部门管理页工具栏「归属映射」按钮（权限 sys:dept:managed-org:list 控制可见）。
 * 部门树数据由父级（useDeptData.departments，含 orgLevel）传入。
 *
 * 设计要点：
 * - 2 个独立 TreeSelect：条线部门（全量部门树）+ 管理归属机构（仅 orgLevel∈{1,2,3}
 *   的机构节点：分公司/中支/支公司，前端过滤 + 后端校验兜底）
 * - 权限 gating 用 useActionPermission 三态门控（对齐 LocationAliasDrawer 惯例）：
 *   - add 缺失 → 新增按钮不渲染；未就绪 → 渲染但 disabled
 *   - delete 缺失 → 操作列删除按钮不渲染；未就绪 → 渲染但 disabled
 * - 写成功后 invalidate managedOrg + dept（部门树用于回显归属机构名）
 */

import { useCallback, useMemo, useState } from "react";
import { confirmDelete } from "@/components/shared/confirmDelete";
import { Drawer, Form, TreeSelect, Button, Table, Space, Tooltip, Typography } from "antd";
import { PlusOutlined, DeleteOutlined } from "@ant-design/icons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { managedOrgApi, type ManagedOrg } from "@/lib/opsApi";
import { queryKeys } from "@/lib/queryKeys";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import { useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import type { Department } from "@/types";
import { AppTextArea } from "@/components/ui";

const { Text } = Typography;

/** 机构候选层级：1=分公司 2=中支 3=支公司（后端 org_level∈{1,2,3} 校验同源） */
const MANAGED_ORG_LEVELS = new Set([1, 2, 3]);

interface OrgTreeNode {
  title: string;
  value: string;
  key: string;
  orgLevel?: number;
  children?: OrgTreeNode[];
}

/** Department 树 → TreeSelect 树（title/value/key + orgLevel 透传） */
function toOrgTreeData(nodes: Department[]): OrgTreeNode[] {
  return (nodes ?? []).map((node) => ({
    title: node.deptName,
    value: node.id,
    key: node.id,
    orgLevel: node.orgLevel,
    children:
      node.children && node.children.length > 0 ? toOrgTreeData(node.children) : undefined,
  }));
}

/**
 * 仅保留 orgLevel∈{1,2,3} 的机构节点及其祖先路径（与 deptUtils.filterExternalOrgDepts
 * 同语义：匹配节点保留全部后代；非匹配但有匹配后代 → 保留为祖先路径；否则丢弃）。
 */
function filterManagedOrgTree(nodes: OrgTreeNode[]): OrgTreeNode[] {
  const walk = (list: OrgTreeNode[]): OrgTreeNode[] =>
    list.reduce<OrgTreeNode[]>((acc, node) => {
      const keptChildren = node.children?.length ? walk(node.children) : [];
      if (MANAGED_ORG_LEVELS.has(node.orgLevel ?? 0) || keptChildren.length > 0) {
        acc.push({
          ...node,
          children: keptChildren.length > 0 ? keptChildren : undefined,
        });
      }
      return acc;
    }, []);
  return walk(nodes ?? []);
}

export interface ManagedOrgDrawerProps {
  open: boolean;
  onClose: () => void;
  /** 部门树（父级 useDeptData.departments，含 orgLevel），用于两个 TreeSelect */
  departments: Department[];
}

export function ManagedOrgDrawer({ open, onClose, departments }: ManagedOrgDrawerProps) {
  const [form] = Form.useForm();
  const [pageNum, setPageNum] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [showAddForm, setShowAddForm] = useState(false);

  // 权限三态门控（单一真相源 permissionCodes + useActionPermission）
  const permGate = useActionPermission();
  const permAdd = permGate.decide(PERMISSION_CODES.sysDeptManagedOrg.add);
  const permDelete = permGate.decide(PERMISSION_CODES.sysDeptManagedOrg.delete);

  // 派生：全量部门树（条线部门候选）+ 仅机构子树（管理归属机构候选）
  const fullDeptTreeData = useMemo(() => toOrgTreeData(departments ?? []), [departments]);
  const orgTreeData = useMemo(() => filterManagedOrgTree(fullDeptTreeData), [fullDeptTreeData]);

  // 数据：映射分页列表
  const {
    data: pageData,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: queryKeys.managedOrg.list({ pageNum, pageSize }),
    queryFn: async () => {
      const res = await managedOrgApi.list({ pageNum, pageSize });
      const payload = res.data ?? { list: [], total: 0, current: 1, pageSize: 10 };
      return payload;
    },
  });

  // 写成功后：刷新列表 + 失效 managedOrg（双保险）
  const queryClient = useQueryClient();
  const invalidateManagedOrgAll = useCallback(
    () => queryClient.invalidateQueries({ queryKey: queryKeys.managedOrg.all }),
    [queryClient]
  );

  const refreshAfterMutation = useCallback(async () => {
    await refetch();
    invalidateManagedOrgAll();
  }, [refetch, invalidateManagedOrgAll]);

  const handleCreate = async () => {
    try {
      const values = (await form.validateFields()) as {
        deptId: string;
        orgId: string;
        remark?: string;
      };
      await managedOrgApi.create({
        deptId: values.deptId,
        orgId: values.orgId,
        remark: values.remark,
      });
      handleSuccess("归属映射创建");
      form.resetFields();
      setShowAddForm(false);
      await refreshAfterMutation();
    } catch (err) {
      // 表单校验错误不弹错误消息（showMessage=false）；后端错误（如「该条线部门已配置
      // 管理归属机构」）由 handleApiError 统一反馈
      handleApiError(err, "创建归属映射", false);
    }
  };

  const handleDelete = useCallback(
    async (id: string) => {
      await managedOrgApi.delete(id);
      await refreshAfterMutation();
    },
    [refreshAfterMutation]
  );

  const list: ManagedOrg[] = pageData?.list ?? [];
  const total: number = pageData?.total ?? 0;

  const columns = useMemo(
    () => [
      {
        title: "条线部门",
        key: "dept",
        ellipsis: true,
        render: (_: unknown, r: ManagedOrg) =>
          r.originDeptName ? <Text>{r.originDeptName}</Text> : <Text code>{r.deptId}</Text>,
      },
      {
        title: "管理归属机构",
        key: "org",
        ellipsis: true,
        render: (_: unknown, r: ManagedOrg) =>
          r.locationDeptName ? <Text>{r.locationDeptName}</Text> : <Text code>{r.locationId}</Text>,
      },
      {
        title: "备注",
        dataIndex: "remark",
        key: "remark",
        ellipsis: true,
      },
      {
        title: "创建时间",
        dataIndex: "createdAt",
        key: "createdAt",
        width: 160,
      },
      {
        title: "操作",
        key: "action",
        width: 80,
        render: (_: unknown, record: ManagedOrg) =>
          // 三态：ready 无 sysDeptManagedOrg.delete → 不渲染；未就绪 → 渲染但 disabled + hint
          permDelete.visible ? (
            <Tooltip title={permDelete.hint ?? "删除归属映射"}>
              <Button
                type="link"
                danger
                icon={<DeleteOutlined />}
                size="small"
                disabled={permDelete.disabled}
                aria-label={`删除 归属映射（${record.originDeptName || record.deptId}）`}
                title={
                  permDelete.hint ?? `删除 归属映射（${record.originDeptName || record.deptId}）`
                }
                onClick={() =>
                  confirmDelete({
                    target: `该条线部门归属映射（${record.originDeptName || record.deptId}）`,
                    // 业务语义说明，不得丢失
                    extra: "删除后，该条线部门的权限数据范围将不再收口到该归属机构。",
                    onDelete: () => handleDelete(record.id),
                  })
                }
              />
            </Tooltip>
          ) : null,
      },
    ],
    [permDelete, handleDelete]
  );

  return (
    <Drawer
      title="条线部门管理归属映射"
      open={open}
      onClose={onClose}
      size="large"
      destroyOnHidden
      mask={{ closable: true }}
    >
      <Space orientation="vertical" size="medium" style={{ width: "100%" }}>
        {/* 新增映射入口 + 折叠表单 */}
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
          <Text strong>映射列表({total} 条)</Text>
          {permAdd.visible && (
            <Tooltip title={permAdd.hint ?? ""}>
              <Button
                type="primary"
                icon={<PlusOutlined />}
                disabled={permAdd.disabled}
                onClick={() => setShowAddForm((v) => !v)}
              >
                {showAddForm ? "收起新增" : "新增映射"}
              </Button>
            </Tooltip>
          )}
        </div>

        {showAddForm && (
          <Form form={form} layout="vertical">
            <Form.Item
              name="deptId"
              label="条线部门(全量)"
              rules={[{ required: true, message: "请选择条线部门" }]}
            >
              <TreeSelect
                treeData={fullDeptTreeData}
                treeDefaultExpandAll={false}
                placeholder="请选择条线部门(系统内任意部门)"
                showSearch
                treeNodeFilterProp="title"
                allowClear
              />
            </Form.Item>
            <Form.Item
              name="orgId"
              label="管理归属机构"
              rules={[{ required: true, message: "请选择管理归属机构" }]}
              extra="仅可选择机构节点(orgLevel 1=分公司 / 2=中支 / 3=支公司)"
            >
              <TreeSelect
                treeData={orgTreeData}
                treeDefaultExpandAll={false}
                placeholder="请选择管理归属机构"
                showSearch
                treeNodeFilterProp="title"
                allowClear
                disabled={orgTreeData.length === 0}
              />
            </Form.Item>
            <Form.Item name="remark" label="备注">
              <AppTextArea rows={2} placeholder="选填" />
            </Form.Item>
            <Space>
              <Button type="primary" onClick={handleCreate}>
                提交
              </Button>
              <Button
                onClick={() => {
                  form.resetFields();
                  setShowAddForm(false);
                }}
              >
                取消
              </Button>
            </Space>
          </Form>
        )}

        <Table
          rowKey="id"
          size="small"
          columns={columns}
          dataSource={list}
          loading={isLoading}
          pagination={{
            current: pageNum,
            pageSize,
            total,
            showSizeChanger: true,
            onChange: (page, size) => {
              setPageNum(page);
              setPageSize(size);
            },
          }}
        />
      </Space>
    </Drawer>
  );
}

export default ManagedOrgDrawer;
