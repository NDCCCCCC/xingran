import { useEffect, useState } from 'react';
import type { FC } from 'react';
import { Table, Button, Space, Form, Input, Select, Card, Row, Col, Statistic, message } from 'antd';
import {
  PlusOutlined,
  SearchOutlined,
  ReloadOutlined,
  ApartmentOutlined,
  TeamOutlined,
  ImportOutlined,
  ExportOutlined,
} from '@ant-design/icons';
import type { Department } from '@/types';
import { useDeptData } from './hooks';
import { getDeptColumns } from './columns';
import { DeptEditModal } from './modals';
import { STATUS_OPTIONS } from './constants';
import { renderTreeData } from './utils';
import ExcelImport from '@/components/shared/ExcelImport';
import ExcelExport from '@/components/shared/ExcelExport';

const { Option } = Select;

const DepartmentManagement: FC = () => {
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

  useEffect(() => {
    loadDepartments();
  }, [loadDepartments]);

  useEffect(() => {
    if (departments?.length > 0) {
      const treeData = renderTreeData(departments);
      const rootKeys = treeData.map(d => d.key);
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
  };

  const handleDelete = async (id: string) => {
    try {
      const { post } = await import('@/lib/api');
      await post(`/system/departments/${id}/delete`);
      message.success('删除成功');
      loadDepartments();
    } catch (error) {
      console.error('删除部门失败:', error);
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
        phone: selectedUser.phone || '',
        email: selectedUser.email || '',
      });
    }
  };

  const handleSave = async () => {
    try {
      const { post } = await import('@/lib/api');
      const values = await editForm.validateFields();

      const submitData: Record<string, unknown> = {
        deptName: values.deptName,
        parentId: values.parentId || null,
        orderNum: values.orderNum,
        status: values.status,
        isExternalOrg: values.isExternalOrg,
        remark: values.remark || '',
        ...(values.leader && {
          leader: values.leader,
          phone: values.phone || '',
          email: values.email || '',
        }),
      };

      const endpoint = editingDept
        ? `/system/departments/${editingDept.id}/update`
        : '/system/departments';

      await post(endpoint, editingDept ? { ...submitData, id: editingDept.id } : submitData);

      message.success(editingDept ? '更新成功' : '创建成功');
      setEditModalVisible(false);
      loadDepartments();
    } catch (error) {
      console.error('保存部门失败:', error);
    }
  };

  const handleImportSuccess = () => {
    message.success('导入成功');
    loadDepartments();
    setImportVisible(false);
  };

  const getRowClassName = (record: Department): string => {
    return record.accessible === false ? 'dept-inaccessible-row' : '';
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
            <Statistic
              title="总部门数"
              value={statistics.total}
              prefix={<ApartmentOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="顶级部门"
              value={statistics.topLevel}
              styles={{ content: { color: '#3f8600' } }}
              prefix={<TeamOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="子部门"
              value={statistics.subLevel}
              styles={{ content: { color: '#1890ff' } }}
              prefix={<ApartmentOutlined />}
            />
          </Card>
        </Col>
      </Row>

      <Card style={{ marginBottom: 16 }}>
        <div
          style={{
            display: 'flex',
            justifyContent: 'space-between',
            alignItems: 'flex-start',
            flexWrap: 'wrap',
            gap: '16px',
          }}
        >
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="deptName" label="部门名称">
              <Input placeholder="请输入部门名称" />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" style={{ width: 120 }} allowClear>
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
                setExpandedRowKeys(expandedRowKeys.filter(k => k !== recordKey));
              }
            },
            childrenColumnName: 'children',
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
        statusOptions={STATUS_OPTIONS as unknown as { value: number; label: string }[]}
        visible={exportVisible}
        onClose={() => setExportVisible(false)}
      />
    </div>
  );
};

export default DepartmentManagement;
