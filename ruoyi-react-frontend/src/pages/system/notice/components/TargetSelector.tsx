import { Spin, Tree, Checkbox, Select } from "antd";
import type { DataNode } from "antd/es/tree";
import type { Target } from "../hooks/useTargetSelector";

interface TargetSelectorProps {
  targetType: number;
  targetDepts: React.Key[];
  targetRoles: React.Key[];
  targetUsers: React.Key[];
  deptTree: Target[];
  roles: Target[];
  users: Target[];
  loadingDepts: boolean;
  loadingRoles: boolean;
  loadingUsers: boolean;
  onDeptChange: (keys: React.Key[]) => void;
  onRoleChange: (values: string[]) => void;
  onUserChange: (values: string[]) => void;
}

/**
 * 目标选择器组件
 * 根据目标类型显示不同的选择器（部门树/角色复选框/用户下拉框）
 */
export const TargetSelector: React.FC<TargetSelectorProps> = ({
  targetType,
  targetDepts,
  targetRoles,
  targetUsers,
  deptTree,
  roles,
  users,
  loadingDepts,
  loadingRoles,
  loadingUsers,
  onDeptChange,
  onRoleChange,
  onUserChange,
}) => {
  // 全部用户
  if (targetType === 0) {
    return <div className="text-gray-500 py-4 text-center">将向所有用户发送此通知</div>;
  }

  // 指定部门
  if (targetType === 1) {
    return (
      <Spin spinning={loadingDepts}>
        <div className="max-h-64 overflow-y-auto border rounded p-4">
          <Tree
            checkable
            checkedKeys={targetDepts}
            onCheck={(checked) => {
              const keys = Array.isArray(checked) ? checked : checked.checked;
              onDeptChange(keys);
            }}
            treeData={deptTree as DataNode[]}
            fieldNames={{ title: "title", key: "key", children: "children" }}
          />
        </div>
        <div className="mt-2 text-sm text-gray-500">已选择 {targetDepts.length} 个部门（包含子部门）</div>
      </Spin>
    );
  }

  // 指定角色
  if (targetType === 2) {
    return (
      <Spin spinning={loadingRoles}>
        <Checkbox.Group
          value={targetRoles as string[]}
          onChange={(values) => onRoleChange(values as string[])}
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
        <div className="mt-2 text-sm text-gray-500">已选择 {targetRoles.length} 个角色</div>
      </Spin>
    );
  }

  // 指定用户
  if (targetType === 3) {
    return (
      <Spin spinning={loadingUsers}>
        <Select
          mode="multiple"
          value={targetUsers as string[]}
          onChange={(values) =>    onUserChange(values as string[])}
          placeholder="请选择用户"
          showSearch
          filterOption={(input, option) => {
            if (!option) return false;
            const user = users.find((u) => u.id === option.value);
            return !!(
              user?.username?.toLowerCase().includes(input.toLowerCase()) ||
              user?.nickname?.toLowerCase().includes(input.toLowerCase())
            );
          }}
          className="w-full"
          options={users.filter((user) => user.id != null).map((user) => ({
            label: `${user.nickname || user.username} (${user.username})`,
            value: user.id,
          }))}
         onSearch={() => {}}/>
        <div className="mt-2 text-sm text-gray-500">已选择 {targetUsers.length} 个用户</div>
      </Spin>
    );
  }

  return null;
};
