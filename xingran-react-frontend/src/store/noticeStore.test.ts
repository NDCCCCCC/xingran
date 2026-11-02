/**
 * noticeStore 通知状态测试
 *
 * 覆盖:setUnreadCount/setNotifications/addNotification(去重+50 条上限+未读自增)/
 * markAsRead/markAllAsRead/removeNotification/setLoading/setWsConnected/
 * handleWsMessage(new_notice + rpa 三类消息 content/data 解析/解析失败/回调抛错)/
 * onRPAProgress 订阅与退订(P1-M4 模块级 Set)/reset。
 */
import { describe, it, expect, beforeEach, vi } from "vitest";
import { useNoticeStore } from "./noticeStore";
import type { RPAProgressMessage } from "@/types/rpa";

const notice = (id: string) => ({
  id,
  noticeTitle: `标题-${id}`,
  noticeType: "announcement",
  priority: "normal",
  isRead: false,
  createdAt: "2026-08-24",
});

const rpaMsg = (overrides: Partial<RPAProgressMessage> = {}): RPAProgressMessage => ({
  type: "rpa_progress",
  executionId: "exec-1",
  taskId: "task-1",
  taskName: "任务",
  step: 1,
  total: 3,
  message: "running",
  status: "running",
  timestamp: 1,
  ...overrides,
});

