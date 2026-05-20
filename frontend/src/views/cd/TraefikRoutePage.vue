<template>
	<div class="space-y-6">
		<ToolbarRoot class="app-toolbar-simple" aria-label="Traefik 工具栏">
			<SearchControl
				v-model="searchText"
				placeholder="搜索名称/规则/服务/提供者"
				:loading="status === 'loading'"
			/>
			<div class="flex items-center gap-3">
				<button class="app-button-primary px-5" @click="openDashboard">
					<ExternalLink class="size-4" />
					打开 Dashboard
				</button>
				<button class="app-button px-5" :disabled="status === 'loading'" @click="fetchRoutes">
					<RefreshCw class="size-4" :class="{ 'animate-spin': status === 'loading' }" />
					刷新
				</button>
			</div>
		</ToolbarRoot>

		<!-- Table Card -->
		<div class="app-surface">
			<AppSpinner v-if="status === 'loading'" class="py-16" />
			<div v-else-if="status === 'error'" class="py-16 text-center">
				<p class="text-sm text-destructive">{{ error || '获取路由失败' }}</p>
				<p class="mt-1 text-xs text-muted-foreground">请检查 Traefik 服务是否正常运行</p>
				<button class="app-link mx-auto mt-3 block text-sm" @click="fetchRoutes">重试</button>
			</div>
			<AppEmptyState v-else-if="filteredRoutes.length === 0" />
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[1080px]">
					<colgroup>
						<col class="w-[20%]" />
						<col class="w-[10%]" />
						<col class="w-[8%]" />
						<col class="w-[30%]" />
						<col class="w-[18%]" />
						<col class="w-[10%]" />
						<col class="w-[4%]" />
					</colgroup>
					<thead>
						<tr>
							<th>名称</th>
							<th>提供者</th>
							<th>状态</th>
							<th>规则</th>
							<th>服务</th>
							<th>入口点</th>
							<th>协议</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="r in filteredRoutes" :key="r.name">
							<td class="max-w-0 truncate text-foreground" :title="r.name">{{ r.name }}</td>
							<td class="whitespace-nowrap text-foreground">{{ r.provider }}</td>
							<td>
								<AppBadge variant="status" :tone="r.status === 'enabled' ? 'success' : 'default'">
									{{ r.status }}
								</AppBadge>
							</td>
							<td class="max-w-0" :title="r.rule">
								<a
									v-if="buildRouteUrl(r.rule, r.tls)"
									:href="buildRouteUrl(r.rule, r.tls)!"
									target="_blank"
									class="app-link flex items-center gap-1"
								>
									<span class="truncate">{{ r.rule }}</span>
									<ExternalLink class="size-3 shrink-0" />
								</a>
								<span v-else class="block truncate text-foreground">{{ r.rule }}</span>
							</td>
							<td class="max-w-0 truncate text-foreground" :title="r.service">{{ r.service }}</td>
							<td class="max-w-0" :title="r.entrypoints.join(', ')">
								<div class="flex flex-nowrap gap-1 overflow-hidden">
									<AppBadge v-for="ep in r.entrypoints" :key="ep">
										{{ ep }}
									</AppBadge>
								</div>
							</td>
							<td class="whitespace-nowrap">
								<AppBadge variant="status" :tone="r.tls ? 'info' : 'default'">
									{{ r.tls ? 'HTTPS' : 'HTTP' }}
								</AppBadge>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
	import { ExternalLink, RefreshCw } from 'lucide-vue-next';
	import { computed, onMounted, ref } from 'vue';
	import { ToolbarRoot } from 'reka-ui';
	import type { TraefikRouter } from '@/api/cd/traefik-route';
	import { traefikRouteApi } from '@/api/cd/traefik-route';
	import AppBadge from '@/components/AppBadge.vue';
	import AppEmptyState from '@/components/AppEmptyState.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import SearchControl from '@/components/SearchControl.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import { useProjectStore } from '@/stores/project';

	const toast = useToast();
	const projectStore = useProjectStore();
	const { status, error, execute } = useStatusAsync();
	const routes = ref<TraefikRouter[]>([]);
	const searchText = ref('');

	const filteredRoutes = computed(() => {
		if (!searchText.value.trim()) {
			return routes.value;
		}
		const search = searchText.value.toLowerCase();
		return routes.value.filter(
			(r) =>
				r.name.toLowerCase().includes(search) ||
				r.rule.toLowerCase().includes(search) ||
				r.service.toLowerCase().includes(search) ||
				r.provider.toLowerCase().includes(search)
		);
	});

	async function fetchRoutes() {
		const projectId = projectStore.activeProjectId;
		if (!projectId) {
			toast.error('未选择项目');
			return;
		}
		try {
			await execute(async () => {
				const data = await traefikRouteApi.list({ project_id: projectId });
				routes.value = data.items;
			});
		} catch {
			// error state handled by useStatusAsync
		}
	}

	function buildRouteUrl(rule: string, tls: boolean): string | null {
		const match = rule.match(/Host\(`([^`]+)`\)/);
		if (!match) {
			return null;
		}
		return `${tls ? 'https' : 'http'}://${match[1]}`;
	}

	async function openDashboard() {
		const projectId = projectStore.activeProjectId;
		if (!projectId) {
			toast.error('未选择项目');
			return;
		}
		try {
			const config = await traefikRouteApi.getConfig({ project_id: projectId });
			window.open(
				`${config.https_enabled ? 'https' : 'http'}://${config.dashboard_domain}/dashboard/`,
				'_blank'
			);
		} catch {
			toast.error('打开 Dashboard 失败');
		}
	}

	onMounted(fetchRoutes);
</script>
