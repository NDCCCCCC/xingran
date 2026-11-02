/**
 * 网络设备页的数据操作 hook（v1.24 HYG-04 / D3、D5）。
 *
 * 从 index.tsx 原样搬迁（零行为变化）：handleProbe / handleQuickCreate / handleCreate /
 * handleDelete / handleBatchExport / handleCollectPorts / openModal / openQuickCreateModal
 * —— 函数体、错误文案、成功提示、调用顺序逐字保留。
 *
 * 依赖纪律（本刀**不改**的行为语义）：`reloadRef` 仍由**外部（页面）持有并注入**，本 hook
 * 不接管它 —— handlers 依旧以 `loadData` **参数**承接刷新回调（见各函数签名）。
 *
 * 行操作（buildRowActions）与行操作权限门控已抽到 hooks/useDeviceRowActions.tsx（D5）：
 * 本 hook 只组装业务 handler 并转交 buildRowActions。
 */

import type { FormInstance } from "antd";
import type { AuthCredential, NetworkDevice } from "@/types";
import type { MessageInstance } from "antd/es/message/interface";
import { post } from "@/lib/api";
import { batchExport } from "@/lib/api/networkApi";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import type { ActionButton } from "@/components/shared/ActionButtons";
import type { ProbeResult } from "./useDeviceModals";
import { useDeviceRowActions, type UseDeviceRowActionsContext } from "./useDeviceRowActions";

/** useDeviceActions 的入参（全部由页面侧注入，hook 内部不持有业务 state） */
export interface UseDeviceActionsParams extends UseDeviceRowActionsContext {
  // —— 表单与当前编辑对象 ——
  /** 快速创建表单实例 */
  quickCreateForm: FormInstance;
  /** 手动新增/编辑表单实例 */
  editForm: FormInstance;
  /** 当前编辑的设备（null = 新增）——handleCreate 判定走更新还是创建 */
  editingDevice: NetworkDevice | null;
  /** antd message 实例 */
  message: MessageInstance;
  // —— 数据加载 ——
  /** 授权凭证列表加载 */
  loadCredentials: () => Promise<void>;
  /** 统计数据加载（任何列表刷新都联动统计） */
  loadStatistics: () => Promise<void>;
  /** 凭证兜底注入（编辑回填 Select 显示 raw UUID 的修复） */
  ensureCredential: (cred: Pick<AuthCredential, "id"> & Partial<AuthCredential>) => void;
  // —— 弹窗 state（来自 useDeviceModals）——
  /** 快速创建弹窗可见性 */
  setQuickCreateModalVisible: (visible: boolean) => void;
  /** 探测结果 */
  setProbeResult: (result: ProbeResult | null) => void;
  /** 探测进行中 */
  setProbing: (probing: boolean) => void;
  /** 关闭快速创建弹窗 */
  closeQuickCreateModal: () => void;
  // —— 页面本地 state setter ——
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

export function useDeviceActions(params: UseDeviceActionsParams): UseDeviceActionsReturn {
  const {
    quickCreateForm,
    editForm,
    editingDevice,
    message,
    loadCredentials,
    loadStatistics,
    ensureCredential,
    setQuickCreateModalVisible,
    setProbeResult,
    setProbing,
    closeQuickCreateModal,
    setCollectingDeviceId,
    setModalVisible,
    setEditingDevice,
    setBatchModalVisible,
    setBatchExporting,
  } = params;

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

  // v1.24 HYG-04 D5：行操作 + 权限门控已抽到 useDeviceRowActions.tsx（上下文经 params 透传）
  const { buildRowActions } = useDeviceRowActions({
    ...params,
    handleCollectPorts,
    handleDelete,
    openModal,
  });

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
