<template>
	<div class="space-y-6">
		<ToolbarRoot class="overflow-x-auto" aria-label="Traefik 工具栏">
			<div class="flex min-w-max items-center gap-3">
				<SearchControl
					v-model="searchText"
					class="shrink-0"
					placeholder="搜索名称/规则/服务/提供者"
					:loading="status === 'loading'"
					@search="handleSearch"
				/>
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
			<div v-if="status === 'loading'" class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
			<div v-else-if="status === 'error'" class="py-16 text-center">
				<p class="text-sm text-destructive">{{ error || '获取路由失败' }}</p>
				<p class="mt-1 text-xs text-muted-foreground">请检查 Traefik 服务是否正常运行</p>
				<button class="app-link mx-auto mt-3 block text-sm" @click="fetchRoutes">重试</button>
			</div>
			<div v-else-if="filteredRoutes.length === 0" class="py-16 text-center text-muted-foreground">
				<p class="text-sm">{{ searchText ? '未找到匹配的路由' : '暂无路由' }}</p>
			</div>
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
								<span
									class="inline-block rounded border px-2 py-0.5 text-sm"
									:class="
										r.status === 'enabled'
											? 'border-green-200 bg-green-50 text-green-700'
											: 'border-border bg-muted text-muted-foreground'
									"
								>
									{{ r.status }}
								</span>
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
									<span
										v-for="ep in r.entrypoints"
										:key="ep"
										class="inline-block shrink-0 rounded bg-muted px-2 py-0.5 text-sm text-muted-foreground"
									>
										{{ ep }}
									</span>
								</div>
							</td>
							<td class="whitespace-nowrap">
								<span
									class="inline-block rounded border px-2 py-0.5 text-sm"
									:class="
										r.tls
											? 'border-blue-200 bg-blue-50 text-blue-700'
											: 'border-border bg-muted text-muted-foreground'
									"
								>
									{{ r.tls ? 'HTTPS' : 'HTTP' }}
								</span>
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
	import SearchControl from '@/components/SearchControl.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';

	const toast = useToast();
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
		try {
			await execute(async () => {
				const data = await traefikRouteApi.list();
				routes.value = data.items;
			});
		} catch {
			// error state handled by useStatusAsync
		}
	}

	function handleSearch() {
		// 前端搜索，无需额外操作
	}

	function buildRouteUrl(rule: string, tls: boolean): string | null {
		const match = rule.match(/Host\(`([^`]+)`\)/);
		if (!match) {
			return null;
		}
		return `${tls ? 'https' : 'http'}://${match[1]}`;
	}

	async function openDashboard() {
		try {
			const config = await traefikRouteApi.getConfig();
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
