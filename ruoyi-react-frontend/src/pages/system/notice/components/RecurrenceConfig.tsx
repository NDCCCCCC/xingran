import { Row, Col, Select, DatePicker, TimePicker, Input, Form } from 'antd';
import type { RecurrenceType } from '@/types/notice';

const { Option } = Select;

interface RecurrenceConfigProps {
  recurrenceType: RecurrenceType;
  onRecurrenceTypeChange: (type: RecurrenceType) => void;
}

/**
 * 周期配置组件
 * 支持每天、每周、每月、自定义Cron等周期类型
 */
export const RecurrenceConfig: React.FC<RecurrenceConfigProps> = ({
  recurrenceType,
  onRecurrenceTypeChange,
}) => {
  return (
    <>
      <Row gutter={16}>
        <Col span={12}>
          <Form.Item label="周期类型" required>
            <Select
              value={recurrenceType}
              onChange={(value) => {
                onRecurrenceTypeChange(value);
                // 清空周期相关字段
                // 由父组件处理
              }}
            >
              <Option value="daily">每天</Option>
              <Option value="weekly">每周</Option>
              <Option value="monthly">每月</Option>
              <Option value="custom">自定义 Cron</Option>
            </Select>
          </Form.Item>
        </Col>
        <Col span={12}>
          <Form.Item name={['recurrenceConfig', 'executeTime']} label="执行时间" required>
            <TimePicker format="HH:mm" className="w-full" />
          </Form.Item>
        </Col>
      </Row>

      {recurrenceType === 'weekly' && (
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name={['recurrenceConfig', 'weekDay']} label="选择星期" required>
              <Select placeholder="请选择星期几">
                <Option value={1}>周日</Option>
                <Option value={2}>周一</Option>
                <Option value={3}>周二</Option>
                <Option value={4}>周三</Option>
                <Option value={5}>周四</Option>
                <Option value={6}>周五</Option>
                <Option value={7}>周六</Option>
              </Select>
            </Form.Item>
          </Col>
        </Row>
      )}

      {recurrenceType === 'monthly' && (
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name={['recurrenceConfig', 'monthDay']} label="每月几号" required>
              <Select placeholder="请选择日期">
                {Array.from({ length: 31 }, (_, i) => (
                  <Option key={i + 1} value={i + 1}>{i + 1}号</Option>
                ))}
              </Select>
            </Form.Item>
          </Col>
        </Row>
      )}

      {recurrenceType === 'custom' && (
        <Row gutter={16}>
          <Col span={24}>
            <Form.Item
              name={['recurrenceConfig', 'cronExpression']}
              label="Cron 表达式"
              required
              rules={[{ required: true, message: '请输入 Cron 表达式' }]}
              extra="格式：秒 分 时 日 月 周，例如：0 0 9 * * ? （每天上午9点）"
            >
              <Input placeholder="0 0 9 * * ?" />
            </Form.Item>
          </Col>
        </Row>
      )}

      <Row gutter={16}>
        <Col span={12}>
          <Form.Item name={['recurrenceConfig', 'endDate']} label="结束时间（可选）">
            <DatePicker format="YYYY-MM-DD HH:mm:ss" className="w-full" />
          </Form.Item>
        </Col>
      </Row>
    </>
  );
};
