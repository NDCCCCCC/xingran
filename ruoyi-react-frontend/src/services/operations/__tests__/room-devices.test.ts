/**
 * Phase 88 Batch300 — services/operations/room-devices 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  return {
    post: vi.fn(async (url: string, data?: any) => ({ data: { url, payload: data } })),
  };
});

import * as api from "@/lib/api";
import { roomDeviceApi } from "../room-devices";

describe("services/operations/room-devices", () => {
  it("list", async () => {
    const r: any = await roomDeviceApi.list({ current: 1, pageSize: 10 });
    expect(r.data.url).toBe("/ops/room-devices/list");
  });

  it("create", async () => {
    const r: any = await roomDeviceApi.create({ deviceName: "R1" } as any);
    expect(r.data.url).toBe("/ops/room-devices");
  });

  it("update", async () => {
    const r: any = await roomDeviceApi.update("rd-1", { deviceName: "R2" });
    expect(r.data.url).toContain("/rd-1/update");
  });

  it("delete", async () => {
    const r: any = await roomDeviceApi.delete("rd-1");
    expect(r.data.url).toContain("/rd-1/delete");
  });

  it("batchDelete", async () => {
    const r: any = await roomDeviceApi.batchDelete(["a", "b"]);
    expect(r.data.url).toBe("/ops/room-devices/batch");
    expect(r.data.payload.action).toBe("delete");
    expect(r.data.payload.ids).toEqual(["a", "b"]);
  });

  it("export", async () => {
    const r: any = await roomDeviceApi.export({ current: 1, pageSize: 20 });
    expect(r.data.url).toBe("/ops/room-devices/export");
  });

  it("import", async () => {
    const fakeFile = new File(["content"], "rooms.csv", { type: "text/csv" });
    const r: any = await roomDeviceApi.import(fakeFile);
    expect(r.data.url).toBe("/ops/room-devices/import");
    expect(r.data.payload).toBeInstanceOf(FormData);
  });

  it("downloadTemplate", async () => {
    const r: any = await roomDeviceApi.downloadTemplate();
    expect(r.data.url).toBe("/ops/room-devices/template");
  });

  it("post spy", async () => {
    await roomDeviceApi.list({ current: 1, pageSize: 5 });
    expect(api.post).toHaveBeenCalled();
  });
});
