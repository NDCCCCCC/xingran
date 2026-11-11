/**
 * 经理同步设置区块（系统设置 · 平铺分类之一）
 *
 * 承载原两个独立菜单页，合并为设置页的一个分类区块：
 *   - 机构定义（原 src/pages/system/manager-orgs/index.tsx）
 *   - 同步规则（原 src/pages/system/manager-rules/index.tsx）
 *
 * 结构：
 *   - 一个 antd Tabs，两个 Tab 各自是一个局部子组件（本文件内定义，不额外拆文件）；
 *   - **机构全量列表提升到本组件顶层，只请求一次**（pageSize=1000）：
 *       机构 Tab 的写操作（新增/编辑/删除/从部门生成）成功后回调顶层重新拉取，
 *       再通过 props 下发规则 Tab 的机构下拉与机构名称渲染，
 *       替换原 manager-rules 自己另发一次全量请求的做法；
 *   - 区块内不放 StatCards 统计卡（设置页要求扁平），表格分页沿用 usePagination。
 *
 * 权限门控（三态语义，单一判定入口 useActionPermission().decide(perm)）：
 *   - 无 ops:manager:rule:list → 不渲染「同步规则」Tab；无 ops:manager:org:list → 不渲染「机构定义」Tab；
 *     两者都不可见 → 整个组件返回 null（未就绪时按三态语义仍渲染但禁用）。
 *   - 新增/编辑/从部门生成 → ops:manager:org:edit / ops:manager:rule:edit（码组无 add 键，就近归 edit）
 *   - 删除 → ops:manager:org:delete / ops:manager:rule:delete
 */

import { useState, useEffect, useCallback, useMemo } from "react";
import type { FC } from "react";
import {
  Card,
  Table,
  Button,
  Space,
  Form,
  InputNumber,
  Radio,
  Switch,
  TreeSelect,
  Modal,
  Tag,
  Tabs,
} from "antd";
import type { TabsProps } from "antd";
// Phase 71 (W2)：裸 Input/Select → 契约组件（Select 全为 options 形态）
import { AppInput, AppSelect } from "@/components/ui";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  ThunderboltOutlined,
} from "@ant-design/icons";
import {
  getManagerOrgList,
  createManagerOrg,
  updateManagerOrg,
  deleteManagerOrg,
  generateManagerOrgsFromDept,
  getManagerRuleList,
  createManagerRule,
  updateManagerRule,
  deleteManagerRule,
  MANAGER_ORG_TYPE,
  MANAGER_ORG_TYPE_LABELS,
  MANAGER_ORG_TARGET_PARENT,
  MANAGER_SCOPE_TYPE,
  MANAGER_SCOPE_LABELS,
  MANAGER_SUBJECT_LABELS,
  MANAGER_TYPE_LABELS,
  isManagerEnabled,
  type ManagerOrg,
  type ManagerOrgQuery,
  type ManagerOrgPayload,
  type ManagerRule,
  type ManagerRuleQuery,
  type ManagerRulePayload,
  type ManagerSubjectType,
  type ManagerType,
} from "@/lib/managerApi";
import type { SimpleDept } from "@/lib/dutyApi";
import { useDeptTree } from "@/hooks/useDeptTree";
// P66 lint 规则要求新列表页统一走 PageTable；但本区块是 v1.56 设置页平铺分类，
// 任务要求**行为等价迁移**原 manager-orgs / manager-rules 的表格（列、工具栏、
// Modal、二次确认、分页逻辑一体），改 PageTable 会改变既有表格结构，
// 故与同属设置页的 backup/components/BackupListTab.tsx 一致，显式行级豁免。
// eslint-disable-next-line no-restricted-imports
import { usePagination } from "@/hooks/usePagination";
import { post } from "@/lib/api";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";
import { confirmDelete, useActionPermission } from "@/components/shared";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

/** 岗位选项（规则表单的主体岗位 / 经理岗位共用） */
interface PostItem {
  id: string;
  postName: string;
}

/** 目标机构选择模式 */
type ManagerOrgMode = "self" | "parent" | "specified";

const SUBJECT_TYPES = Object.keys(MANAGER_SUBJECT_LABELS) as ManagerSubjectType[];
const MANAGER_TYPES = Object.keys(MANAGER_TYPE_LABELS) as ManagerType[];

/** 部门树转 TreeSelect 数据（机构 Tab：编辑时需禁用自身及后代节点） */
function toDeptTreeData(
  depts: SimpleDept[],
  disabledIds: ReadonlySet<string>
): {
  title: string;
  value: string;
  key: string;
  disabled: boolean;
  children?: ReturnType<typeof toDeptTreeData>;
}[] {
  return depts.map((d) => ({
    title: d.deptName,
    value: d.id,
    key: d.id,
    disabled: disabledIds.has(d.id),
    children: d.children ? toDeptTreeData(d.children, disabledIds) : undefined,
  }));
}

