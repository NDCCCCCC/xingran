import { describe, expect, it, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import {
  createActionColumn,
  createDateTimeColumn,
  createIndexColumn,
  createSorter,
  createSorterMeta,
  createStatusColumn,
  createTagColumn,
  formatFileSize,
  type SorterMeta,
} from "./tableHelpers";

type Row = Record<string, unknown>;

describe("createSorterMeta", () => {
  it("返回服务端排序元数据", () => {
    expect(createSorterMeta("username")).toEqual({ field: "username" });
    expect(createSorterMeta<Row>("createdAt", "date")).toEqual({
      field: "createdAt",
      type: "date",
    } satisfies SorterMeta<Row>);
  });
});

describe("createSorter（客户端排序工厂）", () => {
  it("string：本地化比较（中文数字感知）", () => {
    const sorter = createSorter<Row>("name", "string");
    expect(sorter({ name: "a" }, { name: "b" })).toBeLessThan(0);
    expect(sorter({ name: "b" }, { name: "a" })).toBeGreaterThan(0);
    expect(sorter({ name: "same" }, { name: "same" })).toBe(0);
  });

  it("null/undefined 统一排末端", () => {
    const sorter = createSorter<Row>("name", "string");
    expect(sorter({ name: null }, { name: "x" })).toBe(1);
    expect(sorter({ name: "x" }, { name: null })).toBe(-1);
    expect(sorter({ name: null }, { name: undefined })).toBe(0);
  });

  it("number：数值比较，NaN 排末端", () => {
    const sorter = createSorter<Row>("n", "number");
    expect(sorter({ n: 1 }, { n: 10 })).toBeLessThan(0);
    expect(sorter({ n: Number.NaN }, { n: 1 })).toBe(1);
    expect(sorter({ n: Number.NaN }, { n: Number.NaN })).toBe(0);
  });

  it("date：按时间戳比较，非法日期排末端", () => {
    const sorter = createSorter<Row>("d", "date");
    expect(sorter({ d: "2026-01-01" }, { d: "2026-01-02" })).toBeLessThan(0);
    expect(sorter({ d: "garbage" }, { d: "2026-01-01" })).toBe(1);
    expect(sorter({ d: "garbage" }, { d: "trash" })).toBe(0);
  });

  it("boolean：true 排在前", () => {
    const sorter = createSorter<Row>("ok", "boolean");
    expect(sorter({ ok: true }, { ok: false })).toBe(-1);
    expect(sorter({ ok: false }, { ok: true })).toBe(1);
    expect(sorter({ ok: true }, { ok: true })).toBe(0);
  });

  it("custom：委托 compareFn，缺省时返回 0", () => {
    const sorter = createSorter<Row>("x", "custom", (a, b) => Number(a.n) - Number(b.n));
    expect(sorter({ x: 1, n: 5 }, { x: 2, n: 9 })).toBe(-4);
    const fallback = createSorter<Row>("x", "custom");
    expect(fallback({ x: 1 }, { x: 2 })).toBe(0);
  });
});

describe("createStatusColumn", () => {
  it("status 0 渲染正常（success Tag），1 渲染停用（error Tag）", () => {
    const col = createStatusColumn<Row>();
    expect(col.title).toBe("状态");
    const { container, rerender } = render(<>{col.render!(0, {}, 0) as React.ReactElement}</>);
    expect(container.textContent).toBe("正常");
    rerender(<>{col.render!(1, {}, 0) as React.ReactElement}</>);
    expect(container.textContent).toBe("停用");
  });

  it("支持自定义字段与配置覆盖", () => {
    const col = createStatusColumn<Row>("enabled", { title: "启用状态", width: 120 });
    expect(col.dataIndex).toBe("enabled");
    expect(col.title).toBe("启用状态");
    expect(col.width).toBe(120);
  });
});

describe("createDateTimeColumn", () => {
  it("渲染 formatDateTime 结果", () => {
    const col = createDateTimeColumn<Row>("updatedAt", { title: "更新时间" });
    expect(col.title).toBe("更新时间");
    const { container } = render(
      <>{col.render!("2026-08-24T10:00:00", {}, 0) as React.ReactElement}</>
    );
    expect(container.textContent).toBe("2026-08-24 10:00:00");
  });
});

describe("createActionColumn", () => {
  it("编辑/删除按钮回调记录，extraActions 追加", () => {
    const onEdit = vi.fn();
    const onDelete = vi.fn();
    const col = createActionColumn<Row>(onEdit, onDelete, (record) => (
      <button type="button" onClick={() => String(record.id)}>
        extra
      </button>
    ));
    const record = { id: "r1" };
    render(<>{col.render!(null, record, 0) as React.ReactElement}</>);

    fireEvent.click(screen.getByText("编辑"));
    expect(onEdit).toHaveBeenCalledWith(record);
    fireEvent.click(screen.getByText("删除"));
    expect(onDelete).toHaveBeenCalledWith(record);
    expect(screen.getByText("extra")).toBeTruthy();
  });

  it("不传回调时不渲染编辑/删除按钮", () => {
    const col = createActionColumn<Row>();
    render(<>{col.render!(null, { id: "r" }, 0) as React.ReactElement}</>);
    expect(screen.queryByText("编辑")).toBeNull();
    expect(screen.queryByText("删除")).toBeNull();
  });
});

describe("createTagColumn", () => {
  it("按 colorMap/labelMap 渲染，未知值回落原文与 default 色", () => {
    const col = createTagColumn<Row>("type", { A: "blue" }, { A: "类型A" });
    const { container, rerender } = render(<>{col.render!("A", {}, 0) as React.ReactElement}</>);
    expect(container.textContent).toBe("类型A");
    rerender(<>{col.render!("Z", {}, 0) as React.ReactElement}</>);
    expect(container.textContent).toBe("Z");
    rerender(<>{col.render!("", {}, 0) as React.ReactElement}</>);
    expect(container.textContent).toBe("-");
  });
});

describe("createIndexColumn", () => {
  it("跨页序号 = (current-1)*pageSize + index + 1", () => {
    const col = createIndexColumn<Row>(3, 10);
    expect(col.render!(null, {}, 0)).toBe(21);
    expect(col.render!(null, {}, 4)).toBe(25);
    expect(createIndexColumn<Row>(1, 10).render!(null, {}, 0)).toBe(1);
  });
});

describe("formatFileSize", () => {
  it.each([
    [0, "0 B"],
    [512, "512 B"],
    [1024, "1 KB"],
    [1536, "1.5 KB"],
    [1024 * 1024, "1 MB"],
    [1024 * 1024 * 1024, "1 GB"],
    [1024 ** 4, "1 TB"],
  ])("%p 字节 → %p", (bytes, expected) => {
    expect(formatFileSize(bytes)).toBe(expected);
  });
});
