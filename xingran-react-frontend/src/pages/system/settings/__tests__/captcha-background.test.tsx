/**
 * Phase 70-04 — 验证码背景图网格墙 Wave 0 单测（D-08 status 反转语义锁定）
 *
 * 用例覆盖（70-VALIDATION.md Per-Task Map 的 D-08 行）：
 *   1. status=1 记录渲染「启用」徽标（xr-tag-green）—— captcha 例外语义 1=启用。
 *   2. status=0 记录渲染「禁用」徽标（中性 xr-tag，无 green）。
 *   3. 启停操作文案取反：status=1 卡提供「禁用」动作、status=0 卡提供「启用」动作。
 *   4. statistics fixture（totalCount/enabledCount/disabledCount/totalUsage）渲染 4 张统计卡数值。
 *
 * 关键语义（70-RESEARCH Pitfall 1）：captcha 背景 status 与全局「0=启用」惯例相反，
 * 后端契约为 1=启用 / 0=禁用，统计卡与网格墙徽标均按 status===1 取「启用」，勿「纠正」。
 *
 * Mock 策略：vi.mock("@/services/captcha") 直通可控 fixture（vi.hoisted 规避 factory
 * 提升导致的 TDZ）；渲染三件套与 SettingsShell.test.tsx 同款（ResizeObserver stub +
 * MemoryRouter + antd App Wrapper）。
 */

import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, within, waitFor } from "@testing-library/react";
import { App } from "antd";
import { MemoryRouter } from "react-router-dom";
import type {
  CaptchaBackground,
  CaptchaBackgroundStatus,
  StatisticsResponse,
} from "@/types/captcha";

// ---- Polyfill: antd v6 ResizeObserver (jsdom 缺失) ----
class ResizeObserverStub {
  observe() {}
  unobserve() {}
  disconnect() {}
}
if (typeof globalThis.ResizeObserver === "undefined") {
  (globalThis as unknown as { ResizeObserver: typeof ResizeObserverStub }).ResizeObserver =
    ResizeObserverStub;
}

// ---- Mock: @/services/captcha（vi.hoisted 使 mock fn 先于 hoisted factory 可用） ----
const { getListMock, getStatsMock } = vi.hoisted(() => ({
  getListMock: vi.fn(),
  getStatsMock: vi.fn(),
}));

vi.mock("@/services/captcha", () => ({
  getCaptchaBackgroundList: getListMock,
  getCaptchaBackgroundStatistics: getStatsMock,
  uploadCaptchaBackground: vi.fn(),
  updateCaptchaBackground: vi.fn(),
  deleteCaptchaBackground: vi.fn(),
  toggleCaptchaBackgroundStatus: vi.fn(async () => undefined),
  preloadCaptchaCache: vi.fn(async () => ({ preloaded: 0, message: "ok" })),
}));

import CaptchaBackgroundSettingsPage from "../captcha-background";

// ---- 夹具：status 1 与 0 各一条（带 previewUrl），统计固定数值 ----
const makeBackground = (
  id: string,
  fileName: string,
  status: CaptchaBackgroundStatus
): CaptchaBackground => ({
  id,
  fileName,
  filePath: `/uploads/captcha/${fileName}`,
  fileSize: 512000,
  fileWidth: 600,
  fileHeight: 400,
  pieceShape: "circle",
  difficultyLevel: 2,
  useCount: 7,
  sortOrder: 1,
  status,
  createdAt: "2026-08-01T00:00:00Z",
  updatedAt: "2026-08-01T00:00:00Z",
  previewUrl: `/preview/${fileName}`,
});

const enabledBg = makeBackground("bg-enabled-1", "forest-01.png", 1);
const disabledBg = makeBackground("bg-disabled-2", "city-02.png", 0);

const statsFixture: StatisticsResponse = {
  totalCount: 5,
  enabledCount: 3,
  disabledCount: 2,
  totalUsage: 42,
  shapeDistribution: {},
  difficultyDist: {},
};

function renderPage() {
  return render(
    <MemoryRouter initialEntries={["/system/settings-page"]}>
      <App>
        <CaptchaBackgroundSettingsPage />
      </App>
    </MemoryRouter>
  );
}

