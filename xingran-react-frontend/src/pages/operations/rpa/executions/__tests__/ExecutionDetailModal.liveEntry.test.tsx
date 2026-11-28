/**
 * Phase 155 / LIVEVIEW-04：执行详情 footer 的「实时画面」入口
 *
 * 断言点（红绿变异点）：
 *   1. footer 里确实存在「实时画面」按钮（screen.getByText 取到）
 *   2. 点击后 Modal 关闭且导航到 live 页并带上 ?executionId=
 *   3. 既有 footer 按钮（下载执行产物 / 关闭）未被破坏
 *
 * setup 照抄同目录 ExecutionDetailModal.render.test.tsx 的 mock 形态
 * （vi.mock("@/lib/api", createApiTestingModule)）—— 不要修改那里的既有测试。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import { App as AntdApp } from "antd";
import type { Execution } from "@/types/rpa";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { ExecutionDetailModal } from "../ExecutionDetailModal";
import { createApiMock, resetApiMocks } from "@/test/utils/createApiMock";

const exec: Execution = {
  id: "exec-live-1",
  taskName: "liveview-task",
  status: "running",
  startedAt: "2026-11-24 10:00:00",
  duration: 0,
  workerName: "worker-1",
} as unknown as Execution;

beforeEach(() => {
  resetApiMocks();
  // 日志端点返回空数组，避免 3s 轮询干扰断言
  const logsApi = createApiMock("/rpa/executions/exec-live-1/logs");
  logsApi.endpoint.mockResolvedValue({ code: 0, data: { logs: [] } });
});

afterEach(() => {
  resetApiMocks();
});

describe("ExecutionDetailModal footer 实时画面入口", () => {
  it("footer 存在「实时画面」按钮，且既有按钮未丢", async () => {
    render(
      <MemoryRouter initialEntries={["/operations/rpa/executions"]}>
        <AntdApp>
          <ExecutionDetailModal open execution={exec} onClose={vi.fn()} />
        </AntdApp>
      </MemoryRouter>
    );

    // Modal 异步挂载，等 footer 出现
    await waitFor(() => {
      expect(screen.getByText("实时画面")).toBeInTheDocument();
    });
    expect(screen.getByText("下载执行产物")).toBeInTheDocument();
    // antd Button 对「两个汉字的文字」会插入空格（autoInsertSpace），
    // 因此「关闭」渲染为「关 闭」—— 用正则匹配避免依赖这一排版细节。
    expect(screen.getByRole("button", { name: /关\s*闭/ })).toBeInTheDocument();
  });

  it("点击「实时画面」→ 关闭 Modal 并跳到 live 页（带 executionId）", async () => {
    const onClose = vi.fn();
    // jsdom 实现了 window.location.assign（会报 "Not implemented: navigation"）
    // 故 stub 掉它，只断言传参（整页跳转是本 phase 唯一允许的入口形态）。
    const assignSpy = vi.fn();
    const originalAssign = window.location.assign;
    Object.defineProperty(window, "location", {
      value: { ...window.location, assign: assignSpy, search: "", pathname: "/" },
      writable: true,
      configurable: true,
    });

    render(
      <MemoryRouter initialEntries={["/operations/rpa/executions"]}>
        <AntdApp>
          <ExecutionDetailModal open execution={exec} onClose={onClose} />
        </AntdApp>
      </MemoryRouter>
    );

    await waitFor(() => {
      expect(screen.getByText("实时画面")).toBeInTheDocument();
    });

    fireEvent.click(screen.getByText("实时画面"));

    await waitFor(() => {
      expect(onClose).toHaveBeenCalled();
      expect(assignSpy).toHaveBeenCalledWith(
        expect.stringContaining("/vdi/rpa/live?executionId=exec-live-1")
      );
    });

    Object.defineProperty(window, "location", {
      value: { ...window.location, assign: originalAssign, search: "", pathname: "/" },
      writable: true,
      configurable: true,
    });
  });
});
