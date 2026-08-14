/**
 * DataSourceForm - 数据源配置表单
 *
 * 根据数据源类型（api/websocket/static）动态渲染不同的配置表单
 * 支持类型切换时保留公共字段（transform 配置）
 */

import { useState, useEffect } from "react";
import { Form, Select, Input, InputNumber, Space, Alert } from "antd";
import { InfoCircleOutlined } from "@ant-design/icons";
import EndpointSelector from "./EndpointSelector";
import ParamsEditor from "./ParamsEditor";
import type {
  DataSourceConfig,
  DataSourceType,
  ApiDataSourceConfig,
  WebSocketDataSourceConfig,
  StaticDataSourceConfig,
  WidgetType,
  EndpointDetail,
} from "@/types/dashboard";
import type { FormInstance } from "antd";

const { TextArea } = Input;

export interface DataSourceFormProps {
  /** 当前数据源配置 */
  value?: DataSourceConfig;
  /** 配置变化回调 */
  onChange?: (value: DataSourceConfig) => void;
  /** Widget 类型（用于过滤支持的端点） */
  widgetType?: WidgetType;
  /** 表单实例 */
  form?: FormInstance;
  /** 是否禁用 */
  disabled?: boolean;
}

/**
 * 从 DataSourceConfig 中提取数据源类型
 */
const getDataSourceType = (config?: DataSourceConfig): DataSourceType => {
  if (!config) return "api";
  if ("type" in config) return config.type;
  if ("api" in config) return "api";
  if ("websocket" in config) return "websocket";
  if ("static" in config) return "static";
  return "api";
};

/**
 * 从 DataSourceConfig 中提取实际配置对象
 */
const extractConfig = <T extends DataSourceConfig>(
  type: DataSourceType,
  config?: DataSourceConfig
): T | null => {
  if (!config) return null;
  if ("type" in config && config.type === type) return config as T;
  if (type in (config as object)) return (config as unknown as Record<string, T>)[type] as T;
  return null;
};

/**
 * 数据源配置表单组件
 */
