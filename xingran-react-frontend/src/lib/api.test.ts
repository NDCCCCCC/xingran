/**
 * api.ts 加密客户端双轨直测 (Phase 83-03 / D-07)
 *
 * 轨道:api.ts 走真实模块链路,底层依赖全部 mock:
 *   - axios → vi.mock 工厂返回可配置假实例(可调用函数 + get/post/... + 拦截器注册捕获)
 *   - @/utils/sm2 / @/utils/sm4(固定密钥向量) / @/utils/errorHandler
 *   - @/store/authStore(getTokenManager → mock TokenManager) / @/store/menuStore
 *   - @/utils/antdMessage / @/services/encryptionConfig
 *
 * 覆盖:加密配置初始化(重试/fail-secure)与刷新、请求方法封装、请求拦截器
 * (Token 注入/白名单/超时/加密编排/明文备份/加密失败回退)、响应拦截器
 * (信封解密/密钥消费/code 分支/401 刷新队列/登录 401 短路/刷新接口 401 登出/
 * 400 "解密失败"单次重放/网络错误)。所有密钥均为假值(T-83-03-01)。
 */
import { afterAll, afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { Mock, MockInstance } from "vitest";

const h = vi.hoisted(() => {
  // vi.mock("axios") 工厂捕获的实例(rawAxios=created[0], api=created[1])
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const created: any[] = [];

  // AxiosHeaders 替身:请求/响应拦截器只调用 set/get/has
  const makeHeaders = (init: Record<string, string> = {}) => {
    const map = new Map<string, string>(Object.entries(init));
    return {
      set: (k: string, v: string) => {
        map.set(k, v);
      },
      get: (k: string) => (map.has(k) ? (map.get(k) as string) : null),
      has: (k: string) => map.has(k),
    };
  };

  // InternalAxiosRequestConfig 替身
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const makeConfig = (overrides: Record<string, any> = {}) => ({
    url: "/system/users/list",
    method: "post",
    data: { name: "xingran" },
    headers: makeHeaders(),
    ...overrides,
  });

  return {
    created,
    makeHeaders,
    makeConfig,
    mockGetAccessToken: vi.fn<() => Promise<string>>(),
    mockRefreshToken: vi.fn(),
    mockClearTokens: vi.fn(),
    mockClearMenus: vi.fn(),
    mockFetchPublicKey: vi.fn(),
    mockEncryptWithSM2: vi.fn(),
    mockClearPublicKeyCache: vi.fn(),
    mockGenerateSM4Key: vi.fn(),
    mockGenerateIV: vi.fn(),
    mockEncryptRequestBody: vi.fn(),
    mockHexToBase64: vi.fn(),
    mockDecryptSM4CBC: vi.fn(),
    mockBase64ToHex: vi.fn(),
    mockHandleHttpResponseError: vi.fn(),
    mockHandleNetworkError: vi.fn(),
    mockClearEncryptionConfigCache: vi.fn(),
    mockMessageError: vi.fn(),
  };
});

vi.mock("axios", () => {
  const createInstance = () => {
    // 可调用实例(401/400 重试走 api(config))+ HTTP 方法 + 拦截器注册捕获
    const instance = Object.assign(vi.fn(), {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      delete: vi.fn(),
      patch: vi.fn(),
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() },
      },
    });
    h.created.push(instance);
    return instance;
  };
  return { default: { create: () => createInstance() } };
});

vi.mock("@/store/authStore", () => ({
  getTokenManager: () => ({
    getAccessToken: h.mockGetAccessToken,
    refreshToken: h.mockRefreshToken,
    clearTokens: h.mockClearTokens,
  }),
}));

vi.mock("@/store/menuStore", () => ({
  useMenuStore: { getState: () => ({ clearMenus: h.mockClearMenus }) },
}));

vi.mock("@/utils/sm2", () => ({
  fetchPublicKey: h.mockFetchPublicKey,
  encryptWithSM2: h.mockEncryptWithSM2,
  clearPublicKeyCache: h.mockClearPublicKeyCache,
}));

