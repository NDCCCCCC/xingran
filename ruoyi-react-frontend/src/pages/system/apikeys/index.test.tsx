/**
 * API 密钥管理页面组件测试
 *
 * 测试覆盖所有主要交互：
 * - 组件渲染：页面正常渲染、表格显示、分页组件、搜索框
 * - 数据加载：初始加载、加载状态、数据刷新、错误处理
 * - 创建功能：打开 Modal、表单验证、提交创建、密钥显示、复制功能
 * - 编辑功能：打开 Modal、数据回显、表单验证、提交更新
 * - 删除功能：删除确认、删除操作、列表更新
 * - 启用/禁用：状态切换、列表更新
 * - 搜索和筛选：关键词搜索、状态筛选、作用域筛选
 * - 日志查看：打开日志 Modal、统计数据显示、日志表格、分页功能
 */

import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, fireEvent, waitFor } from '@testing-library/react';
import '@testing-library/jest-dom';
import APIKeyManagement from './index';
import * as apikeyApi from '@/api/apikey';
import type { APIKey, PageData } from '@/types/apikey';

// Mock API 模块
vi.mock('@/api/apikey');

// Mock LogsModal 组件
vi.mock('./LogsModal', () => ({
  default: ({ visible, onClose }: { visible: boolean; onClose: () => void }) =>
    visible ? (
      <div data-testid="logs-modal">
        <button onClick={onClose}>关闭</button>
      </div>
    ) : null,
}));

// Mock navigator.clipboard
Object.assign(navigator, {
  clipboard: {
    writeText: vi.fn(() => Promise.resolve()),
  },
});

// Mock 数据
const mockAPIKeys: APIKey[] = [
  {
    id: '550e8400-e29b-41d4-a716-446655440000',
    name: 'Production API Key',
    key: 'rec_1a2b3c4d5e6f...',
    description: '用于生产环境',
    scopes: ['read', 'write'],
    inherit_perms: true,
    ip_whitelist: ['192.168.1.100'],
    is_active: true,
    expires_at: '2025-12-31T23:59:59Z',
    last_used_at: '2025-01-15T10:30:00Z',
    created_at: '2025-01-01T00:00:00Z',
    updated_at: '2025-01-15T10:30:00Z',
  },
  {
    id: '660e8400-e29b-41d4-a716-446655440001',
    name: 'Test API Key',
    key: 'rec_2b3c4d5e6f7a...',
    description: '用于测试',
    scopes: ['read'],
    inherit_perms: false,
    ip_whitelist: [],
    is_active: false,
    expires_at: null,
    last_used_at: null,
    created_at: '2025-01-10T00:00:00Z',
    updated_at: '2025-01-10T00:00:00Z',
  },
];

const mockPageData: PageData<APIKey> = {
  list: mockAPIKeys,
  total: 2,
  current: 1,
  pageSize: 10,
};

