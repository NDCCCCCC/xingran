/**
 * src/constants/anchor.ts 单元测试
 *
 * 锁定机构锚点共享常量的全部行为分支：
 * - normalizeAnchorMode / normalizeOrgLevel：合法值、字符串数字、非法/边界值一律落到 0
 * - anchorModeLabel / orgLevelLabel：null/undefined、已知值、未知脏数据
 * - anchorModeNeedsParam：仅 mode=2（部门 UUID）与 mode=5（层级）需要参数
 * - isAnchorLevelParam / isAnchorDeptParam：字符串类型守卫 + 合法形态
 * - isValidAnchorConfig：mode 与 param 的形状匹配（0/3 不校验参数）
 * - retargetAnchorParam / normalizeAnchorParam：切换模式时的参数归位与后端值归一化
 *
 * 契约来源：.planning/notes/261118-anchor-scope-contract.md §二 2.2 / §五 / §六。
 * 重要不变量：锚点模式 1/4 已弃用，非法值一律归一化到 0（不参与机构授权）。
 */
import { describe, it, expect } from "vitest";
import {
  ANCHOR_MODE_OPTIONS,
  ANCHOR_MODES,
  ANCHOR_MODE_LABELS,
  ORG_LEVEL_OPTIONS,
  ORG_LEVEL_LABELS,
  ANCHOR_LEVEL_OPTIONS,
  ANCHOR_LEVEL_PARAMS,
  normalizeAnchorMode,
  normalizeOrgLevel,
  anchorModeLabel,
  orgLevelLabel,
  anchorModeNeedsParam,
  isAnchorLevelParam,
  isAnchorDeptParam,
  isValidAnchorConfig,
  retargetAnchorParam,
  normalizeAnchorParam,
} from "./anchor";

const UUID = "550e8400-e29b-41d4-a716-446655440000";
const UUID_UPPER = "550E8400-E29B-41D4-A716-446655440000";

describe("constants/anchor — 常量表完整性", () => {
  it("ANCHOR_MODE_OPTIONS/MODES 只含 0/2/3/5，禁止弃用的 1/4", () => {
    expect(ANCHOR_MODE_OPTIONS.map((o) => o.value)).toEqual([0, 2, 3, 5]);
    expect(ANCHOR_MODES).toEqual([0, 2, 3, 5]);
    expect(ANCHOR_MODES).not.toContain(1);
    expect(ANCHOR_MODES).not.toContain(4);
    expect(Object.keys(ANCHOR_MODE_LABELS).map(Number).sort()).toEqual([0, 2, 3, 5]);
  });

  it("模式选项文案与标签表一一对应", () => {
    for (const opt of ANCHOR_MODE_OPTIONS) {
      expect(ANCHOR_MODE_LABELS[opt.value]).toBe(opt.label);
    }
  });

  it("ORG_LEVEL_OPTIONS 为 0~4，层级标签表对齐", () => {
    expect(ORG_LEVEL_OPTIONS.map((o) => o.value)).toEqual([0, 1, 2, 3, 4]);
    expect(Object.keys(ORG_LEVEL_LABELS).map(Number).sort()).toEqual([0, 1, 2, 3, 4]);
    for (const opt of ORG_LEVEL_OPTIONS) {
      expect(ORG_LEVEL_LABELS[opt.value]).toBe(opt.label);
    }
  });

  it("ANCHOR_LEVEL_OPTIONS/PARAMS 只含字符串 '1'~'4'，排除 0=非机构", () => {
    expect(ANCHOR_LEVEL_OPTIONS.map((o) => o.value)).toEqual(["1", "2", "3", "4"]);
    expect([...ANCHOR_LEVEL_PARAMS]).toEqual(["1", "2", "3", "4"]);
    expect(ANCHOR_LEVEL_PARAMS).not.toContain("0");
  });
});

describe("constants/anchor — normalizeAnchorMode", () => {
  it("合法模式 0/2/3/5 原样返回（number 与字符串数字）", () => {
    for (const mode of [0, 2, 3, 5]) {
      expect(normalizeAnchorMode(mode)).toBe(mode);
      expect(normalizeAnchorMode(String(mode))).toBe(mode);
    }
  });

  it("弃用模式 1/4 与任意非法值一律落到 0", () => {
    for (const bad of [1, 4, -1, 6, 2.5, NaN, Infinity, -Infinity]) {
      expect(normalizeAnchorMode(bad)).toBe(0);
    }
  });

  it("非数字输入（undefined/null/空串/文本/对象/布尔）落到 0", () => {
    expect(normalizeAnchorMode(undefined)).toBe(0);
    expect(normalizeAnchorMode(null)).toBe(0);
    expect(normalizeAnchorMode("")).toBe(0);
    expect(normalizeAnchorMode(" 2 ")).toBe(2); // Number(" 2 ") === 2
    expect(normalizeAnchorMode("abc")).toBe(0);
    expect(normalizeAnchorMode({})).toBe(0);
    expect(normalizeAnchorMode([])).toBe(0);
    expect(normalizeAnchorMode(true)).toBe(0);
    expect(normalizeAnchorMode(false)).toBe(0);
  });
});

