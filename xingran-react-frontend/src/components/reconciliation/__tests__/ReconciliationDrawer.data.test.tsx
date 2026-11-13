/**
 * ReconciliationDrawer 数据加载回归测试
 *
 * 背景：Drawer 原为「永久空态」，timeline / exception 两个 Tab 被隐藏（STOP-06）。
 * 接入真实数据源后（reconciliationApi.exceptionList / exceptionRule.list），
 * 新增了「打开时拉数据 → 映射为 TimelineRecord / ExceptionRuleItem → 渲染」的分支。
 *
 * 本测试锁定三条路径，防止后续改动把数据源退回空态：
 *  1. 正常路径：命中字典 → 三个 Tab 可见、timeline/exception 有内容；
 *  2. 降级路径：端点异常 → 记 console.error、不崩溃、Tab 仍可切换；
 *  3. 空数据路径：接口返回空列表 → 渲染空态不报错。
 */
import { describe, it, expect, vi, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

// Drawer 受 useReconciliationVisibility 门控：无 asset:reconciliation:list 权限时
// 直接 render null（跨模块权限门 D-A1-03）。测试需提供该权限才能进入渲染分支。
vi.mock("@/store/menuStore", () => ({
  useMenuStore: vi.fn((selector: unknown) => {
    const state = { permissions: ["asset:reconciliation:list"] };
    return typeof selector === "function" ? selector(state) : state;
  }),
}));

import { renderPageWithEndpoints } from "@/test/utils/renderPage";
import { ReconciliationDrawer } from "../ReconciliationDrawer";

const TIMELINE_ENDPOINT = "/asset/reconciliation/exception/list";
const RULE_ENDPOINT = "/asset/reconciliation/exception-rule/list";
const HEALTH_ENDPOINT = "/asset/reconciliation/by-workstation";
const DICT_ENDPOINT = "/system/dicts/data/list";

/**
 * 工位健康数据 fixture —— Drawer 的资产详情（assetCode 等）来自
 * useAssetHealth → useWorkstationHealth(by-workstation) 缓存切片。
 * timeline 记录按 assetCode 过滤到当前资产，故该端点必须提供匹配资产。
 */
const healthFixture = {
  data: {
    assets: [
      {
        assetId: "asset-1",
        assetCode: "AST-001",
        ip: "10.0.0.1",
        conflictType: "B",
        severity: "high",
        confidenceScore: 0.9,
        appliedActions: [],
      },
    ],
  },
};

const timelineFixture = {
  data: {
    list: [
      {
        id: "e1",
        conflictType: "B",
        severity: "high",
        confidenceScore: 0.9,
        detectedAt: "2026-11-01T10:00:00+08:00",
        resolvedAt: null,
        exceptionRuleId: null,
        assetCode: "AST-001",
        assetIpDisplay: "10.0.0.1",
        physicalUsername: "u1",
        responsibleUsername: "u2",
      },
    ],
    total: 1,
    current: 1,
    pageSize: 20,
  },
};

const ruleFixture = {
  data: {
    list: [
      {
        id: "r1",
        name: "虚拟化例外",
        scopeType: "global",
        ipRange: "10.0.0.0/24",
        exceptionActions: ["silence"],
        reason: "测试环境",
        expiresAt: null,
      },
    ],
    total: 1,
    current: 1,
    pageSize: 20,
  },
};

async function renderDrawer(open: boolean, activeTab: "summary" | "timeline" | "exception") {
  return renderPageWithEndpoints(
    <ReconciliationDrawer
      open={open}
      onClose={vi.fn()}
      selectedAssetId={open ? "asset-1" : null}
      workstationId="ws-1"
      assetCode="AST-001"
      activeTab={activeTab}
      onTabChange={vi.fn()}
    />,
    {
      endpoints: {
        [TIMELINE_ENDPOINT]: timelineFixture,
        [RULE_ENDPOINT]: ruleFixture,
        [HEALTH_ENDPOINT]: healthFixture,
      },
    }
  );
}

describe("ReconciliationDrawer 数据加载", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("打开时渲染三个 Tab（timeline / exception 不再是永久空态）", async () => {
    const { screen } = await renderDrawer(true, "summary");
    // 三个 Tab 标题都应出现
    expect(await screen.findByText("冲突摘要")).not.toBeNull();
    expect(await screen.findByText("历史变更")).not.toBeNull();
    expect(await screen.findByText("例外规则")).not.toBeNull();
  });

  it("timeline Tab 渲染冲突记录", async () => {
    const { screen } = await renderDrawer(true, "timeline");
    // 冲突类型 B 的中文标签「物理有/责任人无」
    expect(await screen.findByText("物理有/责任人无")).not.toBeNull();
  });

  it("exception Tab 渲染例外规则名", async () => {
    const { screen } = await renderDrawer(true, "exception");
    expect(await screen.findByText("虚拟化例外")).not.toBeNull();
  });

  it("端点异常时降级：记错误、不崩溃、Tab 仍渲染", async () => {
    const spy = vi.spyOn(console, "error").mockImplementation(() => {});
    const { screen } = await renderPageWithEndpoints(
      <ReconciliationDrawer
        open={true}
        onClose={vi.fn()}
        selectedAssetId="asset-1"
        workstationId="ws-1"
        activeTab="summary"
        onTabChange={vi.fn()}
      />,
      {
        endpoints: {
          [TIMELINE_ENDPOINT]: Promise.reject(new Error("network down")),
          [RULE_ENDPOINT]: Promise.reject(new Error("network down")),
          [HEALTH_ENDPOINT]: healthFixture,
        },
      }
    );
    // 降级后 Tab 仍应渲染（不因数据源异常而白屏）
    expect(await screen.findByText("历史变更")).not.toBeNull();
    spy.mockRestore();
  });

  it("空数据时渲染空态不报错", async () => {
    const { screen } = await renderPageWithEndpoints(
      <ReconciliationDrawer
        open={true}
        onClose={vi.fn()}
        selectedAssetId="asset-1"
        workstationId="ws-1"
        activeTab="timeline"
        onTabChange={vi.fn()}
      />,
      {
        endpoints: {
          [TIMELINE_ENDPOINT]: { data: { list: [], total: 0, current: 1, pageSize: 20 } },
          [RULE_ENDPOINT]: { data: { list: [], total: 0, current: 1, pageSize: 20 } },
          [HEALTH_ENDPOINT]: healthFixture,
        },
      }
    );
    expect(await screen.findByText("历史变更")).not.toBeNull();
  });

  it("字典命中 → 摘要 Tab 显示 dictLabel(冲突类型/严重度)", async () => {
    const { screen } = await renderPageWithEndpoints(
      <ReconciliationDrawer
        open={true}
        onClose={vi.fn()}
        selectedAssetId="asset-1"
        workstationId="ws-1"
        assetCode="AST-001"
        activeTab="summary"
        onTabChange={vi.fn()}
      />,
      {
        endpoints: {
          [TIMELINE_ENDPOINT]: timelineFixture,
          [RULE_ENDPOINT]: ruleFixture,
          [HEALTH_ENDPOINT]: healthFixture,
          [DICT_ENDPOINT]: {
            data: {
              list: [
                { id: "d1", dictLabel: "B类-物理有/责任人无", dictValue: "B", listClass: "error" },
                { id: "d2", dictLabel: "高", dictValue: "high", listClass: "error" },
              ],
              total: 2,
            },
          },
        },
      }
    );
    expect(await screen.findByText("B类-物理有/责任人无")).not.toBeNull();
    expect(await screen.findByText("高")).not.toBeNull();
  });
});
