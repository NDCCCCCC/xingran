/**
 * WidgetSelector - Widget 选择器
 *
 * 用于选择和添加Widget到仪表盘
 * 集成API端点元数据系统，支持友好的端点选择
 */

import { useState, useCallback } from 'react';
import { Modal, Card, Col, Row, Input, Button, Form, Select, Divider } from 'antd';
import { widgetRegistry, getWidgetTypes } from '../widgets/configs/widgetRegistry';
import EndpointSelector from './EndpointSelector';
import ParamsEditor from './ParamsEditor';
import type { WidgetType, WidgetConfig, ApiDataSourceConfig, EndpointDetail, StatCardDisplayConfig, ChartDisplayConfig, TableDisplayConfig, ListDisplayConfig, ProgressDisplayConfig } from '@/types/dashboard';
import { widgetDefaultSizes } from '@/types/dashboard';

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
	const [form] = Form.useForm();

	const widgetTypes = getWidgetTypes();

	// 如果是编辑模式，初始化表单数据
	useCallback(() => {
		if (editingWidget && visible) {
			const dataSource = editingWidget.dataSource as ApiDataSourceConfig;
			setSelectedType(editingWidget.type);
			form.setFieldsValue({
				type: editingWidget.type,
				title: editingWidget.title,
				endpoint: dataSource.endpoint,
				method: dataSource.method,
			});
		}
	}, [editingWidget, visible, form]);

	// 选择Widget类型
	const handleSelectType = (type: WidgetType) => {
		setSelectedType(type);
		form.setFieldsValue({
			type,
			title: widgetRegistry[type]?.displayName || '',
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

			// 根据Widget类型创建对应的显示配置
			const createDisplayConfig = (widgetType: WidgetType): StatCardDisplayConfig | ChartDisplayConfig | TableDisplayConfig | ListDisplayConfig | ProgressDisplayConfig => {
				switch (widgetType) {
					case 'stat-card':
						return {
							type: 'stat-card',
							icon: '📊',
							iconColor: '#1890ff',
							decimals: 0,
						};
					case 'chart':
						return {
							type: 'chart',
							chartType: 'line',
						};
					case 'table':
						return {
							type: 'table',
							columns: [],
						};
					case 'list':
						return {
							type: 'list',
							titleField: 'title',
						};
					case 'progress':
						return {
							type: 'progress',
							progressType: 'line',
						};
					default:
						return {
							type: 'stat-card',
							icon: '📊',
							iconColor: '#1890ff',
							decimals: 0,
						};
				}
			};

			// 处理参数：将表单的params数组转换为对象
			const paramsObj: Record<string, unknown> = {};
			if (values.params && Array.isArray(values.params)) {
				values.params.forEach((param: { name: string; value: string }) => {
					if (param.name && param.value) {
						paramsObj[param.name] = param.value;
					}
				});
			}

			// 创建API数据源配置 - 按后端期望的格式包装
			const apiDataSourceConfig = {
				type: 'api' as const,
				endpoint: values.endpoint || '/api/default',
				method: (values.method || 'GET') as 'GET' | 'POST',
				params: Object.keys(paramsObj).length > 0 ? paramsObj : undefined,
			};

			// 后端DataSourceConfig格式：{ api: ApiDataSourceConfig }
			const dataSource = {
				api: apiDataSourceConfig,
			};

			const widget: WidgetConfig = {
				id: editingWidgetId || `widget-${Date.now()}-${Math.random().toString(36).substring(2, 11)}`,
				type: values.type,
				title: values.title,
				position: editingWidget?.position || {
					x: 0,
					y: 0,
					...widgetDefaultSizes[values.type as WidgetType],
				},
				dataSource: dataSource as any, // 类型断言以匹配后端格式
				display: createDisplayConfig(values.type as WidgetType),
				enabled: true,
			};

			// 先触发回调（添加 Widget）
			onSelect(widget);

			// 延迟关闭弹窗，确保回调执行完成
			setTimeout(() => {
				setSelectedType(null);
				setSelectedEndpoint(null);
				form.resetFields();
				onClose();
			}, 100);
		} catch (error) {
			console.error('Validation failed:', error);
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
			title={editingWidgetId ? '编辑 Widget' : '添加 Widget'}
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

						<Form form={form} layout="vertical" initialValues={{ method: 'GET' }}>
							<Form.Item name="type" hidden>
								<input />
							</Form.Item>

							<Form.Item
								label="Widget类型"
								name="type"
								rules={[{ required: true }]}
							>
								<Select disabled>
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
								rules={[{ required: true, message: '请输入Widget标题' }]}
							>
								<Input placeholder="请输入Widget标题" />
							</Form.Item>

							<Divider titlePlacement="left">数据源配置</Divider>

							<Form.Item
								label="API端点"
								name="endpoint"
								rules={[{ required: true, message: '请选择API端点' }]}
								tooltip="从下拉列表中选择可用的API端点"
							>
								<EndpointSelector
									widgetType={selectedType || undefined}
									onChange={handleEndpointChange}
								/>
							</Form.Item>

							<Form.Item
								label="请求方法"
								name="method"
								tooltip="根据所选端点自动设置"
							>
								<Select>
									<Select.Option value="GET">GET</Select.Option>
									<Select.Option value="POST">POST</Select.Option>
								</Select>
							</Form.Item>

							<Form.Item
								label="数据路径"
								name="dataPath"
								tooltip="使用JSONata表达式提取数据，根据所选端点自动设置"
								extra="例如：data.list 用于提取分页数据列表"
							>
								<Input placeholder="data.list" />
							</Form.Item>

							<Form.Item
								label="请求参数"
								tooltip="配置API请求的查询参数"
							>
								<ParamsEditor endpoint={selectedEndpoint || undefined} form={form} />
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
