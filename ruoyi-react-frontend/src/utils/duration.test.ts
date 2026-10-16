import { describe, expect, it } from "vitest";
import { formatDurationSeconds } from "./duration";

describe("formatDurationSeconds（秒级人类可读时长）", () => {
  it.each([
    [0, "0s"],
    [-5, "0s"],
    [Number.NaN, "0s"],
    [null, "0s"],
    [undefined, "0s"],
  ])("零值/负值/NaN/空输入 %p → 0s", (input, expected) => {
    expect(formatDurationSeconds(input as number | null | undefined)).toBe(expected);
  });

  it.each([
    [1, "1s"],
    [45, "45s"],
    [45.9, "45s"], // 小数截断
    [59, "59s"],
    [60, "1m"],
    [900, "15m"],
    [3599, "59m"],
    [3600, "1h"],
    [151200, "1d 18h"], // 文档注释中的样例
    [86400, "1d 0h"],
    [90000, "1d 1h"],
  ])("%p 秒 → %p", (seconds, expected) => {
    expect(formatDurationSeconds(seconds)).toBe(expected);
  });
});
