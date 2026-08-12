/**
 * usePersistedState hook 测试
 *
 * 验证 sessionStorage 持久化与跨实例同步行为
 *
 * 注意:本文件测试的是读写入口 `usePersistedStateController`(返回 [value, setValue, reset] tuple)。
 * 只读入口 `usePersistedState` 由同模块的 usePersistedStateInternal 统一实现,共享相同的行为。
 */
import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { usePersistedStateController } from "./usePersistedState";

describe("usePersistedStateController", () => {
  beforeEach(() => {
    sessionStorage.clear();
    localStorage.clear();
  });

  it("returns defaultValue when storage is empty", () => {
    const { result } = renderHook(() =>
      usePersistedStateController<string>({
        keyPrefix: "/system/user",
        keySuffix: "selectedDeptId",
        defaultValue: "",
      })
    );
    expect(result.current[0]).toBe("");
  });

  it("persists value to sessionStorage on setValue", () => {
    const { result } = renderHook(() =>
      usePersistedStateController<string>({
        keyPrefix: "/system/user",
        keySuffix: "selectedDeptId",
        defaultValue: "",
      })
    );
    act(() => result.current[1]("dept-123"));
    expect(sessionStorage.getItem("xingran_table_state_system_user_selectedDeptId")).toBe(
      JSON.stringify("dept-123")
    );
  });

  it("restores value from sessionStorage on remount", () => {
    sessionStorage.setItem(
      "xingran_table_state_system_user_selectedDeptId",
      JSON.stringify("dept-abc")
    );
    const { result } = renderHook(() =>
      usePersistedStateController<string>({
        keyPrefix: "/system/user",
        keySuffix: "selectedDeptId",
        defaultValue: "",
      })
    );
    expect(result.current[0]).toBe("dept-abc");
  });

  it("reset() clears storage and restores default", () => {
    sessionStorage.setItem(
      "xingran_table_state_system_user_selectedDeptId",
      JSON.stringify("dept-abc")
    );
    const { result } = renderHook(() =>
      usePersistedStateController<string>({
        keyPrefix: "/system/user",
        keySuffix: "selectedDeptId",
        defaultValue: "",
      })
    );
    act(() => result.current[2]());
    expect(result.current[0]).toBe("");
    expect(sessionStorage.getItem("xingran_table_state_system_user_selectedDeptId")).toBeNull();
  });

  it("supports functional setValue updates", () => {
    const { result } = renderHook(() =>
      usePersistedStateController<number>({
        keyPrefix: "/test",
        keySuffix: "counter",
        defaultValue: 0,
      })
    );
    act(() => result.current[1]((prev: number) => prev + 1));
    act(() => result.current[1]((prev: number) => prev + 1));
    expect(result.current[0]).toBe(2);
  });

  it("handles corrupted storage data gracefully", () => {
    sessionStorage.setItem("xingran_table_state_test_corrupted", "not-valid-json{{{");
    const { result } = renderHook(() =>
      usePersistedStateController<string>({
        keyPrefix: "/test",
        keySuffix: "corrupted",
        defaultValue: "fallback",
      })
    );
    expect(result.current[0]).toBe("fallback");
    // corrupted entry should have been cleaned up
    expect(sessionStorage.getItem("xingran_table_state_test_corrupted")).toBeNull();
  });
});
