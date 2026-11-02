/**
 * 月字段配置组件
 */

import type { FC } from "react";
import type { RadioChangeEvent } from "antd";
import { Radio, Space } from "antd";
import { AppSelect } from "@/components/ui";
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
              <AppSelect
                mode="multiple"
                style={{ width: 300 }}
                value={value.specific || [1]}
                onChange={(values: unknown) =>
                  onChange({ ...value, periodType: "specific", specific: values as number[] })
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
              <AppSelect
                style={{ width: 100 }}
                value={value.cycleStart ?? 1}
                onChange={(cycleStart: unknown) =>
                  onChange({ ...value, periodType: "cycle", cycleStart: cycleStart as number })
                }
                options={options.map((v) => ({ value: v, label: MONTH_NAMES[v - 1] }))}
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
                options={Array.from({ length: 6 }, (_, i) => i + 1).map((v) => ({
                  value: v,
                  label: v,
                }))}
              />
              <span>个月</span>
            </Space>
          </Radio>

          <Radio value="range">
            <Space align="center">
              <span>范围:</span>
              <span>从</span>
              <AppSelect
                style={{ width: 100 }}
                value={value.rangeStart ?? 1}
                onChange={(rangeStart: unknown) =>
                  onChange({ ...value, periodType: "range", rangeStart: rangeStart as number })
                }
                options={options.map((v) => ({ value: v, label: MONTH_NAMES[v - 1] }))}
              />
              <span>到</span>
              <AppSelect
                style={{ width: 100 }}
                value={value.rangeEnd ?? 12}
                onChange={(rangeEnd: unknown) =>
                  onChange({ ...value, periodType: "range", rangeEnd: rangeEnd as number })
                }
                options={options.map((v) => ({ value: v, label: MONTH_NAMES[v - 1] }))}
              />
            </Space>
          </Radio>
        </Space>
      </Radio.Group>
    </div>
  );
};

export default MonthField;