describe("constants/anchor — normalizeOrgLevel", () => {
  it("合法层级 0~4 原样返回（number 与字符串数字）", () => {
    for (const level of [0, 1, 2, 3, 4]) {
      expect(normalizeOrgLevel(level)).toBe(level);
      expect(normalizeOrgLevel(String(level))).toBe(level);
    }
  });

  it("非法层级（-1/5/小数/NaN/文本/对象/null）落到 0=非机构", () => {
    for (const bad of [-1, 5, 99, 1.5, NaN, Infinity, "abc", {}, null, undefined]) {
      expect(normalizeOrgLevel(bad)).toBe(0);
    }
    // 注意：Number(true) === 1，布尔 true 会被归一化为合法层级 1（与实现口径一致）
    expect(normalizeOrgLevel(true)).toBe(1);
    expect(normalizeOrgLevel(false)).toBe(0);
  });
});

describe("constants/anchor — 展示文案", () => {
  it("anchorModeLabel: undefined/null → 0 的文案；已知值 → 对应文案", () => {
    expect(anchorModeLabel()).toBe("不参与机构授权");
    expect(anchorModeLabel(null)).toBe("不参与机构授权");
    expect(anchorModeLabel(0)).toBe("不参与机构授权");
    expect(anchorModeLabel(2)).toBe("指定部门（含下级）");
    expect(anchorModeLabel(3)).toBe("外部机构（最近）");
    expect(anchorModeLabel(5)).toBe("指定机构层级（最近）");
  });

  it("anchorModeLabel: 未知/弃用模式 → 未知模式(n) 脏数据提示", () => {
    expect(anchorModeLabel(1)).toBe("未知模式(1)");
    expect(anchorModeLabel(4)).toBe("未知模式(4)");
    expect(anchorModeLabel(99)).toBe("未知模式(99)");
  });

  it("orgLevelLabel: undefined/null → 0 的文案；已知值 → 对应文案", () => {
    expect(orgLevelLabel()).toBe("非机构");
    expect(orgLevelLabel(null)).toBe("非机构");
    expect(orgLevelLabel(0)).toBe("非机构");
    expect(orgLevelLabel(1)).toBe("分公司");
    expect(orgLevelLabel(2)).toBe("中支");
    expect(orgLevelLabel(3)).toBe("支公司");
    expect(orgLevelLabel(4)).toBe("网点");
  });

  it("orgLevelLabel: 未知层级 → 未知层级(n) 脏数据提示", () => {
    expect(orgLevelLabel(5)).toBe("未知层级(5)");
    expect(orgLevelLabel(-1)).toBe("未知层级(-1)");
  });
});

describe("constants/anchor — anchorModeNeedsParam", () => {
  it("仅 mode=2（UUID）与 mode=5（层级）需要参数", () => {
    expect(anchorModeNeedsParam(2)).toBe(true);
    expect(anchorModeNeedsParam(5)).toBe(true);
  });

  it("0/3/弃用 1/4/undefined/null 都不需要参数", () => {
    for (const mode of [0, 3, 1, 4, undefined, null, 99]) {
      expect(anchorModeNeedsParam(mode)).toBe(false);
    }
  });
});

describe("constants/anchor — 参数形状守卫", () => {
  it("isAnchorLevelParam: 仅字符串 '1'~'4' 为 true", () => {
    for (const p of ["1", "2", "3", "4"]) expect(isAnchorLevelParam(p)).toBe(true);
    for (const bad of ["0", "5", "", "01", " 1", 1, 2, null, undefined, true]) {
      expect(isAnchorLevelParam(bad as string)).toBe(false);
    }
  });

  it("isAnchorDeptParam: 仅合法 UUID 形态（大小写不敏感）为 true", () => {
    expect(isAnchorDeptParam(UUID)).toBe(true);
    expect(isAnchorDeptParam(UUID_UPPER)).toBe(true);
    for (const bad of [
      "",
      "not-a-uuid",
      "550e8400e29b41d4a716446655440000", // 缺连字符
      "550e8400-e29b-41d4-a716-44665544000", // 末段少一位
      "550e8400-e29b-41d4-a716-4466554400000", // 末段多一位
      "550e8400-e29b-41d4-a716-44665544000g", // 非法字符
      null,
      undefined,
      123,
      {},
    ]) {
      expect(isAnchorDeptParam(bad as string)).toBe(false);
    }
  });
});

