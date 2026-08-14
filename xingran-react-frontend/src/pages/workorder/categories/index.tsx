import { useState, useEffect } from "react";
import type { FC } from "react";
import { Button, Form, Input, Modal, InputNumber, Select, Space, Tag, App, Card, Tree, Radio } from "antd";
import type { DataNode } from "antd/es/tree";
import { PlusOutlined, EditOutlined, DeleteOutlined, ReloadOutlined, FolderOutlined } from "@ant-design/icons";
import {
  getWorkOrderCategoryList,
  createWorkOrderCategory,
  updateWorkOrderCategory,
  deleteWorkOrderCategory,
  type WorkOrderCategory,
  type WorkOrderCategoryCreateRequest,
  type WorkOrderCategoryUpdateRequest,
} from "@/lib/workorderApi";
import { isFormValidationError } from "@/utils/errorHandler";

const { Option } = Select;

const WorkOrderCategoryPage: FC = () => {
  const { message } = App.useApp();
  const [editForm] = Form.useForm();

  const [categories, setCategories] = useState<WorkOrderCategory[]>([]);
  const [treeData, setTreeData] = useState<DataNode[]>([]);

  const [modalVisible, setModalVisible] = useState(false);
  const [editingRecord, setEditingRecord] = useState<WorkOrderCategory | null>(null);

  // 统一错误处理
  const handleApiError = (error: unknown, defaultMessage: string) => {
    if (error && typeof error === "object" && "message" in error) {
      message.error(error.message as string);
    } else {
      message.error(defaultMessage);
    }
  };

  const handleSuccess = (msg: string) => {
    message.success(msg);
  };

  // 获取分类列表
  const fetchList = async () => {
    try {
      const result = await getWorkOrderCategoryList();
      setCategories(result.data || []);

      // 后端返回的是嵌套结构，直接转换为树形数据
      const buildTree = (list: WorkOrderCategory[]): DataNode[] => {
        return list.map((item) => ({
          key: item.id,
          title: (
            <div className="flex items-center justify-between">
              <span>{item.categoryName}</span>
              <Space size="small">
                <Tag color={item.status === 0 ? "green" : "red"}>
                  {item.status === 0 ? "启用" : "停用"}
                </Tag>
                <span className="text-gray-400 text-xs">排序: {item.sortOrder}</span>
              </Space>
            </div>
          ),
          data: item,
          children: item.children && item.children.length > 0 ? buildTree(item.children) : undefined,
        }));
      };

      setTreeData(buildTree(result.data || []));
    } catch (error) {
      handleApiError(error, "获取分类列表失败");
    }
  };

  useEffect(() => {
    // 使用 setTimeout 避免同步 setState
    setTimeout(() => fetchList(), 0);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const handleAdd = (parentId?: string) => {
    setEditingRecord(null);
    editForm.resetFields();
    editForm.setFieldsValue({
      parentId,
      status: 0,
      sortOrder: 0,
    });
    setModalVisible(true);
  };

  const handleEdit = (record: WorkOrderCategory) => {
    setEditingRecord(record);
    setModalVisible(true);
    setTimeout(() => {
      editForm.setFieldsValue({
        categoryName: record.categoryName,
        description: record.description,
        parentId: record.parentId,
        sortOrder: record.sortOrder,
        status: record.status,
      });
    }, 0);
  };

  const handleDelete = async (id: string) => {
    try {
      await deleteWorkOrderCategory(id);
      handleSuccess("删除成功");
      fetchList();
    } catch (error) {
      handleApiError(error, "删除失败");
    }
  };

  const handleModalOk = async () => {
    try {
      const values = await editForm.validateFields();

      if (editingRecord) {
        const updateData: WorkOrderCategoryUpdateRequest = {
          categoryName: values.categoryName,
          description: values.description,
          parentId: values.parentId,
          sortOrder: values.sortOrder,
          status: values.status,
        };
        await updateWorkOrderCategory(editingRecord.id, updateData);
        handleSuccess("更新成功");
      } else {
        const createData: WorkOrderCategoryCreateRequest = {
          categoryName: values.categoryName,
          description: values.description,
          parentId: values.parentId,
          sortOrder: values.sortOrder,
          status: values.status,
        };
        await createWorkOrderCategory(createData);
        handleSuccess("创建成功");
      }
      setModalVisible(false);
      fetchList();
    } catch (error: unknown) {
      if (isFormValidationError(error)) {
        return;
      }
      handleApiError(error, editingRecord ? "更新失败" : "创建失败");
    }
  };

  // 扁平化分类树（用于父分类选择）
  const flatCategories = (list: WorkOrderCategory[]): WorkOrderCategory[] => {
    const result: WorkOrderCategory[] = [];
    list.forEach((cat) => {
      result.push(cat);
      if (cat.children && cat.children.length > 0) {
        result.push(...flatCategories(cat.children));
      }
    });
    return result;
  };

  return (
    <div className="p-6">
      <Card title="工单分类管理">
        <div className="mb-4">
          <Space>
            <Button type="primary" icon={<PlusOutlined />} onClick={() => handleAdd()}>
              新增根分类
            </Button>
            <Button icon={<ReloadOutlined />} onClick={fetchList}>
              刷新
            </Button>
          </Space>
        </div>

        <Tree
          showLine
          switcherIcon={<FolderOutlined />}
          treeData={treeData}
          defaultExpandAll
          titleRender={(node: DataNode & { data?: WorkOrderCategory }) => {
            const title = typeof node.title === "function" ? node.title(node) : node.title;
            return (
              <div className="flex items-center justify-between w-full pr-4">
                {title}
                <Space size="small">
                  <Button
                    type="link"
                    size="small"
                    icon={<PlusOutlined />}
                    onClick={(e) => {
                      e.stopPropagation();
                      handleAdd(node.data?.id || "");
                    }}
                  >
                    新增子分类
                  </Button>
                  <Button
                    type="link"
                    size="small"
                    icon={<EditOutlined />}
                    onClick={(e) => {
                      e.stopPropagation();
                      if (node.data) {
                        handleEdit(node.data);
                      }
                    }}
                  >
                    编辑
                  </Button>
                  <Button
                    type="link"
                    size="small"
                    icon={<DeleteOutlined />} style={{ color: "var(--theme-error, #ff4d4f)" }}
                    onClick={(e) => {
                      e.stopPropagation();
                      Modal.confirm({
                        title: "确定要删除吗？",
                        okText: "确定",
                        cancelText: "取消",
                        okButtonProps: { danger: true },
                        onOk: () => handleDelete(node.data?.id || ""),
                      });
                    }}
                  >
                    删除
                  </Button>
                </Space>
              </div>
            );
          }}
        />
      </Card>

      {/* 新增/编辑弹窗 */}
      <Modal
        title={editingRecord ? "编辑分类" : "新增分类"}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        width={600}
        destroyOnHidden
      >
        <Form form={editForm} layout="vertical" preserve={false}>
          <Form.Item
            name="categoryName"
            label="分类名称"
            rules={[{ required: true, message: "请输入分类名称" }]}
          >
            <Input placeholder="请输入分类名称" />
          </Form.Item>

          <Form.Item name="parentId" label="父分类">
            <Select placeholder="请选择父分类" allowClear onSearch={() => {}}>
              {flatCategories(categories)
                .filter((cat) => cat.id !== editingRecord?.id) // 不能选择自己作为父分类
                .map((cat) => (
                  <Option key={cat.id} value={cat.id}>
                    {cat.categoryName}
                  </Option>
                ))}
            </Select>
          </Form.Item>

          <Form.Item
            name="sortOrder"
            label="排序"
            rules={[{ required: true, message: "请输入排序" }]}
          >
            <InputNumber min={0} placeholder="请输入排序" style={{ width: "100%" }} />
          </Form.Item>

          <Form.Item
            name="status"
            label="状态"
            rules={[{ required: true, message: "请选择状态" }]}
          >
            <Radio.Group>
              <Radio value={0}>启用</Radio>
              <Radio value={1}>停用</Radio>
            </Radio.Group>
          </Form.Item>

          <Form.Item name="description" label="描述">
            <Input.TextArea rows={3} placeholder="请输入描述" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default WorkOrderCategoryPage;

