/**
 * 周字段配置组件
 */

import React from 'react';
import { Radio, Select, Space, Checkbox } from 'antd';
import type { CronFieldConfig } from '../constants';

import { FIELD_RANGES, WEEK_DAY_NAMES } from '../constants';

interface WeekFieldProps {
  value: CronFieldConfig;
  onChange: (value: CronFieldConfig) => void;
}

const WeekField: React.FC<WeekFieldProps> = ({ value, onChange }) => {
  const { periodType } = value;
  const range = FIELD_RANGES.week;

  // 生成星期选项数组 (1=周日, 2=周一, ..., 7=周六)
  const weekOptions = [
    { label: '周日', value: 1 },
    { label: '周一', value: 2 },
    { label: '周二', value: 3 },
    { label: '周三', value: 4 },
    { label: '周四', value: 5 },
    { label: '周五', value: 6 },
    { label: '周六', value: 7 },
  ];

  return (
    <div>
      <Radio.Group value={periodType} onChange={(e) => onChange({ ...value, periodType: e.target.value })}>
        <Space orientation="vertical" style={{ width: '100%' }}>
          <Radio value="every">每周</Radio>

          <Radio value="specific">
            <Space align="center" orientation="vertical">
              <span>指定:</span>
              <Checkbox.Group
                value={value.specific || []}
                onChange={(values) => onChange({ ...value, periodType: 'specific', specific: values as number[] })}
                style={{ marginLeft: 24 }}
              >
                <Space wrap>
                  {weekOptions.map(opt => (
                    <Checkbox key={opt.value} value={opt.value}>{opt.label}</Checkbox>
                  ))}
                </Space>
              </Checkbox.Group>
            </Space>
          </Radio>

          <Radio value="cycle">
            <Space align="center">
              <span>周期:</span>
              <span>从</span>
              <Select
                style={{ width: 100 }}
                value={value.cycleStart ?? 1}
                onChange={(cycleStart) => onChange({ ...value, periodType: 'cycle', cycleStart })}
              >
                {weekOptions.map(opt => (
                  <Select.Option key={opt.value} value={opt.value}>{opt.label}</Select.Option>
                ))}
              </Select>
              <span>开始，每</span>
              <Select
                style={{ width: 80 }}
                value={value.cycleInterval ?? 1}
                onChange={(cycleInterval) => onChange({ ...value, periodType: 'cycle', cycleInterval })}
              >
                {Array.from({ length: 7 }, (_, i) => i + 1).map(v => (
                  <Select.Option key={v} value={v}>{v}</Select.Option>
                ))}
              </Select>
              <span>周</span>
            </Space>
          </Radio>

          <Radio value="range">
            <Space align="center">
              <span>范围:</span>
              <span>从</span>
              <Select
                style={{ width: 100 }}
                value={value.rangeStart ?? 1}
                onChange={(rangeStart) => onChange({ ...value, periodType: 'range', rangeStart })}
              >
                {weekOptions.map(opt => (
                  <Select.Option key={opt.value} value={opt.value}>{opt.label}</Select.Option>
                ))}
              </Select>
              <span>到</span>
              <Select
                style={{ width: 100 }}
                value={value.rangeEnd ?? 7}
                onChange={(rangeEnd) => onChange({ ...value, periodType: 'range', rangeEnd })}
              >
                {weekOptions.map(opt => (
                  <Select.Option key={opt.value} value={opt.value}>{opt.label}</Select.Option>
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
