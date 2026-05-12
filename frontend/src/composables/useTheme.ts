import { computed, ref, watch } from 'vue';

export type Theme = 'system' | 'light' | 'dark';
type EffectiveTheme = 'light' | 'dark';

const THEME_STORAGE_KEY = 'pomelo-orbit-theme';

const theme = ref<Theme>('system');
const systemTheme = ref<EffectiveTheme>('light');
const effectiveTheme = computed<EffectiveTheme>(() =>
	theme.value === 'system' ? systemTheme.value : theme.value
);
let initialized = false;

function isTheme(value: string | null): value is Theme {
	return value === 'system' || value === 'light' || value === 'dark';
}

function getSavedTheme(): Theme | null {
	if (typeof localStorage === 'undefined') {
		return null;
	}
	const savedTheme = localStorage.getItem(THEME_STORAGE_KEY);
	return isTheme(savedTheme) ? savedTheme : null;
}

function getSystemTheme(): EffectiveTheme {
	if (typeof window === 'undefined') {
		return 'light';
	}
	return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light';
}

function applyTheme(nextEffectiveTheme: EffectiveTheme) {
	if (typeof document === 'undefined') {
		return;
	}
	const root = document.documentElement;
	root.classList.toggle('dark', nextEffectiveTheme === 'dark');
	root.dataset.theme = nextEffectiveTheme;
}

function initializeTheme() {
	if (initialized) {
		return;
	}
	initialized = true;

	const savedTheme = getSavedTheme();
	if (savedTheme) {
		theme.value = savedTheme;
	}
	systemTheme.value = getSystemTheme();
	applyTheme(effectiveTheme.value);

	if (typeof window !== 'undefined') {
		const mediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
		mediaQuery.addEventListener('change', (event) => {
			systemTheme.value = event.matches ? 'dark' : 'light';
		});
	}

	watch([theme, effectiveTheme], ([nextTheme, nextEffectiveTheme]) => {
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem(THEME_STORAGE_KEY, nextTheme);
		}
		applyTheme(nextEffectiveTheme);
	});
}

export function useTheme() {
	initializeTheme();

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
		effectiveTheme,
		setTheme,
		cycleTheme,
	};
}
