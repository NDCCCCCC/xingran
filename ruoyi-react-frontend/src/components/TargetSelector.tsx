import React, { useState, useEffect } from 'react';
import { Form, Select, Tree, Checkbox, message, Spin } from 'antd';
import { get, post } from '@/lib/api';
import type { TargetType, DeptTreeNode, RoleOption, UserOption } from '@/types/notice';

interface TargetSelectorProps {
  targetType: TargetType;
  onTargetTypeChange: (type: TargetType) => void;
  targetDepts?: string[];
  targetRoles?: string[];
  targetUsers?: string[];
  onTargetDeptsChange: (depts: string[]) => void;
  onTargetRolesChange: (roles: string[]) => void;
  onTargetUsersChange: (users: string[]) => void;
}

/**
 * 目标选择器组件
 * 支持按部门、角色、用户选择通知接收范围
 */
const TargetSelector: React.FC<TargetSelectorProps> = ({
  targetType,
  onTargetTypeChange,
  targetDepts = [],
  targetRoles = [],
  targetUsers = [],
  onTargetDeptsChange,
  onTargetRolesChange,
  onTargetUsersChange,
}) => {
  const [loading, setLoading] = useState(false);
  const [deptTree, setDeptTree] = useState<DeptTreeNode[]>([]);
  const [roles, setRoles] = useState<RoleOption[]>([]);
  const [users, setUsers] = useState<UserOption[]>([]);
  const [checkedDeptKeys, setCheckedDeptKeys] = useState<React.Key[]>(targetDepts);
  const [expandedKeys, setExpandedKeys] = useState<React.Key[]>([]);

  // 加载部门树
  const loadDeptTree = async () => {
    setLoading(true);
    try {
      const response: { list: DeptTreeNode[] } = await get('/system/departments/tree');
      setDeptTree(response.list || []);
    } catch (error) {
      console.error('加载部门树失败:', error);
      message.error('加载部门树失败');
    } finally {
      setLoading(false);
    }
  };

  // 加载角色列表
  const loadRoles = async () => {
    setLoading(true);
    try {
      const response: { list: RoleOption[] } = await post('/system/roles/all', {});
      setRoles(response.list || []);
    } catch (error) {
      console.error('加载角色列表失败:', error);
      message.error('加载角色列表失败');
    } finally {
      setLoading(false);
    }
  };

  // 加载用户列表
  const loadUsers = async (search = '') => {
    setLoading(true);
    try {
      const response: { list: UserOption[] } = await post('/system/users/list', {
        username: search || undefined,
        current: 1,
        pageSize: 50,
      });
      setUsers(response.list || []);
    } catch (error) {
      console.error('加载用户列表失败:', error);
      message.error('加载用户列表失败');
    } finally {
      setLoading(false);
    }
  };

  // 根据 target_type 加载对应数据
  useEffect(() => {
    if (targetType === 1) {
      // 指定部门
      loadDeptTree();
    } else if (targetType === 2) {
      // 指定角色
      loadRoles();
    } else if (targetType === 3) {
      // 指定用户
      loadUsers();
    }
  }, [targetType]);

  // 部门树选中变化
  const handleDeptCheck = (checkedKeysValue: React.Key[] | { checked: React.Key[]; halfChecked: React.Key[] }) => {
    const keys = Array.isArray(checkedKeysValue) ? checkedKeysValue : checkedKeysValue.checked;
    setCheckedDeptKeys(keys);
    onTargetDeptsChange(keys as string[]);
  };

  // 渲染目标选择器内容
  const renderTargetContent = () => {
    switch (targetType) {
      case 0: // 全部用户
        return (
          <div className="text-gray-500 py-4 text-center">
            将向所有用户发送此通知
          </div>
        );

      case 1: // 指定部门
        return (
          <Spin spinning={loading}>
            <div className="max-h-64 overflow-y-auto border rounded p-4">
              <Tree
                checkable
                checkedKeys={checkedDeptKeys}
                onCheck={handleDeptCheck}
                expandedKeys={expandedKeys}
                onExpand={setExpandedKeys}
                treeData={deptTree}
                fieldNames={{ title: 'title', key: 'key', children: 'children' }}
              />
            </div>
            <div className="mt-2 text-sm text-gray-500">
              已选择 {checkedDeptKeys.length} 个部门（包含子部门）
            </div>
          </Spin>
        );

      case 2: // 指定角色
        return (
          <Spin spinning={loading}>
            <Checkbox.Group
              value={targetRoles}
              onChange={(values) => onTargetRolesChange(values as string[])}
              className="w-full"
            >
              <div className="grid grid-cols-2 gap-2 max-h-64 overflow-y-auto">
                {roles.map((role) => (
                  <Checkbox key={role.id} value={role.id}>
                    {role.roleName}
                    <span className="text-xs text-gray-400 ml-1">({role.roleKey})</span>
                  </Checkbox>
                ))}
              </div>
            </Checkbox.Group>
            <div className="mt-2 text-sm text-gray-500">
              已选择 {targetRoles.length} 个角色
            </div>
          </Spin>
        );

      case 3: // 指定用户
        return (
          <Spin spinning={loading}>
            <Select
              mode="multiple"
              value={targetUsers}
              onChange={onTargetUsersChange}
              placeholder="请选择用户"
              showSearch
              filterOption={false}
              onSearch={(value) => loadUsers(value)}
              className="w-full"
              options={users.map((user) => ({
                label: `${user.nickname || user.username} (${user.username})`,
                value: user.id,
              }))}
            />
            <div className="mt-2 text-sm text-gray-500">
              已选择 {targetUsers.length} 个用户
            </div>
          </Spin>
        );

      default:
        return null;
    }
  };

  return (
    <div className="space-y-4">
      <Form.Item label="接收范围" rules={[{ required: true, message: '请选择接收范围' }]}>
        <Select
          value={targetType}
          onChange={onTargetTypeChange}
          options={[
            { label: '全部用户', value: 0 },
            { label: '指定部门', value: 1 },
            { label: '指定角色', value: 2 },
            { label: '指定用户', value: 3 },
          ]}
        />
      </Form.Item>

      {targetType !== 0 && (
        <Form.Item label=" " colon={false}>
          <div className="border rounded p-4 bg-gray-50">{renderTargetContent()}</div>
        </Form.Item>
      )}
    </div>
  );
};

export default TargetSelector;
