import { useEffect, useState } from "react";
import type { FC } from "react";
import { Modal, Form, InputNumber } from "antd";
import type { FormInstance } from "antd/es/form";
import type { Floor, Building } from "@/types";
import { STATUS_OPTIONS, DEFAULT_FORM_VALUES } from "../constants";
import { DepartmentTreeSelect, type Department } from "@/components/shared/DepartmentTreeSelect";
import FileUpload from "@/components/shared/FileUpload";
import { useImageUpload } from "@/hooks/useImageUpload";
import { MAX_IMAGE_SIZE } from "@/constants/upload";
// Phase 71 (W3)：裸 Input/Select/TextArea → 契约组件。
// 楼宇下拉原为 children + showSearch（optionFilterProp="children"）→ 改写 options 形态，
// 由 AppSelect 统一下沉 optionFilterProp="label"；状态枚举业务禁搜 → 显式 showSearch={false}。
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";

// 楼宇选项基础类型（只需要 id 和 name）
type BuildingOptionBase = Pick<Building, "id" | "name">;

interface FloorModalProps {
  visible: boolean;
  editingFloor: Floor | null;
  buildingOptions: BuildingOptionBase[];
  departments: Department[];
  buildingOptionsByDept: BuildingOptionBase[];
  selectedDeptId: string;
  form: FormInstance;
  onOk: () => void;
  onCancel: () => void;
  onDepartmentChange: (deptId: string) => void;
}

export const FloorModal: FC<FloorModalProps> = ({
  visible,
  editingFloor,
  buildingOptions,
  departments,
  buildingOptionsByDept,
  selectedDeptId,
  form,
  onOk,
  onCancel,
  onDepartmentChange,
}) => {
  const planImageUpload = useImageUpload({
    businessType: "floor-plan",
    maxSize: MAX_IMAGE_SIZE,
  });

  useEffect(() => {
    if (!visible) return;

    if (editingFloor?.planImageId) {
      planImageUpload.setInitialValue(
        editingFloor.planImageId,
        editingFloor.planImageUrl ?? undefined
      );
      // 同步进 form store：用户只改其它字段（不动图片）时保存必须带上原图 id，
      // 否则 getFieldValue("planImageId") 为空，update 会丢掉既有底图。
      form.setFieldValue("planImageId", editingFloor.planImageId);
    } else {
      planImageUpload.resetUpload();
      form.setFieldValue("planImageId", undefined);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [visible, editingFloor?.id, editingFloor?.planImageId, editingFloor?.planImageUrl]);

  const handleCancel = () => {
    form.resetFields();
    planImageUpload.resetUpload();
    onCancel();
  };

  const handleOk = async () => {
    form.setFieldValue("planImageId", planImageUpload.imageId);
    onOk();
  };

  const isDisabled = !!editingFloor;
  const [submitting, setSubmitting] = useState(false);

  // 合并楼宇选项：优先使用部门特定选项，如果为空则使用全部选项
  const effectiveBuildingOptions =
    buildingOptionsByDept.length > 0 ? buildingOptionsByDept : buildingOptions;

  return (
    <Modal
      title={editingFloor ? "编辑楼层" : "新增楼层"}
      open={visible}
      onOk={async () => {
        setSubmitting(true);
        try {
          await handleOk();
        } finally {
          setSubmitting(false);
        }
      }}
      cancelButtonProps={{ disabled: submitting }}
      onCancel={handleCancel}
      width={600}
    >
      <Form form={form} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
        <Form.Item label="所属机构" required>
          <DepartmentTreeSelect
            placeholder="选择机构"
            departments={departments}
            value={selectedDeptId}
            onChange={onDepartmentChange}
            allowClear={!isDisabled}
            disabled={isDisabled}
          />
        </Form.Item>

        <Form.Item
          name="buildingId"
          label="所属楼宇"
          rules={[{ required: true, message: "请选择所属楼宇" }]}
        >
          {/* 形态分类器②③：children + 复合子节点（属性过滤）→ options 改写。
              楼宇 ID 为 UUID：allowClear 原就有，清空后表单值 undefined，由
              Form rules={required} 在提交前拦截，不会把 undefined 送进 UUID 校验。 */}
          <AppSelect
            allowClear
            placeholder="请选择所属楼宇"
            disabled={isDisabled}
            showSearch
            options={effectiveBuildingOptions.map((b) => ({ value: b.id, label: b.name }))}
          />
        </Form.Item>

        {/* 一栋楼跨机构办公（quick-261106-wso）：楼层可覆盖所属机构。
            与上方「所属机构」不同——上方是楼宇下拉的过滤器（编辑态禁用），
            本项才真正写进 ops_floors.org_id。留空 = 继承楼宇 org_id。
            编辑态也允许修改，否则存量楼层无法补设覆盖值。 */}
        <Form.Item
          name="orgId"
          label="楼层归属机构"
          extra="留空表示继承所属楼宇的机构；仅当本楼层与其它楼层分属不同机构时填写"
        >
          <DepartmentTreeSelect
            placeholder="留空则继承楼宇机构"
            departments={departments}
            allowClear
          />
        </Form.Item>

        <Form.Item
          name="floorNo"
          label="楼层号"
          rules={[{ required: true, message: "请输入楼层号" }]}
        >
          <AppInput placeholder="请输入楼层号，如：1F" disabled={!!editingFloor} />
        </Form.Item>

        <Form.Item name="name" label="楼层名称">
          <AppInput placeholder="请输入楼层名称" />
        </Form.Item>

        <Form.Item name="area" label="面积">
          <InputNumber min={0} placeholder="请输入面积(m²)" style={{ width: "100%" }} />
        </Form.Item>

        <Form.Item
          name="status"
          label="状态"
          rules={[{ required: true, message: "请选择状态" }]}
          initialValue={DEFAULT_FORM_VALUES.status}
        >
          {/* 形态分类器②④：状态枚举（业务禁搜），children → options */}
          <AppSelect
            allowClear
            placeholder="请选择状态"
            showSearch={false}
            options={STATUS_OPTIONS}
          />
        </Form.Item>

        <Form.Item name="description" label="描述">
          <AppTextArea rows={3} placeholder="请输入描述" />
        </Form.Item>

        <Form.Item label="平面图">
          <FileUpload
            category="floor-plan"
            accept="image/*"
            maxSize={MAX_IMAGE_SIZE}
            maxCount={1}
            listType="picture-card"
            value={planImageUpload.fileList}
            onChange={planImageUpload.handleUploadChange}
            onUploadSuccess={planImageUpload.handleUploadSuccess}
            onUploadError={planImageUpload.handleUploadError}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
};
