import { useEffect, useState, useCallback } from "react";
import type { FC } from "react";
import { App, Table, Button, Space, Form, Card, Row, Col, Statistic } from "antd";
// Phase 71 (W2)：裸 Input/Select → 契约组件（状态枚举 children → options + 禁搜）
import { AppInput, AppSelect } from "@/components/ui";
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
import ExcelImportLazy from "@/components/shared/ExcelImportLazy";
import ExcelExport from "@/components/shared/ExcelExport";

import { post } from "@/lib/api";
import { queryKeys } from "@/lib/queryKeys";
// Phase 86 W1（RBAC-02）：操作级权限门控 —— 单一真相源 + 三态门控 hook。
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { useActionPermission } from "@/components/shared";

const DepartmentManagement: FC = () => {
  const { message } = App.useApp();
  const [searchForm] = Form.useForm();
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [editingDept, setEditingDept] = useState<Department | null>(null);
  const [editForm] = Form.useForm();
  const [importVisible, setImportVisible] = useState(false);
  const [exportVisible, setExportVisible] = useState(false);
  const [expandedRowKeys, setExpandedRowKeys] = useState<string[]>([]);

  // Phase 86 W1（RBAC-02）：操作级权限门控（三态 ready/loading/error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  // 新增 → system:dept:add；导入 → system:dept:import；导出 → system:dept:export。
  const permAdd = permGate.decide(PERMISSION_CODES.systemDept.add);
  const permImport = permGate.decide(PERMISSION_CODES.systemDept.import);
  const permExport = permGate.decide(PERMISSION_CODES.systemDept.export);

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
              styles={{ content: { color: "var(--theme-success)" } }}
              prefix={<TeamOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="子部门"
              value={statistics.subLevel}
              styles={{ content: { color: "var(--theme-info)" } }}
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
              <AppInput placeholder="请输入部门名称" />
            </Form.Item>
            <Form.Item name="status" label="状态">
              {/* 形态分类器②④：部门状态枚举（业务禁搜）→ options 形态 + 显式 showSearch={false} */}
              <AppSelect
                placeholder="请选择状态"
                style={{ width: 120 }}
                allowClear
                onSearch={() => {}}
                showSearch={false}
                options={STATUS_OPTIONS}
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
            {/* Phase 86 W1（RBAC-02）：Excel 入口的门控点必须落在**触发按钮**上。
                下方 <ExcelImportLazy> / <ExcelExport> 是 modal 容器（visible 由按钮状态驱动），
                触发按钮不渲染 ⇒ 弹窗永远打不开；二者等价于「按钮不渲染」的可观察行为。 */}
            {permAdd.visible && (
              <Button
                type="primary"
                icon={<PlusOutlined />}
                onClick={handleAdd}
                disabled={permAdd.disabled}
                title={permAdd.hint}
              >
                新增部门
              </Button>
            )}
            {permImport.visible && (
              <Button
                icon={<ImportOutlined />}
                onClick={() => setImportVisible(true)}
                disabled={permImport.disabled}
                title={permImport.hint}
              >
                导入
              </Button>
            )}
            {permExport.visible && (
              <Button
                icon={<ExportOutlined />}
                onClick={() => setExportVisible(true)}
                disabled={permExport.disabled}
                title={permExport.hint}
              >
                导出
              </Button>
            )}
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

      <ExcelImportLazy
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
