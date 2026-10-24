import { useState, useEffect, useCallback, useMemo } from "react";
import type { FC } from "react";
import {
  Card,
  Table,
  Button,
  Space,
  Form,
  Input,
  InputNumber,
  Radio,
  Select,
  Switch,
  TreeSelect,
  Modal,
  Tag,
} from "antd";
import type { ColumnsType } from "antd/es/table";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
} from "@ant-design/icons";
import {
  getManagerRuleList,
  createManagerRule,
  updateManagerRule,
  deleteManagerRule,
  getManagerOrgList,
  MANAGER_ORG_TARGET_PARENT,
  MANAGER_SCOPE_TYPE,
  MANAGER_SCOPE_LABELS,
  MANAGER_SUBJECT_LABELS,
  MANAGER_TYPE_LABELS,
  isManagerEnabled,
  type ManagerRule,
  type ManagerRuleQuery,
  type ManagerRulePayload,
  type ManagerOrg,
  type ManagerSubjectType,
  type ManagerType,
} from "@/lib/managerApi";
import type { SimpleDept } from "@/lib/dutyApi";
import { useDeptTree } from "@/hooks/useDeptTree";
import { post } from "@/lib/api";
import { handleApiError, handleSuccess, isFormValidationError } from "@/utils/errorHandler";

interface PostItem {
  id: string;
  postName: string;
}

/** 目标机构选择模式 */
type ManagerOrgMode = "self" | "parent" | "specified";

const SUBJECT_TYPES = Object.keys(MANAGER_SUBJECT_LABELS) as ManagerSubjectType[];
const MANAGER_TYPES = Object.keys(MANAGER_TYPE_LABELS) as ManagerType[];

/** 部门树转 TreeSelect 数据 */
function toTreeData(depts: SimpleDept[]): { title: string; value: string; key: string; children?: ReturnType<typeof toTreeData> }[] {
  return depts.map((d) => ({
    title: d.deptName,
    value: d.id,
    key: d.id,
    children: d.children ? toTreeData(d.children) : undefined,
  }));
}

