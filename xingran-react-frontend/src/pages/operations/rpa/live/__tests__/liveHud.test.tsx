/**
 * Phase 155 / LIVEVIEW-04：叠加层（HUD）渲染断言
 *
 * 覆盖点：
 *   1. 步骤 step/totalSteps 文本
 *   2. 角标「已接管 · <name>」红色语义 vs「自动执行」绿色语义
 *   3. 脱敏变量原样显示 `***`（绝不还原明文）
 *   4. 变量超过 8 条折叠为「更多 N 条」
 *   5. 最近日志单行截断 + tooltip 承载全文
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { HUD_MAX_VARIABLES, LiveHud } from "../LiveHud";
import type { LiveContext } from "../types";

/** 构造情境（默认未接管） */
function makeContext(overrides: Partial<LiveContext> = {}): LiveContext {
  return {
    executionId: "exec-1",
    taskName: "任务一",
    step: 3,
    totalSteps: 10,
    actionType: "click",
    status: "running",
    mode: "copilot",
    lastLog: "点击「登录」按钮",
    variables: { user: "alice", password: "***", token: "***" },
    ...overrides,
  };
}

describe("LiveHud 叠加层", () => {
  it("显示步骤序号/总步数 与 动作类型中文", () => {
    render(<LiveHud context={makeContext()} mode="copilot" onModeChange={vi.fn()} hasFrame />);
    expect(screen.getByTestId("live-hud-step")).toHaveTextContent("步骤 3/10");
    expect(screen.getByTestId("live-hud-action")).toHaveTextContent("点击");
  });

  it("takeoverBy 为空 → 角标「自动执行」", () => {
    render(
      <LiveHud
        context={makeContext({ takeoverBy: "" })}
        mode="auto"
        onModeChange={vi.fn()}
        hasFrame
      />
    );
    const badge = screen.getByTestId("live-hud-badge");
    expect(badge).toHaveTextContent("自动执行");
    // 绿色语义：用主题 success 令牌而非硬编码色值
    expect(badge.style.color).toContain("--theme-success");
  });

  it("takeoverBy 非空 → 角标「已接管 · 张三」（红色语义）", () => {
    render(
      <LiveHud
        context={makeContext({ takeoverBy: "张三" })}
        mode="manual"
        onModeChange={vi.fn()}
        hasFrame
      />
    );
    const badge = screen.getByTestId("live-hud-badge");
    expect(badge).toHaveTextContent("已接管 · 张三");
    expect(badge.style.color).toContain("--theme-error");
  });

  it("脱敏变量原样显示 ***（password/token 都不还原）", () => {
    render(<LiveHud context={makeContext()} mode="copilot" onModeChange={vi.fn()} hasFrame />);
    expect(screen.getByText("user=alice")).toBeInTheDocument();
    // 两个 *** 变量必须各自出现（不是只显示一条）
    expect(screen.getByText("password=***")).toBeInTheDocument();
    expect(screen.getByText("token=***")).toBeInTheDocument();
  });

  it("变量超过 8 条 → 折叠为「更多 N 条」", () => {
    const variables: Record<string, string> = {};
    for (let i = 1; i <= 11; i++) {
      variables[`v${i}`] = "***";
    }
    render(
      <LiveHud
        context={makeContext({ variables })}
        mode="copilot"
        onModeChange={vi.fn()}
        hasFrame
      />
    );
    // 只显示前 8 条
    expect(screen.getByText("v1=***")).toBeInTheDocument();
    expect(screen.getByText(`v${HUD_MAX_VARIABLES}=***`)).toBeInTheDocument();
    expect(screen.queryByText(`v${HUD_MAX_VARIABLES + 1}=***`)).toBeNull();
    // 剩下 3 条收进「更多」
    expect(screen.getByText("更多 3 条")).toBeInTheDocument();
  });

  it("最近日志单行截断 + tooltip 承载全文", () => {
    const longLog = "这是一条很长的执行日志，用来验证 HUD 里只占一行并保留全文到 tooltip";
    render(
      <LiveHud
        context={makeContext({ lastLog: longLog })}
        mode="copilot"
        onModeChange={vi.fn()}
        hasFrame
      />
    );
    const text = screen.getByText(longLog);
    expect(text).toBeInTheDocument();
    expect(text.style.whiteSpace).toBe("nowrap");
    expect(text.style.textOverflow).toBe("ellipsis");
  });

  it("无 context 时不崩、步骤回落 0/0、角标仍是「自动执行」", () => {
    render(<LiveHud context={null} mode="copilot" onModeChange={vi.fn()} hasFrame={false} />);
    expect(screen.getByTestId("live-hud-step")).toHaveTextContent("步骤 0/0");
    expect(screen.getByTestId("live-hud-badge")).toHaveTextContent("自动执行");
    // 未收到首帧 → 等待提示
    expect(screen.getByText("等待画面…")).toBeInTheDocument();
  });

  it("显示帧率/带宽小字（有统计时）", () => {
    render(
      <LiveHud
        context={makeContext()}
        mode="copilot"
        onModeChange={vi.fn()}
        stats={{ fps: 8.5, mbps: 1.2, samples: 26 }}
        hasFrame
      />
    );
    expect(screen.getByText("8.5 fps / 1.2 Mbps")).toBeInTheDocument();
    expect(screen.getByText("模式：副驾")).toBeInTheDocument();
  });
});
