/**
 * Phase 84 84-03a — HealthBadge 静态断言(mock react-query)
 */
import { describe, it, expect, vi } from "vitest";
import { HealthBadge } from "../HealthBadge";

vi.mock("@tanstack/react-query", () => ({
  useQuery: vi.fn(() => ({
    data: { status: "healthy", color: "#52c41a" },
    isLoading: false,
  })),
}));

describe("HealthBadge", () => {
  it("imports HealthBadge module (memo wrapped object)", () => {
    expect(HealthBadge).toBeDefined();
    expect(typeof HealthBadge).toBe("object");
  });

  it("memo component has displayName or function body", () => {
    // React.memo wraps component as $$typeof object
    expect((HealthBadge as any).$$typeof).toBeDefined();
  });
});
