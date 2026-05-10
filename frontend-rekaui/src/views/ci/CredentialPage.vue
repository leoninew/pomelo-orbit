<template>
	<div class="space-y-6">
		<ToolbarRoot class="app-toolbar-simple" aria-label="凭据工具栏">
			<SearchControl
				v-model="searchText"
				placeholder="搜索凭据名称"
				:loading="status === 'loading'"
				@search="handleSearch"
			/>
			<div class="flex items-center gap-3">
				<button class="app-button px-5" @click="triggerImport">
					<Upload class="size-4" />
					导入
				</button>
				<button class="app-button-primary px-5" @click="openCreateModal">
					<Plus class="size-4" />
					新建凭据
				</button>
			</div>
		</ToolbarRoot>

		<div class="app-surface">
			<AppSpinner v-if="status === 'loading'" class="py-16" />
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error || '加载失败' }}</p>
			</div>
			<div v-else-if="credentials.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[880px]">
					<thead>
						<tr>
							<th>名称</th>
							<th>类型</th>
							<th>创建时间</th>
							<th class="text-right">操作</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="cred in credentials" :key="cred.id">
							<td>
								<router-link :to="`/ci/credential/${cred.id}`" class="app-link">
									{{ cred.name }}
								</router-link>
							</td>
							<td>
								<span
									class="inline-flex rounded-md border border-blue-200 bg-blue-50 px-2 py-0.5 text-sm text-blue-700"
								>
									{{ credentialTypeLabels[cred.type] || cred.type }}
								</span>
							</td>
							<td class="text-foreground">{{ formatTime(cred.created_at) }}</td>
							<td class="text-right">
								<button class="app-link mr-3" @click="openEditModal(cred)">编辑</button>
								<button class="app-link-danger" @click="confirmDelete(cred.id)">删除</button>
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

		<input ref="fileInput" type="file" accept=".json" class="hidden" @change="handleFileImport" />
	</div>

	<AppDialog
		v-model:open="showCredentialDialog"
		:title="isEditing ? '编辑凭据' : '新建凭据'"
		width-class="w-[min(600px,calc(100vw-32px))]"
	>
		<div class="space-y-4">
			<div class="space-y-1.5">
				<label class="app-field-label block">凭据名称</label>
				<input
					v-model="form.name"
					type="text"
					placeholder="输入凭据名称"
					class="app-input"
					:class="errors.name ? 'app-input-error' : ''"
				/>
				<p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">凭据类型</label>
				<SelectControl
					v-model="form.type"
					:options="credentialTypeOptions"
					:disabled="isEditing"
					placeholder="选择凭据类型"
				/>
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">
					凭据内容 {{ isEditing ? '（留空表示不修改）' : '' }}
				</label>
				<textarea
					v-model="form.data"
					rows="8"
					:placeholder="getDataPlaceholder(form.type)"
					class="app-textarea font-mono"
					:class="errors.data ? 'app-input-error' : ''"
				/>
				<p v-if="errors.data" class="app-field-error text-xs">{{ errors.data }}</p>
			</div>
		</div>
		<template #footer>
			<button class="app-button" @click="showCredentialDialog = false">取消</button>
			<button class="app-button-primary" :disabled="operating" @click="handleModalOk">
				{{ isEditing ? '保存' : '创建' }}
			</button>
		</template>
	</AppDialog>

	<AppDialog
		v-model:open="showDeleteDialog"
		title="确认删除"
		width-class="w-[min(420px,calc(100vw-32px))]"
	>
		<p class="text-sm text-foreground">确定要删除这个凭据吗？此操作不可恢复。</p>
		<template #footer>
			<button class="app-button" @click="showDeleteDialog = false">取消</button>
			<button class="app-button-destructive" :disabled="operating" @click="handleDelete">
				删除
			</button>
		</template>
	</AppDialog>

	<AppDialog
		v-model:open="showImportDialog"
		title="导入凭据"
		width-class="w-[min(600px,calc(100vw-32px))]"
	>
		<div class="space-y-4">
			<div class="space-y-1.5">
				<label class="app-field-label block">凭据名称</label>
				<input
					v-model="importForm.name"
					type="text"
					placeholder="输入凭据名称"
					class="app-input"
					:class="importErrors.name ? 'app-input-error' : ''"
				/>
				<p v-if="importErrors.name" class="app-field-error text-xs">{{ importErrors.name }}</p>
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">凭据类型</label>
				<SelectControl
					v-model="importForm.type"
					:options="credentialTypeOptions"
					placeholder="选择凭据类型"
				/>
			</div>
			<div class="space-y-1.5">
				<label class="app-field-label block">凭据内容</label>
				<textarea
					v-model="importForm.data"
					rows="8"
					class="app-textarea font-mono"
					:class="importErrors.data ? 'app-input-error' : ''"
				/>
				<p v-if="importErrors.data" class="app-field-error text-xs">{{ importErrors.data }}</p>
			</div>
		</div>
		<template #footer>
			<button class="app-button" @click="showImportDialog = false">取消</button>
			<button class="app-button-primary" :disabled="operating" @click="handleImportOk">导入</button>
		</template>
	</AppDialog>
</template>

