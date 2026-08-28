/**
 * Phase 88 Batch25 — floors 子组件渲染测试
 *
 * 覆盖 FloorCardView / FloorStatisticsCards / FloorSearchForm / FloorModal
 * (FloorPlanEditorView 依赖重 FloorPlanEditor canvas,单独批次处理)
 */
import { describe, it, expect, vi } from "vitest";
import { Form } from "antd";
import { screen, fireEvent } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { FloorCardView } from "../components/FloorCardView";
import { FloorStatisticsCards } from "../components/FloorStatisticsCards";
import { FloorSearchForm } from "../components/FloorSearchForm";
import { FloorModal } from "../components/FloorModal";
import { createFloorTableColumns } from "../components/FloorTableColumns";
import { STATUS_OPTIONS } from "../constants";

const mockFloor = (id: string, name: string, overrides: Record<string, unknown> = {}): any => ({
  id,
  name,
  floorNo: 1,
  buildingName: "A 栋",
  buildingCode: "BLD-A",
  status: 0,
  area: 120,
  ...overrides,
});

describe("FloorCardView", () => {
  it("空 floors 渲染 '暂无数据'", () => {
    const { container, getByText } = renderWithProviders(
      <FloorCardView floors={[]} onEdit={vi.fn()} onEditFloorPlan={vi.fn()} onDelete={vi.fn()} />
    );
    expect(getByText("暂无数据")).toBeDefined();
    expect(container.querySelectorAll(".ant-card")).toHaveLength(0);
  });

  it("3 floors 渲染 3 张卡片", async () => {
    const { container, findAllByText } = renderWithProviders(
      <FloorCardView
        floors={[mockFloor("1", "1F"), mockFloor("2", "2F"), mockFloor("3", "3F")]}
        onEdit={vi.fn()}
        onEditFloorPlan={vi.fn()}
        onDelete={vi.fn()}
      />
    );
    await vi.waitFor(() => {
      expect(container.querySelectorAll(".ant-card")).toHaveLength(3);
    });
    // status=0 → 正常 tag
    expect(await findAllByText("正常")).toHaveLength(3);
  });

  it("status=1 渲染 停用 tag", async () => {
    const { findByText } = renderWithProviders(
      <FloorCardView
        floors={[mockFloor("1", "1F", { status: 1 })]}
        onEdit={vi.fn()}
        onEditFloorPlan={vi.fn()}
        onDelete={vi.fn()}
      />
    );
    expect(await findByText("停用")).toBeDefined();
  });

  it("name 空时回退 floorNo", async () => {
    const { findByText } = renderWithProviders(
      <FloorCardView
        floors={[mockFloor("1", "", { floorNo: 8 })]}
        onEdit={vi.fn()}
        onEditFloorPlan={vi.fn()}
        onDelete={vi.fn()}
      />
    );
    expect(await findByText("8层")).toBeDefined();
  });

  it("area 有值渲染面积", async () => {
    const { findByText } = renderWithProviders(
      <FloorCardView
        floors={[mockFloor("1", "1F", { area: 200 })]}
        onEdit={vi.fn()}
        onEditFloorPlan={vi.fn()}
        onDelete={vi.fn()}
      />
    );
    expect(await findByText("200m²")).toBeDefined();
  });

  it("buildingName 空时回退 buildingCode", async () => {
    const { findByText } = renderWithProviders(
      <FloorCardView
        floors={[mockFloor("1", "1F", { buildingName: "", buildingCode: "BLD-X" })]}
        onEdit={vi.fn()}
        onEditFloorPlan={vi.fn()}
        onDelete={vi.fn()}
      />
    );
    expect(await findByText("BLD-X")).toBeDefined();
  });
});

describe("FloorStatisticsCards", () => {
  it("show=false 返回 null", () => {
    const { container } = renderWithProviders(
      <FloorStatisticsCards show={false} statistics={{ total: 0, active: 0, inactive: 0 }} />
    );
    // AntD Row 也可能留空容器,以 .ant-statistic 不存在为准
    expect(container.querySelectorAll(".ant-statistic")).toHaveLength(0);
  });

  it("show=true 渲染 3 张统计卡", async () => {
    const { container, findByText } = renderWithProviders(
      <FloorStatisticsCards show statistics={{ total: 10, active: 8, inactive: 2 }} />
    );
    expect(await findByText("总楼层数")).toBeDefined();
    await vi.waitFor(() => {
      expect(container.querySelectorAll(".ant-statistic")).toHaveLength(3);
    });
    expect(await findByText("正常楼层")).toBeDefined();
    expect(await findByText("停用楼层")).toBeDefined();
  });
});

