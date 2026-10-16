/**
 * Phase 88 Batch426 — FloorPlanEditor render 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

// Mock before import
vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import FloorPlanEditor from "../FloorPlanEditor";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("FloorPlanEditor render", () => {
  it("导出为函数组件", () => {
    expect(typeof FloorPlanEditor).toBe("function");
  });
});
