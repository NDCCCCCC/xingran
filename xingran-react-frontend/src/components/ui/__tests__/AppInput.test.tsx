/**
 * Phase 65 — AppInput 单测（UI-SPEC §1 行为契约）
 */
import { describe, it, expect, vi } from "vitest";
import { render, fireEvent } from "@testing-library/react";
import { createRef } from "react";
import { App, type InputRef } from "antd";
import { AppInput } from "../AppInput";

const wrapper = ({ children }: { children: React.ReactNode }) => <App>{children}</App>;

describe("components/ui/AppInput", () => {
  it("零配置渲染：不传 props 时 allowClear 默认生效（input 可清除）", () => {
    const { container } = render(<AppInput />, { wrapper });
    const input = container.querySelector("input");
    expect(input).toBeTruthy();
    // antd allowClear 开启后输入值时渲染清除图标
    if (input) {
      fireEvent.change(input, { target: { value: "abc" } });
    }
    expect(container.querySelector(".ant-input-clear-icon")).toBeTruthy();
  });

  it("显式 allowClear={false} 逃生口生效", () => {
    const { container } = render(<AppInput allowClear={false} defaultValue="abc" />, { wrapper });
    expect(container.querySelector(".ant-input-clear-icon")).toBeNull();
  });

  it("value/onChange 受控透传：onChange 收到 e.target.value", () => {
    const onChange = vi.fn();
    const { container } = render(<AppInput value="init" onChange={onChange} />, { wrapper });
    const input = container.querySelector("input") as HTMLInputElement;
    expect(input.value).toBe("init");
    fireEvent.change(input, { target: { value: "hello" } });
    expect(onChange).toHaveBeenCalled();
    const evt = onChange.mock.calls[0][0] as React.ChangeEvent<HTMLInputElement>;
    expect(evt.target.value).toBe("hello");
  });

  it("placeholder 等 rest props 透传", () => {
    const { container } = render(<AppInput placeholder="请输入用户名" />, { wrapper });
    const input = container.querySelector("input") as HTMLInputElement;
    expect(input.placeholder).toBe("请输入用户名");
  });

  it("ref 透传（antd InputRef：focus 方法 + nativeElement 指向真实 input）", () => {
    const ref = createRef<InputRef>();
    render(<AppInput ref={ref} />, { wrapper });
    expect(ref.current).toBeTruthy();
    expect(typeof ref.current?.focus).toBe("function");
    expect(ref.current?.nativeElement).toBeTruthy();
  });
});
