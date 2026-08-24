/**
 * opsApi 端点契约测试 (Phase 83-03)
 *
 * 锁定:通用 CRUD 工厂(list/get/create/update/delete/batch/statistics/searchOptions)
 * + 各实体基座路径 + Excel blob 下载链路(blobAxios) + 地理编码缓存 + 工位设备关联。
 * blobAxios 经 vi.mock("axios") 控制,URL.createObjectURL 在 jsdom 缺失故打桩。
 */
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { Mock } from "vitest";

const h = vi.hoisted(() => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const created: any[] = [];
  return {
    created,
    mockGetAccessToken: vi.fn<() => Promise<string>>(),
    mockCacheGet: vi.fn(),
    mockCacheSet: vi.fn(),
    mockCacheDelete: vi.fn(),
    mockCacheClear: vi.fn(),
    mockCacheGetStats: vi.fn(),
    mockCacheResetStats: vi.fn(),
  };
});

const mockPost = vi.fn();
const mockGet = vi.fn();
const mockPostFormData = vi.fn();
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: (...args: unknown[]) => mockGet(...args),
  postFormData: (...args: unknown[]) => mockPostFormData(...args),
}));
vi.mock("./api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: (...args: unknown[]) => mockGet(...args),
  postFormData: (...args: unknown[]) => mockPostFormData(...args),
}));

// blobAxios(Excel 下载专用 axios 实例)— opsApi 模块加载时 axios.create 的第一个实例
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
    h.created.push(instance);
    return instance;
  };
  return { default: { create: () => createInstance() } };
});

vi.mock("@/utils/authHelpers", () => ({
  getAccessToken: h.mockGetAccessToken,
  getAuthHeaders: vi.fn(),
}));

vi.mock("@/utils/dualLevelCache", () => ({
  getDualLevelCache: () => ({
    get: h.mockCacheGet,
    set: h.mockCacheSet,
    delete: h.mockCacheDelete,
    clear: h.mockCacheClear,
    getStats: h.mockCacheGetStats,
    resetStats: h.mockCacheResetStats,
  }),
}));

import {
  assetApi,
  buildingApi,
  componentApi,
  dedicatedLineApi,
  deptApi,
  doorApi,
  excelApi,
  floorApi,
  floorPlanTextApi,
  geocodeWithCache,
  batchGeocodeWithCache,
  clearGeocodeCache,
  getGeocodeStats,
  infoPointApi,
  locationAliasApi,
  resetGeocodeStats,
  roomDeviceApi,
  roomPhotoApi,
  serverRoomApi,
  wallApi,
  workstationApi,
  workstationDeviceApi,
} from "./opsApi";

// eslint-disable-next-line @typescript-eslint/no-explicit-any
const blobAxios = () => h.created[0] as any;
const blobRequestInterceptor = () =>
  (blobAxios().interceptors.request.use as Mock).mock.calls[0][0] as (config: any) => Promise<any>;

beforeAll(() => {
  // jsdom 不实现 URL.createObjectURL — 打桩以覆盖 triggerBrowserDownload
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
});

beforeEach(() => {
  vi.spyOn(console, "error").mockImplementation(() => {});
  [mockPost, mockGet, mockPostFormData].forEach((m) => m.mockReset());
  [
    blobAxios().get,
    blobAxios().post,
    h.mockGetAccessToken,
    h.mockCacheGet,
    h.mockCacheSet,
    h.mockCacheDelete,
    h.mockCacheClear,
    h.mockCacheGetStats,
    h.mockCacheResetStats,
  ].forEach((m: Mock) => m.mockReset());
  h.mockGetAccessToken.mockResolvedValue("blob-token");
});

