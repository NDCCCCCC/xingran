/**
 * W7.1 / 缺陷 C：空页面 0 帧显式告知（前端渲染断言 + 反向用例）
 *
 * 缺陷实证（生产）：任务脚本动作没执行成功，页面停在 about:blank，
 * Chrome 对空页面**不产 screencast 帧**；前端只有「等待画面…」一句文案，
 * 用户（一般员工）无法判断是「还没开始」还是「永远不会有画面」，
 * 只能一直干等 10 分钟到执行超时。
 *
 * 本文件断言三件事（全部用户视角：能看到哪些字、能点哪个按钮）：
 *   1. 已推流 + 0 帧超时 → 显示「当前页面暂无画面（可能停在空白页或正在加载）」
 *      且出现「刷新状态」按钮，而不是「等待画面…」；
 *   2. 反向：有画面帧 → 不显示该提示；正常等待中（未越线）→ 仍是「等待画面…」；
 *   3. 文案对一般员工友好：不得出现 screencast / CDP / frame 这类术语。
 */
import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import {
  EMPTY_PAGE_HINT_TEXT,
  EMPTY_PAGE_REFRESH_TEXT,
  WAITING_FRAME_TEXT,
  LiveHud,
} from "../LiveHud";
import { parseEmptyPageHint } from "../liveFrameParser";

const COMPACT_PROPS = {
  context: null,
  mode: "copilot" as const,
  onModeChange: vi.fn(),
};

describe("缺陷 C：暂无画面提示", () => {
  it("已推流 0 帧超时 → 显示「暂无画面」而不是「等待画面…」", () => {
    render(<LiveHud {...COMPACT_PROPS} hasFrame={false} emptyPage onRefreshState={vi.fn()} />);

    const hint = screen.getByTestId("live-empty-page-hint");
    expect(hint).toHaveTextContent(EMPTY_PAGE_HINT_TEXT);
    // 关键反向断言：旧的模糊文案不得同时出现（否则用户仍会以为「再等等就好」）
    expect(screen.queryByText(WAITING_FRAME_TEXT)).toBeNull();
    expect(screen.queryByTestId("live-waiting-frame-hint")).toBeNull();
  });

  it("给出「刷新状态」按钮，点一下真的调用回调（用户能操作）", () => {
    const onRefresh = vi.fn();
    render(<LiveHud {...COMPACT_PROPS} hasFrame={false} emptyPage onRefreshState={onRefresh} />);

    const btn = screen.getByTestId("live-empty-page-refresh");
    expect(btn).toHaveTextContent(EMPTY_PAGE_REFRESH_TEXT);
    fireEvent.click(btn);
    expect(onRefresh).toHaveBeenCalledTimes(1);
  });

  it("反向用例：已有画面帧 → 不显示提示、也没有刷新按钮", () => {
    render(<LiveHud {...COMPACT_PROPS} hasFrame emptyPage={false} onRefreshState={vi.fn()} />);
    expect(screen.queryByTestId("live-empty-page-hint")).toBeNull();
    expect(screen.queryByTestId("live-empty-page-refresh")).toBeNull();
    // 帧已到 → 连「等待画面…」也不该出现
    expect(screen.queryByText(WAITING_FRAME_TEXT)).toBeNull();
  });

  it("未越线（已推流但还在正常等待）→ 仍显示「等待画面…」，不提前误报", () => {
    render(<LiveHud {...COMPACT_PROPS} hasFrame={false} emptyPage={false} />);
    expect(screen.getByTestId("live-waiting-frame-hint")).toHaveTextContent(WAITING_FRAME_TEXT);
    expect(screen.queryByTestId("live-empty-page-hint")).toBeNull();
  });

  it("文案面向一般员工：不含 screencast / CDP / frame 等术语", () => {
    const banned = ["screencast", "CDP", "frame", "Frame", "DevTools"];
    for (const word of banned) {
      expect(EMPTY_PAGE_HINT_TEXT).not.toContain(word);
      expect(WAITING_FRAME_TEXT).not.toContain(word);
      expect(EMPTY_PAGE_REFRESH_TEXT).not.toContain(word);
    }
    // 说明性括注必须给出「可能的原因」，否则用户仍不知道该怎么办
    expect(EMPTY_PAGE_HINT_TEXT).toContain("空白页");
    expect(EMPTY_PAGE_HINT_TEXT).toContain("加载");
  });
});

describe("缺陷 C：后端 emptyPageHint 消息解析", () => {
  it("解析后端 janitor 推送的提示信封", () => {
    const content = JSON.stringify({
      context: { executionId: "exec-1" },
      emptyPageHint: true,
      emptyPageWaitingMs: 5123,
    });
    const hint = parseEmptyPageHint(content);
    expect(hint).toEqual({ executionId: "exec-1", waitingMs: 5123 });
  });

  it("反向用例：普通情境消息（无 emptyPageHint）→ 不产生提示", () => {
    const content = JSON.stringify({
      context: { executionId: "exec-1", step: 2, totalSteps: 5, mode: "copilot" },
    });
    expect(parseEmptyPageHint(content)).toBeNull();
  });

  it("脏内容/空内容 → 安全返回 null（不崩、不误报）", () => {
    expect(parseEmptyPageHint("")).toBeNull();
    expect(parseEmptyPageHint("not-json")).toBeNull();
    expect(parseEmptyPageHint(JSON.stringify({ emptyPageHint: false }))).toBeNull();
  });
});
