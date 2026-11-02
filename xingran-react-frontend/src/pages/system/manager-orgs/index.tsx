import { useState, useEffect, useCallback, useMemo } from "react";
import type { FC } from "react";
import { Card, Table, Button, Space, Form, Radio, Switch, TreeSelect, Modal, Tag } from "antd";
// Phase 71 (W2)：裸 Input → AppInput（搜索框带 onPressEnter，清空不误触搜索——见下注记）
import { AppInput } from "@/components/ui";
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
  MANAGER_ORG_TYPE,
  MANAGER_ORG_TYPE_LABELS,
  isManagerEnabled,
  type ManagerOrg,
  type ManagerOrgQuery,
  type ManagerOrgPayload,
} from "@/lib/managerApi";
import type { SimpleDept } from "@/lib/dutyApi";
import { useDeptTree } from "@/hooks/useDeptTree";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";
import { confirmDelete, useActionPermission } from "@/components/shared";
// Phase 86 W1（RBAC-02）：操作级权限门控 —— 单一真相源 + 三态门控 hook。
import { PERMISSION_CODES } from "@/constants/permissionCodes";

/** 把部门树转成 TreeSelect 数据 */
function toTreeData(
  depts: SimpleDept[],
  disabledIds: ReadonlySet<string>
): {
  title: string;
  value: string;
  key: string;
  disabled: boolean;
  children?: ReturnType<typeof toTreeData>;
}[] {
  return depts.map((d) => ({
    title: d.deptName,
    value: d.id,
    key: d.id,
    disabled: disabledIds.has(d.id),
    children: d.children ? toTreeData(d.children, disabledIds) : undefined,
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

const ManagerOrgsPage: FC = () => {
  const [list, setList] = useState<ManagerOrg[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingOrg, setEditingOrg] = useState<ManagerOrg | null>(null);
  const [saving, setSaving] = useState(false);
  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();

  // Phase 86 W1（RBAC-02）：操作级权限门控（三态 ready/loading/error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  // 归码说明（opsManagerOrg 码组只有 list/edit/delete，**无 add / generate 键**）：
  //   · 删除机构 → ops:manager:org:delete
  //   · 新增机构 / 从部门架构生成机构 / 行编辑 / 启停 → ops:manager:org:edit
  //     （写动作就近归最接近的既有键，不自造新码 —— Phase 86 零新增权限码）
  const permEdit = permGate.decide(PERMISSION_CODES.opsManagerOrg.edit);
  const permDelete = permGate.decide(PERMISSION_CODES.opsManagerOrg.delete);

  const { data: deptTree = [] } = useDeptTree();

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
  }, [params]);

  useEffect(() => {
    loadList();
  }, [loadList]);

  // 父机构 TreeSelect：编辑时禁止选自己/自己的后代
  const parentTreeData = useMemo(() => {
    const disabled = new Set<string>();
    if (editingOrg) collectSelfAndDescendants(deptTree, editingOrg.deptId, disabled);
    return toTreeData(deptTree, disabled);
  }, [deptTree, editingOrg]);

  const deptTreeData = useMemo(() => toTreeData(deptTree, new Set()), [deptTree]);

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
      loadList();
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
        loadList();
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
        loadList();
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
              <Button icon={<ReloadOutlined />} onClick={loadList}>
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
          pagination={{
            current,
            pageSize,
            total,
            showSizeChanger: true,
            showTotal: (t) => `共 ${t} 条`,
            onChange: (page, size) => {
              setCurrent(page);
              setPageSize(size);
            },
          }}
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

export default ManagerOrgsPage;
