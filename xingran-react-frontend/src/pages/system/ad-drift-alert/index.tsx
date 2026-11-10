/**
 * 同步漂移告警页（Phase 144 ALRT-03）
 *
 * 菜单驱动动态路由：本文件由 `componentLoader.tsx` 的
 * import.meta.glob("/src/pages/**{index,detail}.tsx") 自动发现，
 * 菜单由后端 migration 238 播种（component=`system/ad-drift-alert/index`）；
 * **不改 App.tsx 静态路由表**。
 *
 * 组合：StatusCards（状态卡）+ TriggerLogTable（触发记录分页/筛选/三态）
 *      + ConfigForm（9 旋钮 + webhook 下拉）+ 「立即检测」按钮
 *      + onADDriftAlert 订阅（复用全局唯一 /system/ws/notices 连接，不新建 WS）。
 *
 * WS 提示口径（PLAN-02 Task 1 要求「选一种并写清」）：
 *   **提示放在页面订阅回调里**，用 `@/utils/antdMessage` 的 getAppMessage()
 *   （context-aware，避免 antd 静态 message 无法消费 ConfigProvider 主题的告警）；
 *   noticeStore 只做纯分流与 listener 派发，不持有 message 实例。
 *
 * 依赖稳定性（CLAUDE.md 防无限循环）：所有 useEffect 依赖均为 primitive /
 * useCallback / useMemo 稳定引用，禁止对象字面量直接进依赖；WS 订阅 cleanup 退订。
 */
import { useCallback, useEffect, useState } from "react";
import type { FC } from "react";
import { Button, Card, Space, Typography } from "antd";
import { ThunderboltOutlined } from "@ant-design/icons";
import StatusCards from "./components/StatusCards";
import TriggerLogTable from "./components/TriggerLogTable";
import ConfigForm from "./components/ConfigForm";
import { useActionPermission } from "@/components/shared/useActionPermission";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { PAGE_TITLES } from "@/constants/pageTitles";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";
import { getAppMessage } from "@/utils/antdMessage";
import { onADDriftAlert } from "@/store/noticeStore";
import {
  getStatus,
  getConfig,
  triggerOnce,
  type ADDriftAlertStatus,
  type ADDriftAlertConfig,
} from "@/lib/adDriftAlertApi";

const { Title } = Typography;

const ADDriftAlertPage: FC = () => {
  const [status, setStatus] = useState<ADDriftAlertStatus | null>(null);
  const [config, setConfig] = useState<ADDriftAlertConfig | null>(null);
  const [statusLoading, setStatusLoading] = useState(true);
  const [configLoading, setConfigLoading] = useState(true);
  const [triggering, setTriggering] = useState(false);
  /** 触发记录表刷新信号（primitive）：立即检测成功 / WS 新告警时 +1 */
  const [logRefreshToken, setLogRefreshToken] = useState(0);

  const gate = useActionPermission();
  const triggerDecision = gate.decide(PERMISSION_CODES.adDriftAlert.trigger);

  /** 刷新状态卡（稳定引用，供 mount / 立即检测 / WS 回调复用） */
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

  // WS 订阅：收到 ad_sync_drift_alert → 全局提示 + 刷新状态卡与日志表。
  // onADDriftAlert 只订阅 noticeStore 的模块级 listener Set，不新建 WebSocket 连接。
  // cleanup 返回退订函数，卸载后不再响应。
  useEffect(() => {
    const unsubscribe = onADDriftAlert((event) => {
      const title = event.title ?? "AD 同步漂移告警";
      const detail =
        event.memberCount !== undefined && event.distinctGroups !== undefined
          ? `成员数 ${event.memberCount} / 组数 ${event.distinctGroups}`
          : "检测到 AD 同步数据漂移";
      getAppMessage().warning(`${title}：${detail}`);
      void refreshStatus();
      bumpLogs();
    });
    return () => {
      unsubscribe();
    };
  }, [refreshStatus, bumpLogs]);

  /** 「立即检测」：写操作，走 :trigger 门控；成功后刷新状态 + 日志 */
  const handleTrigger = useCallback(async () => {
    setTriggering(true);
    try {
      const result = await triggerOnce();
      handleSuccess("立即检测");
      // B-1 / v1.56：trigger 端点实际只返回 {message}，不再按虚构的 drifted 分支判断
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

  const triggerDisabled = triggerDecision.disabled || triggering;

  return (
    <div style={{ padding: 16 }}>
      <Space orientation="vertical" size="medium" style={{ width: "100%" }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <Title level={4} style={{ margin: 0 }}>
            {PAGE_TITLES.AD_DRIFT_ALERT}
          </Title>
          {triggerDecision.visible && (
            <Button
              type="primary"
              icon={<ThunderboltOutlined />}
              onClick={handleTrigger}
              loading={triggering}
              disabled={triggerDisabled}
              aria-label="立即检测"
              title={triggerDecision.hint ?? "立即检测"}
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
    </div>
  );
};

export default ADDriftAlertPage;
