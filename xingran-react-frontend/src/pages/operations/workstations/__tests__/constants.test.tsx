/**
 * Phase 88 Batch94 — operations/workstations/constants 单元测试
 */
import { describe, it, expect } from "vitest";
import {
  getWorkstationTypeText,
  getWorkstationStatusText,
  getWorkstationStatusColor,
  renderWorkstationTypeTag,
  renderWorkstationStatusTag,
  resolveWorkstationTypeText,
  renderWorkstationTypeTagWithDict,
  toWorkstationNode,
  STATUS_OPTIONS,
  TYPE_OPTIONS,
} from "../constants";

describe("workstations constants", () => {
  it("STATUS_OPTIONS 三态", () => {
    expect(STATUS_OPTIONS).toHaveLength(3);
  });

  it("TYPE_OPTIONS 三类型", () => {
    expect(TYPE_OPTIONS).toHaveLength(3);
    expect(TYPE_OPTIONS[0].value).toBe(0);
  });

  it("getWorkstationTypeText 已知/未知", () => {
    expect(getWorkstationTypeText(0)).toBe("固定工位");
    expect(getWorkstationTypeText(1)).toBe("灵活工位");
    expect(getWorkstationTypeText(99)).toBe("-");
  });

  // 字典感知解析：管理员在字典管理页新增的工位类型（如「虚拟工位」）
  // 必须在列表/卡片/平面图正确显示，不能退化为 "-"。
  it("resolveWorkstationTypeText 命中新增字典项", () => {
    const dict = [
      { dictValue: "0", dictLabel: "固定工位" },
      { dictValue: "1", dictLabel: "灵活工位" },
      { dictValue: "2", dictLabel: "管理工位" },
      { dictValue: "3", dictLabel: "虚拟工位" },
    ];
    expect(resolveWorkstationTypeText(3, dict)).toBe("虚拟工位");
    expect(resolveWorkstationTypeText(0, dict)).toBe("固定工位");
  });

  it("resolveWorkstationTypeText 字典未命中降级静态映射", () => {
    const dict = [{ dictValue: "3", dictLabel: "虚拟工位" }];
    // 字典里没有 0 → 降级静态映射（固定工位）
    expect(resolveWorkstationTypeText(0, dict)).toBe("固定工位");
    // 字典与静态映射都没有 → "-"
    expect(resolveWorkstationTypeText(99, dict)).toBe("-");
  });

  it("resolveWorkstationTypeText 无字典时等价静态映射", () => {
    expect(resolveWorkstationTypeText(1)).toBe("灵活工位");
    expect(resolveWorkstationTypeText(99)).toBe("-");
  });

  it("renderWorkstationTypeTagWithDict 命中新增字典项", () => {
    const dict = [{ dictValue: "3", dictLabel: "虚拟工位" }];
    const node = renderWorkstationTypeTagWithDict(3, dict);
    expect(node).toBeDefined();
    expect(node.props.children).toBe("虚拟工位");
  });

  it("getWorkstationStatusText 已知/未知", () => {
    expect(getWorkstationStatusText(0)).toBeDefined();
    expect(getWorkstationStatusText(99)).toBe("-");
  });

  it("getWorkstationStatusColor 已知/未知", () => {
    expect(getWorkstationStatusColor(0)).toBeDefined();
    expect(getWorkstationStatusColor(99)).toBe("default");
  });

  it("renderWorkstationTypeTag 返回 Tag", () => {
    const node = renderWorkstationTypeTag(0);
    expect(node).toBeDefined();
    expect(node.type).toBeDefined();
  });

  it("renderWorkstationStatusTag 返回带 color 的 Tag", () => {
    const node = renderWorkstationStatusTag(0);
    expect(node).toBeDefined();
  });

  it("toWorkstationNode 转换", () => {
    const node = toWorkstationNode({
      id: "w1",
      name: "WS001",
      positionX: 100,
      positionY: 200,
      width: 180,
      depth: 80,
      status: 0,
      type: 1,
      rotation: 90,
    } as unknown);
    expect(node.id).toBe("w1");
    expect(node.x).toBe(100);
    expect(node.y).toBe(200);
    expect(node.rotation).toBe(90);
  });

  it("toWorkstationNode 缺字段回退", () => {
    const node = toWorkstationNode({ id: "w2", name: "WS002" } as unknown);
    expect(node.x).toBe(0);
    expect(node.y).toBe(0);
    expect(node.rotation).toBe(0);
    expect(node.width).toBe(160);
  });
});
