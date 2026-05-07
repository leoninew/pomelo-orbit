<script setup lang="ts">
import { ExternalLink, Plus, RefreshCw } from 'lucide-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';
import {
	DialogClose,
	DialogContent,
	DialogDescription,
	DialogOverlay,
	DialogPortal,
	DialogRoot,
	DialogTitle,
	SwitchRoot,
	SwitchThumb,
	ToolbarRoot,
} from 'reka-ui';
import type { Route } from '@/api/cd/route';
import { routeApi } from '@/api/cd/route';
import ListPagination from '@/components/ListPagination.vue';
import SearchControl from '@/components/SearchControl.vue';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';

const toast = useToast();
const { status, error, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const routes = ref<Route[]>([]);
const searchText = ref('');
const isCreateModalOpen = ref(false);
const pagination = reactive({ current: 1, pageSize: 10, total: 0 });

const filteredRoutes = computed(() => {
	if (!searchText.value.trim()) {
		return routes.value;
	}
	const search = searchText.value.toLowerCase();
	return routes.value.filter(
		(r) =>
			r.name.toLowerCase().includes(search) ||
			r.domain.toLowerCase().includes(search) ||
			r.target_url.toLowerCase().includes(search)
	);
});

const paginatedRoutes = computed(() => {
	const start = (pagination.current - 1) * pagination.pageSize;
	const end = start + pagination.pageSize;
	return filteredRoutes.value.slice(start, end);
});

const displayTotal = computed(() => filteredRoutes.value.length);
const totalPages = computed(() => Math.ceil(displayTotal.value / pagination.pageSize));

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
	try {
		await execute(async () => {
			const res = await routeApi.list(1, 100);
			routes.value = res.items;
			pagination.total = res.total;
		});
	} catch {
		toast.error('获取路由失败');
	}
}

function handleSearch() {
	pagination.current = 1;
}

function goPage(p: number) {
	pagination.current = p;
}

function handlePageSizeChange(pageSize: number) {
	pagination.pageSize = pageSize;
	pagination.current = 1;
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
	isCreateModalOpen.value = true;
}

