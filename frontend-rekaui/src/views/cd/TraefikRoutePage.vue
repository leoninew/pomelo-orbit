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
const { loading, execute } = useStatusAsync();
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
		toast.error('获取路由失败');
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

<template>
	<div class="space-y-6">
		<ToolbarRoot class="flex items-center justify-between gap-6" aria-label="Traefik 工具栏">
			<SearchControl
				v-model="searchText"
				placeholder="搜索名称/规则/服务/提供者"
				:loading="loading"
				@search="handleSearch"
			/>
			<div class="flex items-center gap-3">
				<button
					class="flex h-10 items-center gap-2 rounded-md bg-primary px-5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
					@click="openDashboard"
				>
					<ExternalLink class="size-4" />
					打开 Dashboard
				</button>
				<button
					class="h-10 rounded-md border border-border bg-background px-5 text-sm font-medium text-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50 flex items-center gap-2"
					:disabled="loading"
					@click="fetchRoutes"
				>
					<RefreshCw class="size-4" :class="{ 'animate-spin': loading }" />
					刷新
				</button>
			</div>
		</ToolbarRoot>

		<!-- Table Card -->
		<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div v-if="loading" class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
			<div v-else-if="filteredRoutes.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">{{ searchText ? '未找到匹配的路由' : '暂无路由' }}</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[1100px]">
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
							<td class="max-w-48 truncate text-foreground" :title="r.name">{{ r.name }}</td>
							<td class="text-foreground">{{ r.provider }}</td>
							<td>
								<span
									class="inline-block rounded border px-2 py-0.5 text-sm"
									:class="r.status === 'enabled' ? 'bg-green-50 text-green-700 border-green-200' : 'bg-muted text-muted-foreground border-border'"
								>
									{{ r.status }}
								</span>
							</td>
							<td class="max-w-80 truncate" :title="r.rule">
								<a
									v-if="buildRouteUrl(r.rule, r.tls)"
									:href="buildRouteUrl(r.rule, r.tls)!"
									target="_blank"
									class="flex items-center gap-1 text-primary hover:underline"
								>
									<span class="truncate">{{ r.rule }}</span>
									<ExternalLink class="size-3" />
								</a>
								<span v-else class="block truncate text-foreground">{{ r.rule }}</span>
							</td>
							<td class="max-w-56 truncate text-foreground" :title="r.service">{{ r.service }}</td>
							<td>
								<div class="flex max-w-56 flex-nowrap gap-1 overflow-hidden" :title="r.entrypoints.join(', ')">
									<span v-for="ep in r.entrypoints" :key="ep" class="inline-block shrink-0 rounded bg-muted px-2 py-0.5 text-sm text-muted-foreground">
										{{ ep }}
									</span>
								</div>
							</td>
							<td>
								<span
									class="inline-block rounded border px-2 py-0.5 text-sm"
									:class="r.tls ? 'bg-blue-50 text-blue-700 border-blue-200' : 'bg-muted text-muted-foreground border-border'"
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
