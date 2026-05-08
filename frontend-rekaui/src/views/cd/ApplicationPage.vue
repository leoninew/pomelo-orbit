<script setup lang="ts">
import { Plus, Upload } from 'lucide-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import { applicationApi } from '@/api/cd/application';
import AppDialog from '@/components/AppDialog.vue';
import ListPagination from '@/components/ListPagination.vue';
import SearchControl from '@/components/SearchControl.vue';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type {
	Application,
	ApplicationFormState,
	ApplicationImportReq,
	ApplicationImportState,
} from '@/types/cd/application';
import { appStatusLabel } from '@/utils/status';
import { formatTime } from '@/utils/time';
import { ToolbarRoot } from 'reka-ui';
import ApplicationFormFields from './ApplicationFormFields.vue';

const router = useRouter();
const toast = useToast();
const { status, error, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const applications = ref<Application[]>([]);
const searchText = ref('');
const isCreateDialogOpen = ref(false);
const isImportDialogOpen = ref(false);
const fileInput = ref<HTMLInputElement>();
const operatingAppId = ref<string | null>(null);
const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
const form = reactive<ApplicationFormState>({
	name: '',
	code: '',
	image_pull_policy: 'missing',
	route_managed: false,
});
const formErrors = reactive({ name: '', code: '' });
const importForm = reactive<ApplicationImportState>({
	version: undefined,
	name: '',
	code: '',
	image_pull_policy: 'missing',
	route_managed: false,
	config_files: [],
	service_configs: [],
	routes: [],
});
const importErrors = reactive({ name: '', code: '' });
const importSummary = computed(() =>
	[
		`配置文件 ${importForm.config_files.length}`,
		`服务配置 ${importForm.service_configs.length}`,
		`路由 ${importForm.routes.length}`,
	].join(' / ')
);

const badgeMap: Record<string, string> = {
	deployed: 'border-green-200 bg-green-50 text-green-700',
	deploy_failed: 'border-red-200 bg-red-50 text-red-700',
	deploying: 'border-blue-200 bg-blue-50 text-blue-700',
	undeployed: 'border-border bg-muted text-muted-foreground',
};

function appBadgeClass(s: string) {
	return badgeMap[s] ?? 'border-border bg-muted text-muted-foreground';
}

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

function handlePageSizeChange(pageSize: number) {
	pagination.pageSize = pageSize;
	pagination.current = 1;
	fetchApplications();
}

function validateForm(target: ApplicationFormState, errors: { name: string; code: string }) {
	errors.name = target.name.trim() ? '' : '请输入应用名称';
	errors.code = /^[a-z][a-z0-9-]*$/.test(target.code)
		? ''
		: '必须以小写字母开头，只能包含小写字母、数字和连字符';
	return !errors.name && !errors.code;
}

function resetForm(target: ApplicationFormState) {
	Object.assign(target, {
		name: '',
		code: '',
		image_pull_policy: 'missing',
		route_managed: false,
	});
}

function openCreateDialog() {
	resetForm(form);
	Object.assign(formErrors, { name: '', code: '' });
	isCreateDialogOpen.value = true;
}

async function handleCreateOk() {
	if (!validateForm(form, formErrors)) {
		return;
	}
	try {
		await executeOp(async () => {
			await applicationApi.create({
				name: form.name,
				code: form.code,
				image_pull_policy: form.image_pull_policy,
				route_managed: form.route_managed,
			});
			toast.success('创建成功');
			isCreateDialogOpen.value = false;
			await fetchApplications();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '创建失败');
	}
}

function triggerImport() {
	fileInput.value?.click();
}

async function handleFileImport(event: Event) {
	const target = event.target as HTMLInputElement;
	const file = target.files?.[0];
	if (!file) {
		return;
	}
	try {
		const data = JSON.parse(await file.text()) as ApplicationImportReq;
		if (!data.name || !data.code) {
			toast.error('文件格式错误：缺少必填字段 name 或 code');
			return;
		}
		Object.assign(importForm, {
			version: data.version,
			name: data.name,
			code: data.code,
			image_pull_policy: data.image_pull_policy,
			route_managed: data.route_managed,
			config_files: data.config_files ?? [],
			service_configs: data.service_configs ?? [],
			routes: data.routes ?? [],
		});
		Object.assign(importErrors, { name: '', code: '' });
		isImportDialogOpen.value = true;
	} catch {
		toast.error('解析文件失败');
	} finally {
		target.value = '';
	}
}

async function handleImportOk() {
	if (!validateForm(importForm, importErrors)) {
		return;
	}
	try {
		await executeOp(async () => {
			await applicationApi.importApplication({
				version: importForm.version,
				name: importForm.name,
				code: importForm.code,
				image_pull_policy: importForm.image_pull_policy,
				route_managed: importForm.route_managed,
				config_files: importForm.config_files,
				service_configs: importForm.service_configs,
				routes: importForm.routes,
			});
			toast.success('导入成功');
			isImportDialogOpen.value = false;
			await fetchApplications();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '导入失败');
	}
}

async function handleDeploy(app: Application) {
	operatingAppId.value = app.id;
	try {
		await executeOp(async () => {
			const { deployment_id } = await applicationApi.deploy(app.id);
			toast.success(`${app.name} 部署已触发`);
			router.push(`/cd/deployments/${deployment_id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '部署失败');
	} finally {
		operatingAppId.value = null;
	}
}

async function handleStop(app: Application) {
	operatingAppId.value = app.id;
	try {
		await executeOp(async () => {
			const { deployment_id } = await applicationApi.stop(app.id);
			toast.success(`${app.name} 停止已触发`);
			router.push(`/cd/deployments/${deployment_id}`);
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '停止失败');
	} finally {
		operatingAppId.value = null;
	}
}

onMounted(fetchApplications);
</script>

<template>
	<div class="space-y-6">
		<ToolbarRoot class="overflow-x-auto" aria-label="应用工具栏">
			<div class="flex min-w-max items-center gap-2">
				<SearchControl
					v-model="searchText"
					placeholder="搜索应用名称"
					:loading="status === 'loading'"
					class="shrink-0"
					@search="handleSearch"
				/>
				<button
					class="app-button-primary h-10 px-4"
					:disabled="status === 'loading'"
					@click="openCreateDialog"
				>
					<Plus class="size-4" />
					新建应用
				</button>
				<button class="app-button h-10 px-4" @click="triggerImport">
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
		</ToolbarRoot>

		<!-- Table Card -->
		<div class="app-surface">
			<div v-if="status === 'loading'" class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error || '加载失败' }}</p>
			</div>
			<div v-else-if="applications.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[1040px]">
					<thead>
						<tr>
							<th>名称</th>
							<th>编码</th>
							<th>拉取策略</th>
							<th>状态</th>
							<th>路由管理</th>
							<th>创建时间</th>
							<th>操作</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="app in applications" :key="app.id">
							<td>
								<button class="app-link" @click="router.push(`/cd/applications/${app.id}`)">
									{{ app.name }}
								</button>
							</td>
							<td class="text-foreground">{{ app.code }}</td>
							<td class="text-foreground">{{ app.image_pull_policy }}</td>
							<td>
								<span
									class="inline-flex rounded-md border px-2 py-0.5 text-sm"
									:class="appBadgeClass(app.status)"
								>
									{{ appStatusLabel(app.status) }}
								</span>
							</td>
							<td class="text-foreground">{{ app.route_managed ? '启用' : '未启用' }}</td>
							<td class="text-foreground">{{ formatTime(app.created_at) }}</td>
							<td>
								<div class="flex items-center gap-3">
									<button class="app-link" @click="router.push(`/cd/applications/${app.id}`)">
										查看
									</button>
									<button
										v-if="app.status === 'deployed'"
										class="app-link-danger"
										:disabled="operating"
										@click="handleStop(app)"
									>
										{{ operatingAppId === app.id ? '停止中' : '停止' }}
									</button>
									<button
										v-else
										class="app-link"
										:disabled="app.status === 'deploying' || operating"
										@click="handleDeploy(app)"
									>
										{{ app.status === 'deploying' ? '部署中' : '部署' }}
									</button>
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
			:total="pagination.total"
			:total-pages="totalPages"
			@change-page="goPage"
			@change-page-size="handlePageSizeChange"
		/>

		<AppDialog
			v-model:open="isCreateDialogOpen"
			title="新建应用"
			description="创建持续部署应用的基础信息。"
		>
			<ApplicationFormFields
				:form="form"
				:errors="formErrors"
				@update:form="Object.assign(form, $event)"
			/>
			<template #footer>
				<button class="app-button" @click="isCreateDialogOpen = false">取消</button>
				<button class="app-button-primary" :disabled="operating" @click="handleCreateOk">
					{{ operating ? '保存中...' : '保存' }}
				</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isImportDialogOpen"
			title="导入应用"
			description="从导出的 JSON 文件导入应用，可在导入前调整名称和代码。"
		>
			<ApplicationFormFields
				:form="importForm"
				:errors="importErrors"
				@update:form="Object.assign(importForm, $event)"
			/>
			<div class="app-tip">
				{{ importSummary }}
			</div>
			<template #footer>
				<button class="app-button" @click="isImportDialogOpen = false">取消</button>
				<button class="app-button-primary" :disabled="operating" @click="handleImportOk">
					{{ operating ? '导入中...' : '导入' }}
				</button>
			</template>
		</AppDialog>
	</div>
</template>
