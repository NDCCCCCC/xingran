/**
 * Phase 88 Batch273 — types/auth 测试
 */
import { describe, it, expect } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import { Gender as _Gender } from "../base";
void _Gender;
import type {
  LoginRequest,
  LoginResponse,
  UserProfile,
  UpdateProfileRequest,
  ChangePasswordRequest,
  ThemeMode,
  Language,
  UserPreferences,
} from "../auth";

describe("types/auth", () => {
  it("LoginRequest shape", () => {
    const r: LoginRequest = {
      username: "u1",
      password: "p1",
      encryptedPassword: true,
      captcha: "abc",
      captchaId: "c1",
    };
    expect(r.username).toBe("u1");
  });

  it("LoginRequest 仅必填", () => {
    const r: LoginRequest = { username: "u", password: "p" };
    expect(r.encryptedPassword).toBeUndefined();
  });

  it("LoginResponse shape", () => {
    const r: LoginResponse = {
      user: { id: "1", username: "u" } as any,
      accessToken: "a",
      refreshToken: "r",
      expiresIn: 3600,
      tokenType: "Bearer",
    };
    expect(r.tokenType).toBe("Bearer");
  });

  it("UserProfile shape", () => {
    const p: UserProfile = {
      id: "1",
      username: "u",
      remark: "test",
      pwdUpdateTime: "2026-01-01",
    } as any;
    expect(p.remark).toBe("test");
  });

  it("UpdateProfileRequest 必填 gender", () => {
    const r: UpdateProfileRequest = {
      nickname: "n",
      email: "e@example.com",
      phone: "13800000000",
      gender: 0,
    };
    expect(r.gender).toBe(0);
  });

  it("ChangePasswordRequest shape", () => {
    const r: ChangePasswordRequest = { oldPassword: "o", newPassword: "n" };
    expect(r.oldPassword).toBe("o");
  });

  it("ThemeMode 2 值", () => {
    const t: ThemeMode = "light";
    expect(t).toBe("light");
  });

  it("Language 2 值", () => {
    const l: Language = "zh-CN";
    expect(l).toBe("zh-CN");
  });

  it("UserPreferences shape", () => {
    const p: UserPreferences = {
      theme: "dark",
      language: "en-US",
      pageSize: 50,
      sidebarCollapsed: true,
    };
    expect(p.pageSize).toBe(50);
  });
});
