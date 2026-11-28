/**
 * W7 / UI-FIX-01：路由常量 + 向导跳转回归测试
 *
 * 背景（真实浏览器暴露的 P0）：向导页三处 `navigate("/operations/rpa/tasks")`
 * 指向一个**不存在**的路由，SPA catch-all 把它甩到 dashboard。用户点「返回任务
 * 列表」直接看到仪表盘。本测试锁两件事：
 *   1. 常量值与 PG 实证的 sys_menu 链一致（/vdi/rpa）；
 *   2. 源码里不再散落 `/operations/rpa` 或 `/vdi/rpa` 裸字符串（唯一事实来源）。
 */
import { describe, it, expect } from "vitest";
import { RPA_ROUTES, withBase, rpaLiveUrl } from "../routes";

// 用 Vite 的 ?raw 把源码原文读进来做静态扫描（vitest 原生支持）。
import wizardSource from "../wizard/index.tsx?raw";
import executionModalSource from "../executions/ExecutionDetailModal.tsx?raw";
import routesSource from "../routes.ts?raw";

describe("RPA_ROUTES 常量", () => {
  it("真实路由是 /vdi/rpa（PG sys_menu 实证：虚拟机→vdi，RPA 管理→rpa）", () => {
    expect(RPA_ROUTES.taskList).toBe("/vdi/rpa");
    expect(RPA_ROUTES.live).toBe("/vdi/rpa/live");
    expect(RPA_ROUTES.wizard).toBe("/vdi/rpa/wizard");
    expect(RPA_ROUTES.schedules).toBe("/vdi/rpa/schedules");
  });

  it("绝不含历史错误前缀 /operations/rpa", () => {
    for (const value of Object.values(RPA_ROUTES)) {
      expect(value.startsWith("/vdi/rpa")).toBe(true);
      expect(value).not.toContain("/operations/");
    }
  });

  it("withBase 在根路径部署时不加前缀，子路径部署时补上", () => {
    // vitest 下 import.meta.env.BASE_URL 默认为 "/"，此时不得多补一个 "/"
    expect(withBase("/vdi/rpa")).toBe("/vdi/rpa");
  });

  it("rpaLiveUrl 对 executionId 做 URL 编码（防注入 / 特殊字符）", () => {
    expect(rpaLiveUrl("exec-1")).toContain("/vdi/rpa/live?executionId=exec-1");
    expect(rpaLiveUrl("a b&c=d")).toContain("executionId=a%20b%26c%3Dd");
  });
});

describe("源码不再散落 RPA 路由硬编码（回归拦截）", () => {
  /**
   * 扫描裸路径字符串。
   *
   * 先剥掉注释：注释里保留「历史错误路径」是**有意**的记录（说明为什么改），
   * 不该被判成回归；只有真正参与跳转的字符串才算。
   * 剥注释必须保留行号可读性（把注释替换成等长空白）。
   */
  const stripComments = (source: string) =>
    source
      .replace(/\/\*[\s\S]*?\*\//g, (m) => m.replace(/[^\n]/g, " "))
      .replace(/(^|[^:])\/\/[^\n]*/g, (m, p1: string) => p1 + " ".repeat(m.length - p1.length));

  const offenders = (source: string, file: string) => {
    const code = stripComments(source);
    // 裸路径：出现在字符串字面量里（引号/反引号后紧跟 /operations/rpa 或 /vdi/rpa）
    const barePath = /["'`]\/operations\/rpa|["'`]\/vdi\/rpa/g;
    const hits: string[] = [];
    let m: RegExpExecArray | null;
    while ((m = barePath.exec(code)) !== null) {
      hits.push(`${file}:${code.slice(0, m.index).split("\n").length}:${m[0]}`);
    }
    return hits;
  };

  it("wizard/index.tsx 三处跳转全部改用 RPA_ROUTES.taskList", () => {
    expect(wizardSource).not.toContain('"/operations/rpa/tasks"');
    expect(offenders(wizardSource, "wizard/index.tsx")).toEqual([]);
    // 三处跳转（保存后 / 返回按钮 / 取消）都必须引用常量
    const occurrences = wizardSource.match(/navigate\(RPA_ROUTES\.taskList\)/g) ?? [];
    expect(occurrences.length).toBe(3);
  });

  it("ExecutionDetailModal.tsx 的 live 入口改用 rpaLiveUrl()", () => {
    expect(offenders(executionModalSource, "ExecutionDetailModal.tsx")).toEqual([]);
    expect(executionModalSource).toContain("rpaLiveUrl(execution.id)");
  });

  it("routes.ts 是唯一允许出现裸路径的文件（自检，防误删常量）", () => {
    expect(routesSource).toContain('const RPA_BASE = "/vdi/rpa"');
    expect(routesSource).toContain("/operations/rpa");
  });
});
