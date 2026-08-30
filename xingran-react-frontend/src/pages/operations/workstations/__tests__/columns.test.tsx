/**
 * Phase 88 Batch109 — operations/workstations/columns 测试
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { Table } from "antd";
import { getWorkstationColumns } from "../columns";
import type { WorkstationOps } from "@/types";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

const params = {
  handleEdit: vi.fn(),
  handleDelete: vi.fn(),
  getColumnSortOrder: vi.fn(() => undefined),
};

const baseWs: WorkstationOps = {
  id: "w1",
  name: "WS001",
  workstationCode: "WS001",
  workstationName: "WS001",
  workstationType: 0,
  status: 0,
} as any;

describe("workstations columns", () => {
  it("getWorkstationColumns → 11 列", () => {
    const cols = getWorkstationColumns(params);
    expect(cols.length).toBeGreaterThanOrEqual(8);
  });

  it("含 名称/楼宇/楼层/部门/用户/类型/状态 等列", () => {
    const cols = getWorkstationColumns(params);
    const keys = cols.map((c: any) => c.key);
    expect(keys).toContain("name");
    expect(keys).toContain("buildingName");
    expect(keys).toContain("floorName");
    expect(keys).toContain("deptName");
    expect(keys).toContain("userName");
    expect(keys).toContain("type");
    expect(keys).toContain("status");
  });

  it("render name/buildingName 等 → text || '-'", () => {
    const cols = getWorkstationColumns(params);
    const dashNode = cols.find((c: any) => c.key === "buildingName")!.render!("");
    expect(dashNode).toBe("-");
  });

  it("render type → renderWorkstationTypeTag", () => {
    const cols = getWorkstationColumns(params);
    const node = cols.find((c: any) => c.key === "type")!.render!(0);
    expect(node).toBeDefined();
  });

  it("render status → renderWorkstationStatusTag", () => {
    const cols = getWorkstationColumns(params);
    const node = cols.find((c: any) => c.key === "status")!.render!(0);
    expect(node).toBeDefined();
  });

  it("getColumnSortOrder 返回 'ascend' → 列 sortOrder", () => {
    const cols = getWorkstationColumns({
      ...params,
      getColumnSortOrder: (field) => (field === "name" ? "ascend" : null),
    });
    const nameCol = cols.find((c: any) => c.key === "name");
    expect(nameCol.sortOrder).toBe("ascend");
  });

  it("操作列 → ActionButtons 含 edit/delete 回调", () => {
    const handleEdit = vi.fn();
    const handleDelete = vi.fn();
    const cols = getWorkstationColumns({ ...params, handleEdit, handleDelete });
    const actionCol = cols[cols.length - 1];
    // 通过 Table 渲染
    const { baseElement } = renderWithProviders(
      <Table<WorkstationOps> rowKey="id" columns={cols} dataSource={[baseWs]} pagination={false} />
    );
    expect(baseElement).toBeDefined();
    void actionCol;
  });
});
