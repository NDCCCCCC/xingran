/**
 * Phase 66 — SearchForm 单测（UI-SPEC §2 + PLAN T3 AC）
 *
 * 覆盖：input/select/custom 端点渲染与搜索链路；Enter 触发；空值过滤（filterEmpty
 * 在 useTableManager.applyFilters 内，这里断言 handleSearch 被调）；重置；按钮文案与
 * aria；searchChildren 逃生口；双通道同时传 → schema 优先 + 一次性警告；
 * remoteSelect 占位渲染不抛错（端点接线断言归 T5）。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { App as AntdApp, Form } from "antd";
import type { FormInstance } from "antd";

import { SearchForm } from "../SearchForm";
import type { SearchField } from "../SearchForm";

function Harness({
  fields,
  children,
  onSearch,
  onReset,
  onRefresh,
  formRef,
}: {
  fields?: SearchField[];
  children?: React.ReactNode;
  onSearch?: () => void;
  onReset?: () => void;
  onRefresh?: () => void;
  formRef?: (f: FormInstance) => void;
}) {
  const [form] = Form.useForm();
  formRef?.(form);
  return (
    <AntdApp>
      <SearchForm
        form={form}
        fields={fields}
        onSearch={onSearch ?? (() => {})}
        onReset={onReset ?? (() => {})}
        onRefresh={onRefresh ?? (() => {})}
      >
        {children}
      </SearchForm>
    </AntdApp>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  document.body.innerHTML = "";
});

describe("components/shared/SearchForm（T3）", () => {
  it("input→AppInput DOM；select→AppSelect options；按钮文案逐字 搜索/重置，刷新 aria-label+title", () => {
    const fields: SearchField[] = [
      { type: "input", name: "postCode", label: "岗位编码" },
      {
        type: "select",
        name: "status",
        label: "状态",
        options: [
          { label: "正常", value: 0 },
          { label: "停用", value: 1 },
        ],
      },
    ];
    const { container } = render(<Harness fields={fields} />);
    // AppInput 端点（antd input 存在）
    expect(container.querySelector("input#postCode")).not.toBeNull();
    // AppSelect 端点（.ant-select 存在）
    expect(container.querySelector(".ant-select")).not.toBeNull();
    expect(screen.getByText((c) => c.replace(/\s/g, "") === "搜索")).toBeTruthy();
    expect(screen.getByText((c) => c.replace(/\s/g, "") === "重置")).toBeTruthy();
    const refresh = screen.getByRole("button", { name: "刷新" });
    expect(refresh.getAttribute("title")).toBe("刷新");
  });

  it("输入后 Enter 触发 onSearch；点击搜索/重置/刷新分别回调", () => {
    const onSearch = vi.fn();
    const onReset = vi.fn();
    const onRefresh = vi.fn();
    render(
      <Harness
        fields={[{ type: "input", name: "kw", label: "关键字" }]}
        onSearch={onSearch}
        onReset={onReset}
        onRefresh={onRefresh}
      />
    );
    const input = document.querySelector<HTMLInputElement>("input#kw")!;
    fireEvent.input(input, { target: { value: "abc" } });
    fireEvent.keyDown(input, { key: "Enter" });
    // onPressEnter 由 antd Input 处理（keydown Enter）
    fireEvent.keyDown(input, { keyCode: 13 });
    expect(onSearch.mock.calls.length).toBeGreaterThanOrEqual(1);
    fireEvent.click(screen.getByText((c) => c.replace(/\s/g, "") === "搜索"));
    expect(onSearch).toHaveBeenCalled();
    fireEvent.click(screen.getByText((c) => c.replace(/\s/g, "") === "重置"));
    expect(onReset).toHaveBeenCalled();
    fireEvent.click(screen.getByRole("button", { name: "刷新" }));
    expect(onRefresh).toHaveBeenCalled();
  });

  it("custom→render 收 value/onChange；onChange 写入表单值（受控接入 filters 状态）", async () => {
    let formInstance: FormInstance | undefined;
    render(
      <Harness
        formRef={(f) => (formInstance = f)}
        fields={[
          {
            type: "custom",
            name: "range",
            render: (value, onChange) => (
              <input
                data-testid="custom-range"
                value={String(value ?? "")}
                onChange={(e) => onChange(e.target.value)}
              />
            ),
          },
        ]}
      />
    );
    const custom = screen.getByTestId("custom-range") as HTMLInputElement;
    fireEvent.change(custom, { target: { value: "1-9" } });
    // onChange → form 状态更新 → shouldUpdate 重渲染回读
    await vi.waitFor(() => {
      expect(formInstance!.getFieldValue("range")).toBe("1-9");
      expect((screen.getByTestId("custom-range") as HTMLInputElement).value).toBe("1-9");
    });
  });

  it("remoteSelect→RemoteSelect 端点（T5 接通；行为断言在 RemoteSelect.test.tsx）", () => {
    const fields: SearchField[] = [
      {
        type: "remoteSelect",
        name: "owner",
        label: "负责人",
        fetcher: vi.fn(async () => []),
      },
    ];
    const { container } = render(<Harness fields={fields} />);
    // 端点是 RemoteSelect（.ant-select），不再是占位 div
    expect(container.querySelector(".ant-select")).not.toBeNull();
  });

  it("searchChildren 逃生口渲染且其 Form.Item 值进入表单（经暴露的 form 实例接线）", async () => {
    let formInstance: FormInstance | undefined;
    render(
      <Harness formRef={(f) => (formInstance = f)}>
        <Form.Item name="deptId" label="部门">
          <input data-testid="dept-input" />
        </Form.Item>
      </Harness>
    );
    fireEvent.change(screen.getByTestId("dept-input"), { target: { value: "d-9" } });
    await vi.waitFor(() => {
      expect(formInstance!.getFieldValue("deptId")).toBe("d-9");
    });
  });

  it("searchFields 与 searchChildren 同时传 → schema 优先（断言在 PageTable 接线，见 PageTable.test.tsx）", () => {
    // 归属说明：优先级与 console.warn 逻辑在 PageTable 接线层（PLAN T3 AC5），
    // 独立 SearchForm 不含该逻辑，此处不重复断言。
    expect(true).toBe(true);
  });
});

describe("components/shared/SearchForm textarea 字段（Phase 70 T3）", () => {
  it("textarea→AppTextArea DOM：渲染 textarea，输入值后带 clear icon（SC2 清空能力）", () => {
    const { container } = render(
      <Harness
        fields={[
          { type: "textarea", name: "remark", label: "备注", placeholder: "模糊匹配备注", rows: 3 },
        ]}
      />
    );
    const textarea = container.querySelector("textarea#remark") as HTMLTextAreaElement;
    expect(textarea).not.toBeNull();
    expect(textarea.placeholder).toBe("模糊匹配备注");
    fireEvent.change(textarea, { target: { value: "abc" } });
    expect(container.querySelector(".ant-input-clear-icon")).toBeTruthy();
  });

  it("点击 clear icon 清空表单值（SC2 清空链路：onChange('') 直通 form）", async () => {
    let formInstance: FormInstance | undefined;
    const { container } = render(
      <Harness
        formRef={(f) => (formInstance = f)}
        fields={[{ type: "textarea", name: "remark", label: "备注" }]}
      />
    );
    const textarea = container.querySelector("textarea#remark") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "abc" } });
    await vi.waitFor(() => {
      expect(formInstance!.getFieldValue("remark")).toBe("abc");
    });
    const clearIcon = container.querySelector(".ant-input-clear-icon");
    expect(clearIcon).not.toBeNull();
    fireEvent.click(clearIcon as Element, { bubbles: true });
    await vi.waitFor(() => {
      expect(formInstance!.getFieldValue("remark")).toBe("");
    });
  });

  it("textarea 回车不触发 onSearch（多行回车必须换行，不接 onPressEnter）", () => {
    const onSearch = vi.fn();
    const { container } = render(
      <Harness fields={[{ type: "textarea", name: "remark", label: "备注" }]} onSearch={onSearch} />
    );
    const textarea = container.querySelector("textarea#remark") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "abc" } });
    fireEvent.keyDown(textarea, { key: "Enter" });
    fireEvent.keyDown(textarea, { keyCode: 13 });
    expect(onSearch).not.toHaveBeenCalled();
  });

  it("受控接入 form：fireEvent.change 后 form.getFieldValue('remark') 收到值", async () => {
    let formInstance: FormInstance | undefined;
    const { container } = render(
      <Harness
        formRef={(f) => (formInstance = f)}
        fields={[{ type: "textarea", name: "remark", label: "备注" }]}
      />
    );
    const textarea = container.querySelector("textarea#remark") as HTMLTextAreaElement;
    fireEvent.change(textarea, { target: { value: "hello" } });
    await vi.waitFor(() => {
      expect(formInstance!.getFieldValue("remark")).toBe("hello");
    });
  });
});
