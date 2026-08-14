/**
 * CAD 属性面板组件
 */

import { useMemo, useCallback, useEffect } from "react";
import { Card, Form, InputNumber, Input, Select, ColorPicker, Divider, Empty } from "antd";
import type { Wall, Door, TextElement } from "./types";
import type { WorkstationNode } from "@/components/shared/FloorPlanEditor.types";

export interface CADPropertyPanelProps {
  selectedElement: Wall | Door | WorkstationNode | TextElement | null;
  onUpdate: (changes: Record<string, unknown>) => void;
  readOnly?: boolean;
  style?: React.CSSProperties;
}

// 表单字段配置
type FieldType = "text" | "number" | "select" | "color";

interface FieldConfig {
  label: string;
  name: string | string[];
  type: FieldType;
  options?: { label: string; value: number | string }[];
  min?: number;
  max?: number;
  step?: number;
  placeholder?: string;
  rows?: number;
}

// 元素类型识别
type ElementType = "wall" | "door" | "workstation" | "text" | null;

const PANEL_WIDTH = 280;

// 墙体字段配置
const WALL_FIELDS: readonly FieldConfig[] = [
  { label: "名称", name: "name", type: "text", placeholder: "墙体名称" },
  {
    label: "类型",
    name: "type",
    type: "select",
    options: [
      { label: "直线墙", value: "straight" },
      { label: "弧形墙", value: "curved" },
      { label: "L型墙", value: "l_shaped" },
    ],
  },
  { label: "厚度", name: "thickness", type: "number", min: 1, max: 100 },
  { label: "高度", name: "height", type: "number", min: 0.1, max: 10, step: 0.1 },
  { label: "颜色", name: "color", type: "color" },
] as const;

// 门字段配置
const DOOR_FIELDS: readonly FieldConfig[] = [
  { label: "名称", name: "name", type: "text", placeholder: "门名称" },
  {
    label: "类型",
    name: "type",
    type: "select",
    options: [
      { label: "单开门", value: "single" },
      { label: "双开门", value: "double" },
      { label: "推拉门", value: "sliding" },
      { label: "旋转门", value: "revolving" },
      { label: "紧急出口", value: "emergency" },
    ],
  },
  {
    label: "开启方向",
    name: "direction",
    type: "select",
    options: [
      { label: "左开", value: "left" },
      { label: "右开", value: "right" },
      { label: "双向", value: "double" },
      { label: "推拉", value: "sliding" },
    ],
  },
  { label: "宽度", name: "width", type: "number", min: 20, max: 200 },
  { label: "长度", name: "length", type: "number", min: 10, max: 100 },
  { label: "旋转角度", name: "angle", type: "number", min: 0, max: 360 },
  { label: "颜色", name: "color", type: "color" },
  { label: "X 坐标", name: ["position", "x"], type: "number" },
  { label: "Y 坐标", name: ["position", "y"], type: "number" },
] as const;

// 工位字段配置
const WORKSTATION_FIELDS: readonly FieldConfig[] = [
  { label: "编号", name: "code", type: "text", placeholder: "工位编号" },
  { label: "名称", name: "name", type: "text", placeholder: "工位名称" },
  {
    label: "桌型",
    name: "type",
    type: "select",
    options: [
      { label: "一字型", value: 0 },
      { label: "L型", value: 1 },
    ],
  },
  { label: "宽度", name: "width", type: "number", min: 60, max: 300, step: 10 },
  { label: "深度", name: "height", type: "number", min: 40, max: 200, step: 10 },
  { label: "旋转角度", name: "rotation", type: "number", min: 0, max: 360, step: 15 },
  {
    label: "使用状态",
    name: "status",
    type: "select",
    options: [
      { label: "空闲", value: 0 },
      { label: "占用", value: 1 },
      { label: "维修", value: 2 },
    ],
  },
] as const;

// 文本字段配置
const TEXT_FIELDS: readonly FieldConfig[] = [
  { label: "文本内容", name: "content", type: "text", rows: 3, placeholder: "输入文本内容" },
  { label: "字号", name: "fontSize", type: "number", min: 8, max: 72 },
  { label: "颜色", name: "color", type: "color" },
  { label: "旋转角度", name: "angle", type: "number", min: 0, max: 360 },
  { label: "X 坐标", name: ["position", "x"], type: "number" },
  { label: "Y 坐标", name: ["position", "y"], type: "number" },
] as const;

