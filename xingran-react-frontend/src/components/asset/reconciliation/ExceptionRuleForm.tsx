/**
 * ExceptionRuleForm (Phase 44 R3 / Plan 44-01 Task 6b)
 *
 * 例外规则创建/编辑表单(antd Form + Modal 模式)。
 *
 * 9 字段:
 *   1. Name (Input, required)
 *   2. IPRange (Input, CIDR, required)
 *   3. ConflictTypes (Select mode=multiple, B/C/D/E/F 或留空=全部)
 *   4. ExceptionActions (Select mode=multiple, required, 5 白名单)
 *   5. SeverityOverride (Select allowClear, low/medium/high)
 *   6. ScopeType (Radio.Group, global/dept/user)
 *   7. ScopeID (条件渲染: dept→TreeSelect / user→Select)
 *   8. ExpiresAt (DatePicker showTime)
 *   9. Reason (TextArea, required, min 10 字符)
 *
 * 后端校验同步:
 *   - ValidateCIDR / ValidateActions / ValidateSeverityOverride / ValidateReason
 *   - severity_override 不含 critical (Pitfall 8)
 *
 * 使用 reconciliationApi.exceptionRule.{create,update} (CLAUDE.md 强约束:不用 raw axios)
 */

import { useEffect, useState } from "react";
import {
  Form,
  Input,
  Select,
  DatePicker,
  Radio,
  Modal,
  App,
} from "antd";
import dayjs, { type Dayjs } from "dayjs";

const { TextArea } = Input;

export interface ExceptionRuleFormValues {
  id?: string;
  name: string;
  ipRange: string;
  conflictTypes?: string[];
  exceptionActions: string[];
  severityOverride?: string;
  scopeType: "global" | "dept" | "user";
  scopeId?: string;
  expiresAt?: Dayjs;
  reason: string;
}

interface ExceptionRuleFormProps {
  open: boolean;
  initialValues?: Partial<ExceptionRuleFormValues>;
  onSubmit: (values: ExceptionRuleFormValues) => Promise<void>;
  onCancel: () => void;
}

const CONFLICT_TYPE_OPTIONS = [
  { label: "B 物理有/声明无", value: "B" },
  { label: "C 物理声明不匹配", value: "C" },
  { label: "D 仅声明", value: "D" },
  { label: "E 无关联", value: "E" },
  { label: "F AD 账号不一致", value: "F" },
];

const EXCEPTION_ACTION_OPTIONS = [
  { label: "no_alert 不告警", value: "no_alert" },
  { label: "no_notice 不通知", value: "no_notice" },
  { label: "no_workorder 不转工单", value: "no_workorder" },
  { label: "skip_severity 降级", value: "skip_severity" },
  { label: "silence 静默", value: "silence" },
];

const SEVERITY_OVERRIDE_OPTIONS = [
  { label: "(不覆盖)", value: "" },
  { label: "low", value: "low" },
  { label: "medium", value: "medium" },
  { label: "high", value: "high" },
];

