<template>
	<div class="flex flex-col gap-4">
		<!-- Page header -->
		<div class="flex items-center justify-between gap-3">
			<h1 class="text-xl font-semibold shrink-0">应用管理</h1>
			<div class="flex items-center gap-2 flex-nowrap ml-auto">
				<!-- Search -->
				<label class="input input-sm input-bordered flex items-center gap-2 w-44">
					<Search class="size-3.5 text-base-content/60 shrink-0" />
					<input
						v-model="searchText"
						type="text"
						placeholder="搜索应用名称"
						class="min-w-0 w-full"
						@keyup.enter="handleSearch"
					/>
				</label>
				<!-- View toggle -->
				<div class="join shrink-0">
					<button
						class="join-item btn btn-sm"
						:class="viewMode === 'card' ? 'btn-primary' : 'btn-ghost'"
						@click="viewMode = 'card'"
					>
						<LayoutGrid class="size-4" />
					</button>
					<button
						class="join-item btn btn-sm"
						:class="viewMode === 'table' ? 'btn-primary' : 'btn-ghost'"
						@click="viewMode = 'table'"
					>
						<List class="size-4" />
					</button>
				</div>
				<button class="btn btn-sm btn-primary gap-1.5 shrink-0" @click="openCreateModal">
					<Plus class="size-4" />
					新建应用
				</button>
				<button class="btn btn-sm btn-ghost gap-1.5 shrink-0" @click="triggerImport">
					<Upload class="size-4" />
					导入
				</button>
				<input
					ref="fileInput"
					type="file"
					accept=".json"
					class="hidden"
					@change="handleFileImport"
				/>
			</div>
		</div>

		<!-- Loading -->
		<div v-if="status === 'loading'" class="flex justify-center py-16">
			<span class="loading loading-spinner loading-lg text-primary" />
		</div>
		<div v-else-if="status === 'error'" class="flex justify-center py-16 text-error text-sm">
			{{ error }}
		</div>

		<!-- Card view -->
		<template v-else-if="viewMode === 'card'">
			<div
				v-if="applications.length === 0"
				class="flex flex-col items-center gap-2 py-16 text-base-content/60"
			>
				<Inbox class="size-12" />
				<span>暂无应用</span>
			</div>
			<div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-4">
				<div
					v-for="app in applications"
					:key="app.id"
					class="card bg-base-100 shadow-sm border border-base-200 cursor-pointer hover:shadow-md transition-shadow overflow-hidden group"
					@click="$router.push(`/cd/applications/${app.id}`)"
				>
					<div class="card-body p-4 gap-3">
						<div class="flex items-start justify-between gap-2">
							<span class="font-semibold truncate">{{ app.name }}</span>
							<span class="badge badge-sm" :class="appBadgeClass(app.status)">
								{{ appStatusLabel(app.status) }}
							</span>
						</div>
						<div class="text-xs text-base-content/70">
							<span>
								编码:
								<code>{{ app.code }}</code>
							</span>
							<span class="mx-2 text-base-content/30">|</span>
							<span>拉取策略: {{ app.image_pull_policy }}</span>
						</div>
						<div class="flex items-end justify-between">
							<button class="link link-primary text-xs" @click.stop="viewLastDeployment(app.id)">
								最后部署
							</button>
							<div class="flex items-center gap-1" @click.stop>
								<span
									v-if="app.status === 'deploying'"
									class="loading loading-spinner loading-xs text-info w-6"
								/>
								<template v-else>
									<button
										v-if="app.status === 'deployed'"
										class="btn btn-xs btn-error btn-ghost invisible group-hover:visible"
										@click="handleStop(app)"
									>
										停止
									</button>
									<button
										v-else
										class="btn btn-xs btn-ghost invisible group-hover:visible"
										@click="handleDeploy(app)"
									>
										部署
									</button>
								</template>
							</div>
						</div>
					</div>
				</div>
			</div>

			<!-- Pagination -->
			<div v-if="totalPages > 0" class="flex justify-end mt-2">
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
		</template>

		<!-- Table view -->
		<div v-else class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table min-h-48">
				<thead>
					<tr class="text-base-content/60">
						<th>应用名称</th>
						<th>编码</th>
						<th>状态</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					<tr v-if="applications.length === 0">
						<td colspan="4" class="text-center py-8 text-base-content/60">暂无数据</td>
					</tr>
					<tr v-for="app in applications" :key="app.id" class="hover">
						<td>
							<router-link :to="`/cd/applications/${app.id}`" class="link link-primary font-medium">
								{{ app.name }}
							</router-link>
						</td>
						<td>
							<code class="text-xs">{{ app.code }}</code>
						</td>
						<td>
							<span class="badge badge-sm" :class="appBadgeClass(app.status)">
								{{ appStatusLabel(app.status) }}
							</span>
						</td>
						<td>
							<div class="flex items-center gap-2">
								<button
									class="link link-primary"
									@click="$router.push(`/cd/applications/${app.id}`)"
								>
									查看
								</button>
								<button
									class="link"
									:class="{
										'opacity-30 pointer-events-none':
											app.status === 'deployed' || app.status === 'deploying' || operating,
									}"
									@click="handleDeploy(app)"
								>
									部署
								</button>
								<button
									class="link link-error"
									:class="{
										'opacity-30 pointer-events-none': app.status !== 'deployed' || operating,
									}"
									@click="handleStop(app)"
								>
									停止
								</button>
							</div>
						</td>
					</tr>
				</tbody>
			</table>
			<!-- Table pagination -->
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
		<dialog ref="createDialogRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">新建应用</h3>
				<AppFormFields
					:form="form"
					:errors="formErrors"
					@update:form="Object.assign(form, $event)"
				/>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleCreateOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						保存
					</button>
					<button class="btn btn-ghost" @click="createDialogRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Import modal -->
		<dialog ref="importDialogRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">导入应用</h3>
				<AppFormFields
					:form="importForm"
					:errors="importErrors"
					@update:form="Object.assign(importForm, $event)"
				/>
				<div v-if="importForm.config_files.length > 0" class="text-sm text-base-content/60">
					包含 {{ importForm.config_files.length }} 个配置文件
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleImportOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						导入
					</button>
					<button class="btn btn-ghost" @click="importDialogRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { Search, LayoutGrid, List, Plus, Upload, Inbox } from 'lucide-vue-next';
