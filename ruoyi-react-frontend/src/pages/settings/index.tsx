/**
 * 系统设置页面（重构版）
 * System Settings Page (Refactored)
 *
 * 支持新的嵌套配置结构
 */

import { useEffect, type FC } from 'react';
import {
	Card,
	Form,
	Select,
	Switch,
	Button,
	message,
	Divider,
	Alert,
} from 'antd';
import { useSettingsStore } from '@/store/settingsStore';
import type { UserPreferences } from '@/types/config';

const { Option } = Select;

const SettingsPage: FC = () => {
	const { preferences, loading, initialized, updatePreferences } = useSettingsStore();
	const [form] = Form.useForm();

	// 加载设置
	useEffect(() => {
		if (!initialized) {
			useSettingsStore.getState().initialize();
		}
	}, [initialized]);

	// 同步表单值
	useEffect(() => {
		if (initialized && preferences) {
			form.setFieldsValue(preferences);
		}
	}, [initialized, preferences, form]);

	// 保存设置
	const handleSave = async () => {
		try {
			const values = await form.validateFields();
			await updatePreferences(values);
			message.success('设置保存成功');
		} catch (error) {
			message.error('设置保存失败');
		}
	};

	// 重置设置
	const handleReset = () => {
		form.resetFields();
		form.setFieldsValue(preferences);
	};

	if (!initialized) {
		return <div>加载中...</div>;
	}

	return (
		<div className="p-6">
			<Card title="系统设置" loading={loading}>
				<Alert
					title="配置说明"
					description="所有配置会自动保存到服务器，并在您下次登录时恢复。"
					type="info"
					showIcon
					style={{ marginBottom: 16 }}
				/>

				<Form
					form={form}
					layout="vertical"
					initialValues={preferences}
				>
					<Divider styles={{ content: { margin: 0 } }}>界面设置</Divider>

					{/* 主题模式 */}
					<Form.Item
						label="主题模式"
						name={['theme', 'mode']}
						tooltip="选择系统的颜色模式"
						rules={[{ required: true, message: '请选择主题模式' }]}
					>
						<Select>
							<Option value="light">浅色模式</Option>
							<Option value="dark">深色模式</Option>
						</Select>
					</Form.Item>

					{/* 主题风格 */}
					<Form.Item
						label="主题风格"
						name={['theme', 'style']}
						tooltip="选择系统的视觉风格"
						rules={[{ required: true, message: '请选择主题风格' }]}
					>
						<Select>
							<Option value="minimal">极简风格</Option>
							<Option value="glassmorphism">玻璃拟态</Option>
							<Option value="neumorphism">新拟态</Option>
							<Option value="flat2.0">扁平化 2.0</Option>
							<Option value="luxury-quiet">奢华静谧</Option>
						</Select>
					</Form.Item>

					{/* 语言（标记为即将推出） */}
					<Form.Item
						label="语言"
						name="language"
						tooltip="选择系统显示语言"
						extra="即将推出完整的多语言支持"
					>
						<Select disabled>
							<Option value="zh-CN">简体中文（即将推出）</Option>
							<Option value="en-US">English（即将推出）</Option>
						</Select>
					</Form.Item>

					<Divider styles={{ content: { margin: 0 } }}>布局设置</Divider>

					{/* 布局类型 */}
					<Form.Item
						label="布局类型"
						name={['layout', 'type']}
						tooltip="选择系统的布局方式"
						rules={[{ required: true, message: '请选择布局类型' }]}
					>
						<Select>
							<Option value="classic">经典布局</Option>
							<Option value="hybrid">混合布局</Option>
							<Option value="innovative">创新布局</Option>
						</Select>
					</Form.Item>

					{/* 密度模式 */}
					<Form.Item
						label="密度模式"
						name={['layout', 'density']}
						tooltip="选择界面的紧凑程度"
						rules={[{ required: true, message: '请选择密度模式' }]}
					>
						<Select>
							<Option value="compact">紧凑</Option>
							<Option value="comfortable">舒适</Option>
							<Option value="spacious">宽松</Option>
						</Select>
					</Form.Item>

					{/* 侧边栏折叠 */}
					<Form.Item
						label="默认折叠侧边栏"
						name={['layout', 'sidebar', 'collapsed']}
						valuePropName="checked"
						tooltip="默认折叠侧边栏以节省空间"
					>
						<Switch />
					</Form.Item>

					<Divider styles={{ content: { margin: 0 } }}>数据设置</Divider>

					{/* 默认分页大小 */}
					<Form.Item
						label="默认分页大小"
						name={['data', 'defaultPageSize']}
						tooltip="列表页面默认每页显示的数据条数"
						rules={[{ required: true, message: '请选择默认分页大小' }]}
					>
						<Select>
							<Option value={10}>10 条/页</Option>
							<Option value={20}>20 条/页</Option>
							<Option value={50}>50 条/页</Option>
							<Option value={100}>100 条/页</Option>
						</Select>
					</Form.Item>

					<Form.Item>
						<Button type="primary" onClick={handleSave} className="mr-2">
							保存设置
						</Button>
						<Button onClick={handleReset}>
							重置
						</Button>
					</Form.Item>
				</Form>
			</Card>
		</div>
	);
};

export default SettingsPage;