vi.mock("@/utils/sm4", () => ({
  generateSM4Key: h.mockGenerateSM4Key,
  generateIV: h.mockGenerateIV,
  encryptRequestBody: h.mockEncryptRequestBody,
  hexToBase64: h.mockHexToBase64,
  decryptSM4CBC: h.mockDecryptSM4CBC,
  base64ToHex: h.mockBase64ToHex,
}));

vi.mock("@/utils/errorHandler", () => ({
  handleHttpResponseError: h.mockHandleHttpResponseError,
  handleNetworkError: h.mockHandleNetworkError,
}));

vi.mock("@/utils/antdMessage", () => ({
  getAppMessage: () => ({ error: h.mockMessageError }),
}));

vi.mock("@/services/encryptionConfig", () => ({
  clearEncryptionConfigCache: h.mockClearEncryptionConfigCache,
}));

import {
  del,
  get,
  initEncryptionConfig,
  post,
  postFormData,
  postLongRequest,
  postTyped,
  put,
  refreshEncryptionConfig,
  upload,
} from "./api";

// 模块加载完成即注册了 2 个实例(rawAxios + api)与 2 组拦截器处理器
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const rawAxios = h.created[0] as any;
// eslint-disable-next-line @typescript-eslint/no-explicit-any
const apiInstance = h.created[1] as any;
const requestInterceptor = (apiInstance.interceptors.request.use as Mock).mock.calls[0][0] as (
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  config: any
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
) => Promise<any>;
const onResponseFulfilled = (apiInstance.interceptors.response.use as Mock).mock.calls[0][0] as (
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  response: any
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
) => Promise<any>;
const onResponseRejected = (apiInstance.interceptors.response.use as Mock).mock.calls[0][1] as (
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  error: any
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
) => Promise<any>;

const ALL_MOCKS: Mock[] = [
  h.mockGetAccessToken,
  h.mockRefreshToken,
  h.mockClearTokens,
  h.mockClearMenus,
  h.mockFetchPublicKey,
  h.mockEncryptWithSM2,
  h.mockClearPublicKeyCache,
  h.mockGenerateSM4Key,
  h.mockGenerateIV,
  h.mockEncryptRequestBody,
  h.mockHexToBase64,
  h.mockDecryptSM4CBC,
  h.mockBase64ToHex,
  h.mockHandleHttpResponseError,
  h.mockHandleNetworkError,
  h.mockClearEncryptionConfigCache,
  h.mockMessageError,
];

// 替换 window.location 以断言 401/失败路径的登录页跳转(jsdom location 可配置替换)
const originalLocation = window.location;
let consoleErrorSpy: MockInstance;
let consoleWarnSpy: MockInstance;

beforeAll(() => {
  Object.defineProperty(window, "location", {
    configurable: true,
    writable: true,
    value: { href: "" },
  });
});

afterAll(() => {
  Object.defineProperty(window, "location", {
    configurable: true,
    writable: true,
    value: originalLocation,
  });
});

beforeEach(() => {
  consoleErrorSpy = vi.spyOn(console, "error").mockImplementation(() => {});
  consoleWarnSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
  ALL_MOCKS.forEach((m) => m.mockReset());
  [
    apiInstance,
    apiInstance.get,
    apiInstance.post,
    apiInstance.put,
    apiInstance.delete,
    apiInstance.patch,
    rawAxios,
    rawAxios.get,
  ].forEach((m: Mock) => m.mockReset());
  window.location.href = "";

  // 默认成功态(测试内按需覆盖)
  h.mockGetAccessToken.mockResolvedValue("token-abc");
  h.mockFetchPublicKey.mockResolvedValue("FAKE-PUBKEY");
  h.mockGenerateSM4Key.mockReturnValue("11".repeat(32));
  h.mockGenerateIV.mockReturnValue("22".repeat(32));
  h.mockEncryptRequestBody.mockResolvedValue("cafebabe");
  h.mockEncryptWithSM2.mockResolvedValue("SM2BLOB");
  h.mockHexToBase64.mockImplementation((hex: string) => `B64[${hex}]`);
  h.mockBase64ToHex.mockImplementation((b64: string) => b64.slice(4, -1));
  h.mockDecryptSM4CBC.mockResolvedValue(JSON.stringify({ code: 0, data: { hello: "world" } }));
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllEnvs();
  vi.useRealTimers();
});

