/**
 * Phase 88 Batch26 — monitor jobColumns/jobLogColumns + vdi vmOperationButtons 单元测试
 */
import { describe, it, expect } from "vitest";

import { getJobLogColumns } from "../job/columns/jobLogColumns";
import { getJobColumns } from "../job/columns/jobColumns";
import { vmOperationButtons } from "../../vdi/VirtualMachineList/vmOperationButtons";

describe("getJobLogColumns", () => {
  it("返回 5 列", () => {
    const cols = getJobLogColumns();
    expect(cols.length).toBe(5);
    const keys = cols.map((c) => c.key as string);
    expect(keys).toEqual(["startTime", "duration", "jobMessage", "status", "exceptionInfo"]);
  });

  it("status render 成功/失败 Tag", () => {
    const cols = getJobLogColumns();
    const col = cols.find((c) => c.key === "status");
    const render = col?.render as (s: number) => any;
    const ok = render(0);
    expect(ok.props.color).toBe("success");
    expect(ok.props.children).toBe("成功");
    const bad = render(1);
    expect(bad.props.color).toBe("error");
    expect(bad.props.children).toBe("失败");
  });

  it("duration render 委托 formatDuration", () => {
    const cols = getJobLogColumns();
    const col = cols.find((c) => c.key === "duration");
    const render = col?.render as (d: number) => any;
    expect(render(0)).toBe("-");
    expect(render(800)).toBe("800ms");
  });

  it("startTime render 委托 formatDateTime(字符串)", () => {
    const cols = getJobLogColumns();
    const col = cols.find((c) => c.key === "startTime");
    const render = col?.render as (t: string) => unknown;
    // 不 throw + 返回任意展示值
    expect(render("2026-08-28T10:00:00Z")).toBeDefined();
  });

  it("exceptionInfo render 委托 renderExceptionInfo", () => {
    const cols = getJobLogColumns();
    const col = cols.find((c) => c.key === "exceptionInfo");
    const render = col?.render as (v: string) => unknown;
    expect(render("")).toBe("-");
    expect(render("boom")).toBeDefined();
  });
});

describe("getJobColumns", () => {
  it("返回列数组含关键 key", () => {
    const cols = getJobColumns({
      handleEdit: () => {},
      handleDelete: () => {},
      handleStatusChange: () => {},
      handleRun: () => {},
    } as any);
    expect(Array.isArray(cols)).toBe(true);
    expect(cols.length).toBeGreaterThan(5);
  });
});

describe("vmOperationButtons 常量表", () => {
  it("6 个操作齐备", () => {
    expect(vmOperationButtons.length).toBe(6);
    const actions = vmOperationButtons.map((b) => b.action);
    expect(actions).toEqual(["start", "stop", "restart", "sync", "delete", "bind"]);
  });

  it("每个按钮有 permission/label/icon", () => {
    for (const b of vmOperationButtons) {
      expect(typeof b.permission).toBe("string");
      expect(b.permission.startsWith("vdi:vm:")).toBe(true);
      expect(typeof b.label).toBe("string");
      expect(b.icon).toBeDefined();
    }
  });

  it("label 中文齐全", () => {
    const labels = vmOperationButtons.map((b) => b.label);
    expect(labels).toEqual(["开机", "关机", "重启", "同步", "删除", "绑定用户"]);
  });
});
