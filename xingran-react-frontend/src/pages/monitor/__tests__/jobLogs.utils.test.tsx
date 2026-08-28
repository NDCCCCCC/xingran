/**
 * Phase 88 Batch26 — monitor job/logs utils + useLogActions 补测
 */
import { describe, it, expect, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App } from "antd";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { formatLocalTime, formatDuration, renderJobStatusTag, renderConcurrentTag, renderCronExpression, renderExceptionInfo } from "../job/utils";
import { getBusinessTypeLabel, renderRequestMethodTag, renderLogStatusTag, processTimeRangeParams } from "../logs/utils";
import { useLogActions } from "../logs/hooks/useLogActions";
import * as apiModule from "@/lib/api";

describe("monitor/job/utils", () => {
  it("formatLocalTime null 返回占位", () => {
    // formatDateTime(null) → "-" 或空,只断言不 throw
    const r = formatLocalTime(null);
    expect(typeof r).toBe("string");
  });

  it("formatDuration 三分支", () => {
    expect(formatDuration(0)).toBe("-");
    expect(formatDuration(500)).toBe("500ms");
    expect(formatDuration(1500)).toBe("1.50s");
    expect(formatDuration(123456)).toBe("123.46s");
  });

  it("renderJobStatusTag 正常/暂停", () => {
    // React 元素断言 props
    const normal: any = renderJobStatusTag(0);
    expect(normal.props.color).toBe("success");
    expect(normal.props.children).toBe("正常");
    const paused: any = renderJobStatusTag(1);
    expect(paused.props.color).toBe("warning");
    expect(paused.props.children).toBe("暂停");
  });

  it("renderConcurrentTag 允许/禁止", () => {
    const yes: any = renderConcurrentTag(true);
    expect(yes.props.color).toBe("green");
    expect(yes.props.children).toBe("允许");
    const no: any = renderConcurrentTag(false);
    expect(no.props.color).toBe("orange");
    expect(no.props.children).toBe("禁止");
  });

  it("renderCronExpression 返回 Tooltip 包裹 code", () => {
    const el: any = renderCronExpression("0 0 * * *");
    expect(el.type.displayName ?? el.type.name).toMatch(/Tooltip/);
  });

  it("renderExceptionInfo 空值 '-' 非空 Tooltip", () => {
    expect(renderExceptionInfo("")).toBe("-");
    const el: any = renderExceptionInfo("boom");
    expect(el).toBeDefined();
  });
});

describe("monitor/logs/utils", () => {
  it("getBusinessTypeLabel 已知/未知", () => {
    // BUSINESS_TYPE_OPTIONS 至少含 0
    expect(typeof getBusinessTypeLabel(0)).toBe("string");
    expect(getBusinessTypeLabel(9999)).toBe("-");
  });

  it("renderRequestMethodTag 色彩映射", () => {
    const get: any = renderRequestMethodTag("GET");
    expect(get.props.color).toBe("blue");
    const post: any = renderRequestMethodTag("POST");
    expect(post.props.color).toBe("green");
    const put: any = renderRequestMethodTag("PUT");
    expect(put.props.color).toBe("orange");
    const del: any = renderRequestMethodTag("DELETE");
    expect(del.props.color).toBe("red");
    const other: any = renderRequestMethodTag("PATCH");
    expect(other.props.color).toBe("default");
  });

  it("renderLogStatusTag oper/login 双语义", () => {
    const ok: any = renderLogStatusTag(0, "oper");
    expect(ok.props.children).toBe("正常");
    const bad: any = renderLogStatusTag(1, "oper");
    expect(bad.props.children).toBe("异常");
    const loginOk: any = renderLogStatusTag(0, "login");
    expect(loginOk.props.children).toBe("成功");
    const loginBad: any = renderLogStatusTag(1, "login");
    expect(loginBad.props.children).toBe("失败");
  });

  it("renderLogStatusTag 默认 type=oper", () => {
    const d: any = renderLogStatusTag(0);
    expect(d.props.children).toBe("正常");
  });

  it("processTimeRangeParams 双元素写入 ISO 时间", () => {
    const params: Record<string, any> = {};
    const start = new Date("2026-08-01T00:00:00Z");
    const end = new Date("2026-08-28T00:00:00Z");
    processTimeRangeParams([start, end], params);
    expect(params.startTime).toBe(start.toISOString());
    expect(params.endTime).toBe(end.toISOString());
  });

  it("processTimeRangeParams 空不写", () => {
    const params: Record<string, any> = {};
    processTimeRangeParams(null, params);
    processTimeRangeParams([new Date()], params); // 长度 1 不写
    expect(params.startTime).toBeUndefined();
    expect(params.endTime).toBeUndefined();
  });
});

describe("useLogActions", () => {
  // App.useApp 需要 antd App context;用组件函数形式 wrapper(hook 渲染约定)
  function Wrap({ children }: { children: React.ReactNode }) {
    return <App>{children}</App>;
  }
  const wrap = Wrap;

  it("initial state + handleViewDetail", () => {
    const { result } = renderHook(
      () =>
        useLogActions({
          activeTab: "oper",
          fetchOperLogs: vi.fn(),
          fetchLoginLogs: vi.fn(),
        }),
      { wrapper: wrap }
    );
    expect(result.current.detailModalVisible).toBe(false);
    expect(result.current.selectedLog).toBeNull();

    const rec = { id: "l1" };
    act(() => {
      result.current.handleViewDetail(rec);
    });
    expect(result.current.detailModalVisible).toBe(true);
    expect(result.current.selectedLog).toBe(rec);
  });

  it("handleRefresh 按 activeTab 分发", () => {
    const fetchOper = vi.fn();
    const fetchLogin = vi.fn();
    const { result } = renderHook(
      () =>
        useLogActions({
          activeTab: "oper",
          fetchOperLogs: fetchOper,
          fetchLoginLogs: fetchLogin,
        }),
      { wrapper: wrap }
    );
    act(() => {
      result.current.handleRefresh();
    });
    expect(fetchOper).toHaveBeenCalledTimes(1);
    expect(fetchLogin).not.toHaveBeenCalled();
  });

  it("handleRefresh login tab 分发", () => {
    const fetchOper = vi.fn();
    const fetchLogin = vi.fn();
    const { result } = renderHook(
      () =>
        useLogActions({
          activeTab: "login",
          fetchOperLogs: fetchOper,
          fetchLoginLogs: fetchLogin,
        }),
      { wrapper: wrap }
    );
    act(() => {
      result.current.handleRefresh();
    });
    expect(fetchLogin).toHaveBeenCalledTimes(1);
  });

  it("setters 可直接改 state", () => {
    const { result } = renderHook(
      () =>
        useLogActions({
          activeTab: "oper",
          fetchOperLogs: vi.fn(),
          fetchLoginLogs: vi.fn(),
        }),
      { wrapper: wrap }
    );
    act(() => {
      result.current.setDetailModalVisible(true);
      result.current.setSelectedLog({ x: 1 });
    });
    expect(result.current.detailModalVisible).toBe(true);
    expect(result.current.selectedLog).toEqual({ x: 1 });
  });
});
