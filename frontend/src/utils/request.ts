import type { AxiosRequestConfig } from 'axios';
import axios, { type AxiosError } from 'axios';
import config from '@/config';
import { useAuthStore } from '@/stores/auth';
import router from '@/router';

const request = axios.create({
	baseURL: config.apiBaseUrl,
	timeout: 30000,
});

// 请求拦截器 - 添加 token
request.interceptors.request.use(
	(axiosConfig) => {
		const authStore = useAuthStore();
		if (authStore.token) {
			axiosConfig.headers.Authorization = `Bearer ${authStore.token}`;
		}
		return axiosConfig;
	},
	() => {
		return Promise.reject(new Error('Request error'));
	}
);

// 响应拦截器 - 处理基础错误
request.interceptors.response.use(
	(response) => response.data,
	(error: AxiosError<{ detail?: string }>) => {
		// 401 - token 失效，清除登录状态并跳转登录页
		if (error.response?.status === 401) {
			const authStore = useAuthStore();
			authStore.clearToken();
			router.push({ name: 'Login', query: { redirect: router.currentRoute.value.fullPath } });
			return Promise.reject(new Error('登录已过期，请重新登录'));
		}

		if (error.response?.data?.detail) {
			return Promise.reject(new Error(error.response.data.detail));
		}

		// 处理网络错误等基础错误
		if (error.request) {
			return Promise.reject(new Error('网络连接失败，请检查网络设置'));
		}

		return Promise.reject(error);
	}
);

export default request;
export type { AxiosRequestConfig };
