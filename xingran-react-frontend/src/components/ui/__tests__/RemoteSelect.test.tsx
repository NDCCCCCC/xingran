/**
 * Phase 66 — RemoteSelect 单测（UI-SPEC §4 + PLAN T5 AC）
 *
 * 覆盖：防抖（默认 300ms 内多次输入只发一次）；竞态守卫（慢请求后发先回不覆盖）；
 * 失败 handleApiError(error, "搜索选项") + 保留上次成功选项；keyword 清空回
 * initialOptions 不发请求；无跨实例缓存（initialOptions 不被第二个实例污染）；
 * SearchForm remoteSelect 端点接线（占位测试同步升级为真实断言）。
 */
import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { render, screen, fireEvent, waitFor, cleanup } from "@testing-library/react";
import { App as AntdApp } from "antd";

vi.mock("@/utils/errorHandler", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/utils/errorHandler")>();
  return { ...actual, handleApiError: vi.fn() };
});

import { RemoteSelect } from "../RemoteSelect";
import { SearchForm } from "../../shared/SearchForm";
import { handleApiError } from "@/utils/errorHandler";
import { Form } from "antd";
import type { FormInstance } from "antd";

function openDropdown(container: HTMLElement) {
  fireEvent.mouseDown(container.querySelector(".ant-select")!);
}

function typeKeyword(value: string) {
  const input = document.querySelector<HTMLInputElement>("input.ant-select-input") ?? document.querySelector<HTMLInputElement>(".ant-select-selection-search-input");
  fireEvent.change(input, { target: { value } });
}

beforeEach(() => {
  vi.clearAllMocks();
  document.body.innerHTML = "";
});
afterEach(() => {
  cleanup();
  document.body.innerHTML = "";
});

function Harness(props: React.ComponentProps<typeof RemoteSelect>) {
  return (
    <AntdApp>
      <RemoteSelect {...props} />
    </AntdApp>
  );
}

