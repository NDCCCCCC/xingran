/**
 * DashboardGrid - 仪表盘网格布局容器
 *
 * 基于 react-grid-layout 实现的可拖拽、可调整大小的网格布局
 */

import { type FC, type ReactNode, useRef, useEffect, useState, useCallback } from 'react';
import { Responsive, type Layout, type ResponsiveProps } from 'react-grid-layout';
import { useWindowSize } from '@/hooks/useWindowSize';
import { useDashboardStore } from '@/store/dashboardStore';
import type { WidgetConfig } from '@/types/dashboard';
import { defaultLayoutConfig } from '@/types/dashboard';

import 'react-grid-layout/css/styles.css';
import 'react-resizable/css/styles.css';
import './DashboardGrid.css';

// 扩展 ResponsiveProps 以包含 isDraggable 和 isResizable
interface ExtendedResponsiveProps extends ResponsiveProps {
	isDraggable?: boolean;
	isResizable?: boolean;
	compactType?: 'vertical' | 'horizontal' | null;
	preventCollision?: boolean;
	children: React.ReactNode;
}

interface DashboardGridProps {
	/** Widget 列表 */
	widgets: WidgetConfig[];

	/** 布局变更回调 */
	onLayoutChange: (layouts: Array<{ id: string; position: WidgetConfig['position'] }>) => void;

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
	const isEditable = viewMode === 'edit';

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

	// 响应式列数配置
	const getColumns = () => {
		const width = containerWidth;
		if (width < 768) return layoutConfig.columns.mobile;
		if (width < 1024) return layoutConfig.columns.tablet;
		return layoutConfig.columns.desktop;
	};

	// 将 Widget 配置转换为 react-grid-layout 布局格式
	const layouts: Layout = widgets.map((widget) => ({
		i: widget.id,
		x: widget.position.x,
		y: widget.position.y,
		w: widget.position.w,
		h: widget.position.h,
		minW: widget.position.minW,
		minH: widget.position.minH,
		maxW: widget.position.maxW,
		maxH: widget.position.maxH,
	}));

	// 布局变更处理
	const handleLayoutChange = (currentLayout: Layout) => {
		const updatedWidgets = currentLayout.map((layout) => ({
			id: layout.i,
			position: {
				x: layout.x ?? 0,
				y: layout.y ?? 0,
				w: layout.w ?? 1,
				h: layout.h ?? 1,
			} as WidgetConfig['position'],
		}));
		onLayoutChange(updatedWidgets);
	};

	// 拖拽开始/结束
	const handleDragStart = useCallback(() => {
		useDashboardStore.getState().startDragging('dragging');
	}, []);

	const handleDragStop = useCallback(() => {
		useDashboardStore.getState().stopDragging();
	}, []);

	const handleResizeStart = useCallback(() => {
		useDashboardStore.getState().startDragging('dragging');
	}, []);

	const handleResizeStop = useCallback(() => {
		useDashboardStore.getState().stopDragging();
	}, []);

	// 准备 props - 使用 proper 类型断言
	const gridProps: ExtendedResponsiveProps = {
		className: 'layout',
		width: containerWidth,
		layouts: { lg: layouts },
		breakpoints: { lg: 1024, md: 768, sm: 576 },
		cols: { lg: getColumns(), md: getColumns(), sm: getColumns() },
		rowHeight: layoutConfig.rowHeight,
		margin: layoutConfig.margin,
		containerPadding: [16, 16],
		isDraggable: isEditable && layoutConfig.draggable,
		isResizable: isEditable && layoutConfig.resizable,
		onLayoutChange: handleLayoutChange,
		onDragStart: handleDragStart,
		onDragStop: handleDragStop,
		onResizeStart: handleResizeStart,
		onResizeStop: handleResizeStop,
		compactType: 'vertical',
		preventCollision: false,
		children,
	};

	return (
		<div ref={containerRef} className={`dashboard-grid ${isEditable ? 'dashboard-grid--editable' : ''}`}>
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
