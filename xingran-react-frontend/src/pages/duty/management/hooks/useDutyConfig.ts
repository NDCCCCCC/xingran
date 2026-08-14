import { useState, useCallback } from "react";
import { App } from "antd";
import { getDutyConfig, updateDutyConfig, type DutyConfig } from "@/lib/dutyApi";
import type { Dayjs } from "dayjs";
import dayjs from "dayjs";

export function useDutyConfig() {
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [config, setConfig] = useState<DutyConfig | null>(null);

  // 获取配置
  const fetch = useCallback(async () => {
    setLoading(true);
    try {
      const result = await getDutyConfig();
      setConfig(result.data || null);
      return result.data || null;
    } catch (error) {
      message.error("获取配置失败");
      return null;
    } finally {
      setLoading(false);
    }
  }, []);

  // 保存配置
  const save = useCallback(
    async (values: {
      reminderEnabled: boolean;
      reminderTime: Dayjs;
      reminderChannels: string[];
      beforeReminderMinutes?: number;
    }) => {
      setSaving(true);
      try {
        const data = {
          reminderEnabled: values.reminderEnabled,
          reminderTime: values.reminderTime.format("HH:mm"),
          reminderChannels: Array.isArray(values.reminderChannels)
            ? values.reminderChannels.join(",")
            : values.reminderChannels,
          beforeReminderMinutes: values.beforeReminderMinutes,
        };
        await updateDutyConfig(data);
        message.success("配置保存成功");
        await fetch();
        return true;
      } catch (error) {
        message.error("配置保存失败");
        return false;
      } finally {
        setSaving(false);
      }
    },
    [fetch]
  );

  // 获取表单初始值
  const getFormValues = useCallback(() => {
    if (!config) return {};
    return {
      reminderEnabled: config.reminderEnabled,
      reminderTime: dayjs(config.reminderTime, "HH:mm"),
      reminderChannels: config.reminderChannels ? config.reminderChannels.split(",") : [],
      beforeReminderMinutes: config.beforeReminderMinutes,
    };
  }, [config]);

  return {
    loading,
    saving,
    config,
    fetch,
    save,
    getFormValues,
  };
}