// 通用备注字段
const REMARK_FIELD: FieldConfig = {
  label: "备注",
  name: "remark",
  type: "text",
  rows: 3,
  placeholder: "备注信息",
};

export function CADPropertyPanel({
  selectedElement,
  onUpdate,
  readOnly = false,
  style,
}: CADPropertyPanelProps) {
  const [form] = Form.useForm();

  const elementType = useMemo((): ElementType => {
    if (!selectedElement) return null;
    if ("points" in selectedElement) return "wall";
    if ("content" in selectedElement) return "text";
    if ("width" in selectedElement && "length" in selectedElement) return "door";
    if ("code" in selectedElement) return "workstation";
    return null;
  }, [selectedElement]);

  // 当选中元素变化时，更新表单值
  useEffect(() => {
    if (selectedElement) {
      form.setFieldsValue(selectedElement);
    }
  }, [selectedElement, form]);

  const handleValuesChange = useCallback(
    (_values: unknown, allValues: Record<string, unknown>) => {
      if (readOnly) return;

      const processedValues = { ...allValues };

      // 处理颜色值（ColorPicker 返回的是对象，需要转换为字符串）
      if (
        processedValues.color &&
        typeof processedValues.color === "object" &&
        "toHexString" in processedValues.color
      ) {
        processedValues.color = (
          processedValues.color as { toHexString: () => string }
        ).toHexString();
      }

      onUpdate(processedValues);
    },
    [onUpdate, readOnly]
  );

  // 渲染表单字段
  const renderField = function (field: FieldConfig) {
    const commonProps = {
      disabled: readOnly,
      style: { width: "100%" },
    };

    switch (field.type) {
      case "text":
        if (field.rows) {
          return (
            <Input.TextArea
              key={String(field.name)}
              placeholder={field.placeholder}
              rows={field.rows}
              {...commonProps}
            />
          );
        }
        return <Input key={String(field.name)} placeholder={field.placeholder} {...commonProps} />;

      case "number":
        return (
          <InputNumber
            key={String(field.name)}
            min={field.min}
            max={field.max}
            step={field.step}
            {...commonProps}
          />
        );

      case "select":
        return <Select key={String(field.name)} options={field.options} {...commonProps} />;

      case "color":
        return <ColorPicker key={String(field.name)} showText disabled={readOnly} format="hex" />;

      default:
        return null;
    }
  };

  // 获取字段配置
  const getFields = function (): readonly FieldConfig[] {
    switch (elementType) {
      case "wall":
        return WALL_FIELDS;
      case "door":
        return DOOR_FIELDS;
      case "workstation":
        return WORKSTATION_FIELDS;
      case "text":
        return TEXT_FIELDS;
      default:
        return [];
    }
  };

  const fields = getFields();

  // 空状态
  if (!selectedElement) {
    return (
      <Card
        title="属性"
        size="small"
        style={{ width: PANEL_WIDTH, ...style }}
        styles={{ body: { padding: 16 } }}
      >
        <Empty description="请选择元素" image={Empty.PRESENTED_IMAGE_SIMPLE} />
      </Card>
    );
  }

  return (
    <Card
      title="属性"
      size="small"
      style={{ width: PANEL_WIDTH, ...style }}
      styles={{ body: { padding: 16 } }}
    >
      <Form
        form={form}
        layout="vertical"
        size="small"
        initialValues={selectedElement}
        onValuesChange={handleValuesChange}
      >
        {fields.map((field) => (
          <Form.Item key={String(field.name)} label={field.label} name={field.name}>
            {renderField(field)}
          </Form.Item>
        ))}

        {/* 位置分隔线（门和文本有位置字段） */}
        {(elementType === "door" || elementType === "text") && (
          <Divider titlePlacement="left">位置</Divider>
        )}

        {/* 状态分隔线（工位有状态字段） */}
        {elementType === "workstation" && <Divider titlePlacement="left">状态</Divider>}

        {/* 备注字段（文本没有备注字段） */}
        {elementType !== "text" && (
          <Form.Item label={REMARK_FIELD.label} name={REMARK_FIELD.name}>
            {renderField(REMARK_FIELD)}
          </Form.Item>
        )}
      </Form>
    </Card>
  );
}
