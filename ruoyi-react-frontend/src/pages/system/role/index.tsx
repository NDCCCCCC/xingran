/**
 * 角色管理页面
 * Role Management Page
 */

import { useState, useEffect, useCallback, useRef, type FC } from 'react';
import {
  Table,
  Button,
  Space,
  Modal,
  Form,
  Input,
  Select,
  Card,
  Row,
  Col,
  Statistic,
  Tree,
} from 'antd';
import {
  PlusOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from '@ant-design/icons';
import type { Key } from 'antd/es/table/interface';

import { post } from '@/lib/api';
import { STATUS_OPTIONS, DATA_SCOPE_OPTIONS, DEFAULT_FORM_VALUES } from './constants';
import { getRoleColumns } from './columns';
import { useRoleData, useRoleActions } from './hooks';
import { usePagination } from '@/hooks/usePagination';

const { Option } = Select;

// ==================== 主组件 ====================

const RoleManagement: FC = () => {
  const [searchForm] = Form.useForm();
  const [editForm] = Form.useForm();

  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 使用数据管理 Hook
  const {
    roles,
    loading,
    total,
    menuTree,
    deptTree,
    statistics,
    checkedMenuKeys,
    checkedDeptKeys,
    currentDataScope,
    setCheckedMenuKeys,
    setCheckedDeptKeys,
    setCurrentDataScope,
    loadRoles,
    loadStatistics,
    loadMenuTree,
    loadDeptTree,
  } = useRoleData({
    current: paginationProps.current ?? 1,
    pageSize: paginationProps.pageSize ?? 10,
  });

  // 防重复调用追踪器
  const loadingMenusRef = useRef(new Set<string>());
  const loadingDeptsRef = useRef(new Set<string>());

  // 使用 useCallback 包装的 loadRoleMenus - 稳定引用 + 防重复调用
  const loadRoleMenus = useCallback(async (roleId: string) => {
    // 参数验证
    if (!roleId || roleId === 'undefined' || roleId === 'null') {
      console.warn('[loadRoleMenus] 无效的 roleId:', roleId);
      return [];
    }

    // 防重复调用检查
    if (loadingMenusRef.current.has(roleId)) {
      console.warn('[loadRoleMenus] 正在加载中，跳过重复请求:', roleId);
      return [];
    }

    loadingMenusRef.current.add(roleId);
    console.log('[loadRoleMenus] 开始加载角色菜单权限:', roleId);

    try {
      const result = await post(`/system/menus/role-menu-tree-select/${roleId}`) as { data: { checkedKeys: string[] } };
      const checkedKeys = result.data.checkedKeys || [];
      console.log('[loadRoleMenus] 加载完成，权限数量:', checkedKeys.length, 'roleId:', roleId);
      return checkedKeys;
    } catch (error) {
      console.error('[loadRoleMenus] 加载失败:', error);
      return [];
    } finally {
      loadingMenusRef.current.delete(roleId);
    }
  }, []); // 空依赖数组，保证函数引用稳定

  // 使用 useCallback 包装的 loadRoleDepts - 保持一致性
  const loadRoleDepts = useCallback(async (roleId: string) => {
    if (!roleId || roleId === 'undefined' || roleId === 'null') {
      console.warn('[loadRoleDepts] 无效的 roleId:', roleId);
      return [];
    }

    if (loadingDeptsRef.current.has(roleId)) {
      console.warn('[loadRoleDepts] 正在加载中，跳过重复请求:', roleId);
      return [];
    }

    loadingDeptsRef.current.add(roleId);
    console.log('[loadRoleDepts] 开始加载角色部门权限:', roleId);

    try {
      const result = await post(`/system/departments/role-dept-tree-select/${roleId}`) as { data: { checkedKeys: string[] } };
      const checkedKeys = result.data.checkedKeys || [];
      console.log('[loadRoleDepts] 加载完成，权限数量:', checkedKeys.length, 'roleId:', roleId);
      return checkedKeys;
    } catch (error) {
      console.error('[loadRoleDepts] 加载失败:', error);
      return [];
    } finally {
      loadingDeptsRef.current.delete(roleId);
    }
  }, []);

  // 使用操作管理 Hook
  const {
    editingRole,
    editModalVisible,
    pendingFormData,
    selectedRowKeys,
    setEditingRole,
    setEditModalVisible,
    setSelectedRowKeys,
    handleAdd,
    handleEdit,
    handleDelete,
    handleBatchDelete,
    handleUpdateStatus,
    handleSave,
    handleDataScopeChange,
    handleModalOpenChange,
  } = useRoleActions({
    loadRoles,
    loadStatistics,
    loadRoleMenus,
    loadRoleDepts,
    checkedMenuKeys,
    checkedDeptKeys,
    setCheckedMenuKeys,
    setCheckedDeptKeys,
    currentDataScope,
    setCurrentDataScope,
  });

  // 初始化加载
  useEffect(() => {
    loadRoles();
    loadStatistics();
    loadMenuTree();
    loadDeptTree();
  }, [paginationProps.current, paginationProps.pageSize]);

  // 搜索
  const handleSearch = () => {
    const values = searchForm.getFieldsValue();
    loadRoles(values);
  };

  // 重置搜索
  const handleReset = () => {
    searchForm.resetFields();
    loadRoles();
  };

  // 刷新
  const handleRefresh = () => {
    loadRoles();
    loadStatistics();
  };

  // 表格列
  const columns = getRoleColumns({
    handleEdit,
    handleUpdateStatus,
    handleDelete,
  });

  return (
    <div>
      {/* 统计卡片 */}
      <Row gutter={16} style={{ marginBottom: 16 }}>
        <Col span={8}>
          <Card>
            <Statistic
              title="总角色数"
              value={statistics.total}
              prefix={<SafetyCertificateOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="正常角色"
              value={statistics.active}
              styles={{ content: { color: '#3f8600' } }}
              prefix={<SafetyCertificateOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="停用角色"
              value={statistics.inactive}
              styles={{ content: { color: '#cf1322' } }}
              prefix={<DeleteOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', flexWrap: 'wrap', gap: '16px' }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="roleName" label="角色名称">
              <Input placeholder="请输入角色名称" />
            </Form.Item>
            <Form.Item name="roleKey" label="权限字符">
              <Input placeholder="请输入权限字符" />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" style={{ width: 120 }} allowClear>
                {STATUS_OPTIONS.map(opt => (
                  <Option key={opt.value} value={opt.value}>{opt.label}</Option>
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
              新增角色
            </Button>
            {selectedRowKeys.length > 0 && (
              <Button
                icon={<DeleteOutlined />} style={{ color: '#ff4d4f' }}
                onClick={() => handleBatchDelete(selectedRowKeys)}
              >
                批量删除 ({selectedRowKeys.length})
              </Button>
            )}
          </Space>
        </div>
      </Card>

      {/* 角色表格 */}
      <Card>
        <Table
          columns={columns}
          dataSource={roles}
          rowKey="id"
          loading={loading}
          pagination={paginationProps}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
        />
      </Card>

      {/* 编辑弹窗 */}
      <Modal
        title={editingRole ? '编辑角色' : '新增角色'}
        open={editModalVisible}
        onOk={() => handleSave(editForm)}
        afterOpenChange={(open) => handleModalOpenChange(open, editForm)}
        onCancel={() => setEditModalVisible(false)}
        width={800}
        styles={{ body: { maxHeight: '70vh', overflowY: 'auto' } }}
      >
        <Form form={editForm} layout="vertical" initialValues={DEFAULT_FORM_VALUES}>
          {/* 基本信息 */}
          <Card title="基本信息" size="small" style={{ marginBottom: 16 }}>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  name="roleName"
                  label="角色名称"
                  rules={[{ required: true, message: '请输入角色名称' }]}
                >
                  <Input />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="roleKey"
                  label="权限字符"
                  rules={[{ required: true, message: '请输入权限字符' }]}
                >
                  <Input />
                </Form.Item>
              </Col>
            </Row>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  name="roleSort"
                  label="显示顺序"
                  rules={[{ required: true, message: '请输入显示顺序' }]}
                >
                  <Input type="number" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="status"
                  label="状态"
                >
                  <Select>
                    {STATUS_OPTIONS.map(opt => (
                      <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                    ))}
                  </Select>
                </Form.Item>
              </Col>
            </Row>
            <Form.Item name="remark" label="备注">
              <Input.TextArea rows={2} />
            </Form.Item>
          </Card>

          {/* 菜单权限和数据权限横向排列 */}
          <Row gutter={16}>
            {/* 左侧：菜单权限 */}
            <Col span={12}>
              <Card title="菜单权限" size="small">
                <Form.Item
                  name="menuIds"
                  label="权限菜单"
                  tooltip="选择角色可以访问的菜单（勾选父菜单时自动勾选所有子菜单）"
                >
                  <Tree
                    checkable
                    checkedKeys={checkedMenuKeys}
                    onCheck={(keys) => {
                      const keyArray = (keys as unknown) as Key[];
                      setCheckedMenuKeys(keyArray);
                      editForm.setFieldsValue({ menuIds: keyArray });
                    }}
                    treeData={menuTree}
                    style={{ background: '#f5f5f5', padding: '8px', borderRadius: '4px', maxHeight: '400px', overflowY: 'auto' }}
                  />
                </Form.Item>
              </Card>
            </Col>

            {/* 右侧：数据权限 */}
            <Col span={12}>
              <Card title="数据权限" size="small">
                <Form.Item
                  name="dataScope"
                  label="数据范围"
                >
                  <Select onChange={(value) => handleDataScopeChange(value, editForm)}>
                    {DATA_SCOPE_OPTIONS.map(opt => (
                      <Option key={opt.value} value={opt.value}>{opt.label}</Option>
                    ))}
                  </Select>
                </Form.Item>

                {currentDataScope === 2 && (
                  <Form.Item
                    name="deptIds"
                    label="部门权限"
                    tooltip="选择角色可以访问的部门数据（勾选父部门时自动勾选所有子部门）"
                  >
                    <Tree
                      checkable
                      checkedKeys={checkedDeptKeys}
                      onCheck={(keys) => {
                        const keyArray = (keys as unknown) as Key[];
                        setCheckedDeptKeys(keyArray);
                        editForm.setFieldsValue({ deptIds: keyArray });
                      }}
                      treeData={deptTree}
                      style={{ background: '#f5f5f5', padding: '8px', borderRadius: '4px', maxHeight: '400px', overflowY: 'auto' }}
                    />
                  </Form.Item>
                )}
              </Card>
            </Col>
          </Row>
        </Form>
      </Modal>
    </div>
  );
};

export default RoleManagement;

