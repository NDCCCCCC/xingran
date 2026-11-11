/**
 * quick 261110-tbq — 备份页渲染测试（pages/system per-dir 覆盖率下限守护）。
 *
 * 用真实 hooks + mock API 渲染首屏，覆盖 index.tsx / BackupListTab /
 * BackupPlanTab 的初始化分支、统计卡、表格与能力预检 Alert。
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});
vi.mock("@/lib/download", () => ({
  downloadFile: vi.fn(async () => ""),
}));

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import BackupPage from "../index";

const emptyPlan = {
  enabled: false,
  cron: "0 0 2 * * *",
  retention: 7,
  storage: "local",
  oss: { endpoint: "", bucket: "", accessKey: "", secretKey: "", useSSL: true },
  ftp: { host: "", port: 21, user: "", password: "", remoteDir: "/" },
  smb: { path: "" },
};

const capabilities = {
  dbType: "postgres",
  pgDumpAvailable: true,
  pgRestoreAvailable: true,
  pgDumpPath: "/usr/bin/pg_dump",
  pgRestorePath: "/usr/bin/pg_restore",
  hint: "pg_dump / pg_restore 均可用。",
};

describe("数据备份与恢复页渲染", () => {
  it("首屏渲染 Tab + 统计卡 + 表格（工具可用）", async () => {
    const { screen } = await renderPageWithEndpoints(<BackupPage />, {
      endpoints: {
        "/system/db-backup/list": { data: { list: [], total: 0, current: 1, pageSize: 10 } },
        "/system/db-backup/statistics": {
          data: { total: 2, success: 2, usedSpace: 2048, byStatus: { ready: 2 } },
        },
        "/system/db-backup/capabilities": { data: capabilities },
        "/system/db-backup/plan": { data: emptyPlan },
      },
    });
    expect(await screen.findByText("备份列表")).not.toBeNull();
    expect(await screen.findByText("备份计划")).not.toBeNull();
    expect(await screen.findByText("备份总数")).not.toBeNull();
  }, 15000);

  it("pg_dump 缺失时展示预检告警", async () => {
    const { screen } = await renderPageWithEndpoints(<BackupPage />, {
      endpoints: {
        "/system/db-backup/list": { data: { list: [], total: 0, current: 1, pageSize: 10 } },
        "/system/db-backup/statistics": { data: { total: 0, success: 0, usedSpace: 0 } },
        "/system/db-backup/capabilities": {
          data: {
            ...capabilities,
            pgDumpAvailable: false,
            hint: "部署机未找到 pg_dump，请安装客户端。",
          },
        },
        "/system/db-backup/plan": { data: emptyPlan },
      },
    });
    expect(await screen.findByText("PostgreSQL 备份工具未就绪")).not.toBeNull();
  }, 15000);
});
