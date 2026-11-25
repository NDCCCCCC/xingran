/**
 * Phase 156 / REC-03：前端动作枚举一致性测试
 *
 * 覆盖三方（前端子集）：
 *   1. 共享枚举 RPA_ACTION_TYPES（src/types/rpa.ts）—— 前端唯一真相源，
 *      与 rpa-worker/internal/types/types.go 的 ActionType 常量同源。
 *   2. 动词表 HUMAN_VERBS（wizard/stepLabels.ts）—— 面向员工的步骤措辞。
 *   3. 编辑器映射 ACTION_TYPE_TEXT_MAP / ACTION_TYPE_OPTIONS
 *      （operations/rpa/constants.tsx）—— 下拉选项与中文短名。
 *
 * 跨语言那一维（Go 侧 types.go ↔ 前端三张表）由 Go 侧
 * internal/services/rpa/action_type_consistency_test.go 的
 * TestActionTypeConsistency_GoFrontendStepLabels 三方测试锁住 ——
 * 它直接读前端源文件做比对，是「前端改了枚举但 Go 侧没跟上」的兜底闸门。
 */

import { describe, it, expect } from "vitest";
import { RPA_ACTION_TYPES, type ActionType } from "@/types/rpa";
import { HUMAN_VERBS } from "../stepLabels";
import { ACTION_TYPE_TEXT_MAP, ACTION_TYPE_OPTIONS } from "../../constants";

/** 动作全集规模（与 rpa-worker/internal/types/types.go 的常量个数一致）。 */
const EXPECTED_ACTION_TYPE_COUNT = 26;

/** 纯中文判定：不得含 ASCII 字母（错误信息/下拉项面向普通员工）。 */
const LATIN_LETTER = /[A-Za-z]/;

describe("REC-03 前端动作枚举单一真相源", () => {
  it("共享枚举 RPA_ACTION_TYPES 恰为 26 项且无重复", () => {
    expect(RPA_ACTION_TYPES.length).toBe(EXPECTED_ACTION_TYPE_COUNT);
    expect(new Set(RPA_ACTION_TYPES).size).toBe(EXPECTED_ACTION_TYPE_COUNT);
  });

  it("共享枚举不含 Go 侧不存在的 press", () => {
    // press 是历史双源漂移的产物（Go 侧对应的是 keyboard）
    expect(RPA_ACTION_TYPES as readonly string[]).not.toContain("press");
    expect(RPA_ACTION_TYPES as readonly string[]).toContain("keyboard");
  });

  it("动词表 HUMAN_VERBS 的键集恰等于共享枚举", () => {
    const verbs = Object.keys(HUMAN_VERBS).sort();
    const types = [...RPA_ACTION_TYPES].sort();
    expect(verbs).toEqual(types);
  });

  it("ACTION_TYPE_TEXT_MAP 覆盖共享枚举全部动作", () => {
    for (const type of RPA_ACTION_TYPES) {
      expect(Object.keys(ACTION_TYPE_TEXT_MAP)).toContain(type);
    }
    expect(Object.keys(ACTION_TYPE_TEXT_MAP).length).toBeGreaterThanOrEqual(
      EXPECTED_ACTION_TYPE_COUNT
    );
  });

  it("ACTION_TYPE_OPTIONS 从共享枚举派生（不再是手写的 9 项）", () => {
    expect(ACTION_TYPE_OPTIONS.length).toBe(EXPECTED_ACTION_TYPE_COUNT);
    expect(ACTION_TYPE_OPTIONS.map((o) => o.value).sort()).toEqual(
      [...RPA_ACTION_TYPES].sort()
    );
  });

  it("每个动作都有非空中文名，且中文名不含 ASCII 字母", () => {
    for (const type of RPA_ACTION_TYPES) {
      const label = ACTION_TYPE_TEXT_MAP[type];
      expect(label, `动作 ${type} 缺少中文名`).toBeTruthy();
      expect(label.trim().length, `动作 ${type} 的中文名为空`).toBeGreaterThan(0);
      expect(
        LATIN_LETTER.test(label),
        `动作 ${type} 的中文名含 ASCII 字母: ${label}`
      ).toBe(false);
    }
  });

  it("每个动作都有面向员工的动词（同样不含 ASCII 字母）", () => {
    for (const type of RPA_ACTION_TYPES) {
      const verb = HUMAN_VERBS[type];
      expect(verb, `动作 ${type} 缺少员工动词`).toBeTruthy();
      expect(verb.trim().length, `动作 ${type} 的员工动词为空`).toBeGreaterThan(0);
      expect(
        LATIN_LETTER.test(verb),
        `动作 ${type} 的员工动词含 ASCII 字母: ${verb}`
      ).toBe(false);
    }
  });

  it("下拉选项的 value 都是合法 ActionType（类型与值同源）", () => {
    for (const opt of ACTION_TYPE_OPTIONS) {
      const asType: ActionType = opt.value;
      expect(RPA_ACTION_TYPES as readonly string[]).toContain(asType);
      expect(opt.label).toBe(ACTION_TYPE_TEXT_MAP[asType]);
    }
  });
});
