/**
 * Cron表达式选择器主组件
 *
 * 布局:
 * - 顶部: 输入框(左) + 预设下拉框(右)
 * - 下方: 可视化配置(Tab切换秒/分/时/日/月/周)
 */

import {
  useState,
  useLayoutEffect,
  useImperativeHandle,
  forwardRef,
  useRef,
  useEffect,
} from "react";
import { Input, Select, Tabs, Alert, Space, Row, Col } from "antd";
import { InfoCircleOutlined, ClockCircleOutlined } from "@ant-design/icons";
import type { CronConfig, CronFieldType } from "./constants";
import {
  expressionToCronConfig,
  cronConfigToExpression,
  validateCronExpression,
  getDefaultCronConfig,
  cronToChinese,
  getNextRunTimes,
} from "./utils";
import { DEFAULT_CRON_EXPRESSION, CRON_PRESETS, FIELD_LABELS } from "./constants";
import SecondField from "./fields/SecondField";
import MinuteField from "./fields/MinuteField";
import HourField from "./fields/HourField";
import DayField from "./fields/DayField";
import MonthField from "./fields/MonthField";
import WeekField from "./fields/WeekField";
import dayjs from "dayjs";
import utc from "dayjs/plugin/utc";

dayjs.extend(utc);

const formatTime = (date: Date) => {
  return dayjs(date).format("YYYY-MM-DD HH:mm:ss");
};

const FIELD_ORDER: CronFieldType[] = ["second", "minute", "hour", "day", "month", "week"];

export interface CronSelectorRef {
  getExpression: () => string;
  validate: () => boolean;
}

export interface CronSelectorProps {
  value?: string;
  onChange?: (value: string) => void;
  disabled?: boolean;
  style?: React.CSSProperties;
  className?: string;
}

