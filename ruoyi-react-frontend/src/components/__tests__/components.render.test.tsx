/**
 * Phase 88 batch4 — components 渲染模式重测(真实 hooks + mock API)
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import ExcelImport from "../shared/ExcelImport";
import ExcelImportLazy from "../shared/ExcelImportLazy";
import CronSelector from "../CronSelector";
import { CaptchaModal } from "../captcha";
import DeptTree from "../DeptTree";
import NotificationBell from "../NotificationBell";
import GlobalSearch from "../shared/GlobalSearch";
import FileUpload from "../shared/FileUpload";
import ImageGallery from "../shared/ImageGallery";
import BatchExportModal from "../shared/BatchExportModal";

async function expectRenders(ui: React.ReactElement, endpoints: Record<string, unknown> = {}) {
  const { rendered } = renderPageWithEndpoints(ui, { endpoints });
  await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), { timeout: 8000 });
  return rendered;
}

describe("components 渲染(真实 hooks)", () => {
  it("ExcelImport(closed)渲染", async () => {
    await expectRenders(
      <ExcelImport
        entityType="building"
        visible={false}
        onClose={vi.fn()}
        onImportSuccess={vi.fn()}
      />
    );
  });
  it("ExcelImportLazy 渲染", async () => {
    await expectRenders(
      <ExcelImportLazy
        entityType="building"
        entityName="楼宇"
        visible={false}
        onClose={vi.fn()}
        onImportSuccess={vi.fn()}
      />
    );
  });
  it("CronSelector 渲染", async () => {
    await expectRenders(<CronSelector value="0 0 9 * * ?" onChange={vi.fn()} />);
  });
  it("CaptchaModal(closed)渲染", async () => {
    await expectRenders(<CaptchaModal open={false} onSuccess={vi.fn()} />);
  });
  it("DeptTree 渲染", async () => {
    await expectRenders(<DeptTree onSelect={vi.fn()} />, {
      "/system/departments/tree": { data: [] },
    });
  });
  it("NotificationBell 渲染", async () => {
    await expectRenders(<NotificationBell />);
  });
  it("GlobalSearch 渲染", async () => {
    await expectRenders(<GlobalSearch />);
  });
  it("FileUpload 渲染", async () => {
    await expectRenders(<FileUpload maxCount={1} accept="image/*" />);
  });
  it("ImageGallery(空)渲染", async () => {
    await expectRenders(<ImageGallery photos={[]} />);
  });
  it("BatchExportModal(closed)渲染", async () => {
    await expectRenders(
      <BatchExportModal visible={false} onConfirm={vi.fn()} onCancel={vi.fn()} />
    );
  });
});
