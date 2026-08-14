import { useEffect } from "react";
import type { FC } from "react";
import { Modal, Form, Input, InputNumber, Select } from "antd";
import type { FormInstance } from "antd/es/form";
import type { Floor, Building } from "@/types";
import { STATUS_OPTIONS, DEFAULT_FORM_VALUES } from "../constants";
import { DepartmentTreeSelect, type Department } from "@/components/shared/DepartmentTreeSelect";
import FileUpload from "@/components/shared/FileUpload";
import { useImageUpload } from "@/hooks/useImageUpload";
import { MAX_IMAGE_SIZE } from "@/constants/upload";

const { Option } = Select;
const { TextArea } = Input;

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
    } else {
      planImageUpload.resetUpload();
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

  // 合并楼宇选项：优先使用部门特定选项，如果为空则使用全部选项
  const effectiveBuildingOptions =
    buildingOptionsByDept.length > 0 ? buildingOptionsByDept : buildingOptions;

  return (
    <Modal
      title={editingFloor ? "编辑楼层" : "新增楼层"}
      open={visible}
      onOk={handleOk}
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
          <Select
            placeholder="请选择所属楼宇"
            disabled={isDisabled}
            showSearch
            optionFilterProp="children"
            onSearch={() => {}}
          >
            {effectiveBuildingOptions.map((b) => (
              <Option key={b.id} value={b.id}>
                {b.name}
              </Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item
          name="floorNo"
          label="楼层号"
          rules={[{ required: true, message: "请输入楼层号" }]}
        >
          <Input placeholder="请输入楼层号，如：1F" disabled={!!editingFloor} />
        </Form.Item>

        <Form.Item name="name" label="楼层名称">
          <Input placeholder="请输入楼层名称" />
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
          <Select placeholder="请选择状态" onSearch={() => {}}>
            {STATUS_OPTIONS.map((opt) => (
              <Option key={opt.value} value={opt.value}>
                {opt.label}
              </Option>
            ))}
          </Select>
        </Form.Item>

        <Form.Item name="description" label="描述">
          <TextArea rows={3} placeholder="请输入描述" />
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
