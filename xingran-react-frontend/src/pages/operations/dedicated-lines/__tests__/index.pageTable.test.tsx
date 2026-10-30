/**
 * Phase 68.1 T6a — dedicated-lines 迁移到 PageTable 后的契约断言
 *
 * 1) 数据层：不再直接 import 页面级表格 hook（P66 豁免消失的唯一证据路径）
 * 2) §8-6 硬判据：迁移前批删无二次确认，迁移后必须走 confirmDelete 强确认
 * 3) Q1：行删除入口经 createDeleteAction，可访问名/确认正文含对象名
 */
import { describe, it, expect, vi, afterEach, beforeEach } from "vitest";
import { renderWithProviders } from "@/test/utils/renderWithProviders";
import { screen, fireEvent, waitFor, cleanup } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import * as fs from "fs";
import * as path from "path";
import DedicatedLines from "../index";
import { dedicatedLineApi, serverRoomApi } from "@/lib/opsApi";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/opsApi", () => ({
  dedicatedLineApi: {
    list: vi.fn(),
    statistics: vi.fn(),
    create: vi.fn(),
    update: vi.fn(),
    delete: vi.fn(),
    batch: vi.fn(),
  },
  serverRoomApi: {
    list: vi.fn(),
  },
}));

const ROW = {
  id: "l1",
  name: "专线A",
  lineType: "mpls",
  bandwidth: "100M",
  isp: "ct",
  status: 0,
  monthlyFee: 0,
  createdAt: "2026-01-01T00:00:00Z",
};

beforeEach(() => {
  vi.mocked(dedicatedLineApi.list).mockResolvedValue({
    data: { list: [ROW], total: 1 },
  } as never);
  vi.mocked(dedicatedLineApi.statistics).mockResolvedValue({
    data: { total: 1, normal: 1, fault: 0, disabled: 0 },
  } as never);
  vi.mocked(serverRoomApi.list).mockResolvedValue({ data: { list: [] } } as never);
  vi.mocked(dedicatedLineApi.batch).mockResolvedValue({} as never);
  vi.mocked(dedicatedLineApi.delete).mockResolvedValue({} as never);
});

// 命令式 Modal.confirm 是独立 React root：必须让它自己关闭，手动摘 DOM 会让其
// root unmount 抛 NotFoundError（unhandled error）。用例内关闭后只需 RTL cleanup。
afterEach(() => {
  cleanup();
});

async function waitConfirmClosed() {
  await waitFor(() => expect(document.querySelector(".ant-modal-confirm")).toBeFalsy());
}

function renderLines() {
  const qc = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  return renderWithProviders(
    <QueryClientProvider client={qc}>
      <DedicatedLines />
    </QueryClientProvider>
  );
}

describe("dedicated-lines → PageTable 契约", () => {
  it("数据层契约：不再 import 页面级表格 hook，搜索字段与后端参数一一对应", () => {
    const src = fs.readFileSync(path.resolve(__dirname, "../index.tsx"), "utf-8");
    expect(src).not.toMatch(/from "@\/hooks\/useTableManager"/);
    expect(src).not.toMatch(/from "@\/hooks\/usePagination"/);
    expect(src).toMatch(/<PageTable<DedicatedLine>/);
    // Phase 86 W2：batchDelete 改为**条件传参**（PageTable 的 batchDelete 契约无 permission 位
    // → 页面层按 permDelete.visible 传 / 不传）。契约意图不变：仍走 PageTable.batchDelete，
    // 仍以 handleBatchDelete + target="专线" 为载荷；断言随之放宽为「载荷片段 + 条件传参」。
    expect(src).toMatch(/batchDelete=\{/);
    expect(src).toContain('{ onDelete: handleBatchDelete, target: "专线" }');
    // 门控通道必须在场（本波新增契约）：条件传参由 permDelete.visible 驱动
    expect(src).toMatch(
      /permDelete\.visible[\s\S]{0,120}batchDelete|batchDelete[\s\S]{0,200}permDelete\.visible/
    );
    // 搜索 schema 六字段与迁移前 searchForm 字段同名（红线 1）
    for (const field of ["name", "sourceRoomId", "destRoomId", "lineType", "isp", "status"]) {
      expect(src).toContain(`name: "${field}"`);
    }
    // 批删不再自写按钮（迁移前 :672-678）
    expect(src).not.toMatch(/批量删除 \(\{selectedRowKeys\.length\}\)/);
  });

  it("批删必经二次确认（§8-6），确认后才发请求", async () => {
    renderLines();
    await waitFor(() => expect(screen.getAllByText("专线A").length).toBeGreaterThan(0));

    // 勾选首行数据行复选框（[0] 为表头全选）→ 出现批量删除按钮
    const checkboxes = document.querySelectorAll<HTMLInputElement>(
      ".ant-table-selection-column input[type=checkbox]"
    );
    fireEvent.click(checkboxes[1]);
    await waitFor(() => expect(screen.getByText(/批量删除/)).toBeTruthy());
    fireEvent.click(screen.getByText(/批量删除/).closest("button") as HTMLElement);

    await waitFor(() => {
      const confirmEl = document.querySelector(".ant-modal-confirm");
      expect(confirmEl).toBeTruthy();
      expect(screen.getAllByText("确认批量删除").length).toBeGreaterThanOrEqual(1);
      // INV-8 逐字文案（batchDelete.target 透传 →「选中的 1 项专线」）
      expect(confirmEl?.textContent).toContain("将删除选中的 1 项专线，删除后不可恢复，是否继续？");
    });
    // 确认前绝不发删除请求（迁移前是直接调用）
    expect(dedicatedLineApi.batch).not.toHaveBeenCalled();

    const okButton = document.querySelector(
      ".ant-modal-confirm .ant-btn-dangerous"
    ) as HTMLElement | null;
    expect(okButton).toBeTruthy();
    fireEvent.click(okButton as HTMLElement);
    await waitFor(() =>
      expect(dedicatedLineApi.batch).toHaveBeenCalledWith("delete", { ids: ["l1"] })
    );
    await waitConfirmClosed();
  });

  it("行删除入口经 createDeleteAction：可访问名与确认正文含「专线 专线A」（Q1）", async () => {
    renderLines();
    const deleteButtons = await screen.findAllByLabelText("删除专线 专线A");
    expect(deleteButtons.length).toBeGreaterThan(0);

    fireEvent.click(deleteButtons[0]);
    await waitFor(() => {
      const confirmEl = document.querySelector(".ant-modal-confirm");
      expect(confirmEl?.textContent).toContain("将删除专线 专线A，删除后不可恢复，是否继续？");
    });
    expect(dedicatedLineApi.delete).not.toHaveBeenCalled();

    // 取消关闭（不走删除），让命令式 confirm 的独立 root 正常 unmount
    const cancelButton = document.querySelector(
      ".ant-modal-confirm .ant-btn:not(.ant-btn-dangerous)"
    ) as HTMLElement | null;
    expect(cancelButton).toBeTruthy();
    fireEvent.click(cancelButton as HTMLElement);
    await waitConfirmClosed();
    expect(dedicatedLineApi.delete).not.toHaveBeenCalled();
  });
});