async function handleSave() {
	if (!validate()) {
		return;
	}
	try {
		await executeOp(async () => {
			await routeApi.create(form);
			toast.success('添加成功');
			isCreateModalOpen.value = false;
			fetchData();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '保存失败');
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
	try {
		await executeOp(async () => {
			await routeApi.sync();
			toast.success('同步成功');
			fetchData();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '同步失败');
	}
}

onMounted(fetchData);
</script>

<template>
	<div class="space-y-6">
		<ToolbarRoot class="flex items-center justify-between gap-6" aria-label="路由工具栏">
			<SearchControl
				v-model="searchText"
				placeholder="搜索名称/域名/目标地址"
				:loading="status === 'loading'"
				@search="handleSearch"
			/>
			<div class="flex items-center gap-3">
				<button
					class="flex h-10 items-center gap-2 rounded-md bg-primary px-5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
					@click="openCreateModal"
				>
					<Plus class="size-4" />
					添加路由
				</button>
				<button
					class="h-10 rounded-md border border-border bg-background px-5 text-sm font-medium text-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50 flex items-center gap-2"
					:disabled="operating"
					@click="handleSync"
				>
					<RefreshCw class="size-4" />
					同步全部
				</button>
			</div>
		</ToolbarRoot>

		<!-- Table Card -->
		<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div v-if="status === 'loading'" class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error }}</p>
			</div>
			<div v-else-if="paginatedRoutes.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">{{ searchText ? '未找到匹配的路由' : '暂无路由' }}</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="w-full">
					<thead class="border-b border-border bg-muted/30">
						<tr>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">名称</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">域名</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">路径前缀</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">目标地址</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">状态</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">协议</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">操作</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border">
						<tr v-for="r in paginatedRoutes" :key="r.id" class="transition-colors hover:bg-muted/30">
							<td class="px-6 py-5 text-sm">
								<router-link :to="`/cd/routes/${r.id}`" class="text-primary hover:underline">
									{{ r.name }}
								</router-link>
							</td>
							<td class="px-6 py-5 text-sm">
								<a
									:href="`${r.https_enabled ? 'https' : 'http'}://${r.domain}`"
									target="_blank"
									class="text-primary hover:underline flex items-center gap-1"
								>
									{{ r.domain }}
									<ExternalLink class="size-3" />
								</a>
							</td>
							<td class="px-6 py-5 text-sm text-foreground">{{ r.path_prefix }}</td>
							<td class="px-6 py-5 text-sm text-foreground max-w-48 truncate">{{ r.target_url }}</td>
							<td class="px-6 py-5 text-sm">
								<span
									class="inline-block px-2 py-0.5 text-sm rounded border"
									:class="r.enabled ? 'bg-green-50 text-green-700 border-green-200' : 'bg-muted text-muted-foreground border-border'"
								>
									{{ r.enabled ? '启用' : '停用' }}
								</span>
							</td>
							<td class="px-6 py-5 text-sm">
								<span
									class="inline-block px-2 py-0.5 text-sm rounded border"
									:class="r.https_enabled ? 'bg-blue-50 text-blue-700 border-blue-200' : 'bg-muted text-muted-foreground border-border'"
								>
									{{ r.https_enabled ? 'HTTPS' : 'HTTP' }}
								</span>
							</td>
							<td class="px-6 py-5 text-sm">
								<div class="flex items-center gap-2">
									<router-link :to="`/cd/routes/${r.id}`" class="text-primary hover:underline">查看</router-link>
									<button v-if="!r.enabled" class="text-green-600 hover:underline" @click="handleEnable(r.id)">
										启用
									</button>
									<button v-else class="text-yellow-600 hover:underline" @click="handleDisable(r.id)">停用</button>
								</div>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>

		<ListPagination
			:current="pagination.current"
			:page-size="pagination.pageSize"
			:total="displayTotal"
			:total-pages="totalPages"
			@change-page="goPage"
			@change-page-size="handlePageSizeChange"
		/>

		<!-- Create modal -->
		<DialogRoot v-model:open="isCreateModalOpen">
			<DialogPortal>
				<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
				<DialogContent
					class="fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[min(520px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
				>
					<div class="mb-5 space-y-1">
						<DialogTitle class="text-lg font-semibold text-foreground">添加路由</DialogTitle>
						<DialogDescription class="text-sm text-muted-foreground">
							创建一个可同步到 Traefik 的 HTTP 路由。
						</DialogDescription>
					</div>
					
					<div class="space-y-4">
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">名称</label>
							<input
								v-model="form.name"
								type="text"
								class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
								:class="errors.name ? 'border-destructive' : 'border-input'"
								placeholder="example-route"
							/>
							<p v-if="errors.name" class="text-xs text-destructive">{{ errors.name }}</p>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">域名</label>
							<input
								v-model="form.domain"
								type="text"
								class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
								:class="errors.domain ? 'border-destructive' : 'border-input'"
								placeholder="example.com"
							/>
							<p v-if="errors.domain" class="text-xs text-destructive">{{ errors.domain }}</p>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">路径前缀</label>
							<input
								v-model="form.path_prefix"
								type="text"
								class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
								placeholder="/"
							/>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">目标地址</label>
							<input
								v-model="form.target_url"
								type="text"
								class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
								:class="errors.target_url ? 'border-destructive' : 'border-input'"
								placeholder="http://host:port"
							/>
							<p v-if="errors.target_url" class="text-xs text-destructive">
								{{ errors.target_url }}
							</p>
						</div>
						<label class="flex items-center gap-3 cursor-pointer">
							<SwitchRoot
								v-model:checked="form.enabled"
								class="relative h-6 w-11 cursor-pointer rounded-full bg-input outline-none transition-colors data-[state=checked]:bg-primary"
							>
								<SwitchThumb class="block size-5 translate-x-0.5 rounded-full bg-background shadow-sm transition-transform duration-100 data-[state=checked]:translate-x-[22px]" />
							</SwitchRoot>
							<span class="text-sm text-foreground">启用</span>
						</label>
					</div>
					
					<div class="mt-6 flex justify-end gap-2">
						<DialogClose as-child>
							<button class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50">
								取消
							</button>
						</DialogClose>
						<button
							class="flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
							:disabled="operating"
							@click="handleSave"
						>
							<span v-if="operating" class="size-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent" />
							保存
						</button>
					</div>
				</DialogContent>
			</DialogPortal>
		</DialogRoot>
	</div>
</template>
