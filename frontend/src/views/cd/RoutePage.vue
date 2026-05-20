<template>
	<div class="space-y-6">
		<ToolbarRoot class="app-toolbar-simple" aria-label="路由工具栏">
			<SearchControl
				v-model="searchText"
				class="shrink-0"
				placeholder="搜索名称/域名/目标地址"
				:loading="status === 'loading'"
				@search="handleSearch"
			/>
			<div class="flex items-center gap-3">
				<button class="app-button-primary px-5" @click="openCreateModal">
					<Plus class="size-4" />
					添加路由
				</button>
				<button class="app-button px-5" :disabled="operating" @click="handleSync">
					<RefreshCw class="size-4" :class="{ 'animate-spin': operating }" />
					同步全部
				</button>
			</div>
		</ToolbarRoot>

		<div class="app-surface">
			<AppSpinner v-if="status === 'loading'" class="py-16" />
			<AppEmptyState v-else-if="routes.length === 0" />
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[1200px]">
					<colgroup>
						<col class="w-[14%]" />
						<col class="w-[16%]" />
						<col class="w-[10%]" />
						<col class="w-[20%]" />
						<col class="w-[8%]" />
						<col class="w-[8%]" />
						<col class="w-[14%]" />
						<col class="w-[10%]" />
					</colgroup>
					<thead>
						<tr>
							<th>名称</th>
							<th>域名</th>
							<th>路径前缀</th>
							<th>目标地址</th>
							<th>状态</th>
							<th>协议</th>
							<th>创建时间</th>
							<th>操作</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="route in routes" :key="route.id">
							<td>
								<router-link :to="`/cd/routes/${route.id}`" class="app-link whitespace-nowrap">
									{{ route.name }}
								</router-link>
							</td>
							<td>
								<a
									:href="`${route.https_enabled ? 'https' : 'http'}://${route.domain}`"
									target="_blank"
									class="app-link inline-flex items-center gap-1 whitespace-nowrap"
								>
									{{ route.domain }}
									<ExternalLink class="size-3" />
								</a>
							</td>
							<td class="whitespace-nowrap text-foreground">{{ route.path_prefix }}</td>
							<td class="max-w-0 truncate text-foreground" :title="route.target_url">
								{{ route.target_url }}
							</td>
							<td>
								<AppBadge variant="status" :tone="route.enabled ? 'success' : 'default'">
									{{ route.enabled ? '启用' : '停用' }}
								</AppBadge>
							</td>
							<td>
								<AppBadge variant="status" :tone="route.https_enabled ? 'info' : 'default'">
									{{ route.https_enabled ? 'HTTPS' : 'HTTP' }}
								</AppBadge>
							</td>
							<td class="whitespace-nowrap text-foreground">{{ formatTime(route.created_at) }}</td>
							<td class="whitespace-nowrap">
								<div class="flex items-center gap-3">
									<router-link :to="`/cd/routes/${route.id}`" class="app-link">查看</router-link>
									<button
										v-if="!route.enabled"
										class="app-link-success"
										@click="handleEnable(route.id)"
									>
										启用
									</button>
									<button v-else class="app-link-warning" @click="handleDisable(route.id)">
										停用
									</button>
								</div>
							</td>
						</tr>
					</tbody>
				</table>
			</div>

			<ListPagination
				:current="pagination.current"
				:page-size="pagination.pageSize"
				:total="pagination.total"
				:total-pages="totalPages"
				@change-page="goPage"
				@change-page-size="handlePageSizeChange"
			/>
		</div>
	</div>

	<AppDialog v-model:open="isCreateDialogOpen" title="添加路由">
		<div class="space-y-4">
			<div class="space-y-1.5">
				<label class="app-field-label block">名称</label>
				<input
					v-model="form.name"
					type="text"
					class="app-input"
					:class="errors.name ? 'app-input-error' : ''"
					placeholder="example-route"
				/>
				<p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
				<p v-else class="app-field-hint">小写字母开头，可含数字、点号、下划线和连字符</p>
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">域名</label>
				<input
					v-model="form.domain"
					type="text"
					class="app-input"
					:class="errors.domain ? 'app-input-error' : ''"
					placeholder="example.com"
				/>
				<p v-if="errors.domain" class="app-field-error text-xs">{{ errors.domain }}</p>
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">路径前缀</label>
				<input v-model="form.path_prefix" type="text" class="app-input" placeholder="/" />
				<p class="app-field-hint">匹配以该前缀开头的请求路径，默认 /</p>
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">目标地址</label>
				<input
					v-model="form.target_url"
					type="text"
					class="app-input"
					:class="errors.target_url ? 'app-input-error' : ''"
					placeholder="http://host:port"
				/>
				<p v-if="errors.target_url" class="app-field-error text-xs">
					{{ errors.target_url }}
				</p>
				<p v-else class="app-field-hint">格式：http://host:port，例如 http://127.0.0.1:8080</p>
			</div>
			<label class="flex cursor-pointer items-center gap-3">
				<SwitchRoot v-model:checked="form.enabled" class="app-switch-root">
					<SwitchThumb class="app-switch-thumb" />
				</SwitchRoot>
				<span class="text-sm text-foreground">启用</span>
			</label>
		</div>

		<template #footer>
			<button class="app-button" @click="isCreateDialogOpen = false">取消</button>
			<button class="app-button-primary" :disabled="operating" @click="handleSave">添加</button>
		</template>
	</AppDialog>
