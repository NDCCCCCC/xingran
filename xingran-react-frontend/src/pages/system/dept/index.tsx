import { useEffect, useState, useCallback } from "react";
import type { FC } from "react";
import { App, Table, Button, Space, Form, Input, Select, Card, Row, Col, Statistic } from "antd";
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  ApartmentOutlined,
  TeamOutlined,
  ImportOutlined,
  ExportOutlined,
} from "@ant-design/icons";
import { useQueryClient } from "@tanstack/react-query";
import type { Department } from "@/types";
import { useDeptData } from "./hooks";
import { getDeptColumns } from "./columns";
import { DeptEditModal } from "./modals";
import { STATUS_OPTIONS } from "./constants";
import { renderTreeData } from "./utils";
import ExcelImport from "@/components/shared/ExcelImport";
import ExcelExport from "@/components/shared/ExcelExport";

import { post } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
const { Option } = Select;

const DepartmentManagement: FC = () => {
  const { message } = App.useApp();
  const [searchForm] = Form.useForm();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingDept, setEditingDept] = useState<Department | null>(null);
  const [editForm] = Form.useForm();
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [expandedRowKeys, setExpandedRowKeys] = useState<string[]>([]);

  const {
    departments,
    parentOptions,
    deptUsers,
    loading,
    loadingUsers,
    statistics,
    setDeptUsers,
    loadDepartments,
    loadDeptUsers,
  } = useDeptData();

  // 全局 dept 缓存失效器：每次部门变更后让所有 useDeptTree() 消费者重新拉取 (D-13 Step 2)
  const qc = useQueryClient();
  const invalidateAllDepts = useCallback(() => {
    qc.invalidateQueries({ queryKey: queryKeys.dept.all });
  }, [qc]);

  useEffect(() => {
    loadDepartments();
  }, [loadDepartments]);

  useEffect(() => {
    if (departments?.length > 0) {
      const treeData = renderTreeData(departments);
      const rootKeys = treeData.map((d) => d.key);
      // eslint-disable-next-line react-hooks/set-state-in-effect -- intentional expansion on data load
      setExpandedRowKeys(rootKeys);
    } else {
      setExpandedRowKeys([]);
    }
  }, [departments]);

  const handleSearch = async () => {
    const values = await searchForm.validateFields();
    loadDepartments(values);
  };

  const handleReset = () => {
    searchForm.resetFields();
    loadDepartments();
  };

  const handleRefresh = () => {
    loadDepartments();
  };

  const handleAdd = () => {
    setEditingDept(null);
    editForm.resetFields();
    setDeptUsers([]);
    setEditModalVisible(true);
  };

  const handleEdit = (record: Department) => {
    setEditingDept(record);
    setEditModalVisible(true);
    loadDeptUsers(record.id);
    // 负责人兜底注入(2026-06-30,同 info-points):loadDeptUsers 异步且可能不包含当前 leader
    // (无该部门成员/leader 离职后) → 负责人 Select 显示 raw UUID。用 record.leaderUsername 注入。
    if (record.leader) {
      // 收窄到局部 const,避免 setState 闭包内 record.leaderName 仍为 string|undefined
      const leaderId: string = record.leader;
      const nickname: string = record.leaderName || "";
      setDeptUsers((prev) =>
        prev.find((u) => u.id === leaderId)
          ? prev
          : [
              ...prev,
              {
                id: leaderId,
                username: record.leaderUsername || record.leaderName || "未命名用户",
                nickname,
              },
            ]
      );
    }
  };

  const handleDelete = async (id: string) => {
    try {
      // ✅ 优化：移除动态导入
      await post(`/system/departments/${id}/delete`);
      message.success("删除成功");
      loadDepartments();
      invalidateAllDepts();
    } catch (error) {
      console.error("删除部门失败:", error);
    }
  };

  const handleParentChange = (value: string) => {
    if (value) {
      loadDeptUsers(value);
    } else {
      setDeptUsers([]);
    }
  };

  const handleLeaderChange = (userId: string) => {
    const selectedUser = deptUsers.find((u) => u.id === userId);
    if (selectedUser) {
      editForm.setFieldsValue({
        phone: selectedUser.phone || "",
        email: selectedUser.email || "",
      });
    }
  };

  const handleSave = async () => {
    try {
      // ✅ 优化：移除动态导入
      const values = await editForm.validateFields();

      const submitData: Record<string, unknown> = {
        deptName: values.deptName,
        deptCode: values.deptCode,
        parentId: values.parentId || null,
        orderNum: values.orderNum,
        status: values.status,
        isExternalOrg: values.isExternalOrg,
        remark: values.remark || "",
        ...(values.leader && {
          leader: values.leader,
          phone: values.phone || "",
          email: values.email || "",
        }),
      };

      const endpoint = editingDept
        ? `/system/departments/${editingDept.id}/update`
        : "/system/departments";

      await post(endpoint, editingDept ? { ...submitData, id: editingDept.id } : submitData);

      message.success(editingDept ? "更新成功" : "创建成功");
      setEditModalVisible(false);
      loadDepartments();
      invalidateAllDepts();
    } catch (error) {
      console.error("保存部门失败:", error);
    }
  };

  const handleImportSuccess = () => {
    message.success("导入成功");
    loadDepartments();
    setImportVisible(false);
    invalidateAllDepts();
  };

  const getRowClassName = (record: Department): string => {
    return record.accessible === false ? "dept-inaccessible-row" : "";
  };

  const columns = getDeptColumns({
    handleEdit,
    handleDelete,
  });

  return (
    <div>
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Statistic title="总部门数" value={statistics.total} prefix={<ApartmentOutlined />} />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="顶级部门"
              value={statistics.topLevel}
              styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
              prefix={<TeamOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="子部门"
              value={statistics.subLevel}
              styles={{ content: { color: "var(--theme-info, #337ab0)" } }}
              prefix={<ApartmentOutlined />}
            />
          </Card>
        </Col>
      </Row>

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
            <Form.Item name="deptName" label="部门名称">
              <Input placeholder="请输入部门名称" />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select
                placeholder="请选择状态"
                style={{ width: 120 }}
                allowClear
                onSearch={() => {}}
              >
                {STATUS_OPTIONS.map((opt) => (
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
            <Button type="primary" icon={<PlusOutlined />} onClick={handleAdd}>
              新增部门
            </Button>
            <Button icon={<ImportOutlined />} onClick={() => setImportVisible(true)}>
              导入
            </Button>
            <Button icon={<ExportOutlined />} onClick={() => setExportVisible(true)}>
              导出
            </Button>
          </Space>
        </div>
      </Card>

      <Card>
        <Table
          columns={columns}
          dataSource={renderTreeData(departments)}
          rowKey="key"
          loading={loading}
          pagination={false}
          rowClassName={(record) => getRowClassName(record)}
          expandable={{
            expandedRowKeys: expandedRowKeys,
            onExpand: (expanded, record) => {
              const recordKey = (record as Department & { key: string }).key;
              if (expanded) {
                setExpandedRowKeys([...expandedRowKeys, recordKey]);
              } else {
                setExpandedRowKeys(expandedRowKeys.filter((k) => k !== recordKey));
              }
            },
            childrenColumnName: "children",
          }}
        />
      </Card>

      <DeptEditModal
        open={editModalVisible}
        editingDept={editingDept}
        parentOptions={parentOptions}
        deptUsers={deptUsers}
        loadingUsers={loadingUsers}
        form={editForm}
        onOk={handleSave}
        onCancel={() => setEditModalVisible(false)}
        onParentChange={handleParentChange}
        onLeaderChange={handleLeaderChange}
      />

      <ExcelImport
        entityType="department"
        entityName="部门"
        templateUrl="/api/v1/system/departments/template"
        importUrl="/api/v1/system/departments/import"
        visible={importVisible}
        onClose={() => setImportVisible(false)}
        onImportSuccess={handleImportSuccess}
      />
      <ExcelExport
        entityType="department"
        entityName="部门"
        exportUrl="/api/v1/system/departments/export"
        visible={exportVisible}
        onClose={() => setExportVisible(false)}
      />
    </div>
  );
};

export default DepartmentManagement;