describe("constants/anchor — isValidAnchorConfig", () => {
  it("mode=0/3 恒合法（不携带参数，非法 mode 归一到 0 亦合法）", () => {
    expect(isValidAnchorConfig(0)).toBe(true);
    expect(isValidAnchorConfig(0, "任意脏值")).toBe(true);
    expect(isValidAnchorConfig(3)).toBe(true);
    expect(isValidAnchorConfig(3, UUID)).toBe(true);
    expect(isValidAnchorConfig(1, "x")).toBe(true); // 归一到 0
    expect(isValidAnchorConfig(4, "x")).toBe(true); // 归一到 0
  });

  it("mode=2 必须是 UUID 形态参数", () => {
    expect(isValidAnchorConfig(2, UUID)).toBe(true);
    expect(isValidAnchorConfig(2, UUID_UPPER)).toBe(true);
    expect(isValidAnchorConfig(2, "1")).toBe(false);
    expect(isValidAnchorConfig(2, "not-a-uuid")).toBe(false);
    expect(isValidAnchorConfig(2)).toBe(false);
    expect(isValidAnchorConfig(2, null)).toBe(false);
    expect(isValidAnchorConfig(2, 123 as unknown as string)).toBe(false);
  });

  it("mode=5 必须是 '1'~'4' 层级参数", () => {
    for (const p of ["1", "2", "3", "4"]) expect(isValidAnchorConfig(5, p)).toBe(true);
    for (const bad of ["0", "5", "", UUID, null, undefined, 1]) {
      expect(isValidAnchorConfig(5, bad as string)).toBe(false);
    }
  });

  it("字符串数字 mode 走归一化后按同样规则校验", () => {
    expect(isValidAnchorConfig("2", UUID)).toBe(true);
    expect(isValidAnchorConfig("5", "3")).toBe(true);
    expect(isValidAnchorConfig("2", "3")).toBe(false);
  });
});

describe("constants/anchor — retargetAnchorParam", () => {
  it("mode=2：保留合法 UUID，否则清空", () => {
    expect(retargetAnchorParam(2, UUID)).toBe(UUID);
    expect(retargetAnchorParam(2, UUID_UPPER)).toBe(UUID_UPPER);
    expect(retargetAnchorParam(2, "3")).toBe("");
    expect(retargetAnchorParam(2, "not-a-uuid")).toBe("");
    expect(retargetAnchorParam(2, null)).toBe("");
    expect(retargetAnchorParam(2, undefined)).toBe("");
  });

  it("mode=5：保留合法层级 '1'~'4'，否则清空", () => {
    for (const p of ["1", "2", "3", "4"]) expect(retargetAnchorParam(5, p)).toBe(p);
    expect(retargetAnchorParam(5, UUID)).toBe("");
    expect(retargetAnchorParam(5, "0")).toBe("");
    expect(retargetAnchorParam(5, null)).toBe("");
  });

  it("mode=0/3：无论旧参数是什么恒清空（防静默脏值）", () => {
    for (const mode of [0, 3] as const) {
      expect(retargetAnchorParam(mode, UUID)).toBe("");
      expect(retargetAnchorParam(mode, "3")).toBe("");
      expect(retargetAnchorParam(mode, "dirty")).toBe("");
      expect(retargetAnchorParam(mode)).toBe("");
    }
  });
});

describe("constants/anchor — normalizeAnchorParam", () => {
  it("非字符串输入（undefined/null/对象/布尔）→ 空串", () => {
    expect(normalizeAnchorParam(2, undefined)).toBe("");
    expect(normalizeAnchorParam(2, null)).toBe("");
    expect(normalizeAnchorParam(2, {})).toBe("");
    expect(normalizeAnchorParam(5, true)).toBe("");
  });

  it("数字输入先转字符串再按模式校验", () => {
    // mode=5 时数字 3 → "3" 合法
    expect(normalizeAnchorParam(5, 3)).toBe("3");
    // mode=2 时数字无法构成 UUID → 清空
    expect(normalizeAnchorParam(2, 3)).toBe("");
    // mode=0 时恒清空
    expect(normalizeAnchorParam(0, 3)).toBe("");
  });

  it("mode=2 只接受 UUID 字符串，mode=5 只接受 '1'~'4'，其余恒为空串", () => {
    expect(normalizeAnchorParam(2, UUID)).toBe(UUID);
    expect(normalizeAnchorParam(2, "bad")).toBe("");
    expect(normalizeAnchorParam(5, "4")).toBe("4");
    expect(normalizeAnchorParam(5, "9")).toBe("");
    expect(normalizeAnchorParam(3, UUID)).toBe("");
    expect(normalizeAnchorParam(1, "1")).toBe(""); // 弃用模式归一到 0 → 空串
  });

  it("非法 mode（字符串数字/脏值）先归一化再决定参数归位", () => {
    expect(normalizeAnchorParam("2", UUID)).toBe(UUID);
    expect(normalizeAnchorParam("5", "2")).toBe("2");
    expect(normalizeAnchorParam("abc", "2")).toBe("");
  });
});