import { applicationApi } from '@/api/application';
import { deploymentApi } from '@/api/deployments';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { appStatusLabel } from '@/utils/status';
import { delayAsync } from '@/utils/time';
import type { Application, ApplicationImportReq } from '@/types/api';

// Inline sub-component for shared form fields
import AppFormFields from './ApplicationFormFields.vue';

const $router = useRouter();
const toast = useToast();
const { status, error, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const applications = ref<Application[]>([]);
const searchText = ref('');
const viewMode = ref<'card' | 'table'>('card');
const fileInput = ref<HTMLInputElement>();
const createDialogRef = ref<HTMLDialogElement>();
const importDialogRef = ref<HTMLDialogElement>();

const pagination = reactive({ current: 1, pageSize: 12, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

// ── Status helpers ──
const badgeMap: Record<string, string> = {
	deployed: 'badge-outline badge-success',
	deploy_failed: 'badge-outline badge-error',
	deploying: 'badge-outline badge-info',
	undeployed: 'badge-ghost',
};
function appBadgeClass(s: string) {
	return badgeMap[s] ?? 'badge-ghost';
}

// ── Form state ──
const form = reactive({
	name: '',
	code: '',
	image_pull_policy: 'missing',
});
const formErrors = reactive({ name: '', code: '' });
const importForm = reactive({
	name: '',
	code: '',
	image_pull_policy: 'missing',
	config_files: [] as { path: string; content: string }[],
});
const importErrors = reactive({ name: '', code: '' });

function validateForm(f: typeof form, e: typeof formErrors) {
	e.name = f.name.trim() ? '' : '请输入应用名称';
	e.code = /^[a-z][a-z0-9-]*$/.test(f.code)
		? ''
		: '必须以小写字母开头，只能包含小写字母、数字和连字符';
	return !e.name && !e.code;
}

// ── Data fetching ──
async function fetchApplications() {
	try {
		await execute(async () => {
			const res = await applicationApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				search: searchText.value || undefined,
			});
			applications.value = res.items;
			pagination.total = res.total;
		});
	} catch {
		toast.error('获取应用列表失败');
	}
}

function handleSearch() {
	pagination.current = 1;
	fetchApplications();
}
function goPage(p: number) {
	pagination.current = p;
	fetchApplications();
}

// ── Deploy / Stop ──
async function pollDeployment(deploymentId: string, app: Application) {
	while (true) {
		await delayAsync(3000);
		try {
			const data = await deploymentApi.get(deploymentId);
			if (['ran_to_completion', 'faulted', 'canceled'].includes(data.status)) {
				const target = applications.value.find((a) => a.id === app.id);
				if (data.status === 'ran_to_completion') {
					if (target) target.status = 'deployed';
					toast.success(`${app.name} 部署成功`);
				} else {
					if (target) target.status = 'deploy_failed';
					toast.error(`${app.name} 部署失败`);
				}
				break;
			}
		} catch {
			break;
		}
	}
}

async function handleDeploy(app: Application) {
	const target = applications.value.find((a) => a.id === app.id);
	if (target) target.status = 'deploying';
	try {
		await executeOp(async () => {
			const { deployment_id } = await applicationApi.deploy(app.id);
			toast.success(`${app.name} 部署已触发`);
			pollDeployment(deployment_id, app);
		});
	} catch (error) {
		if (target) target.status = 'deploy_failed';
		toast.error(error instanceof Error ? error.message : '部署失败');
	}
}

async function handleStop(app: Application) {
	const prev = app.status;
	app.status = 'deploying';
	try {
		await executeOp(async () => {
			await applicationApi.stop(app.id);
			app.status = 'undeployed';
			toast.success(`${app.name} 已停止`);
		});
	} catch (error) {
		app.status = prev;
		toast.error(error instanceof Error ? error.message : '停止失败');
	}
}

async function viewLastDeployment(appId: string) {
	const resp = await deploymentApi.list({ application_id: appId, per_page: 1 });
	if (resp.items.length > 0) $router.push(`/cd/deployments/${resp.items[0].id}`);
	else $router.push(`/cd/deployments?application_id=${appId}`);
}

// ── Create ──
function openCreateModal() {
	Object.assign(form, {
		name: '',
		code: '',
		image_pull_policy: 'missing',
	});
	Object.assign(formErrors, { name: '', code: '' });
	createDialogRef.value?.showModal();
}

async function handleCreateOk() {
	if (!validateForm(form, formErrors)) return;
	try {
		await executeOp(async () => {
			await applicationApi.create({
				name: form.name,
				code: form.code,
				image_pull_policy: form.image_pull_policy,
			});
			toast.success('创建成功');
			createDialogRef.value?.close();
			fetchApplications();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '创建失败');
	}
}

// ── Import ──
function triggerImport() {
	fileInput.value?.click();
}

async function handleFileImport(event: Event) {
	const target = event.target as HTMLInputElement;
	const file = target.files?.[0];
	if (!file) return;
	try {
		const data = JSON.parse(await file.text()) as ApplicationImportReq;
		Object.assign(importForm, {
			name: data.name || '',
			code: data.code || '',
			image_pull_policy: data.image_pull_policy || 'missing',
			config_files: data.config_files || [],
		});
		Object.assign(importErrors, { name: '', code: '' });
		importDialogRef.value?.showModal();
	} catch {
		toast.error('解析文件失败');
	} finally {
		target.value = '';
	}
}

async function handleImportOk() {
	if (!validateForm(importForm, importErrors)) return;
	try {
		await executeOp(async () => {
			await applicationApi.importApplication({
				name: importForm.name,
				code: importForm.code,
				image_pull_policy: importForm.image_pull_policy,
				config_files: importForm.config_files,
			});
			toast.success('导入成功');
			importDialogRef.value?.close();
			fetchApplications();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '导入失败');
	}
}

onMounted(fetchApplications);
</script>
