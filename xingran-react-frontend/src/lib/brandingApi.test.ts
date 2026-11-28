/**
 * brandingApi 契约测试（v1.56 PR #114 CI 覆盖率补齐）
 *
 * 锁定：4 个 brand.* 端点的路径与方法、normalizeBranding 的空值回退语义。
 * 存在意义：该文件此前 0 覆盖，把 lib/ 目录整体拉到 floor 以下（lib 93.8%）。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockGet = vi.fn();
const mockPost = vi.fn();
const mockPostFormData = vi.fn();
vi.mock("@/lib/api", () => ({
  get: (...args: unknown[]) => mockGet(...args),
  post: (...args: unknown[]) => mockPost(...args),
  postFormData: (...args: unknown[]) => mockPostFormData(...args),
}));

import { DEFAULT_BRANDING, brandingApi, normalizeBranding } from "./brandingApi";

describe("brandingApi 端点契约", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
    mockPostFormData.mockReset();
  });

  it("getPublic 走免认证 GET /system/branding/public", async () => {
    mockGet.mockResolvedValueOnce({ code: 0, data: DEFAULT_BRANDING });
    await brandingApi.getPublic();
    expect(mockGet).toHaveBeenCalledWith("/system/branding/public");
  });

  it("get 走管理端 GET /system/branding", async () => {
    mockGet.mockResolvedValueOnce({ code: 0, data: DEFAULT_BRANDING });
    await brandingApi.get();
    expect(mockGet).toHaveBeenCalledWith("/system/branding");
  });

  it("update 走 POST /system/branding/update 并透传键值", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: { message: "ok" } });
    await brandingApi.update({ "brand.siteName": "新站名" });
    expect(mockPost).toHaveBeenCalledWith("/system/branding/update", {
      "brand.siteName": "新站名",
    });
  });

  it("uploadLogo 用 multipart 提交 file 字段", async () => {
    mockPostFormData.mockResolvedValueOnce({ code: 0, data: { logoUrl: "/logo.png?v=1" } });
    const file = new File(["x"], "logo.png", { type: "image/png" });
    const res = await brandingApi.uploadLogo(file);
    expect(mockPostFormData).toHaveBeenCalledWith("/system/branding/logo", expect.any(FormData));
    expect(res.data?.logoUrl).toBe("/logo.png?v=1");
  });
});

describe("normalizeBranding 空值回退", () => {
  it("传 null 时全量回落默认值", () => {
    expect(normalizeBranding(null)).toEqual(DEFAULT_BRANDING);
  });

  it("传 undefined 时全量回落默认值", () => {
    expect(normalizeBranding(undefined)).toEqual(DEFAULT_BRANDING);
  });

  it("空串字段不得覆盖默认值（防登录页空白）", () => {
    const out = normalizeBranding({ siteName: "", siteMark: "", tagline: "" });
    expect(out.siteName).toBe(DEFAULT_BRANDING.siteName);
    expect(out.siteMark).toBe(DEFAULT_BRANDING.siteMark);
    expect(out.tagline).toBe(DEFAULT_BRANDING.tagline);
  });

  it("logoUrl 空串是合法值（不清 logo 也不回退）", () => {
    expect(normalizeBranding({ logoUrl: "" }).logoUrl).toBe("");
  });

  it("有值字段原样保留", () => {
    const out = normalizeBranding({ siteName: "运维平台", copyright: "© 2027" });
    expect(out.siteName).toBe("运维平台");
    expect(out.copyright).toBe("© 2027");
  });
});
