/**
 * Phase 88 Batch60 — monitor/job useJobActions 测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { App, ConfigProvider } from "antd";

vi.mock("@/lib/api", () => ({
  post: vi.fn().mockResolvedValue({ data: {} }),
  postLongRequest: vi.fn().mockResolvedValue({ data: {} }),
}));

import { useJobActions } from "../hooks/useJobActions";
import type { JobInfo } from "../types";
import { post, postLongRequest } from "@/lib/api";

beforeEach(() => {
  vi.clearAllMocks();
});

const wrap = ({ children }: { children: React.ReactNode }) => (
  <ConfigProvider>
    <App>{children}</App>
  </ConfigProvider>
);

const baseParams = () => ({
  fetchJobs: vi.fn().mockResolvedValue(undefined),
  fetchJobLogs: vi.fn().mockResolvedValue(undefined),
});

describe("useJobActions — initial state", () => {
  it("modalVisible=false + modalTitle='' + 8 handlers 存在", () => {
    const { result } = renderHook(() => useJobActions(baseParams()), { wrapper: wrap });
    expect(result.current.modalVisible).toBe(false);
    expect(result.current.modalTitle).toBe("");
    expect(result.current.isEdit).toBe(false);
    expect(result.current.logDrawerVisible).toBe(false);
    expect(result.current.selectedJob).toBeNull();
    expect(typeof result.current.openModal).toBe("function");
    expect(typeof result.current.handleSubmit).toBe("function");
    expect(typeof result.current.handleDelete).toBe("function");
    expect(typeof result.current.handleToggleStatus).toBe("function");
    expect(typeof result.current.handleExecute).toBe("function");
    expect(typeof result.current.handleViewLogs).toBe("function");
    expect(typeof result.current.handleReset).toBe("function");
  });
});

describe("useJobActions — openModal / handleViewLogs", () => {
  it("openModal(undefined, form) 设新增标题 + selectedJob=null", () => {
    const { result } = renderHook(() => useJobActions(baseParams()), { wrapper: wrap });
    const form = { setFieldsValue: vi.fn(), resetFields: vi.fn() } as any;
    act(() => result.current.openModal(undefined, form));
    expect(result.current.modalVisible).toBe(true);
    expect(result.current.modalTitle).toContain("新增");
    expect(result.current.isEdit).toBe(false);
    expect(result.current.selectedJob).toBeNull();
  });

  it("openModal(record, form) 设编辑标题", () => {
    const { result } = renderHook(() => useJobActions(baseParams()), { wrapper: wrap });
    const form = { setFieldsValue: vi.fn(), resetFields: vi.fn() } as any;
    act(() =>
      result.current.openModal(
        {
          id: 1,
          jobName: "job1",
          jobGroup: "DEFAULT",
          invokeTarget: "x",
          cronExpression: "0 0 * * *",
          misfirePolicy: 1,
          concurrent: false,
          status: 0,
        } as any,
        form
      )
    );
    expect(result.current.isEdit).toBe(true);
    expect(result.current.modalTitle).toContain("编辑");
    expect(form.setFieldsValue).toHaveBeenCalled();
  });

  it("handleViewLogs 开 logDrawer", () => {
    const { result } = renderHook(() => useJobActions(baseParams()), { wrapper: wrap });
    act(() =>
      result.current.handleViewLogs({
        id: 1,
        jobName: "job1",
        jobGroup: "DEFAULT",
      } as JobInfo)
    );
    expect(result.current.logDrawerVisible).toBe(true);
    expect(result.current.selectedJob?.jobName).toBe("job1");
  });
});

describe("useJobActions — handleSubmit", () => {
  it("create path: validateFields + post 新增", async () => {
    vi.mocked(post).mockResolvedValue({ data: {} });
    const fetchJobs = vi.fn().mockResolvedValue(undefined);
    const { result } = renderHook(() => useJobActions({ fetchJobs, fetchJobLogs: vi.fn() }), {
      wrapper: wrap,
    });
    const form = {
      validateFields: vi.fn().mockResolvedValue({ jobName: "new", cron: "0 0 * * *" }),
      resetFields: vi.fn(),
      setFieldsValue: vi.fn(),
    } as any;
    await act(async () => {
      await result.current.handleSubmit(form);
    });
    expect(post).toHaveBeenCalled();
    expect(result.current.modalVisible).toBe(false);
    expect(fetchJobs).toHaveBeenCalled();
  });

  it("update path: isEdit=true → post 更新", async () => {
    vi.mocked(post).mockResolvedValue({ data: {} });
    const { result } = renderHook(() => useJobActions(baseParams()), { wrapper: wrap });
    act(() => {
      result.current.setIsEdit(true);
    });
    const form = {
      validateFields: vi.fn().mockResolvedValue({ jobName: "edit" }),
      resetFields: vi.fn(),
      setFieldsValue: vi.fn(),
    } as any;
    await act(async () => {
      await result.current.handleSubmit(form);
    });
    expect(post).toHaveBeenCalled();
  });

  it("validate 失败 → 不调 post", async () => {
    const form = {
      validateFields: vi.fn().mockRejectedValue({ errorFields: [] }),
      resetFields: vi.fn(),
      setFieldsValue: vi.fn(),
    } as any;
    const { result } = renderHook(() => useJobActions(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleSubmit(form);
    });
    expect(post).not.toHaveBeenCalled();
  });
});

describe("useJobActions — handleDelete / handleToggleStatus / handleExecute", () => {
  it("handleDelete 调 post", async () => {
    vi.mocked(post).mockResolvedValue({ data: {} });
    const fetchJobs = vi.fn();
    const { result } = renderHook(() => useJobActions({ fetchJobs, fetchJobLogs: vi.fn() }), {
      wrapper: wrap,
    });
    await act(async () => {
      await result.current.handleDelete({ id: 1, jobName: "j1" } as JobInfo);
    });
    expect(post).toHaveBeenCalled();
    expect(fetchJobs).toHaveBeenCalled();
  });

  it("handleToggleStatus isEnabled=true → post stop", async () => {
    vi.mocked(post).mockResolvedValue({ data: {} });
    const { result } = renderHook(() => useJobActions(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleToggleStatus({
        id: 1,
        jobName: "j1",
        isEnabled: true,
      } as unknown as JobInfo);
    });
    expect(post).toHaveBeenCalled();
  });

  it("handleExecute 调 postLongRequest", async () => {
    vi.mocked(postLongRequest).mockResolvedValue({ data: {} });
    const { result } = renderHook(() => useJobActions(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleExecute({ id: 1, jobName: "j1" } as JobInfo);
    });
    expect(postLongRequest).toHaveBeenCalled();
  });

  it("handleDelete 抛错 → 走 message.error", async () => {
    vi.mocked(post).mockRejectedValue(new Error("del fail"));
    const { result } = renderHook(() => useJobActions(baseParams()), { wrapper: wrap });
    await act(async () => {
      await result.current.handleDelete({ id: 1, jobName: "j1" } as JobInfo);
    });
    expect(post).toHaveBeenCalled();
  });
});

describe("useJobActions — handleReset", () => {
  it("重置 searchForm + current + 调 fetchJobs", () => {
    const setSearchForm = vi.fn();
    const setCurrent = vi.fn();
    const fetchJobs = vi.fn();
    const { result } = renderHook(() => useJobActions(baseParams()), { wrapper: wrap });
    act(() => {
      result.current.handleReset(setSearchForm, setCurrent, fetchJobs);
    });
    expect(setSearchForm).toHaveBeenCalled();
    expect(setCurrent).toHaveBeenCalledWith(1);
    expect(fetchJobs).toHaveBeenCalled();
  });
});
