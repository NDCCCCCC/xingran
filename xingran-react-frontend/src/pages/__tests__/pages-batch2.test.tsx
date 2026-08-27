/**
 * Phase 88 — 第二批页面渲染: monitor 全家 + asset + ad-domain 剩余 + operations rpa
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import Dashboard from "../monitor/dashboard";
import ServerMonitor from "../monitor/server";
import CacheManager from "../monitor/cache";
import LogMonitor from "../monitor/logs";
import JobManager from "../monitor/job";
import ADComputers from "../ad-domain/computers";
import ADOus from "../ad-domain/ous";
import ADLogs from "../ad-domain/logs";
import RpaTasks from "../operations/rpa/tasks";
import RpaWorkers from "../operations/rpa/workers";

async function expectRenders(ui: React.ReactElement, endpoints: Record<string, unknown> = {}) {
  const { rendered } = renderPageWithEndpoints(ui, { endpoints });
  await vi.waitFor(
    () => {
      expect(
        rendered.container.querySelector(".ant-table, .ant-card, .ant-empty, .ant-tabs, .ant-form")
      ).not.toBeNull();
    },
    { timeout: 8000 }
  );
  return rendered;
}

describe("第二批页面批量渲染(真实 hooks)", () => {
  it("monitor/dashboard 页", async () => {
    await expectRenders(<Dashboard />);
  });
  it("monitor/server 页", async () => {
    await expectRenders(<ServerMonitor />);
  });
  it("monitor/cache 页", async () => {
    await expectRenders(<CacheManager />);
  });
  it("monitor/logs 页", async () => {
    await expectRenders(<LogMonitor />);
  });
  it("monitor/job 页", async () => {
    await expectRenders(<JobManager />);
  });
  it("ad-domain/computers 页", async () => {
    await expectRenders(<ADComputers />);
  });
  it("ad-domain/ous 页", async () => {
    await expectRenders(<ADOus />);
  });
  it("ad-domain/logs 页", async () => {
    await expectRenders(<ADLogs />);
  });
  it("rpa/tasks 页", async () => {
    await expectRenders(<RpaTasks />);
  });
  it("rpa/workers 页", async () => {
    await expectRenders(<RpaWorkers />);
  });
});
