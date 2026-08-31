/**
 * Phase 88 Batch316 — constants/storage 测试
 */
import { describe, it, expect, beforeEach } from "vitest";
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

  it("TABLE_STATE_PREFIX", () => {
    expect(TABLE_STATE_PREFIX).toBe("xingran_table_state_");
  });

  describe("sanitizePathForKey", () => {
    it("/system/user → system_user", () => {
      expect(sanitizePathForKey("/system/user")).toBe("system_user");
    });

    it("/ops/buildings → ops_buildings", () => {
      expect(sanitizePathForKey("/ops/buildings")).toBe("ops_buildings");
    });

    it("空字符串 → ''", () => {
      expect(sanitizePathForKey("")).toBe("");
    });

    it("无前导斜杠", () => {
      expect(sanitizePathForKey("foo/bar")).toBe("foo_bar");
    });

    it("多斜杠", () => {
      expect(sanitizePathForKey("/a/b/c")).toBe("a_b_c");
    });
  });

  describe("clearTableStateByPath", () => {
    it("只清理指定路径前缀", () => {
      sessionStorage.setItem(`${TABLE_STATE_PREFIX}foo_filter`, "x");
      sessionStorage.setItem(`${TABLE_STATE_PREFIX}foo_page`, "y");
      sessionStorage.setItem(`${TABLE_STATE_PREFIX}bar_filter`, "z");
      sessionStorage.setItem("other-key", "keep");

      clearTableStateByPath("/foo");

      expect(sessionStorage.getItem(`${TABLE_STATE_PREFIX}foo_filter`)).toBeNull();
      expect(sessionStorage.getItem(`${TABLE_STATE_PREFIX}foo_page`)).toBeNull();
      expect(sessionStorage.getItem(`${TABLE_STATE_PREFIX}bar_filter`)).toBe("z");
      expect(sessionStorage.getItem("other-key")).toBe("keep");
    });

    it("空路径 → 不清理", () => {
      sessionStorage.setItem(`${TABLE_STATE_PREFIX}keep`, "v");
      clearTableStateByPath("");
      expect(sessionStorage.getItem(`${TABLE_STATE_PREFIX}keep`)).toBe("v");
    });
  });

  describe("clearAllTableState", () => {
    it("清理所有 TABLE_STATE_PREFIX 开头的 key", () => {
      sessionStorage.setItem(`${TABLE_STATE_PREFIX}a`, "1");
      sessionStorage.setItem(`${TABLE_STATE_PREFIX}b`, "2");
      sessionStorage.setItem("other", "keep");

      clearAllTableState();

      expect(sessionStorage.getItem(`${TABLE_STATE_PREFIX}a`)).toBeNull();
      expect(sessionStorage.getItem(`${TABLE_STATE_PREFIX}b`)).toBeNull();
      expect(sessionStorage.getItem("other")).toBe("keep");
    });

    it("空 sessionStorage 不抛错", () => {
      expect(() => clearAllTableState()).not.toThrow();
    });
  });
});
