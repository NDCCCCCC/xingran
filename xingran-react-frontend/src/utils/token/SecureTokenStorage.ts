/**
 * 安全 Token 存储接口定义
 * 使用国密 SM4 加密保护 RefreshToken
 */

/**
 * Token 元数据
 */
export interface TokenMeta {
	expiresAt: number; // AccessToken 过期时间戳（毫秒）
	issuedAt: number;  // 签发时间戳（毫秒）
	expiresIn: number; // 过期时长（秒）
}

/**
 * Token 刷新响应
 */
export interface TokenRefreshResponse {
	accessToken: string;
	refreshToken: string;
	expiresIn: number;
}

/**
 * Token 刷新错误
 */
export class TokenRefreshError extends Error {
	constructor(
		message: string,
		public code: "NETWORK_ERROR" | "INVALID_TOKEN" | "SERVER_ERROR"
	) {
		super(message);
		this.name = "TokenRefreshError";
	}
}

/**
 * 安全 Token 存储接口
 * 定义 Token 的存储、获取、删除操作
 */
export interface SecureTokenStorage {
	/**
	 * 存储 AccessToken（内存）
	 */
	setAccessToken(token: string): void;

	/**
	 * 获取 AccessToken（从内存）
	 */
	getAccessToken(): string | null;

	/**
	 * 存储 RefreshToken（sessionStorage，SM4 加密）
	 */
	setRefreshToken(token: string): Promise<void>;

	/**
	 * 获取 RefreshToken（从 sessionStorage，SM4 解密）
	 */
	getRefreshToken(): Promise<string | null>;

	/**
	 * 存储 Token 元数据（过期时间等）
	 */
	setTokenMeta(meta: TokenMeta): void;

	/**
	 * 获取 Token 元数据
	 */
	getTokenMeta(): TokenMeta | null;

	/**
	 * 清除所有 Token
	 */
	clear(): Promise<void>;

	/**
	 * 检查 AccessToken 是否即将过期（在指定秒数内）
	 */
	isAccessTokenExpiringWithin(seconds: number): boolean;
}