<script setup lang="ts">
	import { Plus, Upload } from 'lucide-vue-next';
	import { computed, onMounted, reactive, ref } from 'vue';
	import { credentialApi } from '@/api/ci';
	import AppDialog from '@/components/AppDialog.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import ListPagination from '@/components/ListPagination.vue';
	import SearchControl from '@/components/SearchControl.vue';
	import SelectControl from '@/components/SelectControl.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import type { Credential, CredentialImportReq } from '@/types/ci/credential';
	import { credentialTypeLabels } from '@/types/ci/credential';
	import { formatTime } from '@/utils/time';
	import { ToolbarRoot } from 'reka-ui';

	const toast = useToast();
	const { status, error, execute } = useStatusAsync();
	const { loading: operating, execute: executeOp } = useStatusAsync();

	const credentials = ref<Credential[]>([]);
	const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
	const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
	const searchText = ref('');

	const fileInput = ref<HTMLInputElement>();
	const showCredentialDialog = ref(false);
	const showDeleteDialog = ref(false);
	const showImportDialog = ref(false);
	const isEditing = ref(false);
	const currentId = ref('');
	const pendingDeleteId = ref('');

	const form = reactive({ name: '', type: 'git_ssh' as string, data: '' });
	const credentialTypeOptions = [
		{ value: 'git_ssh', label: 'Git SSH 密钥' },
		{ value: 'git_token', label: 'Git Token' },
		{ value: 'gitee_token', label: 'Gitee Token' },
	];
	const errors = reactive({ name: '', data: '' });
	const importForm = reactive({ name: '', type: 'git_ssh' as string, data: '' });
	const importErrors = reactive({ name: '', data: '' });

	function validate() {
		errors.name = form.name.trim() ? '' : '请输入凭据名称';
		errors.data = !isEditing.value && !form.data.trim() ? '请输入凭据内容' : '';
		return !errors.name && !errors.data;
	}

	async function fetchCredentials() {
		try {
			await execute(async () => {
				const res = await credentialApi.list({
					page: pagination.current,
					per_page: pagination.pageSize,
					search: searchText.value || undefined,
				});
				credentials.value = res.items;
				pagination.total = res.total;
			});
		} catch {
			toast.error('获取凭据列表失败');
		}
	}

	function handleSearch() {
		pagination.current = 1;
		fetchCredentials();
	}

	function goPage(p: number) {
		pagination.current = p;
		fetchCredentials();
	}

	function handlePageSizeChange(pageSize: number) {
		pagination.pageSize = pageSize;
		pagination.current = 1;
		fetchCredentials();
	}

	function openCreateModal() {
		isEditing.value = false;
		currentId.value = '';
		Object.assign(form, { name: '', type: 'git_ssh', data: '' });
		Object.assign(errors, { name: '', data: '' });
		showCredentialDialog.value = true;
	}

	function openEditModal(record: Credential) {
		isEditing.value = true;
		currentId.value = record.id;
		Object.assign(form, { name: record.name, type: record.type, data: '' });
		Object.assign(errors, { name: '', data: '' });
		showCredentialDialog.value = true;
	}

	async function handleModalOk() {
		if (!validate()) {
			return;
		}
		try {
			await executeOp(async () => {
				if (isEditing.value) {
					await credentialApi.update(currentId.value, {
						name: form.name,
						...(form.data ? { data: form.data } : {}),
					});
					toast.success('更新成功');
				} else {
					await credentialApi.create({
						name: form.name,
						type: form.type,
						data: form.data,
					});
					toast.success('创建成功');
				}
				showCredentialDialog.value = false;
				fetchCredentials();
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '操作失败');
		}
	}

	function confirmDelete(id: string) {
		pendingDeleteId.value = id;
		showDeleteDialog.value = true;
	}

	async function handleDelete() {
		try {
			await executeOp(async () => {
				await credentialApi.delete(pendingDeleteId.value);
				toast.success('删除成功');
				showDeleteDialog.value = false;
				fetchCredentials();
			});
		} catch (error) {
			toast.error(error instanceof Error ? error.message : '删除失败');
		}
	}

	function getDataPlaceholder(type: string) {
		if (type === 'git_ssh') {
			return '-----BEGIN OPENSSH PRIVATE KEY-----\n...';
		}
		if (type === 'git_token') {
			return 'ghp_xxxxxxxxxxxxxxxxxxxx';
		}
		if (type === 'gitee_token') {
			return 'your_username:your_gitee_token';
		}
		return 'registry_token_here';
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
			const data = JSON.parse(await file.text()) as CredentialImportReq;
			Object.assign(importForm, {
				name: data.name || '',
				type: data.type || 'git_ssh',
				data: data.data || '',
			});
			Object.assign(importErrors, { name: '', data: '' });
			showImportDialog.value = true;
		} catch {
			toast.error('解析文件失败');
		} finally {
			target.value = '';
		}
	}

	async function handleImportOk() {
		importErrors.name = importForm.name.trim() ? '' : '请输入凭据名称';
		importErrors.data = importForm.data.trim() ? '' : '请输入凭据内容';
		if (importErrors.name || importErrors.data) {
			return;
		}
		try {
			await executeOp(async () => {
				await credentialApi.importCredential({
					name: importForm.name,
					type: importForm.type as CredentialImportReq['type'],
					data: importForm.data,
				});
				toast.success('导入成功');
				showImportDialog.value = false;
				fetchCredentials();
			});
		} catch (err) {
			toast.error(err instanceof Error ? err.message : '导入失败');
		}
	}

	onMounted(fetchCredentials);
</script>
