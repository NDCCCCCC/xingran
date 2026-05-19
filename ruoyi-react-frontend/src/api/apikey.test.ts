/**
 * API 密钥管理 API 客户端测试
 *
 * 测试覆盖所有 API 函数：
 * - listAPIKeys: 分页查询、参数传递、响应解析、错误处理
 * - createAPIKey: 正常创建、密钥返回、错误处理
 * - getAPIKey: 正常获取、密钥脱敏、错误处理
 * - updateAPIKey: 正常更新、参数传递、错误处理
 * - deleteAPIKey: 正常删除、错误处理
 * - toggleAPIKeyStatus: 正常切换、错误处理
 * - listUsageLogs: 正常查询、分页参数、响应解析
 * - getUsageSummary: 正常获取、统计数据、错误处理
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import {
  listAPIKeys,
  createAPIKey,
  getAPIKey,
  updateAPIKey,
  deleteAPIKey,
  toggleAPIKeyStatus,
  listUsageLogs,
  getUsageSummary,
} from './apikey';
import { post, get, put, del } from '@/lib/api';
import type {
  APIKey,
  CreateAPIKeyRequest,
  UpdateAPIKeyRequest,
  APIKeyListParams,
  APIKeyUsageLog,
  UsageSummary,
  PageData,
} from '@/types/apikey';

// Mock API 函数
vi.mock('@/lib/api', () => ({
  post: vi.fn(),
  get: vi.fn(),
  put: vi.fn(),
  del: vi.fn(),
}));

// Mock 数据
const mockAPIKey: APIKey = {
  id: '550e8400-e29b-41d4-a716-446655440000',
  name: 'Test API Key',
  key: 'rec_1a2b3c4d5e6f...',
  description: 'Test description',
  scopes: ['read', 'write'],
  inherit_perms: true,
  ip_whitelist: ['192.168.1.100', '10.0.0.0/24'],
  is_active: true,
  expires_at: '2025-12-31T23:59:59Z',
  last_used_at: '2025-01-15T10:30:00Z',
  created_at: '2025-01-01T00:00:00Z',
  updated_at: '2025-01-15T10:30:00Z',
};

const mockPageData: PageData<APIKey> = {
  list: [mockAPIKey],
  total: 1,
  current: 1,
  pageSize: 10,
};

const mockUsageLog: APIKeyUsageLog = {
  id: '660e8400-e29b-41d4-a716-446655440001',
  api_key_id: '550e8400-e29b-41d4-a716-446655440000',
  method: 'GET',
  path: '/system/users/list',
  status_code: 200,
  duration: 45,
  ip_address: '192.168.1.100',
  user_agent: 'TestAgent/1.0',
  created_at: '2025-01-15T10:30:00Z',
};

const mockUsageLogsPageData: PageData<APIKeyUsageLog> = {
  list: [mockUsageLog],
  total: 1,
  current: 1,
  pageSize: 20,
};

const mockUsageSummary: UsageSummary = {
  total_requests: 1000,
  success_rate: 0.95,
  avg_duration: 45,
  requests_by_method: {
    GET: 700,
    POST: 200,
    PUT: 80,
    DELETE: 20,
  },
  requests_by_path: {
    '/system/users/list': 300,
    '/system/apikeys': 200,
  },
  errors_by_status: {
    '400': 10,
    '401': 5,
    '403': 3,
    '500': 2,
  },
};

describe('apikey API Client', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  // ==================== listAPIKeys 测试 ====================

  describe('listAPIKeys', () => {
    it('应该成功获取 API 密钥列表', async () => {
      const mockResponse = { code: 0, message: 'success', data: mockPageData };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const params: APIKeyListParams = {
        current: 1,
        pageSize: 10,
        keyword: 'test',
        status: true,
        scope: 'read',
      };

      const result = await listAPIKeys(params);

      expect(post).toHaveBeenCalledWith('/system/apikeys/list', params);
      expect(result.data).toEqual(mockPageData);
      expect(result.data?.list).toHaveLength(1);
      expect(result.data?.list[0].name).toBe('Test API Key');
    });

    it('应该支持不带参数的查询', async () => {
      const mockResponse = { code: 0, message: 'success', data: mockPageData };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const result = await listAPIKeys();

      expect(post).toHaveBeenCalledWith('/system/apikeys/list', undefined);
      expect(result.data).toEqual(mockPageData);
    });

    it('应该正确传递分页参数', async () => {
      const mockResponse = { code: 0, message: 'success', data: mockPageData };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const params: APIKeyListParams = {
        current: 2,
        pageSize: 20,
      };

      await listAPIKeys(params);

      expect(post).toHaveBeenCalledWith('/system/apikeys/list', params);
    });

    it('应该正确处理关键词搜索参数', async () => {
      const mockResponse = { code: 0, message: 'success', data: mockPageData };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const params: APIKeyListParams = {
        current: 1,
        pageSize: 10,
        keyword: 'production',
      };

      await listAPIKeys(params);

      expect(post).toHaveBeenCalledWith('/system/apikeys/list', params);
    });

    it('应该正确处理状态筛选参数', async () => {
      const mockResponse = { code: 0, message: 'success', data: mockPageData };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const params: APIKeyListParams = {
        current: 1,
        pageSize: 10,
        status: false,
      };

      await listAPIKeys(params);

      expect(post).toHaveBeenCalledWith('/system/apikeys/list', params);
    });

    it('应该正确处理作用域筛选参数', async () => {
      const mockResponse = { code: 0, message: 'success', data: mockPageData };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const params: APIKeyListParams = {
        current: 1,
        pageSize: 10,
        scope: 'admin',
      };

      await listAPIKeys(params);

      expect(post).toHaveBeenCalledWith('/system/apikeys/list', params);
    });

    it('应该处理网络错误', async () => {
      vi.mocked(post).mockRejectedValue(new Error('Network error'));

      await expect(listAPIKeys()).rejects.toThrow('Network error');
    });

    it('应该处理 API 错误响应', async () => {
      const mockResponse = { code: 1001, message: '参数错误', data: null };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const result = await listAPIKeys();

      expect(result.code).toBe(1001);
      expect(result.message).toBe('参数错误');
    });
  });

  // ==================== createAPIKey 测试 ====================

  describe('createAPIKey', () => {
    it('应该成功创建 API 密钥并返回完整密钥', async () => {
      const fullKey = 'rec_1a2b3c4d5e6f7890abcdef1234567890';
      const mockResponse = {
        code: 0,
        message: 'success',
        data: { key: fullKey },
      };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const createData: CreateAPIKeyRequest = {
        name: 'Production API Key',
        description: '用于生产环境的 API 访问',
        scopes: ['read', 'write'],
        inherit_perms: false,
        ip_whitelist: ['192.168.1.100', '10.0.0.0/24'],
        expires_at: '2025-12-31T23:59:59Z',
      };

      const result = await createAPIKey(createData);

      expect(post).toHaveBeenCalledWith('/system/apikeys', createData);
      expect(result.data?.key).toBe(fullKey);
    });

    it('应该支持创建不带 IP 白名单的密钥', async () => {
      const fullKey = 'rec_1a2b3c4d5e6f7890abcdef1234567890';
      const mockResponse = {
        code: 0,
        message: 'success',
        data: { key: fullKey },
      };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const createData: CreateAPIKeyRequest = {
        name: 'Simple Key',
        scopes: ['read'],
        inherit_perms: true,
      };

      await createAPIKey(createData);

      expect(post).toHaveBeenCalledWith('/system/apikeys', createData);
    });

    it('应该支持创建不带过期时间的密钥', async () => {
      const fullKey = 'rec_1a2b3c4d5e6f7890abcdef1234567890';
      const mockResponse = {
        code: 0,
        message: 'success',
        data: { key: fullKey },
      };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const createData: CreateAPIKeyRequest = {
        name: 'Never Expires Key',
        scopes: ['admin'],
        inherit_perms: false,
      };

      await createAPIKey(createData);

      expect(post).toHaveBeenCalledWith('/system/apikeys', createData);
    });

    it('应该处理创建失败的情况', async () => {
      const mockResponse = {
        code: 400,
        message: '名称已存在',
        data: null,
      };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const createData: CreateAPIKeyRequest = {
        name: 'Duplicate Key',
        scopes: ['read'],
        inherit_perms: true,
      };

      const result = await createAPIKey(createData);

      expect(result.code).toBe(400);
      expect(result.message).toBe('名称已存在');
    });

    it('应该处理网络错误', async () => {
      vi.mocked(post).mockRejectedValue(new Error('Network error'));

      const createData: CreateAPIKeyRequest = {
        name: 'Test Key',
        scopes: ['read'],
        inherit_perms: true,
      };

      await expect(createAPIKey(createData)).rejects.toThrow('Network error');
    });
  });

  // ==================== getAPIKey 测试 ====================

  describe('getAPIKey', () => {
    it('应该成功获取 API 密钥详情（脱敏）', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: mockAPIKey,
      };
      vi.mocked(get).mockResolvedValue(mockResponse);

      const result = await getAPIKey(mockAPIKey.id);

      expect(get).toHaveBeenCalledWith(`/system/apikeys/${mockAPIKey.id}`);
      expect(result.data).toEqual(mockAPIKey);
      expect(result.data?.key).toBe('rec_1a2b3c4d5e6f...');
    });

    it('应该处理密钥不存在的情况', async () => {
      const mockResponse = {
        code: 404,
        message: '密钥不存在',
        data: null,
      };
      vi.mocked(get).mockResolvedValue(mockResponse);

      const result = await getAPIKey('non-existent-id');

      expect(result.code).toBe(404);
      expect(result.message).toBe('密钥不存在');
    });

    it('应该处理网络错误', async () => {
      vi.mocked(get).mockRejectedValue(new Error('Network error'));

      await expect(getAPIKey('test-id')).rejects.toThrow('Network error');
    });
  });

  // ==================== updateAPIKey 测试 ====================

  describe('updateAPIKey', () => {
    it('应该成功更新 API 密钥', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };
      vi.mocked(put).mockResolvedValue(mockResponse);

      const updateData: UpdateAPIKeyRequest = {
        name: 'Updated Name',
        description: 'Updated description',
        scopes: ['read', 'write', 'admin'],
        ip_whitelist: ['10.0.0.0/8'],
      };

      await updateAPIKey(mockAPIKey.id, updateData);

      expect(put).toHaveBeenCalledWith(`/system/apikeys/${mockAPIKey.id}`, updateData);
    });

    it('应该支持部分更新', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };
      vi.mocked(put).mockResolvedValue(mockResponse);

      const updateData: UpdateAPIKeyRequest = {
        name: 'Just update name',
      };

      await updateAPIKey(mockAPIKey.id, updateData);

      expect(put).toHaveBeenCalledWith(`/system/apikeys/${mockAPIKey.id}`, updateData);
    });

    it('应该支持更新作用域', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };
      vi.mocked(put).mockResolvedValue(mockResponse);

      const updateData: UpdateAPIKeyRequest = {
        scopes: ['admin'],
      };

      await updateAPIKey(mockAPIKey.id, updateData);

      expect(put).toHaveBeenCalledWith(`/system/apikeys/${mockAPIKey.id}`, updateData);
    });

    it('应该处理更新失败的情况', async () => {
      const mockResponse = {
        code: 400,
        message: '参数错误',
        data: null,
      };
      vi.mocked(put).mockResolvedValue(mockResponse);

      const updateData: UpdateAPIKeyRequest = {
        name: '',
      };

      const result = await updateAPIKey(mockAPIKey.id, updateData);

      expect(result.code).toBe(400);
      expect(result.message).toBe('参数错误');
    });

    it('应该处理密钥不存在的情况', async () => {
      const mockResponse = {
        code: 404,
        message: '密钥不存在',
        data: null,
      };
      vi.mocked(put).mockResolvedValue(mockResponse);

      const updateData: UpdateAPIKeyRequest = {
        name: 'Test',
      };

      const result = await updateAPIKey('non-existent-id', updateData);

      expect(result.code).toBe(404);
      expect(result.message).toBe('密钥不存在');
    });
  });

  // ==================== deleteAPIKey 测试 ====================

  describe('deleteAPIKey', () => {
    it('应该成功删除 API 密钥', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };
      vi.mocked(del).mockResolvedValue(mockResponse);

      await deleteAPIKey(mockAPIKey.id);

      expect(del).toHaveBeenCalledWith(`/system/apikeys/${mockAPIKey.id}`);
    });

    it('应该处理删除不存在的密钥', async () => {
      const mockResponse = {
        code: 404,
        message: '密钥不存在',
        data: null,
      };
      vi.mocked(del).mockResolvedValue(mockResponse);

      const result = await deleteAPIKey('non-existent-id');

      expect(result.code).toBe(404);
      expect(result.message).toBe('密钥不存在');
    });

    it('应该处理网络错误', async () => {
      vi.mocked(del).mockRejectedValue(new Error('Network error'));

      await expect(deleteAPIKey('test-id')).rejects.toThrow('Network error');
    });
  });

  // ==================== toggleAPIKeyStatus 测试 ====================

  describe('toggleAPIKeyStatus', () => {
    it('应该成功切换 API 密钥状态', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: null,
      };
      vi.mocked(post).mockResolvedValue(mockResponse);

      await toggleAPIKeyStatus(mockAPIKey.id);

      expect(post).toHaveBeenCalledWith(`/system/apikeys/${mockAPIKey.id}/toggle`);
    });

    it('应该处理切换失败的情况', async () => {
      const mockResponse = {
        code: 400,
        message: '操作失败',
        data: null,
      };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const result = await toggleAPIKeyStatus(mockAPIKey.id);

      expect(result.code).toBe(400);
      expect(result.message).toBe('操作失败');
    });

    it('应该处理密钥不存在的情况', async () => {
      const mockResponse = {
        code: 404,
        message: '密钥不存在',
        data: null,
      };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const result = await toggleAPIKeyStatus('non-existent-id');

      expect(result.code).toBe(404);
      expect(result.message).toBe('密钥不存在');
    });
  });

  // ==================== listUsageLogs 测试 ====================

  describe('listUsageLogs', () => {
    it('应该成功获取使用日志列表', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: mockUsageLogsPageData,
      };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const params = {
        current: 1,
        pageSize: 20,
      };

      const result = await listUsageLogs(mockAPIKey.id, params);

      expect(post).toHaveBeenCalledWith(`/system/apikeys/${mockAPIKey.id}/logs`, params);
      expect(result.data).toEqual(mockUsageLogsPageData);
      expect(result.data?.list).toHaveLength(1);
      expect(result.data?.list[0].method).toBe('GET');
    });

    it('应该支持不带分页参数的查询', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: mockUsageLogsPageData,
      };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const result = await listUsageLogs(mockAPIKey.id);

      expect(post).toHaveBeenCalledWith(`/system/apikeys/${mockAPIKey.id}/logs`, undefined);
      expect(result.data).toEqual(mockUsageLogsPageData);
    });

    it('应该正确传递分页参数', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: mockUsageLogsPageData,
      };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const params = {
        current: 2,
        pageSize: 50,
      };

      await listUsageLogs(mockAPIKey.id, params);

      expect(post).toHaveBeenCalledWith(`/system/apikeys/${mockAPIKey.id}/logs`, params);
    });

    it('应该处理日志查询失败的情况', async () => {
      const mockResponse = {
        code: 400,
        message: '参数错误',
        data: null,
      };
      vi.mocked(post).mockResolvedValue(mockResponse);

      const result = await listUsageLogs(mockAPIKey.id);

      expect(result.code).toBe(400);
      expect(result.message).toBe('参数错误');
    });

    it('应该处理网络错误', async () => {
      vi.mocked(post).mockRejectedValue(new Error('Network error'));

      await expect(listUsageLogs(mockAPIKey.id)).rejects.toThrow('Network error');
    });
  });

  // ==================== getUsageSummary 测试 ====================

  describe('getUsageSummary', () => {
    it('应该成功获取使用统计汇总', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: mockUsageSummary,
      };
      vi.mocked(get).mockResolvedValue(mockResponse);

      const result = await getUsageSummary(mockAPIKey.id);

      expect(get).toHaveBeenCalledWith(`/system/apikeys/${mockAPIKey.id}/summary`);
      expect(result.data).toEqual(mockUsageSummary);
      expect(result.data?.total_requests).toBe(1000);
      expect(result.data?.success_rate).toBe(0.95);
      expect(result.data?.avg_duration).toBe(45);
    });

    it('应该包含按方法分组的统计数据', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: mockUsageSummary,
      };
      vi.mocked(get).mockResolvedValue(mockResponse);

      const result = await getUsageSummary(mockAPIKey.id);

      expect(result.data?.requests_by_method).toEqual({
        GET: 700,
        POST: 200,
        PUT: 80,
        DELETE: 20,
      });
    });

    it('应该包含按路径分组的统计数据', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: mockUsageSummary,
      };
      vi.mocked(get).mockResolvedValue(mockResponse);

      const result = await getUsageSummary(mockAPIKey.id);

      expect(result.data?.requests_by_path).toEqual({
        '/system/users/list': 300,
        '/system/apikeys': 200,
      });
    });

    it('应该包含按状态码分组的错误统计', async () => {
      const mockResponse = {
        code: 0,
        message: 'success',
        data: mockUsageSummary,
      };
      vi.mocked(get).mockResolvedValue(mockResponse);

      const result = await getUsageSummary(mockAPIKey.id);

      expect(result.data?.errors_by_status).toEqual({
        '400': 10,
        '401': 5,
        '403': 3,
        '500': 2,
      });
    });

    it('应该处理统计查询失败的情况', async () => {
      const mockResponse = {
        code: 404,
        message: '密钥不存在',
        data: null,
      };
      vi.mocked(get).mockResolvedValue(mockResponse);

      const result = await getUsageSummary('non-existent-id');

      expect(result.code).toBe(404);
      expect(result.message).toBe('密钥不存在');
    });

    it('应该处理网络错误', async () => {
      vi.mocked(get).mockRejectedValue(new Error('Network error'));

      await expect(getUsageSummary('test-id')).rejects.toThrow('Network error');
    });
  });
});
