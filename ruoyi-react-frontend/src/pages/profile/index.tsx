import React, { useState, useEffect } from 'react';
import {
  Card,
  Row,
  Col,
  Descriptions,
  Avatar,
  Button,
  Modal,
  Form,
  Input,
  Select,
  message,
  Upload,
} from 'antd';
import {
  UserOutlined,
  EditOutlined,
  LockOutlined,
  UploadOutlined,
} from '@ant-design/icons';
import type { UploadFile } from 'antd/es/upload/interface';
import { getProfileInfo, updateProfileInfo, changePassword, uploadAvatar } from '@/lib/profileApi';
import { useAuthStore } from '@/store/authStore';
import type { UserProfile } from '@/types';

const { Option } = Select;

const ProfilePage: React.FC = () => {
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [loading, setLoading] = useState(false);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [passwordModalVisible, setPasswordModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [passwordForm] = Form.useForm();
  const [avatarFileList, setAvatarFileList] = useState<UploadFile[]>([]);
  const { updateUser } = useAuthStore();

  // 加载个人信息
  const loadProfile = async () => {
    setLoading(true);
    try {
      const data = await getProfileInfo();
      setProfile(data);
    } catch (error) {
      message.error('加载个人信息失败');
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    loadProfile();
  }, []);

  // 更新个人信息
  const handleUpdateProfile = async () => {
    try {
      const values = await editForm.validateFields();
      await updateProfileInfo(values);
      message.success('更新成功');
      setEditModalVisible(false);
      editForm.resetFields();

      // 更新全局用户信息
      updateUser(values);
      loadProfile();
    } catch (error: any) {
      if (error.errorFields) return;
      message.error('更新失败');
    }
  };

  // 打开编辑弹窗
  const openEditModal = () => {
    if (profile) {
      editForm.setFieldsValue({
        nickname: profile.nickname,
        email: profile.email,
        phone: profile.phone,
        gender: profile.gender,
        remark: profile.remark,
      });
    }
    setEditModalVisible(true);
  };

  // 修改密码
  const handleChangePassword = async () => {
    try {
      const values = await passwordForm.validateFields();
      await changePassword(values);
      message.success('密码修改成功，请重新登录');
      setPasswordModalVisible(false);
      passwordForm.resetFields();

      // 延迟后登出
      setTimeout(() => {
        useAuthStore.getState().logout();
        window.location.href = '/login';
      }, 1500);
    } catch (error: any) {
      if (error.errorFields) return;
      if (error?.message?.includes('旧密码错误')) {
        message.error('旧密码错误');
      } else {
        message.error('密码修改失败');
      }
    }
  };

  // 上传头像
  const handleUploadAvatar = async (file: File) => {
    try {
      const result = await uploadAvatar(file);
      message.success('头像上传成功');

      // 更新全局用户信息
      updateUser({ avatar: result.avatar });
      loadProfile();
    } catch (error) {
      message.error('头像上传失败');
    }
    return false; // 阻止默认上传行为
  };

  if (!profile) {
    return <div>加载中...</div>;
  }

  return (
    <div className="p-6">
      <Row gutter={[16, 16]}>
        <Col xs={24} lg={8}>
          <Card title="个人头像" className="text-center">
            <Avatar size={120} src={profile.avatar} icon={<UserOutlined />} />
            <div className="mt-4">
              <Upload
                beforeUpload={handleUploadAvatar}
                showUploadList={false}
                accept="image/*"
              >
                <Button icon={<UploadOutlined />}>更换头像</Button>
              </Upload>
            </div>
          </Card>
        </Col>

        <Col xs={24} lg={16}>
          <Card
            title="基本信息"
            extra={
              <Button icon={<EditOutlined />} onClick={openEditModal}>
                编辑
              </Button>
            }
          >
            <Descriptions column={2} bordered>
              <Descriptions.Item label="用户名">{profile.username}</Descriptions.Item>
              <Descriptions.Item label="昵称">{profile.nickname || '-'}</Descriptions.Item>
              <Descriptions.Item label="性别">
                {profile.gender === 0 ? '未知' : profile.gender === 1 ? '男' : '女'}
              </Descriptions.Item>
              <Descriptions.Item label="邮箱">{profile.email || '-'}</Descriptions.Item>
              <Descriptions.Item label="手机号">{profile.phone || '-'}</Descriptions.Item>
              <Descriptions.Item label="部门">{profile.deptName || '-'}</Descriptions.Item>
              <Descriptions.Item label="状态">
                {profile.status === 0 ? (
                  <span style={{ color: 'green' }}>正常</span>
                ) : (
                  <span style={{ color: 'red' }}>禁用</span>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="最后登录IP">{profile.loginIp || '-'}</Descriptions.Item>
              <Descriptions.Item label="最后登录时间">{profile.loginTime || '-'}</Descriptions.Item>
              <Descriptions.Item label="密码修改时间">{profile.pwdUpdateTime || '-'}</Descriptions.Item>
              <Descriptions.Item label="备注" span={2}>
                {profile.remark || '-'}
              </Descriptions.Item>
            </Descriptions>

            <div className="mt-4">
              <Button icon={<LockOutlined />} onClick={() => setPasswordModalVisible(true)}>
                修改密码
              </Button>
            </div>
          </Card>
        </Col>
      </Row>

      {/* 编辑个人信息弹窗 */}
      <Modal
        title="编辑个人信息"
        open={editModalVisible}
        onOk={handleUpdateProfile}
        onCancel={() => setEditModalVisible(false)}
        width={600}
      >
        <Form form={editForm} layout="vertical">
          <Form.Item
            label="昵称"
            name="nickname"
            rules={[{ max: 64, message: '昵称长度不能超过64个字符' }]}
          >
            <Input placeholder="请输入昵称" />
          </Form.Item>

          <Form.Item
            label="邮箱"
            name="email"
            rules={[{ type: 'email', message: '请输入有效的邮箱地址' }]}
          >
            <Input placeholder="请输入邮箱" />
          </Form.Item>

          <Form.Item
            label="手机号"
            name="phone"
            rules={[{ max: 32, message: '手机号长度不能超过32个字符' }]}
          >
            <Input placeholder="请输入手机号" />
          </Form.Item>

          <Form.Item
            label="性别"
            name="gender"
            initialValue={0}
          >
            <Select>
              <Option value={0}>未知</Option>
              <Option value={1}>男</Option>
              <Option value={2}>女</Option>
            </Select>
          </Form.Item>

          <Form.Item
            label="备注"
            name="remark"
            rules={[{ max: 500, message: '备注长度不能超过500个字符' }]}
          >
            <Input.TextArea rows={3} placeholder="请输入备注" />
          </Form.Item>
        </Form>
      </Modal>

      {/* 修改密码弹窗 */}
      <Modal
        title="修改密码"
        open={passwordModalVisible}
        onOk={handleChangePassword}
        onCancel={() => setPasswordModalVisible(false)}
        width={500}
      >
        <Form form={passwordForm} layout="vertical">
          <Form.Item
            label="旧密码"
            name="oldPassword"
            rules={[{ required: true, message: '请输入旧密码' }]}
          >
            <Input.Password placeholder="请输入旧密码" />
          </Form.Item>

          <Form.Item
            label="新密码"
            name="newPassword"
            rules={[
              { required: true, message: '请输入新密码' },
              { min: 6, message: '密码长度不能少于6个字符' },
              { max: 20, message: '密码长度不能超过20个字符' },
            ]}
          >
            <Input.Password placeholder="请输入新密码（6-20个字符）" />
          </Form.Item>

          <Form.Item
            label="确认密码"
            name="confirmPassword"
            dependencies={['newPassword']}
            rules={[
              { required: true, message: '请确认新密码' },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue('newPassword') === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error('两次输入的密码不一致'));
                },
              }),
            ]}
          >
            <Input.Password placeholder="请再次输入新密码" />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
};

export default ProfilePage;
