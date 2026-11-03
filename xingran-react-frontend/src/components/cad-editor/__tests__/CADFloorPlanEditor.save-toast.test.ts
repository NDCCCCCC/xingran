/**
 * 回归测试：平面图编辑「保存」只弹一次「保存成功」toast。
 *
 * 缺陷现象：在「楼层管理 → 平面图编辑」页面点一次「保存」按钮，弹出两个「保存成功」。
 *
 * 根因：保存链路上存在两处成功提示——
 *   1) CADFloorPlanEditor.handleSave：调用 onSave 后没有 await，随即 message.success("保存成功")；
 *      保存请求还没返回就提示，保存失败时同样会误报成功；
 *   2) useFloorPlanEditor.saveFloorPlan：await 真实保存成功后才提示（有配对的「保存失败」提示）。
 *
 * 修复：删除编辑器组件内的成功提示，提示统一由 useFloorPlanEditor 在真实保存完成后发出。
 *
 * 说明：本文件沿用仓库既有模式（fs.readFileSync 读源码做契约断言），
 * 因为 cad-editor 属重画布低确定性 UI，不适合做组件级渲染断言。
 */
import { describe, it, expect } from "vitest";
import * as fs from "fs";
import * as path from "path";

const EDITOR_SOURCE_PATH = path.resolve(__dirname, "../CADFloorPlanEditor.tsx");
const FLOORS_DIR = path.resolve(__dirname, "../../../pages/operations/floors");

/**
 * 拼接构造成功提示字面量，避免本测试文件自身被「保存链路成功提示」统计误计入。
 */
const SUCCESS_TOAST = ["message.success(", "\"保存成功\"", ")"].join("");

function readSource(filePath: string): string {
  return fs.readFileSync(filePath, "utf-8");
}

/** 提取 handleSave 所在的「保存」代码段 */
function extractSaveSection(source: string): string {
  const start = source.indexOf("保存 ====================");
  const end = source.indexOf("更新元素属性 ====================");
  expect(start).toBeGreaterThan(-1);
  expect(end).toBeGreaterThan(start);
  return source.slice(start, end);
}

/** 递归收集目录下的 ts/tsx 源码文件（跳过测试目录） */
function collectSourceFiles(dir: string): string[] {
  const files: string[] = [];
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const fullPath = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      if (entry.name === "__tests__") continue;
      files.push(...collectSourceFiles(fullPath));
    } else if (/\.tsx?$/.test(entry.name)) {
      files.push(fullPath);
    }
  }
  return files;
}

describe("平面图保存只弹一次「保存成功」toast", () => {
  it("CADFloorPlanEditor 的保存逻辑不再发出「保存成功」提示", () => {
    const saveSection = extractSaveSection(readSource(EDITOR_SOURCE_PATH));
    expect(saveSection).not.toContain(SUCCESS_TOAST);
  });

  it("整个 CADFloorPlanEditor 文件不再出现「保存成功」提示", () => {
    expect(readSource(EDITOR_SOURCE_PATH)).not.toContain(SUCCESS_TOAST);
  });

  it("CADFloorPlanEditor 的保存逻辑显式声明「不等待」语义（void onSave?.）", () => {
    const saveSection = extractSaveSection(readSource(EDITOR_SOURCE_PATH));
    expect(saveSection).toMatch(/void\s+onSave\?\.\(/);
  });

  it("onSave props 类型放宽为 void | Promise<void>", () => {
    const source = readSource(EDITOR_SOURCE_PATH);
    expect(source).toMatch(/onSave\?:\s*\(data:\s*FloorPlanData\)\s*=>\s*void\s*\|\s*Promise<void>/);
  });

  it("保存链路源码中「保存成功」提示恰好只有一处（位于 useFloorPlanEditor）", () => {
    const candidates = [EDITOR_SOURCE_PATH, ...collectSourceFiles(FLOORS_DIR)];
    const hits = candidates
      .filter((file) => readSource(file).includes(SUCCESS_TOAST))
      .map((file) => path.relative(FLOORS_DIR, file));
    expect(hits).toEqual(["useFloorPlanEditor.ts"]);
  });
});