const ManagerRulesPage: FC = () => {
  const [list, setList] = useState<ManagerRule[]>([]);
  const [total, setTotal] = useState(0);
  const [current, setCurrent] = useState(1);
  const [pageSize, setPageSize] = useState(10);
  const [loading, setLoading] = useState(false);
  const [modalOpen, setModalOpen] = useState(false);
  const [editingRule, setEditingRule] = useState<ManagerRule | null>(null);
  const [saving, setSaving] = useState(false);

  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();

  const { data: deptTree = [] } = useDeptTree();

  // 机构选项（全量拉取，用于 subjectOrgId / 指定机构）
  const [orgOptions, setOrgOptions] = useState<ManagerOrg[]>([]);
  useEffect(() => {
    getManagerOrgList({ current: 1, pageSize: 1000 })
      .then((res) => setOrgOptions(res.list ?? []))
      .catch((error) => handleApiError(error, "加载机构选项", false));
  }, []);

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
  const [search, setSearch] = useState<{ ruleName?: string; subjectType?: string; enabled?: boolean }>({});

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
  }, [params]);

  useEffect(() => {
    loadList();
  }, [loadList]);

  const deptTreeData = useMemo(() => toTreeData(deptTree), [deptTree]);

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
      loadList();
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      handleApiError(error, "操作");
    } finally {
      setSaving(false);
    }
  };

  const handleDelete = (record: ManagerRule) => {
    const confirmInstance = Modal.confirm({
      title: "删除规则",
      content: `确定要删除规则「${record.ruleName}」吗？`,
      okText: "确定",
      okType: "danger",
      cancelText: "取消",
      onOk: async () => {
        confirmInstance.update({ cancelButtonProps: { disabled: true } });
        try {
          await deleteManagerRule(record.id);
          handleSuccess("删除");
          loadList();
        } catch (error) {
          handleApiError(error, "删除");
        } finally {
          confirmInstance.update({ cancelButtonProps: { disabled: false } });
        }
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

  /** 可读名兜底：有可读名显示可读名，否则显示原始值，空显示 - */
  const renderName = (readable: string | undefined, raw: string | undefined): string =>
    readable ?? (raw ? raw.slice(0, 8) + "…" : "-");

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
            <span style={{ color: "#888" }}>{renderName(r.subjectPostName, r.subjectPostId)}</span>
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
            <span style={{ color: "#888" }}>{renderName(r.managerPostName, r.managerPostId)}</span>
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
          return <span>{label}：{renderName(r.scopeDeptName, r.scopeId)}</span>;
        if (r.scopeType === MANAGER_SCOPE_TYPE.ORG_SUBTREE)
          return <span>{label}：{renderName(r.scopeOrgName, r.scopeId)}</span>;
        return <span>{label}</span>;
      },
    },
    {
      title: "操作",
      key: "action",
      width: 140,
      render: (_, record) => (
        <Space>
          <Button type="link" size="small" icon={<EditOutlined />} onClick={() => openModal(record)}>
            编辑
          </Button>
          <Button type="link" size="small" danger icon={<DeleteOutlined />} onClick={() => handleDelete(record)}>
            删除
          </Button>
        </Space>
      ),
    },
  ];

  return (
    <div>
      <Card style={{ marginBottom: 16 }}>
        <Form form={searchForm} layout="inline">
          <Form.Item name="ruleName" label="规则名称">
            <Input placeholder="请输入规则名称" allowClear style={{ width: 160 }} onPressEnter={handleSearch} />
          </Form.Item>
          <Form.Item name="subjectType" label="适用主体">
            <Select
              onSearch={() => {}}
              placeholder="请选择适用主体"
              allowClear
              style={{ width: 150 }}
              options={SUBJECT_TYPES.map((t) => ({ value: t, label: MANAGER_SUBJECT_LABELS[t] }))}
            />
          </Form.Item>
          <Form.Item name="enabled" label="状态">
            <Select
              onSearch={() => {}}
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
              <Button icon={<ReloadOutlined />} onClick={loadList}>
                刷新
              </Button>
            </Space>
          </Form.Item>
        </Form>
      </Card>

      <Card
        extra={
          <Button type="primary" icon={<PlusOutlined />} onClick={() => openModal()}>
            新增规则
          </Button>
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
            <Input placeholder="请输入规则名称" />
          </Form.Item>
          <Form.Item
            name="priority"
            label="优先级"
            rules={[{ required: true, message: "请输入优先级" }]}
          >
            <InputNumber min={0} style={{ width: "100%" }} placeholder="数值越小优先级越高" />
          </Form.Item>
          <Form.Item name="enabled" label="是否启用" valuePropName="checked">
            <Switch checkedChildren="启用" unCheckedChildren="停用" />
          </Form.Item>
          <Form.Item
            name="subjectType"
            label="适用主体"
            rules={[{ required: true, message: "请选择适用主体" }]}
          >
            <Select
              onSearch={() => {}}
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
              <Select
                onSearch={() => {}}
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
              treeData={orgOptions.map((o) => ({ title: o.orgName, value: o.id }))}
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
            <Select
              onSearch={() => {}}
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
              <Select
                onSearch={() => {}}
                placeholder="请选择经理岗位"
                showSearch
                optionFilterProp="label"
                options={postOptions.map((p) => ({ value: p.id, label: p.postName }))}
              />
            </Form.Item>
          )}
          <Form.Item name="managerOrgMode" label="目标机构">
            <Select
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
                treeData={orgOptions.map((o) => ({ title: o.orgName, value: o.id }))}
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
                treeData={orgOptions.map((o) => ({ title: o.orgName, value: o.id }))}
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

export default ManagerRulesPage;
