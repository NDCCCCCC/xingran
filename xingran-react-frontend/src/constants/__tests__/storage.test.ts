/**
 * Phase 88 Batch202 — constants/storage 测试
 */
import { describe, it, expect, beforeEach } from "vitest";

vi.mock("@/lib/api", async () => {
  const { createApiTestingModule } = await import("@/test/utils/createApiMock");
  return createApiTestingModule();
});

import {
  ZUSTAND_STORAGE_KEYS,
  STORAGE_KEYS,
  TABLE_STATE_PREFIX,
  sanitizePathForKey,
  clearTableStateByPath,
  clearAllTableState,
} from "../storage";

describe("constants/storage", () => {
  beforeEach(() => {
    sessionStorage.clear();
  });

  it("ZUSTAND_STORAGE_KEYS 3 项", () => {
    expect(ZUSTAND_STORAGE_KEYS.SETTINGS).toBe("settings-storage");
    expect(ZUSTAND_STORAGE_KEYS.LAYOUT).toBe("layout-storage");
    expect(ZUSTAND_STORAGE_KEYS.MENU).toBe("menu-storage");
  });

  it("STORAGE_KEYS.LAST_PATH", () => {
    expect(STORAGE_KEYS.LAST_PATH).toBe("xingran_last_visited_path");
  });

  it("TABLE_STATE_PREFIX 前缀", () => {
    expect(TABLE_STATE_PREFIX).toBe("xingran_table_state_");
  });

  it("sanitizePathForKey 去除前导 /", () => {
    expect(sanitizePathForKey("/system/user")).toBe("system_user");
  });

  it("sanitizePathForKey 替换中间 /", () => {
    expect(sanitizePathForKey("/ops/buildings")).toBe("ops_buildings");
  });

  it("sanitizePathForKey 多斜杠", () => {
    expect(sanitizePathForKey("//a/b/c")).toBe("a_b_c");
  });

  it("sanitizePathForKey 空 → ''", () => {
    expect(sanitizePathForKey("")).toBe("");
  });

  it("clearTableStateByPath 删该路径下 keys", () => {
    sessionStorage.setItem("xingran_table_state_system_user_filters", "x");
    sessionStorage.setItem("xingran_table_state_ops_buildings_pagination", "y");
    clearTableStateByPath("/system/user");
    expect(sessionStorage.getItem("xingran_table_state_system_user_filters")).toBeNull();
    expect(sessionStorage.getItem("xingran_table_state_ops_buildings_pagination")).toBe("y");
  });

  it("clearTableStateByPath 空路径不删", () => {
    sessionStorage.setItem("xingran_table_state_a_filters", "x");
    clearTableStateByPath("");
    expect(sessionStorage.getItem("xingran_table_state_a_filters")).toBe("x");
  });

  it("clearAllTableState 删全部", () => {
    sessionStorage.setItem("xingran_table_state_a_filters", "x");
    sessionStorage.setItem("xingran_table_state_b_pagination", "y");
    sessionStorage.setItem("other_key", "z");
    clearAllTableState();
    expect(sessionStorage.getItem("xingran_table_state_a_filters")).toBeNull();
    expect(sessionStorage.getItem("xingran_table_state_b_pagination")).toBeNull();
    expect(sessionStorage.getItem("other_key")).toBe("z");
  });
});
