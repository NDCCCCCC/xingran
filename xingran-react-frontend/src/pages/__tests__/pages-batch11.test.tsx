/**
 * Phase 88 batch11 — notice 组件/duty 配置与节假日面板/knowledge view 渲染
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
// 原独立页 duty/config、duty/holidays 已并入 duty/management 的 Tab，测试直接渲染其面板组件
import { DutyConfig as DutyConfigPanel } from "../duty/management/components/DutyConfig";
import { HolidayManagement } from "../duty/management/components/HolidayManagement";
import KnowledgeViewPage from "../knowledge/view";

async function expectRenders(ui: React.ReactElement, endpoints: Record<string, unknown> = {}) {
  const { rendered } = renderPageWithEndpoints(ui, { endpoints });
  await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), { timeout: 8000 });
  return rendered;
}

describe("batch11 渲染", () => {
  it("notice/NoticeList(空)", async () => {
    await expectRenders(
      (<NoticeList notices={[]} selectedRowKeys={[]} loading={false} />) as unknown
    );
  });
  it("notice/NoticeStatisticsCard", async () => {
    await expectRenders(
      <NoticeStatisticsCard
        statistics={{ total: 10, published: 5, draft: 3, scheduled: 2 } as unknown}
      />
    );
  });
  it("notice/ChannelSelector", async () => {
    await expectRenders((<ChannelSelector selectedChannels={[]} onChange={vi.fn()} />) as unknown);
  });
  it("duty 配置面板", async () => {
    await expectRenders(
      <DutyConfigPanel config={null} loading={false} saving={false} onSave={vi.fn()} />
    );
  });
  it("duty 节假日面板", async () => {
    await expectRenders(
      <HolidayManagement
        holidays={[]}
        loading={false}
        holidayYear={2026}
        availableYears={[2026]}
        onYearChange={vi.fn()}
        onRefresh={vi.fn()}
        onAdd={vi.fn()}
        onBatchAdd={vi.fn()}
        onEdit={vi.fn()}
        onDelete={vi.fn()}
        onImport={vi.fn()}
        onDownloadTemplate={vi.fn()}
      />
    );
  });
  it("knowledge/view 页", async () => {
    await expectRenders(<KnowledgeViewPage />);
  });
});
