/**
 * Phase 88 Batch403 — components/shared/GlobalSearch 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import GlobalSearch from "../GlobalSearch";
import type { ReactElement, ReactNode } from "react";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <MemoryRouter initialEntries={["/"]}>{children}</MemoryRouter>;
}

describe("components/shared/GlobalSearch", () => {
  it("导出为函数组件", async () => {
    const mod = await import("../GlobalSearch");
    expect(typeof mod.default).toBe("function");
  });

  it("基础渲染不抛错", () => {
    expect(() => render(<GlobalSearch />, { wrapper })).not.toThrow();
  });

  it("自定义 placeholder 不抛错", () => {
    expect(() => render(<GlobalSearch placeholder="搜索…" />, { wrapper })).not.toThrow();
  });
});