describe("通用 CRUD 工厂 — buildingApi(/ops/building)", () => {
  it("list POST /ops/building/list 透传分页筛选", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: { list: [], total: 0 } });
    const params = { current: 1, pageSize: 10, name: "研发楼" };
    await buildingApi.list(params);
    expect(mockPost).toHaveBeenCalledWith("/ops/building/list", params);
  });

  it("get/create/update/delete 按 ID 拼接", async () => {
    mockPost.mockResolvedValue({ code: 0 });
    await buildingApi.get("b1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ops/building/b1", {});
    await buildingApi.create({ name: "新楼" });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ops/building", { name: "新楼" });
    await buildingApi.update("b1", { name: "改名" });
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ops/building/b1/update", { name: "改名" });
    await buildingApi.delete("b1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/ops/building/b1/delete", {});
  });

  it("batch 合并 action 与附加参数", async () => {
    mockPost.mockResolvedValueOnce({ code: 0 });
    await buildingApi.batch("delete", { ids: ["b1"] });
    expect(mockPost).toHaveBeenCalledWith("/ops/building/batch", { action: "delete", ids: ["b1"] });
  });

  it("statistics 解包 data,空 data 回退 {}", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: { total: 5, enabled: 3 } });
    expect(await buildingApi.statistics()).toEqual({ total: 5, enabled: 3 });
    expect(mockPost).toHaveBeenCalledWith("/ops/building/statistics", {});

    mockPost.mockResolvedValueOnce({ code: 0, data: null });
    expect(await buildingApi.statistics({ status: 0 })).toEqual({});
    expect(mockPost).toHaveBeenLastCalledWith("/ops/building/statistics", { status: 0 });
  });

  it("searchOptions 解包 data,空 data 回退 []", async () => {
    mockPost.mockResolvedValueOnce({ code: 0, data: [{ value: "b1", label: "研发楼" }] });
    expect(await buildingApi.searchOptions({ name: "研发" })).toEqual([
      { value: "b1", label: "研发楼" },
    ]);
    expect(mockPost).toHaveBeenCalledWith("/ops/building/dropdown-options", { name: "研发" });

    mockPost.mockResolvedValueOnce({ code: 0, data: null });
    expect(await buildingApi.searchOptions()).toEqual([]);
  });
});

describe("各实体工厂基座路径", () => {
  it("floor / workstation / serverRoom / roomDevice / dedicatedLine / infoPoint / walls / doors / floor-plan-texts", async () => {
    mockPost.mockResolvedValue({ code: 0 });
    await floorApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ops/floor/list", { current: 1, pageSize: 10 });
    await workstationApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ops/workstation/list", {
      current: 1,
      pageSize: 10,
    });
    await serverRoomApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ops/serverRoom/list", {
      current: 1,
      pageSize: 10,
    });
    await roomDeviceApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenNthCalledWith(4, "/ops/roomDevice/list", {
      current: 1,
      pageSize: 10,
    });
    await dedicatedLineApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenNthCalledWith(5, "/ops/dedicatedLine/list", {
      current: 1,
      pageSize: 10,
    });
    await infoPointApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenNthCalledWith(6, "/ops/infoPoint/list", {
      current: 1,
      pageSize: 10,
    });
    await wallApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenNthCalledWith(7, "/ops/walls/list", { current: 1, pageSize: 10 });
    await doorApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenNthCalledWith(8, "/ops/doors/list", { current: 1, pageSize: 10 });
    await floorPlanTextApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenNthCalledWith(9, "/ops/floor-plan-texts/list", {
      current: 1,
      pageSize: 10,
    });
  });

  it("floorApi.tree POST /ops/floor/tree", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValueOnce({ code: 0, data: [] });
    await floorApi.tree();
    expect(mockPost).toHaveBeenCalledWith("/ops/floor/tree", {});
  });

  it("workstationApi 扩展:updatePositions / deptOptions", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValue({ code: 0, data: [] });
    const items = [{ id: "w1", positionX: 1, positionY: 2 }];
    await workstationApi.updatePositions(items);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ops/workstation/positions", { items });
    await workstationApi.deptOptions("org-1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ops/workstation/dept-options", {
      orgId: "org-1",
    });
  });

  it("walls/doors/floor-plan-texts 的 batch 覆写为 {action, ids}", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValue({ code: 0 });
    await wallApi.batch("delete", ["w1"]);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ops/walls/batch", {
      action: "delete",
      ids: ["w1"],
    });
    await doorApi.batch("delete", ["d1"]);
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ops/doors/batch", {
      action: "delete",
      ids: ["d1"],
    });
    await floorPlanTextApi.batch("delete", ["t1"]);
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ops/floor-plan-texts/batch", {
      action: "delete",
      ids: ["t1"],
    });
  });
});

describe("locationAliasApi(Phase 39)", () => {
  it("list 注入默认分页 pageNum=1/pageSize=10", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValueOnce({ code: 0, data: { list: [], total: 0 } });
    await locationAliasApi.list();
    expect(mockPost).toHaveBeenCalledWith("/ops/location-alias/list", { pageNum: 1, pageSize: 10 });
    await locationAliasApi.list({ pageNum: 2, pageSize: 5 });
    expect(mockPost).toHaveBeenLastCalledWith("/ops/location-alias/list", {
      pageNum: 2,
      pageSize: 5,
    });
  });

  it("create 默认 scope=workstation;update/delete 按 ID 拼接", async () => {
    mockPost.mockResolvedValue({ code: 0 });
    await locationAliasApi.create({ deptId: "d1", locationId: "l1" });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ops/location-alias", {
      deptId: "d1",
      locationId: "l1",
      scope: "workstation",
    });
    await locationAliasApi.update("a1", { remark: "备注" });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ops/location-alias/a1/update", {
      remark: "备注",
    });
    await locationAliasApi.delete("a1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ops/location-alias/a1/delete", {});
  });
});

