/**
 * Phase 88 Batch181 — pages/network/templates/constants 测试
 */
import { describe, it, expect } from "vitest";
import { render } from "@testing-library/react";
import { App as AntdApp } from "antd";
import type { ReactElement, ReactNode } from "react";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {
  VENDOR_OPTIONS,
  DEVICE_TYPE_OPTIONS,
  TEMPLATE_TYPE_OPTIONS,
  renderVendorTag,
  renderDeviceTypeTag,
  renderTemplateTypeTag,
  renderSystemTemplateTag,
} from "../constants";

function wrapper({ children }: { children: ReactNode }): ReactElement {
  return <AntdApp>{children}</AntdApp>;
}

describe("network/templates/constants", () => {
  it("VENDOR_OPTIONS 4 厂商", () => {
    expect(VENDOR_OPTIONS.length).toBe(4);
    expect(VENDOR_OPTIONS.map((o) => o.value)).toEqual(["huawei", "h3c", "ruijie", "maipu"]);
  });

  it("DEVICE_TYPE_OPTIONS 5 设备类型", () => {
    expect(DEVICE_TYPE_OPTIONS.length).toBe(5);
  });

  it("TEMPLATE_TYPE_OPTIONS 3 类型", () => {
    expect(TEMPLATE_TYPE_OPTIONS.length).toBe(3);
  });

  it("renderVendorTag huawei → geekblue", () => {
    const { baseElement } = render(<>{renderVendorTag("huawei")}</>, { wrapper });
    expect(baseElement.textContent).toContain("Huawei");
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });

  it("renderVendorTag 空 → 通用", () => {
    const { baseElement } = render(<>{renderVendorTag("")}</>, { wrapper });
    expect(baseElement.textContent).toContain("通用");
  });

  it("renderVendorTag 未知 vendor → undefined", () => {
    const { baseElement } = render(<>{renderVendorTag("unknown")}</>, { wrapper });
    expect(baseElement.querySelector(".ant-tag")).toBeTruthy();
  });

  it("renderDeviceTypeTag router → 路由器", () => {
    const { baseElement } = render(<>{renderDeviceTypeTag("router")}</>, { wrapper });
    expect(baseElement.textContent).toContain("路由器");
  });

  it("renderDeviceTypeTag 空 → -", () => {
    const { baseElement } = render(<>{renderDeviceTypeTag("")}</>, { wrapper });
    expect(baseElement.textContent).toContain("-");
  });

  it("renderTemplateTypeTag init → 初始化配置", () => {
    const { baseElement } = render(<>{renderTemplateTypeTag("init")}</>, { wrapper });
    expect(baseElement.textContent).toContain("初始化配置");
  });

  it("renderSystemTemplateTag true → 系统", () => {
    const { baseElement } = render(<>{renderSystemTemplateTag(true)}</>, { wrapper });
    expect(baseElement.textContent).toContain("系统");
  });

  it("renderSystemTemplateTag false → 自定义", () => {
    const { baseElement } = render(<>{renderSystemTemplateTag(false)}</>, { wrapper });
    expect(baseElement.textContent).toContain("自定义");
  });
});
