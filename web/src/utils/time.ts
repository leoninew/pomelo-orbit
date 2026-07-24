import dayjs, { extend, type Dayjs } from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import timezone from 'dayjs/plugin/timezone';
import utc from 'dayjs/plugin/utc';

extend(utc);
extend(timezone);
extend(relativeTime);

/**
 * 格式化时间 - 将 UTC 时间转换为本地时间显示
 */
export function formatTime(time?: string | null, format = 'YYYY-MM-DD HH:mm:ss'): string {
  if (!time) {
    return '-';
  }
  return dayjs.utc(time).local().format(format);
}

/**
 * 格式化相对时间 (如 "3分钟前")
 */
export function formatRelativeTime(time?: string | null): string {
  if (!time) {
    return '-';
  }
  return dayjs.utc(time).local().fromNow();
}

/**
 * 将本地时间转换为 UTC ISO 8601 格式
 * 用于提交数据到后端
 */
export function toUTC(localTime: string | Date): string {
  return dayjs(localTime).utc().toISOString();
}

/**
 * 获取当前时间的 UTC ISO 8601 格式
 */
export function nowUTC(): string {
  return dayjs().utc().toISOString();
}

/**
 * 获取今天开始时间 (本地时区的 00:00:00)
 */
export function getTodayStart(): Dayjs {
  return dayjs().startOf('day');
}

/**
 * 延迟指定毫秒数
 */
export function delayAsync(ms: number, signal?: AbortSignal): Promise<void> {
  return new Promise((resolve) => {
    if (signal?.aborted) {
      resolve();
      return;
    }
    const timer = setTimeout(finish, ms);
    signal?.addEventListener('abort', finish, { once: true });

    function finish() {
      clearTimeout(timer);
      signal?.removeEventListener('abort', finish);
      resolve();
    }
  });
}

/**
 * 计算耗时 - 返回人类可读的耗时字符串
 * @param startTime 开始时间 (UTC ISO 8601)
 * @param endTime 结束时间 (UTC ISO 8601)
 * @returns 耗时字符串，如 "2分30秒"、"1小时5分"、"—"
 */
export function formatDuration(startTime?: string | null, endTime?: string | null): string {
  if (!startTime || !endTime) {
    return '—';
  }

  const start = dayjs.utc(startTime);
  const end = dayjs.utc(endTime);
  const diffMs = end.diff(start);

  if (diffMs < 0) {
    return '—';
  }

  const seconds = Math.floor(diffMs / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  const days = Math.floor(hours / 24);

  if (days > 0) {
    const remainHours = hours % 24;
    return remainHours > 0 ? `${days}天${remainHours}小时` : `${days}天`;
  }

  if (hours > 0) {
    const remainMinutes = minutes % 60;
    return remainMinutes > 0 ? `${hours}小时${remainMinutes}分` : `${hours}小时`;
  }

  if (minutes > 0) {
    const remainSeconds = seconds % 60;
    return remainSeconds > 0 ? `${minutes}分${remainSeconds}秒` : `${minutes}分`;
  }

  return `${seconds}秒`;
}
