/**
 * Phase 88 Batch206 — services/captcha 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const actual = await vi.importActual<typeof import("@/lib/api")>("@/lib/api");
  return {
    ...actual,
    post: vi.fn(async (url: string) => ({ data: { url, ok: true } })),
    postFormData: vi.fn(async (url: string) => ({ data: { url, ok: true } })),
  };
});

import * as api from "@/lib/api";
import {
  getCaptcha,
  getCaptchaConfig,
  verifySliderCaptcha,
  getCaptchaBackgroundList,
  uploadCaptchaBackground,
  getCaptchaBackground,
  updateCaptchaBackground,
  deleteCaptchaBackground,
  toggleCaptchaBackgroundStatus,
  getCaptchaBackgroundStatistics,
  preloadCaptchaCache,
} from "../captcha";

describe("services/captcha", () => {
  it("getCaptcha", async () => {
    const r = await getCaptcha();
    expect((r as any).url).toContain("/system/auth/captcha");
  });

  it("getCaptchaConfig", async () => {
    const r = await getCaptchaConfig();
    expect((r as any).url).toContain("/captcha/config");
  });

  it("verifySliderCaptcha", async () => {
    const r = await verifySliderCaptcha({ captchaId: "c1", xPos: 100, token: "t1" });
    expect((r as any).url).toContain("/verify/slider");
  });

  it("getCaptchaBackgroundList", async () => {
    const r = await getCaptchaBackgroundList({ current: 1, pageSize: 10 });
    expect((r as any).url).toContain("/list");
  });

  it("uploadCaptchaBackground 必填字段", async () => {
    const file = new File(["content"], "bg.png", { type: "image/png" });
    const r = await uploadCaptchaBackground(file, {
      pieceShape: "circle",
      difficultyLevel: 2,
    });
    expect((r as any).url).toContain("/upload");
  });

  it("uploadCaptchaBackground allowedShapes + remark", async () => {
    const file = new File(["x"], "bg.png", { type: "image/png" });
    const r = await uploadCaptchaBackground(file, {
      pieceShape: "star",
      difficultyLevel: 3,
      allowedShapes: ["circle", "square"],
      remark: "test",
    });
    expect((r as any).url).toContain("/upload");
  });

  it("getCaptchaBackground 详情", async () => {
    const r = await getCaptchaBackground("bg1");
    expect((r as any).url).toContain("/bg1");
  });

  it("updateCaptchaBackground", async () => {
    const r = await updateCaptchaBackground("bg1", { sortOrder: 5 });
    expect((r as any).url).toContain("/update");
  });

  it("deleteCaptchaBackground", async () => {
    const r = await deleteCaptchaBackground("bg1");
    expect((r as any).url).toContain("/delete");
  });

  it("toggleCaptchaBackgroundStatus", async () => {
    const r = await toggleCaptchaBackgroundStatus("bg1");
    expect((r as any).url).toContain("/toggle");
  });

  it("getCaptchaBackgroundStatistics", async () => {
    const r = await getCaptchaBackgroundStatistics();
    expect((r as any).url).toContain("/statistics");
  });

  it("preloadCaptchaCache", async () => {
    const r = await preloadCaptchaCache();
    expect((r as any).url).toContain("/preload");
  });

  it("调用 mock post/postFormData", async () => {
    const postSpy = vi.mocked(api.post);
    await getCaptcha();
    expect(postSpy).toHaveBeenCalled();
  });
});
