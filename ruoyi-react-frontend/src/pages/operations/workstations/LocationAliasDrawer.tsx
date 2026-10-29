/**
 * LocationAliasDrawer (Phase 39-07)
 *
 * 工位部门物理位置映射管理 Drawer — 承载 alias 完整 CRUD(D-02/D-08 决策)。
 * 在工位列表页工具栏 [⚙ 映射] 按钮中打开,不新增菜单项(D-02)。
 *
 * 设计要点:
 * - Drawer width = 600(D-02 Claude's discretion)
 * - 2 个独立 TreeSelect:dept(全量部门树) + location(仅外部机构)
 * - scope 字段在 UI 隐藏(后端默认 "workstation",前端兜底)
 * - 权限 gating 严格对齐 D-08（Phase 86 W2 已迁为 useActionPermission 三态门控）:
 *   - opsLocationAlias.list 缺失 → 由 index.tsx 控制 [⚙ 映射] 按钮不渲染（本组件默认信任父级 gating）
 *   - opsLocationAlias.add 缺失 → "新增映射"按钮**不渲染**；未就绪 → 渲染但 disabled + 提示
 *   - opsLocationAlias.delete 缺失 → 操作列删除按钮**不渲染**；未就绪 → 渲染但 disabled + 提示
 * - 写操作成功后调 useInvalidateDept + invalidate locationAlias(双失效)
 */

