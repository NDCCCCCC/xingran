/**
 * Phase 144 ALRT-03 — AD 同步漂移告警 WS 订阅契约测试
 *
 * 本 phase **未新建第二条 WebSocket 连接**，也未抽独立 useADDriftAlertWebSocket
 * hook：页面 index.tsx 直接订阅 noticeStore 导出的 onADDriftAlert（复用
 * NotificationBell.tsx 建立的全局唯一 /system/ws/notices 连接）。
 *
 * 因此本文件直接测「订阅链路」的契约（PLAN Task 4 允许的等价覆盖）：
 *   1. 收到 ad_sync_drift_alert → 刷新回调被调用一次；
 *   2. 退订后不再触发（对应页面 useEffect cleanup）；
 *   3. 其它 WS 类型（new_notice / rpa_* / port_write_*）不误触发；
 *   4. 静态断言：本 phase 前端不出现第二条 new WebSocket（全局唯一连接口径）。
 *
 * 另：页面内订阅/退订的 React 生命周期覆盖见
 * src/pages/system/ad-drift-alert/__tests__/index.render.test.tsx。
 */
import { describe, it, expect, vi } from "vitest";
import { readFileSync } from "node:fs";
import path from "node:path";
import { useNoticeStore } from "@/store/noticeStore";

const driftEvent = {
  type: "ad_sync_drift_alert",
  noticeId: "notice-ws-1",
  memberCount: 15,
  distinctGroups: 3,
  baseline: 20,
  tolerance: 0,
  throttled: false,
  timestamp: 1766380800,
};

function emitDrift() {
  useNoticeStore.getState().handleWsMessage({
    type: "ad_sync_drift_alert",
    data: null,
    timestamp: 1,
    content: JSON.stringify(driftEvent),
  } as never);
}

describe("ALRT-03 onADDriftAlert 订阅契约（复用全局唯一 WS）", () => {
  it("收到 ad_sync_drift_alert 时刷新回调被调用一次", () => {
    const refresh = vi.fn();
    const unsub = useNoticeStore.getState().onADDriftAlert(refresh);
    useNoticeStore.getState().reset(); // reset 不清模块级 listener（同 RPA 约定）

    emitDrift();

    expect(refresh).toHaveBeenCalledTimes(1);
    expect(refresh).toHaveBeenCalledWith(
      expect.objectContaining({ memberCount: 15, noticeId: "notice-ws-1" })
    );
    unsub();
  });

  it("退订（页面 cleanup）后不再触发", () => {
    const refresh = vi.fn();
    const unsub = useNoticeStore.getState().onADDriftAlert(refresh);

    emitDrift();
    expect(refresh).toHaveBeenCalledTimes(1);

    unsub();
    emitDrift();
    expect(refresh).toHaveBeenCalledTimes(1);
  });

  it("其它 WS 类型不触发 drift 回调", () => {
    const refresh = vi.fn();
    const unsub = useNoticeStore.getState().onADDriftAlert(refresh);

    useNoticeStore.getState().handleWsMessage({
      type: "new_notice",
      data: {
        noticeId: "n1",
        noticeTitle: "普通通知",
        noticeType: "1",
        priority: 0,
        isMarkdown: false,
        createdAt: "2026-11-09",
      },
      timestamp: 1,
    } as never);
    useNoticeStore.getState().handleWsMessage({
      type: "rpa_progress",
      data: null,
      timestamp: 1,
      content: JSON.stringify({ type: "rpa_progress", executionId: "e1" }),
    } as never);
    useNoticeStore.getState().handleWsMessage({
      type: "port_write_progress",
      data: null,
      timestamp: 1,
      content: JSON.stringify({ type: "port_write_progress", batchId: "b1" }),
    } as never);

    expect(refresh).not.toHaveBeenCalled();
    unsub();
  });

  it("全局唯一连接：本 phase 前端文件不出现第二条 new WebSocket", () => {
    const files = [
      "src/store/noticeStore.ts",
      "src/pages/system/ad-drift-alert/index.tsx",
      "src/pages/system/ad-drift-alert/components/StatusCards.tsx",
      "src/pages/system/ad-drift-alert/components/TriggerLogTable.tsx",
      "src/pages/system/ad-drift-alert/components/ConfigForm.tsx",
      "src/lib/adDriftAlertApi.ts",
    ];
    // 进程 cwd 为仓库根（vitest 从前端根启动时用相对前端根的路径回退）
    const roots = [process.cwd(), path.resolve(process.cwd(), "xingran-react-frontend")];
    for (const rel of files) {
      let text: string | null = null;
      for (const root of roots) {
        try {
          text = readFileSync(path.join(root, rel), "utf8");
          break;
        } catch {
          // 尝试下一个根
        }
      }
      expect(text, `应能读取 ${rel}`).not.toBeNull();
      expect(text).not.toMatch(/new\s+WebSocket/);
    }
  });
});
