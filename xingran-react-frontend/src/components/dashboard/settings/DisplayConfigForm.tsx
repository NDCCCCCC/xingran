/**
 * DisplayConfigForm - 显示配置表单
 *
 * 根据 Widget 类型渲染对应的显示配置表单
 * 支持 stat-card、chart、table、list、progress 五种类型
 */

import { useState, useEffect } from "react";
import { Form, Input, InputNumber, Select, Switch, ColorPicker, Space, Divider } from "antd";
import { QuestionCircleOutlined } from "@ant-design/icons";
import type {
  DisplayConfig,
  WidgetType,
  StatCardDisplayConfig,
  ChartDisplayConfig,
  TableDisplayConfig,
  ListDisplayConfig,
  ProgressDisplayConfig,
} from "@/types/dashboard";
import type { FormInstance } from "antd";

export interface DisplayConfigFormProps {
  /** 当前显示配置 */
  value?: DisplayConfig;
  /** 配置变化回调 */
  onChange?: (value: DisplayConfig) => void;
  /** Widget 类型 */
  widgetType: WidgetType;
  /** 表单实例 */
  form?: FormInstance;
  /** 是否禁用 */
  disabled?: boolean;
}

/**
 * 获取默认显示配置
 */
const getDefaultDisplayConfig = (widgetType: WidgetType): DisplayConfig => {
  switch (widgetType) {
    case "stat-card":
      return {
        type: "stat-card",
        icon: "📊",
        iconColor: "var(--theme-info, #1890ff)",
        decimals: 0,
        showTrend: false,
      } as StatCardDisplayConfig;
    case "chart":
      return {
        type: "chart",
        chartType: "line",
        showLegend: true,
        showLabels: false,
        smooth: true,
      } as ChartDisplayConfig;
    case "table":
      return {
        type: "table",
        columns: [],
        bordered: true,
        pagination: { enabled: true, pageSize: 10 },
      } as TableDisplayConfig;
    case "list":
      return {
        type: "list",
        titleField: "title",
        maxItems: 10,
        showIndex: false,
      } as ListDisplayConfig;
    case "progress":
      return {
        type: "progress",
        progressType: "line",
        target: 100,
      } as ProgressDisplayConfig;
    case "metric":
      return {
        type: "progress",
        progressType: "circle",
        target: 100,
      } as ProgressDisplayConfig;
    default:
      return {
        type: "stat-card",
        icon: "📊",
        iconColor: "var(--theme-info, #1890ff)",
      } as StatCardDisplayConfig;
  }
};

/**
 * 显示配置表单组件
 */
