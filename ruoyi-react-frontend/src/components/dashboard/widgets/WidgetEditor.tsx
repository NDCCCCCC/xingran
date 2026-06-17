/**
 * WidgetEditor - Widget 编辑侧边抽屉组件
 *
 * 实现 Widget 编辑和删除功能
 * 从右侧滑出，可同时看到 Widget 变化
 * 支持实时预览
 */

import { useState, useEffect } from "react";
import { App, Drawer, Form, Input, Button, Modal, Spin, Divider, Space } from "antd";
import { SaveOutlined, DeleteOutlined, CloseOutlined } from "@ant-design/icons";
import { useDashboardStore } from "@/store/dashboardStore";
import DataSourceForm from "../settings/DataSourceForm";
import DisplayConfigForm from "../settings/DisplayConfigForm";
import type { WidgetConfig, DataSourceConfig, DisplayConfig } from "@/types/dashboard";

const { TextArea } = Input;

export interface WidgetEditorProps {
	/** 是否可见 */
	visible: boolean;
	/** 当前编辑的 Widget */
	widget: WidgetConfig | null;
	/** 关闭回调 */
	onClose: () => void;
}

/**
 * Widget 编辑侧边抽屉组件
 */
export const WidgetEditor: React.FC<WidgetEditorProps> = ({ visible, widget, onClose }) => {
	const { message } = App.useApp();
	const { updateWidget, removeWidget } = useDashboardStore();
	const [form] = Form.useForm();
	const [loading, setLoading] = useState(false);
	const [dataSource, setDataSource] = useState<DataSourceConfig | undefined>(undefined);
	const [displayConfig, setDisplayConfig] = useState<DisplayConfig | undefined>(undefined);

	// 当抽屉打开时初始化表单
	useEffect(() => {
		if (visible && widget) {
			form.setFieldsValue({
				title: widget.title,
				description: widget.description || "",
			});
			setDataSource(widget.dataSource);
			setDisplayConfig(widget.display);
		} else if (!visible) {
			// 关闭时重置状态
			form.resetFields();
			setDataSource(undefined);
			setDisplayConfig(undefined);
		}
	}, [visible, widget, form]);

	// 处理保存
	const handleSave = async () => {
		if (!widget) {
			message.warning("没有选择要编辑的 Widget");
			return;
		}

		try {
			setLoading(true);
			const values = await form.validateFields();

			// 更新 Widget
			updateWidget(widget.id, {
				title: values.title,
				dataSource: dataSource,
				display: displayConfig,
			});

			message.success("Widget 已更新");
			onClose();
		} catch (error) {
			console.error("保存 Widget 失败:", error);
			if (error instanceof Error) {
				message.error(`保存失败: ${error.message}`);
			}
		} finally {
			setLoading(false);
		}
	};

	// 处理删除
	const handleDelete = () => {
		if (!widget) return;

		Modal.confirm({
			title: "确认删除",
			content: (
				<div>
					<p>确定要删除 Widget 吗？</p>
					<p style={{ fontWeight: "bold", marginTop: 8 }}>「{widget.title}」</p>
					<p style={{ color: "var(--theme-text-tertiary, #999)", marginTop: 8 }}>此操作不可撤销</p>
				</div>
			),
			okText: "删除",
			okType: "danger",
			cancelText: "取消",
			onOk: () => {
				removeWidget(widget.id);
				message.success("Widget 已删除");
				onClose();
			},
		});
	};

	// 处理取消
	const handleCancel = () => {
		form.resetFields();
		onClose();
	};

	if (!widget) {
		return null;
	}

	return (
		<Drawer
			title="编辑 Widget"
			placement="right"
			open={visible}
			onClose={handleCancel}
			width={520}
			footer={
				<div style={{ display: "flex", justifyContent: "space-between" }}>
					<Button danger icon={<DeleteOutlined />} onClick={handleDelete}>
						删除
					</Button>
					<Space>
						<Button icon={<CloseOutlined />} onClick={handleCancel}>
							取消
						</Button>
						<Button
							type="primary"
							icon={<SaveOutlined />}
							loading={loading}
							onClick={handleSave}
						>
							保存
						</Button>
					</Space>
				</div>
			}
		>
			<Spin spinning={loading}>
				<Form form={form} layout="vertical" initialValues={widget}>
					{/* 基本信息 */}
					<Divider titlePlacement="left" plain>
						基本信息
					</Divider>

					<Form.Item
						label="标题"
						name="title"
						rules={[
							{ required: true, message: "请输入 Widget 标题" },
							{ max: 100, message: "标题不能超过 100 个字符" },
						]}
					>
						<Input placeholder="请输入 Widget 标题" />
					</Form.Item>

					<Form.Item
						label="描述"
						name="description"
						rules={[{ max: 500, message: "描述不能超过 500 个字符" }]}
					>
						<TextArea rows={2} placeholder="可选的 Widget 描述" />
					</Form.Item>

					{/* 数据源配置 */}
					<Divider titlePlacement="left" plain>
						数据源配置
					</Divider>

					<Form.Item>
						<DataSourceForm
							value={dataSource}
							onChange={setDataSource}
							widgetType={widget.type}
							form={form}
						/>
					</Form.Item>

					{/* 显示配置 */}
					<Divider titlePlacement="left" plain>
						显示配置
					</Divider>

					<Form.Item>
						<DisplayConfigForm
							value={displayConfig}
							onChange={setDisplayConfig}
							widgetType={widget.type}
							form={form}
						/>
					</Form.Item>

					{/* Widget 信息 */}
					<Divider titlePlacement="left" plain>
						Widget 信息
					</Divider>

					<div style={{ color: "#666", fontSize: 12 }}>
						<p>
							<strong>ID:</strong> {widget.id}
						</p>
						<p>
							<strong>类型:</strong> {widget.type}
						</p>
						<p>
							<strong>位置:</strong> ({widget.position.x}, {widget.position.y})
						</p>
						<p>
							<strong>尺寸:</strong> {widget.position.w} × {widget.position.h}
						</p>
						{widget.createdAt && (
							<p>
								<strong>创建时间:</strong> {widget.createdAt}
							</p>
						)}
						{widget.updatedAt && (
							<p>
								<strong>更新时间:</strong> {widget.updatedAt}
							</p>
						)}
					</div>
				</Form>
			</Spin>
		</Drawer>
	);
};

export default WidgetEditor;
