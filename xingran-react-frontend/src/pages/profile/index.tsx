import { useState, useEffect } from "react";
import type { FC } from "react";
import {
  App,
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
  Upload,
  Statistic,
  Tag,
  Space,
} from "antd";
import {
  UserOutlined,
  EditOutlined,
  LockOutlined,
  UploadOutlined,
  CalendarOutlined,
} from "@ant-design/icons";
import { getProfileInfo, updateProfileInfo, changePassword, uploadAvatar } from "@/lib/profileApi";
import { getMyDutyStats, type MyDutyStats } from "@/lib/dutyApi";
import { useAuthStore } from "@/store/authStore";
import { isFormValidationError } from "@/utils/errorHandler";
import { LOGIN, DUTY_MY_DUTY } from "@/constants/routes";
import type { UserProfile } from "@/types";
import { formatDateTime } from "@/utils/datetime";
import { Link } from "react-router-dom";

const { Option } = Select;

const ProfilePage: FC = () => {
  const { message } = App.useApp();
  const [profile, setProfile] = useState<UserProfile | null>(null);
  const [editModalVisible, setEditModalVisible] = useState(false);
  const [passwordModalVisible, setPasswordModalVisible] = useState(false);
  const [editForm] = Form.useForm();
  const [passwordForm] = Form.useForm();
  const [dutyStats, setDutyStats] = useState<MyDutyStats | null>(null);
  const [dutyStatsLoading, setDutyStatsLoading] = useState(false);
  const { updateUser } = useAuthStore();

  // 加载个人信息
  const loadProfile = async () => {
    try {
      const data = await getProfileInfo();
      setProfile(data);
    } catch (error) {
      message.error("加载个人信息失败");
    }
  };

  // 加载值班统计
  const loadDutyStats = async () => {
    setDutyStatsLoading(true);
    try {
      const result = await getMyDutyStats();
      setDutyStats(result.data || null);
    } catch (error) {
      console.error("加载值班统计失败", error);
    } finally {
      setDutyStatsLoading(false);
    }
  };

  useEffect(() => {
    loadProfile();
    loadDutyStats();
    // eslint-disable-next-line react-hooks/exhaustive-deps -- intentional mount-only load
  }, []);

  // 更新个人信息
  const handleUpdateProfile = async () => {
    try {
      const values = await editForm.validateFields();
      await updateProfileInfo(values);
      message.success("更新成功");
      setEditModalVisible(false);
      editForm.resetFields();

      // 更新全局用户信息
      updateUser(values);
      loadProfile();
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      message.error("更新失败");
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
      message.success("密码修改成功，请重新登录");
      setPasswordModalVisible(false);
      passwordForm.resetFields();

      // 延迟后登出
      setTimeout(async () => {
        try {
          await useAuthStore.getState().logout();
          // 等待 React 状态更新完成
          await new Promise(resolve => setTimeout(resolve, 0));
          window.location.href = LOGIN;
        } catch (error) {
          console.error("自动登出失败:", error);
          window.location.href = LOGIN;
        }
      }, 1500);
    } catch (error: unknown) {
      if (isFormValidationError(error)) return;
      if ((error as { message?: string })?.message?.includes("旧密码错误")) {
        message.error("旧密码错误");
      } else {
        message.error("密码修改失败");
      }
    }
  };

  // 上传头像
  const handleUploadAvatar = async (file: File) => {
    try {
      const result = await uploadAvatar(file);
      message.success("头像上传成功");

      // 更新全局用户信息
      updateUser({ avatar: result.avatar });
      loadProfile();
    } catch (error) {
      message.error("头像上传失败");
    }
    return false; // 阻止默认上传行为
  };

  // 获取值班类型颜色
  const getDutyTypeColor = (type: string) => {
    switch (type) {
      case "weekday": return "blue";
      case "weekend": return "orange";
      case "holiday": return "red";
      default: return "default";
    }
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
              <Descriptions.Item label="昵称">{profile.nickname || "-"}</Descriptions.Item>
              <Descriptions.Item label="性别">
                {profile.gender === 0 ? "未知" : profile.gender === 1 ? "男" : "女"}
              </Descriptions.Item>
              <Descriptions.Item label="邮箱">{profile.email || "-"}</Descriptions.Item>
              <Descriptions.Item label="手机号">{profile.phone || "-"}</Descriptions.Item>
              <Descriptions.Item label="部门">{profile.deptName || "-"}</Descriptions.Item>
              <Descriptions.Item label="状态">
                {profile.status === 0 ? (
                  <span style={{ color: "green" }}>正常</span>
                ) : (
                  <span style={{ color: "red" }}>禁用</span>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="最后登录IP">{profile.loginIp || "-"}</Descriptions.Item>
              <Descriptions.Item label="最后登录时间">{profile.loginTime || "-"}</Descriptions.Item>
              <Descriptions.Item label="密码修改时间">{profile.pwdUpdateTime || "-"}</Descriptions.Item>
              <Descriptions.Item label="备注" span={2}>
                {profile.remark || "-"}
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

      {/* 值班摘要卡片 */}
      <Row gutter={[16, 16]} className="mt-4">
        <Col span={24}>
          <Card
            title={
              <Space>
                <CalendarOutlined />
                <span>我的值班</span>
              </Space>
            }
            extra={
              <Link to={DUTY_MY_DUTY}>
                <Button type="link">
                  查看详情 →
                </Button>
              </Link>
            }
            loading={dutyStatsLoading}
          >
            {dutyStats && (
              <Row gutter={[16, 16]}>
                <Col xs={12} sm={6}>
                  <Statistic
                    title="今日状态"
                    value={dutyStats.isOnDutyToday ? "值班中" : "休息"}
                    styles={{ content: { color: dutyStats.isOnDutyToday ? "var(--theme-success, #3f8600)" : "var(--theme-text-tertiary, #8c8c8c)" } }}
                  />
                </Col>
                <Col xs={12} sm={6}>
                  <Statistic
                    title="本月次数"
                    value={dutyStats.thisMonthCount}
                    suffix="次"
                  />
                </Col>
                <Col xs={12} sm={6}>
                  <Statistic
                    title="累计值班"
                    value={dutyStats.totalCount}
                    suffix="次"
                  />
                </Col>
                <Col xs={12} sm={6}>
                  <div>
                    <div style={{ fontSize: "14px", color: "rgba(0,0,0,0.45)", marginBottom: "4px" }}>下次值班</div>
                    <div style={{ fontSize: "20px", fontWeight: 500 }}>
                      {dutyStats.nextDutyDate ? (
                        <Space>
                          <span>{formatDateTime(dutyStats.nextDutyDate, "MM-DD")}</span>
                          {dutyStats.nextDutyPoolName && (
                            <Tag color="blue">{dutyStats.nextDutyPoolName}</Tag>
                          )}
                        </Space>
                      ) : (
                        <span style={{ color: "var(--theme-text-tertiary, #8c8c8c)" }}>暂无排班</span>
                      )}
                    </div>
                  </div>
                </Col>
              </Row>
            )}

            {/* 今日值班详情 */}
            {dutyStats?.isOnDutyToday && dutyStats.todayDutyRecords && dutyStats.todayDutyRecords.length > 0 && (
              <div className="mt-4 p-3 bg-green-50 border border-green-200 rounded">
                <div style={{ fontWeight: "bold", marginBottom: "8px" }}>今日值班任务</div>
                <Space wrap>
                  {dutyStats.todayDutyRecords.map((record, index) => (
                    <Tag key={index} color={getDutyTypeColor(record.dutyType)}>
                      {record.poolName} ({record.dutyType === "weekday" ? "工作日" : record.dutyType === "weekend" ? "周末" : "节假日"})
                    </Tag>
                  ))}
                </Space>
              </div>
            )}
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
            rules={[{ max: 64, message: "昵称长度不能超过64个字符" }]}
          >
            <Input placeholder="请输入昵称" />
          </Form.Item>

          <Form.Item
            label="邮箱"
            name="email"
            rules={[{ type: "email", message: "请输入有效的邮箱地址" }]}
          >
            <Input placeholder="请输入邮箱" />
          </Form.Item>

          <Form.Item
            label="手机号"
            name="phone"
            rules={[{ max: 32, message: "手机号长度不能超过32个字符" }]}
          >
            <Input placeholder="请输入手机号" />
          </Form.Item>

          <Form.Item
            label="性别"
            name="gender"
            initialValue={0}
          >
            <Select onSearch={() => {}}>
              <Option value={0}>未知</Option>
              <Option value={1}>男</Option>
              <Option value={2}>女</Option>
            </Select>
          </Form.Item>

          <Form.Item
            label="备注"
            name="remark"
            rules={[{ max: 500, message: "备注长度不能超过500个字符" }]}
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
            rules={[{ required: true, message: "请输入旧密码" }]}
          >
            <Input.Password placeholder="请输入旧密码" />
          </Form.Item>

          <Form.Item
            label="新密码"
            name="newPassword"
            rules={[
              { required: true, message: "请输入新密码" },
              { min: 6, message: "密码长度不能少于6个字符" },
              { max: 20, message: "密码长度不能超过20个字符" },
            ]}
          >
            <Input.Password placeholder="请输入新密码（6-20个字符）" />
          </Form.Item>

          <Form.Item
            label="确认密码"
            name="confirmPassword"
            dependencies={["newPassword"]}
            rules={[
              { required: true, message: "请确认新密码" },
              ({ getFieldValue }) => ({
                validator(_, value) {
                  if (!value || getFieldValue("newPassword") === value) {
                    return Promise.resolve();
                  }
                  return Promise.reject(new Error("两次输入的密码不一致"));
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
