/**
 * Phase 88 Batch190 — pages/system/role/utils 测试
 */
import { describe, it, expect, vi } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/utils/datetime", () => ({
  formatDateTime: vi.fn(() => "2026-08-30 10:00:00"),
}));

import {
  processTreeData,
  renderStatusTag,
  renderRoleName,
  renderRoleKeyTag,
  formatLocalTime,
} from "../utils";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("system/role/utils", () => {
  it("processTreeData 扁平节点", () => {
    const nodes = [
      { id: "1", title: "A" },
      { id: "2", title: "B" },
    ];
    const result = processTreeData(nodes);
    expect(result.length).toBe(2);
    expect(result[0].key).toBe("1");
    expect(result[0].title).toBe("A");
  });

  it("processTreeData 嵌套 children", () => {
    const nodes = [
      {
        id: "1",
        title: "A",
        children: [{ id: "2", title: "B" }],
      },
    ];
    const result = processTreeData(nodes);
    expect(result.length).toBe(1);
    expect(result[0].children?.length).toBe(1);
    expect(result[0].children?.[0].title).toBe("B");
  });

  it("processTreeData 空 children → undefined", () => {
    const nodes = [{ id: "1", title: "A", children: [] }];
    const result = processTreeData(nodes);
    expect(result[0].children).toBeUndefined();
  });

  it("processTreeData 自定义 keyField/titleField", () => {
    const nodes = [{ uuid: "x1", name: "Test" }];
    const result = processTreeData(nodes, "uuid", "name");
    expect(result[0].title).toBe("Test");
  });

  it("renderStatusTag 0 → 正常", () => {
    const { baseElement } = render(<>{renderStatusTag(0)}</>, { wrapper });
    expect(baseElement.textContent).toContain("正常");
  });

  it("renderStatusTag 1 → 停用", () => {
    const { baseElement } = render(<>{renderStatusTag(1)}</>, { wrapper });
    expect(baseElement.textContent).toContain("停用");
  });

  it("renderRoleName 包含图标 + 文本", () => {
    const { baseElement } = render(<>{renderRoleName("admin")}</>, { wrapper });
    expect(baseElement.textContent).toContain("admin");
    expect(baseElement.querySelector(".anticon")).toBeTruthy();
  });

  it("renderRoleKeyTag → 蓝色 Tag", () => {
    const { baseElement } = render(<>{renderRoleKeyTag("admin")}</>, { wrapper });
    expect(baseElement.textContent).toContain("admin");
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });

  it("formatLocalTime 调 formatDateTime", () => {
    expect(formatLocalTime("2026-01-01T00:00:00Z")).toBe("2026-08-30 10:00:00");
  });
});
