/**
 * 两页设置分类注册表完整性测试（Phase 70 · D-02/D-07/D-08；v1.56 目录化更新）
 *
 * 目的：把系统设置页与用户设置页的分类注册表结构锁在测试层 ——
 * key 集合 / 唯一性 / label 非空 / icon 有效 / maxWidth 宽度策略任一
 * 被后续重构破坏即 fail，防止 Shell 侧导航渲染退化（缺 icon/空 label）
 * 或 URL ?cat= 参数与注册表 key 脱钩（key 改名而 defaultCat 未跟）。
 *
 * 纯数据断言：不渲染组件、不需要 Router/antd Wrapper；导入注册表模块
 * 即执行页面模块级 JSX（icon/content 为静态元素，无副作用）。
 *
 * 覆盖：
 * - v1.56：系统设置 5 分类 email/api/captcha/manager/backup（经理同步 + 备份与恢复并入），
 *   表格/网格类无 maxWidth（撑满，D-02）；仅「经理同步」带权限码（无 list 权限时隐藏）。
 * - D-06：用户设置 3 分类 appearance/layout/data，表单类 maxWidth 760（限宽，D-02）
 * - SettingsShell defaultCat 消费契约：两页默认分类 key 存在于各自注册表
 */

import { describe, it, expect } from "vitest";
import { isValidElement } from "react";
import { systemSettingsCategories } from "@/pages/system/settings/categories";
import { userSettingsCategories } from "@/pages/settings/categories";
import { PERMISSION_CODES } from "@/constants/permissionCodes";

describe("systemSettingsCategories（系统设置注册表）", () => {
  it("恰好 5 个分类：邮箱 / API通知 / 验证码背景图 / 经理同步 / 备份与恢复", () => {
    expect(systemSettingsCategories).toHaveLength(5);
  });

  it("key 集合 = email/api/captcha/manager/backup 且唯一（?cat= 参数值契约）", () => {
    const keys = systemSettingsCategories.map((c) => c.key);
    expect(keys).toEqual(["email", "api", "captcha", "manager", "backup"]);
    expect(new Set(keys).size).toBe(keys.length);
  });

  it("每项 label 为非空字符串（目录导航可读）", () => {
    for (const c of systemSettingsCategories) {
      expect(typeof c.label).toBe("string");
      expect(c.label.trim().length).toBeGreaterThan(0);
    }
  });

  it("每项 icon 为有效 ReactNode（isValidElement）", () => {
    for (const c of systemSettingsCategories) {
      expect(isValidElement(c.icon)).toBe(true);
    }
  });

  it("每项 content 为有效 ReactNode（平铺区块内容）", () => {
    for (const c of systemSettingsCategories) {
      expect(isValidElement(c.content)).toBe(true);
    }
  });

  it("全部无 maxWidth：表格/网格类分类撑满容器（D-02 混合宽度策略）", () => {
    for (const c of systemSettingsCategories) {
      expect(c.maxWidth).toBeUndefined();
    }
  });

  it("仅经理同步带权限码 ops:manager:rule:list（无权限隐藏目录项与区块）", () => {
    const manager = systemSettingsCategories.find((c) => c.key === "manager");
    expect(manager?.permission).toBe(PERMISSION_CODES.opsManagerRule.list);

    // 其余分类常驻：备份页的 F 按钮码在存量库中未必授给超管（240 不写 sys_role_menu），
    // 若按 F 码隐藏整块会让超管彻底失去功能入口。
    for (const c of systemSettingsCategories) {
      if (c.key !== "manager") {
        expect(c.permission).toBeUndefined();
      }
    }
  });

  it("默认分类 key=email 存在（SettingsShell defaultCat 消费契约）", () => {
    expect(systemSettingsCategories.some((c) => c.key === "email")).toBe(true);
  });
});

describe("userSettingsCategories（用户设置注册表）", () => {
  it("恰好 3 个分类：界面 / 布局 / 数据", () => {
    expect(userSettingsCategories).toHaveLength(3);
  });

  it("key 集合 = appearance/layout/data 且唯一（?cat= 参数值契约）", () => {
    const keys = userSettingsCategories.map((c) => c.key);
    expect(keys).toEqual(["appearance", "layout", "data"]);
    expect(new Set(keys).size).toBe(keys.length);
  });

  it("每项 label 为非空字符串（侧栏导航可读）", () => {
    for (const c of userSettingsCategories) {
      expect(typeof c.label).toBe("string");
      expect(c.label.trim().length).toBeGreaterThan(0);
    }
  });

  it("每项 icon 为有效 ReactNode（isValidElement）", () => {
    for (const c of userSettingsCategories) {
      expect(isValidElement(c.icon)).toBe(true);
    }
  });

  it("每项 maxWidth === 760：表单类分类限宽（D-02 混合宽度策略）", () => {
    for (const c of userSettingsCategories) {
      expect(c.maxWidth).toBe(760);
    }
  });

  it("默认分类 key=appearance 存在（SettingsShell defaultCat 消费契约）", () => {
    expect(userSettingsCategories.some((c) => c.key === "appearance")).toBe(true);
  });
});
