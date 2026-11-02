/**
 * 网络设备页的行操作 + 行操作权限门控 hook（v1.24 HYG-04 / D5）。
 *
 * 从 useDeviceActions.tsx 原样搬迁（零行为变化）：
 *   - resolveDeviceActionPerm（门控键解析）
 *   - editPerm / removePerm 的解析（useActionPermission 三态 + useMenuStore 权限集）
 *   - buildRowActions（采集端口 / 详情 / MAC 历史 / 编辑 / 删除 五个行操作）
 *
 * 依赖纪律（本刀**不改**的行为语义）：
 *   - `buildRowActions` 仍是 useCallback，且**保留原有的 eslint-disable
 *     （依赖数组故意不完整）**：采集/详情/编辑 handler 均为稳定闭包，
 *     依赖数组一旦"补全"会因 handler 每次渲染重建而反复重建整列定义（v1.23 FAIL-1 同源风险）。
 *   - `reloadRef` 仍由**外部（页面）持有并注入**（见 components/DeviceTablePanel.tsx）：
 *     `buildRowActions` 的第二参 `loadData` 即页面侧 `reloadRef.current`，
 *     签名与调用时机逐字不变。
 *
 * 权限门控（v1.24 #16 前端侧）：
 *   行操作「编辑 / 删除」接入 Phase 78 的三态门控（useActionPermission +
 *   ActionButton.permission 契约字段），门控键的解析见 resolveDeviceActionPerm。
 */

import { useCallback } from "react";
import { Button } from "antd";
import { CloudUploadOutlined, EditOutlined, EyeOutlined, HistoryOutlined } from "@ant-design/icons";
import type { NetworkDevice } from "@/types";
import { createDeleteAction } from "@/components/shared/ActionButtons";
import type { ActionButton } from "@/components/shared/ActionButtons";
import { useActionPermission } from "@/components/shared";
import { useMenuStore } from "@/store/menuStore";
import { DEVICE_PERMISSION_CODES } from "../constants";

/**
 * 行操作的公共上下文入参（由 useDeviceActions 透传）。
 *
 * 这四项**只服务于行操作**：页面持有它们，本 hook 只消费不改写。
 */
export interface UseDeviceRowActionsContext {
  /** 采集进行中的设备 id（「采集端口」降级 disabled 的依据） */
  collectingDeviceId: string | null;
  /** 是否展示「查看 MAC 历史」入口（network:mac:list，14-05b 联动） */
  canViewMACHistory: boolean;
  /** 路由跳转 */
  navigate: (path: string) => void;
  /** 打开详情弹窗 */
  openDetailModal: (device: NetworkDevice) => void;
}

/** useDeviceRowActions 的入参（行操作上下文 + 由 useDeviceActions 产出的业务 handler） */
export interface UseDeviceRowActionsParams extends UseDeviceRowActionsContext {
  /** 采集端口 handler（来自 useDeviceActions） */
  handleCollectPorts: (device: NetworkDevice) => Promise<void>;
  /** 删除 handler（来自 useDeviceActions） */
  handleDelete: (id: string, loadData: () => void) => Promise<void>;
  /** 打开编辑弹窗 handler（来自 useDeviceActions） */
  openModal: (record?: NetworkDevice) => void;
}

/**
 * 解析设备模块某个动作实际生效的门控权限码。
 *
 * 规则（按序）：
 *  1. 权限集里存在任一 `network:device:*` **动作级**码 ⇒ 该环境已为设备模块播种
 *     F 型按钮并授予了动作粒度 ⇒ 按**动作级码**逐动作门控（#16 的目标语义：
 *     只读用户看不到「编辑 / 删除」）。
 *  2. 否则（当前 sys_menu 只播种了页面级 `network:device`）⇒ 退化为**页面级码**门控，
 *     语义 =「能进入设备管理页面的用户可见本页操作入口」，与拆分前行为完全一致。
 *
 * ⚠️ 为什么必须有第 2 条兜底：动作级码在 sys_menu 里**没有授权来源**，
 *   若直接按动作级码门控，会对**所有用户**（含超管）隐藏编辑/删除按钮 ——
 *   那是功能回归，不是权限修复。待后端播种 F 型按钮后，第 1 条自然生效。
 */
export function resolveDeviceActionPerm(permissions: string[], actionCode: string): string {
  const hasActionLevelPerm = permissions.some((perm) =>
    perm.startsWith(`${DEVICE_PERMISSION_CODES.page}:`)
  );
  return hasActionLevelPerm ? actionCode : DEVICE_PERMISSION_CODES.page;
}

export interface UseDeviceRowActionsReturn {
  buildRowActions: (record: NetworkDevice, loadData: () => void) => ActionButton[];
}

