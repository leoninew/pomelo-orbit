<template>
	<header
		class="sticky top-0 z-40 flex min-h-14 shrink-0 items-center overflow-hidden border-b border-border bg-card/95 px-3 shadow-sm backdrop-blur md:h-16 md:px-4"
	>
		<RouterLink
			to="/"
			class="flex h-14 w-auto shrink-0 items-center gap-3 rounded-md px-2 text-foreground outline-none transition-colors hover:text-primary focus-visible:ring-2 focus-visible:ring-ring/20 sm:px-3 md:h-full md:w-60"
			aria-label="Pomelo Orbit 首页"
		>
			<span class="flex size-9 items-center justify-center rounded-md bg-primary/10 text-primary">
				<Code2 class="size-5" />
			</span>
			<span class="hidden text-base font-semibold tracking-normal sm:inline">Pomelo Orbit</span>
		</RouterLink>

		<NavigationMenuRoot
			:model-value="currentModule ?? undefined"
			class="flex h-14 min-w-0 flex-1 overflow-x-auto md:h-full md:flex-none"
			aria-label="一级模块导航"
			:delay-duration="100"
			:skip-delay-duration="200"
		>
			<NavigationMenuList class="flex h-full min-w-max items-stretch gap-0">
				<NavigationMenuItem
					v-for="item in localizedPrimaryNavigation"
					:key="item.key"
					:value="item.key"
				>
					<NavigationMenuLink as-child :active="isActive(item.key)">
						<RouterLink
							:to="item.path"
							class="flex h-full min-w-24 items-center justify-center border-b-2 px-3 text-sm font-medium outline-none transition-colors focus-visible:ring-2 focus-visible:ring-ring/20 md:min-w-28 md:px-5"
							:class="
								isActive(item.key)
									? 'border-primary text-primary'
									: 'border-transparent text-muted-foreground hover:text-foreground'
							"
						>
							{{ item.label }}
						</RouterLink>
					</NavigationMenuLink>
				</NavigationMenuItem>
			</NavigationMenuList>
		</NavigationMenuRoot>

		<div class="hidden flex-1 md:block" />

		<ToolbarRoot class="hidden items-center gap-3 md:flex" aria-label="全局工具">
			<ToolbarButton
				class="inline-flex size-9 items-center justify-center rounded-md text-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/20"
				:aria-label="t('theme.' + theme)"
				@click="cycleTheme"
			>
				<Monitor v-if="theme === 'system'" class="size-5" />
				<Sun v-else-if="theme === 'light'" class="size-5" />
				<Moon v-else class="size-5" />
			</ToolbarButton>
			<ToolbarButton
				class="inline-flex h-9 min-w-12 items-center justify-center gap-1.5 rounded-md px-2 text-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/20"
				:aria-label="switchLocaleLabel"
				:title="switchLocaleLabel"
				@click="toggleLocale"
			>
				<Languages class="size-5" />
				<span class="text-xs font-semibold leading-none">{{ nextLocaleShortName }}</span>
			</ToolbarButton>
		</ToolbarRoot>

		<DropdownMenuRoot>
			<DropdownMenuTrigger
				class="ml-1 flex h-10 cursor-pointer items-center gap-2 rounded-md px-2 text-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/20 data-[state=open]:bg-accent data-[state=open]:text-accent-foreground md:ml-0 md:h-11 md:gap-3 md:px-3"
				aria-label="用户菜单"
			>
				<span
					class="flex size-8 items-center justify-center rounded-full bg-primary text-sm font-medium text-primary-foreground"
				>
					{{ userInitial }}
				</span>
				<span class="hidden text-sm lg:inline">{{ userName }}</span>
				<ChevronDown class="hidden size-4 text-muted-foreground sm:block" />
			</DropdownMenuTrigger>
			<DropdownMenuPortal>
				<DropdownMenuContent
					class="z-50 min-w-48 rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-lg outline-none data-[state=open]:animate-slideDownAndFade"
					align="end"
					:side-offset="8"
				>
					<DropdownMenuItem
						class="flex cursor-pointer items-center gap-2 rounded px-3 py-2 text-sm outline-none transition-colors data-[highlighted]:bg-accent data-[highlighted]:text-accent-foreground"
						@select="handleLogout"
					>
						<LogOut class="size-4" />
						{{ t('user.logout') }}
					</DropdownMenuItem>
				</DropdownMenuContent>
			</DropdownMenuPortal>
		</DropdownMenuRoot>
	</header>
</template>

<script setup lang="ts">
	import { ChevronDown, Code2, Languages, LogOut, Monitor, Moon, Sun } from 'lucide-vue-next';
	import { computed } from 'vue';
	import { useRouter } from 'vue-router';
	import { useI18n } from 'vue-i18n';
	import { primaryNavigation, type PrimaryNavigationKey } from '@/navigation';
	import { useAuthStore } from '@/stores/auth';
	import { useTheme } from '@/composables/useTheme';
	import { setLocale, type Locale } from '@/i18n';
	import {
		DropdownMenuContent,
		DropdownMenuItem,
		DropdownMenuPortal,
		DropdownMenuRoot,
		DropdownMenuTrigger,
		NavigationMenuItem,
		NavigationMenuLink,
		NavigationMenuList,
		NavigationMenuRoot,
		ToolbarButton,
		ToolbarRoot,
	} from 'reka-ui';

	const props = defineProps<{
		currentModule: PrimaryNavigationKey | null
	}>();

	const router = useRouter();
	const authStore = useAuthStore();
	const { theme, cycleTheme } = useTheme();
	const { t, locale } = useI18n({ useScope: 'global' });

	const userName = computed(() => authStore.user?.username || 'admin');
	const userInitial = computed(() => userName.value.slice(0, 1).toUpperCase());

	const localizedPrimaryNavigation = computed(() =>
		primaryNavigation.map((item) => ({
			...item,
			label: t(item.labelKey),
		}))
	);

	const currentLocale = computed(() => locale.value as Locale);
	const nextLocale = computed<Locale>(() => (currentLocale.value === 'zh-CN' ? 'en-US' : 'zh-CN'));
	const nextLocaleShortName = computed(() => (nextLocale.value === 'zh-CN' ? '中' : 'EN'));
	const nextLocaleName = computed(() =>
		t(`language.${nextLocale.value === 'zh-CN' ? 'zhCN' : 'enUS'}`)
	);
	const switchLocaleLabel = computed(() =>
		t('language.switchTo', { language: nextLocaleName.value })
	);

	function toggleLocale() {
		setLocale(nextLocale.value);
	}

	function isActive(moduleKey: PrimaryNavigationKey) {
		return props.currentModule === moduleKey;
	}

	async function handleLogout() {
		await authStore.logout();
		router.push('/login');
	}
</script>
