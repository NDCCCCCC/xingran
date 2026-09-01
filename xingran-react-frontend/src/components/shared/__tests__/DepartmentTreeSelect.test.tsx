/**
 * Phase 88 Batch403 — components/shared/DepartmentTreeSelect 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import DepartmentTreeSelect from "../DepartmentTreeSelect";
import type { ReactElement, ReactNode } from "react";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("components/shared/DepartmentTreeSelect", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../DepartmentTreeSelect");
    expect(typeof mod.default).toBe("function");
  });

  it("基础渲染不抛错", () => {
    expect(() =>
      render(<DepartmentTreeSelect value={undefined} onChange={vi.fn()} />, {
        wrapper,
      })
    ).not.toThrow();
  });

  it("自定义 placeholder 不抛错", () => {
    expect(() =>
      render(
        <DepartmentTreeSelect value={undefined} onChange={vi.fn()} placeholder="选择部门" />,
        { wrapper }
      )
    ).not.toThrow();
  });
});