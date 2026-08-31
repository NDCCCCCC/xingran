/**
 * Phase 88 Batch343 — services/captcha 测试
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  return {
    post: vi.fn(async (url: string, data?: any) => ({
      data: { url, payload: data, success: true },
    })),
    postFormData: vi.fn(async (url: string, formData: FormData) => ({
      data: { url, payload: formData },
    })),
  };
});

import { post, postFormData } from "@/lib/api";
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
  it("getCaptcha 调用 /system/auth/captcha", async () => {
    await getCaptcha();
    expect(post).toHaveBeenCalledWith("/system/auth/captcha", {});
  });

  it("getCaptchaConfig 调用 /config", async () => {
    await getCaptchaConfig();
    expect(post).toHaveBeenCalledWith("/system/auth/captcha/config", {});
  });

  it("verifySliderCaptcha 调用 /verify/slider", async () => {
    await verifySliderCaptcha({ sliderX: 100, sliderY: 50 } as any);
    expect(post).toHaveBeenCalledWith("/system/auth/captcha/verify/slider", {
      sliderX: 100,
      sliderY: 50,
    });
  });

  it("getCaptchaBackgroundList 调用 /list", async () => {
    await getCaptchaBackgroundList({ current: 1, pageSize: 10 });
    expect(post).toHaveBeenCalledWith("/system/captcha-backgrounds/list", {
      current: 1,
      pageSize: 10,
    });
  });

  it("uploadCaptchaBackground 调用 postFormData + FormData", async () => {
    const file = new File(["content"], "bg.png", { type: "image/png" });
    await uploadCaptchaBackground(file, {
      pieceShape: "square",
      difficultyLevel: 2,
    });
    expect(postFormData).toHaveBeenCalled();
    const callArgs =
      vi.mocked(postFormData).mock.calls[vi.mocked(postFormData).mock.calls.length - 1];
    expect(callArgs[0]).toBe("/system/captcha-backgrounds/upload");
    expect(callArgs[1]).toBeInstanceOf(FormData);
  });

  it("uploadCaptchaBackground 含 allowedShapes + remark", async () => {
    const file = new File(["x"], "bg.png");
    await uploadCaptchaBackground(file, {
      pieceShape: "circle",
      difficultyLevel: 3,
      allowedShapes: ["square", "circle"],
      remark: "test",
    });
    expect(postFormData).toHaveBeenCalled();
  });

  it("getCaptchaBackground id", async () => {
    await getCaptchaBackground("bg-1");
    expect(post).toHaveBeenCalledWith("/system/captcha-backgrounds/bg-1", {});
  });

  it("updateCaptchaBackground id + data", async () => {
    await updateCaptchaBackground("bg-2", { pieceShape: "circle" } as any);
    expect(post).toHaveBeenCalledWith("/system/captcha-backgrounds/bg-2/update", {
      pieceShape: "circle",
    });
  });

  it("deleteCaptchaBackground id", async () => {
    await deleteCaptchaBackground("bg-3");
    expect(post).toHaveBeenCalledWith("/system/captcha-backgrounds/bg-3/delete", {});
  });

  it("toggleCaptchaBackgroundStatus id", async () => {
    await toggleCaptchaBackgroundStatus("bg-4");
    expect(post).toHaveBeenCalledWith("/system/captcha-backgrounds/bg-4/toggle", {});
  });

  it("getCaptchaBackgroundStatistics", async () => {
    await getCaptchaBackgroundStatistics();
    expect(post).toHaveBeenCalledWith("/system/captcha-backgrounds/statistics", {});
  });

  it("preloadCaptchaCache", async () => {
    await preloadCaptchaCache();
    expect(post).toHaveBeenCalledWith("/system/captcha-backgrounds/preload", {});
  });
});
