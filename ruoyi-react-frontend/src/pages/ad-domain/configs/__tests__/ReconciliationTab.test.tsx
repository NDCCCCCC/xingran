/**
 * Phase 59 / GUARD-02 — ReconciliationTab 组件测试（tracer 链路）
 *
 * 本文件覆盖：权限门（403 + 零请求，D-17/T-59-09）与空态文案。
 * 有数据渲染 / 错误态 / 筛选提交断言由后续任务在同类例中扩展。
 */
import { beforeEach, describe, expect, it, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/adDomainApi", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/adDomainApi")>();
  return {
    ...actual,
    getDeptOUMismatchList: vi.fn(),
  };
});

vi.mock("@/hooks/useDeptTree", () => ({
  useDeptTree: () => ({ data: [], isLoading: false }),
}));

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import ReconciliationTab, { RECONCILIATION_PERM } from "../ReconciliationTab";
import { getDeptOUMismatchList } from "@/lib/adDomainApi";
import { useMenuStore } from "@/store/menuStore";

const mockGetList = vi.mocked(getDeptOUMismatchList);
const PERM = RECONCILIATION_PERM;

const EMPTY_PAGE = { data: { list: [], total: 0 } };

describe("ReconciliationTab 权限门 (D-17)", () => {
  beforeEach(() => {
    mockGetList.mockReset();
    useMenuStore.setState({ permissions: [] });
  });

  it("无权限渲染 403 文案且不发起任何请求", async () => {
    const { findByText, queryByText } = renderWithProviders(<ReconciliationTab />);
    expect(await findByText("无对账查看权限")).toBeTruthy();
    expect(
      await findByText(
        "请联系管理员为当前角色授予 ops:ad:config:reconciliation:list 权限后重新登录。"
      )
    ).toBeTruthy();
    expect(mockGetList).not.toHaveBeenCalled();
    expect(queryByText("未发现不一致条目")).toBeNull();
  });

  it("有权限且返回空列表渲染空态『未发现不一致条目』并请求默认分页", async () => {
    useMenuStore.setState({ permissions: [PERM] });
    mockGetList.mockResolvedValue(EMPTY_PAGE);
    const { findByText, queryByText } = renderWithProviders(<ReconciliationTab />);
    expect(await findByText("未发现不一致条目")).toBeTruthy();
    expect(
      await findByText("当前视图为实时计算。可调整筛选条件，或先在「配置列表」完成部门同步后再查看。")
    ).toBeTruthy();
    // 空态不渲染空表头
    expect(queryByText("不一致类型")).toBeNull();
    expect(mockGetList).toHaveBeenCalledTimes(1);
    expect(mockGetList).toHaveBeenCalledWith({ current: 1, pageSize: 20 });
  });
});
