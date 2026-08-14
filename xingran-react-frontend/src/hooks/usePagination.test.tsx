/**
 * usePagination 持久化集成测试
 *
 * 验证接入 usePersistedStateController 后的行为：
 * - current / pageSize 按 location.pathname 隔离写入 sessionStorage
 * - remount 从 sessionStorage 恢复
 * - options.pageSize 硬约束覆盖持久化值
 * - reset 清理持久化
 */
import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";
import type { ReactNode } from "react";
import { usePagination } from "./usePagination";

const wrapper =
  (initialPath: string) =>
  ({ children }: { children: ReactNode }) => (
    <MemoryRouter initialEntries={[initialPath]}>{children}</MemoryRouter>
  );

describe("usePagination 持久化", () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
  });

  it("setCurrent 持久化到 sessionStorage（按 pathname 隔离）", () => {
    const { result } = renderHook(() => usePagination(), {
      wrapper: wrapper("/system/user"),
    });
    act(() => result.current.setCurrent(5));
    expect(result.current.current).toBe(5);
    expect(sessionStorage.getItem("xingran_table_state_system_user_current")).toBe("5");
  });

  it("remount 从 sessionStorage 恢复 current", () => {
    sessionStorage.setItem("xingran_table_state_system_user_current", "3");
    const { result } = renderHook(() => usePagination(), {
      wrapper: wrapper("/system/user"),
    });
    expect(result.current.current).toBe(3);
  });

  it("pageSize 持久化与恢复", () => {
    const { result } = renderHook(() => usePagination(), {
      wrapper: wrapper("/system/user"),
    });
    act(() => result.current.setPageSize(50));
    expect(sessionStorage.getItem("xingran_table_state_system_user_pageSize")).toBe("50");
  });

  it("options.pageSize 硬约束覆盖持久化值", () => {
    // 历史选择过 50
    sessionStorage.setItem("xingran_table_state_system_user_pageSize", "50");
    const { result } = renderHook(() => usePagination({ pageSize: 20 }), {
      wrapper: wrapper("/system/user"),
    });
    expect(result.current.pageSize).toBe(20);
  });

  it("不同 pathname 互不干扰（隔离性）", () => {
    sessionStorage.setItem("xingran_table_state_system_user_current", "7");
    const { result } = renderHook(() => usePagination(), {
      wrapper: wrapper("/system/role"),
    });
    // /system/role 没有 current 记录，应回默认值 1
    expect(result.current.current).toBe(1);
    expect(sessionStorage.getItem("xingran_table_state_system_role_current")).toBeNull();
  });

  it("reset 清理持久化", () => {
    const { result } = renderHook(() => usePagination(), {
      wrapper: wrapper("/system/user"),
    });
    act(() => result.current.setCurrent(7));
    act(() => result.current.reset());
    expect(sessionStorage.getItem("xingran_table_state_system_user_current")).toBeNull();
    expect(result.current.current).toBe(1);
  });
});
