/**
 * Phase 66 — FormModal 单测（UI-SPEC §3 + PLAN T4 AC）
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, cleanup } from "@testing-library/react";
import { App as AntdApp, Form, Modal } from "antd";
import type { FormInstance } from "antd";

vi.mock("@/utils/errorHandler", () => ({
  handleApiError: vi.fn(),
  handleSuccess: vi.fn(),
  // 与真实现语义对齐：antd validateFields reject 携带 errorFields
  isFormValidationError: (e: unknown) =>
    Boolean(e && typeof e === "object" && "errorFields" in (e as Record<string, unknown>)),
}));

import { FormModal, useFormModalForm } from "../FormModal";
import { handleApiError } from "@/utils/errorHandler";

function Harness({
  open,
  onFinish,
  onCancel,
  initialValues,
  dirtyCheck = true,
  formRef,
  okText,
}: {
  open: boolean;
  onFinish: (v: Record<string, unknown>) => Promise<void>;
  onCancel: () => void;
  initialValues?: Record<string, unknown> | null;
  dirtyCheck?: boolean;
  formRef?: (f: FormInstance) => void;
  okText?: string;
}) {
  const form = useFormModalForm();
  formRef?.(form);
  return (
    <AntdApp>
      <FormModal
        open={open}
        title="岗位管理"
        form={form}
        onFinish={onFinish}
        onCancel={onCancel}
        initialValues={initialValues}
        dirtyCheck={dirtyCheck}
        okText={okText}
      >
        <Form.Item name="postName" label="岗位名称" rules={[{ required: true, message: "请输入岗位名称" }]}>
          <input data-testid="name-input" />
        </Form.Item>
      </FormModal>
    </AntdApp>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  document.body.innerHTML = "";
});
afterEach(() => {
  cleanup();
  document.body.innerHTML = "";
});

describe("components/shared/FormModal（T4）", () => {
  it("open=true：initialValues 回填（resetFields+setFieldsValue）", async () => {
    render(
      <Harness open onFinish={async () => {}} onCancel={() => {}} initialValues={{ postName: "总经理" }} />
    );
    await waitFor(() => {
      expect((screen.getByTestId("name-input") as HTMLInputElement).value).toBe("总经理");
    });
  });

  it("填写后点保存：onFinish 收 validateFields 值", async () => {
    const onFinish = vi.fn(async () => {});
    render(<Harness open onFinish={onFinish} onCancel={() => {}} />);
    fireEvent.input(screen.getByTestId("name-input"), { target: { value: "CTO" } });
    fireEvent.click(screen.getByText((c) => c.replace(/\s/g, "") === "保存"));
    await waitFor(() => {
      expect(onFinish).toHaveBeenCalledWith(expect.objectContaining({ postName: "CTO" }));
    });
  });

  it("onFinish 抛业务错：handleApiError(error, 保存) + 弹窗保持打开", async () => {
    const onFinish = vi.fn(async () => {
      throw new Error("网络错误");
    });
    render(<Harness open onFinish={onFinish} onCancel={() => {}} />);
    fireEvent.input(screen.getByTestId("name-input"), { target: { value: "CTO" } });
    fireEvent.click(screen.getByText((c) => c.replace(/\s/g, "") === "保存"));
    await waitFor(() => {
      expect(handleApiError).toHaveBeenCalledWith(expect.any(Error), "保存");
    });
    // 弹窗保持打开
    expect(document.querySelector(".ant-modal")).not.toBeNull();
  });

  it("表单校验失败：isFormValidationError 分流 → handleApiError 不被调", async () => {
    const onFinish = vi.fn(async () => {});
    render(<Harness open onFinish={onFinish} onCancel={() => {}} />);
    // 不填必填字段直接保存
    fireEvent.click(screen.getByText((c) => c.replace(/\s/g, "") === "保存"));
    await waitFor(() => {
      expect(screen.getByText("请输入岗位名称")).toBeTruthy();
    });
    expect(handleApiError).not.toHaveBeenCalled();
    expect(onFinish).not.toHaveBeenCalled();
  });

  it("dirtyCheck：touched 后点取消 → Modal.confirm 文案逐字 → onOk 后 onCancel", async () => {
    const confirmSpy = vi.spyOn(Modal, "confirm");
    const onCancel = vi.fn();
    render(<Harness open onFinish={async () => {}} onCancel={onCancel} />);
    fireEvent.input(screen.getByTestId("name-input"), { target: { value: "改过" } });
    fireEvent.click(screen.getByText("取 消"));
    await waitFor(() => {
      expect(confirmSpy).toHaveBeenCalledTimes(1);
    });
    const opts = confirmSpy.mock.calls[0][0];
    expect(opts.title).toBe("更改尚未保存");
    expect(opts.content).toBe("当前修改尚未保存，关闭后将丢失，是否确认关闭？");
    expect(opts.okText).toBe("放弃更改");
    expect(opts.cancelText).toBe("继续编辑");
    // ok 非 danger：okButtonProps 不含 danger:true
    expect((opts.okButtonProps ?? {}).danger).toBeUndefined();
    // 确认放弃 → onCancel
    await opts.onOk?.();
    expect(onCancel).toHaveBeenCalledTimes(1);
    confirmSpy.mockRestore();
  });

  it("dirtyCheck=false：取消直接 onCancel，不出确认", () => {
    const confirmSpy = vi.spyOn(Modal, "confirm");
    const onCancel = vi.fn();
    render(<Harness open onFinish={async () => {}} onCancel={onCancel} dirtyCheck={false} />);
    fireEvent.input(screen.getByTestId("name-input"), { target: { value: "改过" } });
    fireEvent.click(screen.getByText("取 消"));
    expect(confirmSpy).not.toHaveBeenCalled();
    expect(onCancel).toHaveBeenCalledTimes(1);
    confirmSpy.mockRestore();
  });

  it("未 touched 时取消：不弹确认直接 onCancel", () => {
    const confirmSpy = vi.spyOn(Modal, "confirm");
    const onCancel = vi.fn();
    render(<Harness open onFinish={async () => {}} onCancel={onCancel} />);
    fireEvent.click(screen.getByText("取 消"));
    expect(confirmSpy).not.toHaveBeenCalled();
    expect(onCancel).toHaveBeenCalledTimes(1);
    confirmSpy.mockRestore();
  });
});
