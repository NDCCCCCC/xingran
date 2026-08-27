/**
 * Phase 84 84-01a Task 1 — NetworkExport 组件测试
 */
import { describe, it, expect, vi, beforeEach } from "vitest";
import { screen, fireEvent } from "@testing-library/react";

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import NetworkExport from "../NetworkExport";

describe("NetworkExport", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders export trigger button with icon (D-11)", () => {
    renderWithProviders(<NetworkExport entityType="device" entityName="设备" />);
    expect(screen.getByText("导出")).not.toBeNull();
  });

  it("passes entityType and entityName as props without error", () => {
    renderWithProviders(<NetworkExport entityType="ports" entityName="端口" />);
    expect(screen.getByText("导出")).not.toBeNull();
  });
});
