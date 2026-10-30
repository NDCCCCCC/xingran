import type { FC } from "react";
import { Modal, Form, Row, Col, InputNumber, DatePicker } from "antd";
import type { FormInstance } from "antd";
import type { RoomDevice } from "@/types";
import { AppInput, AppSelect, AppTextArea } from "@/components/ui";
import { ROOM_DEVICE_TYPE_OPTIONS, ROOM_DEVICE_STATUS_OPTIONS } from "../constants";

export interface RoomDeviceModalProps {
  visible: boolean;
  editingDevice: RoomDevice | null;
  form: FormInstance;
  submitting: boolean;
  roomSelectOptions: Array<{ label: string; value: string }>;
  onOk: () => void;
  onCancel: () => void;
}

export const RoomDeviceModal: FC<RoomDeviceModalProps> = ({
  visible,
  editingDevice,
  form,
  submitting,
  roomSelectOptions,
  onOk,
  onCancel,
}) => {
  return (
    <Modal
      title={editingDevice ? "编辑设备" : "新增设备"}
      open={visible}
      onOk={onOk}
      cancelButtonProps={{ disabled: submitting }}
      confirmLoading={submitting}
      onCancel={onCancel}
      width={700}
    >
      <Form form={form} layout="horizontal" labelCol={{ span: 5 }} wrapperCol={{ span: 19 }}>
        <Form.Item
          name="roomId"
          label="所属机房"
          rules={[{ required: true, message: "请选择所属机房" }]}
        >
          <AppSelect
            allowClear
            placeholder="请选择所属机房"
            showSearch
            filterOption={(input, option) =>
              String(option?.label ?? "")
                .toLowerCase()
                .includes(input.toLowerCase())
            }
            onSearch={() => {}}
            options={roomSelectOptions}
          />
        </Form.Item>
        <Form.Item
          name="name"
          label="设备名称"
          rules={[{ required: true, message: "请输入设备名称" }]}
        >
          <AppInput placeholder="请输入设备名称" />
        </Form.Item>
        <Form.Item
          name="deviceCode"
          label="设备编码"
          rules={[{ required: true, message: "请输入设备编码" }]}
        >
          <AppInput placeholder="请输入设备编码" />
        </Form.Item>
        <Form.Item
          name="deviceType"
          label="设备类型"
          rules={[{ required: true, message: "请选择设备类型" }]}
        >
          <AppSelect
            allowClear
            placeholder="请选择设备类型"
            showSearch={false}
            onSearch={() => {}}
            options={ROOM_DEVICE_TYPE_OPTIONS}
          />
        </Form.Item>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="vendor" label="厂商">
              <AppInput placeholder="请输入厂商" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="model" label="型号">
              <AppInput placeholder="请输入型号" />
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="serialNumber" label="序列号">
              <AppInput placeholder="请输入序列号" />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="positionDesc" label="位置描述">
              <AppInput placeholder="请输入位置描述" />
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="positionU" label="U位置">
              <InputNumber min={0} placeholder="U位置" style={{ width: "100%" }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="heightU" label="高度(U)">
              <InputNumber min={0} placeholder="高度" style={{ width: "100%" }} />
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="powerConsumption" label="功耗(W)">
              <InputNumber min={0} placeholder="功耗" style={{ width: "100%" }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item
              name="status"
              label="状态"
              rules={[{ required: true, message: "请选择状态" }]}
            >
              <AppSelect
                allowClear
                placeholder="请选择状态"
                showSearch={false}
                onSearch={() => {}}
                options={ROOM_DEVICE_STATUS_OPTIONS}
              />
            </Form.Item>
          </Col>
        </Row>
        <Row gutter={16}>
          <Col span={12}>
            <Form.Item name="purchaseDate" label="购买日期">
              <DatePicker style={{ width: "100%" }} />
            </Form.Item>
          </Col>
          <Col span={12}>
            <Form.Item name="warrantyDate" label="保修到期">
              <DatePicker style={{ width: "100%" }} />
            </Form.Item>
          </Col>
        </Row>
        <Form.Item name="description" label="描述">
          <AppTextArea rows={3} placeholder="请输入描述" />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default RoomDeviceModal;
