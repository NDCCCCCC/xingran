/**
 * download 契约测试 (Phase 94-01 D-11;Phase 100 D-100-7 增补)
 *
 * 锁定: blobAxios 异步 Bearer 注入拦截器 / downloadFile GET 链(含非 2xx 抛错) /
 * downloadFilePost POST 链(content-disposition URL 编码文件名提取 + 默认回退 +
 * 非 2xx 抛错) / triggerBrowserDownload 触发顺序 / 5min 超时锁值 /
 * 200+application/json 错误体检测(GET/POST 双挂,content-type 强判据)。
 * (Phase 100 V130R-11: executionApi.downloadReport 判 dead 随 rpaApi 裁剪摘除;
 * downloadFilePost 本身保留,活消费者 opsApi excel/asset export。)
 * axios create 工厂 mock 照抄 opsApi.test.ts:41-55,URL.createObjectURL 打桩 :103-115。
 */
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { Mock } from "vitest";

const h = vi.hoisted(() => {
  const createConfigs: Record<string, unknown>[] = [];
  return {
    createConfigs,
    mockGetAccessToken: vi.fn<() => Promise<string>>(),
  };
});

vi.mock("./api", () => ({
  post: vi.fn(),
  get: vi.fn(),
}));

vi.mock("axios", () => {
  const createInstance = () => {
    const instance = Object.assign(vi.fn(), {
      get: vi.fn(),
      post: vi.fn(),
      interceptors: {
        request: { use: vi.fn() },
        response: { use: vi.fn() },
      },
    });
    return instance;
  };
  return {
    default: {
      create: (config: Record<string, unknown>) => {
        h.createConfigs.push(config);
        return createInstance();
      },
    },
  };
});

vi.mock("@/utils/authHelpers", () => ({
  getAccessToken: h.mockGetAccessToken,
  getAuthHeaders: vi.fn(),
}));

import { blobAxios, downloadFile, downloadFilePost, triggerBrowserDownload } from "./download";

/** blobAxios 的测试视型:暴露 mock 的 get/post 与请求拦截器注册点 */
const mockedBlobAxios = blobAxios as unknown as {
  get: Mock;
  post: Mock;
  interceptors: { request: { use: Mock } };
};

/** 拦截器可处理的 config 最小形状(AxiosHeaders 的 set/get 二元组) */
interface InterceptableConfig {
  url: string;
  headers: {
    set: (key: string, value: string) => void;
    get: (key: string) => string | null;
  };
}

const getRequestInterceptor = () =>
  mockedBlobAxios.interceptors.request.use.mock.calls[0][0] as (
    config: InterceptableConfig
  ) => Promise<InterceptableConfig>;

const createObjectURLMock = () => URL.createObjectURL as unknown as Mock;
const revokeObjectURLMock = () => URL.revokeObjectURL as unknown as Mock;

// jsdom 不实现 URL.createObjectURL — 打桩以覆盖 triggerBrowserDownload;
// 同时捕获其内部创建的 <a> 元素并覆写 click,用于断言文件名与触发顺序。
const anchorCapture: { element?: HTMLAnchorElement; click?: Mock } = {};

beforeAll(() => {
  Object.defineProperty(URL, "createObjectURL", {
    configurable: true,
    writable: true,
    value: vi.fn(() => "blob:fake-url"),
  });
  Object.defineProperty(URL, "revokeObjectURL", {
    configurable: true,
    writable: true,
    value: vi.fn(),
  });
  const originalCreateElement = document.createElement.bind(document);
  vi.spyOn(document, "createElement").mockImplementation(((tag: string) => {
    const el = originalCreateElement(tag);
    if (tag === "a") {
      const click = vi.fn();
      Object.defineProperty(el, "click", { value: click, configurable: true, writable: true });
      anchorCapture.element = el as HTMLAnchorElement;
      anchorCapture.click = click;
    }
    return el;
  }) as unknown as typeof document.createElement);
});

beforeEach(() => {
  mockedBlobAxios.get.mockReset();
  mockedBlobAxios.post.mockReset();
  h.mockGetAccessToken.mockReset();
  h.mockGetAccessToken.mockResolvedValue("tok");
  createObjectURLMock().mockClear();
  revokeObjectURLMock().mockClear();
});

describe("blobAxios 请求拦截器 (T-94-01)", () => {
  it("异步 getAccessToken resolve tok 后注入 Authorization: Bearer tok", async () => {
    const headersMap = new Map<string, string>();
    const config: InterceptableConfig = {
      url: "/ops/building/template",
      headers: {
        set: (k, v) => {
          headersMap.set(k, v);
        },
        get: (k) => headersMap.get(k) ?? null,
      },
    };

    const result = await getRequestInterceptor()(config);

    expect(result).toBe(config);
    expect(headersMap.get("Authorization")).toBe("Bearer tok");
    expect(h.mockGetAccessToken).toHaveBeenCalledTimes(1);
  });
});

describe("downloadFile (GET 链)", () => {
  it("GET 请求带 responseType blob 并触发浏览器下载", async () => {
    mockedBlobAxios.get.mockResolvedValueOnce({ status: 200, data: new Blob(["x"]) });

    await downloadFile("/ops/building/template", "building_template.xlsx");

    expect(mockedBlobAxios.get).toHaveBeenCalledWith("/ops/building/template", {
      responseType: "blob",
    });
    expect(createObjectURLMock()).toHaveBeenCalled();
  });

  it("非 2xx 状态 reject 且错误信息含「下载失败」", async () => {
    mockedBlobAxios.get.mockResolvedValueOnce({ status: 500, data: new Blob([]) });

    await expect(downloadFile("/x", "y.xlsx")).rejects.toThrow("下载失败");
  });
});

