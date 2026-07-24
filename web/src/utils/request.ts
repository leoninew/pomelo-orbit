import axios, { type AxiosError, type AxiosRequestConfig } from 'axios';
import runtimeConfig from '@/config';
import { useAuthStore } from '@/stores/auth';
import { handleUnauthorized } from '@/utils/handle-unauthorized';

/**
 * API 错误类，保留 HTTP 状态码
 */
interface ErrorResp {
  detail?: string;
}

export class ApiError extends Error {
  constructor(
    message: string,
    public status?: number
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

const request = axios.create({
  baseURL: runtimeConfig.publicUrl,
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
  (error: AxiosError<ErrorResp>) => {
    // 401 - token 失效，清除登录状态并跳转登录页
    if (error.response?.status === 401) {
      handleUnauthorized();
      return Promise.reject(new ApiError('登录已过期，请重新登录', 401));
    }

    if (error.response?.data?.detail) {
      return Promise.reject(new ApiError(error.response.data.detail, error.response.status));
    }

    // 处理网络错误等基础错误
    if (error.request) {
      return Promise.reject(new ApiError('网络连接失败，请检查网络设置'));
    }

    return Promise.reject(error);
  }
);

export default request;
export type { AxiosRequestConfig };
