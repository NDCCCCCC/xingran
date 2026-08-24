import { describe, expect, it } from "vitest";
import { isValidElement } from "react";
import { render, screen } from "@testing-library/react";
import {
  fullIconNameMap,
  getAllIcons,
  getIconComponent,
  iconCategories,
  iconDescriptionMap,
  searchIcons,
} from "./iconUtils";

describe("getIconComponent", () => {
  it("空输入返回 undefined", () => {
    expect(getIconComponent(null)).toBeUndefined();
    expect(getIconComponent(undefined)).toBeUndefined();
    expect(getIconComponent("")).toBeUndefined();
  });

  it("完整图标名直接命中并渲染为 React 元素", () => {
    const node = getIconComponent("UserOutlined");
    expect(node).toBeTruthy();
    expect(isValidElement(node)).toBe(true);
    render(node as React.ReactElement);
    expect(screen.getByRole("img", { hidden: true })).toBeTruthy();
  });

  it("未知图标名返回 undefined", () => {
    expect(getIconComponent("NoSuchIconExists")).toBeUndefined();
  });

  it("fullIconNameMap 映射路径可命中（CloudServerOutlined → server）", () => {
    const node = getIconComponent("CloudServerOutlined");
    expect(isValidElement(node)).toBe(true);
  });
});

describe("图标分类与描述", () => {
  it("iconCategories 覆盖 14 个分类", () => {
    expect(Object.keys(iconCategories)).toHaveLength(14);
    expect(iconCategories["基础"]).toContain("DashboardOutlined");
  });

  it("iconDescriptionMap 提供中文描述", () => {
    expect(iconDescriptionMap["UserOutlined"]).toBe("用户");
    expect(iconDescriptionMap["DashboardOutlined"]).toBe("仪表盘");
  });

  it("fullIconNameMap 提供向后兼容短名映射", () => {
    expect(fullIconNameMap["CloudServerOutlined"]).toBe("server");
    expect(fullIconNameMap["BookOutlined"]).toBe("file");
  });
});

describe("getAllIcons / searchIcons", () => {
  it("getAllIcons 扁平化并去重（DatabaseOutlined 跨 3 个分类只出现一次）", () => {
    const all = getAllIcons();
    expect(all).toContain("UserOutlined");
    expect(all.filter((name) => name === "DatabaseOutlined")).toHaveLength(1);
  });

  it("searchIcons 空关键词返回全部", () => {
    expect(searchIcons("")).toEqual(getAllIcons());
  });

  it("searchIcons 按图标名匹配（不区分大小写）", () => {
    const hits = searchIcons("user");
    expect(hits).toContain("UserOutlined");
    expect(hits).toEqual(["UserOutlined"]); // 分类清单中仅该图标名含 user
    expect(hits).not.toContain("DatabaseOutlined");
  });

  it("searchIcons 按中文描述匹配", () => {
    expect(searchIcons("数据库")).toContain("DatabaseOutlined");
    expect(searchIcons("仪表盘")).toContain("DashboardOutlined");
  });
});
