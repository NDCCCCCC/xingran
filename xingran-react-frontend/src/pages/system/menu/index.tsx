/**
 * 系统菜单管理页面
 * System Menu Management Page
 */

import { useState, useEffect, type FC } from "react";
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Tag,
  Card,
  Row,
  Col,
  Statistic,
  InputNumber,
  Switch,
  TreeSelect,
  Alert,
} from "antd";
// Phase 71 (W2)：裸 Input/Select/Input.TextArea → 契约组件。菜单类型/状态为固定枚举
// → children 改写 options 形态；状态类业务下拉显式 showSearch={false}
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import {
  PlusOutlined,
  EditOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  MenuOutlined,
  FolderOutlined,
  FileOutlined,
  AppstoreOutlined,
} from "@ant-design/icons";
import type { ColumnsType } from "antd/es/table";
import type { Menu } from "@/types";
import IconSelect from "@/components/IconSelect";
import { formatDateTime } from "@/utils/datetime";

// 导入提取的模块
import {
  MENU_TYPE_OPTIONS,
  MENU_STATUS_OPTIONS,
  getMenuTypeTag,
  DEFAULT_FORM_VALUES,
} from "./constants";
import { renderTreeData, renderMenuName } from "./utils";
import { useMenuData, useMenuActions } from "./hooks";
// Phase 86 W1（RBAC-02）：操作级权限门控 —— 单一真相源 + 三态门控 hook。
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { useActionPermission } from "@/components/shared";
import type { ActionPermissionDecision } from "@/components/shared";

// ==================== 表格列定义 ====================

interface MenuTableColumnsProps {
  handleEdit: (record: Menu) => void;
  handleDeleteConfirm: (record: Menu) => void;
  /**
   * Phase 86 W1（RBAC-02）：操作列门控决策（由页面层 decide 后透传，列工厂保持纯函数、
   * 不在内部调 hook）。
   * 三态：visible=false → 不渲染；disabled=true → 渲染但禁用（hint 作提示）。
   */
  permEdit: ActionPermissionDecision;
  permRemove: ActionPermissionDecision;
}

function getMenuTableColumns(props: MenuTableColumnsProps): ColumnsType<Menu> {
  const { handleEdit, handleDeleteConfirm, permEdit, permRemove } = props;

  return [
    {
      title: "菜单名称",
      dataIndex: "menuName",
      key: "menuName",
      render: (_: unknown, record: Menu) => renderMenuName(record),
    },
    {
      title: "菜单类型",
      dataIndex: "menuType",
      key: "menuType",
      render: (menuType) => getMenuTypeTag(menuType),
    },
    {
      title: "权限标识",
      dataIndex: "perms",
      key: "perms",
      render: (perms) => perms && <Tag color="cyan">{perms}</Tag>,
    },
    {
      title: "路由地址",
      dataIndex: "path",
      key: "path",
    },
    {
      title: "组件路径",
      dataIndex: "component",
      key: "component",
    },
    {
      title: "显示顺序",
      dataIndex: "orderNum",
      key: "orderNum",
    },
    {
      title: "显示状态",
      dataIndex: "visible",
      key: "visible",
      render: (visible) => (
        <Tag color={visible === 1 ? "success" : "default"}>{visible === 1 ? "显示" : "隐藏"}</Tag>
      ),
    },
    {
      title: "菜单状态",
      dataIndex: "status",
      key: "status",
      render: (status) => (
        <Tag color={status === 0 ? "success" : "error"}>{status === 0 ? "正常" : "停用"}</Tag>
      ),
    },
    {
      title: "创建时间",
      dataIndex: "createdAt",
      key: "createdAt",
      render: (text) => formatDateTime(text),
    },
    {
      title: "操作",
      key: "action",
      render: (_: unknown, record: Menu) => (
        <Space size="medium">
          {/* Phase 86 W1：行编辑 → system:menu:edit；行删除 → system:menu:remove。
              ready 无权限 → 入口不在 DOM；未就绪 → 渲染但 disabled + 提示。 */}
          {permEdit.visible && (
            <Button
              type="link"
              icon={<EditOutlined />}
              onClick={() => handleEdit(record)}
              disabled={permEdit.disabled}
              title={permEdit.hint}
            >
              编辑
            </Button>
          )}
          {permRemove.visible && (
            <Button
              type="link"
              icon={<DeleteOutlined />}
              style={{ color: "var(--theme-error)" }}
              onClick={() => handleDeleteConfirm(record)}
              disabled={permRemove.disabled}
              title={permRemove.hint}
            >
              删除
            </Button>
          )}
        </Space>
      ),
    },
  ];
}

