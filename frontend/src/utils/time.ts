import dayjs, { type Dayjs } from 'dayjs';
import utc from 'dayjs/plugin/utc';
import timezone from 'dayjs/plugin/timezone';
import relativeTime from 'dayjs/plugin/relativeTime';

dayjs.extend(utc);
dayjs.extend(timezone);
dayjs.extend(relativeTime);

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
export function delayAsync(ms: number): Promise<void> {
	return new Promise((resolve) => setTimeout(resolve, ms));
}
