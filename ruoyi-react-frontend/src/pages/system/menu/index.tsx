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
  Input,
  Select,
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

const { Option } = Select;

// ==================== 表格列定义 ====================

interface MenuTableColumnsProps {
  handleEdit: (record: Menu) => void;
  handleDeleteConfirm: (record: Menu) => void;
}

function getMenuTableColumns(props: MenuTableColumnsProps): ColumnsType<Menu> {
  const { handleEdit, handleDeleteConfirm } = props;

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
        <Space size="middle">
          <Button type="link" icon={<EditOutlined />} onClick={() => handleEdit(record)}>
            编辑
          </Button>
          <Button
            type="link"
            icon={<DeleteOutlined />}
            style={{ color: "var(--theme-error, #ba3630)" }}
            onClick={() => handleDeleteConfirm(record)}
          >
            删除
          </Button>
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
  const [selectedRowKeys, setSelectedRowKeys] = useState<string[]>([]);

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
  const columns = getMenuTableColumns({ handleEdit: handleOpenEditModal, handleDeleteConfirm });

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
              styles={{ content: { color: "var(--theme-info, #337ab0)" } }}
              prefix={<FolderOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="菜单数量"
              value={statistics.menus}
              styles={{ content: { color: "var(--theme-success, #2d8949)" } }}
              prefix={<FileOutlined />}
            />
          </Card>
        </Col>
        <Col span={6}>
          <Card>
            <Statistic
              title="按钮数量"
              value={statistics.buttons}
              styles={{ content: { color: "#fa8c16" } }}
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
              <Input placeholder="请输入菜单名称" />
            </Form.Item>
            <Form.Item name="status" label="菜单状态">
              <Select
                placeholder="请选择状态"
                style={{ width: 120 }}
                allowClear
                onSearch={() => {}}
              >
                {MENU_STATUS_OPTIONS.map((opt) => (
                  <Option key={opt.value} value={opt.value}>
                    {opt.label}
                  </Option>
                ))}
              </Select>
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
            {selectedRowKeys.length > 0 && (
              <Button
                icon={<DeleteOutlined />}
                style={{ color: "var(--theme-error, #ba3630)" }}
                onClick={handleBatchDelete}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
            <Button type="primary" icon={<PlusOutlined />} onClick={handleOpenAddModal}>
              新增菜单
            </Button>
          </Space>
        </div>
        {selectedRowKeys.length > 0 && (
          <Alert
            message={
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
        onOk={() => handleSave(editForm)}
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
                <Select placeholder="请选择菜单类型" onSearch={() => {}}>
                  {MENU_TYPE_OPTIONS.map((opt) => (
                    <Option key={opt.value} value={opt.value}>
                      {opt.label}
                    </Option>
                  ))}
                </Select>
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
                <Input />
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
                <Input placeholder="请输入路由地址" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="component" label="组件路径">
                <Input placeholder="请输入组件路径" />
              </Form.Item>
            </Col>
          </Row>

          <Row gutter={16}>
            <Col span={12}>
              <Form.Item name="perms" label="权限标识">
                <Input placeholder="请输入权限标识" />
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
              <Form.Item
                name="visible"
                label="显示状态"
                valuePropName="checked"
                initialValue={true}
              >
                <Switch checkedChildren="显示" unCheckedChildren="隐藏" />
              </Form.Item>
            </Col>
            <Col span={12}>
              <Form.Item name="status" label="菜单状态" valuePropName="checked" initialValue={true}>
                <Switch checkedChildren="正常" unCheckedChildren="停用" />
              </Form.Item>
            </Col>
          </Row>

          <Form.Item name="remark" label="备注">
            <Input.TextArea rows={4} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default MenuManagement;