export const DataSourceForm: React.FC<DataSourceFormProps> = ({
  value,
  onChange,
  widgetType,
  form,
  disabled = false,
}) => {
  const [dataSourceType, setDataSourceType] = useState<DataSourceType>(() =>
    getDataSourceType(value)
  );
  const [selectedEndpoint, setSelectedEndpoint] = useState<EndpointDetail | null>(null);
  const [localForm] = Form.useForm();
  const activeForm = form || localForm;

  // 当外部 value 变化时同步表单
  useEffect(() => {
    if (value) {
      const type = getDataSourceType(value);
      setDataSourceType(type);

      if (type === "api") {
        const apiConfig = extractConfig<ApiDataSourceConfig>("api", value);
        if (apiConfig) {
          activeForm.setFieldsValue({
            endpoint: apiConfig.endpoint,
            method: apiConfig.method,
            params: apiConfig.params
              ? Object.entries(apiConfig.params).map(([name, val]) => ({
                  name,
                  value: String(val),
                }))
              : [],
            transformExpression: apiConfig.transform?.expression || "",
          });
        }
      } else if (type === "websocket") {
        const wsConfig = extractConfig<WebSocketDataSourceConfig>("websocket", value);
        if (wsConfig) {
          activeForm.setFieldsValue({
            channel: wsConfig.channel,
            transformExpression: wsConfig.transform?.expression || "",
          });
        }
      } else if (type === "static") {
        const staticConfig = extractConfig<StaticDataSourceConfig>("static", value);
        if (staticConfig) {
          activeForm.setFieldsValue({
            staticData: JSON.stringify(staticConfig.data, null, 2),
          });
        }
      }
    }
  }, [value, activeForm]);

  // 处理数据源类型切换
  const handleTypeChange = (newType: DataSourceType) => {
    setDataSourceType(newType);
    // 类型切换时清空表单，但保留 transform 配置
    const transformExpression = activeForm.getFieldValue("transformExpression");
    activeForm.resetFields();
    activeForm.setFieldValue("transformExpression", transformExpression);
  };

  // 处理端点选择变化
  const handleEndpointChange = (route: string, endpoint: EndpointDetail) => {
    setSelectedEndpoint(endpoint);
    activeForm.setFieldsValue({
      endpoint: route,
      method: endpoint.method,
    });
  };

  // 构建并发送配置变化
  const emitChange = () => {
    const values = activeForm.getFieldsValue();
    let config: DataSourceConfig;

    if (dataSourceType === "api") {
      // 处理参数
      const paramsObj: Record<string, unknown> = {};
      if (values.params && Array.isArray(values.params)) {
        values.params.forEach((param: { name: string; value: string }) => {
          if (param.name && param.value) {
            paramsObj[param.name] = param.value;
          }
        });
      }

      const apiConfig: ApiDataSourceConfig = {
        type: "api",
        endpoint: values.endpoint || "/api/default",
        method: (values.method || "GET") as "GET" | "POST",
        params: Object.keys(paramsObj).length > 0 ? paramsObj : undefined,
        transform: values.transformExpression
          ? { expression: values.transformExpression }
          : undefined,
      };
      config = { api: apiConfig };
    } else if (dataSourceType === "websocket") {
      const wsConfig: WebSocketDataSourceConfig = {
        type: "websocket",
        channel: values.channel || "",
        transform: values.transformExpression
          ? { expression: values.transformExpression }
          : undefined,
      };
      config = { websocket: wsConfig };
    } else {
      let staticData: unknown;
      try {
        staticData = values.staticData ? JSON.parse(values.staticData) : null;
      } catch {
        staticData = values.staticData;
      }
      const staticConfig: StaticDataSourceConfig = {
        type: "static",
        data: staticData,
      };
      config = { static: staticConfig };
    }

    onChange?.(config);
  };

  return (
    <div className="data-source-form">
      <Form form={activeForm} layout="vertical" onValuesChange={emitChange}>
        {/* 数据源类型选择 */}
        <Form.Item
          label="数据源类型"
          name="dataSourceType"
          tooltip="选择数据来源方式：API（REST接口）、WebSocket（实时推送）、Static（静态数据）"
          initialValue={dataSourceType}
        >
          <Select
            value={dataSourceType}
            onChange={handleTypeChange}
            disabled={disabled}
            options={[
              { label: "API 接口", value: "api" },
              { label: "WebSocket 实时", value: "websocket" },
              { label: "静态数据", value: "static" },
            ]}
          />
        </Form.Item>

        {/* API 数据源配置 */}
        {dataSourceType === "api" && (
          <>
            <Form.Item
              label="API 端点"
              name="endpoint"
              rules={[{ required: true, message: "请选择 API 端点" }]}
              tooltip="从下拉列表中选择可用的 API 端点"
            >
              <EndpointSelector
                widgetType={widgetType}
                onChange={handleEndpointChange}
                disabled={disabled}
              />
            </Form.Item>

            <Form.Item
              label="请求方法"
              name="method"
              tooltip="根据所选端点自动设置"
              initialValue="GET"
            >
              <Select disabled={disabled} onSearch={() => {}}>
                <Select.Option value="GET">GET</Select.Option>
                <Select.Option value="POST">POST</Select.Option>
              </Select>
            </Form.Item>

            <Form.Item label="请求参数" tooltip="配置 API 请求的查询参数">
              <ParamsEditor endpoint={selectedEndpoint || undefined} form={activeForm} />
            </Form.Item>

            <Form.Item
              label="数据转换表达式"
              name="transformExpression"
              tooltip="使用 JSONata 表达式提取和转换数据"
              extra="例如：data.list[0].value 或 $sum(data.items.price)"
            >
              <Input placeholder="data.list" disabled={disabled} />
            </Form.Item>
          </>
        )}

        {/* WebSocket 数据源配置 */}
        {dataSourceType === "websocket" && (
          <>
            <Form.Item
              label="WebSocket 频道"
              name="channel"
              rules={[{ required: true, message: "请输入 WebSocket 频道" }]}
              tooltip="订阅的 WebSocket 频道/主题名称"
            >
              <Input placeholder="dashboard/realtime" disabled={disabled} />
            </Form.Item>

            <Form.Item
              label="数据转换表达式"
              name="transformExpression"
              tooltip="使用 JSONata 表达式转换推送数据"
            >
              <Input placeholder="data.value" disabled={disabled} />
            </Form.Item>

            <Alert
              message="WebSocket 连接将在仪表盘加载时自动建立"
              type="info"
              showIcon
              icon={<InfoCircleOutlined />}
              style={{ marginTop: 8 }}
            />
          </>
        )}

        {/* 静态数据源配置 */}
        {dataSourceType === "static" && (
          <>
            <Form.Item
              label="静态数据"
              name="staticData"
              rules={[
                { required: true, message: "请输入静态数据" },
                {
                  validator: (_, value) => {
                    if (!value) return Promise.resolve();
                    try {
                      JSON.parse(value);
                      return Promise.resolve();
                    } catch {
                      return Promise.reject(new Error("请输入有效的 JSON 格式"));
                    }
                  },
                },
              ]}
              tooltip="输入 JSON 格式的静态数据"
            >
              <TextArea
                rows={6}
                placeholder='{"value": 100, "label": "示例数据"}'
                disabled={disabled}
              />
            </Form.Item>

            <Alert
              message="静态数据不会自动更新，适合展示固定内容"
              type="info"
              showIcon
              icon={<InfoCircleOutlined />}
              style={{ marginTop: 8 }}
            />
          </>
        )}
      </Form>
    </div>
  );
};

export default DataSourceForm;
