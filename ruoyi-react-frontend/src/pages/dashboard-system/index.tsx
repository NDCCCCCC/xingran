/**
 * 仪表盘列表页
 *
 * 显示所有仪表盘，支持创建、编辑、删除等操作
 */

import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import {
	Card,
	Table,
	Button,
	Space,
	Tag,
	message,
	Popconfirm,
	Input,
	Row,
	Col,
} from 'antd';
import {
	PlusOutlined,
	EditOutlined,
	DeleteOutlined,
	CopyOutlined,
	SettingOutlined,
	EyeOutlined,
	SearchOutlined,
} from '@ant-design/icons';
import { useDashboardStore } from '@/store/dashboardStore';
import { TemplateSelector } from '@/components/dashboard/layout/TemplateSelector';
import type { Dashboard } from '@/types/dashboard';
import type { ColumnsType } from 'antd/es/table';

import './index.css';

const DashboardList: React.FC = () => {
	const navigate = useNavigate();
	const [searchKeyword, setSearchKeyword] = useState('');
	const [showTemplateSelector, setShowTemplateSelector] = useState(false);

	const {
		dashboards,
		listLoading,
		listPagination,
		fetchDashboards,
		createDashboard,
		deleteDashboard,
		duplicateDashboard,
		setDefaultDashboard,
	} = useDashboardStore();

	// 加载仪表盘列表
	useEffect(() => {
		fetchDashboards({ current: 1, pageSize: 10 });
	}, [fetchDashboards]);

	// 刷新列表
	const handleRefresh = () => {
		fetchDashboards({
			current: listPagination.current,
			pageSize: listPagination.pageSize,
			keyword: searchKeyword,
		});
	};

	// 搜索
	const handleSearch = () => {
		fetchDashboards({
			current: 1,
			pageSize: listPagination.pageSize,
			keyword: searchKeyword,
		});
	};

	// 创建仪表盘
	const handleCreate = async (_templateType: string, name: string) => {
		try {
			const dashboard = await createDashboard({
				name,
				layout: { widgets: [], columns: { desktop: 24, tablet: 12, mobile: 6 }, rowHeight: 60, margin: [16, 16], draggable: true, resizable: true },
				refreshInterval: 60,
			});
			message.success('创建成功');
			navigate(`/dashboard/${dashboard.id}/edit`);
		} catch (error) {
			message.error(`创建失败: ${(error as Error).message}`);
		}
	};

	// 删除仪表盘
	const handleDelete = async (id: string) => {
		try {
			await deleteDashboard(id);
			message.success('删除成功');
			handleRefresh();
		} catch (error) {
			message.error(`删除失败: ${(error as Error).message}`);
		}
	};

	// 复制仪表盘
	const handleDuplicate = async (id: string) => {
		try {
			await duplicateDashboard(id);
			message.success('复制成功');
			handleRefresh();
		} catch (error) {
			message.error(`复制失败: ${(error as Error).message}`);
		}
	};

	// 设置为默认
	const handleSetDefault = async (id: string) => {
		try {
			await setDefaultDashboard(id);
			message.success('设置成功');
			handleRefresh();
		} catch (error) {
			message.error(`设置失败: ${(error as Error).message}`);
		}
	};

	// 表格列配置
	const columns: ColumnsType<Dashboard> = [
		{
			title: '名称',
			dataIndex: 'name',
			key: 'name',
			width: 200,
		},
		{
			title: '描述',
			dataIndex: 'description',
			key: 'description',
			width: 300,
			ellipsis: true,
		},
		{
			title: 'Widget数',
			key: 'widgetCount',
			width: 100,
			render: (_, record) => record.layout.widgets.length,
		},
		{
			title: '刷新间隔',
			key: 'refreshInterval',
			width: 100,
			render: (_, record) => `${record.refreshInterval}秒`,
		},
		{
			title: '类型',
			key: 'type',
			width: 100,
			render: (_, record) => (
				<>
					{record.isDefault && <Tag color="blue">默认</Tag>}
					{record.isTemplate && <Tag color="green">模板</Tag>}
				</>
			),
		},
		{
			title: '状态',
			dataIndex: 'status',
			key: 'status',
			width: 80,
			render: (status) => (
				<Tag color={status === 0 ? 'green' : 'red'}>
					{status === 0 ? '正常' : '停用'}
				</Tag>
			),
		},
		{
			title: '操作',
			key: 'actions',
			width: 200,
			fixed: 'right',
			render: (_, record) => (
				<Space size="small">
					<Button
						type="link"
						size="small"
						icon={<EyeOutlined />}
						onClick={() => navigate(`/dashboard/${record.id}`)}
					>
						查看
					</Button>
					<Button
						type="link"
						size="small"
						icon={<EditOutlined />}
						onClick={() => navigate(`/dashboard/${record.id}/edit`)}
					>
						编辑
					</Button>
					<Button
						type="link"
						size="small"
						icon={<CopyOutlined />}
						onClick={() => handleDuplicate(record.id)}
					>
						复制
					</Button>
					{!record.isDefault && (
						<Button
							type="link"
							size="small"
							icon={<SettingOutlined />}
							onClick={() => handleSetDefault(record.id)}
						>
							设为默认
						</Button>
					)}
					<Popconfirm
						title="确定要删除这个仪表盘吗？"
						onConfirm={() => handleDelete(record.id)}
						okText="确定"
						cancelText="取消"
					>
						<Button
							type="link"
							size="small"
							danger
							icon={<DeleteOutlined />}
						>
							删除
						</Button>
					</Popconfirm>
				</Space>
			),
		},
	];

	return (
		<div className="dashboard-list-page">
			<div className="dashboard-list-page__header">
				<h2 className="dashboard-list-page__title">仪表盘管理</h2>
				<Button
					type="primary"
					icon={<PlusOutlined />}
					onClick={() => setShowTemplateSelector(true)}
				>
					新建仪表盘
				</Button>
			</div>

			<Card>
				<Row gutter={16} style={{ marginBottom: 16 }}>
					<Col flex="auto">
						<Input
							placeholder="搜索仪表盘名称或描述"
							prefix={<SearchOutlined />}
							value={searchKeyword}
							onChange={(e) => setSearchKeyword(e.target.value)}
							onPressEnter={handleSearch}
							allowClear
						/>
					</Col>
					<Col>
						<Button onClick={handleSearch}>搜索</Button>
					</Col>
					<Col>
						<Button onClick={handleRefresh}>刷新</Button>
					</Col>
				</Row>

				<Table
					columns={columns}
					dataSource={dashboards}
					rowKey="id"
					loading={listLoading}
					scroll={{ x: 1200 }}
					pagination={{
						current: listPagination.current,
						pageSize: listPagination.pageSize,
						total: listPagination.total,
						showSizeChanger: true,
						showQuickJumper: true,
						showTotal: (total) => `共 ${total} 条`,
						onChange: (page, pageSize) => {
							fetchDashboards({
								current: page,
								pageSize,
								keyword: searchKeyword,
							});
						},
					}}
				/>
			</Card>

			<TemplateSelector
				visible={showTemplateSelector}
				onClose={() => setShowTemplateSelector(false)}
				onSelect={(templateType, name) => handleCreate(templateType, name)}
			/>
		</div>
	);
};

export default DashboardList;