describe("roomPhotoApi", () => {
  it("upload 构造 FormData(roomId/primaryIndex/files)", async () => {
    mockPostFormData.mockReset();
    mockPostFormData.mockResolvedValueOnce({ code: 0 });
    const file = new File(["photo"], "room.png");

    await roomPhotoApi.upload("r1", [file], 2);

    const [url, formData] = mockPostFormData.mock.calls[0];
    expect(url).toBe("/ops/rooms/photos/upload");
    expect(formData).toBeInstanceOf(FormData);
    expect(formData.get("roomId")).toBe("r1");
    expect(formData.get("primaryIndex")).toBe("2");
    expect(formData.getAll("files")).toEqual([file]);
  });

  it("各端点 URL:get/setPrimary/updateDescription/delete/batchDelete/getPrimary/count", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValue({ code: 0 });
    await roomPhotoApi.list("r1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ops/rooms/photos/list/r1", {});
    await roomPhotoApi.get("p1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ops/rooms/photos/p1", {});
    await roomPhotoApi.setPrimary("p1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ops/rooms/photos/p1/primary", {});
    await roomPhotoApi.updateDescription("p1", "机房全景");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/ops/rooms/photos/p1/description", {
      description: "机房全景",
    });
    await roomPhotoApi.delete("p1");
    expect(mockPost).toHaveBeenNthCalledWith(5, "/ops/rooms/photos/p1", {});
    await roomPhotoApi.batchDelete(["p1", "p2"]);
    expect(mockPost).toHaveBeenNthCalledWith(6, "/ops/rooms/photos/batch-delete", {
      ids: ["p1", "p2"],
    });
    await roomPhotoApi.getPrimary("r1");
    expect(mockPost).toHaveBeenNthCalledWith(7, "/ops/rooms/photos/primary?roomId=r1", {});
    await roomPhotoApi.count("r1");
    expect(mockPost).toHaveBeenNthCalledWith(8, "/ops/rooms/photos/count?roomId=r1", {});
  });
});

describe("Excel 导入导出(blobAxios 链路)", () => {
  it("blobAxios 请求拦截器注入 Bearer Token", async () => {
    const headersMap = new Map<string, string>();
    const headers = {
      set: (k: string, v: string) => {
        headersMap.set(k, v);
      },
      get: (k: string) => headersMap.get(k) ?? null,
    };
    const config = { url: "/ops/building/template", headers };

    const result = await blobRequestInterceptor()(config);

    expect(result).toBe(config);
    expect(headersMap.get("Authorization")).toBe("Bearer blob-token");
  });

  it("downloadTemplate GET /ops/:type/template 并触发浏览器下载", async () => {
    blobAxios().get.mockResolvedValueOnce({ status: 200, data: new Blob(["x"]) });

    await excelApi.downloadTemplate("building");

    expect(blobAxios().get).toHaveBeenCalledWith("/ops/building/template", {
      responseType: "blob",
    });
    expect(URL.createObjectURL).toHaveBeenCalled();
  });

  it("downloadTemplate 非 2xx 时抛出下载失败", async () => {
    blobAxios().get.mockResolvedValueOnce({ status: 500, data: new Blob([]) });
    await expect(excelApi.downloadTemplate("building")).rejects.toThrow(
      "下载失败: building_template.xlsx"
    );
  });

  it("import 构造 FormData 上传文件", async () => {
    mockPostFormData.mockResolvedValueOnce({ code: 0 });
    const file = new File(["rows"], "import.xlsx");

    await excelApi.import("workstation", file);

    const [url, formData] = mockPostFormData.mock.calls[0];
    expect(url).toBe("/ops/workstation/import");
    expect(formData).toBeInstanceOf(FormData);
    expect(formData.get("file")).toBe(file);
  });

  it("export 从 content-disposition 提取文件名", async () => {
    blobAxios().post.mockResolvedValueOnce({
      status: 200,
      data: new Blob(["xlsx"]),
      headers: { "content-disposition": 'attachment; filename="%E6%A5%BC%E5%AE%87.xlsx"' },
    });

    await excelApi.export("building", { status: 0 });

    expect(blobAxios().post).toHaveBeenCalledWith(
      "/ops/building/export",
      { status: 0 },
      {
        responseType: "blob",
      }
    );
    expect(URL.createObjectURL).toHaveBeenCalled();
  });

  it("export 无 content-disposition 时使用默认文件名", async () => {
    blobAxios().post.mockResolvedValueOnce({ status: 200, data: new Blob(["x"]), headers: {} });
    await expect(excelApi.export("floor", {})).resolves.toBeUndefined();
    expect(URL.createObjectURL).toHaveBeenCalled();
  });

  it("getStatusOptions POST /ops/:type/status-options", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValueOnce({ code: 0, data: [] });
    await excelApi.getStatusOptions("building");
    expect(mockPost).toHaveBeenCalledWith("/ops/building/status-options", {});
  });

  it("deptApi.exportMapping GET /ops/workstation/dept-mapping-template", async () => {
    blobAxios().get.mockReset();
    blobAxios().get.mockResolvedValueOnce({ status: 200, data: new Blob(["map"]) });
    await deptApi.exportMapping();
    expect(blobAxios().get).toHaveBeenCalledWith("/ops/workstation/dept-mapping-template", {
      responseType: "blob",
    });
  });
});

