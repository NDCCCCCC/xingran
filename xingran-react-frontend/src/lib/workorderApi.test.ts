/**
 * workorderApi 端点契约测试 (Phase 83-03)
 *
 * 锁定:工单 CRUD / 指派 / 状态 / 评论 / 评价 / 分类 / 周期模板 / 配置
 * 各端点 URL 与请求体形状,以及 buildCategoryTree 纯函数。
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

import {
  addWorkOrderComment,
  assignToTodayDuty,
  assignWorkOrder,
  batchDeleteWorkOrders,
  buildCategoryTree,
  createWorkOrder,
  createWorkOrderRating,
  deletePeriodicTemplate,
  deleteWorkOrder,
  disablePeriodicTemplate,
  enablePeriodicTemplate,
  generateWorkOrderNow,
  getMyPendingWorkOrders,
  getPeriodicLogs,
  getPeriodicTemplateList,
  getRatingStatistics,
  getWorkOrder,
  getWorkOrderRatings,
  getWorkOrderCategoryList,
  getWorkOrderComments,
  getWorkOrderConfig,
  getWorkOrderHistory,
  getWorkOrderList,
  getWorkOrderStatusStatistics,
  updatePeriodicTemplate,
  updateWorkOrder,
  updateWorkOrderConfig,
  updateWorkOrderStatus,
} from "./workorderApi";

const OK = { code: 0 };

describe("workorderApi — 工单主体", () => {
  beforeEach(() => mockPost.mockReset());

  it("getWorkOrderList POST /workorder/orders/list", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const params = { current: 1, pageSize: 10, title: "断网" };
    await getWorkOrderList(params);
    expect(mockPost).toHaveBeenCalledWith("/workorder/orders/list", params);
  });

  it("getWorkOrderStatusStatistics POST /workorder/orders/status-statistics", async () => {
    mockPost.mockResolvedValueOnce(OK);
    await getWorkOrderStatusStatistics();
    expect(mockPost).toHaveBeenCalledWith("/workorder/orders/status-statistics", {});
  });

  it("getMyPendingWorkOrders POST /workorder/orders/my-pending(params 可省略)", async () => {
    mockPost.mockResolvedValue(OK);
    await getMyPendingWorkOrders();
    expect(mockPost).toHaveBeenNthCalledWith(1, "/workorder/orders/my-pending", {});
    await getMyPendingWorkOrders({ limit: 5 });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/workorder/orders/my-pending", { limit: 5 });
  });

  it("getWorkOrderRatings POST /workorder/orders/:id/ratings", async () => {
    mockPost.mockReset();
    mockPost.mockResolvedValueOnce(OK);
    await getWorkOrderRatings("w1");
    expect(mockPost).toHaveBeenCalledWith("/workorder/orders/w1/ratings", {});
  });

  it("create / get / update / delete / batchDelete 按 ID 拼接", async () => {
    mockPost.mockResolvedValue(OK);
    const create = { title: "新建工单", categoryId: "c1", type: "fault", priority: 2 };
    await createWorkOrder(create);
    expect(mockPost).toHaveBeenNthCalledWith(1, "/workorder/orders", create);
    await getWorkOrder("w1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/workorder/orders/w1", {});
    await updateWorkOrder("w1", { title: "改标题" });
    expect(mockPost).toHaveBeenNthCalledWith(3, "/workorder/orders/w1/update", { title: "改标题" });
    await deleteWorkOrder("w1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/workorder/orders/w1/delete");
    await batchDeleteWorkOrders(["w1", "w2"]);
    expect(mockPost).toHaveBeenNthCalledWith(5, "/workorder/orders/batch-delete", {
      ids: ["w1", "w2"],
    });
  });

  it("assignWorkOrder / assignToTodayDuty / updateWorkOrderStatus", async () => {
    mockPost.mockResolvedValue(OK);
    await assignWorkOrder("w1", { assigneeId: "u1" });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/workorder/orders/w1/assign", {
      assigneeId: "u1",
    });
    await assignToTodayDuty("w1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/workorder/orders/w1/assign-duty");
    await updateWorkOrderStatus("w1", { status: 1, comment: "开始处理" });
    expect(mockPost).toHaveBeenNthCalledWith(3, "/workorder/orders/w1/status", {
      status: 1,
      comment: "开始处理",
    });
  });

  it("评论与历史", async () => {
    mockPost.mockResolvedValue(OK);
    await getWorkOrderComments("w1");
    expect(mockPost).toHaveBeenNthCalledWith(1, "/workorder/orders/w1/comments/list", {});
    await addWorkOrderComment("w1", { content: "已联系现场" });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/workorder/orders/w1/comments", {
      content: "已联系现场",
    });
    await getWorkOrderHistory("w1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/workorder/orders/w1/history", {});
  });

  it("评价:创建 / 统计", async () => {
    mockPost.mockResolvedValue(OK);
    await createWorkOrderRating("w1", { rating: 5, comment: "处理迅速" });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/workorder/orders/w1/rating", {
      rating: 5,
      comment: "处理迅速",
    });
    await getRatingStatistics("w1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/workorder/orders/w1/rating-statistics", {});
  });
});

describe("workorderApi — 分类与配置", () => {
  beforeEach(() => mockPost.mockReset());

  it("getWorkOrderCategoryList POST /workorder/categories/list", async () => {
    mockPost.mockResolvedValueOnce(OK);
    await getWorkOrderCategoryList();
    expect(mockPost).toHaveBeenCalledWith("/workorder/categories/list", {});
  });

  it("getWorkOrderConfig / updateWorkOrderConfig", async () => {
    mockPost.mockResolvedValue(OK);
    await getWorkOrderConfig();
    expect(mockPost).toHaveBeenNthCalledWith(1, "/workorder/config", {});
    await updateWorkOrderConfig({ autoAssign: true });
    expect(mockPost).toHaveBeenNthCalledWith(2, "/workorder/config/update", { autoAssign: true });
  });
});

describe("workorderApi — 周期性模板", () => {
  beforeEach(() => mockPost.mockReset());

  it("getPeriodicTemplateList POST /workorder/periodic/templates/list", async () => {
    mockPost.mockResolvedValueOnce(OK);
    const params = { current: 1, pageSize: 10 };
    await getPeriodicTemplateList(params);
    expect(mockPost).toHaveBeenCalledWith("/workorder/periodic/templates/list", params);
  });

  it("update / enable / disable / generate / logs 按 ID 拼接", async () => {
    mockPost.mockResolvedValue(OK);
    await updatePeriodicTemplate("t1", { name: "周报" });
    expect(mockPost).toHaveBeenNthCalledWith(1, "/workorder/periodic/templates/t1/update", {
      name: "周报",
    });
    await enablePeriodicTemplate("t1");
    expect(mockPost).toHaveBeenNthCalledWith(2, "/workorder/periodic/templates/t1/enable");
    await disablePeriodicTemplate("t1");
    expect(mockPost).toHaveBeenNthCalledWith(3, "/workorder/periodic/templates/t1/disable");
    await generateWorkOrderNow("t1");
    expect(mockPost).toHaveBeenNthCalledWith(4, "/workorder/periodic/templates/t1/generate");
    await getPeriodicLogs("t1");
    expect(mockPost).toHaveBeenNthCalledWith(5, "/workorder/periodic/templates/t1/logs", {});
    await deletePeriodicTemplate("t1");
    expect(mockPost).toHaveBeenNthCalledWith(6, "/workorder/periodic/templates/t1/delete");
  });
});

describe("workorderApi — buildCategoryTree 纯函数", () => {
  it("递归映射为 antd TreeSelect 节点,空 children 省略", () => {
    const categories = [
      {
        id: "c1",
        categoryName: "网络故障",
        children: [{ id: "c1-1", categoryName: "断网", children: [] }],
      },
      { id: "c2", categoryName: "桌面运维", children: undefined },
    ];

    const tree = buildCategoryTree(categories);

    expect(tree).toEqual([
      {
        title: "网络故障",
        value: "c1",
        key: "c1",
        children: [{ title: "断网", value: "c1-1", key: "c1-1", children: undefined }],
      },
      { title: "桌面运维", value: "c2", key: "c2", children: undefined },
    ]);
  });
});