export const DisplayConfigForm: React.FC<DisplayConfigFormProps> = ({
  value,
  onChange,
  widgetType,
  form,
  disabled = false,
}) => {
  const [localForm] = Form.useForm();
  const activeForm = form || localForm;

  // 当外部 value 或 widgetType 变化时同步表单
  useEffect(() => {
    const config = value || getDefaultDisplayConfig(widgetType);
    activeForm.setFieldsValue(config);
  }, [value, widgetType, activeForm]);

  // 构建并发送配置变化
  const emitChange = () => {
    const values = activeForm.getFieldsValue();
    const config: DisplayConfig = {
      ...values,
      type: widgetType === "metric" ? "progress" : widgetType,
    } as DisplayConfig;
    onChange?.(config);
  };

  // 渲染统计卡片配置
  const renderStatCardConfig = () => (
    <>
      <Form.Item label="前缀" name="prefix" tooltip="数值前显示的文本">
        <Input placeholder="¥" disabled={disabled} />
      </Form.Item>
      <Form.Item label="后缀" name="suffix" tooltip="数值后显示的文本">
        <Input placeholder="%" disabled={disabled} />
      </Form.Item>
      <Form.Item label="小数位数" name="decimals" tooltip="保留的小数位数" initialValue={0}>
        <InputNumber min={0} max={10} disabled={disabled} style={{ width: "100%" }} />
      </Form.Item>
      <Form.Item label="图标" name="icon" tooltip="显示的图标（emoji 或图标名称）">
        <Input placeholder="📊" disabled={disabled} />
      </Form.Item>
      <Form.Item label="图标颜色" name="iconColor" tooltip="图标的颜色">
        <ColorPicker disabled={disabled} showText />
      </Form.Item>
      <Form.Item
        label="显示趋势"
        name="showTrend"
        tooltip="显示与上一周期的对比趋势"
        valuePropName="checked"
        initialValue={false}
      >
        <Switch disabled={disabled} />
      </Form.Item>
    </>
  );

  // 渲染图表配置
  const renderChartConfig = () => (
    <>
      <Form.Item label="图表类型" name="chartType" tooltip="选择图表的展示形式" initialValue="line">
        <Select disabled={disabled} onSearch={() => {}}>
          <Select.Option value="line">折线图</Select.Option>
          <Select.Option value="bar">柱状图</Select.Option>
          <Select.Option value="pie">饼图</Select.Option>
          <Select.Option value="area">面积图</Select.Option>
          <Select.Option value="gauge">仪表盘</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item label="X 轴字段" name="xField" tooltip="X 轴对应的数据字段">
        <Input placeholder="date" disabled={disabled} />
      </Form.Item>
      <Form.Item label="Y 轴字段" name="yField" tooltip="Y 轴对应的数据字段">
        <Input placeholder="value" disabled={disabled} />
      </Form.Item>
      <Form.Item label="系列字段" name="seriesField" tooltip="用于区分多系列的字段">
        <Input placeholder="category" disabled={disabled} />
      </Form.Item>
      <Form.Item
        label="显示图例"
        name="showLegend"
        tooltip="是否显示图例说明"
        valuePropName="checked"
        initialValue={true}
      >
        <Switch disabled={disabled} />
      </Form.Item>
      <Form.Item
        label="显示数据标签"
        name="showLabels"
        tooltip="在图表上显示数据值"
        valuePropName="checked"
        initialValue={false}
      >
        <Switch disabled={disabled} />
      </Form.Item>
      <Form.Item
        label="平滑曲线"
        name="smooth"
        tooltip="折线是否平滑显示"
        valuePropName="checked"
        initialValue={true}
      >
        <Switch disabled={disabled} />
      </Form.Item>
    </>
  );

  // 渲染表格配置
  const renderTableConfig = () => (
    <>
      <Form.Item
        label="显示边框"
        name="bordered"
        tooltip="是否显示表格边框"
        valuePropName="checked"
        initialValue={true}
      >
        <Switch disabled={disabled} />
      </Form.Item>
      <Form.Item label="表格大小" name="size" tooltip="表格行高大小" initialValue="middle">
        <Select disabled={disabled} onSearch={() => {}}>
          <Select.Option value="small">紧凑</Select.Option>
          <Select.Option value="middle">默认</Select.Option>
          <Select.Option value="large">宽松</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item label="启用分页" tooltip="是否启用分页功能">
        <Space>
          <Form.Item
            name={["pagination", "enabled"]}
            valuePropName="checked"
            noStyle
            initialValue={true}
          >
            <Switch disabled={disabled} />
          </Form.Item>
          <Form.Item name={["pagination", "pageSize"]} noStyle initialValue={10}>
            <InputNumber
              min={5}
              max={100}
              placeholder="每页数量"
              disabled={disabled}
              style={{ width: 100 }}
            />
          </Form.Item>
          <span style={{ color: "var(--theme-text-tertiary, #999)" }}>条/页</span>
        </Space>
      </Form.Item>
      <Divider titlePlacement="left" plain>
        列配置
      </Divider>
      <Form.List name="columns">
        {(fields, { add, remove }) => (
          <>
            {fields.map(({ key, name, ...restField }) => (
              <Space key={key} style={{ display: "flex", marginBottom: 8 }} align="baseline">
                <Form.Item
                  {...restField}
                  name={[name, "dataIndex"]}
                  rules={[{ required: true, message: "请输入字段名" }]}
                >
                  <Input placeholder="字段名" disabled={disabled} style={{ width: 120 }} />
                </Form.Item>
                <Form.Item
                  {...restField}
                  name={[name, "title"]}
                  rules={[{ required: true, message: "请输入列标题" }]}
                >
                  <Input placeholder="列标题" disabled={disabled} style={{ width: 120 }} />
                </Form.Item>
                <Form.Item {...restField} name={[name, "width"]}>
                  <InputNumber placeholder="宽度" disabled={disabled} style={{ width: 80 }} />
                </Form.Item>
              </Space>
            ))}
            <Form.Item>
              <Select
                disabled={disabled}
                placeholder="添加列..."
                onChange={(value) => {
                  add({ dataIndex: value, title: value });
                }}
                value={undefined}
                options={[
                  { value: "id", label: "ID" },
                  { value: "name", label: "名称" },
                  { value: "status", label: "状态" },
                  { value: "createTime", label: "创建时间" },
                ]}
              />
            </Form.Item>
          </>
        )}
      </Form.List>
    </>
  );

  // 渲染列表配置
  const renderListConfig = () => (
    <>
      <Form.Item
        label="标题字段"
        name="titleField"
        tooltip="列表项标题对应的数据字段"
        rules={[{ required: true, message: "请输入标题字段" }]}
        initialValue="title"
      >
        <Input placeholder="title" disabled={disabled} />
      </Form.Item>
      <Form.Item label="描述字段" name="descriptionField" tooltip="列表项描述对应的数据字段">
        <Input placeholder="description" disabled={disabled} />
      </Form.Item>
      <Form.Item label="时间字段" name="timeField" tooltip="列表项时间对应的数据字段">
        <Input placeholder="createTime" disabled={disabled} />
      </Form.Item>
      <Form.Item label="图标字段" name="iconField" tooltip="列表项图标对应的数据字段">
        <Input placeholder="icon" disabled={disabled} />
      </Form.Item>
      <Form.Item
        label="最大显示数量"
        name="maxItems"
        tooltip="最多显示多少条数据"
        initialValue={10}
      >
        <InputNumber min={1} max={100} disabled={disabled} style={{ width: "100%" }} />
      </Form.Item>
      <Form.Item
        label="显示序号"
        name="showIndex"
        tooltip="是否显示列表序号"
        valuePropName="checked"
        initialValue={false}
      >
        <Switch disabled={disabled} />
      </Form.Item>
    </>
  );

  // 渲染进度条配置
  const renderProgressConfig = () => (
    <>
      <Form.Item
        label="进度类型"
        name="progressType"
        tooltip="进度条的展示形式"
        initialValue="line"
      >
        <Select disabled={disabled} onSearch={() => {}}>
          <Select.Option value="line">线性进度条</Select.Option>
          <Select.Option value="circle">圆形进度</Select.Option>
          <Select.Option value="dashboard">仪表盘</Select.Option>
        </Select>
      </Form.Item>
      <Form.Item label="目标值" name="target" tooltip="用于计算百分比的目标值" initialValue={100}>
        <InputNumber min={1} disabled={disabled} style={{ width: "100%" }} />
      </Form.Item>
      <Divider titlePlacement="left" plain>
        颜色阈值
      </Divider>
      <Form.List name="colorThresholds">
        {(fields, { add, remove }) => (
          <>
            {fields.map(({ key, name, ...restField }) => (
              <Space key={key} style={{ display: "flex", marginBottom: 8 }} align="baseline">
                <Form.Item {...restField} name={[name, "value"]}>
                  <InputNumber placeholder="阈值" disabled={disabled} style={{ width: 100 }} />
                </Form.Item>
                <Form.Item {...restField} name={[name, "color"]}>
                  <ColorPicker disabled={disabled} showText />
                </Form.Item>
              </Space>
            ))}
            <Form.Item>
              <Select
                disabled={disabled}
                placeholder="添加阈值..."
                onChange={() => {
                  add({ value: 50, color: "var(--theme-warning, #faad14)" });
                }}
                value={undefined}
                options={[{ value: "add", label: "添加颜色阈值" }]}
              />
            </Form.Item>
          </>
        )}
      </Form.List>
    </>
  );

  // 根据 Widget 类型渲染对应的配置表单
  const renderConfigForm = () => {
    switch (widgetType) {
      case "stat-card":
        return renderStatCardConfig();
      case "chart":
        return renderChartConfig();
      case "table":
        return renderTableConfig();
      case "list":
        return renderListConfig();
      case "progress":
      case "metric":
        return renderProgressConfig();
      default:
        return (
          <div
            style={{ color: "var(--theme-text-tertiary, #999)", textAlign: "center", padding: 20 }}
          >
            暂无额外配置项
          </div>
        );
    }
  };

  return (
    <div className="display-config-form">
      <Form form={activeForm} layout="vertical" onValuesChange={emitChange}>
        {renderConfigForm()}
      </Form>
    </div>
  );
};

export default DisplayConfigForm;
