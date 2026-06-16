/**
 * FloorPlanEditor 缩放和平移逻辑 Hook
 */

import { useState, useCallback } from "react";
import type { ViewState } from "./FloorPlanEditor.types";
import { DEFAULT_SCALE, MIN_SCALE, MAX_SCALE, GRID_SIZE } from "./FloorPlanEditor.constants";

interface UsePanZoomOptions {
  containerRef: React.RefObject<SVGSVGElement | null>;
}

interface UsePanZoomReturn {
  viewState: ViewState;
  zoomIn: () => void;
  zoomOut: () => void;
  resetView: () => void;
  fitToScreen: () => void;
  handlePanStart: (clientX: number, clientY: number) => void;
  handlePanMove: (clientX: number, clientY: number) => void;
  handlePanEnd: () => void;
  handleWheel: (e: WheelEvent) => void;
  screenToSvg: (x: number, y: number) => { x: number; y: number };
}

export function usePanZoom(
  options: UsePanZoomOptions
): UsePanZoomReturn {
  const { containerRef } = options;

  const [viewState, setViewState] = useState<ViewState>({
    scale: DEFAULT_SCALE,
    offsetX: 0,
    offsetY: 0,
    isDragging: false,
    dragStartX: 0,
    dragStartY: 0,
  });

  // 缩放
  const zoomIn = useCallback(() => {
    setViewState(prev => ({
      ...prev,
      scale: Math.min(MAX_SCALE, prev.scale + 0.1),
    }));
  }, []);

  const zoomOut = useCallback(() => {
    setViewState(prev => ({
      ...prev,
      scale: Math.max(MIN_SCALE, prev.scale - 0.1),
    }));
  }, []);

  // 重置视图
  const resetView = useCallback(() => {
    setViewState({
      scale: DEFAULT_SCALE,
      offsetX: 0,
      offsetY: 0,
      isDragging: false,
      dragStartX: 0,
      dragStartY: 0,
    });
  }, []);

  // 适配屏幕
  const fitToScreen = useCallback(() => {
    const rect = containerRef.current?.getBoundingClientRect();
    if (!rect) return;

    const contentWidth = 2000; // 假设内容宽度
    const contentHeight = 1500; // 假设内容高度

    const scaleX = rect.width / contentWidth;
    const scaleY = rect.height / contentHeight;
    const fitScale = Math.min(scaleX, scaleY, 1) * 0.9; // 留10%边距

    setViewState({
      scale: Math.max(MIN_SCALE, Math.min(MAX_SCALE, fitScale)),
      offsetX: (rect.width - contentWidth * fitScale) / 2,
      offsetY: (rect.height - contentHeight * fitScale) / 2,
      isDragging: false,
      dragStartX: 0,
      dragStartY: 0,
    });
  }, [containerRef]);

  // 开始平移
  const handlePanStart = useCallback((clientX: number, clientY: number) => {
    setViewState(prev => ({
      ...prev,
      isDragging: true,
      dragStartX: clientX - prev.offsetX,
      dragStartY: clientY - prev.offsetY,
    }));
  }, []);

  // 平移移动
  const handlePanMove = useCallback((clientX: number, clientY: number) => {
    setViewState(prev => ({
      ...prev,
      offsetX: clientX - prev.dragStartX,
      offsetY: clientY - prev.dragStartY,
    }));
  }, []);

  // 结束平移
  const handlePanEnd = useCallback(() => {
    setViewState(prev => ({
      ...prev,
      isDragging: false,
    }));
  }, []);

  // 处理滚轮 - 缩放画布
  const handleWheel = useCallback((e: WheelEvent) => {
    // Ctrl + 滚轮缩放
    const isCtrlPressed = e.ctrlKey || e.metaKey;

    if (isCtrlPressed) {
      e.preventDefault();
      const delta = e.deltaY > 0 ? -0.1 : 0.1;
      const newScale = Math.min(MAX_SCALE, Math.max(MIN_SCALE, viewState.scale + delta));

      // 以鼠标位置为中心缩放
      const rect = containerRef.current?.getBoundingClientRect();
      if (!rect) return;

      const mouseX = e.clientX - rect.left;
      const mouseY = e.clientY - rect.top;

      const scaleChange = newScale / viewState.scale;

      setViewState(prev => ({
        ...prev,
        scale: newScale,
        offsetX: mouseX - (mouseX - prev.offsetX) * scaleChange,
        offsetY: mouseY - (mouseY - prev.offsetY) * scaleChange,
      }));
    } else {
      // 滚轮平移（带吸附）
      e.preventDefault();
      const delta = GRID_SIZE;
      setViewState(prev => ({
        ...prev,
        offsetX: prev.offsetX - (e.deltaX > 0 ? delta : e.deltaX < 0 ? -delta : 0),
        offsetY: prev.offsetY - (e.deltaY > 0 ? delta : e.deltaY < 0 ? -delta : 0),
      }));
    }
  }, [containerRef, viewState.scale]);

  // 屏幕坐标转换为 SVG 坐标
  const screenToSvg = useCallback((x: number, y: number) => {
    const rect = containerRef.current?.getBoundingClientRect();
    if (!rect) return { x: 0, y: 0 };

    return {
      x: (x - rect.left - viewState.offsetX) / viewState.scale,
      y: (y - rect.top - viewState.offsetY) / viewState.scale,
    };
  }, [viewState.offsetX, viewState.offsetY, viewState.scale, containerRef]);

  return {
    viewState,
    zoomIn,
    zoomOut,
    resetView,
    fitToScreen,
    handlePanStart,
    handlePanMove,
    handlePanEnd,
    handleWheel,
    screenToSvg,
  };
}
