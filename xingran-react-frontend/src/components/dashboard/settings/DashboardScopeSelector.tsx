/**
 * 仪表盘可见范围选择器
 * 根据用户数据范围权限动态显示可用选项
 */
import { useMemo } from "react";
import { Form, Select, Switch, Space } from "antd";
import { useAuthStore } from "@/store/authStore";
import type { DashboardScope } from "@/types/dashboard";

interface DashboardScopeSelectorProps {
	value?: {
		scope: DashboardScope;
		deptId?: string;
		isSystem?: boolean;
	};
	onChange: (value: { scope: DashboardScope; deptId?: string; isSystem?: boolean }) => void;
	disabled?: boolean;
}

export const DashboardScopeSelector: React.FC<DashboardScopeSelectorProps> = ({
	value,
	onChange,
	disabled = false,
}) => {
	const { user } = useAuthStore();
	const isAdmin = user?.isAdmin || false;

	// 获取用户数据范围
	const getDataScope = () => {
		// 从用户角色获取数据范围
		return user?.dataScope || "dept";
	};

	const dataScope = getDataScope();

	// 构建可用选项
	const scopeOptions = useMemo(() => {
		const options = [
			{ label: "私有（仅自己可见）", value: "private" },
		];

		// 部门可见
		options.push({
			label: dataScope === "all" ? "部门可见（可选择部门）" : "部门可见（本部门）",
			value: "dept",
		});

		// 全局可见仅管理员可设置
		if (isAdmin) {
			options.push({
				label: "全局可见（所有人）",
				value: "global",
			});
		}

		return options;
	}, [dataScope, isAdmin]);

	const handleScopeChange = (scope: DashboardScope) => {
		const newValue: { scope: DashboardScope; isSystem: boolean; deptId?: string } = {
			scope,
			isSystem: value?.isSystem || false,
		};

		if (scope === "dept") {
			if (dataScope === "all") {
				// 管理员可以设置部门，这里暂时使用用户部门
				newValue.deptId = user?.deptId;
			} else {
				// 自动使用用户本部门
				newValue.deptId = user?.deptId;
			}
		}

		onChange(newValue);
	};

	const handleIsSystemChange = (isSystem: boolean) => {
		const newValue = {
			scope: value?.scope || "private",
			isSystem,
		};

		// 系统仪表盘必须是全局可见的
		if (isSystem) {
			newValue.scope = "global";
		}

		onChange(newValue);
	};

	// 是否显示系统仪表盘选项
	const showSystemOption = isAdmin;

	return (
		<Space direction="vertical" style={{ width: "100%" }}>
			<Form.Item label="可见范围" style={{ marginBottom: 8 }}>
				{/* eslint-disable-next-line local/no-large-dropdown-list -- fixed option list, no server search needed */}
				<Select
					value={value?.scope || "private"}
					onChange={handleScopeChange}
					options={scopeOptions}
					style={{ width: "100%" }}
					disabled={disabled || (value?.isSystem && value?.scope === "global")}
				/>
			</Form.Item>

			{showSystemOption && (
				<Form.Item label="系统仪表盘" style={{ marginBottom: 0 }}>
					<Switch
						checked={value?.isSystem || false}
						onChange={handleIsSystemChange}
						disabled={disabled}
						checkedChildren="是"
						unCheckedChildren="否"
					/>
					<div style={{ color: "var(--theme-text-tertiary, #999)", fontSize: 12, marginTop: 4 }}>
						系统仪表盘将作为无默认仪表盘用户的兜底选项
					</div>
				</Form.Item>
			)}
		</Space>
	);
};

export default DashboardScopeSelector;
