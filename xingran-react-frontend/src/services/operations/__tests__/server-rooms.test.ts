/**
 * Phase 88 Batch215 — services/operations/server-rooms + room-devices 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  return {
    post: vi.fn(async (url: string, data?: unknown) => ({
      code: 0,
      data: { url, payload: data },
    })),
  };
});

import * as api from "@/lib/api";
import { serverRoomApi } from "../server-rooms";
import { roomDeviceApi } from "../room-devices";

describe("services/operations/server-rooms", () => {
  it("list", async () => {
    const r: unknown = await serverRoomApi.list({ current: 1, pageSize: 10 });
    expect(r.data.url).toBe("/ops/server-rooms/list");
  });
  it("create/update/delete/batchDelete", async () => {
    const c: unknown = await serverRoomApi.create({ name: "R1" } as unknown);
    expect(c.data.url).toBe("/ops/server-rooms");
    const u: unknown = await serverRoomApi.update("r1", { name: "R2" });
    expect(u.data.url).toContain("/r1/update");
    const d: unknown = await serverRoomApi.delete("r1");
    expect(d.data.url).toContain("/r1/delete");
    const b: unknown = await serverRoomApi.batchDelete(["a"]);
    expect(b.data.url).toBe("/ops/server-rooms/batch");
  });
  it("export/import/template", async () => {
    const e: unknown = await serverRoomApi.export({ current: 1, pageSize: 10 });
    expect(e.data.url).toBe("/ops/server-rooms/export");
    const i: unknown = await serverRoomApi.import(new File(["x"], "rooms.xlsx"));
    expect(i.data.url).toBe("/ops/server-rooms/import");
    const t: unknown = await serverRoomApi.downloadTemplate();
    expect(t.data.url).toBe("/ops/server-rooms/template");
  });
});

describe("services/operations/room-devices", () => {
  it("list", async () => {
    const r: unknown = await roomDeviceApi.list({ current: 1, pageSize: 10 });
    expect(r.data.url).toBe("/ops/room-devices/list");
  });
  it("create/update/delete/batchDelete", async () => {
    const c: unknown = await roomDeviceApi.create({ name: "RD1" } as unknown);
    expect(c.data.url).toBe("/ops/room-devices");
    const u: unknown = await roomDeviceApi.update("rd1", { name: "RD2" });
    expect(u.data.url).toContain("/rd1/update");
    const d: unknown = await roomDeviceApi.delete("rd1");
    expect(d.data.url).toContain("/rd1/delete");
    const b: unknown = await roomDeviceApi.batchDelete(["a"]);
    expect(b.data.url).toBe("/ops/room-devices/batch");
  });
  it("export/import/template", async () => {
    const e: unknown = await roomDeviceApi.export({ current: 1, pageSize: 10 });
    expect(e.data.url).toBe("/ops/room-devices/export");
    const i: unknown = await roomDeviceApi.import(new File(["x"], "rds.xlsx"));
    expect(i.data.url).toBe("/ops/room-devices/import");
    const t: unknown = await roomDeviceApi.downloadTemplate();
    expect(t.data.url).toBe("/ops/room-devices/template");
  });
  it("post spy", async () => {
    await roomDeviceApi.list({ current: 1, pageSize: 10 });
    expect(api.post).toHaveBeenCalled();
  });
});
