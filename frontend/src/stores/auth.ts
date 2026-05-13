import { defineStore } from 'pinia';
import { computed, ref } from 'vue';
import { authApi } from '@/api/auth';
import type { UserInfo } from '@/types/auth';

const TOKEN_KEY = 'pomelo_orbit_token';

export const useAuthStore = defineStore('auth', () => {
	const token = ref(localStorage.getItem(TOKEN_KEY));
	const user = ref<UserInfo>();

	const isAuthenticated = computed(() => !!token.value);

	// 设置 token
	function setToken(newToken: string) {
		token.value = newToken;
		localStorage.setItem(TOKEN_KEY, newToken);
	}

	// 设置用户信息
	function setUser(userInfo: UserInfo) {
		user.value = userInfo;
	}

	// 清除 token
	function clearToken() {
		token.value = null;
		user.value = undefined;
		localStorage.removeItem(TOKEN_KEY);
	}

	// 登录
	async function login(
		username: string,
		password: string,
		csrfToken: string,
		captchaToken: string,
		captchaAnswer: string
	) {
		const response = await authApi.login({
			username,
			password,
			csrf_token: csrfToken,
			captcha_token: captchaToken,
			captcha_answer: captchaAnswer,
		});
		setToken(response.access_token);
		await fetchUser();
		return response;
	}

	// 登出
	async function logout() {
		try {
			await authApi.logout();
		} catch (error) {
			console.error('Failed to logout:', error);
		} finally {
			clearToken();
		}
	}

	// 获取当前用户信息
	async function fetchUser() {
		if (!token.value) {
			return undefined;
		}
		try {
			const userInfo = await authApi.getCurrentUser();
			user.value = userInfo;
			return userInfo;
		} catch (error) {
			console.error('Failed to fetch user:', error);
			clearToken();
			return undefined;
		}
	}

	// 修改密码
	async function changePassword(oldPassword: string, newPassword: string) {
		await authApi.changePassword({
			old_password: oldPassword,
			new_password: newPassword,
		});
	}

	return {
		token,
		user,
		isAuthenticated,
		setToken,
		setUser,
		clearToken,
		login,
		logout,
		fetchUser,
		changePassword,
	};
});
