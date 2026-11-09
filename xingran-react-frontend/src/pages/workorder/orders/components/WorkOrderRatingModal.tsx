/**
 * 工单评价弹窗（FIX-06 / Phase 136 最小可用前端入口）。
 *
 * - 列表/统计：复用包装 API 函数 getWorkOrderRatings / getWorkOrderRatingStatistics
 *   （内部走 @/lib/api 的 post，非裸 axios）。
 * - 提交：createWorkOrderRating，后端 POST /workorder/orders/ratings 接
 *   RatingService.Create，成功路径由后端记 operlog（工单管理 / OperTypeCreate）。
 */
import { useCallback, useEffect, useState, type FC } from "react";
import { Modal, Form, Radio, Rate, List, Tag, Spin, Empty, Typography } from "antd";
import { AppTextArea } from "@/components/ui";
import {
  submitWorkOrderRating,
  listWorkOrderRatings,
  fetchWorkOrderRatingStats,
  type WorkOrderRatingRecordV2,
  type WorkOrderRatingStatisticsV2,
} from "@/lib/workorderApi";
import { handleApiError, handleSuccess } from "@/utils/errorHandler";

const { Text } = Typography;

export interface WorkOrderRatingModalProps {
  /** 为空时弹窗关闭 */
  workOrderId: string | null;
  workOrderTitle?: string;
  onClose: () => void;
}

const RATING_TYPE_OPTIONS = [
  { label: "用户评价（完成度）", value: "user" },
  { label: "处理人评价（配合度）", value: "handler" },
];

const WorkOrderRatingModal: FC<WorkOrderRatingModalProps> = ({
  workOrderId,
  workOrderTitle,
  onClose,
}) => {
  const [form] = Form.useForm<{
    ratingType: "user" | "handler";
    score: number;
    comment?: string;
  }>();
  const [submitting, setSubmitting] = useState(false);
  const [loading, setLoading] = useState(false);
  const [ratings, setRatings] = useState<WorkOrderRatingRecordV2[]>([]);
  const [stats, setStats] = useState<WorkOrderRatingStatisticsV2 | null>(null);
  const ratingType = Form.useWatch("ratingType", form);

  const loadRatings = useCallback(async (id: string) => {
    setLoading(true);
    try {
      // 包装 API 函数；result.data 直接是业务数据（api.ts 响应拦截器已拆包）
      const [listRes, statsRes] = await Promise.all([
        listWorkOrderRatings(id),
        fetchWorkOrderRatingStats(id),
      ]);
      setRatings(listRes.data ?? []);
      setStats(statsRes.data ?? null);
    } catch (error) {
      handleApiError(error, "加载工单评价失败");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    if (!workOrderId) return;
    form.setFieldsValue({ ratingType: "user", score: 5, comment: undefined });
    void loadRatings(workOrderId);
  }, [workOrderId, form, loadRatings]);

  const handleSubmit = async () => {
    if (!workOrderId) return;
    const values = await form.validateFields();
    setSubmitting(true);
    try {
      // 按 ratingType 把评分落到对应字段，与 RatingCreateRequest 对齐
      await submitWorkOrderRating({
        workOrderId,
        ratingType: values.ratingType,
        completionScore: values.ratingType === "user" ? values.score : 0,
        cooperationScore: values.ratingType === "handler" ? values.score : 0,
        comment: values.comment,
      });
      handleSuccess("评价提交成功");
      form.setFieldsValue({ score: 5, comment: undefined });
      void loadRatings(workOrderId);
    } catch (error) {
      handleApiError(error, "提交工单评价失败");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      open={!!workOrderId}
      title={`工单评价${workOrderTitle ? ` - ${workOrderTitle}` : ""}`}
      onCancel={onClose}
      onOk={handleSubmit}
      okText="提交评价"
      confirmLoading={submitting}
      destroyOnHidden
      width={560}
    >
      <Spin spinning={loading}>
        {stats && (
          <div style={{ marginBottom: 12 }}>
            <Text type="secondary">评价数：{stats.totalCount ?? 0}</Text>
            {stats.avgCompletionScore != null && (
              <Text style={{ marginLeft: 12 }}>
                平均完成度：{Number(stats.avgCompletionScore).toFixed(1)}
              </Text>
            )}
            {stats.avgCooperationScore != null && (
              <Text style={{ marginLeft: 12 }}>
                平均配合度：{Number(stats.avgCooperationScore).toFixed(1)}
              </Text>
            )}
          </div>
        )}

        <Form form={form} layout="vertical">
          <Form.Item name="ratingType" label="评价类型" rules={[{ required: true }]}>
            <Radio.Group options={RATING_TYPE_OPTIONS} />
          </Form.Item>
          <Form.Item
            name="score"
            label={ratingType === "handler" ? "配合度评分" : "完成度评分"}
            rules={[{ required: true, message: "请选择 1-5 分" }]}
          >
            <Rate allowClear={false} />
          </Form.Item>
          <Form.Item name="comment" label="评价内容">
            <AppTextArea rows={2} maxLength={500} showCount placeholder="可选" />
          </Form.Item>
        </Form>

        {ratings.length > 0 ? (
          <List
            size="small"
            header={<Text strong>已有评价</Text>}
            dataSource={ratings}
            renderItem={(item) => (
              <List.Item>
                <List.Item.Meta
                  title={
                    <span>
                      <Tag color={item.ratingType === "user" ? "blue" : "green"}>
                        {item.ratingType === "user" ? "用户评价" : "处理人评价"}
                      </Tag>
                      <Rate
                        disabled
                        allowHalf
                        value={
                          item.ratingType === "user" ? item.completionScore : item.cooperationScore
                        }
                      />
                    </span>
                  }
                  description={item.comment || "（无评价内容）"}
                />
              </List.Item>
            )}
          />
        ) : (
          !loading && <Empty description="暂无评价" image={Empty.PRESENTED_IMAGE_SIMPLE} />
        )}
      </Spin>
    </Modal>
  );
};

export default WorkOrderRatingModal;
