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

import { create } from "zustand";
import { persist } from "zustand/middleware";
import { post } from "@/lib/api";
import type { User, LoginRequest } from "@/types";
import { useMenuStore } from "./menuStore";
import { getEncryptedLoginRequest } from "@/utils/sm2";
import { TokenManager } from "@/utils/token/TokenManager";
import { SecureTokenStorageImpl } from "@/utils/token/SecureTokenStorageImpl";
import { STORAGE_KEYS, clearAllTableState } from "@/constants/storage";

/**
 * Token 管理器单例
 * 使用 SM4 加密存储 RefreshToken
 */
const tokenStorage = new SecureTokenStorageImpl();
const tokenManager = new TokenManager(tokenStorage, {
	refreshEndpoint: "/system/auth/refresh",
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
						encryptedPassword: encryptedRequest.encryptedPassword,
						captcha: credentials.captcha,
						captchaId: credentials.captchaId,
					};

					const response = await post("/system/auth/login", loginRequest) as {
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
				try {
					await tokenManager.clearTokens();
				} catch (e) {
					console.error("[AuthStore] clearTokens 失败，继续清理状态:", e);
				}

				// 清除菜单
				try {
					useMenuStore.getState().clearMenus();
				} catch (e) {
					console.error("[AuthStore] clearMenus 失败，继续清理状态:", e);
				}

				// 清除保存的最后访问路径
				// 直接读 sessionStorage，避免动态 import 与 DynamicRoutes 循环依赖
				try {
					sessionStorage.removeItem(STORAGE_KEYS.LAST_PATH);
					// 清理表格状态（筛选/分页/排序）+ 标签页元信息，确保换人登录无上一用户痕迹
					clearAllTableState();
					localStorage.removeItem("tabs-storage");
				} catch {
					// Ignore sessionStorage/localStorage errors
				}

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
					console.error("Failed to load menus after login:", error);
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

				// 关键修复：使用 try/finally 确保 initialized: true 在所有路径上都被设置，
				// 否则 DynamicRoutes 的 <InitializingFallback /> 会永久卡住。
				try {
					// 检查 sessionStorage 中是否有有效的 RefreshToken
					const hasRefreshToken = await tokenManager.getRefreshToken();

					if (hasRefreshToken) {
						// 有 RefreshToken，尝试刷新获取新的 AccessToken
						try {
							await tokenManager.refreshToken();
							// 刷新成功
							set({ initialized: true, isAuthenticated: true });
							return;
						} catch (error) {
							console.error("[AuthStore] 刷新 Token 失败，清除认证状态:", error);
							// 刷新失败时主动清理 token 存储，避免 HMR 后反复重试
							try {
								await tokenManager.clearTokens();
							} catch (e) {
								console.error("[AuthStore] clearTokens 失败:", e);
							}
						}
					}

					// 没有 RefreshToken 或刷新失败 — 直接设置未认证状态
					// 不用 state.logout()，因为 logout 内部的副作用（动态 import DynamicRoutes）
					// 在 HMR 路径上可能再次失败/循环；这里只需要最简的状态重置。
					set({
						user: null,
						isAuthenticated: false,
						menusLoaded: false,
						initialized: true,
					});
				} catch (fatalError) {
					// 兜底：任何未预期的异常都不能让 initialized 停在 false
					console.error("[AuthStore] initializeFromStorage 兜底:", fatalError);
					set({
						user: null,
						isAuthenticated: false,
						menusLoaded: false,
						initialized: true,
					});
				}
			},
		}),
		{
			name: "auth-storage",
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
				// 关键：委托给 initializeFromStorage，避免与它内部的 refreshToken 形成双调用
				// （之前的实现在这里直接调 refreshToken + state.initializeFromStorage，
				//   而 initializeFromStorage 内部又会再调一次 refreshToken，HMR 后会竞态）
				setTimeout(() => {
					state.initializeFromStorage();
				}, 0);
			},
		}
	)
);

/**
 * 导出 TokenManager 实例供拦截器使用
 */
export const getTokenManager = () => tokenManager;
