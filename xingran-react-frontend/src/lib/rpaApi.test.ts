/**
 * rpaApi 端点契约测试 (Phase 83-03)
 *
 * 锁定:CRUD 工厂基座(/rpa/{tasks,workers,executions,schedules,variables,templates,notifications})
 * + 任务执行链路 + AI 端点 + 统计端点 + rpaApi 聚合对象结构。
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

import { aiApi, rpaApi, scheduleApi, scriptApi, statisticsApi, taskApi, workerApi } from "./rpaApi";

const OK = { code: 0 };

describe("rpaApi CRUD 工厂基座", () => {
  beforeEach(() => mockPost.mockReset());

  it("taskApi.list/get/create/update/delete 使用 /rpa/tasks 基座", async () => {
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

  it("workerApi 基座 + register/heartbeat", async () => {
    mockPost.mockResolvedValue(OK);
    await workerApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/workers/list", { current: 1, pageSize: 10 });
    const register = { workerName: "node-1", os: "windows" };
    await workerApi.register(register);
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/workers/register", register);
    const heartbeat = { cpuUsage: 12.5, memoryUsage: 40 };
    await workerApi.heartbeat("w1", heartbeat);
    expect(mockPost).toHaveBeenNthCalledWith(3, "/rpa/workers/w1/heartbeat", heartbeat);
  });
});

describe("rpaApi — 任务执行链路", () => {
  beforeEach(() => mockPost.mockReset());

  it("execute 携带 variables;cancelExecution/duplicate/executions 按 ID 拼接", async () => {
    mockPost.mockResolvedValue(OK);
    await taskApi.execute("t1", { env: "prod" });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/tasks/t1/execute", {
      variables: { env: "prod" },
    });
    await taskApi.cancelExecution("t1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/tasks/t1/cancel", {});
    await taskApi.duplicate("t1", "副本");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/rpa/tasks/t1/duplicate", { newName: "副本" });
    const params = { current: 1, pageSize: 5 };
    await taskApi.executions("t1", params);
    expect(mockPost).toHaveBeenNthCalledWith(4, "/rpa/tasks/t1/executions", params);
  });

  it("脚本:validateScript/format/testAction", async () => {
    mockPost.mockResolvedValue(OK);
    const script = { name: "s", actions: [] };
    await taskApi.validateScript(script as never);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/tasks/validate-script", { script });
    await scriptApi.format({ name: "s" } as never);
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/scripts/format", { script: { name: "s" } });
    const action = { type: "click", target: "#btn" };
    await scriptApi.testAction(action as never, "https://target.example.com");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/rpa/scripts/test-action", {
      action,
      url: "https://target.example.com",
    });
  });

  it("脚本 CRUD 使用 /rpa/scripts 基座", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValue(OK);
    await scriptApi.list({ current: 1, pageSize: 10 });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/scripts/list", { current: 1, pageSize: 10 });
    await scriptApi.get("s1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/scripts/s1", {});
    await scriptApi.create({ name: "脚本" } as never);
    expect(mockPost).toHaveBeenNthCalledWith(3, "/rpa/scripts", { name: "脚本" });
    await scriptApi.update("s1", { name: "改" } as never);
    expect(mockPost).toHaveBeenNthCalledWith(4, "/rpa/scripts/s1/update", { name: "改" });
    await scriptApi.delete("s1");
    expect(mockPost).toHaveBeenNthCalledWith(5, "/rpa/scripts/s1/delete", {});
  });

  it("导入导出:export POST /:id/export,import POST /rpa/tasks/import", async () => {
    mockPost.mockResolvedValue(OK);
    await taskApi.export("t1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/tasks/t1/export", {});
    await taskApi.import("base64-payload");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/tasks/import", { data: "base64-payload" });
  });
});

describe("rpaApi — 调度/变量/模板", () => {
  beforeEach(() => mockPost.mockReset());

  it("scheduleApi:activate/pause/disable/run-now", async () => {
    mockPost.mockResolvedValue(OK);
    await scheduleApi.activate("sc1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/schedules/sc1/activate", {});
    await scheduleApi.pause("sc1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/schedules/sc1/pause", {});
    await scheduleApi.disable("sc1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/rpa/schedules/sc1/disable", {});
    await scheduleApi.runNow("sc1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/rpa/schedules/sc1/run-now", {});
  });

  it("variableApi:getGlobal / getByTask / batchSet / decrypt", async () => {
    mockPost.mockResolvedValue(OK);
    const variableApi = rpaApi.variable;
    await variableApi.getGlobal();
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/variables/global", {});
    await variableApi.getByTask("t1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/variables/task/t1", {});
    await variableApi.batchSet([{ name: "k", value: "v" }] as never);
    expect(mockPost).toHaveBeenNthCalledWith(3, "/rpa/variables/batch-set", {
      variables: [{ name: "k", value: "v" }],
    });
    await variableApi.decrypt("v1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/rpa/variables/v1/decrypt", {});
  });

  it("templateApi:categories/use/rate/favorite/unfavorite", async () => {
    mockPost.mockResolvedValue(OK);
    const templateApi = rpaApi.template;
    await templateApi.categories();
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/templates/categories", {});
    await templateApi.useTemplate("tp1", "我的任务");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/templates/tp1/use", { taskName: "我的任务" });
    await templateApi.rate("tp1", 5);
    expect(mockPost).toHaveBeenNthCalledWith(3, "/rpa/templates/tp1/rate", { rating: 5 });
    await templateApi.favorite("tp1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/rpa/templates/tp1/favorite", {});
    await templateApi.unfavorite("tp1");
    expect(mockPost).toHaveBeenNthCalledWith(5, "/rpa/templates/tp1/unfavorite", {});
  });
});

describe("rpaApi — AI 与统计", () => {
  beforeEach(() => mockPost.mockReset());

  it("aiApi:generate/decide 端点", async () => {
    mockPost.mockResolvedValue(OK);
    const request = { prompt: "打开浏览器并登录" };
    await aiApi.generateScript(request as never);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/ai/generate", request);
    const decide = { context: "弹窗出现", options: ["确认", "取消"] };
    await aiApi.decide(decide as never);
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/ai/decide", decide);
  });

  it("statisticsApi.overview POST /rpa/statistics/overview", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValueOnce(OK);
    await statisticsApi.overview();
    expect(mockPost).toHaveBeenCalledWith("/rpa/statistics/overview", {});
  });

  it("通知:enable/disable/test/global", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValue(OK);
    const notificationApi = rpaApi.notification;
    await notificationApi.enable("n1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/rpa/notifications/n1/enable", {});
    await notificationApi.disable("n1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/rpa/notifications/n1/disable", {});
    await notificationApi.test("n1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/rpa/notifications/n1/test", {});
    await notificationApi.getGlobal();
    expect(mockPost).toHaveBeenNthCalledWith(4, "/rpa/notifications/global", {});
  });
});

describe("rpaApi 聚合对象结构", () => {
  it("rpaApi 暴露 10 个子 API", () => {
    expect(Object.keys(rpaApi).sort()).toEqual(
      [
        "task",
        "script",
        "worker",
        "execution",
        "schedule",
        "variable",
        "template",
        "ai",
        "notification",
        "statistics",
      ].sort()
    );
  });
});
