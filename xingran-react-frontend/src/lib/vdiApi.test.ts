/**
 * vdiApi 端点契约测试 (Phase 83-03;Phase 100 V130R-10/11 端态收缩)
 *
 * 锁定:虚拟机 CRUD / VDI 操作 / 用户绑定 / 同步 / 资源组查询 /
 * 创建资源枚举 / VDI 服务器管理 各端点 URL 与 snake_case 请求体。
 * (Phase 100 D-100-11: accounts 族 4 方法已删——后端零实现,运行时 404;
 *  D-100-1: 幽灵 batch/statistics/searchOptions 已随 spread→pick 消失。)
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockPost = vi.fn();
vi.mock("@/lib/api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: vi.fn(),
}));
vi.mock("./api", () => ({
  post: (...args: unknown[]) => mockPost(...args),
  get: vi.fn(),
}));

import { vdiServerApi, vmApi } from "./vdiApi";

const OK = { code: 0 };

describe("vmApi — 基础 CRUD", () => {
  beforeEach(() => mockPost.mockReset());

  it("list POST /vdi/vms/list", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const params = { current: 1, pageSize: 10, vmName: "桌面" };
    await vmApi.list(params);
    expect(mockPost).toHaveBeenCalledWith("/vdi/vms/list", params);
  });

  it("get / create / update / delete 按 ID 拼接", async () => {
    mockPost.mockResolvedValue(OK);
    await vmApi.get("vm1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/vdi/vms/vm1", {});
    const create = { vmName: "新桌面", vdiServerId: "s1" };
    await vmApi.create(create);
    expect(mockPost).toHaveBeenNthCalledWith(2, "/vdi/vms", create);
    await vmApi.update("vm1", { vmName: "改名" });
    expect(mockPost).toHaveBeenNthCalledWith(3, "/vdi/vms/vm1/update", { vmName: "改名" });
    await vmApi.delete("vm1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/vdi/vms/vm1/delete", {});
  });
});

describe("vmApi — VDI 操作与用户绑定", () => {
  beforeEach(() => mockPost.mockReset());

  it("operate POST /vdi/vms/operate(路由已于 Phase 100 D-100-10 注册)", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const request = { vmId: "vm1", action: "startup" };
    await vmApi.operate(request);
    expect(mockPost).toHaveBeenCalledWith("/vdi/vms/operate", request);
  });

  it("batchOperate POST /vdi/vms/operate(vm_ids + action, D-100-10 补路由后可用)", async () => {
    mockPost.mockResolvedValueOnce(OK);
    await vmApi.batchOperate(["vm1", "vm2"], "shutdown");
    expect(mockPost).toHaveBeenCalledWith("/vdi/vms/operate", {
      vm_ids: ["vm1", "vm2"],
      action: "shutdown",
    });
  });

  it("bindUser / unbindUser / sync", async () => {
    mockPost.mockResolvedValue(OK);
    const request = { userId: "u1" };
    await vmApi.bindUser("vm1", request);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/vdi/vms/vm1/bind_user", request);
    await vmApi.unbindUser("vm1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/vdi/vms/vm1/unbind_user", {});
    await vmApi.sync("vm1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/vdi/vms/vm1/sync", {});
  });
});

describe("vmApi — 资源与创建枚举查询", () => {
  beforeEach(() => mockPost.mockReset());

  it("listResourceGroups 空 serverId 归一为空串", async () => {
    mockPost.mockResolvedValueOnce(OK);
    await vmApi.listResourceGroups();
    expect(mockPost).toHaveBeenCalledWith("/vdi/vms/resource-groups", { vdi_server_id: "" });
  });

  it("listResources 携带 serverId + group_id", async () => {
    mockPost.mockResolvedValueOnce(OK);
    await vmApi.listResources("s1", "g1");
    expect(mockPost).toHaveBeenCalledWith("/vdi/vms/resources", {
      vdi_server_id: "s1",
      group_id: "g1",
    });
  });

  it("listVTPPlatforms / listRunPositions / listStorages / listNetworks", async () => {
    mockPost.mockResolvedValue(OK);
    await vmApi.listVTPPlatforms("s1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/vdi/vms/vtp-platforms", { vdi_server_id: "s1" });
    await vmApi.listRunPositions("s1", 2);
    expect(mockPost).toHaveBeenNthCalledWith(2, "/vdi/vms/run-positions", {
      vdi_server_id: "s1",
      vtp_id: 2,
    });
    await vmApi.listStorages("s1", 2);
    expect(mockPost).toHaveBeenNthCalledWith(3, "/vdi/vms/storages", {
      vdi_server_id: "s1",
      vtp_id: 2,
    });
    await vmApi.listNetworks("s1", 2);
    expect(mockPost).toHaveBeenNthCalledWith(4, "/vdi/vms/networks", {
      vdi_server_id: "s1",
      vtp_id: 2,
    });
  });
});

describe("vmApi/vdiServerApi — Phase 100 方法集端态", () => {
  it("vmApi 不含幽灵 batch/statistics/searchOptions 与已删 accounts 族", () => {
    const keys = Object.keys(vmApi);
    expect(keys).not.toContain("batch");
    expect(keys).not.toContain("statistics");
    expect(keys).not.toContain("searchOptions");
    expect(keys).not.toContain("listAccounts");
    expect(keys).not.toContain("createAccount");
    expect(keys).not.toContain("resetAccountPassword");
    expect(keys).not.toContain("deleteAccount");
  });

  it("方法集 == 后端已注册路由实测存活集(与 invariants POST_CLEANUP_BASELINE 互指)", () => {
    expect(Object.keys(vmApi).sort()).toEqual([
      "batchOperate",
      "bindUser",
      "create",
      "delete",
      "get",
      "list",
      "listNetworks",
      "listResourceGroups",
      "listResources",
      "listRunPositions",
      "listStorages",
      "listVTPPlatforms",
      "operate",
      "sync",
      "unbindUser",
      "update",
    ]);
    expect(Object.keys(vdiServerApi).sort()).toEqual([
      "create",
      "delete",
      "get",
      "list",
      "testConnection",
      "update",
    ]);
  });
});

describe("vdiServerApi — 服务器管理", () => {
  beforeEach(() => mockPost.mockReset());

  it("list / get / create / update / delete / testConnection", async () => {
    mockPost.mockResolvedValue(OK);
    const params = { current: 1, pageSize: 10 };
    await vdiServerApi.list(params);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/vdi/servers/list", params);
    await vdiServerApi.get("s1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/vdi/servers/s1", {});
    const create = { serverName: "VDI-01", serverUrl: "https://vdi.example.com" };
    await vdiServerApi.create(create);
    expect(mockPost).toHaveBeenNthCalledWith(3, "/vdi/servers", create);
    await vdiServerApi.update("s1", { serverName: "改名" });
    expect(mockPost).toHaveBeenNthCalledWith(4, "/vdi/servers/s1/update", { serverName: "改名" });
    await vdiServerApi.delete("s1");
    expect(mockPost).toHaveBeenNthCalledWith(5, "/vdi/servers/s1/delete", {});
    await vdiServerApi.testConnection("s1");
    expect(mockPost).toHaveBeenNthCalledWith(6, "/vdi/servers/s1/test", {});
  });
});
