/**
 * DashboardView - 仪表盘视图组件
 *
 * 集成实时数据订阅，支持：
 * - WebSocket 实时数据推送
 * - 轮询数据刷新
 * - 网络状态检测和提示
 */

import { useEffect, useMemo, useState, useCallback } from "react";
import { App, notification, Badge, Tooltip, Spin, Alert, Result, Button, Empty } from "antd";
import { WifiOutlined, DisconnectOutlined, ReloadOutlined, SyncOutlined, WarningOutlined, ReloadOutlined as RefreshIcon } from "@ant-design/icons";
import { useDashboardStore } from "@/store/dashboardStore";
import { useWidgetPolling } from "@/hooks/useWidgetPolling";
import { useNetworkStatus } from "@/hooks/useNetworkStatus";
import { useWebSocket } from "@/hooks/useWebSocket";
import { DashboardGrid } from "./layout/DashboardGrid";
import type { Dashboard } from "@/types/dashboard";

import "./DashboardView.css";
import "./DashboardView.css";

export interface DashboardViewProps {
	/** 仪表盘数据 */
	dashboard: Dashboard;
	/** 是否为只读模式 */
	readonly?: boolean;
	/** 加载状态 */
	loading?: boolean;
	/** 错误信息 */
	error?: string | null;
	/** 重试回调 */
	onRetry?: () => void;
}

/**
 * 仪表盘视图组件
 */
