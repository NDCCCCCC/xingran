/**
 * Phase 88 Batch331 — pages/network/templates/constants 测试
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import {
  VENDOR_OPTIONS,
  DEVICE_TYPE_OPTIONS,
  TEMPLATE_TYPE_OPTIONS,
  renderVendorTag,
  renderDeviceTypeTag,
  renderTemplateTypeTag,
  renderSystemTemplateTag,
} from "../constants";

describe("pages/network/templates/constants", () => {
  it("VENDOR_OPTIONS 4 项", () => {
    expect(VENDOR_OPTIONS.length).toBe(4);
    expect(VENDOR_OPTIONS.map((o) => o.value)).toEqual(["huawei", "h3c", "ruijie", "maipu"]);
  });

  it("DEVICE_TYPE_OPTIONS 5 项", () => {
    expect(DEVICE_TYPE_OPTIONS.length).toBe(5);
    expect(DEVICE_TYPE_OPTIONS[0].value).toBe("router");
  });

  it("TEMPLATE_TYPE_OPTIONS 3 项", () => {
    expect(TEMPLATE_TYPE_OPTIONS.length).toBe(3);
  });

  it("renderVendorTag huawei", () => {
    render(renderVendorTag("huawei"));
    expect(screen.getByText("Huawei")).toBeInTheDocument();
  });

  it("renderVendorTag 空 → 通用", () => {
    render(renderVendorTag(""));
    expect(screen.getByText("通用")).toBeInTheDocument();
  });

  it("renderVendorTag 未知 vendor → undefined text", () => {
    const { container } = render(renderVendorTag("unknown"));
    expect(container.querySelector(".ant-tag")).toBeTruthy();
  });

  it("renderDeviceTypeTag switch", () => {
    render(renderDeviceTypeTag("switch"));
    expect(screen.getByText("交换机")).toBeInTheDocument();
  });

  it("renderDeviceTypeTag 空 → '-'", () => {
    render(renderDeviceTypeTag(""));
    expect(screen.getByText("-")).toBeInTheDocument();
  });

  it("renderTemplateTypeTag init", () => {
    render(renderTemplateTypeTag("init"));
    expect(screen.getByText("初始化配置")).toBeInTheDocument();
  });

  it("renderSystemTemplateTag true → 系统", () => {
    render(renderSystemTemplateTag(true));
    expect(screen.getByText("系统")).toBeInTheDocument();
  });

  it("renderSystemTemplateTag false → 自定义", () => {
    render(renderSystemTemplateTag(false));
    expect(screen.getByText("自定义")).toBeInTheDocument();
  });
});
