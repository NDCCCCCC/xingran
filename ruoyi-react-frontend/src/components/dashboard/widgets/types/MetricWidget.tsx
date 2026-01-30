/**
 * MetricWidget - 指标 Widget
 *
 * 圆形指标仪表盘（类似ProgressWidget的circle类型）
 */

import { useMemo } from 'react';
import { Progress } from 'antd';
import type { ProgressDisplayConfig, WidgetConfig } from '@/types/dashboard';
import { BaseWidget } from '../base/BaseWidget';
import type { BaseWidgetProps } from '../base/BaseWidget';
import { useWidgetData } from '@/hooks/useWidgetData';

interface MetricWidgetProps {
	widget: WidgetConfig;
	display: ProgressDisplayConfig;
	onEdit?: () => void;
	onDelete?: () => void;
}

export const MetricWidget: React.FC<MetricWidgetProps> = ({
	widget,
	display,
	onEdit,
	onDelete,
}) => {
	// 使用useWidgetData直接获取数据
	const { data, loading, error, refresh } = useWidgetData(widget);
	const { percent, color } = useMemo(() => {
		if (!data || typeof data !== 'object') {
			return { percent: 0, color: undefined };
		}

		const d = data as Record<string, unknown>;
		const value = Number(d.value ?? d.percent ?? 0);
		const target = display.target ?? 100;
		const percent = Math.min(Math.round((value / target) * 100), 100);

		let strokeColor = undefined;
		if (display.colorThresholds) {
			const sorted = [...display.colorThresholds].sort((a, b) => b.value - a.value);
			for (const threshold of sorted) {
				if (percent >= threshold.value) {
					strokeColor = threshold.color;
					break;
				}
			}
		}

		return { percent, color: strokeColor };
	}, [data, display]);

	return (
		<BaseWidget
			widget={widget}
			data={data}
			loading={loading}
			error={error}
			onEdit={onEdit}
			onDelete={onDelete}
			onRefresh={refresh}
		>
			<div className="metric-widget">
				<Progress
					type="circle"
					percent={percent}
					strokeColor={color}
					width={100}
					format={(percent) => (
						<span className="metric-widget-value">{percent}%</span>
					)}
				/>
			</div>
		</BaseWidget>
	);
};
