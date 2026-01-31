/**
 * CAD 属性面板组件
 */

import { useMemo, useEffect } from 'react';
import { Card, Form, InputNumber, Input, Select, ColorPicker, Divider, Empty } from 'antd';
import type { Wall, Door, TextElement } from './types';
import type { WorkstationNode } from '@/components/shared/FloorPlanEditor.types';

export interface CADPropertyPanelProps {
  selectedElement: Wall | Door | WorkstationNode | TextElement | null;
  onUpdate: (changes: Record<string, unknown>) => void;
  readOnly?: boolean;
  style?: React.CSSProperties;
}

export const CADPropertyPanel = function CADPropertyPanel({
  selectedElement,
  onUpdate,
  readOnly = false,
  style,
}: CADPropertyPanelProps) {
  const [form] = Form.useForm();

  const elementType = useMemo(() => {
    if (!selectedElement) return null;
    if ('points' in selectedElement) return 'wall';
    if ('content' in selectedElement) return 'text'; // 文本有 content 字段
    if ('width' in selectedElement && 'length' in selectedElement) return 'door'; // 门有 width 和 length
    if ('code' in selectedElement) return 'workstation'; // 工位有 code 字段
    return null;
  }, [selectedElement]);

  // 当选中元素变化时，更新表单值
  useEffect(() => {
    if (selectedElement) {
      form.setFieldsValue(selectedElement);
    }
  }, [selectedElement, form]);

  const handleValuesChange = function handleValuesChange(
    _values: unknown,
    allValues: Record<string, unknown>
  ) {
    if (readOnly) return;

    // 处理颜色值（ColorPicker 返回的是对象，需要转换为字符串）
    const processedValues = { ...allValues };
    if (processedValues.color && typeof processedValues.color === 'object' && 'toHexString' in processedValues.color) {
      processedValues.color = (processedValues.color as any).toHexString();
    }

    onUpdate(processedValues);
  };

  // 获取初始值，处理颜色格式
  const getInitialValues = () => {
    if (!selectedElement) return {};
    const values = { ...selectedElement };
    // 颜色值已经是字符串，不需要转换
    return values;
  };

  if (!selectedElement) {
    return (
      <Card
        title="属性"
        size="small"
        style={{ width: 280, ...style }}
        styles={{ body: { padding: 16 } }}
      >
        <Empty
          description="请选择元素"
          image={Empty.PRESENTED_IMAGE_SIMPLE}
        />
      </Card>
    );
  }

  return (
    <Card
      title="属性"
      size="small"
      style={{ width: 280, ...style }}
      styles={{ body: { padding: 16 } }}
    >
      <Form
        form={form}
        layout="vertical"
        size="small"
        initialValues={getInitialValues()}
        onValuesChange={handleValuesChange}
      >
        {elementType === 'wall' && (
          <>
            <Form.Item label="名称" name="name">
              <Input placeholder="墙体名称" disabled={readOnly} />
            </Form.Item>

            <Form.Item label="类型" name="type">
              <Select
                options={[
                  { label: '直线墙', value: 'straight' },
                  { label: '弧形墙', value: 'curved' },
                  { label: 'L型墙', value: 'l_shaped' },
                ]}
                disabled={readOnly}
              />
            </Form.Item>

            <Form.Item label="厚度" name="thickness">
              <InputNumber min={1} max={100} style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>

            <Form.Item label="高度" name="height">
              <InputNumber min={0.1} max={10} step={0.1} style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>

            <Form.Item label="颜色" name="color">
              <ColorPicker showText disabled={readOnly} format="hex" />
            </Form.Item>
          </>
        )}

        {elementType === 'door' && (
          <>
            <Form.Item label="名称" name="name">
              <Input placeholder="门名称" disabled={readOnly} />
            </Form.Item>

            <Form.Item label="类型" name="type">
              <Select
                options={[
                  { label: '单开门', value: 'single' },
                  { label: '双开门', value: 'double' },
                  { label: '推拉门', value: 'sliding' },
                  { label: '旋转门', value: 'revolving' },
                  { label: '紧急出口', value: 'emergency' },
                ]}
                disabled={readOnly}
              />
            </Form.Item>

            <Form.Item label="开启方向" name="direction">
              <Select<string>
                options={[
                  { label: '左开', value: 'left' },
                  { label: '右开', value: 'right' },
                  { label: '双向', value: 'double' },
                  { label: '推拉', value: 'sliding' },
                ]}
                disabled={readOnly}
              />
            </Form.Item>

            <Form.Item label="宽度" name="width">
              <InputNumber min={20} max={200} style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>

            <Form.Item label="长度" name="length">
              <InputNumber min={10} max={100} style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>

            <Form.Item label="旋转角度" name="angle">
              <InputNumber min={0} max={360} style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>

            <Form.Item label="颜色" name="color">
              <ColorPicker showText disabled={readOnly} format="hex" />
            </Form.Item>

            <Divider titlePlacement="left">位置</Divider>

            <Form.Item label="X 坐标" name={['position', 'x']}>
              <InputNumber style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>

            <Form.Item label="Y 坐标" name={['position', 'y']}>
              <InputNumber style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>
          </>
        )}

        {elementType === 'workstation' && (
          <>
            <Form.Item label="编号" name="code">
              <Input placeholder="工位编号" disabled={readOnly} />
            </Form.Item>

            <Form.Item label="名称" name="name">
              <Input placeholder="工位名称" disabled={readOnly} />
            </Form.Item>

            <Form.Item label="桌型" name="type">
              <Select<number>
                options={[
                  { label: '一字型', value: 0 },
                  { label: 'L型', value: 1 },
                ]}
                disabled={readOnly}
              />
            </Form.Item>

            <Form.Item label="宽度" name="width">
              <InputNumber min={60} max={300} step={10} style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>

            <Form.Item label="深度" name="height">
              <InputNumber min={40} max={200} step={10} style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>

            <Form.Item label="旋转角度" name="rotation">
              <InputNumber min={0} max={360} step={15} style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>

            <Divider titlePlacement="left">状态</Divider>

            <Form.Item label="使用状态" name="status">
              <Select<number>
                options={[
                  { label: '空闲', value: 0 },
                  { label: '占用', value: 1 },
                  { label: '维修', value: 2 },
                ]}
                disabled={readOnly}
              />
            </Form.Item>
          </>
        )}

        {elementType === 'text' && (
          <>
            <Form.Item label="文本内容" name="content">
              <Input.TextArea rows={3} placeholder="输入文本内容" disabled={readOnly} />
            </Form.Item>

            <Form.Item label="字号" name="fontSize">
              <InputNumber min={8} max={72} style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>

            <Form.Item label="颜色" name="color">
              <ColorPicker showText disabled={readOnly} format="hex" />
            </Form.Item>

            <Form.Item label="旋转角度" name="angle">
              <InputNumber min={0} max={360} style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>

            <Divider titlePlacement="left">位置</Divider>

            <Form.Item label="X 坐标" name={['position', 'x']}>
              <InputNumber style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>

            <Form.Item label="Y 坐标" name={['position', 'y']}>
              <InputNumber style={{ width: '100%' }} disabled={readOnly} />
            </Form.Item>
          </>
        )}

        {elementType !== 'text' && (
          <Form.Item label="备注" name="remark">
            <Input.TextArea rows={3} placeholder="备注信息" disabled={readOnly} />
          </Form.Item>
        )}
      </Form>
    </Card>
  );
};

export default CADPropertyPanel;
