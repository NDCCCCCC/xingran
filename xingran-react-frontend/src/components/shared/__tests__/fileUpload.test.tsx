/**
 * Phase 84 84-01a Task 1 — FileUpload 组件测试
 * antd Upload picture-card 形态 + props 渲染断言(D-12)
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import FileUpload from "../FileUpload";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

describe("FileUpload", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders picture-card upload trigger with plus icon (default)", () => {
    const onChange = vi.fn();
    renderWithProviders(
      <FileUpload maxCount={3} maxSize={10} accept=".jpg,.png" onChange={onChange} />
    );
    // picture-card 默认形态:PlusOutlined 图标
    expect(document.querySelector(".anticon-plus")).not.toBeNull();
    expect(document.querySelector(".ant-upload")).not.toBeNull();
  });

  it("renders text list type when listType=text", () => {
    renderWithProviders(<FileUpload maxCount={1} listType="text" accept=".xlsx" />);
    expect(document.querySelector(".ant-upload-select")).not.toBeNull();
  });

  it("renders upload button variant with disabled prop", () => {
    renderWithProviders(<FileUpload maxCount={2} listType="text" disabled accept="image/*" />);
    const select = document.querySelector(".ant-upload-select");
    expect(select).not.toBeNull();
    // disabled 时 antd Upload 有 ant-upload-disabled 类
    expect(document.querySelector(".ant-upload-disabled")).not.toBeNull();
  });

  it("renders with category prop without error", () => {
    renderWithProviders(
      <FileUpload maxCount={1} maxSize={5} accept="image/*" category="floor-plan" />
    );
    expect(document.querySelector(".ant-upload")).not.toBeNull();
  });
});