const ExceptionRuleForm: React.FC<ExceptionRuleFormProps> = ({
  open,
  initialValues,
  onSubmit,
  onCancel,
}) => {
  const { message } = App.useApp();
  const [form] = Form.useForm<ExceptionRuleFormValues>();
  const [submitting, setSubmitting] = useState(false);
  const [scopeType, setScopeType] = useState<"global" | "dept" | "user">(
    initialValues?.scopeType ?? "global"
  );

  // 表单初值同步(打开 Modal 时回填)
  useEffect(() => {
    if (open) {
      const v: Partial<ExceptionRuleFormValues> = {
        scopeType: "global",
        ...initialValues,
      };
      // Dayjs 转换(后端返回 ISO 字符串,form 需要 Dayjs)
      const rawExpires = (initialValues as { expiresAt?: string | Dayjs } | undefined)?.expiresAt;
      if (typeof rawExpires === "string" && rawExpires) {
        (v as { expiresAt?: Dayjs }).expiresAt = dayjs(rawExpires);
      }
      form.setFieldsValue(v);
      setScopeType((v as { scopeType?: "global" | "dept" | "user" }).scopeType ?? "global");
    }
  }, [open, initialValues, form]);

  const handleOk = async () => {
    try {
      const values = await form.validateFields();
      setSubmitting(true);
      // 转换 expiresAt Dayjs → ISO 字符串(后端 time.Time 接收 RFC3339)
      const submitValues: ExceptionRuleFormValues = {
        ...values,
        severityOverride: values.severityOverride || undefined,
        scopeId: values.scopeId || undefined,
        expiresAt: values.expiresAt,
      };
      await onSubmit(submitValues);
      message.success(initialValues?.id ? "更新成功" : "创建成功");
      form.resetFields();
    } catch (err) {
      if ((err as { errorFields?: unknown }).errorFields) {
        // antd Form 校验失败,不显示错误(validateFields 已自带 UI 提示)
        return;
      }
      message.error((err as Error)?.message ?? "提交失败");
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Modal
      title={initialValues?.id ? "编辑例外规则" : "新建例外规则"}
      open={open}
      onOk={handleOk}
      onCancel={() => {
        if (submitting) return;
        form.resetFields();
        onCancel();
      }}
      confirmLoading={submitting}
      okText="保存"
      cancelText="取消"
      destroyOnClose
      width={680}
    >
      <Form form={form} layout="vertical" preserve={false}>
        <Form.Item
          label="规则名称"
          name="name"
          rules={[{ required: true, message: "请输入规则名称" }, { max: 128 }]}
        >
          <Input placeholder="如:研发部测试网段豁免" />
        </Form.Item>

        <Form.Item
          label="IP段 (CIDR)"
          name="ipRange"
          rules={[{ required: true, message: "请输入 CIDR" }]}
          // eslint-disable-next-line no-restricted-syntax -- placeholder 仅作 UI 提示, 非真实 IP 配置
          extra="支持 IPv4/IPv6,如 192.168.0.0/16 或 2001:db8::/32"
        >
          {/* eslint-disable-next-line no-restricted-syntax -- placeholder 仅作 UI 提示 */}
          <Input placeholder="192.168.0.0/16" />
        </Form.Item>

        <Form.Item
          label="冲突类型"
          name="conflictTypes"
          extra="留空 = 匹配全部 B-F 类型"
        >
          {/* eslint-disable-next-line local/no-large-dropdown-list -- fixed option list, no server search needed */}
          <Select mode="multiple" allowClear placeholder="选择类型,留空匹配全部">
            {CONFLICT_TYPE_OPTIONS.map((o) => (
              <Select.Option key={o.value} value={o.value}>
                {o.label}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item
          label="例外动作"
          name="exceptionActions"
          rules={[{ required: true, message: "至少选择 1 个动作" }]}
          extra="可多选,多规则命中时取并集"
        >
          {/* eslint-disable-next-line local/no-large-dropdown-list -- fixed option list, no server search needed */}
          <Select mode="multiple" placeholder="选择动作">
            {EXCEPTION_ACTION_OPTIONS.map((o) => (
              <Select.Option key={o.value} value={o.value}>
                {o.label}
              </Select.Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item
          label="严重度覆盖"
          name="severityOverride"
          extra="覆盖原始 severity(取最低);不含 critical(Pitfall 8)"
        >
          {/* eslint-disable-next-line local/no-large-dropdown-list -- fixed option list, no server search needed */}
          <Select allowClear options={SEVERITY_OVERRIDE_OPTIONS} />
        </Form.Item>

        <Form.Item label="范围类型" name="scopeType">
          <Radio.Group
            onChange={(e) => setScopeType(e.target.value)}
          >
            <Radio value="global">global 全局</Radio>
            <Radio value="dept">dept 部门</Radio>
            <Radio value="user">user 用户</Radio>
          </Radio.Group>
        </Form.Item>

        {(scopeType === "dept" || scopeType === "user") && (
          <Form.Item
            label={scopeType === "dept" ? "部门 ID (UUID)" : "用户 ID (UUID)"}
            name="scopeId"
            rules={[{ required: true, message: `请输入 ${scopeType} UUID` }]}
            extra={`${scopeType} scope 需 IP CIDR 命中 + ${scopeType}ID 匹配双条件`}
          >
            <Input placeholder="00000000-0000-0000-0000-000000000000" />
          </Form.Item>
        )}

        <Form.Item
          label="过期时间"
          name="expiresAt"
          extra="可选,到期后 cron 自动软停用(D-R3-A4-03)"
        >
          <DatePicker showTime style={{ width: "100%" }} />
        </Form.Item>

        <Form.Item
          label="原因说明"
          name="reason"
          rules={[
            { required: true, message: "请填写原因" },
            {
              validator: (_, value: string) =>
                value && [...value].length >= 10
                  ? Promise.resolve()
                  : Promise.reject(new Error("原因至少 10 字符(告警风暴缓解强制说明)")),
            },
          ]}
          extra="强制 ≥10 字符,降低 0.0.0.0/0 silence 误配风险(T-44-04)"
        >
          <TextArea rows={3} maxLength={500} showCount placeholder="说明该规则的业务背景,如:研发部测试网段误报高,确认无生产数据" />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default ExceptionRuleForm;
