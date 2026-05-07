<script setup lang="ts">
import {
	PanelLeftClose,
	PanelLeftOpen
} from 'lucide-vue-next'
import { computed, ref } from 'vue'
import { useRoute } from 'vue-router'
import AppTopBar from '@/components/AppTopBar.vue'
import { useToast } from '@/composables/useToast'
import { getNavigationScope, getPrimaryNavigationKey, secondaryNavigation } from '@/navigation'

const route = useRoute()
const { toasts } = useToast()

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
	<div class="h-screen bg-muted/30 text-foreground">
		<!-- Login page: no layout -->
		<RouterView v-if="isLoginPage" />

		<!-- Main layout -->
		<div v-else class="flex h-full flex-col gap-4 p-4">
			<AppTopBar :current-module="currentPrimaryModule" />

			<div class="flex min-h-0 flex-1 gap-4 overflow-hidden">
				<!-- Sidebar -->
				<aside
					v-if="currentScope"
					class="flex shrink-0 flex-col overflow-hidden rounded-lg border border-border bg-card shadow-sm transition-all duration-200"
					:class="collapsed ? 'w-16' : 'w-60'"
				>
					<nav class="flex-1 overflow-y-auto p-4">
						<div class="space-y-2">
							<RouterLink
								v-for="item in sidebarItems"
								:key="item.key"
								:to="item.path"
								class="flex h-10 items-center gap-3 rounded-md px-3 text-sm font-medium transition-colors"
								:class="
									selectedKey === item.key
										? 'bg-primary/10 text-primary'
										: 'text-muted-foreground hover:bg-muted/60 hover:text-foreground'
								"
								:title="collapsed ? item.label : undefined"
							>
								<component :is="item.icon" class="size-4 shrink-0" />
								<span v-if="!collapsed" class="truncate">{{ item.label }}</span>
							</RouterLink>
						</div>
					</nav>

					<button
						class="flex h-11 items-center justify-center border-t border-border text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground"
						@click="collapsed = !collapsed"
					>
						<PanelLeftClose v-if="!collapsed" class="size-4" />
						<PanelLeftOpen v-else class="size-4" />
					</button>
				</aside>

				<!-- Main content -->
				<main class="min-w-0 flex-1 overflow-y-auto rounded-lg border border-border bg-background p-6 shadow-sm">
					<RouterView />
				</main>
			</div>
		</div>

		<!-- Toast notifications -->
		<div class="fixed top-4 right-4 z-50 space-y-2">
			<div
				v-for="t in toasts"
				:key="t.id"
				class="px-4 py-3 rounded-lg shadow-lg text-white min-w-64 animate-slideIn"
				:class="{
					'bg-green-500': t.type === 'success',
					'bg-destructive': t.type === 'error',
					'bg-yellow-500': t.type === 'warning',
					'bg-primary': t.type === 'info'
				}"
			>
				{{ t.text }}
			</div>
		</div>
	</div>
</template>
