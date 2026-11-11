/**
 * quick 261110-tbq — lib/dbBackupApi 测试（per-dir 覆盖率下限守护）。
 *
 * 断言：10 个 API 全走 @/lib/api 包装函数（post/get/postFormData）与
 * @/lib/download 的 downloadFile；路径正确、直接返回 result.data（不手动判
 * code===0）、空响应有兜底、statistics 多键兼容、restore 用 FormData 字段名 file。
 *
 * ⚠️ vi.mock 工厂会被 hoist 到文件顶部，用 vi.hoisted() 定义 spy 后再引用。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

const { postMock, getMock, postFormDataMock, downloadFileMock } = vi.hoisted(() => ({
  postMock: vi.fn(async () => ({ code: 0, data: {} })),
  getMock: vi.fn(async () => ({ code: 0, data: {} })),
  postFormDataMock: vi.fn(async () => ({ code: 0, data: {} })),
  downloadFileMock: vi.fn(async () => "/tmp/x.zip"),
}));

vi.mock("@/lib/api", () => ({
  post: postMock,
  get: getMock,
  postFormData: postFormDataMock,
}));
vi.mock("@/lib/download", () => ({ downloadFile: downloadFileMock }));

import { dbBackupApi } from "../dbBackupApi";
import type { Plan } from "@/types/backup";

const emptyPlan: Plan = {
  enabled: false,
  cron: "0 0 2 * * *",
  retention: 7,
  storage: "local",
  oss: { endpoint: "", bucket: "", accessKey: "", secretKey: "", useSSL: true },
  ftp: { host: "", port: 21, user: "", password: "", remoteDir: "/" },
  smb: { path: "" },
};

beforeEach(() => {
  postMock.mockClear();
  getMock.mockClear();
  postFormDataMock.mockClear();
  downloadFileMock.mockClear();
});

describe("dbBackupApi — 包装函数与路径契约", () => {
  it("list POST /list，返回 result.data，空体兜底", async () => {
    postMock.mockResolvedValueOnce({
      code: 0,
      data: { list: [{ id: "b1" }], total: 1, current: 1, pageSize: 10 },
    });
    const res = await dbBackupApi.list({ current: 1, pageSize: 10 });
    expect(postMock).toHaveBeenCalledWith("/system/db-backup/list", { current: 1, pageSize: 10 });
    expect(res.total).toBe(1);
    expect(res.list[0].id).toBe("b1");

    postMock.mockResolvedValueOnce({ code: 0, data: undefined });
    const fallback = await dbBackupApi.list({ current: 2, pageSize: 20 });
    expect(fallback).toEqual({ list: [], total: 0, current: 1, pageSize: 10 });
  });

  it("capabilities GET /capabilities，空体兜底为不可用", async () => {
    getMock.mockResolvedValueOnce({
      code: 0,
      data: {
        dbType: "postgres",
        pgDumpAvailable: true,
        pgRestoreAvailable: true,
        pgDumpPath: "/usr/bin/pg_dump",
        pgRestorePath: "/usr/bin/pg_restore",
        hint: "ok",
      },
    });
    const cap = await dbBackupApi.capabilities();
    expect(getMock).toHaveBeenCalledWith("/system/db-backup/capabilities");
    expect(cap.pgDumpAvailable).toBe(true);

    getMock.mockResolvedValueOnce({ code: 0, data: undefined });
    const fallback = await dbBackupApi.capabilities();
    expect(fallback.dbType).toBe("");
    expect(fallback.pgDumpAvailable).toBe(false);
    expect(fallback.hint).toContain("能力预检");
  });

  it("statistics 多键兼容读取 usedSpace/totalSize、success/ready", async () => {
    postMock.mockResolvedValueOnce({
      code: 0,
      data: { total: 3, success: 2, usedSpace: 1024, byStatus: {} },
    });
    let s = await dbBackupApi.statistics();
    expect(postMock).toHaveBeenCalledWith("/system/db-backup/statistics");
    expect(s.total).toBe(3);
    expect(s.success).toBe(2);
    expect(s.usedSpace).toBe(1024);

    postMock.mockResolvedValueOnce({
      code: 0,
      data: { totalCount: 5, ready: 4, total_size: 2048 },
    });
    s = await dbBackupApi.statistics();
    expect(s.total).toBe(5);
    expect(s.success).toBe(4);
    expect(s.usedSpace).toBe(2048);

    postMock.mockResolvedValueOnce({ code: 0, data: { total: "bad", usedSpace: NaN } });
    s = await dbBackupApi.statistics();
    expect(s.total).toBeUndefined();
    expect(s.usedSpace).toBeUndefined();
  });

  it("task GET /tasks/:id 返回 data", async () => {
    getMock.mockResolvedValueOnce({ code: 0, data: { id: "t1", status: "running", progress: 30 } });
    const t = await dbBackupApi.task("t1");
    expect(getMock).toHaveBeenCalledWith("/system/db-backup/tasks/t1");
    expect(t.status).toBe("running");
  });

  it("export POST /export 返回 taskId，空体兜底", async () => {
    postMock.mockResolvedValueOnce({ code: 0, data: { taskId: "task-1" } });
    expect((await dbBackupApi.export()).taskId).toBe("task-1");
    expect(postMock).toHaveBeenCalledWith("/system/db-backup/export");

    postMock.mockResolvedValueOnce({ code: 0, data: undefined });
    expect((await dbBackupApi.export()).taskId).toBe("");
  });

  it("restore 用 FormData 字段名 file 并 POST /restore", async () => {
    postFormDataMock.mockResolvedValueOnce({ code: 0, data: { taskId: "task-2" } });
    const file = new File(["zip"], "bak.zip");
    const res = await dbBackupApi.restore(file);
    expect(postFormDataMock).toHaveBeenCalledTimes(1);
    const [url, formData] = postFormDataMock.mock.calls[0] as unknown as [string, FormData];
    expect(url).toBe("/system/db-backup/restore");
    expect(formData.get("file")).toBe(file);
    expect(res.taskId).toBe("task-2");

    postFormDataMock.mockResolvedValueOnce({ code: 0, data: undefined });
    expect((await dbBackupApi.restore(file)).taskId).toBe("");
  });

  it("remove POST /:id/delete", async () => {
    await dbBackupApi.remove("b9");
    expect(postMock).toHaveBeenCalledWith("/system/db-backup/b9/delete");
  });

  it("download 走唯一权威 downloadFile", async () => {
    const p = await dbBackupApi.download("b9", "bak.zip");
    expect(downloadFileMock).toHaveBeenCalledWith("/system/db-backup/b9/download", "bak.zip");
    expect(p).toBe("/tmp/x.zip");
  });

  it("getPlan / savePlan 走 GET/POST /plan", async () => {
    getMock.mockResolvedValueOnce({ code: 0, data: emptyPlan });
    const plan = await dbBackupApi.getPlan();
    expect(getMock).toHaveBeenCalledWith("/system/db-backup/plan");
    expect(plan.storage).toBe("local");

    const next: Plan = { ...emptyPlan, enabled: true, retention: 10 };
    await dbBackupApi.savePlan(next);
    expect(postMock).toHaveBeenCalledWith("/system/db-backup/plan", next);
  });
});
