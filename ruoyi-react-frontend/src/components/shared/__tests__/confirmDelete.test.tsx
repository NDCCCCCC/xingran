/**
 * Phase 65 — confirmDelete helper 单测（UI-SPEC §4 行为契约）
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { Modal } from "antd";
import { confirmDelete } from "../confirmDelete";
import { handleApiError } from "@/utils/errorHandler";
import { setAppMessageInstance } from "@/utils/antdMessage";

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: vi.fn(),
}));

const messageMock = { success: vi.fn(), error: vi.fn() } as never;

describe("components/shared/confirmDelete", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setAppMessageInstance(messageMock);
  });

  const spyConfirm = () => vi.spyOn(Modal, "confirm").mockReturnValue({ destroy: vi.fn(), update: vi.fn() } as never);

  it("文案模板逐字匹配：title/content/okText/cancelText/okButtonProps.danger", () => {
    const confirmSpy = spyConfirm();
    confirmDelete({ target: "用户 张三", onDelete: async () => {} });
    expect(confirmSpy).toHaveBeenCalledTimes(1);
    const opts = confirmSpy.mock.calls[0][0];
    expect(opts.title).toBe("确认删除");
    expect(opts.content).toBe("将删除用户 张三，删除后不可恢复，是否继续？");
    expect(opts.okText).toBe("删除");
    expect(opts.cancelText).toBe("取消");
    expect(opts.okButtonProps).toEqual({ danger: true });
  });

  it("title 可覆盖默认标题（批量场景 确认批量删除）", () => {
    const confirmSpy = spyConfirm();
    confirmDelete({ target: "选中的 3 项", onDelete: async () => {}, title: "确认批量删除" });
    expect(confirmSpy.mock.calls[0][0].title).toBe("确认批量删除");
  });

  it("onOk resolve：成功触发 message.success(已删除)", async () => {
    spyConfirm();
    const onDelete = vi.fn().mockResolvedValue(undefined);
    confirmDelete({ target: "x", onDelete });
    const opts = vi.mocked(Modal.confirm).mock.calls[0][0];
    await opts.onOk();
    expect(onDelete).toHaveBeenCalledTimes(1);
    expect(messageMock.success).toHaveBeenCalledWith("已删除");
  });

  it("onOk reject：handleApiError 被调用（可见反馈）且 Promise reject（antd 保持 Modal 打开）", async () => {
    spyConfirm();
    const boom = new Error("网络错误");
    const onDelete = vi.fn().mockRejectedValue(boom);
    confirmDelete({ target: "x", onDelete });
    const opts = vi.mocked(Modal.confirm).mock.calls[0][0];
    await expect(opts.onOk()).rejects.toBe(boom);
    expect(handleApiError).toHaveBeenCalledWith(boom, "删除");
    expect(messageMock.success).not.toHaveBeenCalled();
  });

  it("extra 节点透传进 content 组合（数组 children）", () => {
    spyConfirm();
    const extra = <div data-testid="extra">同时删除子项</div>;
    confirmDelete({ target: "部门 市场部", onDelete: async () => {}, extra });
    const content = vi.mocked(Modal.confirm).mock.calls[0][0].content as unknown[];
    expect(Array.isArray(content)).toBe(true);
    expect(content[0]).toBe("将删除部门 市场部，删除后不可恢复，是否继续？");
    expect(content[1]).toBe(extra);
  });
});
