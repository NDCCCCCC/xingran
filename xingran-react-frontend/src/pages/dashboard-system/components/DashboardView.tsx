/**
 * 仪表盘查看视图组件
 *
 * 以只读模式查看仪表盘
 */

import { useEffect } from "react";
import { LayoutToolbar } from "@/components/dashboard/layout/LayoutToolbar";
import { DashboardGrid, DashboardGridPlaceholder } from "@/components/dashboard/layout/DashboardGrid";
import { GridItem } from "@/components/dashboard/layout/GridItem";
import { Suspense } from "react";
import { useDashboardStore } from "@/store/dashboardStore";
import { Spin, Space, Button } from "antd";
import { useNavigate } from "react-router-dom";
import { UnorderedListOutlined, EditOutlined } from "@ant-design/icons";
import { DASHBOARD } from "@/constants/routes";
import { Widget } from "@/components/dashboard/Widget";

interface DashboardViewProps {
	dashboardId: string;
	isHome?: boolean; // 是否在首页显示（控制工具栏）
}

export const DashboardView: React.FC<DashboardViewProps> = ({ dashboardId, isHome = false }) => {
	const navigate = useNavigate();
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
		if (dashboardId) {
			fetchDashboard(dashboardId);
			setViewMode("view");
		}

		return () => {
			clearCurrentDashboard();
		};
	}, [dashboardId, fetchDashboard, setViewMode, clearCurrentDashboard]);

	if (currentLoading) {
		return (
			<div className="dashboard-view loading" style={{ display: "flex", justifyContent: "center", alignItems: "center", height: "400px" }}>
				<Spin size="large" />
			</div>
		);
	}

	if (!currentDashboard) {
		return (
			<div className="dashboard-view empty" style={{ display: "flex", justifyContent: "center", alignItems: "center", height: "400px" }}>
				<p>仪表盘不存在</p>
			</div>
		);
	}

	const widgets = currentDashboard.layout.widgets;

	// 顶部操作栏
	// - 首页模式：显示简化的操作栏（列表按钮 + 编辑按钮）
	// - 非首页模式：显示完整的工具栏（列表按钮 + 返回按钮 + 完整操作）
	const headerActions = (
		<Space style={{ marginBottom: isHome ? 16 : 0 }}>
			<Button icon={<UnorderedListOutlined />} onClick={() => navigate(`${DASHBOARD}?mode=list`)}>
				仪表盘列表
			</Button>
			{isHome ? (
				// 首页模式：只显示编辑按钮
				<Button type="primary" icon={<EditOutlined />} onClick={() => navigate(`${DASHBOARD}/${dashboardId}?mode=edit`)}>
					编辑仪表盘
				</Button>
			) : (
				// 非首页模式：显示完整的工具栏
				<LayoutToolbar dashboardId={dashboardId} showBackButton={true} />
			)}
		</Space>
	);

	return (
		<div className="dashboard-view">
			{headerActions}

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
