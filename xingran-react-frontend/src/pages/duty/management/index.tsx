/**
 * 值班管理页面
 * Duty Management Page
 */

import { useState, useEffect, useCallback } from "react";
import { useLocation } from "react-router-dom";
import { usePersistedStateController } from "@/hooks/usePersistedState";
import { Tabs, Button, Modal, Form } from "antd";
import { LeftOutlined, RightOutlined, CalendarOutlined, SettingOutlined, TeamOutlined } from "@ant-design/icons";
import dayjs from "dayjs";
import { getDutyPoolList, getUserList, type Holiday } from "@/lib/dutyApi";
import type { DutyPool, SimpleUser } from "@/lib/dutyApi";
import {
  WeeklyView,
  HolidayManagement,
  DutyConfig,
} from "./components";
import { HolidayModal, BatchHolidayModal } from "./modals";
import { useScheduleData, useHolidayData, useDutyConfig } from "./hooks";
import { downloadHolidayTemplate, handleHolidayImport } from "./utils";
import type { DutyConfigValues, ImportOptions, BatchHolidayFormValues } from "./types";
import DutyPoolPage from "../pools";

// ==================== 主组件 ====================

export default function DutyManagement() {
  // ==================== 排班数据 ====================
  const scheduleData = useScheduleData();

  // ==================== 节假日数据 ====================
  const holidayData = useHolidayData();

  // ==================== 值班配置 ====================
  const dutyConfig = useDutyConfig();

  // ==================== 其他状态 ====================
  const location = useLocation();
  const [activeTab, setActiveTab] = usePersistedStateController<string>({
    keyPrefix: location.pathname,
    keySuffix: "activeTab",
    defaultValue: "weekly",
  });
  const [_pools, setPools] = useState<DutyPool[]>([]);
  const [_users, setUsers] = useState<SimpleUser[]>([]);

  // 模态框状态
  const [holidayModalVisible, setHolidayModalVisible] = useState(false);
  const [holidayBatchModalVisible, setHolidayBatchModalVisible] = useState(false);

  // 表单
  const [holidayForm] = Form.useForm();

  // ==================== 初始化 ====================

  useEffect(() => {
    fetchPools();
    fetchUsers();
    scheduleData.fetchWeeklyDuty(scheduleData.currentWeekStart);
    holidayData.fetchYears();
    dutyConfig.fetch();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // ==================== 数据获取 ====================

  const fetchPools = async () => {
    try {
      const result = await getDutyPoolList({ current: 1, pageSize: 100 });
      setPools(result.data?.list || []);
    } catch (_error) {
      Modal.error({ title: "获取值班池列表失败" });
    }
  };

  const fetchUsers = async () => {
    try {
      const result = await getUserList({ current: 1, pageSize: 50 });
      setUsers(result.data?.list || []);
    } catch (error) {
      console.error("获取用户列表失败", error);
    }
  };

  // ==================== 节假日操作 ====================

  const handleHolidayYearChange = useCallback((year: number) => {
    holidayData.setHolidayYear(year);
    holidayData.fetchList(year);
  }, [holidayData]);

  const handleHolidayAdd = useCallback(() => {
    setHolidayModalVisible(true);
  }, []);

  const handleHolidayBatchAdd = useCallback(() => {
    setHolidayBatchModalVisible(true);
  }, []);

  const handleHolidayEdit = useCallback((record: Holiday) => {
    holidayForm.setFieldsValue({
      ...record,
      holidayDate: dayjs(record.holidayDate),
    });
    setHolidayModalVisible(true);
  }, [holidayForm]);

  const handleHolidayDelete = useCallback(async (id: string) => {
    await holidayData.deleteOne(id);
  }, [holidayData]);

  const handleHolidayModalOk = useCallback(async () => {
    try {
      const values = await holidayForm.validateFields();
      const data = {
        holidayDate: values.holidayDate.format("YYYY-MM-DD"),
        holidayName: values.holidayName,
        isOffday: values.isOffday ?? true,
        holidayType: (values.holidayType || "custom") as "legal" | "workday" | "custom",
        year: values.holidayDate.year(),
        remark: values.remark,
      };

      const id = holidayForm.getFieldValue("id");
      if (id) {
        await holidayData.update(id, data);
      } else {
        await holidayData.create(data);
      }
      setHolidayModalVisible(false);
    } catch (_error) {
      // 表单验证失败
    }
  }, [holidayData, holidayForm]);

  const handleImport = useCallback((options: ImportOptions) => {
    handleHolidayImport(options, holidayData.batchCreate);
  }, [holidayData]);

  const handleBatchHolidayOk = useCallback(async (values: BatchHolidayFormValues) => {
    const { dateRange, holidayName, holidayType, isOffday } = values;

    const startDate = dateRange[0];
    const endDate = dateRange[1];
    const holidays = [];
    let current = startDate;

    while (current.isBefore(endDate) || current.isSame(endDate, "day")) {
      holidays.push({
        holidayDate: current.format("YYYY-MM-DD"),
        holidayName,
        isOffday: isOffday ?? true,
        holidayType: holidayType || "custom",
        year: current.year(),
        remark: values.remark,
      });
      current = current.add(1, "day");
    }

    await holidayData.batchCreate(holidays);
    setHolidayBatchModalVisible(false);
  }, [holidayData]);

  // ==================== 配置保存 ====================

  const handleConfigSave = useCallback(async (values: DutyConfigValues) => {
    return await dutyConfig.save(values);
  }, [dutyConfig]);

  // ==================== 渲染 ====================

  return (
    <div>
      <Tabs
        activeKey={activeTab}
        onChange={setActiveTab}
        items={[
          {
            key: "weekly",
            label: (
              <span>
                <CalendarOutlined />
                周视图
              </span>
            ),
            children: (
              <>
                <div style={{ marginBottom: 16, display: "flex", justifyContent: "flex-end", gap: 8 }}>
                  <Button icon={<LeftOutlined />} onClick={scheduleData.prevWeek}>
                    上一周
                  </Button>
                  <Button onClick={scheduleData.todayWeek}>
                    今天
                  </Button>
                  <Button icon={<RightOutlined />} onClick={scheduleData.nextWeek}>
                    下一周
                  </Button>
                </div>
                <WeeklyView
                  currentWeekStart={scheduleData.currentWeekStart}
                  weeklyDutyData={scheduleData.weeklyDutyData}
                />
              </>
            ),
          },
          {
            key: "holiday",
            label: "节假日管理",
            children: (
              <HolidayManagement
                holidays={holidayData.holidays}
                loading={holidayData.loading}
                holidayYear={holidayData.holidayYear}
                availableYears={holidayData.availableYears}
                onYearChange={handleHolidayYearChange}
                onRefresh={() => holidayData.fetchList()}
                onAdd={handleHolidayAdd}
                onBatchAdd={handleHolidayBatchAdd}
                onEdit={handleHolidayEdit}
                onDelete={handleHolidayDelete}
                onImport={handleImport}
                onDownloadTemplate={downloadHolidayTemplate}
              />
            ),
          },
          {
            key: "config",
            label: (
              <span>
                <SettingOutlined />
                值班配置
              </span>
            ),
            children: (
              <DutyConfig
                config={dutyConfig.config}
                loading={dutyConfig.loading}
                saving={dutyConfig.saving}
                onSave={handleConfigSave}
              />
            ),
          },
          {
            key: "pools",
            label: (
              <span>
                <TeamOutlined />
                值班池管理
              </span>
            ),
            children: <DutyPoolPage />,
          },
        ]}
      />

      {/* 节假日模态框 */}
      <HolidayModal
        visible={holidayModalVisible}
        onOk={handleHolidayModalOk}
        onCancel={() => setHolidayModalVisible(false)}
      />

      {/* 批量添加节假日模态框 */}
      <BatchHolidayModal
        visible={holidayBatchModalVisible}
        onOk={handleBatchHolidayOk}
        onCancel={() => setHolidayBatchModalVisible(false)}
      />
    </div>
  );
}