describe("地理编码缓存", () => {
  it("空地址直接返回 null 不发请求", async () => {
    expect(await geocodeWithCache("")).toBeNull();
    expect(await geocodeWithCache("   ")).toBeNull();
    expect(mockPost).not.toHaveBeenCalled();
  });

  it("缓存命中直接返回,不调用后端", async () => {
    const cached = { lng: 116.4, lat: 39.9 };
    h.mockCacheGet.mockReturnValueOnce(cached);

    const result = await geocodeWithCache("北京市海淀区");

    expect(result).toBe(cached);
    expect(h.mockCacheGet).toHaveBeenCalledWith("geocode_北京市海淀区");
    expect(mockPost).not.toHaveBeenCalled();
  });

  it("缓存未命中调用后端并写缓存", async () => {
    const remote = { lng: 116.4, lat: 39.9, formattedAddress: "北京市海淀区" };
    mockPost.mockResolvedValueOnce({ code: 0, data: remote });

    const result = await geocodeWithCache("北京市海淀区");

    expect(mockPost).toHaveBeenCalledWith("/ops/building/geocode", { address: "北京市海淀区" });
    expect(result).toBe(remote);
    expect(h.mockCacheSet).toHaveBeenCalledWith("geocode_北京市海淀区", remote);
  });

  it("后端返回非成功码返回 null 不写缓存", async () => {
    mockPost.mockResolvedValueOnce({ code: 500, message: "配额超限" });
    expect(await geocodeWithCache("某地")).toBeNull();
    expect(h.mockCacheSet).not.toHaveBeenCalled();
  });

  it("后端抛错时吞错返回 null", async () => {
    mockPost.mockRejectedValueOnce(new Error("network down"));
    expect(await geocodeWithCache("某地")).toBeNull();
  });

  it("batchGeocodeWithCache 聚合成功的地址为 Map", async () => {
    const remote = { lng: 1, lat: 2 };
    h.mockCacheGet.mockReturnValueOnce(null).mockReturnValueOnce({ lng: 3, lat: 4 });
    mockPost.mockResolvedValueOnce({ code: 0, data: remote });

    const results = await batchGeocodeWithCache(["地址A", "地址B"]);

    expect(results.size).toBe(2);
    expect(results.get("地址A")).toBe(remote);
    expect(results.get("地址B")).toEqual({ lng: 3, lat: 4 });
  });

  it("getGeocodeStats / clearGeocodeCache / resetGeocodeStats 委托缓存", () => {
    h.mockCacheGetStats.mockReturnValueOnce({
      hits: 5,
      misses: 2,
      memorySize: 3,
      storageSize: 4,
      totalSize: 7,
    });
    const stats = getGeocodeStats();
    expect(stats).toMatchObject({
      apiCalls: 2,
      cacheSize: 7,
      memoryCacheSize: 3,
      storageCacheSize: 4,
    });

    clearGeocodeCache("地址A");
    expect(h.mockCacheDelete).toHaveBeenCalledWith("geocode_地址A");
    clearGeocodeCache();
    expect(h.mockCacheClear).toHaveBeenCalledTimes(1);

    resetGeocodeStats();
    expect(h.mockCacheResetStats).toHaveBeenCalledTimes(1);
  });
});

