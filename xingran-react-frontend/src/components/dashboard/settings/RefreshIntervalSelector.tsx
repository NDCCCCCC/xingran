/**
 * RefreshIntervalSelector - 刷新间隔选择器
 *
 * 提供预设选项和自定义输入功能
 * 预设选项：30秒/1分钟/5分钟/15分钟/30分钟/1小时
 * 支持自定义输入（数字 + 单位）
 */

import { useState, useEffect } from "react";
import { Select, InputNumber, Space } from "antd";
import type { SelectProps } from "antd";

export interface RefreshIntervalSelectorProps extends Omit<SelectProps<number>, "options" | "onChange"> {
	/** 当前刷新间隔（秒） */
	value?: number;
	/** 变化回调 */
	onChange?: (value: number) => void;
	/** 是否禁用 */
	disabled?: boolean;
}

// 预设选项配置
const PRESET_OPTIONS = [
	{ label: "30 秒", value: 30 },
	{ label: "1 分钟", value: 60 },
	{ label: "5 分钟", value: 300 },
	{ label: "15 分钟", value: 900 },
	{ label: "30 分钟", value: 1800 },
	{ label: "1 小时", value: 3600 },
];

// 自定义选项标识
const CUSTOM_VALUE = -1;

// 单位选项
const UNIT_OPTIONS = [
	{ label: "秒", value: 1 },
	{ label: "分钟", value: 60 },
	{ label: "小时", value: 3600 },
];

/**
 * 刷新间隔选择器组件
 */
export const RefreshIntervalSelector: React.FC<RefreshIntervalSelectorProps> = ({
	value,
	onChange,
	disabled = false,
	...selectProps
}) => {
	// 判断是否为自定义值
	const isCustomValue = (val?: number): boolean => {
		if (val === undefined || val === null) return false;
		return !PRESET_OPTIONS.some((opt) => opt.value === val);
	};

	const [isCustom, setIsCustom] = useState(() => isCustomValue(value));
	const [customNumber, setCustomNumber] = useState<number>(() => {
		if (isCustomValue(value) && value) {
			// 尝试转换为合适的单位
			if (value! % 3600 === 0) return value! / 3600;
			if (value! % 60 === 0) return value! / 60;
			return value!;
		}
		return 1;
	});
	const [customUnit, setCustomUnit] = useState<number>(() => {
		if (isCustomValue(value) && value) {
			if (value! % 3600 === 0) return 3600;
			if (value! % 60 === 0) return 60;
			return 1;
		}
		return 60; // 默认分钟
	});

	// 当外部 value 变化时同步状态
	useEffect(() => {
		const custom = isCustomValue(value);
		// eslint-disable-next-line react-hooks/set-state-in-effect -- intentional sync on external value change
		setIsCustom(custom);
		if (custom && value) {
			// 尝试转换为合适的单位
			if (value % 3600 === 0) {
				setCustomNumber(value / 3600);
				setCustomUnit(3600);
			} else if (value % 60 === 0) {
				setCustomNumber(value / 60);
				setCustomUnit(60);
			} else {
				setCustomNumber(value);
				setCustomUnit(1);
			}
		}
	}, [value]);

	// 处理预设选项选择
	const handlePresetChange = (val: number) => {
		if (val === CUSTOM_VALUE) {
			setIsCustom(true);
			// 默认自定义值为 1 分钟
			const defaultValue = 60;
			onChange?.(defaultValue);
		} else {
			setIsCustom(false);
			onChange?.(val);
		}
	};

	// 处理自定义数值变化
	const handleCustomNumberChange = (val: number | null) => {
		const num = val || 1;
		setCustomNumber(num);
		onChange?.(num * customUnit);
	};

	// 处理自定义单位变化
	const handleCustomUnitChange = (val: number) => {
		setCustomUnit(val);
		onChange?.(customNumber * val);
	};

	// 计算当前显示的 Select 值
	const selectValue = isCustom ? CUSTOM_VALUE : value;

	return (
		<div className="refresh-interval-selector">
			<Space direction="vertical" style={{ width: "100%" }}>
				<Select
					value={selectValue}
					onChange={handlePresetChange}
					disabled={disabled}
					style={{ width: "100%" }}
					{...selectProps}
				 onSearch={() => {}}>
					{PRESET_OPTIONS.map((opt) => (
						<Select.Option key={opt.value} value={opt.value}>
							{opt.label}
						</Select.Option>
					))}
					<Select.Option value={CUSTOM_VALUE}>自定义</Select.Option>
				</Select>

				{isCustom && (
					<Space>
						<InputNumber
							min={1}
							max={1440} // 最大 1440 分钟 = 24 小时
							value={customNumber}
							onChange={handleCustomNumberChange}
							disabled={disabled}
							style={{ width: 100 }}
						/>
						<Select
							value={customUnit}
							onChange={handleCustomUnitChange}
							disabled={disabled}
							style={{ width: 80 }}
						 onSearch={() => {}}>
							{UNIT_OPTIONS.map((opt) => (
								<Select.Option key={opt.value} value={opt.value}>
									{opt.label}
								</Select.Option>
							))}
						</Select>
					</Space>
				)}
			</Space>
		</div>
	);
};

export default RefreshIntervalSelector;