/** 收集某节点及其全部后代 id */
function collectSelfAndDescendants(depts: SimpleDept[], rootId: string, acc: Set<string>): void {
  for (const d of depts) {
    if (d.id === rootId) {
      const collect = (node: SimpleDept): void => {
        acc.add(node.id);
        node.children?.forEach(collect);
      };
      collect(d);
      return;
    }
    if (d.children) collectSelfAndDescendants(d.children, rootId, acc);
  }
}

/** 部门树转 TreeSelect 数据（规则 Tab：不做禁用，保持原 manager-rules 行为） */
function toPlainDeptTreeData(depts: SimpleDept[]): {
  title: string;
  value: string;
  key: string;
  children?: ReturnType<typeof toPlainDeptTreeData>;
}[] {
  return depts.map((d) => ({
    title: d.deptName,
    value: d.id,
    key: d.id,
    children: d.children ? toPlainDeptTreeData(d.children) : undefined,
  }));
}

// ==================== 机构定义 Tab ====================

interface ManagerOrgsTabProps {
  /** 机构发生写变更（新增/编辑/删除/生成）后回调，用于刷新顶层的共享全量机构数据 */
  onOrgsChanged: () => void;
}

const ManagerOrgsTab: FC<ManagerOrgsTabProps> = ({ onOrgsChanged }) => {
  const [list, setList] = useState<ManagerOrg[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingOrg, setEditingOrg] = useState<ManagerOrg | null>(null);
  const [saving, setSaving] = useState(false);
  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();

  // 操作级权限门控（三态 ready/loading/error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  // 归码说明（opsManagerOrg 码组只有 list/edit/delete，**无 add / generate 键**）：
  //   · 删除机构 → ops:manager:org:delete
  //   · 新增机构 / 从部门架构生成机构 / 行编辑 / 启停 → ops:manager:org:edit
  //     （写动作就近归最接近的既有键，不自造新码 —— Phase 86 零新增权限码）
  const permEdit = permGate.decide(PERMISSION_CODES.opsManagerOrg.edit);
  const permDelete = permGate.decide(PERMISSION_CODES.opsManagerOrg.delete);

  const { data: deptTree = [] } = useDeptTree();

  // 分页沿用全局 hook（current/pageSize 持久化、total 运行时写入）
  const { current, pageSize, setCurrent, setTotal, paginationProps } = usePagination();

  const [searchOrgName, setSearchOrgName] = useState<string | undefined>(undefined);

  // useMemo 包住请求参数，防止对象引用不稳定造成无限请求循环
  const params = useMemo<ManagerOrgQuery>(
    () => ({ current, pageSize, orgName: searchOrgName }),
    [current, pageSize, searchOrgName]
  );

  const loadList = useCallback(async () => {
    setLoading(true);
    try {
      const result = await getManagerOrgList(params);
      setList(result.list ?? []);
      setTotal(result.total ?? 0);
    } catch (error) {
      handleApiError(error, "加载机构列表");
    } finally {
      setLoading(false);
    }
  }, [params, setTotal]);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  // 父机构 TreeSelect：编辑时禁止选自己/自己的后代
  const parentTreeData = useMemo(() => {
    const disabled = new Set<string>();
    if (editingOrg) collectSelfAndDescendants(deptTree, editingOrg.deptId, disabled);
    return toDeptTreeData(deptTree, disabled);
  }, [deptTree, editingOrg]);

  const deptTreeData = useMemo(() => toDeptTreeData(deptTree, new Set()), [deptTree]);

  const openModal = (record?: ManagerOrg) => {
    setEditingOrg(record ?? null);
    setModalOpen(true);
    if (record) {
      editForm.setFieldsValue({
        orgName: record.orgName,
        deptId: record.deptId,
        parentOrgId: record.parentOrgId,
        orgType: record.orgType,
        enabled: isManagerEnabled(record.enabled),
      });
    } else {
      editForm.setFieldsValue({
        orgName: undefined,
        deptId: undefined,
        parentOrgId: undefined,
        orgType: MANAGER_ORG_TYPE.MIDDLE_BRANCH,
        enabled: true,
      });
    }
  };

  const handleModalClose = () => {
    setModalOpen(false);
    setEditingOrg(null);
    editForm.resetFields();
  };

  const handleSubmit = async () => {
    try {
      const values = (await editForm.validateFields()) as ManagerOrgPayload;
      setSaving(true);
      const payload: ManagerOrgPayload = {
        ...values,
        parentOrgId: values.parentOrgId || undefined,
      };
      if (editingOrg) {
        await updateManagerOrg(editingOrg.id, payload);
        handleSuccess("更新");
      } else {
        await createManagerOrg(payload);
        handleSuccess("创建");
      }
      handleModalClose();
      void loadList();
      // 机构已变动 → 顶层共享全量数据同步刷新（规则 Tab 的机构下拉随之更新）
      onOrgsChanged();
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      handleApiError(error, "操作");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = (record: ManagerOrg) => {
    // 68.1-T4 附加条目（T2 清单）：删除类 → confirmDelete（提交中禁取消由 helper 内建）
    confirmDelete({
      target: `机构「${record.orgName}」`,
      onDelete: async () => {
        await deleteManagerOrg(record.id);
        void loadList();
        onOrgsChanged();
      },
    });
  };

  const handleSearch = () => {
    const values = searchForm.getFieldsValue() as { orgName?: string };
    setCurrent(1);
    setSearchOrgName(values.orgName || undefined);
  };

  /** 从部门架构自动生成机构映射（幂等，不覆盖手工修改） */
  const handleGenerateFromDept = () => {
    // 68.1-T4 §1.9 非删除破坏性变体：从部门架构生成机构（幂等语义保留在 contentText）
    confirmDelete({
      title: "从部门架构生成机构",
      target: "扫描「外部机构」部门并自动生成机构记录",
      actionLabel: "生成",
      actionContext: "生成机构",
      contentText: "将{target}，已存在的机构不会被覆盖，可放心重复点击。是否继续？",
      successText: "已生成机构",
      onDelete: async () => {
        const result = await generateManagerOrgsFromDept();
        Modal.success({
          title: "生成完成",
          content: (
            <div>
              <p>候选机构部门：{result.totalCandidates} 个</p>
              <p>新建机构：{result.created} 个</p>
              <p>已存在跳过：{result.skippedExisting} 个</p>
              <p>顶层机构（无上级）：{result.orphanCount} 个</p>
              {result.warnings && result.warnings.length > 0 && (
                <p style={{ color: "var(--theme-warning)" }}>{result.warnings.join("；")}</p>
              )}
            </div>
          ),
        });
        void loadList();
        onOrgsChanged();
      },
    });
  };

  const handleReset = () => {
    searchForm.resetFields();
    setCurrent(1);
    setSearchOrgName(undefined);
  };

  const columns: ColumnsType<ManagerOrg> = [
    { title: "机构名称", dataIndex: "orgName", key: "orgName", width: 180 },
    {
      title: "对应部门",
      dataIndex: "deptName",
      key: "deptName",
      width: 180,
      render: (v: string | undefined, r) => v ?? r.deptId,
    },
    {
      title: "上级机构",
      dataIndex: "parentOrgName",
      key: "parentOrgName",
      width: 160,
      render: (v: string | undefined, r) => v ?? r.parentOrgId ?? "-",
    },
    {
      title: "机构类型",
      dataIndex: "orgType",
      key: "orgType",
      width: 120,
      render: (v: number) => (
        <Tag
          color={
            v === MANAGER_ORG_TYPE.BRANCH_COMPANY
              ? "blue"
              : v === MANAGER_ORG_TYPE.MIDDLE_BRANCH
                ? "cyan"
                : "green"
          }
        >
          {MANAGER_ORG_TYPE_LABELS[v] ?? String(v)}
        </Tag>
      ),
    },
    {
      title: "状态",
      dataIndex: "enabled",
      key: "enabled",
      width: 100,
      render: (v: boolean | number | undefined) =>
        isManagerEnabled(v) ? <Tag color="success">启用</Tag> : <Tag color="default">停用</Tag>,
    },
    {
      title: "操作",
      key: "action",
      width: 140,
      render: (_, record) => (
        <Space>
          {/* Phase 86 W1：行编辑 → ops:manager:org:edit；行删除 → ops:manager:org:delete。
              ready 无权限 → 入口不在 DOM；未就绪 → 渲染但 disabled + 提示。 */}
          {permEdit.visible && (
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => openModal(record)}
              disabled={permEdit.disabled}
              title={permEdit.hint}
            >
              编辑
            </Button>
          )}
          {permDelete.visible && (
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleDelete(record)}
              disabled={permDelete.disabled}
              title={permDelete.hint}
            >
              删除
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        <Form form={searchForm} layout="inline">
          <Form.Item name="orgName" label="机构名称">
            {/* 搜索框：清空只改受控值，不触发 onPressEnter（该回调仅回车时进搜索） */}
            <AppInput
              placeholder="请输入机构名称"
              style={{ width: 180 }}
              onPressEnter={handleSearch}
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                搜索
              </Button>
              <Button onClick={handleReset}>重置</Button>
              <Button icon={<ReloadOutlined />} onClick={() => void loadList()}>
                刷新
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      <Card
        extra={
          <Space>
            {/* Phase 86 W1：「从部门架构生成机构」是**写动作**（内部落库 generateManagerOrgsFromDept），
                归 ops:manager:org:edit（该码组无 generate 键，就近归最接近的 edit 键，不自造新码）。
                ⚠️ 不可归 delete —— 否则持 delete 无 edit 的用户可批量生成机构（越权写）。 */}
            {permEdit.visible && (
              <Button
                icon={<ThunderboltOutlined />}
                onClick={handleGenerateFromDept}
                disabled={permEdit.disabled}
                title={permEdit.hint}
              >
                从部门架构生成
              </Button>
            )}
            {/* 新增机构 → ops:manager:org:edit（该码组无 add 键，按最近码归属，不自造新码）。 */}
            {permEdit.visible && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={() => openModal()}
                disabled={permEdit.disabled}
                title={permEdit.hint}
              >
                新增机构
              </Button>
            )}
          </Space>
        }
      >
        <Table
          columns={columns}
          dataSource={list}
          loading={loading}
          rowKey="id"
          pagination={paginationProps}
        />
      </Card>

      <Modal
        title={editingOrg ? "编辑机构" : "新增机构"}
        open={modalOpen}
        onOk={handleSubmit}
        cancelButtonProps={{ disabled: saving }}
        onCancel={handleModalClose}
        destroyOnHidden
        width={560}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 18 }}>
          <Form.Item
            name="orgName"
            label="机构名称"
            rules={[{ required: true, message: "请输入机构名称" }]}
          >
            <AppInput placeholder="请输入机构名称" />
          </Form.Item>
          <Form.Item
            name="deptId"
            label="对应部门"
            rules={[{ required: true, message: "请选择对应部门" }]}
          >
            <TreeSelect
              placeholder="请选择对应部门"
              treeData={deptTreeData}
              treeDefaultExpandAll
              showSearch
              treeNodeFilterProp="title"
              allowClear
            />
          </Form.Item>
          <Form.Item name="parentOrgId" label="上级机构">
            <TreeSelect
              placeholder="不选则为顶级机构"
              treeData={parentTreeData}
              treeDefaultExpandAll
              showSearch
              treeNodeFilterProp="title"
              allowClear
            />
          </Form.Item>
          <Form.Item name="orgType" label="机构类型" rules={[{ required: true }]}>
            <Radio.Group>
              <Radio value={MANAGER_ORG_TYPE.BRANCH_COMPANY}>分公司</Radio>
              <Radio value={MANAGER_ORG_TYPE.MIDDLE_BRANCH}>中支</Radio>
              <Radio value={MANAGER_ORG_TYPE.SUB_BRANCH}>支公司</Radio>
            </Radio.Group>
          </Form.Item>
          {/* Phase 86 W1：弹窗内「是否启用」Switch 是表单字段（随新增/编辑机构提交生效），
              归 ops:manager:org:edit；ready 无权限 → 整个 Switch 不渲染（不是只禁用）。 */}
          {permEdit.visible && (
            <Form.Item name="enabled" label="是否启用" valuePropName="checked">
              <Switch
                checkedChildren="启用"
                unCheckedChildren="停用"
                disabled={permEdit.disabled}
                title={permEdit.hint}
              />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </div>
  );
};

// ==================== 同步规则 Tab ====================

interface ManagerRulesTabProps {
  /** 顶层共享的机构全量列表（下拉选项 + 机构名称渲染），全应用只拉取一次 */
  allOrgs: ManagerOrg[];
}

const ManagerRulesTab: FC<ManagerRulesTabProps> = ({ allOrgs }) => {
  const [list, setList] = useState<ManagerRule[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<ManagerRule | null>(null);
  const [saving, setSaving] = useState(false);

  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();

  // Phase 86 W1（RBAC-02）：操作级权限门控（三态 ready/loading/error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  // 归码说明（opsManagerRule 码组只有 list/edit/delete，**无 add 键**）：
  //   · 删除规则 → ops:manager:rule:delete
  //   · 新增规则 / 行编辑 / 启停 → ops:manager:rule:edit
  //     （写动作就近归最接近的既有键，不自造新码 —— Phase 86 零新增权限码）
  const permEdit = permGate.decide(PERMISSION_CODES.opsManagerRule.edit);
  const permDelete = permGate.decide(PERMISSION_CODES.opsManagerRule.delete);

  const { data: deptTree = [] } = useDeptTree();

  // 机构下拉数据源：直接复用顶层共享的全量列表（本组件不再自己发请求）
  const orgTreeData = useMemo(
    () => allOrgs.map((o) => ({ title: o.orgName, value: o.id })),
    [allOrgs]
  );
  const orgNameById = useMemo(() => {
    const map = new Map<string, string>();
    allOrgs.forEach((o) => map.set(o.id, o.orgName));
    return map;
  }, [allOrgs]);

  // 分页沿用全局 hook（current/pageSize 持久化、total 运行时写入）
  const { current, pageSize, setCurrent, setTotal, paginationProps } = usePagination();

  // 岗位选项
  const [postOptions, setPostOptions] = useState<PostItem[]>([]);
  useEffect(() => {
    post<{ list: PostItem[]; total: number }>("/system/posts/list", {
      current: 1,
      pageSize: 1000,
    })
      .then((res) => setPostOptions(res.data?.list ?? []))
      .catch((error) => handleApiError(error, "加载岗位选项", false));
  }, []);

  // 搜索条件（提交时快照）
  const [search, setSearch] = useState<{
    ruleName?: string;
    subjectType?: string;
    enabled?: boolean;
  }>({});

  const params = useMemo<ManagerRuleQuery>(
    () => ({ current, pageSize, ...search }),
    [current, pageSize, search]
  );

  const loadList = useCallback(async () => {
    setLoading(true);
    try {
      const result = await getManagerRuleList(params);
      setList(result.list ?? []);
      setTotal(result.total ?? 0);
    } catch (error) {
      handleApiError(error, "加载规则列表");
    } finally {
      setLoading(false);
    }
  }, [params, setTotal]);

  useEffect(() => {
    void loadList();
  }, [loadList]);

  const deptTreeData = useMemo(() => toPlainDeptTreeData(deptTree), [deptTree]);

  // 表单联动监听
  const subjectType = Form.useWatch("subjectType", editForm);
  const managerType = Form.useWatch("managerType", editForm);
  const managerOrgMode = Form.useWatch("managerOrgMode", editForm);
  const scopeType = Form.useWatch("scopeType", editForm);

  const openModal = (record?: ManagerRule) => {
    setEditingRule(record ?? null);
    setModalOpen(true);
    if (record) {
      let mode: ManagerOrgMode = "self";
      if (record.managerOrgId === MANAGER_ORG_TARGET_PARENT) mode = "parent";
      else if (record.managerOrgId) mode = "specified";
      editForm.setFieldsValue({
        ruleName: record.ruleName,
        priority: record.priority,
        enabled: isManagerEnabled(record.enabled),
        subjectType: record.subjectType,
        subjectPostId: record.subjectPostId,
        subjectOrgId: record.subjectOrgId,
        managerType: record.managerType,
        managerPostId: record.managerPostId,
        managerOrgMode: mode,
        managerOrgId: mode === "specified" ? record.managerOrgId : undefined,
        scopeType: record.scopeType,
        scopeId: record.scopeId,
      });
    } else {
      editForm.setFieldsValue({
        ruleName: undefined,
        priority: 100,
        enabled: true,
        subjectType: "employee",
        subjectPostId: undefined,
        subjectOrgId: undefined,
        managerType: "dept_leader",
        managerPostId: undefined,
        managerOrgMode: "self",
        managerOrgId: undefined,
        scopeType: MANAGER_SCOPE_TYPE.ALL,
        scopeId: undefined,
      });
    }
  };

  const handleModalClose = () => {
    setModalOpen(false);
    setEditingRule(null);
    editForm.resetFields();
  };

  const handleSubmit = async () => {
    try {
      const values = await editForm.validateFields();
      setSaving(true);
      let managerOrgId: string | undefined;
      if (values.managerOrgMode === "parent") managerOrgId = MANAGER_ORG_TARGET_PARENT;
      else if (values.managerOrgMode === "specified") managerOrgId = values.managerOrgId;
      const payload: ManagerRulePayload = {
        ruleName: values.ruleName,
        priority: values.priority,
        enabled: values.enabled,
        subjectType: values.subjectType,
        subjectPostId: values.subjectType === "post_holder" ? values.subjectPostId : undefined,
        subjectOrgId: values.subjectOrgId || undefined,
        managerType: values.managerType,
        managerPostId:
          values.managerType === "post_holder" || values.managerType === "org_head"
            ? values.managerPostId
            : undefined,
        managerOrgId,
        scopeType: values.scopeType,
        scopeId: values.scopeType !== MANAGER_SCOPE_TYPE.ALL ? values.scopeId : undefined,
      };
      if (editingRule) {
        await updateManagerRule(editingRule.id, payload);
        handleSuccess("更新");
      } else {
        await createManagerRule(payload);
        handleSuccess("创建");
      }
      handleModalClose();
      void loadList();
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      handleApiError(error, "操作");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = (record: ManagerRule) => {
    confirmDelete({
      target: `管理规则 ${record.ruleName}`,
      onDelete: async () => {
        await deleteManagerRule(record.id);
        handleSuccess("删除");
        void loadList();
      },
    });
  };

  const handleSearch = () => {
    const values = searchForm.getFieldsValue() as {
      ruleName?: string;
      subjectType?: ManagerSubjectType;
      enabled?: boolean;
    };
    setCurrent(1);
    setSearch({
      ruleName: values.ruleName || undefined,
      subjectType: values.subjectType,
      enabled: values.enabled,
    });
  };

  const handleReset = () => {
    searchForm.resetFields();
    setCurrent(1);
    setSearch({});
  };

  /**
   * 可读名兜底：可读名 > 共享机构全量数据里的机构名 > 原始值截断 > "-"
   * （resolveOrg=true 时用于「范围机构」这类只可能拿到机构 id 的位置）
   */
  const renderName = (
    readable: string | undefined,
    raw: string | undefined,
    resolveOrg = false
  ): string => {
    if (readable) return readable;
    if (!raw) return "-";
    if (resolveOrg) {
      const orgName = orgNameById.get(raw);
      if (orgName) return orgName;
    }
    return raw.slice(0, 8) + "…";
  };

  const columns: ColumnsType<ManagerRule> = [
    { title: "规则名称", dataIndex: "ruleName", key: "ruleName", width: 160 },
    { title: "优先级", dataIndex: "priority", key: "priority", width: 80 },
    {
      title: "状态",
      dataIndex: "enabled",
      key: "enabled",
      width: 90,
      render: (v: boolean | number | undefined) =>
        isManagerEnabled(v) ? <Tag color="success">启用</Tag> : <Tag color="default">停用</Tag>,
    },
    {
      title: "适用主体",
      key: "subject",
      width: 160,
      render: (_, r) => (
        <Space size={4}>
          <Tag color="blue">{MANAGER_SUBJECT_LABELS[r.subjectType] ?? r.subjectType}</Tag>
          {r.subjectType === "post_holder" && (
            <span style={{ color: "var(--theme-text-tertiary)" }}>
              {renderName(r.subjectPostName, r.subjectPostId)}
            </span>
          )}
        </Space>
      ),
    },
    {
      title: "经理来源",
      key: "manager",
      width: 200,
      render: (_, r) => (
        <Space size={4}>
          <Tag color="purple">{MANAGER_TYPE_LABELS[r.managerType] ?? r.managerType}</Tag>
          {(r.managerType === "post_holder" || r.managerType === "org_head") && (
            <span style={{ color: "var(--theme-text-tertiary)" }}>
              {renderName(r.managerPostName, r.managerPostId)}
            </span>
          )}
          {r.managerOrgId === MANAGER_ORG_TARGET_PARENT && <Tag>上级机构</Tag>}
        </Space>
      ),
    },
    {
      title: "作用范围",
      key: "scope",
      width: 180,
      render: (_, r) => {
        const label = MANAGER_SCOPE_LABELS[r.scopeType] ?? String(r.scopeType);
        if (r.scopeType === MANAGER_SCOPE_TYPE.DEPT_SUBTREE)
          return (
            <span>
              {label}：{renderName(r.scopeDeptName, r.scopeId)}
            </span>
          );
        if (r.scopeType === MANAGER_SCOPE_TYPE.ORG_SUBTREE)
          return (
            <span>
              {label}：{renderName(r.scopeOrgName, r.scopeId, true)}
            </span>
          );
        return <span>{label}</span>;
      },
    },
    {
      title: "操作",
      key: "action",
      width: 140,
      render: (_, record) => (
        <Space>
          {/* Phase 86 W1：行编辑 → ops:manager:rule:edit；行删除 → ops:manager:rule:delete。
              ready 无权限 → 入口不在 DOM；未就绪 → 渲染但 disabled + 提示。 */}
          {permEdit.visible && (
            <Button
              type="link"
              size="small"
              icon={<EditOutlined />}
              onClick={() => openModal(record)}
              disabled={permEdit.disabled}
              title={permEdit.hint}
            >
              编辑
            </Button>
          )}
          {permDelete.visible && (
            <Button
              type="link"
              size="small"
              danger
              icon={<DeleteOutlined />}
              onClick={() => handleDelete(record)}
              disabled={permDelete.disabled}
              title={permDelete.hint}
            >
              删除
            </Button>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        <Form form={searchForm} layout="inline">
          <Form.Item name="ruleName" label="规则名称">
            {/* 搜索框：清空只改受控值，不触发 onPressEnter */}
            <AppInput
              placeholder="请输入规则名称"
              style={{ width: 160 }}
              onPressEnter={handleSearch}
            />
          </Form.Item>
          <Form.Item name="subjectType" label="适用主体">
            <AppSelect
              placeholder="请选择适用主体"
              allowClear
              style={{ width: 150 }}
              options={SUBJECT_TYPES.map((t) => ({ value: t, label: MANAGER_SUBJECT_LABELS[t] }))}
            />
          </Form.Item>
          <Form.Item name="enabled" label="状态">
            <AppSelect
              placeholder="请选择状态"
              allowClear
              style={{ width: 110 }}
              options={[
                { value: true, label: "启用" },
                { value: false, label: "停用" },
              ]}
            />
          </Form.Item>
          <Form.Item>
            <Space>
              <Button type="primary" icon={<SearchOutlined />} onClick={handleSearch}>
                搜索
              </Button>
              <Button onClick={handleReset}>重置</Button>
              <Button icon={<ReloadOutlined />} onClick={() => void loadList()}>
                刷新
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      <Card
        extra={
          // Phase 86 W1：新增规则 → ops:manager:rule:edit（该码组无 add 键，按最近码归属，不自造新码）。
          permEdit.visible ? (
            <Button
              type="primary"
              icon={<PlusOutlined />}
              onClick={() => openModal()}
              disabled={permEdit.disabled}
              title={permEdit.hint}
            >
              新增规则
            </Button>
          ) : undefined
        }
      >
        <Table
          columns={columns}
          dataSource={list}
          loading={loading}
          rowKey="id"
          pagination={paginationProps}
        />
      </Card>

      <Modal
        title={editingRule ? "编辑规则" : "新增规则"}
        open={modalOpen}
        onOk={handleSubmit}
        cancelButtonProps={{ disabled: saving }}
        onCancel={handleModalClose}
        destroyOnHidden
        width={620}
      >
        <Form form={editForm} layout="horizontal" labelCol={{ span: 6 }} wrapperCol={{ span: 17 }}>
          <Form.Item
            name="ruleName"
            label="规则名称"
            rules={[{ required: true, message: "请输入规则名称" }]}
          >
            <AppInput placeholder="请输入规则名称" />
          </Form.Item>
          <Form.Item
            name="priority"
            label="优先级"
            rules={[{ required: true, message: "请输入优先级" }]}
          >
            <InputNumber min={0} style={{ width: "100%" }} placeholder="数值越小优先级越高" />
          </Form.Item>
          {/* Phase 86 W1：弹窗内「是否启用」Switch 是表单字段（随新增/编辑规则提交生效），
              归 ops:manager:rule:edit；ready 无权限 → 整个 Switch 不渲染（不是只禁用）。 */}
          {permEdit.visible && (
            <Form.Item name="enabled" label="是否启用" valuePropName="checked">
              <Switch
                checkedChildren="启用"
                unCheckedChildren="停用"
                disabled={permEdit.disabled}
                title={permEdit.hint}
              />
            </Form.Item>
          )}
          <Form.Item
            name="subjectType"
            label="适用主体"
            rules={[{ required: true, message: "请选择适用主体" }]}
          >
            <AppSelect
              allowClear
              options={SUBJECT_TYPES.map((t) => ({ value: t, label: MANAGER_SUBJECT_LABELS[t] }))}
              placeholder="请选择适用主体"
            />
          </Form.Item>
          {subjectType === "post_holder" && (
            <Form.Item
              name="subjectPostId"
              label="主体岗位"
              rules={[{ required: true, message: "请选择主体岗位" }]}
            >
              <AppSelect
                allowClear
                placeholder="请选择主体岗位"
                showSearch
                optionFilterProp="label"
                options={postOptions.map((p) => ({ value: p.id, label: p.postName }))}
              />
            </Form.Item>
          )}
          <Form.Item name="subjectOrgId" label="限定机构">
            <TreeSelect
              placeholder="不选则不限定机构"
              treeData={orgTreeData}
              showSearch
              treeNodeFilterProp="title"
              allowClear
            />
          </Form.Item>
          <Form.Item
            name="managerType"
            label="经理来源"
            rules={[{ required: true, message: "请选择经理来源" }]}
          >
            <AppSelect
              allowClear
              options={MANAGER_TYPES.map((t) => ({ value: t, label: MANAGER_TYPE_LABELS[t] }))}
              placeholder="请选择经理来源"
            />
          </Form.Item>
          {(managerType === "post_holder" || managerType === "org_head") && (
            <Form.Item
              name="managerPostId"
              label="经理岗位"
              rules={[{ required: true, message: "请选择经理岗位" }]}
            >
              <AppSelect
                allowClear
                placeholder="请选择经理岗位"
                showSearch
                optionFilterProp="label"
                options={postOptions.map((p) => ({ value: p.id, label: p.postName }))}
              />
            </Form.Item>
          )}
          <Form.Item name="managerOrgMode" label="目标机构">
            <AppSelect
              allowClear
              onSearch={() => {}}
              options={[
                { value: "self", label: "自身机构" },
                { value: "parent", label: "上级机构" },
                { value: "specified", label: "指定机构" },
              ]}
            />
          </Form.Item>
          {managerOrgMode === "specified" && (
            <Form.Item
              name="managerOrgId"
              label="指定机构"
              rules={[{ required: true, message: "请选择指定机构" }]}
            >
              <TreeSelect
                placeholder="请选择指定机构"
                treeData={orgTreeData}
                showSearch
                treeNodeFilterProp="title"
                allowClear
              />
            </Form.Item>
          )}
          <Form.Item name="scopeType" label="作用范围">
            <Radio.Group
              options={Object.entries(MANAGER_SCOPE_LABELS).map(([value, label]) => ({
                value: Number(value),
                label,
              }))}
            />
          </Form.Item>
          {scopeType === MANAGER_SCOPE_TYPE.DEPT_SUBTREE && (
            <Form.Item
              name="scopeId"
              label="范围部门"
              rules={[{ required: true, message: "请选择范围部门" }]}
            >
              <TreeSelect
                placeholder="请选择范围根部门"
                treeData={deptTreeData}
                treeDefaultExpandAll
                showSearch
                treeNodeFilterProp="title"
                allowClear
              />
            </Form.Item>
          )}
          {scopeType === MANAGER_SCOPE_TYPE.ORG_SUBTREE && (
            <Form.Item
              name="scopeId"
              label="范围机构"
              rules={[{ required: true, message: "请选择范围机构" }]}
            >
              <TreeSelect
                placeholder="请选择范围根机构"
                treeData={orgTreeData}
                showSearch
                treeNodeFilterProp="title"
                allowClear
              />
            </Form.Item>
          )}
        </Form>
      </Modal>
    </div>
  );
};

// ==================== 设置区块入口 ====================

const ManagerSyncSettings: FC = () => {
  // 操作级权限门控（三态语义）：Tab 级判定同样走 decide —— 未就绪时保持可见（禁用态由内部按钮承担），
  // ready 且无 list 码才不渲染对应 Tab。
  const permGate = useActionPermission();
  const ruleListPerm = permGate.decide(PERMISSION_CODES.opsManagerRule.list);
  const orgListPerm = permGate.decide(PERMISSION_CODES.opsManagerOrg.list);

  // 机构全量数据（唯一请求点）：机构 Tab 的机构下拉、规则 Tab 的机构下拉/名称渲染共用这一份。
  const [allOrgs, setAllOrgs] = useState<ManagerOrg[]>([]);

  // 拉取全量机构并写入共享状态：setState 放在 promise 回调里（不在 effect 同步路径上），
  // 与原 manager-rules 的 `.then(setOrgOptions)` 写法一致。
  const loadAllOrgs = useCallback(
    () =>
      getManagerOrgList({ current: 1, pageSize: 1000 })
        .then((result) => setAllOrgs(result.list ?? []))
        .catch((error) => {
          // 与原实现一致：机构选项加载失败静默（不打断页面）
          handleApiError(error, "加载机构选项", false);
        }),
    []
  );

  // 至少有一个 Tab 需要机构数据时才拉全量；权限未就绪时不发（allowed 在 loading/error 下为 false）。
  const needOrgs =
    permGate.allowed(PERMISSION_CODES.opsManagerOrg.list) ||
    permGate.allowed(PERMISSION_CODES.opsManagerRule.list);

  useEffect(() => {
    if (!needOrgs) return;
    void loadAllOrgs();
  }, [needOrgs, loadAllOrgs]);

  // 机构 Tab 写操作成功后的统一刷新入口（机构列表分页请求仍由 Tab 自己负责）
  const handleOrgsChanged = useCallback(() => {
    void loadAllOrgs();
  }, [loadAllOrgs]);

  const tabs: NonNullable<TabsProps["items"]> = [];
  if (orgListPerm.visible) {
    tabs.push({
      key: "orgs",
      label: "机构定义",
      children: <ManagerOrgsTab onOrgsChanged={handleOrgsChanged} />,
    });
  }
  if (ruleListPerm.visible) {
    tabs.push({
      key: "rules",
      label: "同步规则",
      children: <ManagerRulesTab allOrgs={allOrgs} />,
    });
  }

  // 两个 list 权限都没有 → 整个区块不渲染
  if (tabs.length === 0) return null;
  // 只剩一个 Tab 时不必渲染单页签头
  if (tabs.length === 1) return tabs[0]?.children ?? null;

  return <Tabs items={tabs} />;
};

export default ManagerSyncSettings;