/** 通过 rawAxios 注入后端加密开关,驱动 api.ts 模块级 ENABLE_REQUEST_ENCRYPTION */
async function setEncryption(enabled: boolean) {
  rawAxios.get.mockResolvedValueOnce({
    data: { code: 0, data: { enabled, key: "fake-key", source: "db" } },
  });
  await initEncryptionConfig();
}

describe("initEncryptionConfig / refreshEncryptionConfig", () => {
  it("成功获取加密配置后不再重试", async () => {
    rawAxios.get.mockResolvedValueOnce({
      data: { code: 0, data: { enabled: true, key: "fake-key", source: "db" } },
    });
    await initEncryptionConfig();
    expect(rawAxios.get).toHaveBeenCalledTimes(1);
    expect(rawAxios.get).toHaveBeenCalledWith("/system/auth/encryption-config", { timeout: 3000 });
    expect(consoleErrorSpy).not.toHaveBeenCalled();
  });

  it("首次失败后按指数退避重试成功", async () => {
    rawAxios.get
      .mockRejectedValueOnce(new Error("timeout"))
      .mockResolvedValueOnce({ data: { code: 0, data: { enabled: false } } });

    vi.useFakeTimers();
    const pending = initEncryptionConfig();
    await vi.advanceTimersByTimeAsync(1000);
    await pending;

    expect(rawAxios.get).toHaveBeenCalledTimes(2);
    expect(consoleWarnSpy).toHaveBeenCalledTimes(1);

    // enabled=false 已应用到模块状态:后续请求不走加密
    const config = h.makeConfig();
    await requestInterceptor(config);
    expect(h.mockFetchPublicKey).not.toHaveBeenCalled();
  });

  it("三次重试耗尽后 fail-secure 启用加密", async () => {
    rawAxios.get.mockRejectedValue(new Error("backend down"));

    vi.useFakeTimers();
    const pending = initEncryptionConfig();
    await vi.advanceTimersByTimeAsync(1000 + 2000 + 3000);
    await pending;

    expect(rawAxios.get).toHaveBeenCalledTimes(3);
    expect(consoleErrorSpy).toHaveBeenCalled();

    // fail-secure:后续 POST 请求进入加密编排
    const config = h.makeConfig();
    await requestInterceptor(config);
    expect(h.mockFetchPublicKey).toHaveBeenCalledTimes(1);
  });

  it("refreshEncryptionConfig 成功时清前端缓存并返回 true", async () => {
    rawAxios.get.mockResolvedValueOnce({
      data: { code: 0, data: { enabled: true, key: "fake-key", source: "db" } },
    });
    await expect(refreshEncryptionConfig()).resolves.toBe(true);
    expect(h.mockClearEncryptionConfigCache).toHaveBeenCalledTimes(1);
    expect(rawAxios.get).toHaveBeenCalledWith("/system/auth/encryption-config", { timeout: 5000 });
  });

  it("refreshEncryptionConfig 服务端非成功码返回 false(保持当前设置)", async () => {
    rawAxios.get.mockResolvedValueOnce({ data: { code: 500, message: "boom" } });
    await expect(refreshEncryptionConfig()).resolves.toBe(false);
    expect(consoleErrorSpy).toHaveBeenCalledTimes(1);
  });

  it("refreshEncryptionConfig 网络失败返回 false(保持当前设置)", async () => {
    rawAxios.get.mockRejectedValueOnce(new Error("network down"));
    await expect(refreshEncryptionConfig()).resolves.toBe(false);
    expect(consoleErrorSpy).toHaveBeenCalledTimes(1);
  });
});

