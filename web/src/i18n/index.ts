import { createI18n } from 'vue-i18n';
import zhCN from './locales/zh-CN';
import enUS from './locales/en-US';

export const SUPPORTED_LOCALES = ['zh-CN', 'en-US'] as const;
export type Locale = (typeof SUPPORTED_LOCALES)[number];

const LOCALE_STORAGE_KEY = 'pomelo-orbit-locale';

function isLocale(value: string | null): value is Locale {
  return SUPPORTED_LOCALES.includes(value as Locale);
}

// 获取默认语言
function getDefaultLocale(): Locale {
  // 1. 从 localStorage 读取
  const savedLocale =
    typeof localStorage === 'undefined' ? null : localStorage.getItem(LOCALE_STORAGE_KEY);
  if (isLocale(savedLocale)) {
    return savedLocale;
  }

  // 2. 从浏览器语言推断
  const browserLang = typeof navigator === 'undefined' ? '' : navigator.language;
  if (browserLang.startsWith('zh')) {
    return 'zh-CN';
  }
  if (browserLang.startsWith('en')) {
    return 'en-US';
  }

  // 3. 默认中文
  return 'zh-CN';
}

function syncDocumentLocale(locale: Locale) {
  if (typeof document !== 'undefined') {
    document.documentElement.setAttribute('lang', locale);
  }
}

const defaultLocale = getDefaultLocale();
syncDocumentLocale(defaultLocale);

const i18n = createI18n({
  legacy: false, // 使用 Composition API 模式
  locale: defaultLocale,
  fallbackLocale: 'zh-CN',
  messages: {
    'zh-CN': zhCN,
    'en-US': enUS,
  },
});

export default i18n;

// 导出切换语言的函数
export function setLocale(locale: Locale) {
  i18n.global.locale.value = locale;
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(LOCALE_STORAGE_KEY, locale);
  }
  syncDocumentLocale(locale);
}

// 导出获取当前语言的函数
export function getLocale(): Locale {
  return i18n.global.locale.value as Locale;
}
