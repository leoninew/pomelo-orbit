<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">Traefik HTTP Routers</h1>
			<div class="flex items-center gap-2">
				<button class="btn btn-sm btn-primary gap-1.5" @click="openDashboard">
					<ExternalLink class="size-4" />打开 Dashboard
				</button>
				<button class="btn btn-sm btn-ghost gap-1.5" :disabled="loading" @click="fetchRoutes">
					<RefreshCw class="size-4" :class="{ 'animate-spin': loading }" />刷新
				</button>
			</div>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table">
				<thead>
					<tr class="text-base-content/60">
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
					<tr v-if="loading">
						<td colspan="7" class="text-center py-8"><span class="loading loading-spinner loading-md text-primary" /></td>
					</tr>
					<tr v-else-if="routes.length === 0">
						<td colspan="7" class="text-center py-8 text-base-content/60">暂无路由</td>
					</tr>
					<tr v-for="r in routes" :key="r.name" class="hover">
						<td>{{ r.name }}</td>
						<td class="cell-muted">{{ r.provider }}</td>
						<td><span class="badge badge-sm" :class="r.status === 'enabled' ? 'badge-outline badge-success' : 'badge-ghost'">{{ r.status }}</span></td>
						<td>
							<a v-if="buildRouteUrl(r.rule, r.tls)" :href="buildRouteUrl(r.rule, r.tls)!" target="_blank" class="link link-primary font-mono text-xs flex items-center gap-1">
								{{ r.rule }}<ExternalLink class="size-3" />
							</a>
							<span v-else class="font-mono text-xs">{{ r.rule }}</span>
						</td>
						<td class="cell-muted">{{ r.service }}</td>
						<td>
							<div class="flex flex-wrap gap-1">
								<span v-for="ep in r.entrypoints" :key="ep" class="badge badge-sm badge-ghost">{{ ep }}</span>
							</div>
						</td>
						<td><span class="badge badge-sm" :class="r.tls ? 'badge-outline badge-info' : 'badge-ghost'">{{ r.tls ? 'HTTPS' : 'HTTP' }}</span></td>
					</tr>
				</tbody>
			</table>
		</div>
	</div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { ExternalLink, RefreshCw } from 'lucide-vue-next';
import { traefikRouteApi } from '@/api/traefik-route';
import type { TraefikRouter } from '@/api/traefik-route';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';

const toast = useToast();
const { loading, execute } = useStatusAsync();
const routes = ref<TraefikRouter[]>([]);

async function fetchRoutes() {
	try {
		await execute(async () => { const data = await traefikRouteApi.list(); routes.value = data.items; });
	} catch { toast.error('获取路由失败'); }
}

function buildRouteUrl(rule: string, tls: boolean): string | null {
	const match = rule.match(/Host\(`([^`]+)`\)/);
	if (!match) return null;
	return `${tls ? 'https' : 'http'}://${match[1]}`;
}

async function openDashboard() {
	try {
		const config = await traefikRouteApi.getConfig();
		window.open(`${config.https_enabled ? 'https' : 'http'}://${config.dashboard_domain}/dashboard/`, '_blank');
	} catch { toast.error('打开 Dashboard 失败'); }
}

onMounted(fetchRoutes);
</script>
