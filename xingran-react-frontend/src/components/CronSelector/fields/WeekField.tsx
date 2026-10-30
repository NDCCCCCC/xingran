/**
 * 周字段配置组件
 */

import type { FC } from "react";
import type { RadioChangeEvent } from "antd";
import { Radio, Space, Checkbox } from "antd";
import { AppSelect } from "@/components/ui";
import type { CronFieldConfig, PeriodType } from "../constants";

interface WeekFieldProps {
  value: CronFieldConfig;
  onChange: (value: CronFieldConfig) => void;
}

// 模块级静态选项（Phase m76-js-misc 6.3）
const WEEK_OPTIONS = [
  { label: "周日", value: 1 },
  { label: "周一", value: 2 },
  { label: "周二", value: 3 },
  { label: "周三", value: 4 },
  { label: "周四", value: 5 },
  { label: "周五", value: 6 },
  { label: "周六", value: 7 },
];

const WeekField: FC<WeekFieldProps> = ({ value, onChange }) => {
  const { periodType } = value;

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
                onChange={(values: unknown) =>
                  onChange({ ...value, periodType: "specific", specific: values as number[] })
                }
                style={{ marginLeft: 24, display: "flex", flexWrap: "wrap", gap: "8px" }}
              >
                {WEEK_OPTIONS.map((opt) => (
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
              <AppSelect
                style={{ width: 100 }}
                value={value.cycleStart ?? 1}
                onChange={(cycleStart: unknown) =>
                  onChange({ ...value, periodType: "cycle", cycleStart: cycleStart as number })
                }
                options={WEEK_OPTIONS}
              />
              <span>开始，每</span>
              <AppSelect
                style={{ width: 80 }}
                value={value.cycleInterval ?? 1}
                onChange={(cycleInterval: unknown) =>
                  onChange({
                    ...value,
                    periodType: "cycle",
                    cycleInterval: cycleInterval as number,
                  })
                }
                options={Array.from({ length: 7 }, (_, i) => i + 1).map((v) => ({
                  value: v,
                  label: v,
                }))}
              />
              <span>周</span>
            </Space>
          </Radio>

          <Radio value="range">
            <Space align="center" wrap>
              <span>范围:</span>
              <span>从</span>
              <AppSelect
                style={{ width: 100 }}
                value={value.rangeStart ?? 1}
                onChange={(rangeStart: unknown) =>
                  onChange({ ...value, periodType: "range", rangeStart: rangeStart as number })
                }
                options={WEEK_OPTIONS}
              />
              <span>到</span>
              <AppSelect
                style={{ width: 100 }}
                value={value.rangeEnd ?? 7}
                onChange={(rangeEnd: unknown) =>
                  onChange({ ...value, periodType: "range", rangeEnd: rangeEnd as number })
                }
                options={WEEK_OPTIONS}
              />
            </Space>
          </Radio>
        </Space>
      </Radio.Group>
    </div>
  );
};

export default WeekField;
