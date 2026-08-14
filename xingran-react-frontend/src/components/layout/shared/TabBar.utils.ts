/**
 * TabBar 组件工具函数
 * TabBar component utility functions
 */

import type { RefObject } from "react";
import { SCROLL_TOLERANCE } from "./TabBar.constants";

/** 滚动状态 */
export interface ScrollState {
  canScrollLeft: boolean;
  canScrollRight: boolean;
  scrollLeft: number;
}

/**
 * 检测滚动容器状态
 */
export function checkScrollState(container: HTMLElement | null): ScrollState {
  if (!container) {
    return { canScrollLeft: false, canScrollRight: false, scrollLeft: 0 };
  }

  const { scrollLeft, scrollWidth, clientWidth } = container;
  const canScrollLeft = scrollLeft > 0;
  const canScrollRight =
    scrollWidth > clientWidth && scrollLeft < scrollWidth - clientWidth - SCROLL_TOLERANCE;

  return { canScrollLeft, canScrollRight, scrollLeft };
}

/**
 * 执行滚动操作
 */
export function scrollContainer(
  container: HTMLElement | null,
  direction: "left" | "right",
  step: number
): void {
  if (!container) return;

  const currentScroll = container.scrollLeft;
  container.scrollTo({
    left: direction === "left" ? currentScroll - step : currentScroll + step,
    behavior: "smooth",
  });
}

/**
 * 设置延迟定时器
 */
export function setupDelayedChecks(callback: () => void, delays: readonly number[]): number[] {
  return delays.map((delay) => setTimeout(callback, delay));
}
