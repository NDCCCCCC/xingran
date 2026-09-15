import { describe, it, expect } from "vitest";
import { computeDiff } from "./utils";

describe("computeDiff", () => {
  it("preserves order after push+reverse refactor (regression)", () => {
    // Regression: push+reverse must produce same line order as original unshift
    const content1 = "a\nb\nc\nd\ne";
    const content2 = "a\nx\nc\ny\ne";

    const result = computeDiff(content1, content2);

    // Lines should be in original document order, not reverse
    const sameLines = result.leftLines.filter((l) => l.type === "same");
    expect(sameLines[0]?.content).toBe("a");
    expect(sameLines[1]?.content).toBe("c");
    expect(sameLines[2]?.content).toBe("e");

    const removedLines = result.leftLines.filter((l) => l.type === "removed");
    const addedLines = result.rightLines.filter((l) => l.type === "added");

    expect(removedLines.map((l) => l.content)).toEqual(["b", "d"]);
    expect(addedLines.map((l) => l.content)).toEqual(["x", "y"]);

    // Both sides must have same total length
    expect(result.leftLines).toHaveLength(result.rightLines.length);
  });

  it("handles empty inputs", () => {
    // "" split → [""], so both sides get one empty-same line
    const r1 = computeDiff("", "");
    expect(r1.leftLines).toHaveLength(1);
    expect(r1.leftLines[0].type).toBe("same");
    expect(r1.leftLines[0].content).toBe("");

    // content2="" split→[""], dp[2][1]=0 → added branch first, then 2 removed
    const r2 = computeDiff("a\nb", "");
    expect(r2.leftLines.filter((l) => l.type === "removed")).toHaveLength(2);
    expect(r2.rightLines).toHaveLength(3); // 1 empty placeholder + 2 added
  });

  it("handles identical content", () => {
    const content = "line1\nline2\nline3";
    const result = computeDiff(content, content);
    expect(result.leftLines.every((l) => l.type === "same")).toBe(true);
    expect(result.leftLines).toHaveLength(3);
  });
});
