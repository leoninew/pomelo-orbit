<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">路由配置</h1>
			<div class="flex items-center gap-2">
				<button class="btn btn-sm btn-primary gap-1.5" @click="openCreateModal">
					<Plus class="size-4" />
					添加路由
				</button>
				<button class="btn btn-sm btn-ghost gap-1.5" :disabled="operating" @click="handleSync">
					<RefreshCw class="size-4" />
					同步全部
				</button>
			</div>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table min-h-48">
				<thead>
					<tr class="text-base-content/60">
						<th>路由名称</th>
						<th>域名</th>
						<th>路径前缀</th>
						<th>目标地址</th>
						<th>状态</th>
						<th>协议</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					<tr v-if="status === 'loading'">
						<td colspan="7" class="text-center py-8">
							<span class="loading loading-spinner loading-md text-primary" />
						</td>
					</tr>
					<tr v-else-if="status === 'error'">
						<td colspan="7" class="text-center py-8 text-error">{{ error }}</td>
					</tr>
					<tr v-else-if="routes.length === 0">
						<td colspan="7" class="text-center py-8 text-base-content/60">暂无路由</td>
					</tr>
					<tr v-for="r in routes" :key="r.id" class="hover">
						<td>
							<router-link :to="`/cd/routes/${r.id}`" class="link link-primary font-medium">
								{{ r.name }}
							</router-link>
						</td>
						<td>
							<a
								:href="`${r.https_enabled ? 'https' : 'http'}://${r.domain}`"
								target="_blank"
								class="link link-primary flex items-center gap-1"
							>
								{{ r.domain }}
								<ExternalLink class="size-3" />
							</a>
						</td>
						<td class="cell-muted">{{ r.path_prefix }}</td>
						<td class="cell-muted max-w-48 truncate">{{ r.target_url }}</td>
						<td>
							<span
								class="badge badge-sm"
								:class="r.enabled ? 'badge-outline badge-success' : 'badge-ghost'"
							>
								{{ r.enabled ? '启用' : '停用' }}
							</span>
						</td>
						<td>
							<span
								class="badge badge-sm"
								:class="r.https_enabled ? 'badge-outline badge-info' : 'badge-ghost'"
							>
								{{ r.https_enabled ? 'HTTPS' : 'HTTP' }}
							</span>
						</td>
						<td>
							<div class="flex items-center gap-2">
								<router-link :to="`/cd/routes/${r.id}`" class="link link-primary">查看</router-link>
								<button v-if="!r.enabled" class="link link-success" @click="handleEnable(r.id)">
									启用
								</button>
								<button v-else class="link link-warning" @click="handleDisable(r.id)">停用</button>
							</div>
						</td>
					</tr>
				</tbody>
			</table>
			<div v-if="totalPages > 0" class="flex justify-end p-3 border-t border-base-200">
				<div class="join">
					<button
						v-for="p in totalPages"
						:key="p"
						class="join-item btn btn-sm"
						:class="p === pagination.current ? 'btn-primary' : 'btn-ghost'"
						@click="goPage(p)"
					>
						{{ p }}
					</button>
				</div>
			</div>
		</div>

		<!-- Create modal -->
		<dialog ref="createModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">添加路由</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">路由名称</legend>
						<input
							v-model="form.name"
							type="text"
							class="input w-full"
							:class="{ 'input-error': errors.name }"
							placeholder="my-route"
						/>
						<p v-if="errors.name" class="fieldset-label text-error">{{ errors.name }}</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">域名</legend>
						<input
							v-model="form.domain"
							type="text"
							class="input w-full"
							:class="{ 'input-error': errors.domain }"
							placeholder="example.com"
						/>
						<p v-if="errors.domain" class="fieldset-label text-error">{{ errors.domain }}</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">路径前缀</legend>
						<input v-model="form.path_prefix" type="text" class="input w-full" placeholder="/" />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">目标地址</legend>
						<input
							v-model="form.target_url"
							type="text"
							class="input w-full"
							:class="{ 'input-error': errors.target_url }"
							placeholder="http://host:port"
						/>
						<p v-if="errors.target_url" class="fieldset-label text-error">
							{{ errors.target_url }}
						</p>
					</fieldset>
					<label class="flex items-center gap-3 cursor-pointer">
						<input v-model="form.enabled" type="checkbox" class="toggle toggle-sm toggle-primary" />
						<span class="text-sm">启用</span>
					</label>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleSave">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						保存
					</button>
					<button class="btn btn-ghost" @click="createModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { Plus, RefreshCw, ExternalLink } from 'lucide-vue-next';
import { routeApi } from '@/api/cd/route';
import type { Route } from '@/api/cd/route';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';

const toast = useToast();
const { status, error, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const routes = ref<Route[]>([]);
const createModalRef = ref<HTMLDialogElement>();
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
	try {
		await execute(async () => {
			const res = await routeApi.list(pagination.current, pagination.pageSize);
			routes.value = res.items;
			pagination.total = res.total;
		});
	} catch {
		toast.error('获取路由失败');
	}
}

function goPage(p: number) {
	pagination.current = p;
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
	createModalRef.value?.showModal();
}

async function handleSave() {
	if (!validate()) return;
	try {
		await executeOp(async () => {
			await routeApi.create(form);
			toast.success('添加成功');
			createModalRef.value?.close();
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