describe("请求方法封装", () => {
  it("get/post/put/del 调用 axios 对应方法并透传参数", async () => {
    apiInstance.get.mockResolvedValue({ code: 0 });
    apiInstance.post.mockResolvedValue({ code: 0 });
    apiInstance.put.mockResolvedValue({ code: 0 });
    apiInstance.delete.mockResolvedValue({ code: 0 });
    apiInstance.patch.mockResolvedValue({ code: 0 });

    await get("/a", { page: 1 });
    expect(apiInstance.get).toHaveBeenCalledWith("/a", { params: { page: 1 } });

    await post("/b", { x: 1 });
    expect(apiInstance.post).toHaveBeenCalledWith("/b", { x: 1 });

    await put("/c", { y: 2 });
    expect(apiInstance.put).toHaveBeenCalledWith("/c", { y: 2 });

    await del("/d");
    expect(apiInstance.delete).toHaveBeenCalledWith("/d");

    // 兼容别名
    expect(postTyped).toBe(post);
  });

  it("upload 以 multipart 提交 FormData 并携带文件", async () => {
    apiInstance.post.mockResolvedValue({ code: 0 });
    const file = new File(["content"], "test.xlsx");

    await upload("/system/profile/avatar", file);

    const [url, formData, options] = apiInstance.post.mock.calls[0];
    expect(url).toBe("/system/profile/avatar");
    expect(formData).toBeInstanceOf(FormData);
    expect(formData.get("file")).toBe(file);
    expect(options.headers["Content-Type"]).toBe("multipart/form-data");
  });

  it("postFormData 透传 FormData,postLongRequest 透传自定义 timeout", async () => {
    apiInstance.post.mockResolvedValue({ code: 0 });

    const fd = new FormData();
    await postFormData("/ops/building/import", fd);
    expect(apiInstance.post).toHaveBeenCalledWith("/ops/building/import", fd, {
      headers: { "Content-Type": "multipart/form-data" },
    });

    await postLongRequest("/ad-domain/users/batch-sync", { ids: [] }, 12345);
    expect(apiInstance.post).toHaveBeenLastCalledWith(
      "/ad-domain/users/batch-sync",
      { ids: [] },
      {
        timeout: 12345,
      }
    );

    await postLongRequest("/slow-default");
    expect(apiInstance.post).toHaveBeenLastCalledWith("/slow-default", undefined, {
      timeout: 300000,
    });
  });
});