const CronSelector = forwardRef<CronSelectorRef, CronSelectorProps>(
  ({ value = "", onChange, disabled, style, className }, ref) => {
    // 内部状态
    const [cronExpression, setCronExpression] = useState(value || DEFAULT_CRON_EXPRESSION);
    const [cronConfig, setCronConfig] = useState<CronConfig>(
      value && validateCronExpression(value)
        ? expressionToCronConfig(value)
        : getDefaultCronConfig()
    );
    const [activeField, setActiveField] = useState<CronFieldType>("minute");
    const [initialized, setInitialized] = useState(false);

    // 使用 ref 存储稳定的函数引用，遵循 Vercel React Best Practices
    const validateRef = useRef(validateCronExpression);
    const toConfigRef = useRef(expressionToCronConfig);
    const toExpressionRef = useRef(cronConfigToExpression);
    const onChangeRef = useRef(onChange);

    useEffect(() => {
      validateRef.current = validateCronExpression;
      toConfigRef.current = expressionToCronConfig;
      toExpressionRef.current = cronConfigToExpression;
      onChangeRef.current = onChange;
    });

    // 同步外部value变化 - 使用 ref 避免依赖问题
    useLayoutEffect(() => {
      if (value !== cronExpression) {
        setCronExpression(value || DEFAULT_CRON_EXPRESSION);
        if (validateRef.current(value)) {
          setCronConfig(toConfigRef.current(value));
        }
      }
    }, [value, cronExpression]);

    // 初始化时，如果没有外部值，使用默认值并触发onChange
    useLayoutEffect(() => {
      if (!initialized) {
        setInitialized(true);
        if (!value) {
          // 使用默认配置生成表达式
          const defaultConfig = getDefaultCronConfig();
          const defaultExpression = toExpressionRef.current(defaultConfig);
          setCronConfig(defaultConfig);
          setCronExpression(defaultExpression);
          // 触发onChange，将默认值传递给表单
          onChangeRef.current?.(defaultExpression);
        }
      }
    }, [initialized, value]);

    // 暴露给父组件的方法
    useImperativeHandle(ref, () => ({
      getExpression: () => cronExpression,
      validate: () => validateCronExpression(cronExpression),
    }));

    // 处理表达式输入变化
    const handleExpressionInputChange = (newValue: string) => {
      setCronExpression(newValue);

      // 如果是有效的表达式，解析为配置对象并触发onChange
      if (validateCronExpression(newValue)) {
        const newConfig = expressionToCronConfig(newValue);
        setCronConfig(newConfig);
        onChange?.(newValue);
      }
    };

    // 处理预设选择
    const handlePresetSelect = (presetValue: string) => {
      setCronExpression(presetValue);
      if (validateCronExpression(presetValue)) {
        const newConfig = expressionToCronConfig(presetValue);
        setCronConfig(newConfig);
        onChange?.(presetValue);
      }
    };

    // 处理配置对象变化
    const handleConfigChange = (
      fieldKey: keyof CronConfig,
      fieldValue: CronConfig[keyof CronConfig]
    ) => {
      const newConfig = { ...cronConfig, [fieldKey]: fieldValue };
      setCronConfig(newConfig);
      const newExpression = cronConfigToExpression(newConfig);
      setCronExpression(newExpression);
      onChange?.(newExpression);
    };

    // 渲染字段组件
    const renderField = (fieldType: CronFieldType) => {
      const fieldConfig = cronConfig[fieldType];
      const fieldProps = {
        value: fieldConfig,
        onChange: (newValue: typeof fieldConfig) => handleConfigChange(fieldType, newValue),
      };

      switch (fieldType) {
        case "second":
          return <SecondField key="second" {...fieldProps} />;
        case "minute":
          return <MinuteField key="minute" {...fieldProps} />;
        case "hour":
          return <HourField key="hour" {...fieldProps} />;
        case "day":
          return <DayField key="day" {...fieldProps} />;
        case "month":
          return <MonthField key="month" {...fieldProps} />;
        case "week":
          return <WeekField key="week" {...fieldProps} />;
        default:
          return null;
      }
    };

    const isValid = validateCronExpression(cronExpression);
    const chineseDesc = cronToChinese(cronExpression);

    return (
      <div style={style} className={className}>
        {/* 顶部输入框 + 预设下拉框 */}
        <Row gutter={8} style={{ marginBottom: 12 }}>
          <Col flex="1">
            <Input
              placeholder="请输入Cron表达式，如: 0 0 9 * * ?"
              value={cronExpression}
              onChange={(e) => handleExpressionInputChange(e.target.value)}
              disabled={disabled}
              status={!isValid && cronExpression ? "error" : undefined}
            />
          </Col>
          <Col style={{ width: 200 }}>
            <Select
              placeholder="选择预设"
              value={CRON_PRESETS.find((p) => p.value === cronExpression)?.value}
              onChange={handlePresetSelect}
              disabled={disabled}
              allowClear
              showSearch
              optionFilterProp="label"
              style={{ width: "100%" }}
              onSearch={() => {}}
            >
              {CRON_PRESETS.map((preset) => (
                <Select.Option key={preset.value} value={preset.value}>
                  {preset.label}
                  <span
                    style={{
                      color: "var(--theme-text-tertiary, #999)",
                      fontSize: 12,
                      marginLeft: 8,
                    }}
                  >
                    ({preset.value})
                  </span>
                </Select.Option>
              ))}
            </Select>
          </Col>
        </Row>

        {/* 中文描述 */}
        {isValid && (
          <div style={{ marginBottom: 12, color: "var(--theme-info, #1890ff)", fontSize: 13 }}>
            {chineseDesc}
          </div>
        )}

        {/* 字段配置 Tab */}
        <Tabs
          activeKey={activeField}
          onChange={(key) => setActiveField(key as CronFieldType)}
          size="small"
          items={FIELD_ORDER.map((fieldType) => ({
            key: fieldType,
            label: FIELD_LABELS[fieldType],
            children: renderField(fieldType),
          }))}
          className="cron-selector-tabs"
          style={{
            overflow: "hidden",
          }}
          tabBarStyle={{
            marginBottom: 12,
          }}
        />

        {/* 最近五次执行时间 */}
        {isValid && (
          <div style={{ marginTop: 16 }}>
            <Space orientation="vertical" style={{ width: "100%" }} size="small">
              <div style={{ display: "flex", alignItems: "center", gap: 8 }}>
                <ClockCircleOutlined style={{ color: "var(--theme-info, #1890ff)" }} />
                <span style={{ fontWeight: 500 }}>最近五次执行时间</span>
              </div>
              <div style={{ background: "#fafafa", padding: "8px 16px", borderRadius: 4 }}>
                {getNextRunTimes(cronExpression, 5).map((item, index) => (
                  <div key={index} style={{ padding: "4px 0" }}>
                    <Space>
                      <span style={{ color: "var(--theme-text-tertiary, #999)", minWidth: 60 }}>
                        第{index + 1}次
                      </span>
                      <span style={{ fontFamily: "monospace", color: "#333" }}>
                        {formatTime(item)}
                      </span>
                    </Space>
                  </div>
                ))}
              </div>
            </Space>
          </div>
        )}

        {/* 错误提示 */}
        {!isValid && cronExpression && (
          <Alert
            title="无效的Cron表达式"
            description="请检查表达式格式是否正确"
            type="error"
            showIcon
            icon={<InfoCircleOutlined />}
            style={{ marginTop: 12 }}
          />
        )}
      </div>
    );
  }
);

CronSelector.displayName = "CronSelector";

export default CronSelector;
