import axios, { type AxiosError, type AxiosRequestConfig } from 'axios';
import runtimeConfig from '@/config';
import { useAuthStore } from '@/stores/auth';
import { handleUnauthorized } from '@/utils/handle-unauthorized';

/**
 * API 错误契约。
 */
export interface HttpErrorResponse {
  code: string;
  error: string;
  requestId: string;
}

export type ApiErrorKind = 'api' | 'contract_mismatch' | 'network';

export class ApiError extends Error {
  constructor(
    message: string,
    public status?: number,
    public code?: string,
    public requestId?: string,
    public kind: ApiErrorKind = 'api'
  ) {
    super(message);
    this.name = 'ApiError';
  }
}

export function isHttpErrorResponse(value: unknown): value is HttpErrorResponse {
  if (typeof value !== 'object' || value === null) {
    return false;
  }
  const response = value as Record<string, unknown>;
  return [response.code, response.error, response.requestId].every(
    (item) => typeof item === 'string' && item.trim() !== ''
  );
}

export function toApiError(status: number | undefined, value: unknown): ApiError {
  if (isHttpErrorResponse(value)) {
    return new ApiError(value.error, status, value.code, value.requestId);
  }
  return new ApiError(
    '服务响应格式异常',
    status,
    'contract_mismatch',
    undefined,
    'contract_mismatch'
  );
}

export const DEFAULT_REQUEST_TIMEOUT = 30_000;
export const REMOTE_OPERATION_TIMEOUT = 120_000;

export function remoteRequestConfig(config: AxiosRequestConfig = {}): AxiosRequestConfig {
  return { ...config, timeout: REMOTE_OPERATION_TIMEOUT };
}

const request = axios.create({
  baseURL: runtimeConfig.publicUrl,
  timeout: DEFAULT_REQUEST_TIMEOUT,
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
  (error: AxiosError<unknown>) => {
    const status = error.response?.status;

    // 401 - token 失效，清除登录状态并跳转登录页
    if (status === 401) {
      handleUnauthorized();
      const apiError = toApiError(status, error.response?.data);
      if (apiError.kind === 'contract_mismatch') {
        return Promise.reject(apiError);
      }
      return Promise.reject(
        new ApiError('登录已过期，请重新登录', status, apiError.code, apiError.requestId)
      );
    }

    if (error.response) {
      return Promise.reject(toApiError(status, error.response.data));
    }

    // 处理网络错误等基础错误
    if (error.request) {
      return Promise.reject(
        new ApiError(
          '网络连接失败，请检查网络设置',
          undefined,
          'network_error',
          undefined,
          'network'
        )
      );
    }

    return Promise.reject(error);
  }
);

export default request;
export type { AxiosRequestConfig };