import { useCallback, useMemo, useState } from "react";
import { confirmDelete } from "@/components/shared/confirmDelete";
import {
  Drawer,
  Form,
  TreeSelect,
  Button,
  Table,
  Space,
  Tooltip,
  Typography,
} from "antd";
import { PlusOutlined, DeleteOutlined } from "@ant-design/icons";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { locationAliasApi, type LocationAlias } from "@/lib/opsApi";
import { queryKeys } from "@/lib/queryKeys";
import { useDeptTree, useInvalidateDept } from "@/hooks/useDeptTree";
import {
  dedupTreeByKey,
  filterExternalOrgDepts,
  toFullPathTree,
  trimTitleToLastSegment,
} from "@/utils/deptUtils";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import { useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import type { DeptTreeNode } from "./types";
// Phase 71 (W3)：裸 Input/Input.TextArea → AppInput/AppTextArea。
// scope 为隐藏字段（恒有初值，不受清空影响）；remark 为可选备注输入。
import { AppInput, AppTextArea } from "@/components/ui";

const { Text } = Typography;

export interface LocationAliasDrawerProps {
  open: boolean;
  onClose: () => void;
}

export function LocationAliasDrawer({ open, onClose }: LocationAliasDrawerProps) {
  const [form] = Form.useForm();
  const [pageNum, setPageNum] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [showAddForm, setShowAddForm] = useState(false);

  // 权限(Phase 86 W2 / D-08)：旧式判定（menuStore 权限集直读 + 自造 hasPermission + 硬编码字面量）
  // 已迁为 useActionPermission 三态门控 —— 消除第二真相源，并拿到「未就绪 → 渲染但 disabled」
  // 的语义。权限码来自单一真相源 src/constants/permissionCodes.ts。
  // ⚠️ 行为变更登记：旧写法在未就绪时隐藏入口；新写法渲染但 disabled + hint（收紧到更安全一侧）。
  const permGate = useActionPermission();
  const permAdd = permGate.decide(PERMISSION_CODES.opsLocationAlias.add);
  const permDelete = permGate.decide(PERMISSION_CODES.opsLocationAlias.delete);

  // 数据:部门树(SimpleDept 形状,与 DeptTreeNode 结构兼容 — title/value/key + isExternalOrg)
  const { data: deptTreeData = [] } = useDeptTree();

  // 数据:alias 分页列表
  const {
    data: aliasPage,
    isLoading,
    refetch,
  } = useQuery({
    queryKey: queryKeys.locationAlias.list({ pageNum, pageSize }),
    queryFn: async () => {
      const res = await locationAliasApi.list({ pageNum, pageSize });
      const payload = res.data ?? { list: [], total: 0, current: 1, pageSize: 10 };
      return payload;
    },
  });

  // 派生:全量部门树 + 仅外部机构子树
  // 三步组合:toFullPathTree 把 SimpleDept{id,deptName} 转成 TreeSelect 需要的
  // {title, value:id, key:id, isExternalOrg, children}(修复 value=undefined);
  // trimTitleToLastSegment 把全路径 title 裁成末段(只显示当前部门名);
  // dedupTreeByKey 兜底去重(后端 buildDeptTree 会把 Ancestors 为空但有 ParentID
  // 的部门提升到根 + 嵌套父节点下,导致同一部门重复 → antd "Same key" 告警)。
  const fullDeptTreeData = useMemo(
    () =>
      dedupTreeByKey(
        trimTitleToLastSegment(toFullPathTree(deptTreeData ?? []))
      ) as unknown as DeptTreeNode[],
    [deptTreeData]
  );
  const locationTreeData: DeptTreeNode[] = useMemo(
    () =>
      dedupTreeByKey(
        filterExternalOrgDepts<DeptTreeNode>(fullDeptTreeData)
      ) as unknown as DeptTreeNode[],
    [fullDeptTreeData]
  );

  // 仅展开第一层(根节点),避免全树展开导致下拉太长难找部门。
  const deptRootKeys = useMemo(() => fullDeptTreeData.map((n) => n.value), [fullDeptTreeData]);
  const locationRootKeys = useMemo(() => locationTreeData.map((n) => n.value), [locationTreeData]);

  // 双失效工具
  const invalidateDept = useInvalidateDept();
  const queryClient = useQueryClient();
  const invalidateAliasAll = useCallback(
    () => queryClient.invalidateQueries({ queryKey: queryKeys.locationAlias.all }),
    [queryClient]
  );

  const refreshAfterMutation = useCallback(async () => {
    await refetch();
    invalidateDept();
    invalidateAliasAll();
  }, [refetch, invalidateDept, invalidateAliasAll]);

  const handleCreate = async () => {
    try {
      const values = (await form.validateFields()) as {
        deptId: string;
        locationId: string;
        scope?: string;
        remark?: string;
      };
      await locationAliasApi.create({
        deptId: values.deptId,
        locationId: values.locationId,
        scope: values.scope ?? "workstation",
        remark: values.remark,
      });
      handleSuccess("映射创建");
      form.resetFields();
      setShowAddForm(false);
      await refreshAfterMutation();
    } catch (err) {
      // 表单校验错误不需要弹错误消息(showMessage=false);后端错误需要
      handleApiError(err, "创建映射", false);
    }
  };

  // 68.1-T3：删除确认改走 confirmDelete，失败上抛由 helper 统一反馈
  const handleDelete = useCallback(
    async (id: string) => {
      await locationAliasApi.delete(id);
      await refreshAfterMutation();
    },
    [refreshAfterMutation]
  );

  const aliasList: LocationAlias[] = aliasPage?.list ?? [];
  const total: number = aliasPage?.total ?? 0;

  const columns = useMemo(
    () => [
      {
        title: "所属部门",
        key: "dept",
        ellipsis: true,
        render: (_: unknown, r: LocationAlias) =>
          r.originDeptName ? <Text>{r.originDeptName}</Text> : <Text code>{r.deptId}</Text>,
      },
      {
        title: "物理位置",
        key: "location",
        ellipsis: true,
        render: (_: unknown, r: LocationAlias) =>
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
        render: (_: unknown, record: LocationAlias) =>
          // 三态：ready 无 opsLocationAlias.delete → 不渲染；未就绪 → 渲染但 disabled + hint
          permDelete.visible ? (
            <Tooltip title={permDelete.hint ?? "删除映射"}>
              <Button
                type="link"
                danger
                icon={<DeleteOutlined />}
                size="small"
                disabled={permDelete.disabled}
                aria-label={`删除 位置别名映射（${record.originDeptName || record.deptId}）`}
                title={permDelete.hint ?? `删除 位置别名映射（${record.originDeptName || record.deptId}）`}
                onClick={() =>
                  confirmDelete({
                    target: `该工位位置别名映射（${record.originDeptName || record.deptId}）`,
                    // 业务语义说明，不得丢失
                    extra: "删除后,工位编辑下拉将立即不再追加该 alias 部门。",
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
      title="工位部门物理位置映射管理"
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
          {/* 三态：ready 无 opsLocationAlias.add → 不渲染；未就绪 → 渲染但 disabled + hint */}
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
          <Form form={form} layout="vertical" initialValues={{ scope: "workstation" }}>
            <Form.Item
              name="deptId"
              label="所属部门(全量)"
              rules={[{ required: true, message: "请选择所属部门" }]}
            >
              <TreeSelect
                treeData={fullDeptTreeData}
                treeDefaultExpandedKeys={deptRootKeys}
                placeholder="请选择部门(系统内任意部门)"
                showSearch
                treeNodeFilterProp="title"
                allowClear
              />
            </Form.Item>
            <Form.Item
              name="locationId"
              label="物理位置(仅外部机构)"
              rules={[{ required: true, message: "请选择物理位置" }]}
              extra="仅可选择 isExternalOrg=1 的外部机构子树下的节点"
            >
              <TreeSelect
                treeData={locationTreeData}
                treeDefaultExpandedKeys={locationRootKeys}
                placeholder="请选择物理位置(外部机构)"
                showSearch
                treeNodeFilterProp="title"
                allowClear
                disabled={locationTreeData.length === 0}
              />
            </Form.Item>
            {/* scope 隐藏字段(后端默认 workstation,前端兜底) */}
            <Form.Item name="scope" hidden>
              <AppInput />
            </Form.Item>
            <Form.Item name="remark" label="备注">
              <AppTextArea rows={2} placeholder="选填,如:分公司本部 4F 工位映射" />
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
          dataSource={aliasList}
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

export default LocationAliasDrawer;
