/**
 * Phase 69 DICT-03 — src/constants/status.ts 自洽性单元测试
 *
 * 锁定行为（对齐后端 internal/models 常量注释，前端镜像的唯一定期校验点）:
 * - ENABLE_DISABLE 组: 0=启用 / 1=禁用（对齐 models.UserStatus, base.go）
 * - NORMAL_STOP 组: 0=正常 / 1=停用（对齐 models.RoleStatus/DeptStatus/PostStatus/MenuStatus, base.go）
 * - WORKSTATION_STATUS 组: 0=空闲 / 1=占用 / 2=维护（对齐 models.WorkstationStatus, workstation.go 三态簇）
 * - 每组 options 的 value 唯一且完整覆盖；tag 配置键集合与 options value 集合一致；
 *   tag text 与 options label 一致（防止两处文案漂移）
 *
 * 后端侧对齐由 internal/models/status_constants_test.go AST 锁值测试双向守卫。
 * Source: 69-06-PLAN.md Task 1 acceptance_criteria
 */
import { describe, it, expect } from "vitest";
import {
  ENABLE_DISABLE_OPTIONS,
  ENABLE_DISABLE_TAG_CONFIG,
  NORMAL_STOP_OPTIONS,
  NORMAL_STOP_TAG_CONFIG,
  WORKSTATION_STATUS_OPTIONS,
  WORKSTATION_STATUS_TAG_CONFIG,
  type StatusOption,
  type StatusTagConfig,
} from "./status";

/** 断言一组 options 的 value 集合与 label 序列精确锁定（数值 + 中文字面） */
function expectOptions(options: StatusOption[], expected: Array<{ label: string; value: number }>) {
  expect(options).toEqual(expected);
}

/** 断言 tag 配置键集合 = options value 集合，且 text 与对应 label 一致、color 非空 */
function expectTagConfigAligned(options: StatusOption[], config: StatusTagConfig) {
  const optionValues = options.map((o) => o.value).sort();
  const configKeys = Object.keys(config).map(Number).sort();
  expect(configKeys).toEqual(optionValues);

  for (const opt of options) {
    const tag = config[opt.value];
    expect(tag, `tag config missing for value ${opt.value}`).toBeDefined();
    expect(tag.text).toBe(opt.label);
    expect(typeof tag.color).toBe("string");
    expect(tag.color.length).toBeGreaterThan(0);
  }
}

describe("shared status constants (Phase 69 DICT-03)", () => {
  describe("ENABLE_DISABLE — 对齐 models.UserStatus (base.go)", () => {
    it("options lock 0=启用 / 1=禁用 (literal alignment with backend comment)", () => {
      expectOptions(ENABLE_DISABLE_OPTIONS, [
        { label: "启用", value: 0 },
        { label: "禁用", value: 1 },
      ]);
    });

    it("values are unique and cover exactly {0, 1}", () => {
      const values = ENABLE_DISABLE_OPTIONS.map((o) => o.value);
      expect(new Set(values).size).toBe(values.length);
      expect([...new Set(values)].sort()).toEqual([0, 1]);
    });

    it("tag config keys match option values and texts match labels", () => {
      expectTagConfigAligned(ENABLE_DISABLE_OPTIONS, ENABLE_DISABLE_TAG_CONFIG);
    });
  });

  describe("NORMAL_STOP — 对齐 models.RoleStatus/DeptStatus/PostStatus/MenuStatus (base.go)", () => {
    it("options lock 0=正常 / 1=停用 (literal alignment with backend comments)", () => {
      expectOptions(NORMAL_STOP_OPTIONS, [
        { label: "正常", value: 0 },
        { label: "停用", value: 1 },
      ]);
    });

    it("values are unique and cover exactly {0, 1}", () => {
      const values = NORMAL_STOP_OPTIONS.map((o) => o.value);
      expect(new Set(values).size).toBe(values.length);
      expect([...new Set(values)].sort()).toEqual([0, 1]);
    });

    it("tag config keys match option values and texts match labels", () => {
      expectTagConfigAligned(NORMAL_STOP_OPTIONS, NORMAL_STOP_TAG_CONFIG);
    });
  });

  describe("WORKSTATION_STATUS — 对齐 models.WorkstationStatus (workstation.go 三态簇)", () => {
    it("options lock 0=空闲 / 1=占用 / 2=维护 (三态业务簇，严禁两态化)", () => {
      expectOptions(WORKSTATION_STATUS_OPTIONS, [
        { label: "空闲", value: 0 },
        { label: "占用", value: 1 },
        { label: "维护", value: 2 },
      ]);
    });

    it("values are unique and cover exactly {0, 1, 2}", () => {
      const values = WORKSTATION_STATUS_OPTIONS.map((o) => o.value);
      expect(new Set(values).size).toBe(values.length);
      expect([...new Set(values)].sort()).toEqual([0, 1, 2]);
    });

    it("tag config keys match option values and texts match labels", () => {
      expectTagConfigAligned(WORKSTATION_STATUS_OPTIONS, WORKSTATION_STATUS_TAG_CONFIG);
    });
  });

  describe("两组通用启停语义组不得互相漂移（0/1 语义恒定）", () => {
    it("both generic groups use value 0 for the affirmative state (启用/正常)", () => {
      expect(ENABLE_DISABLE_OPTIONS[0].value).toBe(0);
      expect(NORMAL_STOP_OPTIONS[0].value).toBe(0);
    });

    it("both generic groups use value 1 for the negative state (禁用/停用)", () => {
      expect(ENABLE_DISABLE_OPTIONS[1].value).toBe(1);
      expect(NORMAL_STOP_OPTIONS[1].value).toBe(1);
    });

    it("group labels are distinct across the two generic groups (启用/禁用 vs 正常/停用)", () => {
      const ed = ENABLE_DISABLE_OPTIONS.map((o) => o.label);
      const ns = NORMAL_STOP_OPTIONS.map((o) => o.label);
      expect(ed).not.toEqual(ns);
    });
  });
});
