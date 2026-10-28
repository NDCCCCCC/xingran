/**
 * 网络设备页的数据操作 + 行操作 hook（v1.24 HYG-04 / D3）。
 *
 * 从 index.tsx 原样搬迁（零行为变化）：
 *   - handleProbe / handleQuickCreate / handleCreate / handleDelete / handleBatchExport /
 *     handleCollectPorts / openModal / openQuickCreateModal / buildRowActions
 *     —— 函数体、错误文案、成功提示、调用顺序逐字保留。
 *
 * 依赖纪律（本刀**不改**的行为语义）：
 *   - `buildRowActions` 仍是 useCallback，且**保留原有的 eslint-disable
 *     （依赖数组故意不完整）**：采集/详情/编辑 handler 均为稳定闭包，
 *     依赖数组一旦"补全"会因 handler 每次渲染重建而反复重建整列定义（v1.23 FAIL-1 同源风险）。
 *   - `reloadRef` 仍由**外部（页面）持有并注入**，本 hook 不接管它：
 *     handlers 依旧以 `loadData` **参数**承接刷新回调（见各函数签名）。
 *
 * 权限门控（v1.24 #16 前端侧，本刀与拆分同时落地）：
 *   行操作「编辑 / 删除」接入 Phase 78 的三态门控（useActionPermission +
 *   ActionButton.permission 契约字段），门控键的解析见 resolveDeviceActionPerm。
 */

import { useCallback } from "react";
import type { FormInstance } from "antd";
import { Button } from "antd";
import {
  CloudUploadOutlined,
  EditOutlined,
  EyeOutlined,
  HistoryOutlined,
} from "@ant-design/icons";
import type { AuthCredential, NetworkDevice } from "@/types";
import type { MessageInstance } from "antd/es/message/interface";
import { post } from "@/lib/api";
import { batchExport } from "@/lib/api/networkApi";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import { createDeleteAction } from "@/components/shared/ActionButtons";
import type { ActionButton } from "@/components/shared/ActionButtons";
import { useActionPermission } from "@/components/shared";
import { useMenuStore } from "@/store/menuStore";
import { DEVICE_PERMISSION_CODES } from "../constants";
import type { ProbeResult } from "./useDeviceModals";

/** antd App.useApp() 的 message 实例类型（不深引 antd 内部路径） */
export interface UseDeviceActionsParams {
  /** 快速创建表单实例 */
  quickCreateForm: FormInstance;
  /** 手动新增/编辑表单实例 */
  editForm: FormInstance;
  /** 当前编辑的设备（null = 新增）——handleCreate 判定走更新还是创建 */
  editingDevice: NetworkDevice | null;
  /** 采集进行中的设备 id（行操作「采集端口」降级 disabled 的依据） */
  collectingDeviceId: string | null;
  /** 是否展示「查看 MAC 历史」入口（network:mac:list，14-05b 联动） */
  canViewMACHistory: boolean;
  /** 路由跳转 */
  navigate: (path: string) => void;
  /** antd message 实例 */
  message: MessageInstance;
  /** 授权凭证列表加载 */
  loadCredentials: () => Promise<void>;
  /** 统计数据加载（任何列表刷新都联动统计） */
  loadStatistics: () => Promise<void>;
  /** 凭证兜底注入（编辑回填 Select 显示 raw UUID 的修复） */
  ensureCredential: (cred: Pick<AuthCredential, "id"> & Partial<AuthCredential>) => void;
  /** 快速创建弹窗可见性 */
  setQuickCreateModalVisible: (visible: boolean) => void;
  /** 探测结果 */
  setProbeResult: (result: ProbeResult | null) => void;
  /** 探测进行中 */
  setProbing: (probing: boolean) => void;
  /** 关闭快速创建弹窗 */
  closeQuickCreateModal: () => void;
  /** 打开详情弹窗 */
  openDetailModal: (device: NetworkDevice) => void;
  /** 采集中的设备 id */
  setCollectingDeviceId: (deviceId: string | null) => void;
  /** 编辑弹窗可见性 */
  setModalVisible: (visible: boolean) => void;
  /** 当前编辑的设备（null = 新增） */
  setEditingDevice: (device: NetworkDevice | null) => void;
  /** 批量导出弹窗可见性 */
  setBatchModalVisible: (visible: boolean) => void;
  /** 批量导出进行中 */
  setBatchExporting: (exporting: boolean) => void;
}

