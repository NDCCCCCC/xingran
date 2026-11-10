/**
 * components/shared/StatCards —— 统一统计卡（.stat-cards/.stat-card）
 *
 * 覆盖三段文本 + 色条语义 + 底边占比轨（ratio）+ 可点下钻（onClick）+ hint。
 */
import { describe, it, expect, vi } from "vitest";
import { fireEvent, render, screen } from "@testing-library/react";
import StatCards from "../StatCards";

describe("components/shared/StatCards", () => {
  it("渲染 label / value / trend 三段", () => {
    render(<StatCards items={[{ label: "用户总数", value: 128, trend: "全部门合计" }]} />);
    expect(screen.getByText("用户总数")).toBeInTheDocument();
    expect(screen.getByText("128")).toBeInTheDocument();
    expect(screen.getByText("全部门合计")).toBeInTheDocument();
  });

  it("show=false → 不渲染", () => {
    const { container } = render(<StatCards items={[{ label: "A", value: 1 }]} show={false} />);
    expect(container.firstChild).toBeNull();
  });

  it("tone 映射到色条 class", () => {
    const { container } = render(
      <StatCards
        items={[
          { label: "默认", value: 1 },
          { label: "正常", value: 2, tone: "green" },
          { label: "禁用", value: 3, tone: "gray" },
          { label: "部门", value: 4, tone: "gold" },
          { label: "危险", value: 5, tone: "danger" },
        ]}
      />
    );
    const cards = container.querySelectorAll(".stat-card");
    expect(cards[0].className).not.toMatch(/sc-/);
    expect(cards[1]).toHaveClass("sc-green");
    expect(cards[2]).toHaveClass("sc-gray");
    expect(cards[3]).toHaveClass("sc-gold");
    expect(cards[4]).toHaveClass("sc-danger");
  });

  it("ratio 注入 --ratio 并夹取到 0–1", () => {
    const { container } = render(
      <StatCards
        items={[
          { label: "无占比", value: 1 },
          { label: "正常", value: 2, ratio: 0.825 },
          { label: "越界高", value: 3, ratio: 3 },
          { label: "越界低", value: 4, ratio: -1 },
        ]}
      />
    );
    const cards = container.querySelectorAll(".stat-card");
    expect(cards[0].getAttribute("style")).toContain("--ratio: 0");
    expect(cards[1].getAttribute("style")).toContain("--ratio: 0.825");
    expect(cards[2].getAttribute("style")).toContain("--ratio: 1");
    expect(cards[3].getAttribute("style")).toContain("--ratio: 0");
  });

  it("onClick → 可点语义（role/tabIndex/is-clickable）且点击触发", () => {
    const onClick = vi.fn();
    const { container } = render(
      <StatCards items={[{ label: "禁用用户", value: 3, onClick, hint: "只看禁用用户" }]} />
    );
    const card = container.querySelector(".stat-card") as HTMLElement;
    expect(card).toHaveClass("is-clickable");
    expect(card).toHaveAttribute("role", "button");
    expect(card).toHaveAttribute("tabindex", "0");
    expect(card).toHaveAttribute("title", "只看禁用用户");
    fireEvent.click(card);
    expect(onClick).toHaveBeenCalledTimes(1);
  });

  it("无 onClick → 不可点（无 role / 无 is-clickable）", () => {
    const { container } = render(<StatCards items={[{ label: "部门数", value: 9 }]} />);
    const card = container.querySelector(".stat-card") as HTMLElement;
    expect(card).not.toHaveClass("is-clickable");
    expect(card).not.toHaveAttribute("role");
    expect(card).not.toHaveAttribute("tabindex");
  });

  it("键盘 Enter / Space 等价于点击", () => {
    const onClick = vi.fn();
    const { container } = render(<StatCards items={[{ label: "正常用户", value: 7, onClick }]} />);
    const card = container.querySelector(".stat-card") as HTMLElement;
    fireEvent.keyDown(card, { key: "Enter" });
    fireEvent.keyDown(card, { key: " " });
    expect(onClick).toHaveBeenCalledTimes(2);
  });
});
