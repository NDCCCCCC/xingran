/**
 * Phase 88 Batch183 — pages/knowledge/articles/constants 测试
 */
import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { KnowledgeArticleStatus } from "@/lib/knowledgeApi";
import { STATUS_CONFIG, STATUS_OPTIONS, renderStatusTag } from "../constants";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("knowledge/articles/constants", () => {
  it("STATUS_CONFIG 含 Draft + Published", () => {
    expect(STATUS_CONFIG[KnowledgeArticleStatus.Draft].text).toBe("草稿");
    expect(STATUS_CONFIG[KnowledgeArticleStatus.Published].text).toBe("已发布");
  });

  it("STATUS_OPTIONS 2 项", () => {
    expect(STATUS_OPTIONS.length).toBe(2);
    expect(STATUS_OPTIONS.map((o) => o.value)).toEqual([
      KnowledgeArticleStatus.Draft,
      KnowledgeArticleStatus.Published,
    ]);
  });

  it("renderStatusTag Draft(0) → 草稿", () => {
    const { baseElement } = render(<>{renderStatusTag(KnowledgeArticleStatus.Draft)}</>, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("草稿");
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });

  it("renderStatusTag Published(1) → 已发布", () => {
    const { baseElement } = render(<>{renderStatusTag(KnowledgeArticleStatus.Published)}</>, {
      wrapper,
    });
    expect(baseElement.textContent).toContain("已发布");
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });
});
