import { useState, useEffect, useMemo } from "react";
import type { FC, Key } from "react";
import { App, Form, Select, Tree, Checkbox, Spin } from "antd";
import { post } from "@/lib/api";
import type { TargetType, RoleOption, UserOption } from "@/types/notice";
import { useDeptTree, type DeptTreeNode } from "@/hooks/useDeptTree";
import { toShortNameDataNode } from "@/utils/deptUtils";

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
const TargetSelector: FC<TargetSelectorProps> = ({
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
  const { message } = App.useApp();
  // 部门树:消费 canonical useDeptTree (Phase 37 收敛),不再本地 GET fetch。
  // 共享 ['dept','tree'] 缓存 (5min stale / 30min gc / refetchOnWindowFocus:false)。
  const { data: rawDept = [], isLoading: loadingDeptTree } = useDeptTree();
  // 派生 antd <Tree> 期望的 DataNode 形状 (短名 title + key + children)。
  // 旧实现的 GET fetch 直接把后端返回的 SimpleDept[] (字段 id/deptName/children) 喂给
  // <Tree fieldNames={{ title:"title", key:"key", children:"children" }}>,
  // 但后端节点没有 title/key 字段——本地转一道与 useTargetSelector (37-02) 一致,行为等价。
  const deptTree = useMemo(
    () => toShortNameDataNode(rawDept as DeptTreeNode[]),
    [rawDept]
  );
  const [roles, setRoles] = useState<RoleOption[]>([]);
  const [users, setUsers] = useState<UserOption[]>([]);
  const [checkedDeptKeys, setCheckedDeptKeys] = useState<Key[]>(targetDepts);
  const [expandedKeys, setExpandedKeys] = useState<Key[]>([]);

  // 加载角色列表
  const loadRoles = async () => {
    setLoading(true);
    try {
      const response = await post<RoleOption[]>("/system/roles/all", {});
      setRoles(response.data || []);
    } catch (error) {
      console.error("加载角色列表失败:", error);
      message.error("加载角色列表失败");
    } finally {
      setLoading(false);
    }
  };

  // 加载用户列表
  const loadUsers = async (search = "") => {
    setLoading(true);
    try {
      const response = await post<{ list: UserOption[] }>("/system/users/list", {
        username: search || undefined,
        current: 1,
        pageSize: 50,
      });
      setUsers(response.data?.list || []);
    } catch (error) {
      console.error("加载用户列表失败:", error);
      message.error("加载用户列表失败");
    } finally {
      setLoading(false);
    }
  };

  // 根据 target_type 加载对应数据
  // 注意:部门树(targetType===1)数据由 useDeptTree 顶层 hook 自动提供,无需在此触发。
  useEffect(() => {
    if (targetType === 2) {
      // 指定角色
      loadRoles();
    } else if (targetType === 3) {
      // 指定用户
      loadUsers();
    }
  }, [targetType]);

  // 部门树选中变化
  const handleDeptCheck = (checkedKeysValue: Key[] | { checked: Key[]; halfChecked: Key[] }) => {
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
          <Spin spinning={loadingDeptTree}>
            <div className="max-h-64 overflow-y-auto border rounded p-4">
              <Tree
                checkable
                checkedKeys={checkedDeptKeys}
                onCheck={handleDeptCheck}
                expandedKeys={expandedKeys}
                onExpand={setExpandedKeys}
                treeData={deptTree}
                fieldNames={{ title: "title", key: "key", children: "children" }}
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
      <Form.Item label="接收范围" rules={[{ required: true, message: "请选择接收范围" }]}>
        <Select
          value={targetType}
          onChange={onTargetTypeChange}
          options={[
            { label: "全部用户", value: 0 },
            { label: "指定部门", value: 1 },
            { label: "指定角色", value: 2 },
            { label: "指定用户", value: 3 },
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
