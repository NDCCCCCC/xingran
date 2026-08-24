/**
 * adDomainApi 端点契约测试 (Phase 83-03)
 *
 * 锁定:AD 配置 / OU / 用户组 / 用户 / 同步日志 / 电脑 / OU 部门与组映射 /
 * 批量同步 / 服务账号池 各端点 URL、withDefaultPagination 默认分页注入。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

const mockGet = vi.fn();
const mockPost = vi.fn();
vi.mock("@/lib/api", () => ({
  get: (...args: unknown[]) => mockGet(...args),
  post: (...args: unknown[]) => mockPost(...args),
}));
vi.mock("./api", () => ({
  get: (...args: unknown[]) => mockGet(...args),
  post: (...args: unknown[]) => mockPost(...args),
}));

import {
  addADGroupMember,
  batchSyncADUsersDirect,
  createADConfig,
  createADServiceAccount,
  createMapping,
  createOUGroupMapping,
  deleteADConfig,
  deleteADServiceAccount,
  deleteMapping,
  deleteOUGroupMapping,
  disableADServiceAccount,
  disableADUser,
  enableADServiceAccount,
  enableADUser,
  getADComputerDetail,
  getADComputerList,
  getADConfig,
  getADConfigList,
  getADGroupDetail,
  getADGroupList,
  getADGroupMembers,
  getADGroupSyncStatus,
  getADOUTree,
  getADServiceAccountStats,
  getADSyncLogs,
  getADUserDetail,
  getADUserIds,
  getADUserList,
  getMapping,
  getMappingList,
  getOUDeptMapping,
  getOUGroupMappings,
  getOUGroupMappingsByOU,
  listADServiceAccounts,
  moveADUser,
  removeADGroupMember,
  syncADData,
  syncADGroups,
  syncADSingleGroup,
  testADConnection,
  unlockADServiceAccount,
  updateADConfig,
  updateADGroup,
  updateADServiceAccount,
  updateADUser,
  updateMapping,
  updateOUDeptMapping,
  updateOUGroupMapping,
} from "./adDomainApi";

const OK = { code: 0 };

describe("adDomainApi — 配置管理", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
  });

  it("getADConfigList 注入默认分页 current=1/pageSize=10 且允许覆盖", async () => {
    mockPost.mockResolvedValue(OK);
    await getADConfigList();
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ad-domain/configs/list", {
      current: 1,
      pageSize: 10,
    });
    await getADConfigList({ current: 2, pageSize: 20 });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ad-domain/configs/list", {
      current: 2,
      pageSize: 20,
    });
  });

  it("create/get/update/delete/test/sync 按 ID 拼接", async () => {
    mockPost.mockResolvedValue(OK);
    mockGet.mockResolvedValue(OK);
    const create = {
      configName: "主域",
      serverAddress: "dc01.example.com",
      serverPort: 636,
      domainName: "example.com",
      baseDn: "DC=example,DC=com",
    };
    await createADConfig(create);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ad-domain/configs", create);
    await getADConfig("c1");
    expect(mockGet).toHaveBeenCalledWith("/ad-domain/configs/c1");
    await updateADConfig("c1", {
      ...create,
      useSsl: true,
      useTls: false,
      syncEnabled: true,
      syncInterval: 60,
    });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ad-domain/configs/c1/update", {
      ...create,
      useSsl: true,
      useTls: false,
      syncEnabled: true,
      syncInterval: 60,
    });
    await deleteADConfig("c1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ad-domain/configs/c1/delete");
    await testADConnection("c1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/ad-domain/configs/c1/test", {});
    await syncADData("c1");
    expect(mockPost).toHaveBeenNthCalledWith(5, "/ad-domain/configs/c1/sync", { syncType: "full" });
    await syncADData("c1", "incremental");
    expect(mockPost).toHaveBeenNthCalledWith(6, "/ad-domain/configs/c1/sync", {
      syncType: "incremental",
    });
  });

  it("getADOUTree POST /ad-domain/ous/tree", async () => {
    mockPost.mockResolvedValueOnce(OK);
    await getADOUTree("c1");
    expect(mockPost).toHaveBeenCalledWith("/ad-domain/ous/tree", { configId: "c1" });
  });
});

describe("adDomainApi — 用户组", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
  });

  it("getADGroupList 注入默认分页", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const params = { configId: "c1", groupName: "运维" };
    await getADGroupList(params);
    expect(mockPost).toHaveBeenCalledWith("/ad-domain/groups/list", {
      configId: "c1",
      groupName: "运维",
      current: 1,
      pageSize: 10,
    });
  });

  it("getADGroupDetail GET /:id,updateADGroup POST /:id/update 携带 configId", async () => {
    mockGet.mockResolvedValueOnce(OK);
    mockPost.mockResolvedValueOnce(OK);
    await getADGroupDetail("g1");
    expect(mockGet).toHaveBeenCalledWith("/ad-domain/groups/g1");
    await updateADGroup("g1", "c1", { groupName: "改名" });
    expect(mockPost).toHaveBeenCalledWith("/ad-domain/groups/g1/update", {
      configId: "c1",
      groupName: "改名",
    });
  });

  it("成员管理:getADGroupMembers(默认分页)/addADGroupMember/removeADGroupMember", async () => {
    mockPost.mockResolvedValue(OK);
    await getADGroupMembers("g1", "c1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ad-domain/groups/g1/members", {
      configId: "c1",
      current: 1,
      pageSize: 10,
    });
    await addADGroupMember("g1", "c1", "CN=u1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ad-domain/groups/g1/members/add", {
      configId: "c1",
      userDn: "CN=u1",
    });
    await removeADGroupMember("g1", "c1", "CN=u1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ad-domain/groups/g1/members/remove", {
      configId: "c1",
      userDn: "CN=u1",
    });
  });

  it("组同步:syncADGroups / syncADSingleGroup / getADGroupSyncStatus", async () => {
    mockPost.mockResolvedValue(OK);
    await syncADGroups("c1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ad-domain/groups/sync-by-config/c1", {});
    await syncADSingleGroup("c1", "CN=g1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ad-domain/groups/sync-single", {
      configId: "c1",
      groupDn: "CN=g1",
    });
    await getADGroupSyncStatus("c1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ad-domain/groups/sync-status", {
      configId: "c1",
    });
  });
});

describe("adDomainApi — 用户与电脑", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
  });

  it("getADUserList / getADUserIds / getADUserDetail / updateADUser", async () => {
    mockPost.mockResolvedValue(OK);
    await getADUserList({ configId: "c1" });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ad-domain/users/list", {
      configId: "c1",
      current: 1,
      pageSize: 10,
    });
    // ids 端点不注入默认分页(全量 ID 导出)
    await getADUserIds({ configId: "c1" });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ad-domain/users/ids", { configId: "c1" });
    await getADUserDetail("u1", "c1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ad-domain/users/u1", { configId: "c1" });
    await updateADUser("u1", "c1", { displayName: "新名" });
    expect(mockPost).toHaveBeenNthCalledWith(4, "/ad-domain/users/u1/update", {
      configId: "c1",
      update: { displayName: "新名" },
    });
  });

  it("moveADUser / enableADUser / disableADUser", async () => {
    mockPost.mockResolvedValue(OK);
    await moveADUser("u1", "c1", "OU=New,DC=x");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ad-domain/users/u1/move", {
      configId: "c1",
      move: { newOuDn: "OU=New,DC=x" },
    });
    await enableADUser("u1", "c1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ad-domain/users/u1/enable", { configId: "c1" });
    await disableADUser("u1", "c1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ad-domain/users/u1/disable", { configId: "c1" });
  });

  it("getADSyncLogs:configId 可空 + 默认分页", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValueOnce(OK);
    await getADSyncLogs(undefined);
    expect(mockPost).toHaveBeenCalledWith("/ad-domain/logs/list", {
      configId: undefined,
      current: 1,
      pageSize: 10,
    });
  });

  it("getADComputerList / getADComputerDetail", async () => {
    mockPost.mockResolvedValue(OK);
    await getADComputerList({ configId: "c1" });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ad-domain/computers/list", {
      configId: "c1",
      current: 1,
      pageSize: 10,
    });
    await getADComputerDetail("c1", "CN=PC1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ad-domain/computers/detail", {
      configId: "c1",
      computerDn: "CN=PC1",
    });
  });
});

describe("adDomainApi — 部门组映射(legacy)", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
  });

  it("getMappingList 注入默认分页;getMapping GET /:id", async () => {
    mockPost.mockResolvedValueOnce(OK);
    mockGet.mockResolvedValueOnce(OK);
    await getMappingList({ adConfigId: "c1" });
    expect(mockPost).toHaveBeenCalledWith("/ad-domain/mappings/list", {
      adConfigId: "c1",
      current: 1,
      pageSize: 10,
    });
    await getMapping("m1");
    expect(mockGet).toHaveBeenCalledWith("/ad-domain/mappings/m1");
  });

  it("createMapping POST 根端点;updateMapping POST /:id/update", async () => {
    mockPost.mockResolvedValue(OK);
    const create = { deptId: "d1", adGroupId: "g1", adConfigId: "c1" };
    await createMapping(create);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ad-domain/mappings", create);
    await updateMapping("m1", { syncEnabled: false });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ad-domain/mappings/m1/update", {
      syncEnabled: false,
    });
  });

  it("deleteMapping 按 actual 锁定(URL 含历史笔误 '}',后端无对应路由,见 deferred-items)", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValueOnce(OK);
    await deleteMapping("m1");
    expect(mockPost).toHaveBeenCalledWith("/ad-domain/mappings/m1/delete}", {});
  });
});

describe("adDomainApi — OU 部门/组映射", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
  });

  it("getOUDeptMapping 对 ouDn 做 encodeURIComponent", async () => {
    mockGet.mockResolvedValueOnce(OK);
    await getOUDeptMapping("OU=一线,DC=x");
    expect(mockGet).toHaveBeenCalledWith(
      `/ad-domain/ou/${encodeURIComponent("OU=一线,DC=x")}/dept-mapping`
    );
  });

  it("updateOUDeptMapping POST 同路径携带 deptId", async () => {
    mockPost.mockResolvedValueOnce(OK);
    await updateOUDeptMapping("OU=一线,DC=x", { deptId: "d1" });
    expect(mockPost).toHaveBeenCalledWith(
      `/ad-domain/ou/${encodeURIComponent("OU=一线,DC=x")}/dept-mapping`,
      { deptId: "d1" }
    );
  });

  it("OU 组映射 CRUD + 按 OU 查询", async () => {
    mockPost.mockResolvedValue(OK);
    mockGet.mockResolvedValue(OK);
    await getOUGroupMappings({ adConfigId: "c1" });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ad-domain/ou-group-mappings/list", {
      adConfigId: "c1",
      current: 1,
      pageSize: 10,
    });
    await getOUGroupMappingsByOU("OU=x");
    expect(mockGet).toHaveBeenCalledWith(
      `/ad-domain/ou-group-mappings/ou/${encodeURIComponent("OU=x")}`
    );
    const create = { adConfigId: "c1", ouDn: "OU=x", ouName: "x", adGroupId: "g1" };
    await createOUGroupMapping(create);
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ad-domain/ou-group-mappings", create);
    await updateOUGroupMapping("m1", { syncEnabled: true });
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ad-domain/ou-group-mappings/m1/update", {
      syncEnabled: true,
    });
    await deleteOUGroupMapping("m1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/ad-domain/ou-group-mappings/m1/delete", {});
  });
});

describe("adDomainApi — 批量同步与服务账号池", () => {
  beforeEach(() => {
    mockGet.mockReset();
    mockPost.mockReset();
  });

  it("batchSyncADUsersDirect POST /ad-domain/users/batch-sync", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const data = { configId: "c1", userDns: ["CN=u1"], defaultRoleId: "r1" };
    await batchSyncADUsersDirect(data);
    expect(mockPost).toHaveBeenCalledWith("/ad-domain/users/batch-sync", data);
  });

  it("服务账号池 8 端点(Phase 36)", async () => {
    mockPost.mockResolvedValue(OK);
    await listADServiceAccounts({ configId: "c1" });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/ad-domain/accounts/list", { configId: "c1" });
    await createADServiceAccount({ configId: "c1", username: "svc-adm", password: "fake-pass" });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/ad-domain/accounts/create", {
      configId: "c1",
      username: "svc-adm",
      password: "fake-pass",
    });
    await updateADServiceAccount({ id: "a1", username: "svc-adm2" });
    expect(mockPost).toHaveBeenNthCalledWith(3, "/ad-domain/accounts/update", {
      id: "a1",
      username: "svc-adm2",
    });
    await deleteADServiceAccount("a1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/ad-domain/accounts/delete", { id: "a1" });
    await enableADServiceAccount("a1");
    expect(mockPost).toHaveBeenNthCalledWith(5, "/ad-domain/accounts/enable", { id: "a1" });
    await disableADServiceAccount("a1");
    expect(mockPost).toHaveBeenNthCalledWith(6, "/ad-domain/accounts/disable", { id: "a1" });
    await unlockADServiceAccount({ id: "a1", reason: "管理员确认账号已解锁，可恢复使用" });
    expect(mockPost).toHaveBeenNthCalledWith(7, "/ad-domain/accounts/unlock", {
      id: "a1",
      reason: "管理员确认账号已解锁，可恢复使用",
    });
    await getADServiceAccountStats("c1");
    expect(mockPost).toHaveBeenNthCalledWith(8, "/ad-domain/accounts/stats", { configId: "c1" });
  });
});
