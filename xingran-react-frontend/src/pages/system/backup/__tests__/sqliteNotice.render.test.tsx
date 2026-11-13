/**
 * SQLite 前置提示渲染测试（quick 261112 恢复一致性整改）。
 *
 * 依据 dbBackupApi.capabilities() 返回的 dbType === "sqlite"：
 *   - 列表页（BackupListTab）常驻显示「SQLite 整库恢复需停服离线执行」说明、
 *     离线命令入口与「备份包含全库敏感数据」说明；
 *   - 恢复弹窗（RestoreModal）在 sqlite 分支同样常驻显示离线提示与命令入口，
 *     而不是只在失败后把命令塞进 errorMessage；
 *   - 非 sqlite（postgres）分支不应出现离线提示。
 *
 * 直接对组件传 capabilities prop 渲染，避免依赖权限门控/API 轮询时序。
 */
import { describe, it, expect, vi } from "vitest";
import { screen } from "@testing-library/react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});
vi.mock("@/lib/download", () => ({
  downloadFile: vi.fn(async () => ""),
}));

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import type { Capabilities } from "@/types/backup";
import { SQLITE_OFFLINE_NOTICE, SQLITE_OFFLINE_COMMAND } from "../types";
import RestoreModal from "../components/RestoreModal";
import BackupListTab from "../components/BackupListTab";

const sqliteCapabilities: Capabilities = {
  dbType: "sqlite",
  pgDumpAvailable: true,
  pgRestoreAvailable: true,
  pgDumpPath: "",
  pgRestorePath: "",
  hint: "SQLite 模式：导出走纯 Go VACUUM INTO；整库恢复需停服后离线执行。",
};

const pgCapabilities: Capabilities = {
  ...sqliteCapabilities,
  dbType: "postgres",
  pgDumpPath: "/usr/bin/pg_dump",
  pgRestorePath: "/usr/bin/pg_restore",
  hint: "pg_dump / pg_restore 均可用。",
};

describe("RestoreModal — sqlite 前置提示", () => {
  it("dbType=sqlite 时常驻显示离线恢复说明与离线命令入口", () => {
    renderWithProviders(
      <RestoreModal
        open
        taskRunning={false}
        capabilities={sqliteCapabilities}
        onClose={() => {}}
        onStartTask={() => {}}
      />
    );
    expect(screen.getByText("SQLite 整库恢复需停服离线执行")).not.toBeNull();
    expect(screen.getByText(SQLITE_OFFLINE_NOTICE)).not.toBeNull();
    expect(screen.getByText(SQLITE_OFFLINE_COMMAND)).not.toBeNull();
    // RESTORE_CONFIRM_TEXT 已补充「将覆盖现有全部数据、期间不可写入、恢复后需重新登录」
    expect(screen.getAllByText(/恢复后需重新登录/).length).toBeGreaterThan(0);
  });

  it("dbType=postgres 时不显示 SQLite 离线提示", () => {
    renderWithProviders(
      <RestoreModal
        open
        taskRunning={false}
        capabilities={pgCapabilities}
        onClose={() => {}}
        onStartTask={() => {}}
      />
    );
    expect(screen.queryByText(SQLITE_OFFLINE_NOTICE)).toBeNull();
    expect(screen.queryByText(SQLITE_OFFLINE_COMMAND)).toBeNull();
  });
});

describe("BackupListTab — sqlite 前置提示", () => {
  it("dbType=sqlite 时首屏常驻离线提示 + 全库敏感数据说明", async () => {
    // 表格/统计/能力预检/capabilities 端点由 createApiTestingModule 通用兜底，
    // 这里通过 capabilities() 的 mock 返回 sqlite；直接传 capabilities 无法覆盖
    // 组件内部 effect，故用 renderPageWithEndpoints 注册 capabilities 端点。
    const { renderPageWithEndpoints } = await import("@/test/utils/renderPage");
    const { screen: pageScreen } = renderPageWithEndpoints(
      <BackupListTab onStartTask={() => {}} refreshToken={0} taskRunning={false} />,
      {
        endpoints: {
          "/system/db-backup/list": { data: { list: [], total: 0, current: 1, pageSize: 10 } },
          "/system/db-backup/statistics": { data: { total: 0, success: 0, usedSpace: 0 } },
          "/system/db-backup/capabilities": { data: sqliteCapabilities },
        },
      }
    );
    expect(await pageScreen.findByText("SQLite 整库恢复需停服离线执行")).not.toBeNull();
    expect(await pageScreen.findByText("备份包含全库敏感数据")).not.toBeNull();
    expect(await pageScreen.findByText("恢复后需重新登录")).not.toBeNull();
    expect(await pageScreen.findByText(SQLITE_OFFLINE_COMMAND)).not.toBeNull();
  });
});
