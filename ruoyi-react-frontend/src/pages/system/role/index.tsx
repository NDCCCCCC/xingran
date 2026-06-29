/**
 * 角色管理页面
 * Role Management Page
 */

import { useState, useEffect, useCallback, useRef, useMemo, type FC } from "react";
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
} from "antd";
import {
  PlusOutlined,
  DeleteOutlined,
  SearchOutlined,
  ReloadOutlined,
  SafetyCertificateOutlined,
} from "@ant-design/icons";
import type { Key } from "antd/es/table/interface";

import { post } from "@/lib/api";
import type { Role } from "@/types";
import { STATUS_OPTIONS, DATA_SCOPE_OPTIONS, DEFAULT_FORM_VALUES } from "./constants";
import { getRoleColumns } from "./columns";
import { useRoleData, useRoleActions } from "./hooks";
import { usePagination } from "@/hooks/usePagination";
import { useTableManager } from "@/hooks/useTableManager";
import { createSorterMeta } from "@/utils/tableHelpers";

const { Option } = Select;

// ==================== 主组件 ====================

const RoleManagement: FC = () => {
  // 使用全局分页 hook
  const { paginationProps, setCurrent, setPageSize, setTotal } = usePagination();

  // 服务端排序：field 对应后端 roleAllowedSortFields 白名单 key
  const sorterMetas = useMemo(
    () => [
      createSorterMeta<Role>("roleName"),
      createSorterMeta<Role>("roleKey"),
      createSorterMeta<Role>("roleSort"),
      createSorterMeta<Role>("status"),
      createSorterMeta<Role>("createdAt", "date"),
    ],
    []
  );

  // 角色列表加载函数
  const fetchRoles = useCallback(async (params: Record<string, unknown>) => {
    const result = await post("/system/roles/list", params) as { data: { list: Role[]; total: number } };
    return { list: result.data?.list || [], total: result.data?.total || 0 };
  }, []);

  // 使用 useTableManager 管理角色列表（分页 + 搜索 + 服务端排序）
  const {
    loading,
    data: roles,
    searchForm,
    editForm,
    loadData: loadRoles,
    applyFilters,
    handleReset: tableHandleReset,
    handleTableChange,
    getColumnSortOrder,
  } = useTableManager<Role>(fetchRoles, {
    externalPagination: {
      current: paginationProps.current ?? 1,
      pageSize: paginationProps.pageSize ?? 10,
      setCurrent,
      setPageSize,
      setTotal,
    },
    sorterMetas,
  });

  // 辅助数据 Hook（菜单树 / 部门树 / 统计 / 权限勾选）
  const {
    menuTree,
    deptTree,
    statistics,
    checkedMenuKeys,
    checkedDeptKeys,
    currentDataScope,
    setCheckedMenuKeys,
    setCheckedDeptKeys,
    setCurrentDataScope,
    loadStatistics,
    loadMenuTree,
    loadDeptTree,
  } = useRoleData();

  // 防重复调用追踪器
  const loadingMenusRef = useRef(new Set<string>());
  const loadingDeptsRef = useRef(new Set<string>());

  // 使用 useCallback 包装的 loadRoleMenus - 稳定引用 + 防重复调用
  const loadRoleMenus = useCallback(async (roleId: string) => {
    // 参数验证
    if (!roleId || roleId === "undefined" || roleId === "null") {
      console.warn("[loadRoleMenus] 无效的 roleId:", roleId);
      return [];
    }

    // 防重复调用检查
    if (loadingMenusRef.current.has(roleId)) {
      console.warn("[loadRoleMenus] 正在加载中，跳过重复请求:", roleId);
      return [];
    }

    loadingMenusRef.current.add(roleId);

    try {
      const result = await post(`/system/menus/role-menu-tree-select/${roleId}`) as { data: { checkedKeys: string[] } };
      const checkedKeys = result.data.checkedKeys || [];
      return checkedKeys;
    } catch (error) {
      console.error("[loadRoleMenus] 加载失败:", error);
      return [];
    } finally {
      loadingMenusRef.current.delete(roleId);
    }
  }, []); // 空依赖数组，保证函数引用稳定

  // 使用 useCallback 包装的 loadRoleDepts - 保持一致性
  const loadRoleDepts = useCallback(async (roleId: string) => {
    if (!roleId || roleId === "undefined" || roleId === "null") {
      console.warn("[loadRoleDepts] 无效的 roleId:", roleId);
      return [];
    }

    if (loadingDeptsRef.current.has(roleId)) {
      console.warn("[loadRoleDepts] 正在加载中，跳过重复请求:", roleId);
      return [];
    }

    loadingDeptsRef.current.add(roleId);
    console.log("[loadRoleDepts] 开始加载角色部门权限:", roleId);

    try {
      const result = await post(`/system/departments/role-dept-tree-select/${roleId}`) as { data: { checkedKeys: string[] } };
      const checkedKeys = result.data.checkedKeys || [];
      console.log("[loadRoleDepts] 加载完成，权限数量:", checkedKeys.length, "roleId:", roleId);
      return checkedKeys;
    } catch (error) {
      console.error("[loadRoleDepts] 加载失败:", error);
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

  // 初始化加载（loadStatistics 由 useRoleData 内部 useEffect 自动触发；分页由 handleTableChange 驱动）
  useEffect(() => {
    loadRoles();
    loadMenuTree();
    loadDeptTree();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 搜索：applyFilters 读 searchForm，自动带上当前排序
  const handleSearch = () => {
    applyFilters();
  };

  // 重置搜索：清表单 + 清排序，回到第 1 页
  const handleReset = () => {
    tableHandleReset();
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
    getColumnSortOrder,
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
              styles={{ content: { color: "var(--theme-success, #3f8600)" } }}
              prefix={<SafetyCertificateOutlined />}
            />
          </Card>
        </Col>
        <Col span={8}>
          <Card>
            <Statistic
              title="停用角色"
              value={statistics.inactive}
              styles={{ content: { color: "var(--theme-error, #cf1322)" } }}
              prefix={<DeleteOutlined />}
            />
          </Card>
        </Col>
      </Row>

      {/* 搜索表单和操作按钮 */}
      <Card style={{ marginBottom: 16 }}>
        <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", flexWrap: "wrap", gap: "16px" }}>
          <Form form={searchForm} layout="inline" style={{ flex: 1, minWidth: 0 }}>
            <Form.Item name="roleName" label="角色名称">
              <Input placeholder="请输入角色名称" />
            </Form.Item>
            <Form.Item name="roleKey" label="权限字符">
              <Input placeholder="请输入权限字符" />
            </Form.Item>
            <Form.Item name="status" label="状态">
              <Select placeholder="请选择状态" style={{ width: 120 }} allowClear onSearch={() => {}}>
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
                icon={<DeleteOutlined />} style={{ color: "var(--theme-error, #ff4d4f)" }}
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
          onChange={handleTableChange}
          rowSelection={{
            selectedRowKeys,
            onChange: setSelectedRowKeys,
          }}
        />
      </Card>

      {/* 编辑弹窗 */}
      <Modal
        title={editingRole ? "编辑角色" : "新增角色"}
        open={editModalVisible}
        onOk={() => handleSave(editForm)}
        afterOpenChange={(open) => handleModalOpenChange(open, editForm)}
        onCancel={() => setEditModalVisible(false)}
        width={800}
        styles={{ body: { maxHeight: "70vh", overflowY: "auto" } }}
      >
        <Form form={editForm} layout="vertical" initialValues={DEFAULT_FORM_VALUES}>
          {/* 基本信息 */}
          <Card title="基本信息" size="small" style={{ marginBottom: 16 }}>
            <Row gutter={16}>
              <Col span={12}>
                <Form.Item
                  name="roleName"
                  label="角色名称"
                  rules={[{ required: true, message: "请输入角色名称" }]}
                >
                  <Input />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="roleKey"
                  label="权限字符"
                  rules={[{ required: true, message: "请输入权限字符" }]}
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
                  rules={[{ required: true, message: "请输入显示顺序" }]}
                >
                  <Input type="number" />
                </Form.Item>
              </Col>
              <Col span={12}>
                <Form.Item
                  name="status"
                  label="状态"
                >
                  <Select onSearch={() => {}}>
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
                    style={{ background: "#f5f5f5", padding: "8px", borderRadius: "4px", maxHeight: "400px", overflowY: "auto" }}
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
                  <Select onChange={(value) => handleDataScopeChange(value, editForm)} onSearch={() => {}}>
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
                      style={{ background: "#f5f5f5", padding: "8px", borderRadius: "4px", maxHeight: "400px", overflowY: "auto" }}
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

