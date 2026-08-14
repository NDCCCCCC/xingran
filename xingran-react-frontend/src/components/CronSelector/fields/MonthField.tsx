/**
 * 月字段配置组件
 */

import type { FC } from "react";
import type { RadioChangeEvent } from "antd";
import { Radio, Select, Space } from "antd";
import type { CronFieldConfig, PeriodType } from "../constants";
import { FIELD_RANGES, MONTH_NAMES } from "../constants";

interface MonthFieldProps {
  value: CronFieldConfig;
  onChange: (value: CronFieldConfig) => void;
}

const MonthField: FC<MonthFieldProps> = ({ value, onChange }) => {
  const { periodType } = value;
  const range = FIELD_RANGES.month;

  // 生成月份选项数组
  const options = Array.from({ length: range.max - range.min + 1 }, (_, i) => range.min + i);

  return (
    <div>
      <Radio.Group
        value={periodType}
        onChange={(e: RadioChangeEvent) =>
          onChange({ ...value, periodType: e.target.value as PeriodType })
        }
      >
        <Space orientation="vertical" style={{ width: "100%" }}>
          <Radio value="every">每月</Radio>

          <Radio value="specific">
            <Space align="center">
              <span>指定:</span>
              {/* eslint-disable-next-line local/no-large-dropdown-list -- fixed option list, no server search needed */}
              <Select
                mode="multiple"
                style={{ width: 300 }}
                value={value.specific || [1]}
                onChange={(values: number[]) =>
                  onChange({ ...value, periodType: "specific", specific: values })
                }
                options={options.map((v) => ({ label: MONTH_NAMES[v - 1], value: v }))}
                placeholder="选择月份"
              />
            </Space>
          </Radio>

          <Radio value="cycle">
            <Space align="center">
              <span>周期:</span>
              <span>从</span>
              <Select
                style={{ width: 100 }}
                value={value.cycleStart ?? 1}
                onChange={(cycleStart: number) =>
                  onChange({ ...value, periodType: "cycle", cycleStart })
                }
                onSearch={() => {}}
              >
                {options.map((v) => (
                  <Select.Option key={v} value={v}>
                    {MONTH_NAMES[v - 1]}
                  </Select.Option>
                ))}
              </Select>
              <span>开始，每</span>
              <Select
                style={{ width: 80 }}
                value={value.cycleInterval ?? 1}
                onChange={(cycleInterval: number) =>
                  onChange({ ...value, periodType: "cycle", cycleInterval })
                }
                onSearch={() => {}}
              >
                {Array.from({ length: 6 }, (_, i) => i + 1).map((v) => (
                  <Select.Option key={v} value={v}>
                    {v}
                  </Select.Option>
                ))}
              </Select>
              <span>个月</span>
            </Space>
          </Radio>

          <Radio value="range">
            <Space align="center">
              <span>范围:</span>
              <span>从</span>
              <Select
                style={{ width: 100 }}
                value={value.rangeStart ?? 1}
                onChange={(rangeStart: number) =>
                  onChange({ ...value, periodType: "range", rangeStart })
                }
                onSearch={() => {}}
              >
                {options.map((v) => (
                  <Select.Option key={v} value={v}>
                    {MONTH_NAMES[v - 1]}
                  </Select.Option>
                ))}
              </Select>
              <span>到</span>
              <Select
                style={{ width: 100 }}
                value={value.rangeEnd ?? 12}
                onChange={(rangeEnd: number) =>
                  onChange({ ...value, periodType: "range", rangeEnd })
                }
                onSearch={() => {}}
              >
                {options.map((v) => (
                  <Select.Option key={v} value={v}>
                    {MONTH_NAMES[v - 1]}
                  </Select.Option>
                ))}
              </Select>
            </Space>
          </Radio>
        </Space>
      </Radio.Group>
    </div>
  );
};

export default MonthField;
