/**
 * 日字段配置组件
 */

import type { FC } from "react";
import type { RadioChangeEvent } from "antd";
import { Radio, Select, Space } from "antd";
import type { CronFieldConfig, PeriodType } from "../constants";
import { FIELD_RANGES } from "../constants";

interface DayFieldProps {
  value: CronFieldConfig;
  onChange: (value: CronFieldConfig) => void;
}

const DayField: FC<DayFieldProps> = ({ value, onChange }) => {
  const { periodType } = value;
  const range = FIELD_RANGES.day;

  // 生成日期选项数组
  const options = Array.from({ length: range.max - range.min + 1 }, (_, i) => range.min + i);

  return (
    <div>
      <Radio.Group value={periodType} onChange={(e: RadioChangeEvent) => onChange({ ...value, periodType: e.target.value as PeriodType })}>
        <Space orientation="vertical" style={{ width: "100%" }}>
          <Radio value="every">每日</Radio>

          <Radio value="specific">
            <Space align="center">
              <span>指定:</span>
              <Select
                mode="multiple"
                style={{ width: 300 }}
                value={value.specific || [1]}
                onChange={(values: number[]) => onChange({ ...value, periodType: "specific", specific: values })}
                options={options.map(v => ({ label: `${v}号`, value: v }))}
                placeholder="选择日期"
              />
            </Space>
          </Radio>

          <Radio value="cycle">
            <Space align="center">
              <span>周期:</span>
              <span>从</span>
              <Select
                style={{ width: 80 }}
                value={value.cycleStart ?? 1}
                onChange={(cycleStart: number) => onChange({ ...value, periodType: "cycle", cycleStart })}
               onSearch={() => {}}>
                {options.map(v => <Select.Option key={v} value={v}>{v}</Select.Option>)}
              </Select>
              <span>号开始，每</span>
              <Select
                style={{ width: 80 }}
                value={value.cycleInterval ?? 1}
                onChange={(cycleInterval: number) => onChange({ ...value, periodType: "cycle", cycleInterval })}
               onSearch={() => {}}>
                {Array.from({ length: 15 }, (_, i) => i + 1).map(v => (
                  <Select.Option key={v} value={v}>{v}</Select.Option>
                ))}
              </Select>
              <span>天</span>
            </Space>
          </Radio>

          <Radio value="range">
            <Space align="center">
              <span>范围:</span>
              <span>从</span>
              <Select
                style={{ width: 80 }}
                value={value.rangeStart ?? 1}
                onChange={(rangeStart: number) => onChange({ ...value, periodType: "range", rangeStart })}
               onSearch={() => {}}>
                {options.map(v => <Select.Option key={v} value={v}>{v}</Select.Option>)}
              </Select>
              <span>到</span>
              <Select
                style={{ width: 80 }}
                value={value.rangeEnd ?? 31}
                onChange={(rangeEnd: number) => onChange({ ...value, periodType: "range", rangeEnd })}
               onSearch={() => {}}>
                {options.map(v => <Select.Option key={v} value={v}>{v}</Select.Option>)}
              </Select>
              <span>号</span>
            </Space>
          </Radio>
        </Space>
      </Radio.Group>
    </div>
  );
};

export default DayField;