</template>

<script setup lang="ts">
	import { ExternalLink, Plus, RefreshCw } from 'lucide-vue-next';
	import { computed, onMounted, reactive, ref } from 'vue';
	import { SwitchRoot, SwitchThumb, ToolbarRoot } from 'reka-ui';
	import type { Route } from '@/api/cd/route';
	import { routeApi } from '@/api/cd/route';
	import AppBadge from '@/components/AppBadge.vue';
	import AppDialog from '@/components/AppDialog.vue';
	import AppEmptyState from '@/components/AppEmptyState.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import ListPagination from '@/components/ListPagination.vue';
	import SearchControl from '@/components/SearchControl.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import { useProjectStore } from '@/stores/project';
	import { formatTime } from '@/utils/time';

	const toast = useToast();
	const projectStore = useProjectStore();
	const { status, execute } = useStatusAsync();
	const { loading: operating, execute: executeOp } = useStatusAsync();

	const routes = ref<Route[]>([]);
	const searchText = ref('');
	const isCreateDialogOpen = ref(false);
	const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
	const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

	const form = reactive({
		name: '',
		domain: '',
		path_prefix: '/',
		target_url: 'http://',
		enabled: false,
	});
	const errors = reactive({ name: '', domain: '', target_url: '' });

	function validate() {
		errors.name = /^[a-z][a-z0-9._-]*$/.test(form.name)
			? ''
			: '必须以小写字母开头，只能包含小写字母、数字、点号、下划线和连字符';
		errors.domain = form.domain.trim() ? '' : '请输入域名';
		errors.target_url = /^https?:\/\/[a-zA-Z0-9.-]+:\d+$/.test(form.target_url)
			? ''
			: '格式应为 http://host:port';
		return !errors.name && !errors.domain && !errors.target_url;
	}

	async function fetchData() {
		const projectId = projectStore.activeProjectId;
		if (!projectId) {
			toast.error('未选择项目');
			return;
		}
		try {
			await execute(async () => {
				const res = await routeApi.list({
					page: pagination.current,
					per_page: pagination.pageSize,
					search: searchText.value || undefined,
					project_id: projectId,
				});
				routes.value = res.items;
				pagination.total = res.total;
			});
		} catch {
			toast.error('获取路由失败');
		}
	}

	function handleSearch() {
		pagination.current = 1;
		fetchData();
	}

	function goPage(page: number) {
		pagination.current = page;
		fetchData();
	}

	function handlePageSizeChange(pageSize: number) {
		pagination.pageSize = pageSize;
		pagination.current = 1;
		fetchData();
	}

	function openCreateModal() {
		Object.assign(form, {
			name: '',
			domain: '',
			path_prefix: '/',
			target_url: 'http://',
			enabled: false,
		});
		Object.assign(errors, { name: '', domain: '', target_url: '' });
		isCreateDialogOpen.value = true;
	}

	async function handleSave() {
		if (!validate()) {
			return;
		}
		const projectId = projectStore.activeProjectId;
		if (!projectId) {
			toast.error('未选择项目');
			return;
		}
		try {
			await executeOp(async () => {
				await routeApi.create(form, { project_id: projectId });
				toast.success('添加成功');
				isCreateDialogOpen.value = false;
				fetchData();
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '添加失败');
		}
	}

	async function handleEnable(id: string) {
		try {
			await executeOp(async () => {
				await routeApi.enable(id);
				toast.success('启用成功');
				fetchData();
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '启用失败');
		}
	}

	async function handleDisable(id: string) {
		try {
			await executeOp(async () => {
				await routeApi.disable(id);
				toast.success('停用成功');
				fetchData();
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '停用失败');
		}
	}

	async function handleSync() {
		const projectId = projectStore.activeProjectId;
		if (!projectId) {
			toast.error('未选择项目');
			return;
		}
		try {
			await executeOp(async () => {
				await routeApi.sync({ project_id: projectId });
				toast.success('同步成功');
				fetchData();
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '同步失败');
		}
	}

	onMounted(fetchData);
</script>
