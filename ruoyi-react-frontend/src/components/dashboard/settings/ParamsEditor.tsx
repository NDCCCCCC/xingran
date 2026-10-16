/**
 * 参数编辑器组件
 * Params Editor Component
 *
 * 用于编辑API请求参数，支持动态添加/删除参数键值对
 */
import { useCallback, useEffect } from "react";
import { Form, Input, Button, Space, Alert, Empty } from "antd";
import { PlusOutlined, MinusCircleOutlined, InfoCircleOutlined } from "@ant-design/icons";
import type { FormInstance } from "antd";
import type { EndpointDetail } from "@/types/dashboard";

export interface ParamsEditorProps {
  /** 端点元数据 */
  endpoint?: EndpointDetail;
  /** 表单实例 */
  form?: FormInstance;
  /** 是否显示示例参数提示 */
  showExampleHint?: boolean;
}

/**
 * 参数项类型
 */
interface ParamItem {
  key?: string;
  name: string;
  value: string;
}

/**
 * 参数编辑器组件
 */
export const ParamsEditor: React.FC<ParamsEditorProps> = ({
  endpoint,
  form,
  showExampleHint = true,
}) => {
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const exampleParams = endpoint?.exampleParams || {};

  // 将示例参数转换为数组格式
  const exampleParamsArray = useCallback((): ParamItem[] => {
    return Object.entries(exampleParams).map(([key, value]) => ({
      key,
      name: key,
      value: String(value),
    }));
  }, [exampleParams]);

  // 当端点变化时，自动填充示例参数
  useEffect(() => {
    if (form && endpoint && Object.keys(exampleParams).length > 0) {
      const examples = exampleParamsArray();
      form.setFieldValue("params", examples);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [endpoint, form, exampleParamsArray]);

  // 加载示例参数按钮点击处理
  const handleLoadExample = () => {
    if (form) {
      const examples = exampleParamsArray();
      form.setFieldValue("params", examples.length > 0 ? examples : []);
    }
  };

  return (
    <div>
      {/* 示例参数提示 */}
      {showExampleHint && Object.keys(exampleParams).length > 0 && (
        <Alert
          message="示例参数可用"
          description="点击下方按钮加载该端点的示例参数，可作为参考进行修改。"
          type="info"
          showIcon
          icon={<InfoCircleOutlined />}
          action={
            <Button size="small" type="link" onClick={handleLoadExample}>
              加载示例参数
            </Button>
          }
          style={{ marginBottom: 16 }}
        />
      )}

      {/* 参数编辑表单 */}
      <Form.List name="params">
        {(fields, { add, remove }) => (
          <>
            {fields.length === 0 ? (
              <Empty
                description="该端点无需额外参数或暂未配置参数"
                image={Empty.PRESENTED_IMAGE_SIMPLE}
                style={{ padding: "20px 0" }}
              />
            ) : null}

            {fields.map(({ key, name, ...restField }) => (
              <Space
                key={key}
                style={{
                  display: "flex",
                  marginBottom: 8,
                  alignItems: "flex-start",
                }}
                align="baseline"
              >
                <Form.Item
                  {...restField}
                  name={[name, "name"]}
                  rules={[{ required: true, message: "请输入参数名" }]}
                  style={{ marginBottom: 0 }}
                >
                  <Input placeholder="参数名" style={{ width: 150 }} aria-label="参数名" />
                </Form.Item>

                <Form.Item
                  {...restField}
                  name={[name, "value"]}
                  rules={[{ required: true, message: "请输入参数值" }]}
                  style={{ marginBottom: 0 }}
                >
                  <Input placeholder="参数值" style={{ width: 200 }} aria-label="参数值" />
                </Form.Item>

                <MinusCircleOutlined
                  onClick={() => remove(name)}
                  style={{
                    fontSize: 16,
                    color: "var(--theme-text-tertiary, #999)",
                    cursor: "pointer",
                    marginTop: 8,
                  }}
                />
              </Space>
            ))}

            <Form.Item style={{ marginTop: fields.length > 0 ? 8 : 0 }}>
              <Button
                type="dashed"
                onClick={() => add({ name: "", value: "" })}
                block
                icon={<PlusOutlined />}
              >
                添加参数
              </Button>
            </Form.Item>
          </>
        )}
      </Form.List>

      {/* 参数说明 */}
      {endpoint && (
        <div style={{ marginTop: 16, padding: "12px", background: "#f5f5f5", borderRadius: 4 }}>
          <div style={{ fontSize: 12, color: "#666" }}>
            <strong>参数说明：</strong>
          </div>
          <div style={{ fontSize: 12, color: "var(--theme-text-tertiary, #999)", marginTop: 4 }}>
            添加的参数将作为请求参数发送到API端点。支持动态变量替换， 如使用{" "}
            <code>{"{{userId}}"}</code> 将在请求时替换为当前用户ID。
          </div>
        </div>
      )}
    </div>
  );
};

export default ParamsEditor;
