/**
 * GridItem - 网格项包装器
 *
 * 包装每个Widget，提供额外的样式和交互功能
 */

import { useState, lazy, Suspense } from "react";
import { Space, Button } from "antd";
import { EditOutlined, DeleteOutlined } from "@ant-design/icons";
import { useDashboardStore } from "@/store/dashboardStore";
import type { WidgetConfig } from "@/types/dashboard";
import "./GridItem.css";

// 懒加载 WidgetSelector 组件
const WidgetSelector = lazy(() => import("@/components/dashboard/settings/WidgetSelector"));

interface GridItemProps {
	/** Widget 配置 */
	widget: WidgetConfig;

	/** 是否选中 */
	selected?: boolean;

	/** 点击事件 */
	onClick?: () => void;

	/** 子元素 */
	children: React.ReactNode;
}

export const GridItem: React.FC<GridItemProps> = ({
	widget,
	selected = false,
	onClick,
	children,
}) => {
	const { viewMode, selectedWidgetId, removeWidget, updateWidget } = useDashboardStore();
	const [showEditModal, setShowEditModal] = useState(false);

	// 判断是否选中
	const isSelected = selected || selectedWidgetId === widget.id;

	// 判断是否可编辑
	const isEditable = viewMode === "edit";

	// 点击处理
	const handleClick = () => {
		if (isEditable && onClick) {
			onClick();
		}
	};

	// 删除 widget
	const handleRemove = (e: React.MouseEvent) => {
		e.stopPropagation();
		removeWidget(widget.id);
	};

	// 编辑 widget
	const handleEdit = (e: React.MouseEvent) => {
		e.stopPropagation();
		setShowEditModal(true);
	};

	// 更新 widget
	const handleUpdateWidget = (updatedWidget: WidgetConfig) => {
		updateWidget(widget.id, updatedWidget);
		setShowEditModal(false);
	};

	return (
		<>
			<div
				className={`grid-item ${isSelected ? "grid-item--selected" : ""} ${isEditable ? "grid-item--editable" : ""}`}
				onClick={handleClick}
				data-widget-id={widget.id}
			>
				{/* 编辑/删除按钮（仅选中时显示） */}
				{isEditable && isSelected && (
					<div className="grid-item__actions">
						<Space size="small">
							<Button
								type="primary"
								size="small"
								icon={<EditOutlined />}
								onClick={handleEdit}
							>
								编辑
							</Button>
							<Button
								danger
								size="small"
								icon={<DeleteOutlined />}
								onClick={handleRemove}
							>
								删除
							</Button>
						</Space>
					</div>
				)}

				<div className="grid-item__content">
					{children}
				</div>
			</div>

			{/* Widget 编辑对话框 */}
			{showEditModal && (
				<Suspense fallback={null}>
					<WidgetSelector
						visible={showEditModal}
						onClose={() => setShowEditModal(false)}
						onSelect={handleUpdateWidget}
						editingWidgetId={widget.id}
						editingWidget={widget}
					/>
				</Suspense>
			)}
		</>
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
