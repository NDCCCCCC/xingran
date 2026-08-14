/**
 * 周字段配置组件
 */

import type { FC } from "react";
import type { RadioChangeEvent } from "antd";
import { Radio, Select, Space, Checkbox } from "antd";
import type { CronFieldConfig, PeriodType } from "../constants";

interface WeekFieldProps {
  value: CronFieldConfig;
  onChange: (value: CronFieldConfig) => void;
}

const WeekField: FC<WeekFieldProps> = ({ value, onChange }) => {
  const { periodType } = value;

  // 生成星期选项数组 (1=周日, 2=周一, ..., 7=周六)
  const weekOptions = [
    { label: "周日", value: 1 },
    { label: "周一", value: 2 },
    { label: "周二", value: 3 },
    { label: "周三", value: 4 },
    { label: "周四", value: 5 },
    { label: "周五", value: 6 },
    { label: "周六", value: 7 },
  ];

  return (
    <div className="cron-selector-week-field">
      <Radio.Group
        value={periodType}
        onChange={(e: RadioChangeEvent) =>
          onChange({ ...value, periodType: e.target.value as PeriodType })
        }
      >
        <Space orientation="vertical" style={{ width: "100%" }}>
          <Radio value="every">每周</Radio>

          <Radio value="specific">
            <Space align="center" orientation="vertical">
              <span>指定:</span>
              <Checkbox.Group
                value={value.specific || []}
                onChange={(values: number[]) =>
                  onChange({ ...value, periodType: "specific", specific: values })
                }
                style={{ marginLeft: 24, display: "flex", flexWrap: "wrap", gap: "8px" }}
              >
                {weekOptions.map((opt) => (
                  <Checkbox key={opt.value} value={opt.value}>
                    {opt.label}
                  </Checkbox>
                ))}
              </Checkbox.Group>
            </Space>
          </Radio>

          <Radio value="cycle">
            <Space align="center" wrap>
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
                {weekOptions.map((opt) => (
                  <Select.Option key={opt.value} value={opt.value}>
                    {opt.label}
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
                {Array.from({ length: 7 }, (_, i) => i + 1).map((v) => (
                  <Select.Option key={v} value={v}>
                    {v}
                  </Select.Option>
                ))}
              </Select>
              <span>周</span>
            </Space>
          </Radio>

          <Radio value="range">
            <Space align="center" wrap>
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
                {weekOptions.map((opt) => (
                  <Select.Option key={opt.value} value={opt.value}>
                    {opt.label}
                  </Select.Option>
                ))}
              </Select>
              <span>到</span>
              <Select
                style={{ width: 100 }}
                value={value.rangeEnd ?? 7}
                onChange={(rangeEnd: number) =>
                  onChange({ ...value, periodType: "range", rangeEnd })
                }
                onSearch={() => {}}
              >
                {weekOptions.map((opt) => (
                  <Select.Option key={opt.value} value={opt.value}>
                    {opt.label}
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

export default WeekField;
