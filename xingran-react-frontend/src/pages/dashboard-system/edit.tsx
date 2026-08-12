/**
 * 仪表盘编辑页
 *
 * 编辑仪表盘的Widget布局和配置
 */

import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { App, Spin } from "antd";
import type { WidgetConfig } from "@/types";
import {
	LayoutToolbar,
	DashboardGrid,
	DashboardGridPlaceholder,
} from "@/components/dashboard/layout";
import { GridItem, GridItemPlaceholder } from "@/components/dashboard/layout";
import { Suspense, lazy } from "react";
import { useDashboardStore } from "@/store/dashboardStore";
import { Widget } from "@/components/dashboard/Widget";

import "./edit.css";

const WidgetSelector = lazy(() => import("@/components/dashboard/settings/WidgetSelector"));

const DashboardEdit: React.FC = () => {
	const { message } = App.useApp();
	const { id } = useParams<{ id: string }>();
	const [showWidgetSelector, setShowWidgetSelector] = useState(false);
	const [editingWidgetId, setEditingWidgetId] = useState<string | null>(null);

	const {
		currentDashboard,
		currentLoading,
		fetchDashboard,
		setViewMode,
		updateWidgetLayouts,
		addWidget,
		selectWidget,
		selectedWidgetId,
		clearCurrentDashboard,
	} = useDashboardStore();

	// 加载仪表盘
	useEffect(() => {
		if (id) {
			fetchDashboard(id);
			setViewMode("edit");
		}

		return () => {
			clearCurrentDashboard();
		};
	}, [id, fetchDashboard, setViewMode, clearCurrentDashboard]);

	// 处理Widget添加
	const handleAddWidget = (widgetConfig: WidgetConfig) => {
		addWidget(widgetConfig);
		message.success("Widget添加成功");
	};

	if (currentLoading) {
		return (
			<div className="dashboard-edit loading">
				<Spin size="large" />
			</div>
		);
	}

	if (!currentDashboard) {
		return (
			<div className="dashboard-edit empty">
				<p>仪表盘不存在</p>
			</div>
		);
	}

	const widgets = currentDashboard.layout.widgets;

	return (
		<div className="dashboard-edit">
			<LayoutToolbar
				dashboardId={id}
				showBackButton={true}
				onAddWidget={() => setShowWidgetSelector(true)}
			/>

			<div className="dashboard-edit__content">
				{widgets.length === 0 ? (
					<DashboardGridPlaceholder message="点击'添加Widget'开始构建仪表盘" />
				) : (
					<Suspense fallback={<Spin />}>
						<DashboardGrid
							widgets={widgets}
							onLayoutChange={updateWidgetLayouts}
						>
							{widgets.map((widget) => (
								<div key={widget.id}>
									<GridItem
										widget={widget}
										selected={selectedWidgetId === widget.id}
										onClick={() => selectWidget(selectedWidgetId === widget.id ? null : widget.id)}
									>
										<Widget widget={widget} />
									</GridItem>
								</div>
							))}
							{/* 添加Widget占位符 */}
							<div key="add-widget-placeholder">
								<GridItemPlaceholder
									widgetId="add-widget"
									onClick={() => setShowWidgetSelector(true)}
								/>
							</div>
						</DashboardGrid>
					</Suspense>
				)}
			</div>

			<Suspense fallback={null}>
				<WidgetSelector
					visible={showWidgetSelector}
					onClose={() => {
						setShowWidgetSelector(false);
						setEditingWidgetId(null);
					}}
					onSelect={handleAddWidget}
					editingWidgetId={editingWidgetId}
				/>
			</Suspense>
		</div>
	);
};

export default DashboardEdit;
