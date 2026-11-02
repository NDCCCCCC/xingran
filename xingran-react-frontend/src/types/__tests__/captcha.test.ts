/**
 * Phase 88 Batch204 — types/captcha 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import type {
  CaptchaType,
  CaptchaEnabled,
  PieceShape,
  DifficultyLevel,
  CaptchaBackgroundStatus,
  BackgroundMode,
} from "../captcha";

describe("types/captcha", () => {
  it("CaptchaType 文字/滑动", () => {
    const types: CaptchaType[] = ["normal", "slider"];
    expect(types.length).toBe(2);
  });

  it("CaptchaEnabled disabled/normal/slider", () => {
    const enabled: CaptchaEnabled[] = ["disabled", "normal", "slider"];
    expect(enabled.length).toBe(3);
  });

  it("PieceShape 4 种", () => {
    const shapes: PieceShape[] = ["circle", "square", "star", "heart"];
    expect(shapes.length).toBe(4);
  });

  it("DifficultyLevel 1/2/3", () => {
    const levels: DifficultyLevel[] = [1, 2, 3];
    expect(levels.length).toBe(3);
  });

  it("CaptchaBackgroundStatus 0/1", () => {
    const s0: CaptchaBackgroundStatus = 0;
    const s1: CaptchaBackgroundStatus = 1;
    expect(s0).toBe(0);
    expect(s1).toBe(1);
  });

  it("BackgroundMode auto/custom/mixed", () => {
    const modes: BackgroundMode[] = ["auto", "custom", "mixed"];
    expect(modes.length).toBe(3);
  });
});
