/**
 * Phase 88 batch12 — table AssetRow/VDIRow + IconSelect + MarkdownEditor 渲染
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import { AssetRow } from "../AssetRow";
import { VDIRow } from "../VDIRow";
import IconSelect from "../../IconSelect";
import MarkdownEditor from "../../markdown/MarkdownEditor";

async function expectRenders(ui: React.ReactElement) {
  const { rendered } = renderPageWithEndpoints(ui, {});
  await vi.waitFor(() => expect(rendered.container.firstChild).not.toBeNull(), { timeout: 8000 });
  return rendered;
}

describe("零散组件渲染", () => {
  it("AssetRow", async () => {
    await expectRenders(
      <AssetRow record={{ id: "a1", assetName: "笔记本", ipAddress: "1.1.1.1" } as any} />
    );
  });
  it("VDIRow", async () => {
    await expectRenders(
      (
        <VDIRow record={{ id: "v1", vmName: "虚拟机-01" } as any} buttons={[]} permissions={[]} />
      ) as any
    );
  });
  it("IconSelect", async () => {
    await expectRenders(<IconSelect value="DesktopOutlined" onChange={vi.fn()} />);
  });
  it("MarkdownEditor", async () => {
    await expectRenders(<MarkdownEditor value="# 标题" onChange={vi.fn()} />);
  });
});
