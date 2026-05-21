/**
 * 认证状态管理（重构版 - 安全 Token 存储）
 * 使用国密 SM4 加密保护 RefreshToken
 *
 * 变更说明：
 * - AccessToken: 内存存储（不持久化）
 * - RefreshToken: sessionStorage 存储（SM4 加密）
 * - 移除 localStorage 直接调用
 * - 集成 TokenManager 实现自动刷新
 */

import { create } from 'zustand';
import { persist } from 'zustand/middleware';
import { post } from '@/lib/api';
import type { User, LoginRequest } from '@/types';
import { useMenuStore } from './menuStore';
import { getEncryptedLoginRequest } from '@/utils/sm2';
import { TokenManager } from '@/utils/token/TokenManager';
import { SecureTokenStorageImpl } from '@/utils/token/SecureTokenStorageImpl';

/**
 * Token 管理器单例
 * 使用 SM4 加密存储 RefreshToken
 */
const tokenStorage = new SecureTokenStorageImpl();
const tokenManager = new TokenManager(tokenStorage, {
	refreshEndpoint: '/system/auth/refresh',
	refreshBeforeSeconds: 30, // 提前 30 秒刷新
	refreshTimeout: 10000,    // 10 秒超时
});

interface AuthState {
	user: User | null;
	isAuthenticated: boolean;
	loading: boolean;
	menusLoaded: boolean; // 登录时是否已加载菜单
	initialized: boolean; // 是否已尝试从存储恢复
}

interface AuthActions {
	login: (credentials: LoginRequest) => Promise<void>;
	logout: () => Promise<void>;
	updateUser: (userData: Partial<User>) => void;
	loadMenusAfterLogin: () => Promise<void>;
	getTokenManager: () => TokenManager;
	initializeFromStorage: () => Promise<void>;
}

type AuthStore = AuthState & AuthActions;

export const useAuthStore = create<AuthStore>()(
	persist(
		(set, get) => ({
			user: null,
			isAuthenticated: false,
			loading: false,
			menusLoaded: false,
			initialized: false, // 是否已尝试从存储恢复

			login: async (credentials: LoginRequest) => {
				set({ loading: true });
				try {
					const encryptedRequest = await getEncryptedLoginRequest(
						credentials.username,
						credentials.password
					);

					const loginRequest = {
						username: encryptedRequest.username,
						password: encryptedRequest.password,
						authMode: credentials.authMode,
						encryptedPassword: encryptedRequest.encryptedPassword,
						captcha: credentials.captcha,
						captchaId: credentials.captchaId,
					};

					const response = await post('/system/auth/login', loginRequest) as {
						data: { user: User; accessToken: string; refreshToken: string; expiresIn: number }
					};
					const { user, accessToken, refreshToken, expiresIn } = response.data;

					// 使用 TokenManager 初始化 Token（不再存储到 localStorage）
					await tokenManager.initializeTokens(accessToken, refreshToken, expiresIn);

					set({
						user,
						isAuthenticated: true,
						loading: false,
						menusLoaded: false, // 重置菜单加载状态
						initialized: true, // 标记为已初始化
					});

					// 登录成功后自动加载菜单和权限
					await get().loadMenusAfterLogin();
				} catch (error) {
					set({ loading: false, menusLoaded: false });
					throw error;
				}
			},

			logout: async () => {
				// 清除 Token（包括 sessionStorage 中的加密 RefreshToken）
				await tokenManager.clearTokens();

				// 清除菜单
				useMenuStore.getState().clearMenus();

				// 清除保存的最后访问路径
				const { clearLastPath } = await import('@/router/DynamicRoutes');
				clearLastPath();

				set({
					user: null,
					isAuthenticated: false,
					menusLoaded: false,
					initialized: false, // 重置初始化标志
				});
			},

			updateUser: (userData: Partial<User>) => {
				const { user } = get();
				if (user) {
					set({
						user: { ...user, ...userData },
					});
				}
			},

			loadMenusAfterLogin: async () => {
				try {
					await useMenuStore.getState().fetchAll(true);
					set({ menusLoaded: true });
				} catch (error) {
					console.error('Failed to load menus after login:', error);
					set({ menusLoaded: false });
					throw error;
				}
			},

			getTokenManager: () => tokenManager,

			// 从存储恢复状态（页面刷新时调用）
			initializeFromStorage: async () => {
				// 防止重复初始化
				const { initialized } = get();
				if (initialized) {
					return;
				}

				// 检查 sessionStorage 中是否有有效的 RefreshToken
				const hasRefreshToken = await tokenManager.getRefreshToken();

				if (hasRefreshToken) {
					// 有 RefreshToken，尝试刷新获取新的 AccessToken
					try {
						await tokenManager.refreshToken();
						// 刷新成功，isAuthenticated 会在后续 API 请求中保持
						// 用户信息已经从 localStorage 恢复
						set({ initialized: true, isAuthenticated: true });
						return;
					} catch (error) {
						console.error('[AuthStore] 刷新 Token 失败，清除认证状态:', error);
					}
				}

				// 没有 RefreshToken 或刷新失败，清除认证状态
				set({
					initialized: true,
					isAuthenticated: false,
					menusLoaded: false,
				});
			},
		}),
		{
			name: 'auth-storage',
			// 只持久化用户基本信息（非敏感），不持久化 Token
			partialize: (state) => ({
				user: state.user, // 用户基本信息（用户名、昵称等）
				// isAuthenticated 不持久化，需要根据 Token 存在与否动态判断
				// initialized 不持久化，每次页面加载都需要重新初始化
			}),
			// 在 localStorage 数据恢复完成后触发初始化
			onRehydrateStorage: () => (state) => {
				if (!state) return;

				// 异步初始化（不阻塞恢复过程）
				setTimeout(async () => {
					// 防止重复初始化
					if (state.initialized) {
						return;
					}

					// 检查 sessionStorage 中是否有有效的 RefreshToken
					const hasRefreshToken = await tokenManager.getRefreshToken();

					if (hasRefreshToken) {
						// 有 RefreshToken，尝试刷新获取新的 AccessToken
						try {
							await tokenManager.refreshToken();
							// 刷新成功，设置认证状态
							state.initializeFromStorage();
							return;
						} catch (error) {
							console.error('[AuthStore] 刷新 Token 失败，清除认证状态:', error);
						}
					}

					// 没有 RefreshToken 或刷新失败，清除认证状态
					const { user } = state;

					// 清除用户信息（如果有）
					if (user) {
						state.logout();
					} else {
						// 直接设置未认证状态
						useAuthStore.setState({
							initialized: true,
							isAuthenticated: false,
							menusLoaded: false,
						});
					}
				}, 0);
			},
		}
	)
);

/**
 * 导出 TokenManager 实例供拦截器使用
 */
export const getTokenManager = () => tokenManager;
