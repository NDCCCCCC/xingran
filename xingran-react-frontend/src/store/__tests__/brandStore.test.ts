/**
 * brandStore 单测 — 站点品牌前端消费层
 *
 * 覆盖：
 *   - 拉取成功：用后端返回值覆盖 DEFAULT_BRANDING，loaded=true
 *   - 拉取失败：回落 DEFAULT_BRANDING（登录页永不空白），loaded 仍置 true（不无限重试）
 *   - 部分字段为空：normalizeBranding 逐字段回退默认值
 *   - refresh()：先置 loaded=false 再重新拉取
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

const getPublic = vi.fn();

vi.mock("@/lib/brandingApi", async () => {
  const actual = await vi.importActual<typeof import("@/lib/brandingApi")>("@/lib/brandingApi");
  return {
    ...actual,
    brandingApi: {
      getPublic: (...args: unknown[]) => getPublic(...args),
      get: vi.fn(),
      update: vi.fn(),
      uploadLogo: vi.fn(),
    },
  };
});

import { DEFAULT_BRANDING } from "@/lib/brandingApi";
import { useBrandStore } from "../brandStore";

describe("brandStore", () => {
  beforeEach(() => {
    getPublic.mockReset();
    useBrandStore.setState({ config: DEFAULT_BRANDING, loaded: false });
  });

  it("拉取成功 → 用后端值覆盖默认值", async () => {
    getPublic.mockResolvedValue({
      code: 0,
      data: {
        siteName: "云运维平台",
        siteMark: "云",
        logoUrl: "",
        tagline: "自研调度",
        copyright: "© 2027 云运维",
        subtitle: "请登录",
      },
    });

    await useBrandStore.getState().fetchPublic();

    const state = useBrandStore.getState();
    expect(state.loaded).toBe(true);
    expect(state.config.siteName).toBe("云运维平台");
    expect(state.config.siteMark).toBe("云");
    expect(state.config.copyright).toBe("© 2027 云运维");
  });

  it("拉取失败 → 回落 DEFAULT_BRANDING 且 loaded=true（不阻断 UI）", async () => {
    getPublic.mockRejectedValue(new Error("network down"));

    await useBrandStore.getState().fetchPublic();

    const state = useBrandStore.getState();
    expect(state.loaded).toBe(true);
    expect(state.config).toEqual(DEFAULT_BRANDING);
  });

  it("后端返回空字段 → 逐字段回退默认值", async () => {
    getPublic.mockResolvedValue({
      code: 0,
      data: { siteName: "", siteMark: "", logoUrl: "", tagline: "", copyright: "", subtitle: "" },
    });

    await useBrandStore.getState().fetchPublic();

    const cfg = useBrandStore.getState().config;
    expect(cfg.siteName).toBe(DEFAULT_BRANDING.siteName);
    expect(cfg.siteMark).toBe(DEFAULT_BRANDING.siteMark);
    expect(cfg.tagline).toBe(DEFAULT_BRANDING.tagline);
    expect(cfg.logoUrl).toBe(""); // logo 允许为空（无图时前端显示方块字）
  });

  it("refresh() → 重新拉取并应用最新值", async () => {
    getPublic.mockResolvedValue({
      code: 0,
      data: { siteName: "改名后", siteMark: "改" },
    });

    await useBrandStore.getState().refresh();

    expect(useBrandStore.getState().loaded).toBe(true);
    expect(useBrandStore.getState().config.siteName).toBe("改名后");
    expect(getPublic).toHaveBeenCalledTimes(1);
  });
});
