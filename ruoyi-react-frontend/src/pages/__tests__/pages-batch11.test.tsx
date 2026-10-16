/**
 * Phase 88 batch11 — notice 组件/duty config+holidays/knowledge view 渲染
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import { NoticeList } from "../system/notice/components/NoticeList";
import { NoticeStatisticsCard } from "../system/notice/components/NoticeStatistics";
import { ChannelSelector } from "../system/notice/components/ChannelSelector";
import DutyConfigPage from "../duty/config";
import DutyHolidaysPage from "../duty/holidays";
import KnowledgeViewPage from "../knowledge/view";

async function expectRenders(ui: React.ReactElement, endpoints: Record<string, unknown> = {}) {
  const { rendered } = renderPageWithEndpoints(ui, { endpoints });
  await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), { timeout: 8000 });
  return rendered;
}

describe("batch11 渲染", () => {
  it("notice/NoticeList(空)", async () => {
    await expectRenders((<NoticeList notices={[]} selectedRowKeys={[]} loading={false} />) as any);
  });
  it("notice/NoticeStatisticsCard", async () => {
    await expectRenders(
      <NoticeStatisticsCard
        statistics={{ total: 10, published: 5, draft: 3, scheduled: 2 } as any}
      />
    );
  });
  it("notice/ChannelSelector", async () => {
    await expectRenders((<ChannelSelector selectedChannels={[]} onChange={vi.fn()} />) as any);
  });
  it("duty/config 页", async () => {
    await expectRenders(<DutyConfigPage />);
  });
  it("duty/holidays 页", async () => {
    await expectRenders(<DutyHolidaysPage />);
  });
  it("knowledge/view 页", async () => {
    await expectRenders(<KnowledgeViewPage />);
  });
});
