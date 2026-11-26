/**
 * W7 / UI-FIX-02：接管者判定纯函数测试
 *
 * 这些用例直接编码「生产复现路径」：
 *   · nickname 与 username 不一致时，旧逻辑（只比展示名）会让遮罩不消失；
 *   · 新逻辑以 userID 为权威，展示名漂移不再影响判定。
 */
import { describe, it, expect } from "vitest";
import { resolveIsController, controllerDisplayName, isTakenOver } from "../controller";

const ME_ID = "7fd49999-af45-42b5-a936-fc4fe64cb2ad";

describe("resolveIsController：userID 为权威判据", () => {
  it("接管者 userID == 我的 userID → true（即使展示名完全不同）", () => {
    expect(
      resolveIsController({
        myUserId: ME_ID,
        myName: "超级管理员",
        takeoverById: ME_ID,
        takeoverBy: "ninedrunk",
      })
    ).toBe(true);
  });

  it("污点复现：后端 backup 只有展示名 ninedrunk、我是 ninedrunk → 兜底为 true", () => {
    expect(
      resolveIsController({
        myUserId: ME_ID,
        myName: "ninedrunk",
        takeoverBy: "ninedrunk",
        takeoverById: "",
      })
    ).toBe(true);
  });

  it("缺陷复现：展示名不同但 userID 相同 → true（修前这里是 false，遮罩不消失）", () => {
    // 修前逻辑：takeoverBy("ninedrunk") !== myName("超级管理员") → false
    // 用变量而非字面量表达，避免触发 no-constant-binary-expression（字面量比较会被判为常量）
    const myName = "超级管理员";
    const takeoverBy = "ninedrunk";
    const oldLogic = myName !== "" && takeoverBy !== "" && takeoverBy === myName;
    expect(oldLogic).toBe(false);
    // 修后逻辑：userID 相等 → true
    expect(
      resolveIsController({
        myUserId: ME_ID,
        myName: "超级管理员",
        takeoverById: ME_ID,
        takeoverBy: "ninedrunk",
      })
    ).toBe(true);
  });

  it("他人接管（userID 不同）→ false", () => {
    expect(
      resolveIsController({
        myUserId: ME_ID,
        myName: "超级管理员",
        takeoverById: "other-user-id",
        takeoverBy: "超级管理员",
      })
    ).toBe(false);
  });

  it("无人接管 → false", () => {
    expect(resolveIsController({ myUserId: ME_ID, myName: "超级管理员" })).toBe(false);
  });

  it("我有 userID 但后端没下发 takeoverById、展示名也不匹配 → false", () => {
    expect(
      resolveIsController({
        myUserId: ME_ID,
        myName: "甲",
        takeoverBy: "乙",
      })
    ).toBe(false);
  });

  it("userID 存在时不回落到展示名（防止双口径再次漂移）", () => {
    // takeoverById 有值但不是我 → 即使展示名碰巧等于我的名字也必须是 false。
    expect(
      resolveIsController({
        myUserId: ME_ID,
        myName: "ninedrunk",
        takeoverById: "other-user-id",
        takeoverBy: "ninedrunk",
      })
    ).toBe(false);
  });

  it("前后空格不参与比较（trim 归一）", () => {
    expect(
      resolveIsController({
        myUserId: ME_ID,
        myName: " 张三 ",
        takeoverById: "", // 无 ID 走兜底
        takeoverBy: "张三",
      })
    ).toBe(true);
  });

  it("我的 userID 为空时，即使后端 takeoverById 也为空也不能误判为接管者", () => {
    expect(resolveIsController({ myUserId: "", myName: "", takeoverBy: "" })).toBe(false);
  });
});

describe("HUD 展示名与接管态", () => {
  it("优先用展示名", () => {
    expect(controllerDisplayName("ninedrunk", ME_ID)).toBe("ninedrunk");
  });

  it("只有 userID 时给可读占位，不产出「已接管 · 」半截文案", () => {
    expect(controllerDisplayName("", ME_ID)).toBe("其他用户");
  });

  it("都没有时为空串（不渲染已接管角标）", () => {
    expect(controllerDisplayName("", "")).toBe("");
  });

  it("isTakenOver：任意一份非空即为真", () => {
    expect(isTakenOver("ninedrunk", "")).toBe(true);
    expect(isTakenOver("", ME_ID)).toBe(true);
    expect(isTakenOver("", "")).toBe(false);
  });
});
