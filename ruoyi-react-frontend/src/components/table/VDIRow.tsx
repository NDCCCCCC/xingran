/**
 * VDIRow - VDI 虚拟机列表行操作列
 *
 * 提取自 `src/pages/vdi/VirtualMachineList/index.tsx` 操作列的 render 函数，
 * 用 React.memo 包装以避免父组件重渲染时无谓地重建按钮子树。
 *
 * 父组件应当：
 * - 通过 useCallback 稳定 onOperate / onDelete / onSync / onBind 引用
 * - 通过 useMemo 稳定 props 对象（强烈推荐，否则 memo 不生效）
 */

import { memo } from "react";
import { Button, Popconfirm, Space, Tooltip } from "antd";
import type { VirtualMachine } from "@/types/vdi";
import type { VMOprationButton } from "@/pages/vdi/VirtualMachineList/vmOperationButtons";

export interface VDIRowProps {
  vm: VirtualMachine;
  /** 当前用户权限列表（用于过滤可见按钮） */
  permissions: string[];
  /** 全部按钮定义 */
  buttons: VMOprationButton[];
  /** 电源操作（start/stop/restart） */
  onOperate: (action: string, vmIds: string[]) => void;
  /** 删除 VM */
  onDelete: (vmId: string) => void;
  /** 同步 VM */
  onSync: (vmId: string) => void;
  /** 绑定用户 */
  onBind: (vm: VirtualMachine) => void;
}

function VDIRowImpl({
  vm,
  permissions,
  buttons,
  onOperate,
  onDelete,
  onSync,
  onBind,
}: VDIRowProps) {
  const allowedButtons = buttons.filter(btn => permissions.includes(btn.permission));

  return (
    <Space size="small">
      {allowedButtons.map(btn => {
        if (btn.action === "delete") {
          return (
            <Popconfirm
              key={btn.action}
              title="确定要删除这个虚拟机吗？此操作将调用 VDI API 删除虚拟机。"
              onConfirm={() => onDelete(vm.id)}
              okText="确定"
              cancelText="取消"
            >
              <Tooltip title={btn.label}>
                <Button type="text" danger icon={btn.icon} />
              </Tooltip>
            </Popconfirm>
          );
        }

        if (btn.action === "bind") {
          return (
            <Tooltip key={btn.action} title={btn.label}>
              <Button
                type="text"
                icon={btn.icon}
                onClick={() => onBind(vm)}
              />
            </Tooltip>
          );
        }

        if (btn.action === "sync") {
          return (
            <Tooltip key={btn.action} title={btn.label}>
              <Button
                type="text"
                icon={btn.icon}
                onClick={() => onSync(vm.id)}
              />
            </Tooltip>
          );
        }

        // 电源操作按钮（start, stop, restart）
        // power_state 业务取值：pending | stopped | suspended | in_use
        let disabled = false;
        if (btn.action === "start") {
          // 仅 stopped 时可开机
          disabled = vm.power_state !== "stopped";
        } else if (btn.action === "stop") {
          // 已停 / 已挂起 / 等待 Agent 时无法关机
          disabled =
            vm.power_state === "stopped" ||
            vm.power_state === "suspended" ||
            vm.power_state === "pending";
        } else if (btn.action === "restart") {
          // 仅 in_use 或 suspended 可重启（注意：业务无 'running' 状态值）
          disabled = vm.power_state !== "in_use" && vm.power_state !== "suspended";
        }

        return (
          <Tooltip key={btn.action} title={btn.label}>
            <Button
              type="text"
              icon={btn.icon}
              disabled={disabled}
              onClick={() => onOperate(btn.action, [vm.id])}
            />
          </Tooltip>
        );
      })}
    </Space>
  );
}

export const VDIRow = memo(VDIRowImpl);
VDIRow.displayName = "VDIRow";

export default VDIRow;
