/**
 * AD 域控配置页 —「漂移告警」Tab（migration_242：同步漂移告警并入 AD 域控配置）
 *
 * 背景：漂移告警是 AD 域控同步的附属监控（后端 internal/services/addomain/
 * ad_sync_drift_alert_service.go 监控 sys_ad_group_member 计数漂移，
 * scheduler/ad_sync_tasks.go 每小时检测）。原为独立菜单（migration_238 播 C 型
 * 「同步漂移告警」），现并入本页第三个 Tab，不再单独占菜单位。
 *
 * 组件复用：StatusCards / ConfigForm / TriggerLogTable 仍留在
 * `src/pages/system/ad-drift-alert/components/`（旧页面保留为薄壳，直达链接可用），
 * 本 Tab 直接 import 复用，避免两份实现漂移。
 *
 * WS 订阅：**不在本组件内订阅**。`onADDriftAlert` 由宿主页面（ad-domain/configs/
 * index.tsx）页面级挂载，收到 ad_sync_drift_alert 后通过 `refreshToken` prop
 * 把刷新信号下发到这里 —— 这样即使当前停在「配置列表」Tab，全局提示也不会丢。
 *
 * 依赖稳定性（CLAUDE.md 防无限循环）：所有 effect 依赖均为 primitive /
 * useCallback 稳定引用，禁止对象字面量直接进依赖。
 */
import { useCallback, useEffect, useRef, useState } from "react";
import type { FC } from "react";
import { Button, Card, Space } from "antd";
import { ThunderboltOutlined } from "@ant-design/icons";
import StatusCards from "@/pages/system/ad-drift-alert/components/StatusCards";
import TriggerLogTable from "@/pages/system/ad-drift-alert/components/TriggerLogTable";
import ConfigForm from "@/pages/system/ad-drift-alert/components/ConfigForm";
import { useActionPermission } from "@/components/shared/useActionPermission";
import type { ActionPermissionDecision } from "@/components/shared/useActionPermission";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import { getAppMessage } from "@/utils/antdMessage";
import {
  getStatus,
  getConfig,
  triggerOnce,
  type ADDriftAlertStatus,
  type ADDriftAlertConfig,
} from "@/lib/adDriftAlertApi";

export interface DriftAlertTabProps {
  /**
   * 页面级 WS 订阅派发的刷新信号（primitive，每次 +1）。
   * 变化时刷新状态卡与触发记录；首帧不触发（避免与 mount 首屏加载重复请求）。
   */
  refreshToken?: number;
  /**
   * 「立即检测」门控决策（宿主页面注入；缺省时本组件自判）。
   * 宿主注入用于让页面级文件保有三态门控特征（check-gating-coverage 口径）。
   */
  triggerDecision?: ActionPermissionDecision;
}

const DriftAlertTab: FC<DriftAlertTabProps> = ({ refreshToken = 0, triggerDecision }) => {
  const [status, setStatus] = useState<ADDriftAlertStatus | null>(null);
  const [config, setConfig] = useState<ADDriftAlertConfig | null>(null);
  const [statusLoading, setStatusLoading] = useState(true);
  const [configLoading, setConfigLoading] = useState(true);
  const [triggering, setTriggering] = useState(false);
  /** 触发记录表刷新信号（primitive）：立即检测成功 / WS 新告警时 +1 */
  const [logRefreshToken, setLogRefreshToken] = useState(0);

  const gate = useActionPermission();
  const ownDecision = gate.decide(PERMISSION_CODES.adDriftAlert.trigger);
  const decision = triggerDecision ?? ownDecision;

  /** 刷新状态卡（稳定引用，供 mount / 立即检测 / 页面 WS 信号复用） */
  const refreshStatus = useCallback(async () => {
    try {
      const next = await getStatus();
      setStatus(next);
    } catch (error) {
      handleApiError(error, "加载告警状态");
    } finally {
      setStatusLoading(false);
    }
  }, []);

  /** 刷新配置（稳定引用） */
  const refreshConfig = useCallback(async () => {
    try {
      const next = await getConfig();
      setConfig(next);
    } catch (error) {
      handleApiError(error, "加载告警配置");
    } finally {
      setConfigLoading(false);
    }
  }, []);

  /** 刷新日志表（primitive setState，稳定引用） */
  const bumpLogs = useCallback(() => {
    setLogRefreshToken((n) => n + 1);
  }, []);

  // 首屏加载：空依赖，mount 一次。不在依赖里放对象字面量。
  useEffect(() => {
    void refreshStatus();
    void refreshConfig();
  }, [refreshStatus, refreshConfig]);

  // 页面级 WS 信号 → 刷新状态卡 + 触发记录（首帧跳过，避免与首屏加载重复请求）
  const lastRefreshTokenRef = useRef(refreshToken);
  useEffect(() => {
    if (refreshToken === lastRefreshTokenRef.current) return;
    lastRefreshTokenRef.current = refreshToken;
    void refreshStatus();
    bumpLogs();
  }, [refreshToken, refreshStatus, bumpLogs]);

  /** 「立即检测」：写操作，走 :trigger 门控；成功后刷新状态 + 日志 */
  const handleTrigger = useCallback(async () => {
    setTriggering(true);
    try {
      const result = await triggerOnce();
      handleSuccess("立即检测");
      // trigger 端点实际只返回 {message}，不按虚构的 drifted 分支判断
      getAppMessage().info(result?.message ?? "检测完成");
      void refreshStatus();
      bumpLogs();
    } catch (error) {
      // 唯一错误出口
      handleApiError(error, "立即检测");
    } finally {
      setTriggering(false);
    }
  }, [refreshStatus, bumpLogs]);

  /** ConfigForm 保存成功回调：更新本地 config + 刷新状态卡 */
  const handleConfigSaved = useCallback(
    (next: ADDriftAlertConfig) => {
      setConfig(next);
      void refreshStatus();
    },
    [refreshStatus]
  );

  const triggerDisabled = decision.disabled || triggering;

  return (
    <Space orientation="vertical" size="medium" style={{ width: "100%" }}>
      <div style={{ display: "flex", alignItems: "center", justifyContent: "flex-end" }}>
        {decision.visible && (
          <Button
            type="primary"
            icon={<ThunderboltOutlined />}
            onClick={handleTrigger}
            loading={triggering}
            disabled={triggerDisabled}
            aria-label="立即检测"
            title={decision.hint ?? "立即检测"}
          >
            立即检测
          </Button>
        )}
      </div>

      <StatusCards status={status} loading={statusLoading} />

      <ConfigForm config={config} loading={configLoading} onSaved={handleConfigSaved} />

      <Card title="触发记录" size="small">
        <TriggerLogTable refreshToken={logRefreshToken} onMarkRead={bumpLogs} />
      </Card>
    </Space>
  );
};

export default DriftAlertTab;
