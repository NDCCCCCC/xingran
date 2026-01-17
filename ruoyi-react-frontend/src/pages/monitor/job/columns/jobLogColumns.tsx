/**
 * 任务日志表格列定义
 */

import { Tag } from 'antd';
import type { ColumnsType } from 'antd/es/table';
import type { JobLog } from '../types';
import { formatDateTime } from '@/utils/datetime';
import { formatDuration, renderJobStatusTag, renderExceptionInfo } from '../utils';

export function getJobLogColumns(): ColumnsType<JobLog> {
  return [
    {
      title: '执行时间',
      dataIndex: 'startTime',
      key: 'startTime',
      width: 180,
      render: (text: string) => formatDateTime(text),
    },
    {
      title: '执行时长',
      dataIndex: 'duration',
      key: 'duration',
      width: 100,
      render: formatDuration,
    },
    {
      title: '执行消息',
      dataIndex: 'jobMessage',
      key: 'jobMessage',
      width: 200,
      ellipsis: true,
    },
    {
      title: '执行状态',
      dataIndex: 'status',
      key: 'status',
      width: 100,
      render: (status: number) => (
        <Tag color={status === 0 ? 'success' : 'error'}>
          {status === 0 ? '成功' : '失败'}
        </Tag>
      ),
    },
    {
      title: '异常信息',
      dataIndex: 'exceptionInfo',
      key: 'exceptionInfo',
      ellipsis: true,
      render: renderExceptionInfo,
    },
  ];
}
