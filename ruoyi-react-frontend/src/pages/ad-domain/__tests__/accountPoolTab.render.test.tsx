/**
 * Phase 88 Batch28 — ad-domain AccountPoolTab 渲染测试(mock adDomainApi 账号池函数)
 */
import { describe, it, expect, vi } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

vi.mock("@/lib/adDomainApi", async (importOriginal) => {
  const actual = await importOriginal<typeof import("@/lib/adDomainApi")>();
  return {
    ...actual,
    listADServiceAccounts: vi.fn().mockResolvedValue({
      code: 0,
      data: {
        list: [
          {
            id: "acc-1",
            username: "svc_bind1",
            status: 0,
            failCount: 0,
            breakerUntil: null,
            lastUsedAt: "2026-08-28T09:00:00Z",
            createdAt: "2026-08-01T08:00:00Z",
          },
          {
            id: "acc-2",
            username: "svc_bind2",
            status: 2,
            failCount: 3,
            breakerUntil: "2026-08-28T12:00:00Z",
            createdAt: "2026-08-01T08:00:00Z",
          },
          {
            id: "acc-3",
            username: "svc_bind3",
            status: 1,
            failCount: 0,
            createdAt: "2026-08-01T08:00:00Z",
          },
        ],
        total: 3,
      },
    }),
    getADServiceAccountStats: vi.fn().mockResolvedValue({
      code: 0,
      data: { total: 3, available: 1, disabled: 1, breaker: 1 },
    }),
  };
});

import { renderWithProviders } from "@/test/utils/renderWithProviders";
import AccountPoolTab from "../configs/AccountPoolTab";
import { listADServiceAccounts, getADServiceAccountStats } from "@/lib/adDomainApi";

describe("ad-domain AccountPoolTab 渲染", () => {
  it("挂载拉取 list+stats,渲染统计卡与三状态 tag", async () => {
    const { container, findByText, findAllByText } = renderWithProviders(
      <AccountPoolTab configId="cfg-1" />,
      { route: "/ad-domain/configs" }
    );

    // 等表格出现(统计卡 + 表格)
    await vi.waitFor(
      () => {
        expect(container.querySelector(".ant-table")).not.toBeNull();
      },
      { timeout: 8000 }
    );
    expect(listADServiceAccounts).toHaveBeenCalled();
    expect(getADServiceAccountStats).toHaveBeenCalled();

    // 三状态 tag(统计卡同文案 → 全部 findAllByText)
    expect((await findAllByText("可用")).length).toBeGreaterThanOrEqual(2);
    expect((await findAllByText("熔断中")).length).toBeGreaterThanOrEqual(1);
    expect((await findAllByText("已停用")).length).toBeGreaterThanOrEqual(1);
  }, 15000);
});
