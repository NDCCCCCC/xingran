/**
 * ListWidget - 列表 Widget
 *
 * 以简洁的列表形式展示数据
 */

import { useMemo } from "react";
import { List } from "antd";
import { ClockCircleOutlined } from "@ant-design/icons";
import type { ListDisplayConfig, WidgetConfig } from "@/types/dashboard";
import { BaseWidget } from "../base/BaseWidget";
import type { BaseWidgetProps } from "../base/BaseWidget";
import { useWidgetData } from "@/hooks/useWidgetData";

interface ListWidgetProps {
	widget: WidgetConfig;
	display: ListDisplayConfig;
	onEdit?: () => void;
	onDelete?: () => void;
}

export const ListWidget: React.FC<ListWidgetProps> = ({
	widget,
	display,
	onEdit,
	onDelete,
}) => {
	// 使用useWidgetData直接获取数据
	const { data, loading, error, refresh } = useWidgetData(widget);
	// 提取列表数据
	const listData = useMemo(() => {
		if (!data || typeof data !== "object") return [];
		const d = data as Record<string, unknown>;
		const items = extractArray(d.list ?? d.data ?? d.items ?? []);

		// 限制显示数量
		const maxItems = display.maxItems ?? 10;
		return items.slice(0, maxItems);
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
			<List
				size="small"
				dataSource={listData}
				renderItem={(item, index) => (
					<List.Item key={String((item as Record<string, unknown>).id ?? index)}>
						<div className="list-widget-item">
							{display.showIndex && (
								<span className="list-widget-item__index">{index + 1}.</span>
							)}
							{display.iconField && (
								<span className="list-widget-item__icon">
									{(item as Record<string, unknown>)[display.iconField] as string}
								</span>
							)}
							<div className="list-widget-item__content">
								<div className="list-widget-item__title">
									{(item as Record<string, unknown>)[display.titleField] as string}
								</div>
								{display.descriptionField && (
									<div className="list-widget-item__description">
										{(item as Record<string, unknown>)[display.descriptionField] as string}
									</div>
								)}
							</div>
							{display.timeField && (
								<div className="list-widget-item__time">
									<ClockCircleOutlined />
									<span>{formatTime((item as Record<string, unknown>)[display.timeField] as string)}</span>
								</div>
							)}
						</div>
					</List.Item>
				)}
			/>
		</BaseWidget>
	);
};

// 提取数组数据
function extractArray(val: unknown): Record<string, unknown>[] {
	if (Array.isArray(val)) return val as Record<string, unknown>[];
	if (val !== null && val !== undefined) return [val as Record<string, unknown>];
	return [];
}

// 格式化时间
function formatTime(timeStr: string): string {
	const date = new Date(timeStr);
	const now = new Date();
	const diff = now.getTime() - date.getTime();
	const minutes = Math.floor(diff / 60000);
	const hours = Math.floor(minutes / 60);
	const days = Math.floor(hours / 24);

	if (days > 0) return `${days}天前`;
	if (hours > 0) return `${hours}小时前`;
	if (minutes > 0) return `${minutes}分钟前`;
	return "刚刚";
}
