/**
 * Phase 88 Batch336 — pages/system/role/utils 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  processTreeData,
  renderStatusTag,
  renderRoleName,
  renderRoleKeyTag,
  formatLocalTime,
} from "../utils";

describe("pages/system/role/utils", () => {
  describe("processTreeData", () => {
    it("基本字段映射", () => {
      const result = processTreeData([{ id: "1", title: "Root" }]);
      expect(result[0].key).toBe("1");
      expect(result[0].title).toBe("Root");
      expect(result[0].value).toBe("1");
      expect(result[0].children).toBeUndefined();
    });

    it("使用 key 字段而非 title", () => {
      const result = processTreeData([{ key: "k1", name: "n1" }], "key", "name");
      expect(result[0].key).toBe("k1");
      expect(result[0].title).toBe("n1");
      expect(result[0].value).toBe("k1");
    });

    it("递归 children", () => {
      const result = processTreeData([
        { id: "1", title: "P", children: [{ id: "2", title: "C" }] },
      ]);
      expect(result[0].children?.length).toBe(1);
      expect(result[0].children?.[0].key).toBe("2");
    });

    it("空 children 不渲染", () => {
      const result = processTreeData([{ id: "1", title: "X", children: [] }]);
      expect(result[0].children).toBeUndefined();
    });

    it("自定义 keyField/titleField", () => {
      const result = processTreeData([{ deptId: "d1", deptName: "Dept 1" }], "deptId", "deptName");
      expect(result[0].key).toBe("d1");
      expect(result[0].title).toBe("Dept 1");
    });
  });

  it("renderStatusTag 0 → 正常", () => {
    render(renderStatusTag(0));
    expect(screen.getByText("正常")).toBeInTheDocument();
  });

  it("renderStatusTag 1 → 停用", () => {
    render(renderStatusTag(1));
    expect(screen.getByText("停用")).toBeInTheDocument();
  });

  it("renderRoleName 含 icon", () => {
    const { container } = render(renderRoleName("Admin"));
    expect(container.querySelector(".anticon")).toBeTruthy();
    expect(screen.getByText("Admin")).toBeInTheDocument();
  });

  it("renderRoleKeyTag 蓝色 tag", () => {
    const { container } = render(renderRoleKeyTag("admin"));
    expect(container.querySelector(".ant-tag-blue")).toBeTruthy();
    expect(screen.getByText("admin")).toBeInTheDocument();
  });

  it("formatLocalTime", () => {
    expect(formatLocalTime("2026-08-31T10:00:00")).toMatch(/2026-08-31/);
  });
});
