/**
 * 秒字段配置组件
 */

import type { FC } from "react";
import type { RadioChangeEvent } from "antd";
import { Radio, Space } from "antd";
import { AppSelect } from "@/components/ui";
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
              <AppSelect
                mode="multiple"
                style={{ width: 300 }}
                value={value.specific || [0]}
                onChange={(values: unknown) =>
                  onChange({ ...value, periodType: "specific", specific: values as number[] })
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
              <AppSelect
                style={{ width: 80 }}
                value={value.cycleStart ?? 0}
                onChange={(cycleStart: unknown) =>
                  onChange({ ...value, periodType: "cycle", cycleStart: cycleStart as number })
                }
                options={options.map((v) => ({ value: v, label: v }))}
              />
              <span>秒开始，每</span>
              <AppSelect
                style={{ width: 80 }}
                value={value.cycleInterval ?? 1}
                onChange={(cycleInterval: unknown) =>
                  onChange({ ...value, periodType: "cycle", cycleInterval: cycleInterval as number })
                }
                options={Array.from({ length: Math.min(20, range.max) }, (_, i) => i + 1).map((v) => ({
                  value: v,
                  label: v,
                }))}
              />
              <span>秒</span>
            </Space>
          </Radio>

          <Radio value="range">
            <Space align="center">
              <span>范围:</span>
              <span>从</span>
              <AppSelect
                style={{ width: 80 }}
                value={value.rangeStart ?? 0}
                onChange={(rangeStart: unknown) =>
                  onChange({ ...value, periodType: "range", rangeStart: rangeStart as number })
                }
                options={options.map((v) => ({ value: v, label: v }))}
              />
              <span>到</span>
              <AppSelect
                style={{ width: 80 }}
                value={value.rangeEnd ?? 59}
                onChange={(rangeEnd: unknown) =>
                  onChange({ ...value, periodType: "range", rangeEnd: rangeEnd as number })
                }
                options={options.map((v) => ({ value: v, label: v }))}
              />
              <span>秒</span>
            </Space>
          </Radio>
        </Space>
      </Radio.Group>
    </div>
  );
};

export default SecondField;
