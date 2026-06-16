/**
 * 仪表盘查看页
 *
 * 以只读模式查看仪表盘
 */

import { useEffect } from "react";
import { useParams } from "react-router-dom";
import { LayoutToolbar } from "@/components/dashboard/layout/LayoutToolbar";
import { DashboardGrid, DashboardGridPlaceholder } from "@/components/dashboard/layout/DashboardGrid";
import { GridItem } from "@/components/dashboard/layout/GridItem";
import { Suspense } from "react";
import { useDashboardStore } from "@/store/dashboardStore";
import { Spin } from "antd";
import { Widget } from "@/components/dashboard/Widget";

import "./view.css";

const DashboardView: React.FC = () => {
	const { id } = useParams<{ id: string }>();
	const {
		currentDashboard,
		currentLoading,
		fetchDashboard,
		setViewMode,
		updateWidgetLayouts,
		clearCurrentDashboard,
	} = useDashboardStore();

	// 加载仪表盘
	useEffect(() => {
		if (id) {
			fetchDashboard(id);
			setViewMode("view");
		}

		return () => {
			clearCurrentDashboard();
		};
	}, [id, fetchDashboard, setViewMode, clearCurrentDashboard]);

	if (currentLoading) {
		return (
			<div className="dashboard-view loading">
				<Spin size="large" />
			</div>
		);
	}

	if (!currentDashboard) {
		return (
			<div className="dashboard-view empty">
				<p>仪表盘不存在</p>
			</div>
		);
	}

	const widgets = currentDashboard.layout.widgets;

	return (
		<div className="dashboard-view">
			<LayoutToolbar
				dashboardId={id}
				showBackButton={true}
			/>

			<div className="dashboard-view__content">
				{widgets.length === 0 ? (
					<DashboardGridPlaceholder message="此仪表盘暂无Widget" />
				) : (
					<Suspense fallback={<Spin />}>
						<DashboardGrid
							widgets={widgets}
							onLayoutChange={updateWidgetLayouts}
						>
							{widgets.map((widget) => (
								<div key={widget.id}>
									<GridItem widget={widget}>
										<Widget widget={widget} />
									</GridItem>
								</div>
							))}
						</DashboardGrid>
					</Suspense>
				)}
			</div>
		</div>
	);
};

export default DashboardView;
