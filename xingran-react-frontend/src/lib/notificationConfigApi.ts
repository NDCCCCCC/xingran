import { post, get, put, del } from "./api";

// ==================== 邮箱配置 ====================

export interface EmailConfig {
  id: string;
  configName: string;
  host: string;
  port: number;
  username: string;
  fromName: string;
  fromEmail: string;
  useSsl: boolean;
  useStartTls: boolean;
  isDefault: boolean;
  status: number;
  remark: string;
  createdAt: string;
  updatedAt: string;
}

export interface EmailConfigListParams {
  page?: number;
  pageSize?: number;
  status?: number;
}

export interface EmailConfigCreateRequest {
  configName: string;
  host: string;
  port: number;
  username: string;
  password: string;
  fromName?: string;
  fromEmail?: string;
  useSsl?: boolean;
  useStartTls?: boolean;
  isDefault?: boolean;
  status?: number;
  remark?: string;
}

export interface EmailConfigUpdateRequest {
  configName?: string;
  host?: string;
  port?: number;
  username?: string;
  password?: string;
  fromName?: string;
  fromEmail?: string;
  useSsl?: boolean;
  isDefault?: boolean;
  status?: number;
  remark?: string;
}

// 获取邮箱配置列表
export const getEmailConfigList = (params: EmailConfigListParams) => {
  return post("/system/settings/notification/email-configs/list", params);
};

// 获取邮箱配置详情
export const getEmailConfig = (id: string) => {
  return get(`/system/settings/notification/email-configs/${id}`);
};

// 创建邮箱配置
export const createEmailConfig = (data: EmailConfigCreateRequest) => {
  return post("/system/settings/notification/email-configs", data);
};

// 更新邮箱配置
export const updateEmailConfig = (id: string, data: EmailConfigUpdateRequest) => {
  return put(`/system/settings/notification/email-configs/${id}`, data);
};

// 删除邮箱配置
export const deleteEmailConfig = (id: string) => {
  return del(`/system/settings/notification/email-configs/${id}`);
};

// 测试邮箱配置
export const testEmailConfig = (id: string, testTo: string) => {
  return post(`/system/settings/notification/email-configs/${id}/test`, { testTo });
};

// ==================== API通知配置 ====================

// 导出类型常量
export const APIConfigTypes = {
  SMS: "sms",
  WEBHOOK: "webhook",
  PUSH: "push",
} as const;

export type APIConfigType = typeof APIConfigTypes[keyof typeof APIConfigTypes];

export const AuthTypes = {
  NONE: "none",
  BASIC: "basic",
  BEARER: "bearer",
  APIKEY: "apikey",
} as const;

export type AuthType = typeof AuthTypes[keyof typeof AuthTypes];

export interface APINotificationConfig {
  id: string;
  configName: string;
  configType: APIConfigType;
  apiUrl: string;
  apiMethod: string;
  headers: Record<string, string>;
  templateBody: string;
  authType: AuthType;
  authConfig: Record<string, string | number | boolean>;
  retryCount: number;
  timeout: number;
  isDefault: boolean;
  status: number;
  remark: string;
  createdAt: string;
  updatedAt: string;
}

export interface APINotificationConfigListParams {
  page?: number;
  pageSize?: number;
  configType?: APIConfigType;
  status?: number;
}

export interface APINotificationConfigCreateRequest {
  configName: string;
  configType: APIConfigType;
  apiUrl: string;
  apiMethod?: string;
  headers?: Record<string, string>;
  templateBody?: string;
  authType?: AuthType;
  authConfig?: Record<string, string | number | boolean>;
  retryCount?: number;
  timeout?: number;
  isDefault?: boolean;
  status?: number;
  remark?: string;
}

export interface APINotificationConfigUpdateRequest {
  configName?: string;
  apiUrl?: string;
  apiMethod?: string;
  headers?: Record<string, string>;
  templateBody?: string;
  authType?: AuthType;
  authConfig?: Record<string, string | number | boolean>;
  retryCount?: number;
  timeout?: number;
  isDefault?: boolean;
  status?: number;
  remark?: string;
}

// 获取API通知配置列表
export const getAPINotificationConfigList = (params: APINotificationConfigListParams) => {
  return post("/system/settings/notification/api-notification-configs/list", params);
};

// 获取API通知配置详情
export const getAPINotificationConfig = (id: string) => {
  return get(`/system/settings/notification/api-notification-configs/${id}`);
};

// 创建API通知配置
export const createAPINotificationConfig = (data: APINotificationConfigCreateRequest) => {
  return post("/system/settings/notification/api-notification-configs", data);
};

// 更新API通知配置
export const updateAPINotificationConfig = (id: string, data: APINotificationConfigUpdateRequest) => {
  return put(`/system/settings/notification/api-notification-configs/${id}`, data);
};

// 删除API通知配置
export const deleteAPINotificationConfig = (id: string) => {
  return del(`/system/settings/notification/api-notification-configs/${id}`);
};

// 测试API通知配置
export const testAPINotificationConfig = (id: string) => {
  return post(`/system/settings/notification/api-notification-configs/${id}/test`);
};

// ==================== Cron表达式 ====================

export interface CronExpression {
  name: string;
  expression: string;
  description: string;
}

// 获取常用Cron表达式
export const getCommonCronExpressions = () => {
  return get("/system/notices/cron-expressions");
};
