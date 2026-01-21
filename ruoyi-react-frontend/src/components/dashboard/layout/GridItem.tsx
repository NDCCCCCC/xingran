/**
 * GridItem - 网格项包装器
 *
 * 包装每个Widget，提供额外的样式和交互功能
 */

import type { ReactNode } from 'react';
import { useDashboardStore } from '@/store/dashboardStore';
import type { WidgetConfig } from '@/types/dashboard';
import './GridItem.css';

interface GridItemProps {
	/** Widget 配置 */
	widget: WidgetConfig;

	/** 是否选中 */
	selected?: boolean;

	/** 点击事件 */
	onClick?: () => void;

	/** 子元素 */
	children: ReactNode;
}

export const GridItem: React.FC<GridItemProps> = ({
	widget,
	selected = false,
	onClick,
	children,
}) => {
	const { viewMode, selectedWidgetId } = useDashboardStore();

	// 判断是否选中
	const isSelected = selected || selectedWidgetId === widget.id;

	// 判断是否可编辑
	const isEditable = viewMode === 'edit';

	// 点击处理
	const handleClick = () => {
		if (isEditable && onClick) {
			onClick();
		}
	};

	return (
		<div
			className={`grid-item ${isSelected ? 'grid-item--selected' : ''} ${isEditable ? 'grid-item--editable' : ''}`}
			onClick={handleClick}
			data-widget-id={widget.id}
		>
			<div className="grid-item__content">
				{children}
			</div>
		</div>
	);
};

/**
 * GridItemPlaceholder - 空Widget占位符
 */
export const GridItemPlaceholder: React.FC<{
	widgetId: string;
	onClick: () => void;
}> = ({ widgetId, onClick }) => {
	return (
		<div
			className="grid-item-placeholder"
			onClick={onClick}
			data-widget-id={widgetId}
		>
			<div className="grid-item-placeholder__content">
				<div className="grid-item-placeholder__icon">➕</div>
				<span className="grid-item-placeholder__text">添加 Widget</span>
			</div>
		</div>
	);
};
