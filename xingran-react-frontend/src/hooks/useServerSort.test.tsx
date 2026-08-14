/**
 * useServerSort 持久化集成测试
 *
 * 验证接入 usePersistedStateController 后的行为：
 * - handleTableChange 持久化 orderByColumn / isAsc
 * - remount 从 sessionStorage 恢复排序，sortOrder 正确派生
 */
import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";
import { useServerSort } from "./useServerSort";

const wrapper =
  (initialPath: string) =>
  ({ children }: { children: ReactNode }) => (
    <MemoryRouter initialEntries={[initialPath]}>{children}</MemoryRouter>
  );

// 宽松构造 sorterMetas：handleTableChange 仅读取 m.field
const sorterMetas = [{ field: "username" }] as unknown as Array<{ field: string } | undefined>;

describe("useServerSort 持久化", () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
  });

  it("handleTableChange 持久化 orderByColumn / isAsc", () => {
    const { result } = renderHook(() => useServerSort({ sorterMetas }), {
      wrapper: wrapper("/system/user"),
    });
    act(() => {
      result.current.handleTableChange({} as never, {}, {
        field: "username",
        order: "descend",
      } as never);
    });
    expect(result.current.orderByColumn).toBe("username");
    expect(result.current.isAsc).toBe(false);
    expect(result.current.sortOrder).toBe("descend");
    expect(sessionStorage.getItem("xingran_table_state_system_user_orderByColumn")).toBe(
      JSON.stringify("username")
    );
  });

  it("remount 恢复排序，sortOrder 正确派生", () => {
    sessionStorage.setItem(
      "xingran_table_state_system_user_orderByColumn",
      JSON.stringify("username")
    );
    sessionStorage.setItem("xingran_table_state_system_user_isAsc", "false");
    const { result } = renderHook(() => useServerSort({ sorterMetas }), {
      wrapper: wrapper("/system/user"),
    });
    expect(result.current.orderByColumn).toBe("username");
    expect(result.current.isAsc).toBe(false);
    expect(result.current.sortOrder).toBe("descend");
  });

  it("resetSort 清空排序", () => {
    const { result } = renderHook(() => useServerSort({ sorterMetas }), {
      wrapper: wrapper("/system/user"),
    });
    act(() => {
      result.current.handleTableChange({} as never, {}, {
        field: "username",
        order: "ascend",
      } as never);
    });
    act(() => result.current.resetSort());
    expect(result.current.orderByColumn).toBeUndefined();
    expect(result.current.sortOrder).toBeNull();
  });
});