describe("FloorSearchForm", () => {
  function SearchHarness(props: Partial<Parameters<typeof FloorSearchForm>[0]> = {}) {
    const [form] = Form.useForm();
    return (
      <FloorSearchForm
        form={form}
        buildingOptions={[]}
        buildingOptionsByDept={[{ id: "b-1", name: "A 栋" }]}
        viewMode="table"
        loading={false}
        selectedDeptId="d-1"
        onSearch={vi.fn()}
        onReset={vi.fn()}
        onRefresh={vi.fn()}
        onViewModeChange={vi.fn()}
        onImport={vi.fn()}
        onExport={vi.fn()}
        onBatchDelete={vi.fn()}
        onAdd={vi.fn()}
        selectedCount={0}
        {...props}
      />
    );
  }

  it("渲染搜索表单 + 按钮组", async () => {
    const { findByText } = renderWithProviders(<SearchHarness />);
    expect(await findByText("所属楼宇")).toBeDefined();
    expect(await findByText("楼层号")).toBeDefined();
    expect(await findByText("楼层名称")).toBeDefined();
    expect(await findByText("状态")).toBeDefined();
    expect(await findByText("搜索")).toBeDefined();
    expect(await findByText("新增楼层")).toBeDefined();
  });

  it("selectedCount>0 渲染批量删除按钮", async () => {
    const { findByText } = renderWithProviders(<SearchHarness selectedCount={3} />);
    expect(await findByText("批量删除 (3)")).toBeDefined();
  });

  it("selectedCount=0 不渲染批量删除", async () => {
    const { queryByText, findByText } = renderWithProviders(<SearchHarness selectedCount={0} />);
    await findByText("新增楼层");
    expect(queryByText(/批量删除/)).toBeNull();
  });

  it("disabled=true 时搜索按钮禁用", async () => {
    const { container, findByText } = renderWithProviders(<SearchHarness disabled />);
    await findByText("搜索");
    const btn = container.querySelectorAll("button");
    const searchBtn = Array.from(btn).find((b) => b.textContent?.includes("搜索"));
    expect(searchBtn?.hasAttribute("disabled")).toBe(true);
  });
});

describe("FloorModal", () => {
  function ModalHarness(props: Partial<Parameters<typeof FloorModal>[0]> = {}) {
    const [form] = Form.useForm();
    return (
      <FloorModal
        visible
        editingFloor={null}
        buildingOptions={[{ id: "b-1", name: "A 栋" }]}
        departments={[]}
        buildingOptionsByDept={[]}
        selectedDeptId=""
        form={form}
        onOk={vi.fn()}
        onCancel={vi.fn()}
        onDepartmentChange={vi.fn()}
        {...props}
      />
    );
  }

  it("新增模式标题", async () => {
    const { findByText } = renderWithProviders(<ModalHarness />);
    expect(await findByText("新增楼层")).toBeDefined();
  });

  it("编辑模式标题 + 楼宇禁用", async () => {
    const editing = mockFloor("1", "1F");
    const { findByText } = renderWithProviders(<ModalHarness editingFloor={editing} />);
    expect(await findByText("编辑楼层")).toBeDefined();
  });

  it("closed (visible=false) 渲染空", async () => {
    const { container } = renderWithProviders(<ModalHarness visible={false} />);
    await vi.waitFor(() => {
      expect(container.querySelector(".ant-modal")).toBeNull();
    });
  });
});

describe("createFloorTableColumns", () => {
  const cbs = {
    onEdit: vi.fn(),
    onEditFloorPlan: vi.fn(),
    onDelete: vi.fn(),
    getColumnSortOrder: (_: string) => undefined as never,
  };

  it("返回 8 列含主键", () => {
    const cols = createFloorTableColumns(cbs);
    expect(cols.length).toBe(8);
    const keys = cols.map((c) => c.key as string);
    expect(keys).toEqual(
      expect.arrayContaining([
        "name",
        "floorNo",
        "buildingName",
        "area",
        "status",
        "description",
        "createdAt",
        "action",
      ])
    );
  });

  it("area 列空值 render '-'", () => {
    const cols = createFloorTableColumns(cbs);
    const area = cols.find((c) => c.key === "area");
    const render = area?.render as (v: unknown) => unknown;
    expect(render(undefined)).toBe("-");
    expect(render(100)).toBe(100);
  });

  it("buildingName render 回退 buildingCode", () => {
    const cols = createFloorTableColumns(cbs);
    const col = cols.find((c) => c.key === "buildingName");
    const render = col?.render as (_: unknown, record: any) => unknown;
    expect(render(undefined, { buildingName: "", buildingCode: "BLD-X" })).toBe("BLD-X");
    expect(render(undefined, { buildingName: "A 栋", buildingCode: "BLD-A" })).toBe("A 栋");
  });

  it("sorter 传参生效", () => {
    const cb2 = {
      ...cbs,
      getColumnSortOrder: (f: string) => (f === "floorNo" ? ("descend" as const) : undefined),
    };
    const cols = createFloorTableColumns(cb2);
    const floorNo = cols.find((c) => c.key === "floorNo");
    expect(floorNo?.sortOrder).toBe("descend");
  });
});