export const DashboardView: React.FC<DashboardViewProps> = ({
	dashboard,
	readonly = false,
	loading: externalLoading = false,
	error: externalError = null,
	onRetry,
}) => {
	const { message } = App.useApp();
	const {
		viewMode,
		widgetDataCache,
		setWsStatus,
		setIsRefreshing,
		updateWidgetData,
		wsStatus,
		isRefreshing,
	} = useDashboardStore();

	// 内部状态
	const [initialLoadComplete, setInitialLoadComplete] = useState(false);

	// 网络状态检测
	const { isOnline, wasOffline, resetWasOffline } = useNetworkStatus();

	// Widget ID 列表
	const widgetIds = useMemo(() => {
		return dashboard?.layout.widgets.map((w) => w.id) || [];
	}, [dashboard]);

	// 轮询数据刷新
	const { loading, refresh, isPaused, pause, resume } = useWidgetPolling({
		widgetIds,
		interval: dashboard?.refreshInterval || 300,
		enabled: isOnline && viewMode === "view" && !readonly,
	});

	// WebSocket 连接（检查是否有 WebSocket 数据源）
	const hasWebSocketSource = useMemo(() => {
		return dashboard?.layout.widgets.some((w) => {
			const ds = w.dataSource;
			return "websocket" in ds || (ds && "type" in ds && ds.type === "websocket");
		}) || false;
	}, [dashboard]);

	// WebSocket 消息类型守卫
	const isWidgetMessage = (data: unknown): data is { widgetId: string; data: unknown } => {
		return typeof data === "object" && data !== null &&
			"widgetId" in data && "data" in data &&
			typeof (data as Record<string, unknown>).widgetId === "string";
	};

	// WebSocket 消息处理
	const handleWebSocketMessage = (data: unknown) => {
		if (isWidgetMessage(data) && widgetIds.includes(data.widgetId)) {
			updateWidgetData(data.widgetId, data.data);
		}
	};

	// WebSocket URL（使用环境变量配置，支持部署到非默认路径）
	const wsUrl = useMemo(() => {
		const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
		const host = import.meta.env.VITE_WS_HOST || window.location.host;
		const basePath = import.meta.env.VITE_WS_BASE_PATH || "/ws";
		return `${protocol}//${host}${basePath}/dashboard/${dashboard?.id}`;
	}, [dashboard?.id]);

	// WebSocket 连接
	const { status: wsConnStatus, connect, disconnect } = useWebSocket({
		url: wsUrl,
		onMessage: handleWebSocketMessage,
		onOpen: () => {
			setWsStatus("connected");
		},
		onClose: () => {
			setWsStatus("disconnected");
		},
		onError: () => {
			setWsStatus("error");
		},
		reconnect: true,
	});

	// 根据是否有 WebSocket 数据源决定是否连接
	useEffect(() => {
		if (hasWebSocketSource && isOnline && viewMode === "view" && !readonly) {
			connect();
		}
		return () => {
			if (hasWebSocketSource) {
				disconnect();
			}
		};
	}, [hasWebSocketSource, isOnline, viewMode, readonly, connect, disconnect]);

	// 网络状态提示
	useEffect(() => {
		if (!isOnline) {
			message.warning({
				content: "网络已断开，数据将无法更新",
				key: "network-status",
				duration: 0,
			});
		} else if (wasOffline) {
			message.destroy("network-status");
			notification.success({
				message: "网络已恢复",
				description: "数据正在重新加载...",
				duration: 3,
			});
			resetWasOffline();
			// 网络恢复后刷新数据
			refresh();
		}
	}, [isOnline, wasOffline, resetWasOffline, refresh]);

	// 更新刷新状态
	useEffect(() => {
		setIsRefreshing(loading);
	}, [loading, setIsRefreshing]);

	// 首次加载完成
	useEffect(() => {
		if (!loading && !initialLoadComplete) {
			setInitialLoadComplete(true);
		}
	}, [loading, initialLoadComplete]);

	// WebSocket 错误提示
	useEffect(() => {
		if (wsConnStatus === "error") {
			notification.warning({
				message: "WebSocket 连接错误",
				description: "实时数据推送不可用，将使用轮询模式",
				duration: 5,
			});
		}
	}, [wsConnStatus]);

	// 渲染连接状态指示器
	const renderConnectionStatus = () => {
		if (!hasWebSocketSource) return null;

		const statusConfig = {
			connecting: { color: "processing", icon: <SyncOutlined spin />, text: "连接中" },
			connected: { color: "success", icon: <WifiOutlined />, text: "已连接" },
			disconnected: { color: "default", icon: <DisconnectOutlined />, text: "已断开" },
			error: { color: "error", icon: <DisconnectOutlined />, text: "连接错误" },
		};

		const config = statusConfig[wsConnStatus];

		return (
			<Tooltip title={`WebSocket ${config.text}`}>
				<Badge status={config.color as "processing" | "success" | "default" | "error"} />
			</Tooltip>
		);
	};

	// 布局变更处理
	const handleLayoutChange = useCallback((layouts: Array<{ id: string; position: unknown }>) => {
		// 布局变更处理
	}, []);

	if (!dashboard) {
		return (
			<div className="dashboard-view dashboard-view--empty">
				<Empty description="仪表盘不存在" />
			</div>
		);
	}

	// 全局错误状态
	if (externalError) {
		return (
			<div className="dashboard-view dashboard-view--error">
				<Result
					status="error"
					title="加载仪表盘失败"
					subTitle={externalError}
					extra={
						onRetry && (
							<Button type="primary" icon={<RefreshIcon />} onClick={onRetry}>
								重试
							</Button>
						)
					}
				/>
			</div>
		);
	}

	// 首次加载骨架屏
	if (!initialLoadComplete && externalLoading) {
		return (
			<div className="dashboard-view dashboard-view--loading">
				<Spin size="large">
					<div style={{ minHeight: 120 }} />
				</Spin>
				<div style={{ marginTop: 8, color: "rgba(0, 0, 0, 0.45)" }}>加载仪表盘中...</div>
			</div>
		);
	}

	return (
		<div className="dashboard-view">
			{/* 离线提示条 */}
			{!isOnline && (
				<Alert
					title="当前处于离线模式"
					description="数据可能不是最新，网络恢复后将自动刷新"
					type="warning"
					icon={<WarningOutlined />}
					showIcon
					closable
					style={{ marginBottom: 16 }}
				/>
			)}

			{/* WebSocket 断开提示 */}
			{hasWebSocketSource && wsConnStatus === "disconnected" && isOnline && (
				<Alert
					title="实时连接已断开"
					description="正在尝试重新连接..."
					type="info"
					showIcon
					style={{ marginBottom: 16 }}
				/>
			)}

			{/* 工具栏 */}
			<div className="dashboard-view__toolbar">
				<div className="dashboard-view__title">
					<h2>{dashboard.name}</h2>
					{dashboard.description && (
						<p>{dashboard.description}</p>
					)}
				</div>
				<div className="dashboard-view__status">
					{renderConnectionStatus()}
					{isPaused && (
						<Tooltip title="轮询已暂停">
							<Badge status="warning" text="已暂停" />
						</Tooltip>
					)}
					{!isOnline && (
						<Tooltip title="离线模式">
							<Badge status="error" text="离线" />
						</Tooltip>
					)}
				</div>
			</div>

			{/* 仪表盘网格 */}
			<DashboardGrid
				widgets={dashboard.layout.widgets}
				onLayoutChange={handleLayoutChange}
			>
				{dashboard.layout.widgets.map((widget) => (
					<div key={widget.id} data-widget-id={widget.id}>
						{widget.title}
					</div>
				))}
			</DashboardGrid>
		</div>
	);
};

export default DashboardView;