export interface UseDeviceActionsReturn {
  handleProbe: () => Promise<void>;
  handleQuickCreate: (loadData: () => void) => Promise<void>;
  handleCreate: (loadData: () => void) => Promise<void>;
  handleDelete: (id: string, loadData: () => void) => Promise<void>;
  handleBatchExport: (entityTypes: string[]) => Promise<void>;
  handleCollectPorts: (device: NetworkDevice) => Promise<void>;
  openModal: (record?: NetworkDevice) => void;
  openQuickCreateModal: () => void;
  buildRowActions: (record: NetworkDevice, loadData: () => void) => ActionButton[];
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
export function resolveDeviceActionPerm(
  permissions: string[],
  actionCode: string
): string {
  const hasActionLevelPerm = permissions.some((perm) =>
    perm.startsWith(`${DEVICE_PERMISSION_CODES.page}:`)
  );
  return hasActionLevelPerm ? actionCode : DEVICE_PERMISSION_CODES.page;
}

export function useDeviceActions(params: UseDeviceActionsParams): UseDeviceActionsReturn {
  const {
    quickCreateForm,
    editForm,
    editingDevice,
    collectingDeviceId,
    canViewMACHistory,
    navigate,
    message,
    loadCredentials,
    loadStatistics,
    ensureCredential,
    setQuickCreateModalVisible,
    setProbeResult,
    setProbing,
    closeQuickCreateModal,
    openDetailModal,
    setCollectingDeviceId,
    setModalVisible,
    setEditingDevice,
    setBatchModalVisible,
    setBatchExporting,
  } = params;

  // v1.24 #16 前端侧：行操作权限门控（三态 ready/loading/error）。
  // ⚠️ 判定必须走 useActionPermission —— 禁止用 permissions.length === 0 判「未就绪」。
  const permGate = useActionPermission();
  const menuPermissions = useMenuStore((s) => s.permissions);

  // 门控键：权限集已就绪且环境具备动作粒度 ⇒ 用动作级码；否则用页面级码（行为不变）。
  // 未就绪（loading/error）时权限集不可信 ⇒ 退回页面级码，三态门控会把按钮
  // 渲染成 disabled + 提示（入口不消失），与 Phase 78 语义一致。
  const readyPermissions = permGate.state === "ready" ? menuPermissions : [];
  const editPerm = resolveDeviceActionPerm(readyPermissions, DEVICE_PERMISSION_CODES.edit);
  const removePerm = resolveDeviceActionPerm(
    readyPermissions,
    DEVICE_PERMISSION_CODES.remove
  );

  // ==================== 数据操作 ====================

  // 探测设备
  const handleProbe = async () => {
    try {
      const values = await quickCreateForm.validateFields(["ipAddress", "credentialId"]);
      setProbing(true);
      setProbeResult(null);

      const result = await post<ProbeResult>("/network/devices/discover", {
        ipAddress: values.ipAddress,
        credentialId: values.credentialId,
      });

      if (result.data?.success) {
        setProbeResult(result.data);
        handleSuccess("设备探测");
      } else {
        handleApiError(result.data?.message || "探测失败", "探测");
      }
    } catch (error) {
      if (error && typeof error === "object" && "errorFields" in error) {
        handleApiError(error, "请先填写 IP 地址和授权凭证", false);
      } else {
        handleApiError(error, "探测");
      }
    } finally {
      setProbing(false);
    }
  };

  // 快速创建设备（loadData 由外部注入：提交成功后刷新列表）
  const handleQuickCreate = async (loadData: () => void) => {
    try {
      const values = await quickCreateForm.validateFields();

      await post("/network/devices/quick-create", {
        ipAddress: values.ipAddress,
        credentialId: values.credentialId,
        snmpPort: values.snmpPort || 161,
        communities: values.communities || [],
        deptId: values.deptId || undefined,
        location: values.location || "",
        description: values.description || "",
      });

      handleSuccess("设备创建");
      closeQuickCreateModal();
      quickCreateForm.resetFields();
      setProbeResult(null);

      loadData();
      void loadStatistics();
    } catch (error) {
      if (error && typeof error === "object" && "errorFields" in error) {
        return;
      }
      handleApiError(error, "创建");
    }
  };

  // 创建/更新设备
  const handleCreate = async (loadData: () => void) => {
    try {
      const values = await editForm.validateFields();
      if (editingDevice) {
        await post(`/network/devices/${editingDevice.id}/update`, values);
        handleSuccess("更新");
      } else {
        await post("/network/devices", values);
        handleSuccess("创建");
      }
      setModalVisible(false);
      editForm.resetFields();
      loadData();
      void loadStatistics();
    } catch (error) {
      if (error && typeof error === "object" && "errorFields" in error) {
        return;
      }
      handleApiError(error, "操作");
    }
  };

  // 删除设备（createDeleteAction → ConfirmDelete 契约：失败 helper 承接并保持弹窗）
  const handleDelete = async (id: string, loadData: () => void) => {
    await post(`/network/devices/${id}/delete`, {});
    handleSuccess("删除");
    loadData();
    void loadStatistics();
  };

  const handleBatchExport = async (entityTypes: string[]) => {
    setBatchExporting(true);
    try {
      const filename = await batchExport(entityTypes, {}); // 可以根据需要添加筛选条件
      message.success(`批量导出成功，文件: ${filename}`);
      setBatchModalVisible(false);
    } catch (error: unknown) {
      message.error(`批量导出失败：${(error as Error).message}`);
    } finally {
      setBatchExporting(false);
    }
  };

  // 采集设备端口
  const handleCollectPorts = async (device: NetworkDevice) => {
    try {
      setCollectingDeviceId(device.id);
      const result = await post<{
        errorMessage?: string;
        successCount?: number;
        failedCount?: number;
      }>("/network/ports/collect", { deviceId: device.id });

      const data = result.data;
      if (!data) {
        handleApiError("采集失败: 未获取到端口数据", "采集");
      } else if (data.errorMessage) {
        handleApiError(data.errorMessage, "采集", false);
      } else if (data.successCount && data.successCount > 0) {
        handleSuccess(
          `采集成功: 成功 ${data.successCount} 个端口${data.failedCount && data.failedCount > 0 ? `，失败 ${data.failedCount} 个` : ""}`
        );
      } else {
        handleApiError("采集失败: 未获取到端口数据", "采集");
      }
    } catch (error) {
      handleApiError(error, "采集");
    } finally {
      setCollectingDeviceId(null);
    }
  };

  // 打开编辑模态框
  const openModal = (record?: NetworkDevice) => {
    if (record) {
      setEditingDevice(record);
      // 凭证兜底注入(2026-06-30,同 info-points):loadCredentials 异步且 pageSize:50
      // 可能不覆盖当前凭证 → 编辑回填 Select 显示 raw UUID。用 record.credentialName 注入。
      if (record.credentialId) {
        ensureCredential({ id: record.credentialId, credentialName: record.credentialName });
      }
      editForm.setFieldsValue({
        ...record,
        deptId: record.deptId || undefined,
        credentialId: record.credentialId || undefined,
      });
    } else {
      setEditingDevice(null);
      editForm.setFieldsValue({ port: 22, snmpPort: 161, status: 0 });
    }
    setModalVisible(true);
    void loadCredentials();
  };

  // 打开快速创建模态框
  const openQuickCreateModal = () => {
    quickCreateForm.resetFields();
    setProbeResult(null);
    setQuickCreateModalVisible(true);
    void loadCredentials();
  };

  // 行操作（PageTable ctx 注入 loadData 后组装；操作列由 PageTable 追加）
  const buildRowActions = useCallback(
    (record: NetworkDevice, loadData: () => void): ActionButton[] => [
      {
        key: "collect-ports",
        label: "采集端口",
        icon: <CloudUploadOutlined />,
        onClick: () => handleCollectPorts(record),
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
    [collectingDeviceId, canViewMACHistory, navigate, editPerm, removePerm]
  );

  // permGate 的三态由 ActionButtons 侧同源 helper 消费（ActionButton.permission 契约字段）：
  // 就绪且无权 → 不渲染；未就绪 → 渲染但 disabled + 提示。

  return {
    handleProbe,
    handleQuickCreate,
    handleCreate,
    handleDelete,
    handleBatchExport,
    handleCollectPorts,
    openModal,
    openQuickCreateModal,
    buildRowActions,
  };
}
