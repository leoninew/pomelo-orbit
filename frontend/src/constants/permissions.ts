/**
 * 权限常量定义
 *
 * 集中管理所有权限字符串，避免硬编码和拼写错误
 */

export const PERMISSIONS = {
	// 用户管理权限
	USER_READ: 'user:read',
	USER_WRITE: 'user:write',

	// 角色管理权限
	ROLE_READ: 'role:read',
	ROLE_WRITE: 'role:write',
} as const;

/**
 * 权限类型
 */
export type Permission = (typeof PERMISSIONS)[keyof typeof PERMISSIONS];

/**
 * 检查是否为有效权限
 */
export function isValidPermission(permission: string): permission is Permission {
	return Object.values(PERMISSIONS).includes(permission as Permission);
}