describe("请求拦截器", () => {
  it("附加 Authorization 与 X-Request-ID", async () => {
    const config = h.makeConfig({ url: "/system/users/list" });
    const result = await requestInterceptor(config);

    expect(result.headers.get("Authorization")).toBe("Bearer token-abc");
    expect(result.headers.get("X-Request-ID")).toBeTruthy();
    expect(h.mockGetAccessToken).toHaveBeenCalledTimes(1);
  });

  it("AUTH 白名单接口跳过 Token 注入但仍生成 X-Request-ID", async () => {
    const config = h.makeConfig({ url: "/system/auth/login" });
    const result = await requestInterceptor(config);

    expect(h.mockGetAccessToken).not.toHaveBeenCalled();
    expect(result.headers.get("Authorization")).toBeNull();
    expect(result.headers.get("X-Request-ID")).toBeTruthy();
  });

  it("长耗时端点使用 35 分钟超时,普通端点 60 秒", async () => {
    const longConfig = h.makeConfig({ url: "/ad-domain/configs/123/sync" });
    await requestInterceptor(longConfig);
    expect(longConfig.timeout).toBe(35 * 60 * 1000);

    const normalConfig = h.makeConfig({ url: "/system/users/list" });
    await requestInterceptor(normalConfig);
    expect(normalConfig.timeout).toBe(60000);
  });

  it("Token 获取失败时非登录接口跳转登录页", async () => {
    h.mockGetAccessToken.mockRejectedValue(new Error("refresh expired"));
    const config = h.makeConfig({ url: "/system/users/list" });

    const result = await requestInterceptor(config);

    expect(consoleErrorSpy).toHaveBeenCalled();
    expect(window.location.href).toBe("/login");
    // 失败后仍继续设置请求头,不中断请求发送
    expect(result.headers.get("X-Request-ID")).toBeTruthy();
  });

  it("加密启用时 POST 请求体被加密并保留明文备份", async () => {
    await setEncryption(true);
    const plain = { name: "xingran" };
    const config = h.makeConfig({ url: "/system/users/list", data: plain });

    const result = await requestInterceptor(config);

    expect(h.mockEncryptRequestBody).toHaveBeenCalledWith(plain, "11".repeat(32), "22".repeat(32));
    expect(h.mockEncryptWithSM2).toHaveBeenCalledWith("11".repeat(32), "FAKE-PUBKEY");
    expect(result.data).toEqual({
      encrypted: true,
      data: "B64[cafebabe]",
      sm4Key: "SM2BLOB",
      iv: "B64[" + "22".repeat(32) + "]",
      timestamp: expect.any(Number),
      nonce: expect.any(String),
    });
    expect(result.headers.get("X-Request-Encrypted")).toBe("true");
    // 明文备份供 400 解密失败重放时恢复
    expect(result.__originalPlainData).toBe(plain);
  });

  it("GET 方法与加密黑名单端点不加密", async () => {
    await setEncryption(true);

    const getConfig = h.makeConfig({ url: "/system/users/list", method: "get" });
    await requestInterceptor(getConfig);
    expect(h.mockFetchPublicKey).not.toHaveBeenCalled();

    const blacklisted = h.makeConfig({ url: "/system/auth/captcha", method: "post" });
    const result = await requestInterceptor(blacklisted);
    expect(h.mockFetchPublicKey).not.toHaveBeenCalled();
    expect(result.headers.get("X-Request-Encrypted")).toBeNull();
    expect(result.data).toEqual({ name: "xingran" });
  });

  it("加密失败时非生产环境回退明文并告警", async () => {
    await setEncryption(true);
    h.mockEncryptRequestBody.mockRejectedValue(new Error("sm4 unavailable"));
    const config = h.makeConfig();

    const result = await requestInterceptor(config);

    expect(result.data).toEqual({ name: "xingran" });
    expect(result.headers.get("X-Request-Encrypted")).toBeNull();
    expect(consoleErrorSpy).toHaveBeenCalled();
    expect(consoleWarnSpy).toHaveBeenCalled();
  });

  it("加密失败时生产环境直接拒绝请求(绝不回退明文)", async () => {
    await setEncryption(true);
    vi.stubEnv("MODE", "production");
    h.mockEncryptRequestBody.mockRejectedValue(new Error("sm4 unavailable"));
    const config = h.makeConfig();

    await expect(requestInterceptor(config)).rejects.toThrow("sm4 unavailable");
  });
});