describe("assetApi(运维资产)/componentApi", () => {
  it("searchBySerial GET /ops/asset/search-by-serial/:serial", async () => {
    mockGet.mockReset();
    mockGet.mockResolvedValueOnce({ code: 0 });
    await assetApi.searchBySerial("SN123");
    expect(mockGet).toHaveBeenCalledWith("/ops/asset/search-by-serial/SN123", {});
  });

  it("getDeviceTypes / getDeviceCategories / getStatusValues", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValue({ code: 0, data: [] });
    await assetApi.getDeviceTypes();
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ops/asset/device-types", {});
    await assetApi.getDeviceCategories();
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ops/asset/device-categories", {});
    await assetApi.getStatusValues();
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ops/asset/status-values", {});
  });

  it("statistics 解包 res.data", async () => {
    mockPost.mockReset();
    const stat = { total: 10, normal: 8, stopped: 1, nbf: 1 };
    mockPost.mockResolvedValueOnce({ code: 0, data: stat });
    expect(await assetApi.statistics()).toBe(stat);
    expect(mockPost).toHaveBeenCalledWith("/ops/asset/statistics", {});
  });

  it("excel.template / excel.import(FormData)", async () => {
    mockPost.mockReset();
    mockPostFormData.mockReset();
    mockPost.mockResolvedValueOnce({ code: 0 });
    await assetApi.excel.template();
    expect(mockPost).toHaveBeenCalledWith("/ops/asset/template", {});

    mockPostFormData.mockResolvedValueOnce({ code: 0 });
    const file = new File(["rows"], "assets.xlsx");
    await assetApi.excel.import(file);
    const [url, formData] = mockPostFormData.mock.calls[0];
    expect(url).toBe("/ops/asset/import");
    expect(formData.get("file")).toBe(file);
  });

  it("excel.export 走 blobAxios 并返回成功信封", async () => {
    blobAxios().post.mockResolvedValueOnce({ status: 200, data: new Blob(["x"]), headers: {} });
    const result = await assetApi.excel.export({ status: 0 });
    expect(blobAxios().post).toHaveBeenCalledWith(
      "/ops/asset/export",
      { status: 0 },
      {
        responseType: "blob",
      }
    );
    expect(result).toEqual({ code: 0, message: "导出成功", data: null });
  });

  it("componentApi.list GET /ops/asset/components(Phase 48 R4)", async () => {
    mockGet.mockReset();
    mockGet.mockResolvedValueOnce({ code: 0, data: { list: [], total: 0 } });
    const res = await componentApi.list("parent-1");
    expect(mockGet).toHaveBeenCalledWith("/ops/asset/components", { parentAssetId: "parent-1" });
    expect(res.data).toEqual({ list: [], total: 0 });
  });
});

describe("workstationDeviceApi(工位设备关联)", () => {
  it("查询族:getManual/getAD/getAsset/getPhysical/getByWorkstation", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValue({ code: 0, data: [] });
    await workstationDeviceApi.getManual("w1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ops/workstation-device/w1", {});
    await workstationDeviceApi.getAD("w1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ops/workstation-device/w1/ad", {});
    await workstationDeviceApi.getAsset("w1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ops/workstation-device/w1/asset", {});
    await workstationDeviceApi.getPhysical("w1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/ops/workstation-device/w1/physical", {});
    await workstationDeviceApi.getByWorkstation("w1");
    expect(mockPost).toHaveBeenNthCalledWith(5, "/ops/workstation-device/w1", {});
  });

  it("写入族:addManual/syncAD/syncAsset/setPrimaryAndSave/update/delete/setPrimary", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValue({ code: 0 });
    const device = { workstationId: "w1", deviceName: "PC-1", deviceSerial: "SN1" };
    await workstationDeviceApi.addManual(device as never);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ops/workstation-device/manual", device);
    await workstationDeviceApi.syncAD("w1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ops/workstation-device/sync-ad", {
      workstation_id: "w1",
    });
    await workstationDeviceApi.syncAsset("w1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ops/workstation-device/sync-asset", {
      workstation_id: "w1",
    });
    const save = { workstationId: "w1", deviceSerial: "SN1", deviceName: "PC-1" };
    await workstationDeviceApi.setPrimaryAndSave("d1", save);
    expect(mockPost).toHaveBeenNthCalledWith(
      4,
      "/ops/workstation-device/d1/set-primary-and-save",
      save
    );
    await workstationDeviceApi.update("d1", { deviceName: "改名" });
    expect(mockPost).toHaveBeenNthCalledWith(5, "/ops/workstation-device/d1/update", {
      deviceName: "改名",
    });
    await workstationDeviceApi.delete("d1");
    expect(mockPost).toHaveBeenNthCalledWith(6, "/ops/workstation-device/d1/delete", {});
    await workstationDeviceApi.setPrimary("d1");
    expect(mockPost).toHaveBeenNthCalledWith(7, "/ops/workstation-device/d1/set-primary", {});
  });
});
