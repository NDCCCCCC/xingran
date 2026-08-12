/**
 * ProgressWidget - 进度条 Widget
 *
 * 以进度条形式展示百分比或完成度
 */

import { useMemo } from "react";
import { Progress, Row, Col } from "antd";
import type { ProgressDisplayConfig, WidgetConfig } from "@/types/dashboard";
import { BaseWidget } from "../base/BaseWidget";
import type { BaseWidgetProps } from "../base/BaseWidget";
import { useWidgetData } from "@/hooks/useWidgetData";

interface ProgressWidgetProps {
	widget: WidgetConfig;
	display: ProgressDisplayConfig;
	onEdit?: () => void;
	onDelete?: () => void;
}

export const ProgressWidget: React.FC<ProgressWidgetProps> = ({
	widget,
	display,
	onEdit,
	onDelete,
}) => {
	// 使用useWidgetData直接获取数据
	const { data, loading, error, refresh } = useWidgetData(widget);
	// 计算进度百分比和颜色
	const { percent, color, status } = useMemo(() => {
		if (!data || typeof data !== "object") {
			return { percent: 0, color: undefined, status: undefined as "success" | "exception" | "normal" | undefined };
		}

		const d = data as Record<string, unknown>;
		const value = Number(d.value ?? d.percent ?? d.progress ?? 0);
		const target = display.target ?? 100;
		const percent = Math.min(Math.round((value / target) * 100), 100);

		// 根据阈值确定颜色
		let color = undefined;
		let status: "success" | "exception" | "normal" | undefined = undefined;

		if (display.colorThresholds && display.colorThresholds.length > 0) {
			const sorted = [...display.colorThresholds].sort((a, b) => b.value - a.value);
			for (const threshold of sorted) {
				if (percent >= threshold.value) {
					color = threshold.color;
					break;
				}
			}
		}

		// 根据百分比确定状态
		if (percent >= 100) status = "success";
		else if (percent < 30) status = "exception";

		return { percent, color, status };
	}, [data, display]);

	// 多指标进度条
	const multiple = useMemo(() => {
		if (!data || typeof data !== "object") return [];
		const d = data as Record<string, unknown>;
		const items = d.items ?? d.metrics ?? d.data;
		if (!Array.isArray(items)) return [];

		return items.map((item, index) => {
			const i = item as Record<string, unknown>;
			return {
				label: i.label ?? i.name ?? `指标${index + 1}`,
				percent: Math.min(Math.round((Number(i.value ?? 0) / (Number(i.target ?? 100) || 1)) * 100), 100),
				color: i.color as string | undefined,
			};
		});
	}, [data]);

	// 如果有多指标，显示多指标进度条
	if (multiple.length > 0) {
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
				<div className="progress-widget progress-widget--multiple">
					<Row gutter={[8, 16]}>
						{multiple.map((item, index) => (
							<Col span={24 / multiple.length} key={index}>
								<div className="progress-widget-item">
									<div className="progress-widget-item__label">{String(item.label ?? "")}</div>
									<Progress
										percent={item.percent}
										strokeColor={item.color}
										size="small"
									/>
								</div>
							</Col>
						))}
					</Row>
				</div>
			</BaseWidget>
		);
	}

	// 单指标进度条
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
			<div className="progress-widget">
				{display.progressType === "circle" ? (
					<Progress
						type="circle"
						percent={percent}
						strokeColor={color}
						status={status}
						width={120}
					/>
				) : display.progressType === "dashboard" ? (
					<Progress
						type="dashboard"
						percent={percent}
						strokeColor={color}
						status={status}
						gapDegree={120}
					/>
				) : (
					<Progress
						percent={percent}
						strokeColor={color}
						status={status}
					/>
				)}
			</div>
		</BaseWidget>
	);
};