export function useDeviceRowActions(params: UseDeviceRowActionsParams): UseDeviceRowActionsReturn {
  const {
    collectingDeviceId,
    canViewMACHistory,
    navigate,
    openDetailModal,
    handleCollectPorts,
    handleDelete,
    openModal,
  } = params;

  // v1.24 #16 前端侧：行操作权限门控（三态 ready/loading/error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  // ⚠️ Phase 86 W3 说明：此处 `permissions` 仅用于**门控键降级解析**
  // （见下方 resolveDeviceActionPerm：环境有无动作级码决定用动作码还是页面级码）；
  // 门控判定（三态 visible/disabled/hint）仍由 `useActionPermission` 的 `decide` 承担。
  // 保留这处直读是**降级语义的前提**（删了就无法区分「环境已播种动作码」与「只有页面级码」），
  // 它不构成第二判定入口。
  const menuPermissions = useMenuStore((s) => s.permissions);

  // 门控键：权限集已就绪且环境具备动作粒度 ⇒ 用动作级码；否则用页面级码（行为不变）。
  // 未就绪（loading/error）时权限集不可信 ⇒ 退回页面级码，三态门控会把按钮
  // 渲染成 disabled + 提示（入口不消失），与 Phase 78 语义一致。
  const readyPermissions = permGate.state === "ready" ? menuPermissions : [];
  const editPerm = resolveDeviceActionPerm(readyPermissions, DEVICE_PERMISSION_CODES.edit);
  const removePerm = resolveDeviceActionPerm(readyPermissions, DEVICE_PERMISSION_CODES.remove);
  // Phase 86 W3（扩面）：采集端口是对设备连接的**写**操作（落库采集结果），
  // networkDevice 码组无独立 `collect` 键 → 就近归 `edit`（本 phase 零新增码），
  // 且与 edit 共用同一套降级解析结果（窗口期语义一致）。
  const collectPerm = resolveDeviceActionPerm(readyPermissions, DEVICE_PERMISSION_CODES.edit);

  // 行操作（PageTable ctx 注入 loadData 后组装；操作列由 PageTable 追加）
  const buildRowActions = useCallback(
    (record: NetworkDevice, loadData: () => void): ActionButton[] => [
      {
        key: "collect-ports",
        label: "采集端口",
        icon: <CloudUploadOutlined />,
        onClick: () => handleCollectPorts(record),
        // Phase 86 W3（扩面）：采集端口 = 写入口（落库采集结果）→ 归 networkDevice.edit
        // （无独立 `collect` 码；走同一套降级解析 collectPerm，窗口期语义与编辑一致）。
        permission: collectPerm,
        // 契约 ActionButton 无 loading 位：采集进行中降级为 disabled（记 PR 偏差）
        disabled: collectingDeviceId === record.id,
        // 68.1-T5 minor#1：降级语义保持不变，仅补「进行中」状态语义（UI-SPEC §2 #1 / §7）。
        // 走 render 而非 ActionButton 顶层字段的原因：
        // 1) ActionButton 契约（T1 冻结）无 aria-busy/title 透传位；
        // 2) 本页动作数 ≥3 时 ActionButtons 收进「操作」下拉，普通动作的 aria 属性不会落到 DOM。
        // 渲染参数与 ActionButtons 普通按钮分支保持一致（type=link/size=small/className）。
        render: () => {
          const collecting = collectingDeviceId === record.id;
          return (
            <Button
              type="link"
              size="small"
              icon={<CloudUploadOutlined />}
              onClick={() => handleCollectPorts(record)}
              disabled={collecting}
              className="action-btn-link"
              aria-label="采集端口"
              aria-busy={collecting}
              title={collecting ? "采集中" : "采集端口"}
            >
              采集端口
            </Button>
          );
        },
      },
      {
        key: "detail",
        label: "详情",
        icon: <EyeOutlined />,
        onClick: () => openDetailModal(record),
      },
      // 14-05b:网络设备 → MAC 历史联动入口(D-16 锁定)
      ...(canViewMACHistory
        ? [
            {
              key: "mac-history",
              label: "查看 MAC 历史",
              icon: <HistoryOutlined />,
              onClick: () => navigate(`/network/mac/history?deviceId=${record.id}`),
            },
          ]
        : []),
      {
        key: "edit",
        label: "编辑",
        icon: <EditOutlined />,
        onClick: () => openModal(record),
        // v1.24 #16 前端侧：编辑按设备模块权限码门控（门控键见 resolveDeviceActionPerm）
        permission: editPerm,
      },
      createDeleteAction(record.id, () => handleDelete(record.id, loadData), {
        target: record.deviceName,
        // v1.24 #16 前端侧：删除按设备模块权限码门控
        permission: removePerm,
      }),
    ],
    // eslint-disable-next-line react-hooks/exhaustive-deps -- 采集/详情/编辑 handler 均为稳定闭包
    [collectingDeviceId, canViewMACHistory, navigate, editPerm, removePerm, collectPerm]
  );

  // permGate 的三态由 ActionButtons 侧同源 helper 消费（ActionButton.permission 契约字段）：
  // 就绪且无权 → 不渲染；未就绪 → 渲染但 disabled + 提示。

  return { buildRowActions };
}
