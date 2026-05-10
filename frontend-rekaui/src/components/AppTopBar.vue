<template>
	<header
		class="flex min-h-16 shrink-0 items-center overflow-hidden rounded-2xl border border-gray-100 bg-card px-3 shadow-sm md:h-20"
	>
		<RouterLink
			to="/"
			class="flex h-16 w-auto shrink-0 items-center gap-3 rounded-md px-2 text-foreground outline-none transition-colors hover:text-primary focus-visible:ring-2 focus-visible:ring-ring/20 sm:px-3 md:h-full md:w-60"
			aria-label="Pomelo Orbit 首页"
		>
			<span class="flex size-9 items-center justify-center rounded-md bg-primary/10 text-primary">
				<Code2 class="size-5" />
			</span>
			<span class="hidden text-base font-semibold tracking-normal sm:inline">Pomelo Orbit</span>
		</RouterLink>

		<NavigationMenuRoot
			:model-value="currentModule ?? undefined"
			class="flex h-16 min-w-0 flex-1 overflow-x-auto md:h-full md:flex-none"
			aria-label="一级模块导航"
			:delay-duration="100"
			:skip-delay-duration="200"
		>
			<NavigationMenuList class="flex h-full min-w-max items-stretch gap-0">
				<NavigationMenuItem v-for="item in primaryNavigation" :key="item.key" :value="item.key">
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
				aria-label="搜索"
			>
				<Search class="size-5" />
			</ToolbarButton>
			<ToolbarButton
				class="inline-flex size-9 items-center justify-center rounded-md text-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/20"
				aria-label="通知"
			>
				<Bell class="size-5" />
			</ToolbarButton>
			<ToolbarButton
				class="inline-flex size-9 items-center justify-center rounded-md text-foreground outline-none transition-colors hover:bg-accent hover:text-accent-foreground focus-visible:ring-2 focus-visible:ring-ring/20"
				aria-label="帮助"
			>
				<CircleHelp class="size-5" />
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
						退出登录
					</DropdownMenuItem>
				</DropdownMenuContent>
			</DropdownMenuPortal>
		</DropdownMenuRoot>
	</header>
</template>

<script setup lang="ts">
	import { Bell, ChevronDown, CircleHelp, Code2, LogOut, Search } from 'lucide-vue-next';
	import { computed } from 'vue';
	import { useRouter } from 'vue-router';
	import { primaryNavigation, type PrimaryNavigationKey } from '@/navigation';
	import { useAuthStore } from '@/stores/auth';
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

	const userName = computed(() => authStore.user?.username || 'admin');
	const userInitial = computed(() => userName.value.slice(0, 1).toUpperCase());

	function isActive(moduleKey: PrimaryNavigationKey) {
		return props.currentModule === moduleKey;
	}

	async function handleLogout() {
		await authStore.logout();
		router.push('/login');
	}
</script>
