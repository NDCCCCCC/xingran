/**
 * Phase 68 DEBT-02a — statusHelpers 单测
 * 6 域 × 全 status 值的 label/color 断言 + 未知值 + workstation 三态 + visible 反转
 */
import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { statusLabel, statusColor, StatusTag, UNKNOWN_STATUS_LABEL } from "../statusHelpers";

describe("statusLabel", () => {
  it("enable 域：0=启用 / 1=停用（去「已」前缀、不写「禁用」）", () => {
    expect(statusLabel(0, "enable")).toBe("启用");
    expect(statusLabel(1, "enable")).toBe("停用");
  });

  it("online 域：0=在线 / 1=离线", () => {
    expect(statusLabel(0, "online")).toBe("在线");
    expect(statusLabel(1, "online")).toBe("离线");
  });

  it("result 域：0=成功 / 1=失败", () => {
    expect(statusLabel(0, "result")).toBe("成功");
    expect(statusLabel(1, "result")).toBe("失败");
  });

  it("health 域：0=正常 / 1=异常", () => {
    expect(statusLabel(0, "health")).toBe("正常");
    expect(statusLabel(1, "health")).toBe("异常");
  });

  it("workstation 三态：0=空闲 / 1=占用 / 2=维护", () => {
    expect(statusLabel(0, "workstation")).toBe("空闲");
    expect(statusLabel(1, "workstation")).toBe("占用");
    expect(statusLabel(2, "workstation")).toBe("维护");
  });

  it("visible 反转例外：1=显示 / 0=隐藏", () => {
    expect(statusLabel(1, "visible")).toBe("显示");
    expect(statusLabel(0, "visible")).toBe("隐藏");
  });
});

describe("statusColor", () => {
  it("两态域正向 success / 负向 error", () => {
    for (const domain of ["enable", "online", "result", "health"] as const) {
      expect(statusColor(0, domain)).toBe("success");
      expect(statusColor(1, domain)).toBe("error");
    }
  });

  it("workstation：空闲 success / 占用 error / 维护 processing", () => {
    expect(statusColor(0, "workstation")).toBe("success");
    expect(statusColor(1, "workstation")).toBe("error");
    expect(statusColor(2, "workstation")).toBe("processing");
  });

  it("visible：显示 success，隐藏 default（隐藏非 error，不反转错色）", () => {
    expect(statusColor(1, "visible")).toBe("success");
    expect(statusColor(0, "visible")).toBe("default");
  });

  it("未知值不 crash：返回「未知」+ default（enable 域传 2 等）", () => {
    expect(statusLabel(2, "enable")).toBe(UNKNOWN_STATUS_LABEL);
    expect(statusLabel(2, "enable")).toBe("未知");
    expect(statusColor(2, "enable")).toBe("default");
    expect(statusLabel(-1, "result")).toBe("未知");
    expect(statusColor(99, "online")).toBe("default");
    expect(statusLabel(3, "workstation")).toBe("未知");
  });
});

describe("StatusTag", () => {
  it("渲染 Tag：文案 + 语义色", () => {
    render(<StatusTag status={0} domain="enable" />);
    const tag = screen.getByText("启用");
    expect(tag).toBeDefined();
    expect(tag.className).toContain("ant-tag-success");
  });

  it("未知 status 渲染「未知」+ default 色，不 crash", () => {
    render(<StatusTag status={42} domain="health" />);
    const tag = screen.getByText("未知");
    expect(tag).toBeDefined();
    expect(tag.className).toContain("ant-tag-default");
  });

  it("visible 反转域渲染正确", () => {
    render(<StatusTag status={1} domain="visible" />);
    expect(screen.getByText("显示")).toBeDefined();
  });
});