describe("downloadFilePost (POST 链)", () => {
  it("POST 请求带 responseType blob,content-disposition URL 编码文件名提取并解码", async () => {
    mockedBlobAxios.post.mockResolvedValueOnce({
      status: 200,
      data: new Blob(["xlsx"]),
      headers: { "content-disposition": 'attachment; filename="%E6%A5%BC%E5%AE%87.xlsx"' },
    });

    await downloadFilePost("/ops/building/export", { status: 0 }, "building_export.xlsx");

    expect(mockedBlobAxios.post).toHaveBeenCalledWith(
      "/ops/building/export",
      { status: 0 },
      {
        responseType: "blob",
      }
    );
    expect(anchorCapture.element?.download).toBe("楼宇.xlsx");
    expect(createObjectURLMock()).toHaveBeenCalled();
  });

  it("无 content-disposition 时回退 defaultFilename", async () => {
    mockedBlobAxios.post.mockResolvedValueOnce({
      status: 200,
      data: new Blob(["x"]),
      headers: {},
    });

    await downloadFilePost("/ops/floor/export", {}, "floor_export.xlsx");

    expect(anchorCapture.element?.download).toBe("floor_export.xlsx");
  });

  it("非 2xx 状态 reject 且错误信息含「下载失败」", async () => {
    mockedBlobAxios.post.mockResolvedValueOnce({ status: 502, data: new Blob([]), headers: {} });

    await expect(downloadFilePost("/x", {}, "z.xlsx")).rejects.toThrow("下载失败");
  });
});

describe("triggerBrowserDownload 触发顺序", () => {
  it("createObjectURL → a.click → revokeObjectURL", () => {
    triggerBrowserDownload(new Blob(["x"]), "report.xlsx");

    const createOrder = createObjectURLMock().mock.invocationCallOrder.at(-1) as number;
    const clickOrder = anchorCapture.click?.mock.invocationCallOrder.at(-1) as number;
    const revokeOrder = revokeObjectURLMock().mock.invocationCallOrder.at(-1) as number;

    expect(anchorCapture.element?.download).toBe("report.xlsx");
    expect(clickOrder).toBeGreaterThan(createOrder);
    expect(revokeOrder).toBeGreaterThan(clickOrder);
  });
});

describe("blobAxios 实例配置", () => {
  it("timeout 锁值 300000(5min 超时语义不回退)", () => {
    expect(h.createConfigs[0]?.timeout).toBe(300000);
  });
});

describe("200+JSON 错误体检测 (Phase 100 D-100-7)", () => {
  // jsdom 的 Blob 不实现 .text()——JSON 检测只消费 .text(),mock 数据按
  // 被测单元实际消费面提供最小形状即可
  const textBlob = (text: string) =>
    ({ text: async () => text, size: text.length }) as unknown as Blob;

  beforeEach(() => {
    createObjectURLMock().mockClear();
    revokeObjectURLMock().mockClear();
  });

  it("downloadFilePost: 200 + application/json 错误体 → throw message 字段,不触发下载", async () => {
    mockedBlobAxios.post.mockResolvedValueOnce({
      status: 200,
      data: textBlob(JSON.stringify({ code: 500, message: "导出失败" })),
      headers: { "content-type": "application/json" },
    });

    await expect(downloadFilePost("/x", {}, "z.xlsx")).rejects.toThrow("导出失败");
    expect(createObjectURLMock()).not.toHaveBeenCalled();
  });

  it("downloadFilePost: 200 + application/json + 非法 JSON body → 回退默认文案「下载失败」", async () => {
    mockedBlobAxios.post.mockResolvedValueOnce({
      status: 200,
      data: textBlob("<html>not-json</html>"),
      headers: { "content-type": "application/json" },
    });

    await expect(downloadFilePost("/x", {}, "z.xlsx")).rejects.toThrow("下载失败");
    expect(createObjectURLMock()).not.toHaveBeenCalled();
  });

  it("downloadFilePost: 流式 content-type + Blob → 正常下载并返回实际 filename(防误伤)", async () => {
    mockedBlobAxios.post.mockResolvedValueOnce({
      status: 200,
      data: new Blob(["xlsx-bytes"]),
      headers: {
        "content-type": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
        "content-disposition": 'attachment; filename="%E6%A5%BC%E5%AE%87.xlsx"',
      },
    });

    const filename = await downloadFilePost("/ops/building/export", {}, "fallback.xlsx");

    expect(filename).toBe("楼宇.xlsx");
    expect(anchorCapture.element?.download).toBe("楼宇.xlsx");
    expect(createObjectURLMock()).toHaveBeenCalled();
  });

  it("downloadFile(GET): 200 + application/json 错误体 → throw(GET 侧同构防护)", async () => {
    mockedBlobAxios.get.mockResolvedValueOnce({
      status: 200,
      data: textBlob(JSON.stringify({ code: 500, message: "导出失败" })),
      headers: { "content-type": "application/json; charset=utf-8" },
    });

    await expect(downloadFile("/network/history/list?format=xlsx", "mac.xlsx")).rejects.toThrow(
      "导出失败"
    );
    expect(createObjectURLMock()).not.toHaveBeenCalled();
  });
});
