/**
 * Phase 88 Batch31 — services 层 API 封装测试(captcha + operations 5 个 0% 文件)
 *
 * 纯 URL 映射断言:每个 api 方法调用应触发对应端点的 post。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import * as apiModule from "@/lib/api";
import captchaService from "../captcha";
import {
  getCaptcha,
  getCaptchaConfig,
  verifySliderCaptcha,
  getCaptchaBackgroundList,
  getCaptchaBackground,
  updateCaptchaBackground,
  deleteCaptchaBackground,
  toggleCaptchaBackgroundStatus,
  getCaptchaBackgroundStatistics,
  preloadCaptchaCache,
} from "../captcha";
import { floorApi } from "../operations/floors";
import { buildingApi } from "../operations/buildings";
import { dedicatedLineApi } from "../operations/dedicated-lines";
import { serverRoomApi } from "../operations/server-rooms";
import { workstationApi } from "../operations/workstations";

const postSpy = vi.fn();
const postFormDataSpy = vi.fn();

beforeEach(() => {
  postSpy.mockReset().mockResolvedValue({ data: {} });
  postFormDataSpy.mockReset().mockResolvedValue({ data: {} });
  vi.spyOn(apiModule, "post" as any).mockImplementation(postSpy as any);
  vi.spyOn(apiModule, "postFormData" as any).mockImplementation(postFormDataSpy as any);
});

describe("services/captcha", () => {
  it("getCaptcha → /system/auth/captcha", async () => {
    postSpy.mockResolvedValue({ data: { captchaId: "c1" } });
    const r = await getCaptcha();
    expect(postSpy).toHaveBeenCalledWith("/system/auth/captcha", {});
    expect(r.captchaId).toBe("c1");
  });

  it("getCaptchaConfig → /system/auth/captcha/config", async () => {
    postSpy.mockResolvedValue({ data: { type: "slider" } });
    await getCaptchaConfig();
    expect(postSpy).toHaveBeenCalledWith("/system/auth/captcha/config", {});
  });

  it("verifySliderCaptcha → /system/auth/captcha/verify/slider", async () => {
    postSpy.mockResolvedValue({ data: { success: true } });
    const r = await verifySliderCaptcha({ captchaId: "c1", x: 100 });
    expect(postSpy).toHaveBeenCalledWith("/system/auth/captcha/verify/slider", {
      captchaId: "c1",
      x: 100,
    });
    expect(r.success).toBe(true);
  });

  it("getCaptchaBackgroundList → /system/captcha-backgrounds/list", async () => {
    postSpy.mockResolvedValue({ data: { list: [], total: 0 } });
    await getCaptchaBackgroundList({ current: 1, pageSize: 10 });
    expect(postSpy).toHaveBeenCalledWith("/system/captcha-backgrounds/list", {
      current: 1,
      pageSize: 10,
    });
  });

  it("uploadCaptchaBackground → postFormData 带字段", async () => {
    postFormDataSpy.mockResolvedValue({ data: { id: "bg1" } });
    const file = new File(["x"], "bg.png");
    const { uploadCaptchaBackground } = await import("../captcha");
    await uploadCaptchaBackground(file, {
      pieceShape: "square",
      difficultyLevel: 2,
      allowedShapes: ["square"],
      remark: "test",
    });
    expect(postFormDataSpy).toHaveBeenCalledWith(
      "/system/captcha-backgrounds/upload",
      expect.any(FormData)
    );
    const fd = postFormDataSpy.mock.calls[0][1] as FormData;
    expect(fd.get("pieceShape")).toBe("square");
    expect(fd.get("difficultyLevel")).toBe("2");
    expect(fd.get("remark")).toBe("test");
  });

  it("getCaptchaBackground → /system/captcha-backgrounds/:id", async () => {
    await getCaptchaBackground("bg1");
    expect(postSpy).toHaveBeenCalledWith("/system/captcha-backgrounds/bg1", {});
  });

  it("updateCaptchaBackground → :id/update", async () => {
    await updateCaptchaBackground("bg1", { remark: "new" });
    expect(postSpy).toHaveBeenCalledWith("/system/captcha-backgrounds/bg1/update", { remark: "new" });
  });

  it("deleteCaptchaBackground → :id/delete", async () => {
    await deleteCaptchaBackground("bg1");
    expect(postSpy).toHaveBeenCalledWith("/system/captcha-backgrounds/bg1/delete", {});
  });

  it("toggleCaptchaBackgroundStatus → :id/toggle", async () => {
    await toggleCaptchaBackgroundStatus("bg1");
    expect(postSpy).toHaveBeenCalledWith("/system/captcha-backgrounds/bg1/toggle", {});
  });

  it("getCaptchaBackgroundStatistics → /statistics", async () => {
    await getCaptchaBackgroundStatistics();
    expect(postSpy).toHaveBeenCalledWith("/system/captcha-backgrounds/statistics", {});
  });

  it("preloadCaptchaCache → /preload", async () => {
    await preloadCaptchaCache();
    expect(postSpy).toHaveBeenCalledWith("/system/captcha-backgrounds/preload", {});
  });

  it("default export 汇总 11 个方法", () => {
    expect(Object.keys(captchaService).length).toBe(11);
  });
});

describe("services/operations floors/buildings/dedicated-lines/server-rooms/workstations", () => {
  it("floorApi 全方法 URL 映射", async () => {
    await floorApi.list({ current: 1, pageSize: 10 });
    expect(postSpy).toHaveBeenLastCalledWith("/ops/floors/list", { current: 1, pageSize: 10 });

    await floorApi.getTree();
    expect(postSpy).toHaveBeenLastCalledWith("/ops/floors/tree");

    await floorApi.create({ name: "1F" } as any);
    expect(postSpy).toHaveBeenLastCalledWith("/ops/floors", { name: "1F" });

    await floorApi.update("f1", { name: "1F改" });
    expect(postSpy).toHaveBeenLastCalledWith("/ops/floors/f1/update", { name: "1F改" });

    await floorApi.delete("f1");
    expect(postSpy).toHaveBeenLastCalledWith("/ops/floors/f1/delete");

    await floorApi.batchDelete(["f1", "f2"]);
    expect(postSpy).toHaveBeenLastCalledWith("/ops/floors/batch", {
      ids: ["f1", "f2"],
      action: "delete",
    });

    await floorApi.downloadTemplate();
    expect(postSpy).toHaveBeenLastCalledWith("/ops/floors/template", {});
  });

  it("buildingApi 全方法 URL 映射", async () => {
    await buildingApi.list({ current: 1, pageSize: 10 });
    expect(postSpy).toHaveBeenLastCalledWith("/ops/buildings/list", { current: 1, pageSize: 10 });

    await buildingApi.delete("b1");
    expect(postSpy).toHaveBeenLastCalledWith("/ops/buildings/b1/delete");

    await buildingApi.batchDelete(["b1"]);
    expect(postSpy).toHaveBeenLastCalledWith("/ops/buildings/batch", { ids: ["b1"], action: "delete" });
  });

  it("dedicatedLineApi 方法 URL 映射", async () => {
    await dedicatedLineApi.list({ current: 1, pageSize: 10 });
    const url = postSpy.mock.calls[postSpy.mock.calls.length - 1][0];
    expect(url).toMatch(/dedicated-line/);
  });

  it("serverRoomApi 方法 URL 映射", async () => {
    await serverRoomApi.list({ current: 1, pageSize: 10 });
    const url = postSpy.mock.calls[postSpy.mock.calls.length - 1][0];
    expect(url).toMatch(/server-room/);
  });

  it("workstationApi 方法 URL 映射", async () => {
    await workstationApi.list({ current: 1, pageSize: 10 });
    const url = postSpy.mock.calls[postSpy.mock.calls.length - 1][0];
    expect(url).toMatch(/workstation/);
  });
});
