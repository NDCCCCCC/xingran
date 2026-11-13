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
  it("VENDOR_OPTIONS 5 项", () => {
    expect(VENDOR_OPTIONS.length).toBe(5);
    expect(VENDOR_OPTIONS.map((o) => o.value)).toEqual([
      "huawei",
      "h3c",
      "ruijie",
      "maipu",
      "cisco",
    ]);
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

  // 三级回退口径：字典命中优先 → 静态映射 fallback → 原始值 / "-" 兜底
  it("renderDeviceTypeTag 字典命中的 label 优先于静态映射", () => {
    const dict = [
      { id: "1", dictValue: "router", dictLabel: "核心路由器" },
    ] as unknown as import("@/hooks/useDict").DictItem[];
    render(renderDeviceTypeTag("router", dict));
    expect(screen.getByText("核心路由器")).toBeInTheDocument();
    expect(screen.queryByText("路由器")).not.toBeInTheDocument();
  });

  it("renderDeviceTypeTag 管理员新增字典项不再渲染空 Tag", () => {
    const dict = [
      { id: "2", dictValue: "vpn", dictLabel: "VPN 网关" },
    ] as unknown as import("@/hooks/useDict").DictItem[];
    render(renderDeviceTypeTag("vpn", dict));
    expect(screen.getByText("VPN 网关")).toBeInTheDocument();
  });

  it("renderDeviceTypeTag 字典空态 → 静态映射 fallback", () => {
    render(renderDeviceTypeTag("switch", []));
    expect(screen.getByText("交换机")).toBeInTheDocument();
  });

  it("renderDeviceTypeTag 未知值 → 原始值兜底（不吞成空串）", () => {
    render(renderDeviceTypeTag("unknown", []));
    expect(screen.getByText("unknown")).toBeInTheDocument();
  });

  it("renderDeviceTypeTag 字典项 dictLabel 为空 → 不渲染空文案", () => {
    const dict = [
      { id: "3", dictValue: "switch", dictLabel: "" },
    ] as unknown as import("@/hooks/useDict").DictItem[];
    render(renderDeviceTypeTag("switch", dict));
    expect(screen.getByText("交换机")).toBeInTheDocument();
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
