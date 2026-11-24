/**
 * CONTRACT-05（D15）组件测试：AI 编辑器必须**实调** POST /rpa/ai/generate。
 *
 * 断言契约（红绿变异点）：
 *   1. 点「生成脚本」后 /rpa/ai/generate 端点收到真实请求（url + description）
 *   2. 正常情况下不再有任何 setTimeout 造出来的假延迟 —— 用 fake timers 把
 *      「等 1500ms」这条路彻底堵死：不推进时钟也必须拿到结果
 *   3. 三种失败态各有可见提示：failed / timeout / empty
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { App as AntdApp, Form } from "antd";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { createApiMock, resetApiMocks } from "@/test/utils/createApiMock";
import { AIScriptEditor } from "../AIScriptEditor";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

/**
 * 打开弹窗、填入描述，**先配置好端点的响应**，再点生成。
 * （顺序很关键：click 会同步触发请求，事后 mockResolvedValue 已来不及。）
 */
async function generateWith(
  description: string,
  configure?: (handle: ReturnType<typeof createApiMock>) => void
) {
  const handle = createApiMock("/rpa/ai/generate");
  configure?.(handle);
  render(<AIScriptEditor open onClose={vi.fn()} onConfirm={vi.fn()} />, { wrapper });
  const textarea = await screen.findByRole("textbox");
  fireEvent.change(textarea, { target: { value: description } });
  fireEvent.click(screen.getByRole("button", { name: /生成脚本/ }));
  return handle;
}

beforeEach(() => {
  resetApiMocks();
  vi.useFakeTimers({ shouldAdvanceTime: true });
});

afterEach(() => {
  vi.useRealTimers();
  resetApiMocks();
});

describe("AIScriptEditor 实调 POST /rpa/ai/generate", () => {
  it("发出真实 HTTP 请求（url + description），且不依赖 setTimeout 假延迟", async () => {
    const handle = await generateWith("打开登录页并输入用户名", (h) => {
      h.endpoint.mockResolvedValue({
        code: 0,
        data: {
          actions: [
            {
              type: "navigate",
              value: "https://example.com/login",
              attributes: { description: "打开登录页" },
            },
            { type: "fill", selector: "#username", value: "${username}" },
          ],
          explanation: "先生成 2 个基础动作",
          confidence: 0.9,
        },
      });
    });

    await waitFor(() => {
      expect(handle.endpoint).toHaveBeenCalledTimes(1);
    });

    const [url, body] = handle.endpoint.mock.calls[0] as [string, { description: string }];
    expect(url).toBe("/rpa/ai/generate");
    expect(body).toEqual({ description: "打开登录页并输入用户名" });

    // 关键反证：全程**没有**推进虚拟时钟（旧实现要等 1500ms 的 setTimeout），
    // 因此只要出现动作步骤，就证明结果来自真实请求而非定时器。
    await waitFor(() => {
      expect(screen.getByText("打开登录页")).toBeTruthy();
    });
    expect(screen.getByText("先生成 2 个基础动作")).toBeTruthy();
    // 「确认使用」按钮必须变为可用（有动作才可确认）
    expect(screen.getByRole("button", { name: /确认使用/ })).not.toHaveProperty("disabled", true);
  });

  it("失败态：后端报错时展示可读错误并允许重试", async () => {
    const handle = await generateWith("某个任务", (h) => {
      h.endpoint.mockRejectedValue({ message: "API 返回错误 500: provider down" });
    });

    await waitFor(() => {
      expect(screen.getByTestId("ai-generate-error")).toBeTruthy();
    });
    const panel = screen.getByTestId("ai-generate-error");
    // 用 within(panel) 限定范围：antd message toast 里也有「AI 生成失败」文案
    expect(panel.textContent).toMatch(/AI 生成失败/);
    expect(panel.textContent).toMatch(/provider down/);
    expect(screen.getByRole("button", { name: /重试生成/ })).toBeTruthy();
  });

  it("空结果态：200 但 actions 为空时给出警告而非假装成功", async () => {
    const handle = await generateWith("模糊描述", (h) => {
      h.endpoint.mockResolvedValue({ code: 0, data: { actions: [], explanation: "" } });
    });

    await waitFor(() => {
      expect(screen.getByTestId("ai-generate-error")).toBeTruthy();
    });
    expect(screen.getByTestId("ai-generate-error").textContent).toMatch(/未返回可执行动作/);
  });

  it("超时态：请求 AbortError 时提示超时（而非笼统失败）", async () => {
    const handle = await generateWith("长时间任务", (h) => {
      h.endpoint.mockRejectedValue({ name: "AbortError", code: "ERR_CANCELED" });
    });

    await waitFor(() => {
      expect(screen.getByTestId("ai-generate-error")).toBeTruthy();
    });
    expect(screen.getByTestId("ai-generate-error").textContent).toMatch(/AI 生成超时/);
  });
});
