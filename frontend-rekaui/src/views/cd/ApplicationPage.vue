<template>
	<div class="space-y-6">
		<ToolbarRoot class="app-toolbar-simple" aria-label="应用工具栏">
			<SearchControl
				v-model="searchText"
				placeholder="搜索应用名称"
				:loading="status === 'loading'"
				class="shrink-0"
				@search="handleSearch"
			/>
			<div class="flex items-center gap-3">
				<ToggleGroupRoot
					v-model="viewMode"
					type="single"
					class="flex h-10 overflow-hidden rounded-md border border-border bg-background"
					aria-label="应用视图"
				>
					<ToggleGroupItem
						value="card"
						class="flex size-10 items-center justify-center text-muted-foreground outline-none transition-colors hover:bg-muted/50 hover:text-foreground data-[state=on]:bg-primary/10 data-[state=on]:text-primary"
						aria-label="卡片视图"
					>
						<LayoutGrid class="size-4" />
					</ToggleGroupItem>
					<ToggleGroupItem
						value="table"
						class="flex size-10 items-center justify-center text-muted-foreground outline-none transition-colors hover:bg-muted/50 hover:text-foreground data-[state=on]:bg-primary/10 data-[state=on]:text-primary"
						aria-label="表格视图"
					>
						<List class="size-4" />
					</ToggleGroupItem>
				</ToggleGroupRoot>
				<button
					class="app-button-primary px-5"
					:disabled="status === 'loading'"
					@click="openCreateDialog"
				>
					<Plus class="size-4" />
					新建应用
				</button>
				<button class="app-button px-5" @click="triggerImport">
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

		<!-- 加载 / 错误 -->
		<div v-if="status === 'loading'" class="app-surface">
			<AppSpinner class="py-16" />
		</div>
		<div v-else-if="status === 'error'" class="app-surface">
			<div class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error || '加载失败' }}</p>
			</div>
		</div>

		<!-- 卡片视图 -->
		<template v-else-if="viewMode === 'card'">
			<div v-if="applications.length === 0" class="app-surface">
				<div class="flex flex-col items-center justify-center py-16">
					<Inbox class="size-12 text-muted-foreground" />
					<p class="mt-2 text-sm text-muted-foreground">暂无应用</p>
				</div>
			</div>
			<div v-else class="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
				<div
					v-for="app in applications"
					:key="app.id"
					class="app-surface group cursor-pointer p-4 transition-colors hover:border-primary"
					@click="router.push(`/cd/applications/${app.id}`)"
				>
					<div class="mb-3 flex items-start justify-between gap-2">
						<h3
							class="min-w-0 truncate text-sm font-medium text-foreground group-hover:text-primary"
						>
							{{ app.name }}
						</h3>
						<span
							class="inline-flex shrink-0 rounded-md border px-2 py-0.5 text-xs"
							:class="appBadgeClass(app.status)"
						>
							{{ appStatusLabel(app.status) }}
						</span>
					</div>
					<div class="mb-2 flex flex-wrap gap-x-3 gap-y-1 text-xs text-muted-foreground">
						<span>编码: {{ app.code }}</span>
						<span>拉取策略: {{ app.image_pull_policy }}</span>
					</div>
					<div class="mb-4 text-xs text-muted-foreground">
						路由托管:
						<span v-if="app.route_managed" class="font-medium text-green-600">已启用</span>
						<span v-else>未启用</span>
					</div>
					<div class="flex items-center justify-between" @click.stop>
						<span class="text-xs text-muted-foreground">{{ formatTime(app.created_at) }}</span>
						<div class="flex items-center gap-2">
							<button
								v-if="app.status === 'deployed'"
								class="app-link-danger text-xs sm:opacity-0 sm:transition-opacity sm:group-hover:opacity-100"
								:disabled="operating"
								@click="handleStop(app)"
							>
								停止
							</button>
							<button
								v-else
								class="app-link text-xs sm:opacity-0 sm:transition-opacity sm:group-hover:opacity-100"
								:disabled="app.status === 'deploying' || operating"
								@click="handleDeploy(app)"
							>
								部署
							</button>
						</div>
					</div>
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
		</template>

		<!-- 表格视图 -->
		<template v-else>
			<div class="app-surface">
				<div v-if="applications.length === 0" class="text-center py-16 text-muted-foreground">
					<p class="text-sm">暂无应用</p>
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
											停止
										</button>
										<button
											v-else
											class="app-link"
											:disabled="app.status === 'deploying' || operating"
											@click="handleDeploy(app)"
										>
											部署
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
		</template>

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
					保存
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
					导入
				</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { Inbox, LayoutGrid, List, Plus, Upload } from 'lucide-vue-next';
	import { computed, onMounted, reactive, ref } from 'vue';
	import { useRouter } from 'vue-router';
	import { applicationApi } from '@/api/cd/application';
	import AppDialog from '@/components/AppDialog.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
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
	import { ToggleGroupItem, ToggleGroupRoot, ToolbarRoot } from 'reka-ui';
	import ApplicationFormFields from './ApplicationFormFields.vue';

	const router = useRouter();
	const toast = useToast();
	const { status, error, execute } = useStatusAsync();
	const { loading: operating, execute: executeOp } = useStatusAsync();

	const applications = ref<Application[]>([]);
	const searchText = ref('');
	const viewMode = ref<'card' | 'table'>('card');
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
