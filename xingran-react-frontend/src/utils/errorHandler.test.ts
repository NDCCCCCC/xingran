import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const { mockGetAppMessage, mockMessageError, mockMessageSuccess, mockClearPublicKeyCache } =
  vi.hoisted(() => {
    const mockMessageError = vi.fn();
    const mockMessageSuccess = vi.fn();
    return {
      mockMessageError,
      mockMessageSuccess,
      mockGetAppMessage: () => ({
        error: mockMessageError,
        success: mockMessageSuccess,
      }),
      mockClearPublicKeyCache: vi.fn(),
    };
  });

vi.mock("@/utils/antdMessage", () => ({
  getAppMessage: mockGetAppMessage,
}));

vi.mock("@/utils/sm2", () => ({
  fetchPublicKey: vi.fn(),
  clearPublicKeyCache: mockClearPublicKeyCache,
}));

import {
  ErrorHandler,
  HttpErrorType,
  handleApiError,
  handleSuccess,
  handleHttpResponseError,
  handleNetworkError,
  handleParseError,
  isFormValidationError,
  safeAsync,
  safeSync,
  withErrorHandling,
} from "./errorHandler";

describe("handleHttpResponseError", () => {
  beforeEach(() => {
    mockMessageError.mockReset();
    mockClearPublicKeyCache.mockReset();
    vi.spyOn(console, "warn").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("400 使用默认参数错误文案并携带 status/type", () => {
    const error = handleHttpResponseError(400);
    expect(error.message).toBe("请求参数错误");
    expect((error as { status?: number }).status).toBe(400);
    expect((error as { type?: HttpErrorType }).type).toBe(HttpErrorType.BAD_REQUEST);
    expect(mockMessageError).toHaveBeenCalledWith("请求参数错误");
  });

  it("responseData.message 优先于默认文案", () => {
    const error = handleHttpResponseError(404, { message: "工位不存在" });
    expect(error.message).toBe("工位不存在");
    expect((error as { type?: HttpErrorType }).type).toBe(HttpErrorType.NOT_FOUND);
  });

  it("500 映射 INTERNAL_ERROR", () => {
    const error = handleHttpResponseError(500);
    expect((error as { type?: HttpErrorType }).type).toBe(HttpErrorType.INTERNAL_ERROR);
    expect(error.message).toBe("服务器内部错误");
  });

  it("未知状态码回落 INTERNAL_ERROR 文案", () => {
    const error = handleHttpResponseError(418);
    expect((error as { type?: HttpErrorType }).type).toBe(HttpErrorType.INTERNAL_ERROR);
    expect(error.message).toBe("服务器内部错误");
  });

  it("400 且消息含 SM2/解密时清除公钥缓存", () => {
    const error = handleHttpResponseError(400, { message: "SM2 解密失败" });
    expect(mockClearPublicKeyCache).toHaveBeenCalledTimes(1);
    expect(error.message).toContain("SM2");
  });

  it("枚举覆盖关键状态码", () => {
    expect(HttpErrorType.UNAUTHORIZED).toBe(401);
    expect(HttpErrorType.FORBIDDEN).toBe(403);
    expect(HttpErrorType.CONFLICT).toBe(409);
    expect(HttpErrorType.SERVICE_UNAVAILABLE).toBe(503);
  });
});

describe("handleNetworkError / handleParseError", () => {
  beforeEach(() => {
    mockMessageError.mockReset();
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("ECONNABORTED 映射请求超时", () => {
    const error = handleNetworkError({ code: "ECONNABORTED" });
    expect(error.message).toBe("请求超时，请检查网络连接");
    expect((error as { originalError?: unknown }).originalError).toEqual({ code: "ECONNABORTED" });
  });

  it("其他网络错误映射网络异常", () => {
    const error = handleNetworkError(new Error("conn reset"));
    expect(error.message).toBe("网络异常，请检查网络连接");
  });

  it("handleParseError 输出解析失败并记录日志", () => {
    const original = new Error("bad json");
    const error = handleParseError(original);
    expect(error.message).toBe("响应解析失败");
    expect((error as { originalError?: unknown }).originalError).toBe(original);
    expect(console.error).toHaveBeenCalled();
  });
});

describe("ErrorHandler / 便捷函数", () => {
  beforeEach(() => {
    mockMessageError.mockReset();
    mockMessageSuccess.mockReset();
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("错误消息提取优先级：response.data.message 最高", () => {
    handleApiError(
      { response: { data: { message: "resp-msg", msg: "resp-msg2" } }, message: "top-msg" },
      "删除工位"
    );
    expect(mockMessageError).toHaveBeenCalledWith("删除工位失败: resp-msg");
    expect(console.error).toHaveBeenCalled();
  });

  it("依次回落 msg / error / message / 默认操作失败", () => {
    handleApiError({ response: { data: { msg: "inner-msg" } } }, "ctx");
    expect(mockMessageError).toHaveBeenLastCalledWith("ctx失败: inner-msg");

    handleApiError({ response: { data: { error: "inner-error" } } }, "ctx");
    expect(mockMessageError).toHaveBeenLastCalledWith("ctx失败: inner-error");

    handleApiError({ message: "plain-message" }, "ctx");
    expect(mockMessageError).toHaveBeenLastCalledWith("ctx失败: plain-message");

    handleApiError({ msg: "obj-msg" }, "ctx");
    expect(mockMessageError).toHaveBeenLastCalledWith("ctx失败: obj-msg");

    handleApiError({ error: "obj-error" }, "ctx");
    expect(mockMessageError).toHaveBeenLastCalledWith("ctx失败: obj-error");

    handleApiError(null, "ctx");
    expect(mockMessageError).toHaveBeenLastCalledWith("ctx失败: 操作失败");
  });

  it("字符串错误直接透传", () => {
    handleApiError("raw string error", "ctx");
    expect(mockMessageError).toHaveBeenCalledWith("ctx失败: raw string error");
  });

  it("showMessage=false 只记日志不弹消息", () => {
    handleApiError({ message: "m" }, "ctx", false);
    expect(mockMessageError).not.toHaveBeenCalled();
    expect(console.error).toHaveBeenCalled();
  });

  it("handleSuccess 区分新增/编辑文案", () => {
    handleSuccess("新增工位");
    expect(mockMessageSuccess).toHaveBeenCalledWith("新增工位成功");
    handleSuccess("新增工位", true);
    expect(mockMessageSuccess).toHaveBeenLastCalledWith("更新新增工位成功");
  });

  it("createResultHandler 返回 success/error 双闭包", () => {
    const handler = ErrorHandler.createResultHandler("导入");
    handler.success(true);
    expect(mockMessageSuccess).toHaveBeenCalledWith("更新导入成功");
    handler.error({ message: "bad" });
    expect(mockMessageError).toHaveBeenCalledWith("导入失败: bad");
  });
});

describe("withErrorHandling / safeAsync / safeSync", () => {
  beforeEach(() => {
    mockMessageError.mockReset();
    mockMessageSuccess.mockReset();
    vi.spyOn(console, "error").mockImplementation(() => {});
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("成功路径：弹成功消息并回调 onSuccess", async () => {
    const onSuccess = vi.fn();
    const result = await withErrorHandling(async () => 42, {
      successMessage: "导入完成",
      onSuccess,
    });
    expect(result).toBe(42);
    expect(mockMessageSuccess).toHaveBeenCalledWith("导入完成");
    expect(onSuccess).toHaveBeenCalledWith(42);
  });

  it("失败路径：返回 null 并回调 onError", async () => {
    const onError = vi.fn();
    const result = await withErrorHandling(() => Promise.reject(new Error("boom")), {
      errorMessage: "导入",
      onError,
    });
    expect(result).toBeNull();
    expect(mockMessageError).toHaveBeenCalledWith("导入失败: boom");
    expect(onError).toHaveBeenCalled();
  });

  it("safeAsync 成功/失败包装", async () => {
    const ok = await safeAsync(async () => "data");
    expect(ok).toEqual({ success: true, data: "data" });

    const bad = await safeAsync(() => Promise.reject("raw-string"));
    expect(bad.success).toBe(false);
    if (!bad.success) {
      expect(bad.error).toBeInstanceOf(Error);
      expect(bad.error.message).toBe("raw-string");
    }
  });

  it("safeSync 成功/失败包装", () => {
    const ok = safeSync(() => JSON.parse('{"a":1}'));
    expect(ok).toEqual({ success: true, data: { a: 1 } });

    const bad = safeSync(() => JSON.parse("{broken"));
    expect(bad.success).toBe(false);
    if (!bad.success) {
      expect(bad.error).toBeInstanceOf(Error);
    }
  });

  it("isFormValidationError 兼容再导出可用", () => {
    expect(typeof isFormValidationError).toBe("function");
  });
});
