/**
 * Phase 88 Batch61 — useDutyConfig hook 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";
import dayjs from "dayjs";

vi.mock("@/lib/dutyApi", () => ({
  getDutyConfig: vi.fn().mockResolvedValue({
    data: { reminderEnabled: true, reminderTime: "09:00", reminderChannels: "email,sms" },
  }),
  updateDutyConfig: vi.fn().mockResolvedValue({ data: {} }),
}));

import { getDutyConfig, updateDutyConfig } from "@/lib/dutyApi";
import { useDutyConfig } from "../hooks/useDutyConfig";

beforeEach(() => {
  vi.clearAllMocks();
});

const wrap = ({ children }: { children: React.ReactNode }) => (
  <ConfigProvider>
    <App>{children}</App>
  </ConfigProvider>
);

describe("useDutyConfig", () => {
  it("initial state + handlers", () => {
    const { result } = renderHook(() => useDutyConfig(), { wrapper: wrap });
    expect(result.current.loading).toBe(false);
    expect(result.current.saving).toBe(false);
    expect(result.current.config).toBeNull();
    expect(typeof result.current.fetch).toBe("function");
    expect(typeof result.current.save).toBe("function");
    expect(typeof result.current.getFormValues).toBe("function");
  });

  it("fetch 加载配置", async () => {
    const { result } = renderHook(() => useDutyConfig(), { wrapper: wrap });
    await act(async () => {
      await result.current.fetch();
    });
    expect(getDutyConfig).toHaveBeenCalled();
    expect(result.current.config).not.toBeNull();
    expect(result.current.config?.reminderEnabled).toBe(true);
  });

  it("fetch error → message.error + config=null", async () => {
    vi.mocked(getDutyConfig).mockRejectedValueOnce(new Error("fail"));
    const { result } = renderHook(() => useDutyConfig(), { wrapper: wrap });
    let r;
    await act(async () => {
      r = await result.current.fetch();
    });
    expect(r).toBeNull();
    expect(result.current.config).toBeNull();
  });

  it("save 调 updateDutyConfig + 返回 true", async () => {
    const { result } = renderHook(() => useDutyConfig(), { wrapper: wrap });
    await act(async () => {
      await result.current.fetch();
    });
    let ok;
    await act(async () => {
      ok = await result.current.save({
        reminderEnabled: true,
        reminderTime: dayjs("2026-08-29 09:00"),
        reminderChannels: ["email"],
        beforeReminderMinutes: 30,
      });
    });
    expect(updateDutyConfig).toHaveBeenCalled();
    expect(ok).toBe(true);
  });

  it("save error → 返回 false", async () => {
    vi.mocked(updateDutyConfig).mockRejectedValueOnce(new Error("save fail"));
    const { result } = renderHook(() => useDutyConfig(), { wrapper: wrap });
    let ok;
    await act(async () => {
      ok = await result.current.save({
        reminderEnabled: false,
        reminderTime: dayjs("2026-08-29 09:00"),
        reminderChannels: [],
      });
    });
    expect(ok).toBe(false);
  });

  it("getFormValues 在 config=null 时返回空对象", () => {
    const { result } = renderHook(() => useDutyConfig(), { wrapper: wrap });
    expect(result.current.getFormValues()).toEqual({});
  });

  it("getFormValues 在 config 非空时拆分 reminderChannels 字符串", async () => {
    const { result } = renderHook(() => useDutyConfig(), { wrapper: wrap });
    await act(async () => {
      await result.current.fetch();
    });
    const vals = result.current.getFormValues();
    expect(vals.reminderEnabled).toBe(true);
    expect(vals.reminderChannels).toEqual(["email", "sms"]);
  });
});
