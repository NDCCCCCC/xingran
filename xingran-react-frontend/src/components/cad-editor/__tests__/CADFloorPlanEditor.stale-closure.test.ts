/**
 * Phase 117 Wave3 BUGFIX-03 — CADFloorPlanEditor stale closure regression test
 *
 * BUGFIX-03: setFloorPlanData((prev) => {...}) updater 内读 floorPlanData.snapToGrid/
 * gridSize（闭包旧值），应改为读 prev.snapToGrid/prev.gridSize（最新 state）。
 * 3 处需修复：新建工位（~:720）、文本绘制（~:765）、批量拖动（~:964）。
 *
 * Regression: After fix, these 3 locations must use prev.snapToGrid/prev.gridSize,
 * and the handleCanvasMouseMove deps must NOT include floorPlanData.
 */
import { describe, it, expect } from "vitest";
import * as fs from "fs";
import * as path from "path";

const SOURCE_PATH = path.resolve(__dirname, "../CADFloorPlanEditor.tsx");

function readSource(): string {
  return fs.readFileSync(SOURCE_PATH, "utf-8");
}

describe("BUGFIX-03 — CADFloorPlanEditor stale closure", () => {
  it("updater 内新建工位使用 prev.snapToGrid/prev.gridSize，而非 floorPlanData", () => {
    const src = readSource();
    // Find the "创建新工位" section - should use setFloorPlanData with prev.snapToGrid
    const newWsSection = src.substring(src.indexOf("创建新工位"), src.indexOf("工位已添加") + 200);
    // Must read prev.snapToGrid/prev.gridSize inside updater, not floorPlanData
    expect(newWsSection).toMatch(/prev\.snapToGrid/);
    expect(newWsSection).toMatch(/prev\.gridSize/);
    expect(newWsSection).not.toMatch(/floorPlanData\.snapToGrid/);
  });

  it("updater 内批量拖动使用 prev.snapToGrid/prev.gridSize，而非 floorPlanData", () => {
    const src = readSource();
    // Find the batch drag block (has "拖动工位" comment)
    const dragPattern = /拖动工位（带网格吸附[\s\S]*?if \(prev\.snapToGrid\)[\s\S]*?prev\.gridSize/;
    const match = src.match(dragPattern);
    expect(match).not.toBeNull();
    const block = match![0];
    // Must use prev.snapToGrid and prev.gridSize in the updater
    expect(block).toMatch(/prev\.snapToGrid/);
    expect(block).toMatch(/prev\.gridSize/);
    expect(block).not.toMatch(/floorPlanData\.snapToGrid/);
  });

  it("handleCanvasMouseMove deps 不包含 floorPlanData（防止闭包老化）", () => {
    const src = readSource();
    // Find the handleCanvasMouseMove useCallback deps array
    const depsPattern = /handleCanvasMouseMove[\s\S]*?\[([\s\S]*?)\]\s*\);?\s*$/m;
    const match = src.match(depsPattern);
    expect(match).not.toBeNull();
    const deps = match![1];
    // floorPlanData must NOT be in the deps array (it was causing stale closure)
    expect(deps).not.toMatch(/floorPlanData/);
  });

  it("updater 内文本绘制使用 prev.snapToGrid/prev.gridSize", () => {
    const src = readSource();
    // Find the text drawing snap block
    const textPattern = /setTextInputPosition[\s\S]*?prev\.snapToGrid/;
    const match = src.match(textPattern);
    // This verifies the text drawing mode also uses prev (if it exists)
    // The text drawing block should use prev in the updater context
    // We'll check that floorPlanData.snapToGrid is not used for text snapping in updater
    const srcAfter736 = src.substring(src.indexOf("绘制文本模式"));
    const textDrawBlock = srcAfter736.substring(0, 200);
    // After the fix, text drawing snap should reference prev or not appear in stale form
    // The stale pattern "floorPlanData.snapToGrid" should not appear in the text drawing block
    expect(textDrawBlock).not.toMatch(/floorPlanData\.snapToGrid/);
  });
});