describe("响应拦截器 — 成功分支", () => {
  it("code===0 时返回 data 本体(非 axios response)", async () => {
    const payload = { code: 0, message: "success", data: { list: [] } };
    const response = { data: payload, status: 200, headers: {}, config: h.makeConfig() };
    await expect(onResponseFulfilled(response)).resolves.toBe(payload);
  });

  it("code 非 0 时展示后端 message 并 reject", async () => {
    const response = {
      data: { code: 1001, message: "参数错误" },
      status: 200,
      headers: {},
      config: h.makeConfig(),
    };
    await expect(onResponseFulfilled(response)).rejects.toThrow("参数错误");
    expect(h.mockMessageError).toHaveBeenCalledWith("参数错误");
  });

  it("无 code 的响应按格式错误 reject", async () => {
    const response = { data: "plain-text", status: 200, headers: {}, config: h.makeConfig() };
    await expect(onResponseFulfilled(response)).rejects.toThrow("Invalid response format");
    expect(h.mockMessageError).toHaveBeenCalledWith("响应格式错误");
    expect(consoleErrorSpy).toHaveBeenCalled();
  });

  it("后端中间件加密响应解密成功且密钥被一次性消费", async () => {
    await setEncryption(true);
    const config = h.makeConfig();
    await requestInterceptor(config);
    const requestId = config.headers.get("X-Request-ID") as string;

    const encryptedBody = { data: "B64[cipherhex]", iv: "B64[ivhex]" };
    const makeResponse = () => ({
      data: { ...encryptedBody },
      status: 200,
      headers: { "x-response-encrypted": "true", "x-request-id": requestId },
      config,
    });

    const result = await onResponseFulfilled(makeResponse());
    expect(result).toEqual({ code: 0, data: { hello: "world" } });
    expect(h.mockDecryptSM4CBC).toHaveBeenCalledWith("cipherhex", "11".repeat(32), "ivhex");

    // 密钥已被消费:同 requestId 再次解密 → 找不到密钥
    await expect(onResponseFulfilled(makeResponse())).rejects.toThrow("Encryption keys not found");
    expect(h.mockMessageError).toHaveBeenCalledWith("响应解密失败：找不到加密密钥");
  });

  it("后端加密响应缺少必要字段时按格式错误 reject", async () => {
    const response = {
      data: { iv: "B64[ivhex]" }, // 缺 data
      status: 200,
      headers: { "x-response-encrypted": "true", "x-request-id": "req-x" },
      config: h.makeConfig(),
    };
    await expect(onResponseFulfilled(response)).rejects.toThrow("Missing encrypted data fields");
    expect(h.mockMessageError).toHaveBeenCalledWith("响应解密失败：格式错误");
  });

  it("前端加密请求的响应解密(encrypted/data/iv 信封, requestId 取自请求头)", async () => {
    await setEncryption(true);
    const config = h.makeConfig();
    await requestInterceptor(config);
    expect(config.headers.get("X-Request-ID")).toBeTruthy();

    const response = {
      data: { encrypted: true, data: "B64[cipherhex]", iv: "B64[ivhex]" },
      status: 200,
      headers: {},
      config,
    };
    const result = await onResponseFulfilled(response);
    expect(result).toEqual({ code: 0, data: { hello: "world" } });
    expect(h.mockBase64ToHex).toHaveBeenCalledWith("B64[cipherhex]");
  });

  it("加密响应缺少请求 ID 时 reject", async () => {
    const response = {
      data: { encrypted: true, data: "B64[cipherhex]", iv: "B64[ivhex]" },
      status: 200,
      headers: {},
      config: { headers: h.makeHeaders() }, // 无 X-Request-ID
    };
    await expect(onResponseFulfilled(response)).rejects.toThrow("Missing request ID");
    expect(h.mockMessageError).toHaveBeenCalledWith("响应解密失败：缺少请求ID");
  });

  it("解密抛错时提示并 reject", async () => {
    await setEncryption(true);
    const config = h.makeConfig();
    await requestInterceptor(config);
    h.mockDecryptSM4CBC.mockRejectedValueOnce(new Error("padding is invalid"));

    const response = {
      data: { encrypted: true, data: "B64[badcipher]", iv: "B64[ivhex]" },
      status: 200,
      headers: {},
      config,
    };
    await expect(onResponseFulfilled(response)).rejects.toThrow("padding is invalid");
    expect(h.mockMessageError).toHaveBeenCalledWith("响应解密失败: padding is invalid");
  });
});