describe("noticeStore", () => {
  let consoleSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(() => {
    useNoticeStore.getState().reset();
    consoleSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    consoleSpy.mockRestore();
  });

  it("setUnreadCount / setNotifications / setLoading / setWsConnected", () => {
    const s = useNoticeStore.getState();
    s.setUnreadCount(5);
    s.setNotifications([notice("n1")]);
    s.setLoading(true);
    s.setWsConnected(true);

    const state = useNoticeStore.getState();
    expect(state.unreadCount).toBe(5);
    expect(state.notifications).toHaveLength(1);
    expect(state.loading).toBe(true);
    expect(state.wsConnected).toBe(true);
  });

  it("addNotification 顶部插入并自增未读;重复 id 去重", () => {
    const s = useNoticeStore.getState();
    s.addNotification(notice("n1"));
    s.addNotification(notice("n2"));

    let state = useNoticeStore.getState();
    expect(state.notifications.map((n) => n.id)).toEqual(["n2", "n1"]);
    expect(state.unreadCount).toBe(2);

    s.addNotification(notice("n2")); // 已存在
    state = useNoticeStore.getState();
    expect(state.notifications).toHaveLength(2);
    expect(state.unreadCount).toBe(2);
  });

  it("addNotification 最多保留 50 条", () => {
    const s = useNoticeStore.getState();
    for (let i = 0; i < 55; i++) {
      s.addNotification(notice(`n${i}`));
    }
    const state = useNoticeStore.getState();
    expect(state.notifications).toHaveLength(50);
    expect(state.notifications[0].id).toBe("n54"); // 最新的在顶部
    // unreadCount 表达服务器侧未读数,不随本地 50 条淘汰回退
    expect(state.unreadCount).toBe(55);
  });

  it("markAsRead 已读一条且未读数不低于 0;markAllAsRead 全读", () => {
    const s = useNoticeStore.getState();
    s.setNotifications([notice("n1"), notice("n2")]);
    s.setUnreadCount(2);

    s.markAsRead("n1");
    let state = useNoticeStore.getState();
    expect(state.notifications.find((n) => n.id === "n1")!.isRead).toBe(true);
    expect(state.unreadCount).toBe(1);

    s.markAsRead("n1"); // 再次标记不会负数
    expect(useNoticeStore.getState().unreadCount).toBe(0);

    s.markAllAsRead();
    state = useNoticeStore.getState();
    expect(state.notifications.every((n) => n.isRead)).toBe(true);
    expect(state.unreadCount).toBe(0);
  });

  it("removeNotification 移除指定通知", () => {
    const s = useNoticeStore.getState();
    s.setNotifications([notice("n1"), notice("n2")]);
    s.removeNotification("n1");
    expect(useNoticeStore.getState().notifications.map((n) => n.id)).toEqual(["n2"]);
  });

  it("handleWsMessage new_notice:新增通知 + 未读自增", () => {
    useNoticeStore.getState().handleWsMessage({
      type: "new_notice",
      data: {
        noticeId: "ws-1",
        noticeTitle: "WS 通知",
        noticeType: "announcement",
        priority: "high",
        isMarkdown: false,
        createdAt: "2026-08-24",
      },
      timestamp: 1,
    } as never);

    const state = useNoticeStore.getState();
    expect(state.notifications[0]).toMatchObject({ id: "ws-1", noticeTitle: "WS 通知" });
    expect(state.unreadCount).toBe(1);
  });

  it("handleWsMessage rpa 进度:content JSON 字符串触发监听器", () => {
    const cb = vi.fn();
    const unsub = useNoticeStore.getState().onRPAProgress(cb);

    useNoticeStore.getState().handleWsMessage({
      type: "rpa_progress",
      data: null,
      timestamp: 1,
      content: JSON.stringify(rpaMsg({ step: 2 })),
    } as never);

    expect(cb).toHaveBeenCalledWith(expect.objectContaining({ step: 2, executionId: "exec-1" }));

    // 退订后不再接收
    unsub();
    useNoticeStore.getState().handleWsMessage({
      type: "rpa_completed",
      data: null,
      timestamp: 1,
      content: JSON.stringify(rpaMsg({ type: "rpa_completed" })),
    } as never);
    expect(cb).toHaveBeenCalledTimes(1);
  });

  it("handleWsMessage rpa 进度:data 字段直传(content 缺失时)", () => {
    const cb = vi.fn();
    useNoticeStore.getState().onRPAProgress(cb);

    useNoticeStore.getState().handleWsMessage({
      type: "rpa_failed",
      data: rpaMsg({ type: "rpa_failed", executionId: "exec-data" }),
      timestamp: 1,
    } as never);

    expect(cb).toHaveBeenCalledWith(
      expect.objectContaining({ type: "rpa_failed", executionId: "exec-data" })
    );
  });

  it("handleWsMessage rpa 进度:content 与 data 均缺失时静默返回", () => {
    const cb = vi.fn();
    useNoticeStore.getState().onRPAProgress(cb);

    useNoticeStore.getState().handleWsMessage({
      type: "rpa_progress",
      data: null,
      timestamp: 1,
    } as never);

    expect(cb).not.toHaveBeenCalled();
  });

  it("handleWsMessage content 非法 JSON 走 parse error 分支", () => {
    const cb = vi.fn();
    useNoticeStore.getState().onRPAProgress(cb);

    useNoticeStore.getState().handleWsMessage({
      type: "rpa_progress",
      data: null,
      timestamp: 1,
      content: "not-json{{{",
    } as never);

    expect(cb).not.toHaveBeenCalled();
    expect(consoleSpy).toHaveBeenCalled();
  });

  it("监听器回调抛错不影响其他监听器(P1-M4 隔离)", () => {
    const bad = vi.fn(() => {
      throw new Error("listener crash");
    });
    const good = vi.fn();
    useNoticeStore.getState().onRPAProgress(bad);
    useNoticeStore.getState().onRPAProgress(good);

    useNoticeStore.getState().handleWsMessage({
      type: "rpa_progress",
      data: null,
      timestamp: 1,
      content: JSON.stringify(rpaMsg()),
    } as never);

    expect(bad).toHaveBeenCalledTimes(1);
    expect(good).toHaveBeenCalledTimes(1);
    expect(consoleSpy).toHaveBeenCalled();
  });

  it("非 new_notice/rpa 类型消息被忽略", () => {
    const cb = vi.fn();
    useNoticeStore.getState().onRPAProgress(cb);

    useNoticeStore.getState().handleWsMessage({
      type: "unknown_type" as never,
      data: null,
      timestamp: 1,
    } as never);

    expect(useNoticeStore.getState().notifications).toEqual([]);
    expect(cb).not.toHaveBeenCalled();
  });

  it("reset 回初始状态", () => {
    const s = useNoticeStore.getState();
    s.addNotification(notice("n1"));
    s.setWsConnected(true);

    useNoticeStore.getState().reset();

    const state = useNoticeStore.getState();
    expect(state.unreadCount).toBe(0);
    expect(state.notifications).toEqual([]);
    expect(state.loading).toBe(false);
    expect(state.wsConnected).toBe(false);
  });
});
