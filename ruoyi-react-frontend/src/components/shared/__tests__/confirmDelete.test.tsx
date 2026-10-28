/**
 * Phase 65 — 危险删除唯一入口单测（命令式 confirmDelete + 声明式 ConfirmDelete）
 *
 * 偏差注记：PLAN exact file 期望 `__tests__/ConfirmDelete.test.tsx` 与
 * `__tests__/confirmDelete.test.tsx` 并存，但 Windows 文件系统大小写不敏感，
 * 两个文件无法共存；因此合并为单一测试文件（两套 describe 各自保留）。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { Modal } from "antd";
import { render, screen, fireEvent } from "@testing-library/react";
import { confirmDelete } from "../confirmDelete";
import { ConfirmDelete } from "../ConfirmDelete.tsx"; // 显式扩展名：Windows 大小写不敏感解析会命中 confirmDelete.ts
import { handleApiError } from "@/utils/errorHandler";
import { setAppMessageInstance } from "@/utils/antdMessage";

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: vi.fn(),
}));

const messageMock = { success: vi.fn(), error: vi.fn() } as never;

const spyConfirm = () => vi.spyOn(Modal, "confirm").mockReturnValue({ destroy: vi.fn(), update: vi.fn() } as never);

describe("components/shared/confirmDelete（命令式）", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setAppMessageInstance(messageMock);
  });

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

// Phase 68.1 T1 — 契约扩展特性（Q1/Q4 + 焦点落点 + aria-busy + 提交中禁取消）
describe("components/shared/confirmDelete（68.1 扩展）", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setAppMessageInstance(messageMock);
  });

  it("默认形态向后兼容：mask.closable=false / keyboard=false / focusable.autoFocusButton='ok'", () => {
    const confirmSpy = spyConfirm();
    confirmDelete({ target: "x", onDelete: async () => {} });
    const opts = confirmSpy.mock.calls[0][0];
    // Phase 88-04（VISFIX-04）：antd 6 弃用 maskClosable → 新形态 mask.closable，
    // 此处断言传给 antd Modal.confirm 的**新内部实现**。
    expect(opts.mask).toEqual({ closable: false });
    expect(opts.keyboard).toBe(false);
    expect(opts.focusable).toEqual({ autoFocusButton: "ok" });
  });

  it("actionLabel='清空' → okText「清空」+ 成功「已清空」（不串成已删除）", async () => {
    spyConfirm();
    confirmDelete({ target: "缓存", onDelete: async () => {}, actionLabel: "清空" });
    const opts = vi.mocked(Modal.confirm).mock.calls[0][0];
    expect(opts.okText).toBe("清空");
    await opts.onOk();
    expect(messageMock.success).toHaveBeenCalledWith("已清空");
    expect(messageMock.success).not.toHaveBeenCalledWith("已删除");
  });

  it("successText 显式覆盖派生成功文案", async () => {
    spyConfirm();
    confirmDelete({ target: "x", onDelete: async () => {}, successText: "已恢复" });
    const opts = vi.mocked(Modal.confirm).mock.calls[0][0];
    await opts.onOk();
    expect(messageMock.success).toHaveBeenCalledWith("已恢复");
  });

  it("actionContext='批量删除' → 失败 handleApiError 收到「批量删除」", async () => {
    spyConfirm();
    const boom = new Error("boom");
    confirmDelete({ target: "x", onDelete: async () => { throw boom; }, actionContext: "批量删除" });
    const opts = vi.mocked(Modal.confirm).mock.calls[0][0];
    await expect(opts.onOk()).rejects.toBe(boom);
    expect(handleApiError).toHaveBeenCalledWith(boom, "批量删除");
  });

  it("contentText 覆盖正文且 {target} 占位符被替换", () => {
    spyConfirm();
    confirmDelete({ target: "缓存数据", onDelete: async () => {}, contentText: "将清空{target}，操作不可恢复，是否继续？" });
    const opts = vi.mocked(Modal.confirm).mock.calls[0][0];
    expect(opts.content).toBe("将清空缓存数据，操作不可恢复，是否继续？");
  });

  it("提交中：instance.update 被调用（cancelButtonProps.disabled=true + okButtonProps aria-busy=true），finally 恢复", async () => {
    const update = vi.fn();
    vi.spyOn(Modal, "confirm").mockReturnValue({ destroy: vi.fn(), update } as never);
    let resolveDelete: () => void = () => {};
    const onDelete = vi.fn().mockImplementation(() => new Promise<void>((r) => { resolveDelete = r; }));
    confirmDelete({ target: "x", onDelete });
    const opts = vi.mocked(Modal.confirm).mock.calls[0][0];
    const pending = opts.onOk();
    expect(update).toHaveBeenCalledWith({
      cancelButtonProps: { disabled: true },
      okButtonProps: { danger: true, "aria-busy": true },
    });
    resolveDelete();
    await pending;
    expect(update).toHaveBeenLastCalledWith({
      cancelButtonProps: { disabled: false },
      okButtonProps: { danger: true },
    });
  });
});

// Phase 68.1 T1 Q1 — 声明式组件可访问名与弹窗标题解耦
describe("components/shared/ConfirmDelete（68.1 Q1 解耦）", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setAppMessageInstance(messageMock);
  });

  it("aria-label 等于 删除{target}；传 title='确认删除' 时 aria-label 仍为「删除X」（title 不流入可访问名）", () => {
    render(<ConfirmDelete target="用户 张三" onDelete={async () => {}} title="确认删除" />);
    const btn = screen.getByRole("button");
    expect(btn.getAttribute("aria-label")).toBe("删除用户 张三");
    expect(btn.getAttribute("aria-label")).not.toContain("确认删除");
  });

  it("title 仍出现在确认弹窗标题", () => {
    const confirmSpy = spyConfirm();
    render(<ConfirmDelete target="x" onDelete={async () => {}} title="确认删除" />);
    fireEvent.click(screen.getByRole("button"));
    expect(confirmSpy.mock.calls[0][0].title).toBe("确认删除");
  });
});

describe("components/shared/ConfirmDelete（声明式）", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    setAppMessageInstance(messageMock);
  });

  it("点击按钮 → confirmDelete 路径触发一次 Modal.confirm，文案含 target 与不可恢复警示", () => {
    const confirmSpy = spyConfirm();
    render(<ConfirmDelete target="用户 张三" onDelete={async () => {}} />);
    fireEvent.click(screen.getByRole("button"));
    expect(confirmSpy).toHaveBeenCalledTimes(1);
    const opts = confirmSpy.mock.calls[0][0];
    expect(opts.title).toBe("确认删除");
    expect(opts.content).toBe("将删除用户 张三，删除后不可恢复，是否继续？");
    expect(opts.okText).toBe("删除");
    expect(opts.okButtonProps).toEqual({ danger: true });
  });

  it("disabled=true 或 onDelete 未传 → 按钮 disabled；children 替换默认内容", () => {
    render(<ConfirmDelete target="x" onDelete={async () => {}} disabled>移除</ConfirmDelete>);
    expect(screen.getByRole("button", { name: "删除x" })).toBeDisabled();
    expect(screen.getByText("移除")).toBeInTheDocument();
    fireEvent.click(screen.getByRole("button"));
    expect(vi.mocked(Modal.confirm)).not.toHaveBeenCalled();

    render(<ConfirmDelete target="y">移除</ConfirmDelete>);
    expect(screen.getAllByRole("button", { name: "删除y" })[0]).toBeDisabled();
  });

  it("视觉惯例：danger + action-btn-link-danger + DeleteOutlined 图标", () => {
    render(<ConfirmDelete target="x" onDelete={async () => {}} />);
    const btn = screen.getByRole("button", { name: "删除x" });
    expect(btn.className).toContain("action-btn-link-danger");
    expect(btn.querySelector(".anticon-delete")).not.toBeNull();
  });

  it("删除失败 → Modal 保持打开可重试（helper rethrow 语义经组件路径保持）", async () => {
    spyConfirm();
    const onDelete = vi.fn().mockRejectedValueOnce(new Error("网络错误")).mockResolvedValue(undefined);
    render(<ConfirmDelete target="z" onDelete={onDelete} />);
    fireEvent.click(screen.getByRole("button"));
    const opts = vi.mocked(Modal.confirm).mock.calls[0][0];
    await expect(opts.onOk()).rejects.toThrow("网络错误");
    // 失败后按钮仍可点击重试 → 再次触发 confirmDelete 路径
    fireEvent.click(screen.getByRole("button"));
    expect(vi.mocked(Modal.confirm)).toHaveBeenCalledTimes(2);
    // 第二次确认的 onOk 再次执行 onDelete
    const opts2 = vi.mocked(Modal.confirm).mock.calls[1][0];
    await opts2.onOk();
    expect(onDelete).toHaveBeenCalledTimes(2);
  });
});