describe("components/ui/RemoteSelect（T5）", () => {
  it("initialOptions 无关键字时展示；搜索防抖 300ms 内多次输入只发一次", async () => {
    vi.useFakeTimers();
    const fetcher = vi.fn(async () => [{ label: "张三", value: "u1" }]);
    const { container } = render(<Harness fetcher={fetcher} initialOptions={[{ label: "初始", value: "i0" }]} />);
    openDropdown(container);
    expect(document.body.textContent).toContain("初始");
    typeKeyword("张");
    typeKeyword("张三");
    typeKeyword("张三丰");
    expect(fetcher).not.toHaveBeenCalled();
    vi.advanceTimersByTime(300);
    await vi.waitFor(() => {
      expect(fetcher).toHaveBeenCalledTimes(1);
      expect(fetcher).toHaveBeenCalledWith("张三丰");
    });
    vi.useRealTimers();
  });

  it("竞态守卫：先发慢请求、后发快请求，慢结果不覆盖（仅最后请求可写入）", async () => {
    let resolveSlow!: (v: { label: string; value: string }[]) => void;
    const fetcher = vi
      .fn<(kw: string) => Promise<{ label: string; value: string }[]>>()
      .mockImplementationOnce(
        () =>
          new Promise((res) => {
            resolveSlow = res;
          })
      )
      .mockResolvedValueOnce([{ label: "快结果", value: "fast" }]);
    const { container } = render(<Harness fetcher={fetcher} debounceMs={0} />);
    openDropdown(container);
    typeKeyword("a");
    await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(1));
    typeKeyword("ab");
    await waitFor(() => expect(fetcher).toHaveBeenCalledTimes(2));
    await waitFor(() => {
      expect(screen.getAllByText("快结果").length).toBeGreaterThan(0);
    });
    // 慢请求此时才回来：不得覆盖
    resolveSlow([{ label: "慢结果", value: "slow" }]);
    await new Promise((r) => setTimeout(r, 10));
    expect(screen.queryByText("慢结果")).toBeNull();
    expect(screen.getAllByText("快结果").length).toBeGreaterThan(0);
  });

  it("失败：handleApiError(error, 搜索选项) + 保留上次成功选项", async () => {
    const fetcher = vi
      .fn<(kw: string) => Promise<{ label: string; value: string }[]>>()
      .mockResolvedValueOnce([{ label: "成功批", value: "ok" }])
      .mockRejectedValueOnce(new Error("网络失败"));
    const { container } = render(<Harness fetcher={fetcher} debounceMs={0} />);
    openDropdown(container);
    typeKeyword("a");
    await waitFor(() => {
      expect(screen.getAllByText("成功批").length).toBeGreaterThan(0);
    });
    typeKeyword("b");
    await waitFor(() => {
      expect(handleApiError).toHaveBeenCalledWith(expect.any(Error), "搜索选项");
    });
    // 上次成功选项仍在
    expect(screen.getAllByText("成功批").length).toBeGreaterThan(0);
  });

  it("keyword 清空：回 initialOptions 且不发请求", async () => {
    const fetcher = vi.fn(async () => [{ label: "搜索批", value: "s" }]);
    const { container } = render(<Harness fetcher={fetcher} debounceMs={0} initialOptions={[{ label: "初始", value: "i" }]} />);
    openDropdown(container);
    typeKeyword("a");
    await waitFor(() => {
      expect(screen.getAllByText("搜索批").length).toBeGreaterThan(0);
    });
    typeKeyword("");
    await waitFor(() => {
      expect(screen.getAllByText("初始").length).toBeGreaterThan(0);
    });
    expect(screen.queryByText("搜索批")).toBeNull();
    expect(fetcher).toHaveBeenCalledTimes(1);
  });

  it("无跨实例缓存：第二实例不继承第一实例搜索结果", async () => {
    const fetcher = vi.fn(async () => [{ label: "实例1结果", value: "1" }]);
    const first = render(<Harness fetcher={fetcher} debounceMs={0} />);
    openDropdown(first.container);
    typeKeyword("x");
    await waitFor(() => {
      expect(screen.getAllByText("实例1结果").length).toBeGreaterThan(0);
    });
    first.unmount();
    const second = render(
      <AntdApp>
        <RemoteSelect fetcher={fetcher} debounceMs={0} />
      </AntdApp>
    );
    openDropdown(second.container);
    // 未搜索时无上实例残留
    expect(screen.queryByText("实例1结果")).toBeNull();
    expect(fetcher).toHaveBeenCalledTimes(1);
  });
});

describe("SearchForm remoteSelect 端点接线（T5 接通）", () => {
  it("渲染 RemoteSelect 端点并可搜索回填选项", async () => {
    const fetcher = vi.fn(async () => [{ label: "负责人甲", value: "u9" }]);
    // 测试夹具：form 实例冒泡到外部断言（React 19 下 Form.useForm 只能在组件
    // 内调用；react-compiler 禁止 render 内写外部变量，此处为测试专用逃生口）
    let formRef: FormInstance | null = null;
    const onSearch = vi.fn();
    function H() {
      const [form] = Form.useForm();
      // eslint-disable-next-line react-hooks/globals -- 测试夹具 form 实例冒泡
      formRef = form;
      return (
        <SearchForm
          form={form}
          fields={[{ type: "remoteSelect", name: "owner", label: "负责人", fetcher, debounceMs: 0 }]}
          onSearch={onSearch}
          onReset={() => {}}
          onRefresh={() => {}}
        />
      );
    }
    const { container } = render(
      <AntdApp>
        <H />
      </AntdApp>
    );
    openDropdown(container);
    typeKeyword("甲");
    await waitFor(() => {
      expect(fetcher).toHaveBeenCalledWith("甲");
    });
    await waitFor(() => {
      expect(screen.getAllByText("负责人甲").length).toBeGreaterThan(0);
    });
    // 选中 → 进入表单值
    fireEvent.click(screen.getAllByText("负责人甲")[0]);
    await waitFor(() => {
      expect(formRef!.getFieldValue("owner")).toBe("u9");
    });
    void container;
  });
});
