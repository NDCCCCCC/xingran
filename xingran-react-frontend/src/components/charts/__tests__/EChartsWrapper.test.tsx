/**
 * Phase 88 Batch320 — components/charts/EChartsWrapper 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("echarts-for-react", () => ({
  default: (props: any) => (
    <div data-testid="echarts-mock" data-option={JSON.stringify(props.option)} />
  ),
}));

import EChartsWrapper from "../EChartsWrapper";

describe("components/charts/EChartsWrapper", () => {
  it("渲染 → 显示 echarts mock", async () => {
    render(<EChartsWrapper option={{ xAxis: { type: "value" } }} />);
    await waitFor(() => {
      expect(screen.getByTestId("echarts-mock")).toBeInTheDocument();
    });
  });

  it("传递 option 到内部组件", async () => {
    const option = { series: [{ type: "line", data: [1, 2, 3] }] };
    render(<EChartsWrapper option={option} />);
    await waitFor(() => {
      const el = screen.getByTestId("echarts-mock");
      const data = JSON.parse(el.getAttribute("data-option") || "{}");
      expect(data.series[0].data).toEqual([1, 2, 3]);
    });
  });

  it("传递 style 属性", async () => {
    render(<EChartsWrapper option={{}} style={{ height: 300 }} />);
    await waitFor(() => {
      expect(screen.getByTestId("echarts-mock")).toBeInTheDocument();
    });
  });

  it("displayName 正确", () => {
    expect(EChartsWrapper.displayName).toBe("EChartsWrapper");
  });
});