describe("响应拦截器 — 401 分支", () => {
  it("登录接口 401 短路返回后端 message,不进入刷新链路", async () => {
    const config = h.makeConfig({ url: "/system/auth/login" });
    const error = { response: { status: 401, data: { message: "用户名或密码错误" } }, config };

    await expect(onResponseRejected(error)).rejects.toThrow("用户名或密码错误");
    expect(h.mockRefreshToken).not.toHaveBeenCalled();
    expect(h.mockClearMenus).not.toHaveBeenCalled();
    expect(window.location.href).toBe("");
  });

  it("登录接口 401 无 message 时使用默认文案并兼容 msg 字段", async () => {
    const msgOnly = {
      response: { status: 401, data: { msg: "凭据无效" } },
      config: h.makeConfig({ url: "/system/auth/login" }),
    };
    await expect(onResponseRejected(msgOnly)).rejects.toThrow("凭据无效");

    const noMessage = {
      response: { status: 401, data: {} },
      config: h.makeConfig({ url: "/system/auth/login" }),
    };
    await expect(onResponseRejected(noMessage)).rejects.toThrow("用户名或密码错误");
  });

  it("401 触发 Token 刷新并重试原请求", async () => {
    h.mockRefreshToken.mockResolvedValue({ accessToken: "new-token" });
    apiInstance.mockResolvedValue("RETRY-OK");
    const config = h.makeConfig({ url: "/system/users/list" });
    const error = { response: { status: 401, data: {} }, config };

    await expect(onResponseRejected(error)).resolves.toBe("RETRY-OK");
    expect(h.mockClearMenus).toHaveBeenCalledTimes(1);
    expect(h.mockRefreshToken).toHaveBeenCalledTimes(1);
    expect(apiInstance).toHaveBeenCalledWith(config);
  });

  it("并发 401 入队等待,刷新成功后全部重试且只刷新一次", async () => {
    let resolveRefresh!: (value: unknown) => void;
    h.mockRefreshToken.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveRefresh = resolve;
        })
    );
    apiInstance.mockResolvedValue("RETRY");

    const config1 = h.makeConfig({ url: "/a" });
    const config2 = h.makeConfig({ url: "/b" });
    const p1 = onResponseRejected({ response: { status: 401, data: {} }, config: config1 });
    const p2 = onResponseRejected({ response: { status: 401, data: {} }, config: config2 });

    resolveRefresh({});
    await expect(p1).resolves.toBe("RETRY");
    await expect(p2).resolves.toBe("RETRY");

    expect(h.mockRefreshToken).toHaveBeenCalledTimes(1);
    expect(apiInstance).toHaveBeenCalledWith(config1);
    expect(apiInstance).toHaveBeenCalledWith(config2);
  });

  it("刷新失败清空状态、跳转登录并让队列全部失败", async () => {
    let rejectRefresh!: (reason: Error) => void;
    h.mockRefreshToken.mockImplementation(
      () =>
        new Promise((_, reject) => {
          rejectRefresh = reject;
        })
    );

    const config1 = h.makeConfig({ url: "/a" });
    const config2 = h.makeConfig({ url: "/b" });
    const p1 = onResponseRejected({ response: { status: 401, data: {} }, config: config1 });
    const p2 = onResponseRejected({ response: { status: 401, data: {} }, config: config2 });

    const boom = new Error("refresh failed");
    rejectRefresh(boom);

    await expect(p1).rejects.toBe(boom);
    await expect(p2).rejects.toBe(boom);

    expect(h.mockClearTokens).toHaveBeenCalledTimes(1);
    expect(h.mockClearPublicKeyCache).toHaveBeenCalledTimes(1);
    expect(window.location.href).toBe("/login");
    expect(apiInstance).not.toHaveBeenCalled();
  });

  it("刷新接口自身 401 直接登出,不再触发二次刷新", async () => {
    const config = h.makeConfig({ url: "/system/auth/refresh" });
    const error = { response: { status: 401, data: {} }, config };

    await expect(onResponseRejected(error)).rejects.toBe(error);
    expect(h.mockClearTokens).toHaveBeenCalledTimes(1);
    expect(h.mockRefreshToken).not.toHaveBeenCalled();
    expect(window.location.href).toBe("/login");
  });
});

