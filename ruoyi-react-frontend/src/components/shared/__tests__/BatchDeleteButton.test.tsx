/**
 * Phase 88 Batch399 — components/shared/BatchDeleteButton 测试
 * Phase 68.1 T1 — target 透传 / Q4 actionContext / Q2（不加 aria-label）
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, fireEvent, screen } from "@testing-library/react";
import { App } from "antd";
import BatchDeleteButton from "../BatchDeleteButton";
import { setAppMessageInstance } from "@/utils/antdMessage";
import type { ReactElement } from "react";

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: vi.fn(),
}));

const messageMock = { success: vi.fn(), error: vi.fn() } as never;
const wrapper = ({ children }: { children: ReactElement }) => <App>{children}</App>;

const spyConfirm = async () => {
  const { Modal } = await import("antd");
  return vi.spyOn(Modal, "confirm").mockReturnValue({ destroy: vi.fn(), update: vi.fn() } as never);
};

describe("components/shared/BatchDeleteButton", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../BatchDeleteButton");
    expect(typeof mod.default).toBe("function");
  });

  it("disabled 时不抛错", () => {
    expect(() =>
      render(<BatchDeleteButton disabled selectedCount={0} />, { wrapper })
    ).not.toThrow();
  });
});

// Phase 68.1 T1 — target 透传与 Q2（不加 aria-label）
describe("components/shared/BatchDeleteButton（68.1 target/actionContext）", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setAppMessageInstance(messageMock);
  });

  it("传 target='资产' → 确认正文含「选中的 3 项资产」；actionContext='批量删除'", async () => {
    const confirmSpy = await spyConfirm();
    render(<BatchDeleteButton selectedCount={3} onConfirm={async () => {}} target="资产" />, {
      wrapper,
    });
    fireEvent.click(screen.getByRole("button"));
    const opts = confirmSpy.mock.calls[0][0];
    expect(opts.content).toBe("将删除选中的 3 项资产，删除后不可恢复，是否继续？");
    expect(opts.title).toBe("确认批量删除");
    expect(opts.okText).toBe("删除");
  });

  it("actionContext='批量删除' → 失败 handleApiError 收到「批量删除」（Q4）", async () => {
    const confirmSpy = await spyConfirm();
    const { handleApiError } = await import("@/utils/errorHandler");
    const boom = new Error("boom");
    render(
      <BatchDeleteButton
        selectedCount={3}
        target="资产"
        onConfirm={async () => {
          throw boom;
        }}
      />,
      { wrapper }
    );
    fireEvent.click(screen.getByRole("button"));
    const opts = confirmSpy.mock.calls[0][0];
    await expect(opts.onOk()).rejects.toBe(boom);
    expect(vi.mocked(handleApiError)).toHaveBeenLastCalledWith(boom, "批量删除");
  });

  it("不传 target → 正文「选中的 3 项」（缺省逐字不变）；Q2：按钮无 aria-label", async () => {
    const confirmSpy = await spyConfirm();
    const { container } = render(
      <BatchDeleteButton selectedCount={3} onConfirm={async () => {}} />,
      { wrapper }
    );
    fireEvent.click(screen.getByRole("button"));
    expect(confirmSpy.mock.calls[0][0].content).toBe(
      "将删除选中的 3 项，删除后不可恢复，是否继续？"
    );
    // Q2 红线：批量删除按钮（非 icon-only）不得补 aria-label
    expect(container.querySelector("button[aria-label]")).toBeNull();
  });
});
