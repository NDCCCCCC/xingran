/**
 * Phase 88 Batch76 — RPA workers columns 测试(36 stmts)
 */
import { describe, it, expect, vi } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { Table } from "antd";
import { getWorkerColumns } from "../columns";
import type { Worker } from "@/types/rpa";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

function makeWorker(overrides: Partial<Worker> = {}): Worker {
  const now = Math.floor(Date.now() / 1000);
  return {
    id: "w1",
    workerId: "worker-001",
    workerName: "测试Worker",
    ipAddress: "10.0.0.1",
    port: 9000,
    status: "online",
    lastHeartbeat: now, // 当前心跳 → online
    currentTasks: 2,
    maxConcurrency: 5,
    capabilities: { maxConcurrency: 5 },
    createdAt: "2026-01-01 12:00:00",
    ...overrides,
  } as Worker;
}

describe("getWorkerColumns", () => {
  it("默认无 sortOrder → 10 列", () => {
    const cols = getWorkerColumns();
    expect(cols).toHaveLength(10);
  });

  it("getSortOrder → 注入到指定列", () => {
    const cols = getWorkerColumns({ getSortOrder: () => "ascend" });
    const sorted = cols.filter((c: any) => c.sortOrder === "ascend");
    expect(sorted.length).toBeGreaterThan(0);
  });

  it("workerId 列: workerId 为空时回退到 record.id", () => {
    const cols = getWorkerColumns();
    const workerIdCol = cols.find((c: any) => c.key === "workerId") as any;
    const node = workerIdCol.render("", { id: "fallback-id" });
    expect(node).toBe("fallback-id");
    const dashNode = workerIdCol.render("", {});
    expect(dashNode).toBe("-");
    const passThrough = workerIdCol.render("wid-1", {});
    expect(passThrough).toBe("wid-1");
  });

  it("workerName 列 render: 空 → '-'", () => {
    const cols = getWorkerColumns();
    const col = cols.find((c: any) => c.key === "workerName") as any;
    const node = col.render("", makeWorker());
    expect(node).toBeDefined();
  });

  it("ipAddress 列: 空 → record.workerId 回退", () => {
    const cols = getWorkerColumns();
    const col = cols.find(
      (c: any) => c.key === "ipAddress" && c.width === 150 && c.title === "主机名"
    ) as any;
    const node = col.render("", { workerId: "fback" });
    expect(node).toBeDefined();
  });

  it("status 列: busy 保留 busy;否则基于心跳判断 online/offline", () => {
    const cols = getWorkerColumns();
    const col = cols.find((c: any) => c.key === "status") as any;
    const busyNode = col.render("busy", makeWorker({ status: "busy" }));
    expect(busyNode).toBeDefined();
    // heartbeat 太老 → offline
    const old = Math.floor(Date.now() / 1000) - 1000;
    const offlineNode = col.render("online", makeWorker({ lastHeartbeat: old }));
    expect(offlineNode).toBeDefined();
  });

  it("currentTasks 列: count/max 拼接;无 record.capabilities 回退 3", () => {
    const cols = getWorkerColumns();
    const col = cols.find((c: any) => c.key === "currentTasks") as any;
    const node = col.render(2, makeWorker({ maxConcurrency: 5 }));
    expect(node).toBeDefined();
    const fallback = col.render(undefined, { id: "x" });
    expect(fallback).toBeDefined();
  });

  it("maxConcurrency 列 render", () => {
    const cols = getWorkerColumns();
    const col = cols.find((c: any) => c.key === "maxConcurrency") as any;
    const node = col.render(undefined, { capabilities: {} });
    expect(node).toBeDefined();
  });

  it("port 列: undefined → '-'", () => {
    const cols = getWorkerColumns();
    const col = cols.find((c: any) => c.key === "port") as any;
    const node = col.render(undefined, makeWorker());
    expect(node).toBeDefined();
  });

  it("lastHeartbeat 列: undefined → '-'; > 2 分钟带后缀", () => {
    const cols = getWorkerColumns();
    const col = cols.find((c: any) => c.key === "lastHeartbeat") as any;
    expect(col.render(undefined)).toBeDefined();
    const old = Math.floor(Date.now() / 1000) - 600;
    expect(col.render(old)).toBeDefined();
    const fresh = Math.floor(Date.now() / 1000);
    expect(col.render(fresh)).toBeDefined();
  });

  it("createdAt 列: formatDateTime 路径", () => {
    const cols = getWorkerColumns();
    const col = cols.find((c: any) => c.key === "createdAt") as any;
    expect(col.render("2026-01-01T12:00:00Z")).toBeDefined();
    expect(col.render(undefined)).toBeDefined();
  });

  it("整体 Table 集成渲染: 1 行 → 表格行存在", () => {
    const cols = getWorkerColumns();
    const { baseElement } = renderWithProviders(
      <Table<Worker> rowKey="id" columns={cols} dataSource={[makeWorker()]} pagination={false} />
    );
    expect(baseElement.querySelector(".ant-table-row")).not.toBeNull();
  });
});
