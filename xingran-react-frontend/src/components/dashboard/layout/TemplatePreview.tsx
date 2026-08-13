/**
 * TemplatePreview - 模板全屏预览组件
 *
 * 实现模板全屏预览功能，显示完整的仪表盘布局和真实数据
 * 用户可以在预览中点击"使用此模板"创建仪表盘
 */

import { useState } from "react";
import { Modal, Button, Spin, Empty } from "antd";
import { CloseOutlined, CheckOutlined, ReloadOutlined } from "@ant-design/icons";
import { dashboardService } from "@/services/dashboardService";
import { DashboardGrid } from "../layout/DashboardGrid";
import type { Dashboard, WidgetConfig } from "@/types/dashboard";

export interface TemplatePreviewProps {
	/** 是否可见 */
	visible: boolean;
	/** 模板配置 */
	template: Dashboard | null;
	/** 关闭回调 */
	onClose: () => void;
	/** 应用模板回调 */
	onApply: (templateId: string, name: string) => void;
}

/**
 * 模板全屏预览组件
 */
export const TemplatePreview: React.FC<TemplatePreviewProps> = ({
	visible,
	template,
	onClose,
	onApply,
}) => {
	const [loading, setLoading] = useState(false);
	const [previewData, setPreviewData] = useState<Map<string, unknown>>(new Map());
	const [error, setError] = useState<string | null>(null);

	// 加载预览数据
	const loadPreviewData = async () => {
		if (!template) return;

		setLoading(true);
		setError(null);

		try {
			const widgetIds = template.layout.widgets.map((w: WidgetConfig) => w.id);
			if (widgetIds.length === 0) {
				setPreviewData(new Map());
				return;
			}

			const data = await dashboardService.getBatchWidgetData(widgetIds);
			setPreviewData(data);
		} catch (err) {
			console.error("加载预览数据失败:", err);
			setError(err instanceof Error ? err.message : "加载预览数据失败");
		} finally {
			setLoading(false);
		}
	};

	// 当 visible 或 template 变化时加载预览数据
	useEffect(() => {
		if (visible && template) {
			loadPreviewData();
		} else {
			// 关闭时重置状态
			setPreviewData(new Map());
			setError(null);
		}
	}, [visible, template]);

	// 处理应用模板
	const handleApply = () => {
		if (!template) return;
		onApply(template.id, `${template.name} - 副本`);
	};

	// 处理重新加载
	const handleReload = () => {
		loadPreviewData();
	};

	if (!template) {
		return null;
	}

	return (
		<Modal
			open={visible}
			onCancel={onClose}
			footer={null}
			width="100vw"
			style={{ top: 0, paddingBottom: 0, maxWidth: "100vw" }}
			styles={{
				body: {
					height: "calc(100vh - 110px)",
					overflow: "auto",
					padding: 0,
				},
			}}
			closeIcon={<CloseOutlined style={{ fontSize: 18 }} />}
			title={
				<div style={{ display: "flex", alignItems: "center", gap: 8 }}>
					<span>模板预览</span>
					<span style={{ color: "var(--theme-text-tertiary, #999)", fontSize: 14, fontWeight: "normal" }}>
						{template.name}
					</span>
				</div>
			}
		>
			<Spin spinning={loading}>
				{loading && (
					<div style={{ textAlign: "center", padding: 12, color: "rgba(0, 0, 0, 0.45)" }}>
						加载预览数据...
					</div>
				)}
				<div style={{ minHeight: "calc(100vh - 200px)", position: "relative" }}>
					{error ? (
						<div
							style={{
								display: "flex",
								flexDirection: "column",
								alignItems: "center",
								justifyContent: "center",
								height: "100%",
								minHeight: 400,
							}}
						>
							<Empty description={error}>
								<Button icon={<ReloadOutlined />} onClick={handleReload}>
									重新加载
								</Button>
							</Empty>
						</div>
					) : (
						<>
					{/* 仪表盘布局预览 */}
					<DashboardGrid
						widgets={template.layout.widgets}
						onLayoutChange={() => {}}
					>
						{template.layout.widgets.map((widget) => (
							<div key={widget.id}>{widget.title}</div>
						))}
					</DashboardGrid>

							{/* 底部操作按钮 */}
							<div
								style={{
									position: "fixed",
									bottom: 24,
									right: 24,
									display: "flex",
									gap: 8,
									zIndex: 1000,
								}}
							>
								<Button
									size="large"
									icon={<CloseOutlined />}
									onClick={onClose}
								>
									取消
								</Button>
								<Button
									type="primary"
									size="large"
									icon={<CheckOutlined />}
									onClick={handleApply}
								>
									使用此模板
								</Button>
							</div>
						</>
					)}
				</div>
			</Spin>
		</Modal>
	);
};

export default TemplatePreview;
