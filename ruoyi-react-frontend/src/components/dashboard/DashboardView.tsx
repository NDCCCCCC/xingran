/**
 * DashboardView - 仪表盘视图组件
 *
 * 集成实时数据订阅，支持：
 * - WebSocket 实时数据推送
 * - 轮询数据刷新
 * - 网络状态检测和提示
 */

import { useEffect, useMemo } from 'react';
import { message, notification, Badge, Tooltip, Spin } from 'antd';
import { WifiOutlined, DisconnectOutlined, ReloadOutlined, SyncOutlined } from '@ant-design/icons';
import { useDashboardStore } from '@/store/dashboardStore';
import { useWidgetPolling } from '@/hooks/useWidgetPolling';
import { useNetworkStatus } from '@/hooks/useNetworkStatus';
import { useWebSocket } from '@/hooks/useWebSocket';
import { DashboardGrid } from './layout/DashboardGrid';
import type { Dashboard } from '@/types/dashboard';

export interface DashboardViewProps {
	/** 仪表盘数据 */
	dashboard: Dashboard;
	/** 是否为只读模式 */
	readonly?: boolean;
}

/**
 * 仪表盘视图组件
 */
export const DashboardView: React.FC<DashboardViewProps> = ({
	dashboard,
	readonly = false,
}) => {
	const {
		viewMode,
		widgetDataCache,
		setWsStatus,
		setIsRefreshing,
		updateWidgetData,
	} = useDashboardStore();

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
		enabled: isOnline && viewMode === 'view' && !readonly,
	});

	// WebSocket 连接（检查是否有 WebSocket 数据源）
	const hasWebSocketSource = useMemo(() => {
		return dashboard?.layout.widgets.some((w) => {
			const ds = w.dataSource;
			return 'websocket' in ds || (ds && 'type' in ds && ds.type === 'websocket');
		}) || false;
	}, [dashboard]);

	// WebSocket 消息处理
	const handleWebSocketMessage = (data: unknown) => {
		const msg = data as { widgetId?: string; data?: unknown };
		if (msg.widgetId && msg.data) {
			// 验证 widgetId 属于当前仪表盘
			if (widgetIds.includes(msg.widgetId)) {
				updateWidgetData(msg.widgetId, msg.data);
			}
		}
	};

	// WebSocket 连接
	const { status: wsStatus, connect, disconnect } = useWebSocket({
		url: `/ws/dashboard/${dashboard?.id}`,
		onMessage: handleWebSocketMessage,
		onOpen: () => {
			setWsStatus('connected');
		},
		onClose: () => {
			setWsStatus('disconnected');
		},
		onError: () => {
			setWsStatus('error');
		},
		reconnect: true,
	});

	// 根据是否有 WebSocket 数据源决定是否连接
	useEffect(() => {
		if (hasWebSocketSource && isOnline && viewMode === 'view' && !readonly) {
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
				content: '网络已断开，数据将无法更新',
				key: 'network-status',
				duration: 0,
			});
		} else if (wasOffline) {
			message.destroy('network-status');
			notification.success({
				message: '网络已恢复',
				description: '数据正在重新加载...',
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

	// 从缓存获取 Widget 数据
	const getWidgetData = (widgetId: string) => {
		const cached = widgetDataCache.get(widgetId);
		return cached?.data;
	};

	// 渲染连接状态指示器
	const renderConnectionStatus = () => {
		if (!hasWebSocketSource) return null;

		const statusConfig = {
			connecting: { color: 'processing', icon: <SyncOutlined spin />, text: '连接中' },
			connected: { color: 'success', icon: <WifiOutlined />, text: '已连接' },
			disconnected: { color: 'default', icon: <DisconnectOutlined />, text: '已断开' },
			error: { color: 'error', icon: <DisconnectOutlined />, text: '连接错误' },
		};

		const config = statusConfig[wsStatus];

		return (
			<Tooltip title={`WebSocket ${config.text}`}>
				<Badge status={config.color as 'processing' | 'success' | 'default' | 'error'} />
			</Tooltip>
		);
	};

	if (!dashboard) {
		return null;
	}

	return (
		<Spin spinning={loading} tip="加载数据中...">
			<div className="dashboard-view">
				{/* 工具栏 */}
				<div className="dashboard-view__toolbar" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '8px 16px' }}>
					<div className="dashboard-view__title">
						<h2 style={{ margin: 0 }}>{dashboard.name}</h2>
						{dashboard.description && (
							<p style={{ margin: 0, color: '#666', fontSize: 12 }}>{dashboard.description}</p>
						)}
					</div>
					<div className="dashboard-view__status" style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
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
					dashboard={dashboard}
					widgetData={widgetDataCache}
					readonly={readonly}
				/>
			</div>
		</Spin>
	);
};

export default DashboardView;
