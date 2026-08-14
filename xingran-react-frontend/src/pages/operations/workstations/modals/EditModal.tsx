/**
 * Workstation Edit Modal
 * 工位编辑模态框
 *
 * 级联下拉逻辑(Phase: 外部机构化):
 * - 所属机构 → 仅外部机构(orgTreeData 派生自 deptTreeData + filterExternalOrgDepts)
 * - 所属部门 → 选中机构下的全后代部门(从 deptTreeData 找 orgId 节点的子树)
 * - 所属用户 → 当前部门 + 子部门的用户(loadUserOptions 用 recursiveDeptId 触发后端递归)
 *
 * 级联清空:
 * - 改机构 → 清空 floorId/deptId/userId
 * - 改部门 → 清空 userId
 */

import { useLayoutEffect, useMemo } from "react";
import { Form, Input, InputNumber, Select, Row, Col, Cascader, TreeSelect } from "antd";
import type { FormInstance } from "antd";
import type { WorkstationOps } from "@/types";
import type { DeptOption } from "@/lib/opsApi";
import type { DeptTreeNode, UserOption } from "../types";
import { findDeptNode, trimTitleToLastSegment } from "@/utils/deptUtils";
import { STATUS_OPTIONS, TYPE_OPTIONS } from "../constants";
import { BaseEditModal } from "@/components/modal/BaseEditModal";

const { Option } = Select;
const { TextArea } = Input;

// Cascader 选项类型(机构→楼宇→楼层)
type CascaderOption = {
  value: string;
  label: string;
  children?: CascaderOption[];
  isLeaf?: boolean;
};

export interface WorkstationEditModalProps {
  open: boolean;
  form: FormInstance;
  editingWorkstation: WorkstationOps | null;
  /** 仅外部机构(isExternalOrg===1)的部门树,用于"所属机构"下拉 */
  orgTreeData: DeptTreeNode[];
  /** 全量部门树(原始,含 isExternalOrg 信息),用于在选中机构下取子树展示"所属部门" */
  deptTreeData: DeptTreeNode[];
  userOptions: UserOption[];
  cascaderOptions: CascaderOption[];
  loadingCascader: boolean;
  handleCascaderLoadData: (selectedOptions: CascaderOption[]) => Promise<void> | void;
  onOk: (values: Record<string, unknown>) => Promise<void>;
  onCancel: () => void;
  onDeptChange: (deptId: string) => void;
  onOrgChange: (orgId: string) => void;
  /** Phase 39: union 注入的 alias 映射部门列表(isAlias=true 条目)。
   *  由父组件 index.tsx 通过 useAliasByLocation(watchedOrgId) 拉取后透传。
   *  在 subDeptTree 末尾追加为带 `[映射]` 后缀的叶子节点。 */
  aliasList?: DeptOption[];
}

