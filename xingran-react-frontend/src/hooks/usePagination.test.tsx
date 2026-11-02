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

  // UAT FAIL-1 回归（Phase 75）：模拟「切 tab → 切回」的卸载→重挂载场景。
  // 实机缺陷在页面数据加载层把恢复后的页码覆写回 1；本用例锁 hook 层
  // remount 恢复语义：重挂载必须回到持久化页码，且 sessionStorage 不被覆写。
  it("tab 切换卸载→重挂载：恢复持久化页码且 sessionStorage 未被覆写", () => {
    const { unmount } = renderHook(() => usePagination(), {
      wrapper: wrapper("/operations/buildings"),
    });
    // 首次挂载翻到第 2 页（持久化写入）
    sessionStorage.setItem("xingran_table_state_operations_buildings_current", "2");
    unmount(); // 切走 tab → 组件卸载

    // 切回 tab → 重挂载
    const second = renderHook(() => usePagination(), {
      wrapper: wrapper("/operations/buildings"),
    });
    expect(second.result.current.current).toBe(2); // 恢复页码，而非重置 1
    expect(sessionStorage.getItem("xingran_table_state_operations_buildings_current")).toBe("2"); // 未被覆写
  });
});
