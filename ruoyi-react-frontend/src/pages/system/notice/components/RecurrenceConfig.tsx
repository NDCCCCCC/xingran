import { Row, Col, DatePicker, Form } from "antd";
import CronSelector from "@/components/CronSelector";

/**
 * 周期配置组件
 * 使用 Cron 表达式配置周期性任务
 */
export const RecurrenceConfig: React.FC = () => {
  return (
    <>
      <Row gutter={16}>
        <Col span={24}>
          <Form.Item
            name={["recurrenceConfig", "cronExpression"]}
            label="Cron 表达式"
            required
            rules={[{ required: true, message: "请输入 Cron 表达式" }]}
          >
            <CronSelector />
          </Form.Item>
        </Col>
      </Row>

      <Row gutter={16}>
        <Col span={12}>
          <Form.Item
            name={["recurrenceConfig", "endDate"]}
            label="结束时间（可选）"
            extra="留空表示永久执行"
          >
            <DatePicker format="YYYY-MM-DD HH:mm:ss" className="w-full" showTime />
          </Form.Item>
        </Col>
      </Row>
    </>
  );
};