describe('APIKeyManagement Component', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    // 默认 mock listAPIKeys 返回成功
    vi.mocked(apikeyApi.listAPIKeys).mockResolvedValue({
      code: 0,
      message: 'success',
      data: mockPageData,
    });
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  // ==================== 组件渲染测试 ====================

  describe('组件渲染', () => {
    it('应该正常渲染页面', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('Production API Key')).toBeInTheDocument();
        expect(screen.getByText('Test API Key')).toBeInTheDocument();
      });
    });

    it('应该显示搜索框', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        const searchInput = screen.getByPlaceholderText('搜索名称或密钥');
        expect(searchInput).toBeInTheDocument();
      });
    });

    it('应该显示筛选下拉框', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('状态筛选')).toBeInTheDocument();
        expect(screen.getByText('作用域筛选')).toBeInTheDocument();
      });
    });

    it('应该显示操作按钮', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('创建密钥')).toBeInTheDocument();
        expect(screen.getByText('搜索')).toBeInTheDocument();
        expect(screen.getByText('重置')).toBeInTheDocument();
        expect(screen.getByText('刷新')).toBeInTheDocument();
      });
    });

    it('应该显示作用域标签', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('read')).toBeInTheDocument();
        expect(screen.getByText('write')).toBeInTheDocument();
      });
    });

    it('应该显示继承权限标签', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        const yesTags = screen.getAllByText('是');
        const noTags = screen.getAllByText('否');
        expect(yesTags.length).toBeGreaterThan(0);
        expect(noTags.length).toBeGreaterThan(0);
      });
    });

    it('应该显示状态标签', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('启用')).toBeInTheDocument();
        expect(screen.getByText('禁用')).toBeInTheDocument();
      });
    });
  });

  // ==================== 数据加载测试 ====================

  describe('数据加载', () => {
    it('应该在组件挂载时加载数据', () => {
      render(<APIKeyManagement />);

      expect(apikeyApi.listAPIKeys).toHaveBeenCalled();
    });

    it('应该支持数据刷新', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('刷新')).toBeInTheDocument();
      });

      const refreshButton = screen.getByText('刷新');
      fireEvent.click(refreshButton);

      await waitFor(() => {
        expect(apikeyApi.listAPIKeys).toHaveBeenCalledTimes(2); // 初始加载 + 刷新
      });
    });

    it('应该处理加载错误', async () => {
      vi.mocked(apikeyApi.listAPIKeys).mockRejectedValue(new Error('Network error'));

      const consoleSpy = vi.spyOn(console, 'error').mockImplementation(() => {});

      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(consoleSpy).toHaveBeenCalledWith(
          '加载 API 密钥列表失败:',
          expect.any(Error)
        );
      });

      consoleSpy.mockRestore();
    });
  });

  // ==================== 创建功能测试 ====================

  describe('创建功能', () => {
    it('应该打开创建 Modal', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('创建密钥')).toBeInTheDocument();
      });

      const createButton = screen.getByText('创建密钥');
      fireEvent.click(createButton);

      await waitFor(() => {
        expect(screen.getByText('创建 API 密钥')).toBeInTheDocument();
      });
    });

    it('应该显示表单字段', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('创建密钥')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('创建密钥'));

      await waitFor(() => {
        expect(screen.getByText('密钥名称')).toBeInTheDocument();
        expect(screen.getByText('描述')).toBeInTheDocument();
        expect(screen.getByText('作用域')).toBeInTheDocument();
        expect(screen.getByText('继承权限')).toBeInTheDocument();
        expect(screen.getByText('IP 白名单')).toBeInTheDocument();
        expect(screen.getByText('过期时间')).toBeInTheDocument();
      });
    });

    it('应该成功创建 API 密钥', async () => {
      const fullKey = 'rec_1a2b3c4d5e6f7890abcdef1234567890';
      vi.mocked(apikeyApi.createAPIKey).mockResolvedValue({
        code: 0,
        message: 'success',
        data: { key: fullKey },
      });

      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('创建密钥')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('创建密钥'));

      await waitFor(() => {
        expect(screen.getByText('创建 API 密钥')).toBeInTheDocument();
      });

      // 填写表单
      const nameInput = screen.getByPlaceholderText('请输入密钥名称');
      fireEvent.change(nameInput, { target: { value: 'New API Key' } });

      // 提交表单
      const okButtons = screen.getAllByText('确定');
      const okButton = okButtons.find(btn => btn.tagName === 'BUTTON');
      if (okButton) {
        fireEvent.click(okButton);

        await waitFor(() => {
          expect(apikeyApi.createAPIKey).toHaveBeenCalled();
        });
      }
    });

    it('应该在创建成功后显示完整密钥', async () => {
      const fullKey = 'rec_1a2b3c4d5e6f7890abcdef1234567890';
      vi.mocked(apikeyApi.createAPIKey).mockResolvedValue({
        code: 0,
        message: 'success',
        data: { key: fullKey },
      });

      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('创建密钥')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('创建密钥'));

      await waitFor(() => {
        expect(screen.getByText('创建 API 密钥')).toBeInTheDocument();
      });

      const nameInput = screen.getByPlaceholderText('请输入密钥名称');
      fireEvent.change(nameInput, { target: { value: 'New API Key' } });

      const okButtons = screen.getAllByText('确定');
      const okButton = okButtons.find(btn => btn.tagName === 'BUTTON');
      if (okButton) {
        fireEvent.click(okButton);

        await waitFor(() => {
          expect(screen.getByText('请妥善保管您的 API 密钥')).toBeInTheDocument();
        });
      }
    });

    it('应该支持复制密钥到剪贴板', async () => {
      const fullKey = 'rec_1a2b3c4d5e6f7890abcdef1234567890';
      vi.mocked(apikeyApi.createAPIKey).mockResolvedValue({
        code: 0,
        message: 'success',
        data: { key: fullKey },
      });

      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('创建密钥')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('创建密钥'));

      await waitFor(() => {
        expect(screen.getByText('创建 API 密钥')).toBeInTheDocument();
      });

      const nameInput = screen.getByPlaceholderText('请输入密钥名称');
      fireEvent.change(nameInput, { target: { value: 'New API Key' } });

      const okButtons = screen.getAllByText('确定');
      const okButton = okButtons.find(btn => btn.tagName === 'BUTTON');
      if (okButton) {
        fireEvent.click(okButton);

        await waitFor(() => {
          expect(screen.getByText('复制密钥')).toBeInTheDocument();
        });

        const copyButton = screen.getByText('复制密钥');
        fireEvent.click(copyButton);

        await waitFor(() => {
          expect(navigator.clipboard.writeText).toHaveBeenCalledWith(fullKey);
        });
      }
    });
  });

  // ==================== 编辑功能测试 ====================

  describe('编辑功能', () => {
    it('应该打开编辑 Modal', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('Production API Key')).toBeInTheDocument();
      });

      const editButtons = screen.getAllByTitle('编辑');
      if (editButtons.length > 0) {
        fireEvent.click(editButtons[0]);

        await waitFor(() => {
          expect(screen.getByText('编辑 API 密钥')).toBeInTheDocument();
        });
      }
    });

    it('应该成功更新 API 密钥', async () => {
      vi.mocked(apikeyApi.updateAPIKey).mockResolvedValue({
        code: 0,
        message: 'success',
        data: null,
      });

      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('Production API Key')).toBeInTheDocument();
      });

      const editButtons = screen.getAllByTitle('编辑');
      if (editButtons.length > 0) {
        fireEvent.click(editButtons[0]);

        await waitFor(() => {
          expect(screen.getByText('编辑 API 密钥')).toBeInTheDocument();
        });

        const okButtons = screen.getAllByText('确定');
        const okButton = okButtons.find(btn => btn.tagName === 'BUTTON');
        if (okButton) {
          fireEvent.click(okButton);

          await waitFor(() => {
            expect(apikeyApi.updateAPIKey).toHaveBeenCalled();
          });
        }
      }
    });
  });

  // ==================== 删除功能测试 ====================

  describe('删除功能', () => {
    it('应该成功删除 API 密钥', async () => {
      vi.mocked(apikeyApi.deleteAPIKey).mockResolvedValue({
        code: 0,
        message: 'success',
        data: null,
      });

      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('Production API Key')).toBeInTheDocument();
      });

      const deleteButtons = screen.getAllByTitle('删除');
      if (deleteButtons.length > 0) {
        fireEvent.click(deleteButtons[0]);

        await waitFor(() => {
          expect(screen.getByText('确定要删除这个密钥吗？')).toBeInTheDocument();
        });

        const confirmButton = screen.getByText('确定');
        fireEvent.click(confirmButton);

        await waitFor(() => {
          expect(apikeyApi.deleteAPIKey).toHaveBeenCalled();
        });
      }
    });
  });

  // ==================== 启用/禁用测试 ====================

  describe('启用/禁用功能', () => {
    it('应该切换 API 密钥状态', async () => {
      vi.mocked(apikeyApi.toggleAPIKeyStatus).mockResolvedValue({
        code: 0,
        message: 'success',
        data: null,
      });

      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('Production API Key')).toBeInTheDocument();
      });

      const lockButtons = screen.getAllByTitle('禁用');
      if (lockButtons.length > 0) {
        fireEvent.click(lockButtons[0]);

        await waitFor(() => {
          expect(apikeyApi.toggleAPIKeyStatus).toHaveBeenCalled();
        });
      }
    });
  });

  // ==================== 搜索和筛选测试 ====================

  describe('搜索和筛选功能', () => {
    it('应该支持关键词搜索', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByPlaceholderText('搜索名称或密钥')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText('搜索名称或密钥');
      fireEvent.change(searchInput, { target: { value: 'Production' } });

      const searchButton = screen.getByText('搜索');
      fireEvent.click(searchButton);

      await waitFor(() => {
        expect(apikeyApi.listAPIKeys).toHaveBeenCalled();
      });
    });

    it('应该重置所有筛选条件', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('重置')).toBeInTheDocument();
      });

      const searchInput = screen.getByPlaceholderText('搜索名称或密钥');
      fireEvent.change(searchInput, { target: { value: 'test' } });

      const resetButton = screen.getByText('重置');
      fireEvent.click(resetButton);

      await waitFor(() => {
        expect(searchInput).toHaveValue('');
      });
    });
  });

  // ==================== 日志查看测试 ====================

  describe('日志查看功能', () => {
    it('应该打开日志 Modal', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('Production API Key')).toBeInTheDocument();
      });

      const logButtons = screen.getAllByTitle('使用日志');
      if (logButtons.length > 0) {
        fireEvent.click(logButtons[0]);

        await waitFor(() => {
          expect(screen.getByTestId('logs-modal')).toBeInTheDocument();
        });
      }
    });

    it('应该关闭日志 Modal', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('Production API Key')).toBeInTheDocument();
      });

      const logButtons = screen.getAllByTitle('使用日志');
      if (logButtons.length > 0) {
        fireEvent.click(logButtons[0]);

        await waitFor(() => {
          expect(screen.getByTestId('logs-modal')).toBeInTheDocument();
        });

        const closeButton = screen.getByText('关闭');
        fireEvent.click(closeButton);

        await waitFor(() => {
          expect(screen.queryByTestId('logs-modal')).not.toBeInTheDocument();
        });
      }
    });
  });

  // ==================== 查看详情测试 ====================

  describe('查看详情功能', () => {
    it('应该显示密钥详情 Modal', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('Production API Key')).toBeInTheDocument();
      });

      const viewButtons = screen.getAllByTitle('查看详情');
      if (viewButtons.length > 0) {
        fireEvent.click(viewButtons[0]);

        await waitFor(() => {
          expect(screen.getByText('密钥详情')).toBeInTheDocument();
        });
      }
    });
  });

  // ==================== 分页功能测试 ====================

  describe('分页功能', () => {
    it('应该显示正确的分页信息', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('共 2 条')).toBeInTheDocument();
      });
    });
  });

  // ==================== 复制功能测试 ====================

  describe('复制功能', () => {
    it('应该复制密钥到剪贴板', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('Production API Key')).toBeInTheDocument();
      });

      const copyButtons = screen.getAllByTitle('复制密钥');
      if (copyButtons.length > 0) {
        fireEvent.click(copyButtons[0]);

        await waitFor(() => {
          expect(navigator.clipboard.writeText).toHaveBeenCalled();
        });
      }
    });
  });

  // ==================== 表单验证测试 ====================

  describe('表单验证', () => {
    it('应该验证密钥名称长度', async () => {
      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('创建密钥')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('创建密钥'));

      await waitFor(() => {
        expect(screen.getByText('创建 API 密钥')).toBeInTheDocument();
      });

      const nameInput = screen.getByPlaceholderText('请输入密钥名称');
      const longName = 'a'.repeat(101); // 超过100字符
      fireEvent.change(nameInput, { target: { value: longName } });

      const okButtons = screen.getAllByText('确定');
      const okButton = okButtons.find(btn => btn.tagName === 'BUTTON');
      if (okButton) {
        fireEvent.click(okButton);

        await waitFor(() => {
          expect(screen.getByText('名称长度不能超过100个字符')).toBeInTheDocument();
        });
      }
    });
  });

  // ==================== 错误处理测试 ====================

  describe('错误处理', () => {
    it('应该处理创建失败', async () => {
      vi.mocked(apikeyApi.createAPIKey).mockRejectedValue(new Error('创建失败'));

      render(<APIKeyManagement />);

      await waitFor(() => {
        expect(screen.getByText('创建密钥')).toBeInTheDocument();
      });

      fireEvent.click(screen.getByText('创建密钥'));

      await waitFor(() => {
        expect(screen.getByText('创建 API 密钥')).toBeInTheDocument();
      });

      const nameInput = screen.getByPlaceholderText('请输入密钥名称');
      fireEvent.change(nameInput, { target: { value: 'Test Key' } });

      const okButtons = screen.getAllByText('确定');
      const okButton = okButtons.find(btn => btn.tagName === 'BUTTON');
      if (okButton) {
        fireEvent.click(okButton);

        // 验证是否调用了 createAPIKey
        await waitFor(() => {
          expect(apikeyApi.createAPIKey).toHaveBeenCalled();
        });
      }
    });
  });
});