export function WorkstationEditModal({
  open,
  form,
  editingWorkstation,
  orgTreeData,
  deptTreeData,
  aliasList,
  userOptions,
  cascaderOptions,
  loadingCascader,
  handleCascaderLoadData,
  onOk,
  onCancel,
  onDeptChange,
  onOrgChange,
}: WorkstationEditModalProps) {
  // 仅在新增模式下重置表单（编辑模式的值由父组件设置）
  useLayoutEffect(() => {
    if (open && !editingWorkstation?.id) {
      form.resetFields();
      form.setFieldsValue({ status: 0, type: 0 });
    }
  }, [open, editingWorkstation, form]);

  // 派生:在 deptTreeData 中按 orgId 找出节点,取其全后代子树作为"所属部门"的选项。
  // 注意 deptTreeData 是 useWorkstationData 加载的全量部门树(顶层节点可能不是外部机构,
  // 但其子孙可能挂在某个外部机构下),所以查找必须跨顶层边界。
  // 使用 Form.useWatch 订阅 orgId 字段变化,避免手动监听 setFieldsValue 副作用。
  //
  // 标题收窄:subDeptTree 内的 title 由 useWorkstationData.buildTreeData 拼接为
  // 完整路径(如 "中国太平洋财产保险股份有限公司/分公司本部/人力资源部")。
  // 在"所属部门"下拉里这种全路径过于冗长,这里收窄为最后一段
  // (即 "人力资源部"),但仍保留父子结构以便展开查看子部门。
  // 不影响 orgTreeData(它仍展示全路径,有助于"所属机构"的辨识)。
  const watchedOrgId = Form.useWatch("orgId", form) as string | undefined;
  const subDeptTree = useMemo<DeptTreeNode[]>(() => {
    if (!watchedOrgId) return [];
    const node = findDeptNode(deptTreeData, watchedOrgId);
    // D-LOCKED (Phase 37): 基线子树派生语义不可变 — findDeptNode(deptTreeData, orgId).children → trimTitleToLastSegment
    const baseTree: DeptTreeNode[] = node?.children?.length
      ? trimTitleToLastSegment(node.children)
      : [];

    // Phase 39 union 注入 (D-01 决策): 仅 isAlias=true 的映射部门追加到 baseTree 末尾。
    // 数据源为后端 POST /ops/workstation/dept-options 的 isAlias=true 条目,
    // 经父组件 useAliasByLocation → aliasList 透传进来。
    // 注意:aliasList 也含 isAlias=false 的机构子树条目(后端 union 第一段),但那些
    // 已经在 baseTree(deptTreeData 子树)里了,必须过滤掉,否则整棵子树重复注入 →
    // 下拉显示所有部门 + TreeSelect "Same key" 重复告警。
    const aliasOnly = (aliasList ?? []).filter((a) => a?.isAlias === true);
    if (aliasOnly.length === 0) return baseTree;

    const aliasNodes: DeptTreeNode[] = aliasOnly.map((a) => {
      // 走 trimTitleToLastSegment 保持与 baseTree title 收窄规则一致
      // (DeptOption.deptName 一般不含 " / ",函数会原样返回)
      const trimmed =
        trimTitleToLastSegment([{ title: a.deptName, children: [] }])[0]?.title ?? a.deptName;
      return {
        title: `${trimmed} [映射]`, // D-01 锁定: 原名 + " [映射]" 后缀 (UAT 文本断言依赖)
        value: a.deptId,
        key: a.deptId,
        isLeaf: true,
        // 标记 is_alias 便于其他消费者识别(不影响渲染)
        ...({ is_alias: true } as Record<string, unknown>),
      } as DeptTreeNode;
    });

    return [...baseTree, ...aliasNodes];
  }, [deptTreeData, watchedOrgId, aliasList]);

  return (
    <BaseEditModal
      title={editingWorkstation ? "编辑工位" : "新增工位"}
      open={open}
      onOk={async () => {
        // 验证表单并获取值
        const values = await form.validateFields();
        // 将表单值传递给父组件的 onOk 处理
        await onOk(values);
      }}
      onCancel={() => {
        onCancel();
        form.resetFields();
      }}
      width={700}
    >
      <Form form={form} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
        <Form.Item
          name="orgId"
          label="所属机构"
          rules={[{ required: true, message: "请选择所属机构" }]}
        >
          <TreeSelect
            treeData={orgTreeData}
            placeholder="请选择机构（仅外部机构）"
            allowClear
            showSearch
            treeLine={{ showLeafIcon: false }}
            treeDefaultExpandAll={false}
            onChange={(value) => {
              // 机构变化 → 清空楼层、部门、用户,触发父级重新加载楼宇
              form.setFieldsValue({
                floorId: undefined,
                deptId: undefined,
                userId: undefined,
              });
              onOrgChange(value as string);
            }}
          />
        </Form.Item>
        <Form.Item
          name="floorId"
          label="所属楼层"
          rules={[{ required: true, message: "请选择所属楼层" }]}
        >
          <Cascader
            options={cascaderOptions}
            loadData={handleCascaderLoadData}
            loading={loadingCascader}
            placeholder="请先选择楼宇，再选择楼层"
            changeOnSelect
            showSearch={{
              filter: (inputValue, path) =>
                path.some((option) =>
                  option.label?.toLowerCase().includes(inputValue.toLowerCase())
                ),
            }}
          />
        </Form.Item>
        <Form.Item
          name="name"
          label="工位名称"
          rules={[{ required: true, message: "请输入工位名称" }]}
        >
          <Input placeholder="请输入工位名称" />
        </Form.Item>
        <Form.Item
          name="type"
          label="工位类型"
          rules={[{ required: true, message: "请选择工位类型" }]}
        >
          <Select placeholder="请选择工位类型" onSearch={() => {}}>
            {TYPE_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="deptId" label="所属部门">
              <TreeSelect
                treeData={subDeptTree}
                placeholder={subDeptTree.length ? "请选择部门" : "请先选择机构"}
                allowClear
                showSearch
                treeLine={{ showLeafIcon: false }}
                disabled={subDeptTree.length === 0}
                onChange={(value) => {
                  // 部门变化 → 清空用户,触发父级按新部门加载用户
                  form.setFieldsValue({ userId: undefined });
                  onDeptChange((value as string) ?? "");
                }}
              />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="userId" label="所属用户">
              <Select
                placeholder="请先选择部门"
                allowClear
                showSearch
                optionFilterProp="children"
                onChange={(value) => {
                  // ✨ 联动: 选择/清空用户时自动切换 status (Maintain=2 保留)
                  // 后端 Service 层为兜底,绕过前端无效
                  const currentStatus = form.getFieldValue("status");
                  if (currentStatus === 2) return; // 维护状态不变
                  form.setFieldsValue({
                    status: value ? 1 : 0,
                  });
                }}
              >
                {userOptions.map((u) => (
                  <Option key={u.id} value={u.id}>
                    {u.nickname ? `${u.username} (${u.nickname})` : u.username}
                  </Option>
                ))}
              </Select>
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="positionX" label="X坐标">
              <InputNumber placeholder="X坐标" style={{ width: "100%" }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="positionY" label="Y坐标">
              <InputNumber placeholder="Y坐标" style={{ width: "100%" }} />
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="width" label="宽度" initialValue={160}>
              <InputNumber placeholder="宽度" min={50} max={500} style={{ width: "100%" }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="depth" label="深度" initialValue={70}>
              <InputNumber placeholder="深度" min={30} max={300} style={{ width: "100%" }} />
            </Form.Item>
          </Col>
        </Row>
        <Form.Item
          name="status"
          label="状态"
          rules={[{ required: true, message: "请选择状态" }]}
          extra="选择/清空所属用户时,状态会自动切换为占用/空闲(维护状态除外)"
        >
          <Select placeholder="请选择状态">
            {STATUS_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item name="description" label="描述">
          <TextArea rows={3} placeholder="请输入描述" />
        </Form.Item>
      </Form>
    </BaseEditModal>
  );
}