// ==================== 主组件 ====================

const MenuManagement: FC = () => {
  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);

  // Phase 86 W1（RBAC-02）：操作级权限门控（三态 ready/loading/error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  const permAdd = permGate.decide(PERMISSION_CODES.systemMenu.add);
  const permEdit = permGate.decide(PERMISSION_CODES.systemMenu.edit);
  const permRemove = permGate.decide(PERMISSION_CODES.systemMenu.remove);

  // 使用数据管理 Hook
  const { menus, parentOptions, statistics, loading, loadMenus } = useMenuData();

  // 使用操作管理 Hook
  const {
    editingMenu,
    cascadeDelete: _cascadeDelete,
    setCascadeDelete: _setCascadeDelete,
    handleAdd,
    handleEdit,
    handleDeleteConfirm,
    handleBatchDelete,
    handleSave,
    setEditingMenu,
  } = useMenuActions({
    onLoad: loadMenus,
    selectedRowKeys,
    setSelectedRowKeys,
    onSaveSuccess: () => {
      setEditModalVisible(false);
    },
  });

  // 弹窗内「显示状态 / 菜单状态」两个 Switch 是**表单字段**（随弹窗提交生效），
  // 归所属写动作：编辑弹窗 → system:menu:edit；新增弹窗 → system:menu:add。
  // 门控点落在 Switch 本身：ready 无权限 → 整个 Switch 不渲染（不是只禁用）。
  const permFormStatus = permGate.decide(
    editingMenu ? PERMISSION_CODES.systemMenu.edit : PERMISSION_CODES.systemMenu.add
  );

  // 初始化加载
  useEffect(() => {
    loadMenus();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- run once on mount
  }, []);

  // 搜索
  const handleSearch = async () => {
    const values = searchForm.getFieldsValue();
    loadMenus(values);
  };

  // 重置搜索
  const handleReset = () => {
    searchForm.resetFields();
    loadMenus();
  };

  // 刷新
  const handleRefresh = () => {
    loadMenus();
  };

  // 打开新增弹窗
  const handleOpenAddModal = () => {
    handleAdd();
    editForm.resetFields();
    editForm.setFieldsValue(DEFAULT_FORM_VALUES);
    setEditModalVisible(true);
  };

  // 打开编辑弹窗
  const handleOpenEditModal = (record: Menu) => {
    handleEdit(record);
    setEditModalVisible(true);
  };

  // Modal 打开后的回调
  const handleModalOpenChange = (open: boolean) => {
    if (open && editingMenu) {
      // Modal 打开时设置表单值
      // visible: 1->true(显示), 0->false(隐藏)
      // status: 0->true(正常), 1->false(停用)
      editForm.setFieldsValue({
        ...editingMenu,
        visible: editingMenu.visible === 1,
        status: editingMenu.status === 0,
      });
    }
  };

  // 表格列
  const columns = getMenuTableColumns({
    handleEdit: handleOpenEditModal,
    handleDeleteConfirm,
    permEdit,
    permRemove,
  });

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={6}>
          <Card>
            <Statistic title="总菜单数" value={statistics.total} prefix={<MenuOutlined />} />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="目录数量"
              value={statistics.directories}
              styles={{ content: { color: "var(--theme-info)" } }}
              prefix={<FolderOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="菜单数量"
              value={statistics.menus}
              styles={{ content: { color: "var(--theme-success)" } }}
              prefix={<FileOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="按钮数量"
              value={statistics.buttons}
              styles={{ content: { color: "var(--theme-warning)" } }}
              prefix={<AppstoreOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
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
            <Form.Item name="menuName" label="菜单名称">
              <AppInput placeholder="请输入菜单名称" />
            </Form.Item>
            <Form.Item name="status" label="菜单状态">
              {/* 形态分类器②④：菜单状态枚举（业务禁搜） */}
              <AppSelect
                placeholder="请选择状态"
                style={{ width: 120 }}
                allowClear
                onSearch={() => {}}
                showSearch={false}
                options={MENU_STATUS_OPTIONS}
              />
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
            {/* Phase 86 W1：工具栏写入口按权限码条件渲染（ready 无权限 → 不渲染；未就绪 → 渲染但 disabled + 提示） */}
            {selectedRowKeys.length > 0 && permRemove.visible && (
              <Button
                icon={<DeleteOutlined />}
                style={{ color: "var(--theme-error)" }}
                onClick={handleBatchDelete}
                disabled={permRemove.disabled}
                title={permRemove.hint}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
            {permAdd.visible && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={handleOpenAddModal}
                disabled={permAdd.disabled}
                title={permAdd.hint}
              >
                新增菜单
              </Button>
            )}
          </Space>
        </div>
        {selectedRowKeys.length > 0 && (
          <Alert
            title={
              <span>
                已选择 <strong>{selectedRowKeys.length}</strong> 个菜单，
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

      {/* 菜单表格 */}
      <Card>
        <Table
          columns={columns}
          dataSource={renderTreeData(menus)}
          rowKey="id"
          loading={loading}
          pagination={false}
          rowSelection={{
            selectedRowKeys,
            onChange: (selectedKeys) => setSelectedRowKeys(selectedKeys as string[]),
            selections: [Table.SELECTION_ALL, Table.SELECTION_INVERT, Table.SELECTION_NONE],
          }}
          expandable={{
            defaultExpandAllRows: true,
          }}
        />
      </Card>

      {/* 编辑弹窗 */}
      <Modal
        title={editingMenu ? "编辑菜单" : "新增菜单"}
        open={editModalVisible}
        onOk={async () => {
          setSubmitting(true);
          try {
            await handleSave(editForm);
          } finally {
            setSubmitting(false);
          }
        }}
        cancelButtonProps={{ disabled: submitting }}
        afterOpenChange={handleModalOpenChange}
        onCancel={() => {
          setEditModalVisible(false);
          setEditingMenu(null);
        }}
        width={600}
      >
        <Form form={editForm} layout="vertical">
          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="parentId" label="上级菜单">
                <TreeSelect
                  style={{ width: "100%" }}
                  styles={{ popup: { root: { maxHeight: 400, overflow: "auto" } } }}
                  treeData={parentOptions}
                  placeholder="请选择上级菜单"
                  treeDefaultExpandAll
                  allowClear
                />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="menuType"
                label="菜单类型"
                rules={[{ required: true, message: "请选择菜单类型" }]}
              >
                {/* 形态分类器②④：菜单类型固定 3 项（业务禁搜）。MENU_TYPE_OPTIONS 为 as const
                    只读元组 → 展开复制以匹配 antd 可变数组类型 */}
                <AppSelect
                  allowClear
                  placeholder="请选择菜单类型"
                  onSearch={() => {}}
                  showSearch={false}
                  options={[...MENU_TYPE_OPTIONS]}
                />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item
                name="menuName"
                label="菜单名称"
                rules={[{ required: true, message: "请输入菜单名称" }]}
              >
                <AppInput />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item
                name="orderNum"
                label="显示顺序"
                rules={[{ required: true, message: "请输入显示顺序" }]}
              >
                <InputNumber min={0} style={{ width: "100%" }} />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="path" label="路由地址">
                <AppInput placeholder="请输入路由地址" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="component" label="组件路径">
                <AppInput placeholder="请输入组件路径" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="perms" label="权限标识">
                <AppInput placeholder="请输入权限标识" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="icon" label="菜单图标">
                <IconSelect placeholder="请选择菜单图标" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              {/* Phase 86 W1：显示状态 Switch 是弹窗表单字段，随「新增/编辑菜单」提交生效，
                  归所属写动作的权限码；ready 无权限 → 整个 Switch 不渲染（不是只禁用）。 */}
              {permFormStatus.visible && (
                <Form.Item
                  name="visible"
                  label="显示状态"
                  valuePropName="checked"
                  initialValue={true}
                >
                  <Switch
                    checkedChildren="显示"
                    unCheckedChildren="隐藏"
                    disabled={permFormStatus.disabled}
                    title={permFormStatus.hint}
                  />
                </Form.Item>
              )}
            </Col>
            <Col span={12}>
              {/* Phase 86 W1：菜单状态 Switch 同上（与显示状态同归所属写动作）。 */}
              {permFormStatus.visible && (
                <Form.Item name="status" label="菜单状态" valuePropName="checked" initialValue={true}>
                  <Switch
                    checkedChildren="正常"
                    unCheckedChildren="停用"
                    disabled={permFormStatus.disabled}
                    title={permFormStatus.hint}
                  />
                </Form.Item>
              )}
            </Col>
          </Row>

          <Form.Item name="remark" label="备注">
            <AppTextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default MenuManagement;
