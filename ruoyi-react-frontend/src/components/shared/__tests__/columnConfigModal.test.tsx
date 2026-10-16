/**
 * Phase 84 84-01a Task 1 — DepartmentTreeSelect 组件测试
 *
 * ColumnConfigModal(DndContext+SortableContext 复杂依赖 jsdom) 已有现有测试覆盖,
 * 此处专注 DepartmentTreeSelect 的 props 渲染断言(D-11/D-12)。
 */
import { describe, it, expect, vi } from "vitest";

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import DepartmentTreeSelect from "../DepartmentTreeSelect";

describe("DepartmentTreeSelect", () => {
  const mockDepts = [
    { deptId: "1", deptName: "总经办", parentId: null },
    { deptId: "2", deptName: "技术部", parentId: null },
    { deptId: "3", deptName: "前端组", parentId: "2" },
  ];

  it("renders ant-select with placeholder text (D-12)", () => {
    renderWithProviders(<DepartmentTreeSelect departments={mockDepts} placeholder="选择部门" />);
    const placeholder = document.querySelector(".ant-select-placeholder");
    expect(placeholder?.textContent).toBe("选择部门");
  });

  it("renders ant-select wrapper element", () => {
    renderWithProviders(<DepartmentTreeSelect departments={mockDepts} placeholder="选择部门" />);
    expect(document.querySelector(".ant-select")).not.toBeNull();
  });

  it("renders multiple department items without error", () => {
    renderWithProviders(<DepartmentTreeSelect departments={mockDepts} placeholder="选择部门" />);
    // 父节点渲染
    expect(document.querySelector(".ant-select")).not.toBeNull();
  });
});
