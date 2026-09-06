/**
 * rpaApi 存活契约基线测试 (Phase 100 V130R-10/11)
 *
 * 锁定的是「后端已注册路由的实测存活集」(非前端单方面 URL 声明):
 * 真相源 internal/api/v1/rpa/rpa_router.go;全族 116 方法对账后端态
 * 17 方法 (task 6 / worker 2 / execution 5 / ai 4)。对账台账:
 * .planning/phases/100-frontend-contract-fixes/RECONCILIATION.md。
 * 末尾 keys 断言与 apiFactory.invariants.test.ts 的 POST_CLEANUP_BASELINE
 * 数值一致 (两处互指,方法回增/私删双向即红)。
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

import { aiApi, executionApi, rpaApi, taskApi, workerApi } from "./rpaApi";

const OK = { code: 0 };

describe("rpaApi 存活契约 — task (6 方法, rpa_router.go:48-53)", () => {
  beforeEach(() => mockPost.mockReset());

  it("list/get/create/update/delete 使用 /rpa/tasks 基座", async () => {
    mockPost.mockResolvedValue(OK);
    const params = { current: 1, pageSize: 10, name: "巡检" };
    await taskApi.list(params);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/tasks/list", params);
    await taskApi.get("t1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/tasks/t1", {});
    await taskApi.create({ name: "新任务" });
    expect(mockPost).toHaveBeenNthCalledWith(3, "/rpa/tasks", { name: "新任务" });
    await taskApi.update("t1", { name: "改名" });
    expect(mockPost).toHaveBeenNthCalledWith(4, "/rpa/tasks/t1/update", { name: "改名" });
    await taskApi.delete("t1");
    expect(mockPost).toHaveBeenNthCalledWith(5, "/rpa/tasks/t1/delete", {});
  });

  it("execute 携带 variables", async () => {
    mockPost.mockResolvedValue(OK);
    await taskApi.execute("t1", { env: "prod" });
    expect(mockPost).toHaveBeenCalledWith("/rpa/tasks/t1/execute", {
      variables: { env: "prod" },
    });
  });
});

describe("rpaApi 存活契约 — worker (2 方法, rpa_router.go:64/:66)", () => {
  beforeEach(() => mockPost.mockReset());

  it("list + statistics(D-100-9 无参收窄版)", async () => {
    mockPost.mockResolvedValue(OK);
    await workerApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/workers/list", { current: 1, pageSize: 10 });
    await workerApi.statistics();
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/workers/statistics", {});
  });
});

describe("rpaApi 存活契约 — execution (5 方法, rpa_router.go:84-89)", () => {
  beforeEach(() => mockPost.mockReset());

  it("list/get/statistics 工厂 pick + cancel/logs 手写", async () => {
    mockPost.mockResolvedValue(OK);
    const params = { current: 1, pageSize: 10 };
    await executionApi.list(params);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/executions/list", params);
    await executionApi.get("e1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/executions/e1", {});
    await executionApi.statistics();
    expect(mockPost).toHaveBeenNthCalledWith(3, "/rpa/executions/statistics", {});
    await executionApi.cancel("e1", "手动取消");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/rpa/executions/e1/cancel", {
      reason: "手动取消",
    });
    await executionApi.logs("e1", { current: 1, pageSize: 5 });
    expect(mockPost).toHaveBeenNthCalledWith(5, "/rpa/executions/e1/logs", {
      current: 1,
      pageSize: 5,
    });
  });
});

describe("rpaApi 存活契约 — ai (4 方法, rpa_router.go:101-108)", () => {
  beforeEach(() => mockPost.mockReset());

  it("generate/optimize/decide/analyze-failure 端点", async () => {
    mockPost.mockResolvedValue(OK);
    await aiApi.generateScript({ description: "打开浏览器并登录" } as never);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/ai/generate", {
      description: "打开浏览器并登录",
    });
    await aiApi.optimizeScript({ script: {} } as never);
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/ai/optimize", { script: {} });
    const decide = {
      taskDescription: "弹窗出现",
      currentStep: 1,
      failedAction: {},
      availableSelectors: [],
    } as never;
    await aiApi.decide(decide);
    expect(mockPost).toHaveBeenNthCalledWith(3, "/rpa/ai/decide", decide);
    await aiApi.analyzeFailure({
      taskDescription: "x",
      failedStep: 1,
      error: "timeout",
    } as never);
    expect(mockPost).toHaveBeenNthCalledWith(4, "/rpa/ai/analyze-failure", {
      taskDescription: "x",
      failedStep: 1,
      error: "timeout",
    });
  });
});

describe("rpaApi 聚合对象结构 (D-100-2 keys 基线)", () => {
  it("rpaApi 暴露 4 个子 API", () => {
    expect(Object.keys(rpaApi).sort()).toEqual(["ai", "execution", "task", "worker"]);
  });

  it("各子对象方法集 == 后端已注册路由实测存活集 (与 invariants 基线互指)", () => {
    expect(Object.keys(taskApi).sort()).toEqual([
      "create",
      "delete",
      "execute",
      "get",
      "list",
      "update",
    ]);
    expect(Object.keys(workerApi).sort()).toEqual(["list", "statistics"]);
    expect(Object.keys(executionApi).sort()).toEqual([
      "cancel",
      "get",
      "list",
      "logs",
      "statistics",
    ]);
    expect(Object.keys(aiApi).sort()).toEqual([
      "analyzeFailure",
      "decide",
      "generateScript",
      "optimizeScript",
    ]);
  });
});
