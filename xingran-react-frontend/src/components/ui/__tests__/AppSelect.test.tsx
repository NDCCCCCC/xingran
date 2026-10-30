/**
 * Phase 65 — AppSelect 单测（UI-SPEC §2 行为契约）
 *
 * antd v6 语义：Select 容器类 `.ant-select`，showSearch 开启时挂 `ant-select-show-search`
 * 类，搜索输入框类为 `.ant-select-input`；下拉选项渲染在 body portal 下
 * （`.ant-select-item-option-content`）。mouseDown 整个 `.ant-select` 打开下拉。
 */
import { describe, it, expect } from "vitest";
import { render, fireEvent, act, waitFor } from "@testing-library/react";
import { App } from "antd";
import { AppSelect } from "../AppSelect";

const wrapper = ({ children }: { children: React.ReactNode }) => <App>{children}</App>;

const makeOptions = (n: number) =>
  Array.from({ length: n }, (_, i) => ({ label: `选项${i + 1}`, value: i + 1 }));

const openDropdown = async (container: HTMLElement) => {
  const selectEl = container.querySelector(".ant-select") as HTMLElement;
  await act(async () => {
    fireEvent.mouseDown(selectEl);
  });
};

const hasShowSearch = (container: HTMLElement) =>
  (container.querySelector(".ant-select") as HTMLElement)?.classList.contains(
    "ant-select-show-search"
  );

const optionTexts = () =>
  Array.from(document.querySelectorAll(".ant-select-item-option-content")).map(
    (el) => el.textContent
  );

describe("components/ui/AppSelect", () => {
  it("options.length < 8 时 showSearch 自动为 false", async () => {
    const { container } = render(<AppSelect options={makeOptions(3)} />, { wrapper });
    await openDropdown(container);
    expect(hasShowSearch(container)).toBe(false);
  });

  it("options.length >= 8 时 showSearch 自动为 true 且 optionFilterProp 默认 label", async () => {
    const { container } = render(<AppSelect options={makeOptions(10)} />, { wrapper });
    await openDropdown(container);
    expect(hasShowSearch(container)).toBe(true);
    const searchInput = container.querySelector(
      "input.ant-select-input"
    ) as HTMLInputElement | null;
    expect(searchInput).toBeTruthy();
    // 按 label 过滤：输入 "选项9" 应只剩 选项9（不会命中 label 含 "选项1" 的其余项）
    await act(async () => {
      fireEvent.change(searchInput as HTMLInputElement, { target: { value: "选项9" } });
    });
    await waitFor(() => {
      expect(optionTexts()).toContain("选项9");
      expect(optionTexts()).not.toContain("选项1");
    });
  });

  it("显式 showSearch={false} 覆盖自动值", async () => {
    const { container } = render(<AppSelect options={makeOptions(10)} showSearch={false} />, {
      wrapper,
    });
    await openDropdown(container);
    expect(hasShowSearch(container)).toBe(false);
  });

  it("显式 optionFilterProp 覆盖默认 label（按 value 过滤）", async () => {
    const { container } = render(<AppSelect options={makeOptions(10)} optionFilterProp="value" />, {
      wrapper,
    });
    await openDropdown(container);
    const searchInput = container.querySelector(
      "input.ant-select-input"
    ) as HTMLInputElement | null;
    await act(async () => {
      fireEvent.change(searchInput as HTMLInputElement, { target: { value: "10" } });
    });
    await waitFor(() => {
      expect(optionTexts()).toEqual(["选项10"]);
    });
  });

  it("默认 allowClear：单选有值后渲染清除图标", async () => {
    const { container } = render(<AppSelect options={makeOptions(3)} defaultValue={1} />, {
      wrapper,
    });
    await openDropdown(container);
    await act(async () => {
      fireEvent.click(document.querySelector(".ant-select-item-option") as HTMLElement);
    });
    expect(container.querySelector(".ant-select-clear")).toBeTruthy();
  });

  it("显式 allowClear={false}：有值也无清除图标", async () => {
    const { container } = render(
      <AppSelect options={makeOptions(3)} defaultValue={1} allowClear={false} />,
      { wrapper }
    );
    await openDropdown(container);
    await act(async () => {
      fireEvent.click(document.querySelector(".ant-select-item-option") as HTMLElement);
    });
    expect(container.querySelector(".ant-select-clear")).toBeNull();
  });

  it("children 形态（Option 子组件）透传渲染不抛错且下拉可开", async () => {
    const { container } = render(
      <AppSelect>
        <AppSelect.Option value={0}>正常</AppSelect.Option>
        <AppSelect.Option value={1}>停用</AppSelect.Option>
      </AppSelect>,
      { wrapper }
    );
    await openDropdown(container);
    expect(
      (container.querySelector(".ant-select") as HTMLElement).classList.contains("ant-select-open")
    ).toBe(true);
  });
});
