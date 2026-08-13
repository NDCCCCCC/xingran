/**
 * TemplateSelector - 仪表盘模板选择器
 *
 * 允许用户从预设模板创建仪表盘
 * 支持模板预览功能
 */

import { useState, useLayoutEffect } from "react";
import { App, Modal, Card, Row, Col, Button, Tag, Space, Tooltip } from "antd";
import {
	AppstoreOutlined,
	LineChartOutlined,
	ProfileOutlined,
	DatabaseOutlined,
	EyeOutlined,
} from "@ant-design/icons";
import { useNavigate } from "react-router-dom";
import type { PresetTemplateType, Dashboard } from "@/types/dashboard";
import { presetDashboardTemplates } from "@/components/dashboard/templates/presets";
import TemplatePreview from "./TemplatePreview";

import "./TemplateSelector.css";

interface TemplateSelectorProps {
	/** 是否可见 */
	visible: boolean;

	/** 关闭回调 */
	onClose: () => void;

	/** 选择模板回调 */
	onSelect: (templateType: PresetTemplateType, name: string) => void;
}

export const TemplateSelector: React.FC<TemplateSelectorProps> = ({
	visible,
	onClose,
	onSelect,
}) => {
	const [selectedTemplate, setSelectedTemplate] = useState<PresetTemplateType | null>(null);
	const [dashboardName, setDashboardName] = useState("");
	const [previewVisible, setPreviewVisible] = useState(false);
	const [previewTemplate, setPreviewTemplate] = useState<Dashboard | null>(null);
	const { message } = App.useApp();
	const navigate = useNavigate();

	// 重置状态 - 使用 useLayoutEffect 避免级联渲染
	useLayoutEffect(() => {
		if (visible) {
			setSelectedTemplate(null);
			setDashboardName("");
		}
	}, [visible]);

	// 选择模板
	const handleSelectTemplate = (templateType: PresetTemplateType) => {
		setSelectedTemplate(templateType);
		const template = presetDashboardTemplates.find(t => t.type === templateType);
		if (template) {
			setDashboardName(`${template.displayName} - 副本`);
		}
	};

	// 确认创建
	const handleConfirm = () => {
		if (!selectedTemplate) {
			Modal.warning({
				title: "请选择模板",
				content: "请先选择一个仪表盘模板",
			});
			return;
		}

		if (!dashboardName.trim()) {
			Modal.warning({
				title: "请输入名称",
				content: "请输入仪表盘名称",
			});
			return;
		}

		onSelect(selectedTemplate, dashboardName.trim());
		onClose();
	};

	// 处理预览
	const handlePreview = (templateType: PresetTemplateType) => {
		const template = presetDashboardTemplates.find(t => t.type === templateType);
		if (template) {
			// 将预设模板转换为 Dashboard 格式
			const dashboard: Dashboard = {
				...template.dashboard,
				id: `template-${templateType}`,
				name: template.displayName,
				description: template.description,
				isDefault: false,
				isTemplate: true,
				layout: template.dashboard.layout,
				refreshInterval: template.dashboard.refreshInterval || 300,
				status: 0,
				createdAt: new Date().toISOString(),
				updatedAt: new Date().toISOString(),
			} as Dashboard;
			setPreviewTemplate(dashboard);
			setPreviewVisible(true);
		}
	};

	// 处理应用模板
	const handleApplyTemplate = async (templateId: string, name: string) => {
		try {
			// 从模板 ID 提取模板类型
			const templateType = templateId.replace("template-", "") as PresetTemplateType;
			// 调用创建回调
			onSelect(templateType, name);
			setPreviewVisible(false);
			onClose();
		} catch (error) {
			console.error("应用模板失败:", error);
			message.error("应用模板失败");
		}
	};

	// 获取模板图标
	const getTemplateIcon = (type: PresetTemplateType) => {
		switch (type) {
			case "operations-overview":
				return <AppstoreOutlined />;
			case "workorder-management":
				return <ProfileOutlined />;
			case "duty-management":
				return <DatabaseOutlined />;
			case "system-monitor":
				return <LineChartOutlined />;
			default:
				return <AppstoreOutlined />;
		}
	};

	// 获取模板标签
	const getTemplateTag = (type: PresetTemplateType) => {
		switch (type) {
			case "operations-overview":
				return <Tag color="blue">运维</Tag>;
			case "workorder-management":
				return <Tag color="green">工单</Tag>;
			case "duty-management":
				return <Tag color="orange">值班</Tag>;
			case "system-monitor":
				return <Tag color="purple">监控</Tag>;
			default:
				return <Tag>其他</Tag>;
		}
	};

	return (
		<>
			<Modal
				title="选择仪表盘模板"
				open={visible}
				onCancel={onClose}
				onOk={handleConfirm}
				okText="创建仪表盘"
				cancelText="取消"
				width={900}
				okButtonProps={{ disabled: !selectedTemplate }}
			>
				<div className="template-selector">
					<div className="template-selector__templates">
						<Row gutter={[16, 16]}>
							{presetDashboardTemplates.map((template) => (
								<Col xs={24} sm={12} md={12} key={template.type}>
									<Card
										hoverable
										className={`template-card ${selectedTemplate === template.type ? "template-card--selected" : ""}`}
										onClick={() => handleSelectTemplate(template.type)}
									>
										<div className="template-card__header">
											<Space>
												<div className="template-card__icon">
													{getTemplateIcon(template.type)}
												</div>
												<div>
													<h3 className="template-card__title">
														{template.displayName}
													</h3>
													<div className="template-card__tags">
														{getTemplateTag(template.type)}
														<Tag color="default">
															{template.dashboard.layout.widgets.length} 个Widget
														</Tag>
													</div>
												</div>
											</Space>
										</div>
										<p className="template-card__description">
											{template.description}
										</p>
										<div className="template-card__footer">
											<Tooltip title="预览模板">
												<Button
													type="text"
													size="small"
													icon={<EyeOutlined />}
													onClick={(e) => {
														e.stopPropagation();
														handlePreview(template.type);
													}}
												>
													预览
												</Button>
											</Tooltip>
										</div>
									</Card>
								</Col>
							))}
						</Row>
					</div>

					{selectedTemplate && (
						<div className="template-selector__form">
							<label className="template-selector__label">仪表盘名称：</label>
							<input
								type="text"
								className="template-selector__input"
								value={dashboardName}
								onChange={(e) => setDashboardName(e.target.value)}
								placeholder="请输入仪表盘名称"
								autoFocus
							/>
						</div>
					)}
				</div>
			</Modal>

			{/* 模板预览 */}
			<TemplatePreview
				visible={previewVisible}
				template={previewTemplate}
				onClose={() => setPreviewVisible(false)}
				onApply={handleApplyTemplate}
			/>
		</>
	);
};
