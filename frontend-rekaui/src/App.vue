<script setup lang="ts">
import {
	PanelLeftClose,
	PanelLeftOpen
} from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppToaster from '@/components/AppToaster.vue'
import AppTopBar from '@/components/AppTopBar.vue'
import { getNavigationScope, getPrimaryNavigationKey, secondaryNavigation } from '@/navigation'

const route = useRoute()

const collapsed = ref(false)
const currentScope = computed(() => getNavigationScope(route.path))
const selectedKey = computed(() => (route.meta.menuKey as string) ?? '')
const currentPrimaryModule = computed(() => getPrimaryNavigationKey(route.path))

const sidebarItems = computed(() => {
	if (!currentScope.value) {
		return []
	}
	return secondaryNavigation[currentScope.value] ?? []
})

const isLoginPage = computed(() => route.name === 'Login')
</script>

<template>
	<div class="min-h-screen bg-muted/30 text-foreground md:h-screen">
		<!-- Login page: no layout -->
		<RouterView v-if="isLoginPage" />

		<!-- Main layout -->
		<div v-else class="flex min-h-screen flex-col gap-3 p-3 md:h-full md:min-h-0 md:gap-4 md:p-4">
			<AppTopBar :current-module="currentPrimaryModule" />

			<div class="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden md:flex-row md:gap-4">
				<!-- Sidebar -->
				<aside
					v-if="currentScope"
					class="flex shrink-0 flex-col overflow-hidden rounded-lg border border-border bg-card shadow-sm transition-all duration-200"
					:class="collapsed ? 'md:w-16' : 'md:w-60'"
				>
					<nav class="flex-1 overflow-x-auto overflow-y-hidden p-3 md:overflow-y-auto md:p-4">
						<div class="flex gap-2 md:block md:space-y-2">
							<RouterLink
								v-for="item in sidebarItems"
								:key="item.key"
								:to="item.path"
								class="flex h-10 shrink-0 items-center gap-3 rounded-md px-3 text-sm font-medium transition-colors"
								:class="
									selectedKey === item.key
										? 'bg-primary/10 text-primary'
										: 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
								"
								:title="collapsed ? item.label : undefined"
							>
								<component :is="item.icon" class="size-4 shrink-0" />
								<span v-if="!collapsed" class="max-w-28 truncate md:max-w-none">{{ item.label }}</span>
							</RouterLink>
						</div>
					</nav>

					<button
						class="hidden h-11 items-center justify-center border-t border-border text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground md:flex"
						@click="collapsed = !collapsed"
					>
						<PanelLeftClose v-if="!collapsed" class="size-4" />
						<PanelLeftOpen v-else class="size-4" />
					</button>
				</aside>

				<!-- Main content -->
				<main class="min-w-0 flex-1 overflow-x-hidden overflow-y-auto rounded-lg border border-border bg-background p-4 shadow-sm md:p-6">
					<RouterView />
				</main>
			</div>
		</div>

		<AppToaster />
	</div>
</template>
