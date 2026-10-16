/**
 * Job Actions Hook
 * 定时任务操作管理 Hook
 */

import { useState, useCallback } from "react";
import { App } from "antd";
import type { FormInstance } from "antd/es/form";
import type { JobInfo, SearchFormState } from "../types";
import { post, postLongRequest } from "@/lib/api";

export interface UseJobActionsParams {
  fetchJobs: () => Promise<void>;
  fetchJobLogs: (jobName?: string, jobGroup?: string) => Promise<void>;
}

export interface UseJobActionsReturn {
  // 模态框状态
  modalVisible: boolean;
  modalTitle: string;
  isEdit: boolean;
  logDrawerVisible: boolean;
  selectedJob: JobInfo | null;

  // 状态设置
  setModalVisible: React.Dispatch<React.SetStateAction<boolean>>;
  setModalTitle: React.Dispatch<React.SetStateAction<string>>;
  setIsEdit: React.Dispatch<React.SetStateAction<boolean>>;
  setLogDrawerVisible: React.Dispatch<React.SetStateAction<boolean>>;
  setSelectedJob: React.Dispatch<React.SetStateAction<JobInfo | null>>;

  // 操作方法
  openModal: (record: JobInfo | undefined, form: FormInstance<unknown>) => void;
  handleSubmit: (form: FormInstance<unknown>) => Promise<void>;
  handleDelete: (record: JobInfo) => Promise<void>;
  handleToggleStatus: (record: JobInfo) => Promise<void>;
  handleExecute: (record: JobInfo) => Promise<void>;
  handleViewLogs: (record: JobInfo) => void;
  handleReset: (
    setSearchForm: React.Dispatch<React.SetStateAction<SearchFormState>>,
    setCurrent: (page: number) => void,
    fetchJobs: () => Promise<void>
  ) => void;
}

export function useJobActions(params: UseJobActionsParams): UseJobActionsReturn {
  const { fetchJobs, fetchJobLogs } = params;
  const { message } = App.useApp();

  const [modalVisible, setModalVisible] = useState(false);
  const [modalTitle, setModalTitle] = useState("");
  const [isEdit, setIsEdit] = useState(false);
  const [logDrawerVisible, setLogDrawerVisible] = useState(false);
  const [selectedJob, setSelectedJob] = useState<JobInfo | null>(null);

  // 打开新增/编辑模态框
  const openModal = useCallback((record: JobInfo | undefined, form: FormInstance<unknown>) => {
    if (record) {
      setIsEdit(true);
      setModalTitle("编辑定时任务");
      form.setFieldsValue({
        id: record.id,
        jobName: record.jobName,
        jobGroup: record.jobGroup,
        invokeTarget: record.invokeTarget,
        cronExpression: record.cronExpression,
        misfirePolicy: record.misfirePolicy,
        concurrent: record.concurrent,
        status: record.status,
        remark: record.remark || "",
      });
    } else {
      setIsEdit(false);
      setModalTitle("新增定时任务");
      form.resetFields();
      form.setFieldsValue({
        misfirePolicy: 1,
        concurrent: false,
        status: 0,
      });
    }
    setModalVisible(true);
  }, []);

  // 提交表单
  const handleSubmit = useCallback(
    async (form: FormInstance<unknown>) => {
      try {
        const values = (await form.validateFields()) as { id?: string } & Record<string, unknown>;

        if (isEdit) {
          await post(`/monitor/jobs/${values.id}/update`, values);
        } else {
          await post("/monitor/jobs", values);
        }

        message.success(isEdit ? "更新成功" : "新增成功");
        setModalVisible(false);
        fetchJobs();
      } catch (error) {
        console.error("提交失败:", error);
      }
    },
    [isEdit, fetchJobs, message]
  );

  // 删除任务
  const handleDelete = useCallback(
    async (record: JobInfo) => {
      try {
        await post(`/monitor/jobs/${record.id}/delete`);
        message.success("删除成功");
        fetchJobs();
      } catch (error) {
        console.error("删除失败:", error);
      }
    },
    [fetchJobs, message]
  );

  // 启动/停止任务
  const handleToggleStatus = useCallback(
    async (record: JobInfo) => {
      try {
        await post(`/monitor/jobs/${record.id}/status`, {
          status: record.status === 0 ? 1 : 0,
        });

        message.success(record.status === 0 ? "暂停成功" : "启动成功");
        fetchJobs();
      } catch (error) {
        console.error("操作失败:", error);
      }
    },
    [fetchJobs, message]
  );

  // 立即执行任务
  const handleExecute = useCallback(
    async (record: JobInfo) => {
      try {
        // 使用长时间请求函数（5分钟超时），因为设备监控任务可能耗时较长
        await postLongRequest(`/monitor/jobs/${record.id}/execute`, {});
        message.success("执行成功");
        if (selectedJob?.id === record.id) {
          fetchJobLogs(record.jobName, record.jobGroup);
        }
      } catch (error) {
        console.error("执行失败:", error);
      }
    },
    [selectedJob, fetchJobLogs, message]
  );

  // 查看日志
  const handleViewLogs = useCallback(
    (record: JobInfo) => {
      setSelectedJob(record);
      setLogDrawerVisible(true);
      fetchJobLogs(record.jobName, record.jobGroup);
    },
    [fetchJobLogs]
  );

  // 重置
  const handleReset = useCallback(
    (
      setSearchForm: React.Dispatch<React.SetStateAction<SearchFormState>>,
      setCurrent: (page: number) => void,
      fetchJobs: () => Promise<void>
    ) => {
      setSearchForm({
        jobName: "",
        jobGroup: "",
        status: undefined,
      });
      setCurrent(1);
      fetchJobs();
    },
    []
  );

  return {
    modalVisible,
    modalTitle,
    isEdit,
    logDrawerVisible,
    selectedJob,
    setModalVisible,
    setModalTitle,
    setIsEdit,
    setLogDrawerVisible,
    setSelectedJob,
    openModal,
    handleSubmit,
    handleDelete,
    handleToggleStatus,
    handleExecute,
    handleViewLogs,
    handleReset,
  };
}
