/**
 * 用户设置页面（增强版 - 支持明暗模式和自定义颜色）
 * User Settings Page (Enhanced - Light/Dark Mode & Custom Colors)
 */

import { useEffect, useState, type FC } from 'react';
import {
	Card,
	Form,
	Select,
	Switch,
	Button,
	message,
	Divider,
	Alert,
	Radio,
	ColorPicker,
	Row,
	Col,
} from 'antd';
import {
	SunOutlined,
	MoonOutlined,
	BgColorsOutlined,
	MenuOutlined,
} from '@ant-design/icons';
import { useSettingsStore } from '@/store/settingsStore';
import type { UserPreferences } from '@/types/config';
import { themePresets } from '@/design-system/themes';
import type { ColorPickerProps } from 'antd';
import { applyPrimaryColor, applySidebarBackgroundColor } from '@/design-system/themes';

const { Option } = Select;

const SettingsPage: FC = () => {
	const { preferences, loading, initialized, updatePreferences } = useSettingsStore();
	const [form] = Form.useForm();

	// 本地状态存储颜色值（字符串格式）
	const [primaryColor, setPrimaryColor] = useState<string>('#4F46E5');
	const [sidebarColor, setSidebarColor] = useState<string>('#1E293B');

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

			// 同步颜色值到本地状态
			if (typeof preferences.theme.customColors?.primary === 'string') {
				setPrimaryColor(preferences.theme.customColors.primary);
			}
			if (typeof preferences.theme.customColors?.sidebar === 'string') {
				setSidebarColor(preferences.theme.customColors.sidebar);
			}
		}
	}, [initialized, preferences, form]);

	// 保存设置
	const handleSave = async () => {
		try {
			const values = await form.validateFields();

			// 确保颜色值是字符串格式
			const processedValues = {
				...values,
				theme: {
					...values.theme,
					customColors: {
						primary: primaryColor,
						sidebar: sidebarColor,
					},
				},
			};

			await updatePreferences(processedValues);
			message.success('设置保存成功');
		} catch (error) {
			message.error('设置保存失败');
		}
	};

	// 重置设置
	const handleReset = () => {
		form.resetFields();
		form.setFieldsValue(preferences);

		// 重置颜色值
		if (typeof preferences.theme.customColors?.primary === 'string') {
			setPrimaryColor(preferences.theme.customColors.primary);
		} else {
			setPrimaryColor('#4F46E5');
		}
		if (typeof preferences.theme.customColors?.sidebar === 'string') {
			setSidebarColor(preferences.theme.customColors.sidebar);
		} else {
			setSidebarColor('#1E293B');
		}
	};

	// 清除自定义颜色
	const handleClearCustomColors = () => {
		setPrimaryColor('#4F46E5');
		setSidebarColor('#1E293B');
		form.setFieldValue(['theme', 'customColors'], undefined);
		message.info('自定义颜色已清除，将使用主题预设颜色');
	};

	// 主色调变化处理
	const handlePrimaryColorChange: ColorPickerProps['onChange'] = (color) => {
		const hexColor = typeof color === 'string' ? color : color.toHexString();

		// 立即预览
		applyPrimaryColor(hexColor);

		// 更新本地状态
		setPrimaryColor(hexColor);
	};

	// 侧边栏颜色变化处理
	const handleSidebarColorChange: ColorPickerProps['onChange'] = (color) => {
		const hexColor = typeof color === 'string' ? color : color.toHexString();

		// 立即预览
		applySidebarBackgroundColor(hexColor);

		// 更新本地状态
		setSidebarColor(hexColor);
	};

	if (!initialized) {
		return <div>加载中...</div>;
	}

	return (
		<div className="p-6">
			<Card title="用户设置" loading={loading}>
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

					{/* 主题模式 - 明暗切换 */}
					<Form.Item
						label="明暗模式"
						name={['theme', 'mode']}
						tooltip="选择系统的颜色模式"
						rules={[{ required: true, message: '请选择明暗模式' }]}
					>
						<Radio.Group>
							<Radio.Button value="light">
								<SunOutlined /> 浅色模式
							</Radio.Button>
							<Radio.Button value="dark">
								<MoonOutlined /> 深色模式
							</Radio.Button>
						</Radio.Group>
					</Form.Item>

					{/* 主题风格 */}
					<Form.Item
						label="主题风格"
						name={['theme', 'style']}
						tooltip="选择系统的视觉风格"
						rules={[{ required: true, message: '请选择主题风格' }]}
					>
						<Select>
							{themePresets.map((preset) => (
								<Option key={preset.id} value={preset.id}>
									{preset.icon} {preset.name} - {preset.description}
								</Option>
							))}
						</Select>
					</Form.Item>

					{/* 自定义颜色 */}
					<Divider styles={{ content: { margin: 0 } }}>颜色自定义（可选）</Divider>

					<Form.Item
						label="主题色"
						tooltip="自定义主色调（留空使用主题预设）"
						extra="留空则使用当前主题的预设颜色"
					>
						<Row gutter={16} align="middle">
							<Col>
								<ColorPicker
									value={primaryColor}
									onChange={handlePrimaryColorChange}
									format="hex"
									showText
									allowClear
									onClear={() => {
										setPrimaryColor('#4F46E5');
										applyPrimaryColor('#4F46E5');
									}}
								/>
							</Col>
							<Col>
								<div
									style={{
										width: '40px',
										height: '40px',
										borderRadius: '8px',
										background: primaryColor,
										border: '1px solid rgba(0,0,0,0.1)',
									}}
								/>
							</Col>
						</Row>
					</Form.Item>

					<Form.Item
						label="侧边栏颜色"
						tooltip="自定义侧边栏背景色（留空使用主题预设）"
						extra="留空则使用当前主题的预设颜色"
					>
						<Row gutter={16} align="middle">
							<Col>
								<ColorPicker
									value={sidebarColor}
									onChange={handleSidebarColorChange}
									format="hex"
									showText
									allowClear
									onClear={() => {
										setSidebarColor('#1E293B');
										applySidebarBackgroundColor('#1E293B');
									}}
								/>
							</Col>
							<Col>
								<div
									style={{
										width: '40px',
										height: '40px',
										borderRadius: '8px',
										background: sidebarColor,
										border: '1px solid rgba(0,0,0,0.1)',
									}}
								/>
							</Col>
						</Row>
					</Form.Item>

					<Form.Item>
						<Button onClick={handleClearCustomColors} size="small">
							清除自定义颜色
						</Button>
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
