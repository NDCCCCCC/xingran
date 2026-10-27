/**
 * Phase 70 — AppTextArea 单测（70-PLAN T2 行为契约；写法先例 AppInput.test.tsx）
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { App } from "antd";
import { AppTextArea } from "../AppTextArea";

const wrapper = ({ children }: { children: React.ReactNode }) => <App>{children}</App>;

describe("components/ui/AppTextArea", () => {
  it("零配置渲染：渲染出 textarea 且输入值后 allowClear 默认生效（clear icon 出现）", () => {
    const { container } = render(<AppTextArea />, { wrapper });
    const textarea = container.querySelector("textarea");
    expect(textarea).toBeTruthy();
    if (textarea) {
      fireEvent.change(textarea, { target: { value: "abc" } });
    }
    expect(container.querySelector(".ant-input-clear-icon")).toBeTruthy();
  });

  it("显式 allowClear={false} 逃生口生效（clear icon 为 null）", () => {
    const { container } = render(<AppTextArea allowClear={false} defaultValue="abc" />, { wrapper });
    expect(container.querySelector(".ant-input-clear-icon")).toBeNull();
  });

  it("value/onChange 受控透传：onChange 收到 e.target.value", () => {
    const onChange = vi.fn();
    const { container } = render(<AppTextArea value="init" onChange={onChange} />, { wrapper });
    const textarea = container.querySelector("textarea") as HTMLTextAreaElement;
    expect(textarea.value).toBe("init");
    fireEvent.change(textarea, { target: { value: "hello" } });
    expect(onChange).toHaveBeenCalled();
    const evt = onChange.mock.calls[0][0] as React.ChangeEvent<HTMLTextAreaElement>;
    expect(evt.target.value).toBe("hello");
  });

  it("placeholder / rows 等 rest props 透传到 DOM", () => {
    const { container } = render(<AppTextArea placeholder="请输入备注" rows={4} />, { wrapper });
    const textarea = container.querySelector("textarea") as HTMLTextAreaElement;
    expect(textarea.placeholder).toBe("请输入备注");
    expect(textarea.rows).toBe(4);
  });

  it("autoSize 不下沉默认（显式 rows 生效）；显式传入 autoSize 后生效（overflowX/Y hidden）", () => {
    // 默认：不传 autoSize，显式 rows 渲染为静态行高
    const { container, rerender } = render(<AppTextArea rows={3} />, { wrapper });
    const textarea = container.querySelector("textarea") as HTMLTextAreaElement;
    expect(textarea.rows).toBe(3);
    // 显式传入 autoSize：antd ResizableTextArea 挂 overflowX/Y=hidden（autoSize 生效标志）
    rerender(<AppTextArea autoSize={{ minRows: 2, maxRows: 6 }} />);
    const el = container.querySelector("textarea") as HTMLTextAreaElement;
    // jsdom 的 cssstyle 不把 overflow-y 解析进 CSSStyleDeclaration，断言 inline style 属性串
    expect(el.getAttribute("style")).toMatch(/overflow-y:\s*hidden/);
  });
});