/**
 * 秒字段配置组件
 */

import type { FC } from "react";
import type { RadioChangeEvent } from "antd";
import { Radio, Select, Space } from "antd";
import type { CronFieldConfig, PeriodType } from "../constants";
import { FIELD_RANGES } from "../constants";

interface SecondFieldProps {
  value: CronFieldConfig;
  onChange: (value: CronFieldConfig) => void;
}

const SecondField: FC<SecondFieldProps> = ({ value, onChange }) => {
  const { periodType } = value;
  const range = FIELD_RANGES.second;

  // 生成秒选项数组
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
          <Radio value="every">每秒</Radio>

          <Radio value="specific">
            <Space align="center">
              <span>指定:</span>
              <Select
                mode="multiple"
                style={{ width: 300 }}
                value={value.specific || [0]}
                onChange={(values: number[]) =>
                  onChange({ ...value, periodType: "specific", specific: values })
                }
                options={options.map((v) => ({ label: `${v}秒`, value: v }))}
                placeholder="选择秒"
              />
            </Space>
          </Radio>

          <Radio value="cycle">
            <Space align="center">
              <span>周期:</span>
              <span>从</span>
              <Select
                style={{ width: 80 }}
                value={value.cycleStart ?? 0}
                onChange={(cycleStart: number) =>
                  onChange({ ...value, periodType: "cycle", cycleStart })
                }
                onSearch={() => {}}
              >
                {options.map((v) => (
                  <Select.Option key={v} value={v}>
                    {v}
                  </Select.Option>
                ))}
              </Select>
              <span>秒开始，每</span>
              <Select
                style={{ width: 80 }}
                value={value.cycleInterval ?? 1}
                onChange={(cycleInterval: number) =>
                  onChange({ ...value, periodType: "cycle", cycleInterval })
                }
                onSearch={() => {}}
              >
                {Array.from({ length: Math.min(20, range.max) }, (_, i) => i + 1).map((v) => (
                  <Select.Option key={v} value={v}>
                    {v}
                  </Select.Option>
                ))}
              </Select>
              <span>秒</span>
            </Space>
          </Radio>

          <Radio value="range">
            <Space align="center">
              <span>范围:</span>
              <span>从</span>
              <Select
                style={{ width: 80 }}
                value={value.rangeStart ?? 0}
                onChange={(rangeStart: number) =>
                  onChange({ ...value, periodType: "range", rangeStart })
                }
                onSearch={() => {}}
              >
                {options.map((v) => (
                  <Select.Option key={v} value={v}>
                    {v}
                  </Select.Option>
                ))}
              </Select>
              <span>到</span>
              <Select
                style={{ width: 80 }}
                value={value.rangeEnd ?? 59}
                onChange={(rangeEnd: number) =>
                  onChange({ ...value, periodType: "range", rangeEnd })
                }
                onSearch={() => {}}
              >
                {options.map((v) => (
                  <Select.Option key={v} value={v}>
                    {v}
                  </Select.Option>
                ))}
              </Select>
              <span>秒</span>
            </Space>
          </Radio>
        </Space>
      </Radio.Group>
    </div>
  );
};

export default SecondField;
