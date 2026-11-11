/**
 * 同步漂移告警页（Phase 144 ALRT-03）— migration_242 后保留的**兼容薄壳**
 *
 * 为什么保留：漂移告警已并入「AD 域控配置」页的第三个 Tab
 * （`src/pages/ad-domain/configs/index.tsx` + `configs/components/DriftAlertTab.tsx`），
 * 独立菜单在 migration_242 中被隐藏，不再是常态入口。
 *
 * 但本页仍在两处有存在价值：
 *   ① 旧直达链接 / 书签（`/system/ad-drift-alert`）不至于 404；
 *   ② 页面目录本身是「页面级权限门控覆盖率」口径的一部分
 *      （scripts/check-gating-coverage.mjs 的 EXPECTED_PAGE_COUNT），
 *      删页会打破该硬断言 —— 因此按「留壳」处理，而不是删除目录。
 *
 * 门控：本壳调用 useActionPermission 并把「立即检测」的决策下发给 Tab
 *      （Tab 也会在缺省时自判），保证门控点在页面可见、口径不变。
 *
 * WS：页面级订阅 `onADDriftAlert`（复用全局唯一 /system/ws/notices 连接，
 *      不新建 WebSocket），收到事件后通过 refreshToken 让 Tab 刷新数据。
 *      与 ad-domain/configs 页面级订阅同一口径。
 */
import { useEffect, useState } from "react";
import type { FC } from "react";
import { Space, Typography } from "antd";
import DriftAlertTab from "@/pages/ad-domain/configs/components/DriftAlertTab";
import { useActionPermission } from "@/components/shared/useActionPermission";
import { PERMISSION_CODES } from "@/constants/permissionCodes";
import { PAGE_TITLES } from "@/constants/pageTitles";
import { getAppMessage } from "@/utils/antdMessage";
import { onADDriftAlert } from "@/store/noticeStore";

const { Title } = Typography;

const ADDriftAlertPage: FC = () => {
  /** 页面级 WS 刷新信号：primitive，收到告警时 +1 下发给 Tab */
  const [refreshToken, setRefreshToken] = useState(0);

  // 「立即检测」三态门控（页面级判定后下发给 Tab，保持门控点在页面可见）
  const gate = useActionPermission();
  const triggerDecision = gate.decide(PERMISSION_CODES.adDriftAlert.trigger);

  // WS 订阅：收到 ad_sync_drift_alert → 全局提示 + 让 Tab 刷新状态卡与触发记录。
  // onADDriftAlert 只订阅 noticeStore 的模块级 listener Set，不新建 WebSocket 连接。
  useEffect(() => {
    const unsubscribe = onADDriftAlert((event) => {
      const title = event.title ?? "AD 同步漂移告警";
      const detail =
        event.memberCount !== undefined && event.distinctGroups !== undefined
          ? `成员数 ${event.memberCount} / 组数 ${event.distinctGroups}`
          : "检测到 AD 同步数据漂移";
      getAppMessage().warning(`${title}：${detail}`);
      setRefreshToken((n) => n + 1);
    });
    return () => {
      unsubscribe();
    };
  }, []);

  // 进入页面时不再额外 bump：Tab 首帧已自行加载（避免重复请求）。
  // refreshToken 只在 WS 事件时递增。

  return (
    <div style={{ padding: 16 }}>
      <Space orientation="vertical" size="medium" style={{ width: "100%" }}>
        <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <Title level={4} style={{ margin: 0 }}>
            {PAGE_TITLES.AD_DRIFT_ALERT}
          </Title>
        </div>

        <DriftAlertTab refreshToken={refreshToken} triggerDecision={triggerDecision} />
      </Space>
    </div>
  );
};

export default ADDriftAlertPage;