describe("响应拦截器 — 400 解密失败单次重放", () => {
  it("清公钥缓存、恢复明文并重放一次", async () => {
    const headers = h.makeHeaders({
      "X-Request-Encrypted": "true",
      "X-Request-ID": "req-stale",
    });
    const plain = { name: "xingran" };
    const config = h.makeConfig({ headers, __originalPlainData: plain });
    config.data = { encrypted: true, data: "B64[x]" };
    apiInstance.mockResolvedValue("REPLAY-OK");

    const error = {
      response: { status: 400, data: { message: "解密失败: SM2 密钥不匹配" } },
      config,
    };
    await expect(onResponseRejected(error)).resolves.toBe("REPLAY-OK");

    expect(h.mockClearPublicKeyCache).toHaveBeenCalledTimes(1);
    expect(config.data).toBe(plain); // 明文已恢复,供拦截器重新加密
    expect(config.__sm2DecryptRetried).toBe(true);
    expect(apiInstance).toHaveBeenCalledWith(config);
  });

  it("已重放过(__sm2DecryptRetried)不再重放,走通用错误处理", async () => {
    const headers = h.makeHeaders({ "X-Request-Encrypted": "true" });
    const config = h.makeConfig({ headers, __sm2DecryptRetried: true });
    const apiError = new Error("handled-400");
    h.mockHandleHttpResponseError.mockReturnValue(apiError);

    const error = { response: { status: 400, data: { message: "解密失败" } }, config };
    await expect(onResponseRejected(error)).rejects.toBe(apiError);
    expect(apiInstance).not.toHaveBeenCalled();
    expect(h.mockClearPublicKeyCache).not.toHaveBeenCalled();
    expect(h.mockHandleHttpResponseError).toHaveBeenCalledWith(400, { message: "解密失败" });
  });

  it("未加密请求或非解密类 400 不走重放", async () => {
    const plainHeaders = h.makeHeaders(); // 无 X-Request-Encrypted
    const config1 = h.makeConfig({ headers: plainHeaders });
    const apiError = new Error("bad request");
    h.mockHandleHttpResponseError.mockReturnValue(apiError);

    const notEncrypted = {
      response: { status: 400, data: { message: "解密失败" } },
      config: config1,
    };
    await expect(onResponseRejected(notEncrypted)).rejects.toBe(apiError);
    expect(apiInstance).not.toHaveBeenCalled();

    const config2 = h.makeConfig({
      headers: h.makeHeaders({ "X-Request-Encrypted": "true" }),
    });
    const other400 = { response: { status: 400, data: { message: "参数缺失" } }, config: config2 };
    await expect(onResponseRejected(other400)).rejects.toBe(apiError);
    expect(apiInstance).not.toHaveBeenCalled();
  });
});

describe("响应拦截器 — 其他错误", () => {
  it("有 response 的其他状态码走 handleHttpResponseError", async () => {
    const apiError = new Error("server error");
    h.mockHandleHttpResponseError.mockReturnValue(apiError);
    const error = {
      response: { status: 500, data: { message: "internal" } },
      config: h.makeConfig(),
    };

    await expect(onResponseRejected(error)).rejects.toBe(apiError);
    expect(h.mockHandleHttpResponseError).toHaveBeenCalledWith(500, { message: "internal" });
  });

  it("请求超时(ECONNABORTED)与无响应网络错误走 handleNetworkError", async () => {
    const timeoutError = new Error("timeout of 30000ms exceeded");
    (timeoutError as Error & { code?: string }).code = "ECONNABORTED";
    const netError = new Error("Network Error");

    h.mockHandleNetworkError.mockReturnValueOnce(new Error("请求超时，请检查网络连接"));
    h.mockHandleNetworkError.mockReturnValueOnce(new Error("网络异常，请检查网络连接"));

    await expect(onResponseRejected(timeoutError)).rejects.toThrow("请求超时，请检查网络连接");
    await expect(onResponseRejected(netError)).rejects.toThrow("网络异常，请检查网络连接");
    expect(h.mockHandleNetworkError).toHaveBeenCalledTimes(2);
  });
});
