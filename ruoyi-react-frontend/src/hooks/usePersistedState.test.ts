/**
 * usePersistedState hook 测试
 *
 * 验证 sessionStorage 持久化与跨实例同步行为
 */
import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { usePersistedState } from "./usePersistedState";

describe("usePersistedState", () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
  });

  it("returns defaultValue when storage is empty", () => {
    const { result } = renderHook(() =>
      usePersistedState<string>({
        keyPrefix: "/system/user",
        keySuffix: "selectedDeptId",
        defaultValue: "",
      })
    );
    expect(result.current.value).toBe("");
  });

  it("persists value to sessionStorage on setValue", () => {
    const { result } = renderHook(() =>
      usePersistedState<string>({
        keyPrefix: "/system/user",
        keySuffix: "selectedDeptId",
        defaultValue: "",
      })
    );
    act(() => result.current.setValue("dept-123"));
    expect(sessionStorage.getItem("ruoyi_table_state_system_user_selectedDeptId")).toBe(
      JSON.stringify("dept-123")
    );
  });

  it("restores value from sessionStorage on remount", () => {
    sessionStorage.setItem(
      "ruoyi_table_state_system_user_selectedDeptId",
      JSON.stringify("dept-abc")
    );
    const { result } = renderHook(() =>
      usePersistedState<string>({
        keyPrefix: "/system/user",
        keySuffix: "selectedDeptId",
        defaultValue: "",
      })
    );
    expect(result.current.value).toBe("dept-abc");
  });

  it("reset() clears storage and restores default", () => {
    sessionStorage.setItem(
      "ruoyi_table_state_system_user_selectedDeptId",
      JSON.stringify("dept-abc")
    );
    const { result } = renderHook(() =>
      usePersistedState<string>({
        keyPrefix: "/system/user",
        keySuffix: "selectedDeptId",
        defaultValue: "",
      })
    );
    act(() => result.current.reset());
    expect(result.current.value).toBe("");
    expect(sessionStorage.getItem("ruoyi_table_state_system_user_selectedDeptId")).toBeNull();
  });

  it("supports functional setValue updates", () => {
    const { result } = renderHook(() =>
      usePersistedState<number>({
        keyPrefix: "/test",
        keySuffix: "counter",
        defaultValue: 0,
      })
    );
    act(() => result.current.setValue((prev) => prev + 1));
    act(() => result.current.setValue((prev) => prev + 1));
    expect(result.current.value).toBe(2);
  });

  it("handles corrupted storage data gracefully", () => {
    sessionStorage.setItem("ruoyi_table_state_test_corrupted", "not-valid-json{{{");
    const { result } = renderHook(() =>
      usePersistedState<string>({
        keyPrefix: "/test",
        keySuffix: "corrupted",
        defaultValue: "fallback",
      })
    );
    expect(result.current.value).toBe("fallback");
    // corrupted entry should have been cleaned up
    expect(sessionStorage.getItem("ruoyi_table_state_test_corrupted")).toBeNull();
  });
});
