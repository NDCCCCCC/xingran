/**
 * Phase 84 plan 0 — harness 自身契约测试。
 *
 * 守护 D-02/D-03 harness 两件套的关键行为,防止后续 wave 静默破坏:
 *   - createApiTestingModule 经 vi.mock("@/lib/api") 后:端点命中 → endpoint
 *     spy;未命中 → 共享通用 verb spy 回退;mockApiBatch 批量登记;resetApiMocks 清态
 *   - renderWithProviders 默认 MemoryRouter + antd App(App.useApp() context 可用)
 */
import { describe, it, expect, vi } from "vitest";
import { fireEvent, screen } from "@testing-library/dom";
import { App as AntdApp, Button } from "antd";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// 被测组件形态的静态 import 必须在 vi.mock 之后(hoist 由 vitest 保证)
import { post } from "@/lib/api";
import { createApiMock, mockApiBatch, resetApiMocks } from "@/test/utils/createApiMock";
import { renderWithProviders } from "@/test/utils/renderWithProviders";

describe("[smoke] createApiMock interception via vi.mock(@/lib/api)", () => {
  it("routes registered endpoint call to endpoint spy", async () => {
    const handle = createApiMock("/ops/demo/list");
    handle.endpoint.mockResolvedValue({ code: 0, data: { list: [1] } });

    const res = await post("/ops/demo/list", { current: 1 });

    expect(res).toEqual({ code: 0, data: { list: [1] } });
    expect(handle.endpoint).toHaveBeenCalledWith("/ops/demo/list", { current: 1 });
    expect(handle.post).not.toHaveBeenCalled();
  });

  it("falls back to shared generic post spy for unregistered urls", async () => {
    createApiMock("/never/called");

    await post("/some/other/url");

    const generic = createApiMock("/probe").post;
    expect(generic).toHaveBeenCalledWith("/some/other/url");
  });

  it("mockApiBatch registers several endpoints with default responses", async () => {
    const handles = mockApiBatch([
      { endpoint: "/a", response: { code: 0, data: "A" } },
      { endpoint: "/b" },
    ]);

    expect((await post("/a")).data).toBe("A");
    expect(handles["/a"].endpoint).toHaveBeenCalledTimes(1);
    expect(handles["/b"].endpoint).not.toHaveBeenCalled();
  });

  it("resetApiMocks clears registry and spy history", async () => {
    const handle = createApiMock("/reset/me");
    handle.endpoint.mockResolvedValue({ code: 0, data: null });
    await post("/reset/me");
    expect(handle.endpoint).toHaveBeenCalledTimes(1);

    resetApiMocks();
    const res = await post("/reset/me"); // 回退到通用 spy,不再命中端点
    expect(res).toBeUndefined();
    expect(handle.endpoint).toHaveBeenCalledTimes(1);
  });
});

describe("[smoke] renderWithProviders antd App context", () => {
  it("provides working message context via App.useApp()", () => {
    function Probe() {
      const { message } = AntdApp.useApp();
      return <Button onClick={() => void message.success("fired")}>fire-message</Button>;
    }

    renderWithProviders(<Probe />, { route: "/monitor" });
    expect(screen.getByRole("button", { name: "fire-message" })).toBeInTheDocument();
    expect(() =>
      fireEvent.click(screen.getByRole("button", { name: "fire-message" }))
    ).not.toThrow();
  });
});
