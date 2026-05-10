import { ref, watch, onMounted, onUnmounted } from 'vue';

export type Theme = 'system' | 'light' | 'dark';

const THEME_STORAGE_KEY = 'pomelo-orbit-theme';

// 全局状态
const theme = ref<Theme>('system');

function getSystemTheme(): 'light' | 'dark' {
	if (typeof window === 'undefined') {
		return 'light';
	}
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function getEffectiveTheme(currentTheme: Theme): 'light' | 'dark' {
	if (currentTheme === 'system') {
		return getSystemTheme();
	}
	return currentTheme;
}

function applyTheme(effectiveTheme: 'light' | 'dark') {
	const root = document.documentElement;
	if (effectiveTheme === 'dark') {
		root.classList.add('dark');
	} else {
		root.classList.remove('dark');
	}
}

export function useTheme() {
	let mediaQuery: MediaQueryList | undefined;
	let handleSystemThemeChange: (() => void) | undefined;

	onMounted(() => {
		// 从 localStorage 读取保存的主题
		const savedTheme = localStorage.getItem(THEME_STORAGE_KEY) as Theme | null;
		if (savedTheme && ['system', 'light', 'dark'].includes(savedTheme)) {
			theme.value = savedTheme;
		}

		// 应用初始主题
		const effectiveTheme = getEffectiveTheme(theme.value);
		applyTheme(effectiveTheme);

		// 监听系统主题变化
		mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
		handleSystemThemeChange = () => {
			if (theme.value === 'system') {
				const newEffectiveTheme = getSystemTheme();
				applyTheme(newEffectiveTheme);
			}
		};

		mediaQuery.addEventListener('change', handleSystemThemeChange);
	});

	onUnmounted(() => {
		if (mediaQuery && handleSystemThemeChange) {
			mediaQuery.removeEventListener('change', handleSystemThemeChange);
		}
	});

	// 监听主题变化
	watch(theme, (newTheme) => {
		localStorage.setItem(THEME_STORAGE_KEY, newTheme);
		const effectiveTheme = getEffectiveTheme(newTheme);
		applyTheme(effectiveTheme);
	});

	function setTheme(newTheme: Theme) {
		theme.value = newTheme;
	}

	function cycleTheme() {
		const themes: Theme[] = ['system', 'light', 'dark'];
		const currentIndex = themes.indexOf(theme.value);
		const nextIndex = (currentIndex + 1) % themes.length;
		setTheme(themes[nextIndex]);
	}

	return {
		theme,
		setTheme,
		cycleTheme,
	};
}
