/**
 * Phase 156 / NOCODE-03：AI 不可用时降级到「录制」入口。
 *
 * 验收口径（任务书）：AI 可用 / 不可用 / 超时 三态各有明确且可操作的用户引导。
 *
 * 本文件锁定的核心行为（红绿变异点）：
 *   1. 后端报「不可用」（400 + 不可用文案）→ 面板显示「改用录制」主按钮，
 *      且**不显示**「重试生成」（重试无意义，只会让用户白等）。
 *   2. 点「改用录制」→ 调用 onFallbackToRecord 回调（上层据此打开录制流程）。
 *   3. 超时态 → 仍然显示「重试生成」（可重试），同时也给录制出路。
 *   4. 没有 onFallbackToRecord 时，按钮不能变成死按钮（必须给出可读指引）。
 *   5. 不可用态**不能**被误判成普通失败（这是体验分叉的关键）。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import { App as AntdApp } from "antd";
import { MemoryRouter } from "react-router-dom";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { createApiMock, resetApiMocks } from "@/test/utils/createApiMock";
import { AIScriptEditor, isAIDisabledResponse } from "../AIScriptEditor";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return (
    <MemoryRouter>
      <AntdApp>{children}</AntdApp>
    </MemoryRouter>
  );
}

async function renderAndGenerate(
  description: string,
  configure: (handle: ReturnType<typeof createApiMock>) => void,
  extraProps: { onFallbackToRecord?: () => void } = {}
) {
  const handle = createApiMock("/rpa/ai/generate");
  configure(handle);
  const onClose = vi.fn();
  render(
    <AIScriptEditor
      open
      onClose={onClose}
      onConfirm={vi.fn()}
      onFallbackToRecord={extraProps.onFallbackToRecord}
    />,
    { wrapper }
  );
  const textarea = await screen.findByRole("textbox");
  fireEvent.change(textarea, { target: { value: description } });
  fireEvent.click(screen.getByRole("button", { name: /生成脚本/ }));
  return { handle, onClose };
}

beforeEach(() => {
  resetApiMocks();
  vi.useFakeTimers({ shouldAdvanceTime: true });
});

afterEach(() => {
  vi.useRealTimers();
  resetApiMocks();
});

describe("isAIDisabledResponse 判定", () => {
  it("400 + 不可用文案 → 判定为不可用", () => {
    expect(
      isAIDisabledResponse({
        response: { status: 400, data: { message: "智能生成当前不可用（系统未开启该功能）" } },
      })
    ).toBe(true);
  });

  it("400 + 第 N 步（非法动作）→ 不是不可用，应走重试", () => {
    expect(
      isAIDisabledResponse({
        response: { status: 400, data: { message: "AI 返回的 2 个步骤中有 1 个无法执行：第 2 步" } },
      })
    ).toBe(false);
  });

  it("500 超时 → 不是不可用", () => {
    expect(isAIDisabledResponse({ response: { status: 500, data: { message: "脚本生成失败" } } })).toBe(
      false
    );
  });

  it("无响应信息时保守判为非不可用（宁可让用户重试，也不误报功能没开）", () => {
    expect(isAIDisabledResponse(undefined)).toBe(false);
    expect(isAIDisabledResponse({})).toBe(false);
    expect(isAIDisabledResponse(new Error("network down"))).toBe(false);
  });
});

describe("AIScriptEditor AI 不可用降级", () => {
  it("不可用态：显示「改用录制」，且不显示无意义的「重试生成」", async () => {
    await renderAndGenerate("打开报销系统并提交", (h) => {
      h.endpoint.mockRejectedValue({
        response: {
          status: 400,
          data: { message: "智能生成当前不可用（系统未开启该功能）。请改用「录制一段操作」的方式创建任务。" },
        },
      });
    });

    await waitFor(() => {
      expect(screen.getByTestId("ai-generate-error")).toBeTruthy();
    });
    const panel = screen.getByTestId("ai-generate-error");
    // 标题必须是「未开启」而不是笼统的「失败」
    expect(panel.textContent).toMatch(/智能生成未开启/);
    // 必须给出录制出路
    const fallbackBtn = screen.getByTestId("ai-fallback-record");
    expect(fallbackBtn.textContent).toMatch(/改用录制/);
    // 不可用时「重试生成」不应出现（重试必然再失败，是浪费用户时间）
    expect(screen.queryByRole("button", { name: /重试生成/ })).toBeNull();
  });

  it("不可用态：点「改用录制」调用 onFallbackToRecord", async () => {
    const onFallback = vi.fn();
    await renderAndGenerate(
      "打开报销系统并提交",
      (h) => {
        h.endpoint.mockRejectedValue({
          response: { status: 400, data: { message: "智能生成当前不可用" } },
        });
      },
      { onFallbackToRecord: onFallback }
    );

    const btn = await screen.findByTestId("ai-fallback-record");
    fireEvent.click(btn);
    await waitFor(() => {
      expect(onFallback).toHaveBeenCalledTimes(1);
    });
  });

  it("不可用态但未传 onFallbackToRecord：按钮不是死按钮，会关闭弹窗", async () => {
    const { onClose } = await renderAndGenerate("某个任务", (h) => {
      h.endpoint.mockRejectedValue({
        response: { status: 400, data: { message: "智能生成当前不可用" } },
      });
    });

    const btn = await screen.findByTestId("ai-fallback-record");
    fireEvent.click(btn);
    // 无回调时不能什么都不做 —— 必须走 onClose 把用户送回向导
    // （组件本身 open 由父级控制，故断言回调而非 DOM 消失）。
    await waitFor(() => {
      expect(onClose).toHaveBeenCalled();
    });
  });

  it("超时态：仍可重试，同时也能改录制（两条路都开）", async () => {
    await renderAndGenerate("长时间任务", (h) => {
      h.endpoint.mockRejectedValue({ name: "AbortError", code: "ERR_CANCELED" });
    });

    await waitFor(() => {
      expect(screen.getByTestId("ai-generate-error")).toBeTruthy();
    });
    const panel = screen.getByTestId("ai-generate-error");
    expect(panel.textContent).toMatch(/AI 生成超时/);
    // 超时是「可重试」的 → 重试按钮必须在
    expect(screen.getByRole("button", { name: /重试生成/ })).toBeTruthy();
    // 同时也要给录制出路（不是所有人都愿意等第二次）
    expect(screen.getByTestId("ai-fallback-record")).toBeTruthy();
  });

  it("普通失败态：给重试，也保留录制出路", async () => {
    await renderAndGenerate("某个任务", (h) => {
      h.endpoint.mockRejectedValue({ message: "API 返回错误 500: provider down" });
    });

    await waitFor(() => {
      expect(screen.getByTestId("ai-generate-error")).toBeTruthy();
    });
    expect(screen.getByRole("button", { name: /重试生成/ })).toBeTruthy();
    expect(screen.getByTestId("ai-fallback-record")).toBeTruthy();
  });

  it("不可用文案不含技术术语（面向一般员工）", async () => {
    await renderAndGenerate("某个任务", (h) => {
      h.endpoint.mockRejectedValue({
        response: { status: 400, data: { message: "智能生成当前不可用" } },
      });
    });

    await waitFor(() => {
      expect(screen.getByTestId("ai-generate-error")).toBeTruthy();
    });
    const text = screen.getByTestId("ai-generate-error").textContent ?? "";
    for (const banned of ["provider", "API Key", "enabled", "config", "baseURL"]) {
      expect(text.toLowerCase()).not.toContain(banned.toLowerCase());
    }
    // 必须说清「可以怎么办」
    expect(text).toMatch(/录制/);
  });
});
