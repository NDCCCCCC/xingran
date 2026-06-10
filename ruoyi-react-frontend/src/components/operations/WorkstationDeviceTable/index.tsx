/**
 * 工位设备关联子表格组件
 *
 * 用于在工位列表中展示和管理关联的设备
 */

import React, { useState, useEffect, useCallback } from 'react';
import { Table, Button, Space, Tag, Popconfirm, message, Modal, Form, Input, Alert } from 'antd';
import { PlusOutlined, SyncOutlined, EditOutlined, DeleteOutlined, StarOutlined } from '@ant-design/icons';
import type { WorkstationDevice, DeviceFormData, DeviceSource, Asset } from '@/types';
import { DEVICE_SOURCE_LABELS } from '@/types/operations';
import { workstationDeviceApi, assetApi } from '@/lib/opsApi';

export interface WorkstationDeviceTableProps {
  workstationId: string;
  expandable?: boolean;
  onDeviceChange?: () => void;
}

export const WorkstationDeviceTable: React.FC<WorkstationDeviceTableProps> = ({
  workstationId,
  onDeviceChange
}) => {
  const [devices, setDevices] = useState<WorkstationDevice[]>([]);
  const [loading, setLoading] = useState(false);
  const [modalVisible, setModalVisible] = useState(false);
  const [editingDevice, setEditingDevice] = useState<WorkstationDevice | null>(null);
  const [autoFilledAsset, setAutoFilledAsset] = useState<Asset | null>(null);
  const [searchingSerial, setSearchingSerial] = useState(false);
  const [form] = Form.useForm();

  // 序列号搜索函数
  const handleSerialSearch = async (serial: string) => {
    if (!serial) {
      setAutoFilledAsset(null);
      return;
    }
    setSearchingSerial(true);
    try {
      const result = await assetApi.searchBySerial(serial);
      if (result.code === 0 && result.data) {
        setAutoFilledAsset(result.data);
        // 自动填充表单字段
        form.setFieldsValue({
          deviceName: result.data.deviceName,
          deviceModel: result.data.deviceModelName,
          deviceType: result.data.deviceTypeName,
          macAddress: result.data.mac1,
          responsibleUser: result.data.nowUserName,
        });
      } else {
        setAutoFilledAsset(null);
        // 清空自动填充字段
        form.setFieldsValue({
          deviceName: undefined,
          deviceModel: undefined,
          deviceType: undefined,
          macAddress: undefined,
          responsibleUser: undefined,
        });
      }
    } catch (error) {
      setAutoFilledAsset(null);
      // 清空自动填充字段
      form.setFieldsValue({
        deviceName: undefined,
        deviceModel: undefined,
        deviceType: undefined,
        macAddress: undefined,
        responsibleUser: undefined,
      });
    } finally {
      setSearchingSerial(false);
    }
  };

  // 加载设备列表
  const fetchDevices = useCallback(async () => {
    setLoading(true);
    try {
      const result = await workstationDeviceApi.getByWorkstation(workstationId);
      setDevices(result.data || []);
    } catch (error) {
      message.error('加载设备失败');
    } finally {
      setLoading(false);
    }
  }, [workstationId]);

  useEffect(() => {
    fetchDevices();
  }, [fetchDevices]);

  // 添加设备
  const handleAdd = () => {
    setEditingDevice(null);
    form.resetFields();
    setAutoFilledAsset(null);
    setModalVisible(true);
  };

  // 编辑设备
  const handleEdit = (device: WorkstationDevice) => {
    setEditingDevice(device);
    form.setFieldsValue({
      deviceSerial: device.deviceSerial,
      deviceName: device.deviceName,
      deviceModel: device.deviceModel,
      deviceType: device.deviceType,
      macAddress: device.macAddress,
      responsibleUser: device.responsibleUser,
      description: device.description,
    });
    setModalVisible(true);
  };

  // 删除设备
  const handleDelete = async (id: string) => {
    try {
      await workstationDeviceApi.delete(id);
      message.success('删除成功');
      fetchDevices();
      onDeviceChange?.();
    } catch (error) {
      message.error('删除失败');
    }
  };

  // 设置主设备
  const handleSetPrimary = async (id: string) => {
    try {
      await workstationDeviceApi.setPrimary(id);
      message.success('设置成功');
      fetchDevices();
    } catch (error) {
      message.error('设置失败');
    }
  };

  // 同步 AD 设备
  const handleSyncAD = async () => {
    try {
      await workstationDeviceApi.syncAD(workstationId);
      message.success('同步成功');
      fetchDevices();
      onDeviceChange?.();
    } catch (error) {
      message.error('同步失败');
    }
  };

  // 同步资产设备
  const handleSyncAsset = async () => {
    try {
      await workstationDeviceApi.syncAsset(workstationId);
      message.success('同步成功');
      fetchDevices();
      onDeviceChange?.();
    } catch (error) {
      message.error('同步失败');
    }
  };

  // 提交表单
  const handleModalOk = async () => {
    try {
      const values = await form.validateFields();
      const data: DeviceFormData = {
        workstationId,
        deviceSerial: values.deviceSerial,
        deviceName: values.deviceName,
        deviceModel: values.deviceModel,
        deviceType: values.deviceType,
        macAddress: values.macAddress,
        responsibleUser: values.responsibleUser,
        description: values.description,
      };

      if (editingDevice) {
        await workstationDeviceApi.update(editingDevice.id, data);
        message.success('更新成功');
      } else {
        await workstationDeviceApi.addManual(data);
        message.success('添加成功');
      }

      setModalVisible(false);
      fetchDevices();
      onDeviceChange?.();
    } catch (error) {
      message.error(editingDevice ? '更新失败' : '添加失败');
    }
  };

  // 表格列定义
  const columns = [
    {
      title: '设备名称',
      dataIndex: 'deviceName',
      key: 'deviceName',
    },
    {
      title: '序列号',
      dataIndex: 'deviceSerial',
      key: 'deviceSerial',
    },
    {
      title: '来源',
      dataIndex: 'deviceSource',
      key: 'deviceSource',
      render: (source: DeviceSource) => (
        <Tag color={source === 'ad' ? 'blue' : source === 'asset' ? 'green' : 'orange'}>
          {DEVICE_SOURCE_LABELS[source]}
        </Tag>
      ),
    },
    {
      title: '型号',
      dataIndex: 'deviceModel',
      key: 'deviceModel',
    },
    {
      title: 'MAC地址',
      dataIndex: 'macAddress',
      key: 'macAddress',
    },
    {
      title: '责任人',
      dataIndex: 'responsibleUser',
      key: 'responsibleUser',
    },
    {
      title: '状态',
      dataIndex: 'status',
      key: 'status',
      render: (status: number) => (
        <Tag color={status === 0 ? 'success' : 'error'}>
          {status === 0 ? '正常' : '停用'}
        </Tag>
      ),
    },
    {
      title: '主设备',
      dataIndex: 'isPrimary',
      key: 'isPrimary',
      render: (isPrimary: boolean) => (
        isPrimary ? <StarOutlined style={{ color: '#faad14' }} /> : null
      ),
    },
    {
      title: '操作',
      key: 'action',
      render: (_: any, record: WorkstationDevice) => (
        <Space size="small">
          <Button
            type="text"
            size="small"
            icon={<EditOutlined />}
            onClick={() => handleEdit(record)}
            disabled={record.deviceSource === 'ad'}
          >
            编辑
          </Button>
          <Popconfirm
            title="确定删除此设备？"
            onConfirm={() => handleDelete(record.id)}
            okText="确定"
            cancelText="取消"
          >
            <Button
              type="text"
              size="small"
              danger
              icon={<DeleteOutlined />}
              disabled={record.deviceSource === 'ad'}
            >
              删除
            </Button>
          </Popconfirm>
          {!record.isPrimary && (
            <Popconfirm
              title="确定设置为主设备？"
              onConfirm={() => handleSetPrimary(record.id)}
              okText="确定"
              cancelText="取消"
            >
              <Button type="text" size="small" icon={<StarOutlined />}>
                设为主设备
              </Button>
            </Popconfirm>
          )}
        </Space>
      ),
    },
  ];

  return (
    <div style={{ padding: '16px', backgroundColor: '#fafafa' }}>
      <Space style={{ marginBottom: 16 }}>
        <Button
          type="primary"
          icon={<PlusOutlined />}
          onClick={handleAdd}
        >
          手动添加
        </Button>
        <Button
          icon={<SyncOutlined />}
          onClick={handleSyncAD}
        >
          同步AD设备
        </Button>
        <Button
          icon={<SyncOutlined />}
          onClick={handleSyncAsset}
        >
          同步资产设备
        </Button>
      </Space>
      <Table
        columns={columns}
        dataSource={devices}
        loading={loading}
        rowKey="id"
        size="small"
        pagination={false}
      />
      <Modal
        title={editingDevice ? '编辑设备' : '添加设备'}
        open={modalVisible}
        onOk={handleModalOk}
        onCancel={() => setModalVisible(false)}
        destroyOnClose
        width={600}
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="deviceSerial"
            label="序列号"
            rules={[{ required: true, message: '请输入序列号' }]}
            extra={searchingSerial ? '正在查询资产信息...' : autoFilledAsset ? '已从资产系统自动匹配' : '未在资产系统中找到'}
          >
            <Input
              placeholder="请输入设备序列号"
              onBlur={(e) => handleSerialSearch(e.target.value)}
              disabled={!!editingDevice}
            />
          </Form.Item>

          {autoFilledAsset && (
            <div style={{ marginBottom: 16, padding: 12, background: '#f5f5f5', borderRadius: 4 }}>
              <div style={{ marginBottom: 8, fontWeight: 'bold' }}>自动匹配的设备信息：</div>
              <div>设备名称: {autoFilledAsset.deviceName || '-'}</div>
              <div>设备型号: {autoFilledAsset.deviceModelName || '-'}</div>
              <div>设备类型: {autoFilledAsset.deviceTypeName || '-'}</div>
              <div>MAC地址: {autoFilledAsset.mac1 || '-'}</div>
              <div>责任人: {autoFilledAsset.nowUserName || '-'}</div>
            </div>
          )}

          {!autoFilledAsset && form.getFieldValue('deviceSerial') && (
            <Alert
              message="未找到资产信息"
              description="该序列号在资产系统中不存在，您仍可以添加设备，但需要手动填写设备信息"
              type="warning"
              showIcon
              style={{ marginBottom: 16 }}
            />
          )}

          <Form.Item name="description" label="备注">
            <Input.TextArea placeholder="请输入备注（可选）" rows={2} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default WorkstationDeviceTable;