describe("captcha-background 网格墙 — Phase 70-04 D-08 status 反转语义锁定", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("用例1: status=1 记录渲染「启用」徽标（xr-tag-green）—— captcha 例外语义 1=启用", async () => {
    getListMock.mockResolvedValue({ items: [enabledBg], total: 1 });
    getStatsMock.mockResolvedValue(statsFixture);
    renderPage();

    // 列表已加载（文件名出现在卡脚）
    expect(await screen.findByText("forest-01.png")).toBeInTheDocument();

    // 卡脚徽标 = 唯一 xr-tag，status=1 → 启用 + xr-tag-green
    const badges = document.querySelectorAll(".xr-captcha-grid .xr-tag");
    expect(badges).toHaveLength(1);
    expect(badges[0]).toHaveTextContent("启用");
    expect(badges[0].classList.contains("xr-tag-green")).toBe(true);
  });

  it("用例2: status=0 记录渲染「禁用」徽标（中性 xr-tag，无 green）", async () => {
    getListMock.mockResolvedValue({ items: [disabledBg], total: 1 });
    getStatsMock.mockResolvedValue(statsFixture);
    renderPage();

    expect(await screen.findByText("city-02.png")).toBeInTheDocument();

    const badges = document.querySelectorAll(".xr-captcha-grid .xr-tag");
    expect(badges).toHaveLength(1);
    expect(badges[0]).toHaveTextContent("禁用");
    // 禁用不是错误，不用 green（UI-SPEC：禁用/停用态为中性灰）
    expect(document.querySelector(".xr-captcha-grid .xr-tag-green")).toBeNull();
  });

  it("用例3: 启停操作文案取反 — status=1 卡提供「禁用」动作，status=0 卡提供「启用」动作", async () => {
    getListMock.mockResolvedValue({ items: [enabledBg, disabledBg], total: 2 });
    getStatsMock.mockResolvedValue(statsFixture);
    renderPage();

    expect(await screen.findByText("forest-01.png")).toBeInTheDocument();

    const cards = document.querySelectorAll(".xr-captcha-card");
    expect(cards).toHaveLength(2);

    // 启用卡（status=1）：徽标「启用」+ 操作行提供「禁用」（取反动作），无「启用」按钮
    const enabledCard = within(cards[0]);
    expect(enabledCard.getByText("启用")).toBeInTheDocument(); // 徽标 span
    expect(enabledCard.getByRole("button", { name: "禁用" })).toBeInTheDocument();
    expect(enabledCard.queryByRole("button", { name: "启用" })).not.toBeInTheDocument();

    // 禁用卡（status=0）：徽标「禁用」+ 操作行提供「启用」（取反动作），无「禁用」按钮
    const disabledCard = within(cards[1]);
    expect(disabledCard.getByText("禁用")).toBeInTheDocument(); // 徽标 span
    expect(disabledCard.getByRole("button", { name: "启用" })).toBeInTheDocument();
    expect(disabledCard.queryByRole("button", { name: "禁用" })).not.toBeInTheDocument();
  });

  it("用例4: statistics fixture 渲染 4 张统计卡数值（总数量/启用数量/禁用数量/总使用次数）", async () => {
    getListMock.mockResolvedValue({ items: [enabledBg], total: 1 });
    getStatsMock.mockResolvedValue(statsFixture);
    renderPage();

    expect(await screen.findByText("forest-01.png")).toBeInTheDocument();

    // 统计为独立异步请求，waitFor 至 4 卡数值就位（顺序：总数 5 / 启用 3 / 禁用 2 / 使用 42）
    await waitFor(() => {
      const values = Array.from(document.querySelectorAll(".stat-value")).map(
        (el) => el.textContent
      );
      expect(values).toEqual(["5", "3", "2", "42"]);
    });

    expect(screen.getByText("总数量")).toBeInTheDocument();
    expect(screen.getByText("启用数量")).toBeInTheDocument();
    expect(screen.getByText("禁用数量")).toBeInTheDocument();
    expect(screen.getByText("总使用次数")).toBeInTheDocument();
  });
});
