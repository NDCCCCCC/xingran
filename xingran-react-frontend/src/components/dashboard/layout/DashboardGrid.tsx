/**
 * DashboardGrid - 仪表盘网格布局容器
 *
 * 基于 react-grid-layout 实现的可拖拽、可调整大小的网格布局
 * 支持响应式布局和拖拽优化
 */

import { type FC, type ReactNode, useRef, useEffect, useState, useCallback, useMemo } from "react";
import { Responsive, type Layout, type ResponsiveProps } from "react-grid-layout";
import { useWindowSize } from "@/hooks/useWindowSize";
import { useDashboardStore } from "@/store/dashboardStore";
import type { WidgetConfig } from "@/types/dashboard";
import { defaultLayoutConfig } from "@/types/dashboard";

import "react-grid-layout/css/styles.css";
import "react-resizable/css/styles.css";
import "./DashboardGrid.css";

// 扩展 ResponsiveProps 以包含 isDraggable 和 isResizable
interface ExtendedResponsiveProps extends ResponsiveProps {
	isDraggable?: boolean;
	isResizable?: boolean;
	compactType?: "vertical" | "horizontal" | null;
	preventCollision?: boolean;
	draggableHandle?: string;
	useCSSTransforms?: boolean;
	children: React.ReactNode;
}

interface DashboardGridProps {
	/** Widget 列表 */
	widgets: WidgetConfig[];

	/** 布局变更回调 */
	onLayoutChange: (layouts: Array<{ id: string; position: WidgetConfig["position"] }>) => void;

	/** 子元素 */
	children: ReactNode;
}

export const DashboardGrid: FC<DashboardGridProps> = ({
	widgets,
	onLayoutChange,
	children,
}) => {
	const windowSize = useWindowSize();
	const { viewMode } = useDashboardStore();
	const layoutConfig = defaultLayoutConfig;
	const isEditable = viewMode === "edit";

	// 使用 ref 跟踪容器宽度
	const containerRef = useRef<HTMLDivElement>(null);
	const [containerWidth, setContainerWidth] = useState<number>(windowSize.width);

	// 监听容器宽度变化
	useEffect(() => {
		if (!containerRef.current) return;

		const resizeObserver = new ResizeObserver((entries) => {
			for (const entry of entries) {
				setContainerWidth(entry.contentRect.width);
			}
		});

		resizeObserver.observe(containerRef.current);
		return () => resizeObserver.disconnect();
	}, []);

	// 响应式断点配置 - 三档布局
	const breakpoints = useMemo(() => ({
		lg: 1200,  // 桌面端
		md: 996,   // 中等屏幕
		sm: 768,   // 平板端
		xs: 480,   // 小屏幕
		xxs: 0     // 移动端
	}), []);

	// 响应式列数配置
	const cols = useMemo(() => ({
		lg: 24,  // 桌面端 24 列
		md: 12,  // 中等屏幕 12 列
		sm: 8,   // 平板端 8 列
		xs: 4,   // 小屏幕 4 列
		xxs: 2   // 移动端 2 列
	}), []);

	// 根据屏幕宽度获取当前断点
	const getCurrentBreakpoint = useCallback(() => {
		const width = containerWidth;
		if (width >= 1200) return "lg";
		if (width >= 996) return "md";
		if (width >= 768) return "sm";
		if (width >= 480) return "xs";
		return "xxs";
	}, [containerWidth]);

	// 响应式列数获取
	const getColumns = useCallback(() => {
		const breakpoint = getCurrentBreakpoint();
		return cols[breakpoint];
	}, [getCurrentBreakpoint, cols]);

	// 是否为移动端
	const isMobile = useMemo(() => {
		return getCurrentBreakpoint() === "xs" || getCurrentBreakpoint() === "xxs";
	}, [getCurrentBreakpoint]);

	// 将 Widget 配置转换为 react-grid-layout 布局格式
	// 根据屏幕尺寸调整 Widget 的最小/最大尺寸
	const layouts: Layout = useMemo(() => widgets.map((widget) => {
		// 移动端固定全宽
		if (isMobile) {
			return {
				i: widget.id,
				x: 0,
				y: widget.position.y,
				w: getColumns(),
				h: widget.position.h,
				minW: getColumns(),
				minH: widget.position.minH ?? 2,
				maxW: getColumns(),
				maxH: widget.position.maxH,
			};
		}

		return {
			i: widget.id,
			x: widget.position.x,
			y: widget.position.y,
			w: widget.position.w,
			h: widget.position.h,
			minW: widget.position.minW ?? 3,
			minH: widget.position.minH ?? 3,
			maxW: widget.position.maxW ?? 24,
			maxH: widget.position.maxH,
		};
	}), [widgets, isMobile, getColumns]);

	// 布局变更处理
	const handleLayoutChange = (currentLayout: Layout) => {
		const updatedWidgets = currentLayout.map((layout) => ({
			id: layout.i,
			position: {
				x: layout.x ?? 0,
				y: layout.y ?? 0,
				w: layout.w ?? 1,
				h: layout.h ?? 1,
			} as WidgetConfig["position"],
		}));
		onLayoutChange(updatedWidgets);
	};

	// 拖拽开始/结束
	const handleDragStart = useCallback(() => {
		useDashboardStore.getState().startDragging("dragging");
	}, []);

	const handleDragStop = useCallback(() => {
		useDashboardStore.getState().stopDragging();
	}, []);

	const handleResizeStart = useCallback(() => {
		useDashboardStore.getState().startDragging("dragging");
	}, []);

	const handleResizeStop = useCallback(() => {
		useDashboardStore.getState().stopDragging();
	}, []);

	// 准备 props - 使用 proper 类型断言
	// 移动端禁用拖拽和调整大小
	const gridProps: ExtendedResponsiveProps = {
		className: "layout",
		width: containerWidth,
		layouts: { lg: layouts },
		breakpoints,
		cols,
		rowHeight: layoutConfig.rowHeight,
		margin: layoutConfig.margin,
		containerPadding: [16, 16],
		isDraggable: isEditable && layoutConfig.draggable && !isMobile,
		isResizable: isEditable && layoutConfig.resizable && !isMobile,
		onLayoutChange: handleLayoutChange,
		onDragStart: handleDragStart,
		onDragStop: handleDragStop,
		onResizeStart: handleResizeStart,
		onResizeStop: handleResizeStop,
		compactType: "vertical",
		preventCollision: true,  // 防止 Widget 重叠
		draggableHandle: ".widget-drag-handle",  // 指定拖拽手柄
		useCSSTransforms: true,  // 使用 CSS transform 提升性能
		children,
	};

	return (
		<div ref={containerRef} className={`dashboard-grid ${isEditable ? "dashboard-grid--editable" : ""} ${isMobile ? "dashboard-grid--mobile" : ""}`}>
			<Responsive {...gridProps}>
				{children}
			</Responsive>
		</div>
	);
};

/**
 * DashboardGridPlaceholder - 空状态占位符
 */
export const DashboardGridPlaceholder: FC<{ message: string }> = ({ message }) => {
	return (
		<div className="dashboard-grid-placeholder">
			<div className="dashboard-grid-placeholder__content">
				<div className="dashboard-grid-placeholder__icon">📊</div>
				<h3 className="dashboard-grid-placeholder__title">仪表盘为空</h3>
				<p className="dashboard-grid-placeholder__message">{message}</p>
			</div>
		</div>
	);
};
