/**
 * WidgetSelector - Widget 选择器
 *
 * 用于选择和添加Widget到仪表盘
 * 集成API端点元数据系统，支持友好的端点选择
 * 支持完整的数据源配置（API/WebSocket/Static）和显示配置
 */

import { useState, useEffect } from "react";
import { Modal, Card, Col, Row, Input, Button, Form, Select, Divider } from "antd";
import { widgetRegistry, getWidgetTypes } from "../widgets/configs/widgetRegistry";

import DataSourceForm from "./DataSourceForm";
import DisplayConfigForm from "./DisplayConfigForm";
import type { WidgetType, WidgetConfig, EndpointDetail, DataSourceConfig, DisplayConfig } from "@/types/dashboard";
import { widgetDefaultSizes } from "@/types/dashboard";

interface WidgetSelectorProps {
	visible: boolean;
	onClose: () => void;
	onSelect: (widget: WidgetConfig) => void;
	editingWidgetId?: string | null;
	editingWidget?: WidgetConfig | null;
}

const { TextArea } = Input;

export const WidgetSelector: React.FC<WidgetSelectorProps> = ({
	visible,
	onClose,
	onSelect,
	editingWidgetId,
	editingWidget,
}) => {
	const [selectedType, setSelectedType] = useState<WidgetType | null>(null);
	const [selectedEndpoint, setSelectedEndpoint] = useState<EndpointDetail | null>(null);
	const [dataSource, setDataSource] = useState<DataSourceConfig | undefined>(undefined);
	const [displayConfig, setDisplayConfig] = useState<DisplayConfig | undefined>(undefined);
	const [form] = Form.useForm();

	const widgetTypes = getWidgetTypes();

	// 如果是编辑模式，初始化表单数据
	useEffect(() => {
		if (editingWidget && visible) {
			setSelectedType(editingWidget.type);
			setDataSource(editingWidget.dataSource);
			setDisplayConfig(editingWidget.display);
			form.setFieldsValue({
				type: editingWidget.type,
				title: editingWidget.title,
				description: editingWidget.description || "",
			});
		} else if (!visible) {
			// 关闭时重置状态
			setSelectedType(null);
			setSelectedEndpoint(null);
			setDataSource(undefined);
			setDisplayConfig(undefined);
			form.resetFields();
		}
	}, [editingWidget, visible, form]);

	// 选择Widget类型
	const handleSelectType = (type: WidgetType) => {
		setSelectedType(type);
		form.setFieldsValue({
			type,
			title: widgetRegistry[type]?.displayName || "",
		});
	};

	// 端点选择变化
	const handleEndpointChange = (route: string, endpoint: EndpointDetail) => {
		setSelectedEndpoint(endpoint);
		// 自动填充请求方法和数据路径
		form.setFieldsValue({
			endpoint: route,
			method: endpoint.method,
			dataPath: endpoint.dataPath,
		});
	};

	// 确认添加
	const handleConfirm = async () => {
		try {
			const values = await form.validateFields();

			// 使用数据源配置（如果已设置）或创建默认配置
			const finalDataSource: DataSourceConfig = dataSource || {
				api: {
					type: "api",
					endpoint: values.endpoint || "/api/default",
					method: (values.method || "GET") as "GET" | "POST",
				},
			};

			// 使用显示配置（如果已设置）或创建默认配置
			const createDefaultDisplayConfig = (widgetType: WidgetType): DisplayConfig => {
				switch (widgetType) {
					case "stat-card":
						return {
							type: "stat-card",
							icon: "📊",
							iconColor: "var(--theme-info, #1890ff)",
							decimals: 0,
						};
					case "chart":
						return {
							type: "chart",
							chartType: "line",
							showLegend: true,
						};
					case "table":
						return {
							type: "table",
							columns: [],
							bordered: true,
						};
					case "list":
						return {
							type: "list",
							titleField: "title",
							maxItems: 10,
						};
					case "progress":
						return {
							type: "progress",
							progressType: "line",
							target: 100,
						};
					default:
						return {
							type: "stat-card",
							icon: "📊",
							iconColor: "var(--theme-info, #1890ff)",
						};
				}
			};

			const finalDisplayConfig: DisplayConfig = displayConfig || createDefaultDisplayConfig(values.type as WidgetType);

			const widget: WidgetConfig = {
				id: editingWidgetId || `widget-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
				type: values.type,
				title: values.title,
				position: editingWidget?.position || {
					x: 0,
					y: 0,
					...widgetDefaultSizes[values.type as WidgetType],
				},
				dataSource: finalDataSource,
				display: finalDisplayConfig,
				enabled: true,
			};

			// 先触发回调（添加 Widget）
			onSelect(widget);

			// 延迟关闭弹窗，确保回调执行完成
			setTimeout(() => {
				setSelectedType(null);
				setSelectedEndpoint(null);
				setDataSource(undefined);
				setDisplayConfig(undefined);
				form.resetFields();
				onClose();
			}, 100);
		} catch (error) {
			console.error("Validation failed:", error);
		}
	};

	// 取消
	const handleCancel = () => {
		setSelectedType(null);
		setSelectedEndpoint(null);
		form.resetFields();
		onClose();
	};

	return (
		<Modal
			title={editingWidgetId ? "编辑 Widget" : "添加 Widget"}
			open={visible}
			onOk={handleConfirm}
			onCancel={handleCancel}
			okText="确定"
			cancelText="取消"
			width={800}
			destroyOnHidden={true}
		>
			<div className="widget-selector">
				{!selectedType ? (
					<div className="widget-selector__types">
						<Row gutter={[16, 16]}>
							{widgetTypes.map((type: WidgetType) => {
								const config = widgetRegistry[type];
								return (
									<Col xs={12} sm={8} md={6} key={type}>
										<Card
											hoverable
											onClick={() => handleSelectType(type)}
											className="widget-selector-card"
										>
											<div className="widget-selector-card__icon">{config?.icon}</div>
											<div className="widget-selector-card__name">{config?.displayName}</div>
											<div className="widget-selector-card__desc">{config?.description}</div>
										</Card>
									</Col>
								);
							})}
						</Row>
					</div>
				) : (
					<div className="widget-selector__config">
						<Button onClick={() => setSelectedType(null)} style={{ marginBottom: 16 }}>
							← 返回选择Widget类型
						</Button>

				<Form form={form} layout="vertical" initialValues={{ method: "GET" }}>
						<Form.Item name="type" hidden>
							<input />
						</Form.Item>

						<Form.Item
							label="Widget类型"
							name="type"
							rules={[{ required: true }]}
						>
							<Select disabled onSearch={() => {}}>
								{widgetTypes.map((type: WidgetType) => (
									<Select.Option key={type} value={type}>
										{widgetRegistry[type]?.displayName}
									</Select.Option>
								))}
							</Select>
						</Form.Item>

						<Form.Item
							label="标题"
							name="title"
							rules={[{ required: true, message: "请输入Widget标题" }]}
						>
							<Input placeholder="请输入Widget标题" />
						</Form.Item>

						<Divider titlePlacement="left">数据源配置</Divider>

						<Form.Item>
							<DataSourceForm
								value={dataSource}
								onChange={setDataSource}
								widgetType={selectedType || undefined}
								form={form}
							/>
						</Form.Item>

						<Divider titlePlacement="left">显示配置</Divider>

						<Form.Item>
							<DisplayConfigForm
								value={displayConfig}
								onChange={setDisplayConfig}
								widgetType={selectedType!}
								form={form}
							/>
						</Form.Item>

						<Divider titlePlacement="left">其他</Divider>

						<Form.Item
							label="说明"
							name="description"
						>
							<TextArea rows={2} placeholder="可选的Widget说明" />
						</Form.Item>
					</Form>
					</div>
				)}
			</div>
		</Modal>
	);
};

export default WidgetSelector;
