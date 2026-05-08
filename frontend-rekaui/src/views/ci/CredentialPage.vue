<script setup lang="ts">
import { Plus, Upload } from 'lucide-vue-next'
import { computed, onMounted, reactive, ref } from 'vue'
import { credentialApi } from '@/api/ci'
import ListPagination from '@/components/ListPagination.vue'
import SearchControl from '@/components/SearchControl.vue'
import SelectControl from '@/components/SelectControl.vue'
import { useStatusAsync } from '@/composables/useStatusAsync'
import { useToast } from '@/composables/useToast'
import type { Credential, CredentialImportReq } from '@/types/ci/credential'
import { credentialTypeLabels } from '@/types/ci/credential'
import { formatTime } from '@/utils/time'
import {
	DialogClose,
	DialogContent,
	DialogDescription,
	DialogOverlay,
	DialogPortal,
	DialogRoot,
	DialogTitle,
	ToolbarRoot
} from 'reka-ui'

const toast = useToast()
const { status, error, execute } = useStatusAsync()
const { loading: operating, execute: executeOp } = useStatusAsync()

const credentials = ref<Credential[]>([])
const pagination = reactive({ current: 1, pageSize: 10, total: 0 })
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize))
const searchText = ref('')

const fileInput = ref<HTMLInputElement>()
const showCredentialDialog = ref(false)
const showDeleteDialog = ref(false)
const showImportDialog = ref(false)
const isEditing = ref(false)
const currentId = ref('')
const pendingDeleteId = ref('')

const form = reactive({ name: '', type: 'git_ssh' as string, data: '' })
const credentialTypeOptions = [
	{ value: 'git_ssh', label: 'Git SSH 密钥' },
	{ value: 'git_token', label: 'Git Token' },
	{ value: 'gitee_token', label: 'Gitee Token' },
]
const errors = reactive({ name: '', data: '' })
const importForm = reactive({ name: '', type: 'git_ssh' as string, data: '' })
const importErrors = reactive({ name: '', data: '' })

function validate() {
	errors.name = form.name.trim() ? '' : '请输入凭据名称'
	errors.data = !isEditing.value && !form.data.trim() ? '请输入凭据内容' : ''
	return !errors.name && !errors.data
}

async function fetchCredentials() {
	try {
		await execute(async () => {
			const res = await credentialApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				search: searchText.value || undefined
			})
			credentials.value = res.items
			pagination.total = res.total
		})
	} catch {
		toast.error('获取凭据列表失败')
	}
}

function handleSearch() {
	pagination.current = 1
	fetchCredentials()
}

function goPage(p: number) {
	pagination.current = p
	fetchCredentials()
}

function handlePageSizeChange(pageSize: number) {
	pagination.pageSize = pageSize
	pagination.current = 1
	fetchCredentials()
}

function openCreateModal() {
	isEditing.value = false
	currentId.value = ''
	Object.assign(form, { name: '', type: 'git_ssh', data: '' })
	Object.assign(errors, { name: '', data: '' })
	showCredentialDialog.value = true
}

function openEditModal(record: Credential) {
	isEditing.value = true
	currentId.value = record.id
	Object.assign(form, { name: record.name, type: record.type, data: '' })
	Object.assign(errors, { name: '', data: '' })
	showCredentialDialog.value = true
}

async function handleModalOk() {
	if (!validate()) {
		return
	}
	try {
		await executeOp(async () => {
			if (isEditing.value) {
				await credentialApi.update(currentId.value, {
					name: form.name,
					...(form.data ? { data: form.data } : {})
				})
				toast.success('更新成功')
			} else {
				await credentialApi.create({
					name: form.name,
					type: form.type,
					data: form.data
				})
				toast.success('创建成功')
			}
			showCredentialDialog.value = false
			fetchCredentials()
		})
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '操作失败')
	}
}

function confirmDelete(id: string) {
	pendingDeleteId.value = id
	showDeleteDialog.value = true
}

async function handleDelete() {
	try {
		await executeOp(async () => {
			await credentialApi.delete(pendingDeleteId.value)
			toast.success('删除成功')
			showDeleteDialog.value = false
			fetchCredentials()
		})
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '删除失败')
	}
}

function getDataPlaceholder(type: string) {
	if (type === 'git_ssh') {
		return '-----BEGIN OPENSSH PRIVATE KEY-----\n...'
	}
	if (type === 'git_token') {
		return 'ghp_xxxxxxxxxxxxxxxxxxxx'
	}
	if (type === 'gitee_token') {
		return 'your_username:your_gitee_token'
	}
	return 'registry_token_here'
}

function triggerImport() {
	fileInput.value?.click()
}

async function handleFileImport(event: Event) {
	const target = event.target as HTMLInputElement
	const file = target.files?.[0]
	if (!file) {
		return
	}
	try {
		const data = JSON.parse(await file.text()) as CredentialImportReq
		Object.assign(importForm, {
			name: data.name || '',
			type: data.type || 'git_ssh',
			data: data.data || ''
		})
		Object.assign(importErrors, { name: '', data: '' })
		showImportDialog.value = true
	} catch {
		toast.error('解析文件失败')
	} finally {
		target.value = ''
	}
}

async function handleImportOk() {
	importErrors.name = importForm.name.trim() ? '' : '请输入凭据名称'
	importErrors.data = importForm.data.trim() ? '' : '请输入凭据内容'
	if (importErrors.name || importErrors.data) {
		return
	}
	try {
		await executeOp(async () => {
			await credentialApi.importCredential({
				name: importForm.name,
				type: importForm.type as CredentialImportReq['type'],
				data: importForm.data
			})
			toast.success('导入成功')
			showImportDialog.value = false
			fetchCredentials()
		})
	} catch (err) {
		toast.error(err instanceof Error ? err.message : '导入失败')
	}
}

onMounted(fetchCredentials)
</script>

<template>
	<div class="space-y-6">
		<ToolbarRoot class="flex items-center justify-between gap-3" aria-label="凭据工具栏">
			<SearchControl
				v-model="searchText"
				placeholder="搜索凭据名称"
				:loading="status === 'loading'"
				@search="handleSearch"
			/>
			<div class="flex items-center gap-3">
				<button
					class="flex h-10 items-center gap-2 rounded-md border border-border bg-background px-5 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="triggerImport"
				>
					<Upload class="size-4" />
					导入
				</button>
				<button
					class="flex h-10 items-center gap-2 rounded-md bg-primary px-5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
					@click="openCreateModal"
				>
					<Plus class="size-4" />
					新建凭据
				</button>
			</div>
		</ToolbarRoot>

		<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div v-if="status === 'loading'" class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error || '加载失败' }}</p>
			</div>
			<div v-else-if="credentials.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="w-full">
					<thead class="border-b border-border bg-muted/30">
						<tr>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">名称</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">类型</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">创建时间</th>
							<th class="px-6 py-4 text-right text-xs font-normal text-muted-foreground">操作</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border">
						<tr v-for="cred in credentials" :key="cred.id" class="transition-colors hover:bg-muted/30">
							<td class="px-6 py-5 text-sm text-foreground">{{ cred.name }}</td>
							<td class="px-6 py-5 text-sm">
								<span class="inline-flex rounded-md border border-blue-200 bg-blue-50 px-2 py-0.5 text-sm text-blue-700">
									{{ credentialTypeLabels[cred.type] || cred.type }}
								</span>
							</td>
							<td class="px-6 py-5 text-sm text-foreground">{{ formatTime(cred.created_at) }}</td>
							<td class="px-6 py-5 text-right text-sm">
								<button class="mr-3 text-primary hover:underline" @click="openEditModal(cred)">
									编辑
								</button>
								<button class="text-destructive hover:underline" @click="confirmDelete(cred.id)">
									删除
								</button>
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

		<input ref="fileInput" type="file" accept=".json" class="hidden" @change="handleFileImport" />
	</div>

	<DialogRoot v-model:open="showCredentialDialog">
		<DialogPortal>
			<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
			<DialogContent
				class="fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[min(600px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
			>
				<div class="mb-5 space-y-1">
					<DialogTitle class="text-lg font-semibold text-foreground">
						{{ isEditing ? '编辑凭据' : '新建凭据' }}
					</DialogTitle>
					<DialogDescription class="text-sm text-muted-foreground">
						{{ isEditing ? '更新凭据名称，凭据内容留空时不会修改。' : '创建可用于 Git 或镜像仓库访问的凭据。' }}
					</DialogDescription>
				</div>

				<div class="space-y-4">
					<div class="space-y-1.5">
						<label class="block text-sm font-medium text-foreground">凭据名称</label>
						<input
							v-model="form.name"
							type="text"
							placeholder="输入凭据名称"
							class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
							:class="errors.name ? 'border-destructive' : 'border-input'"
						/>
						<p v-if="errors.name" class="text-xs text-destructive">{{ errors.name }}</p>
					</div>
					<div class="space-y-1.5">
						<label class="block text-sm font-medium text-foreground">凭据类型</label>
						<SelectControl
							v-model="form.type"
							:options="credentialTypeOptions"
							:disabled="isEditing"
							placeholder="选择凭据类型"
						/>
					</div>
					<div class="space-y-1.5">
						<label class="block text-sm font-medium text-foreground">
							凭据内容 {{ isEditing ? '（留空表示不修改）' : '' }}
						</label>
						<textarea
							v-model="form.data"
							rows="8"
							:placeholder="getDataPlaceholder(form.type)"
							class="w-full rounded-md border bg-background px-3 py-2 font-mono text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
							:class="errors.data ? 'border-destructive' : 'border-input'"
						/>
						<p v-if="errors.data" class="text-xs text-destructive">{{ errors.data }}</p>
					</div>
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
						@click="handleModalOk"
					>
						<span
							v-if="operating"
							class="size-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent"
						/>
						保存
					</button>
				</div>
			</DialogContent>
		</DialogPortal>
	</DialogRoot>

	<DialogRoot v-model:open="showDeleteDialog">
		<DialogPortal>
			<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
			<DialogContent
				class="fixed left-1/2 top-1/2 z-50 w-[min(420px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
			>
				<div class="space-y-2">
					<DialogTitle class="text-lg font-semibold text-foreground">确认删除</DialogTitle>
					<DialogDescription class="text-sm text-muted-foreground">
						确定要删除这个凭据吗？此操作不可恢复。
					</DialogDescription>
				</div>
				<div class="mt-6 flex justify-end gap-2">
					<DialogClose as-child>
						<button class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50">
							取消
						</button>
					</DialogClose>
					<button
						class="rounded-md bg-destructive px-4 py-2 text-sm font-medium text-destructive-foreground transition-colors hover:bg-destructive/90 disabled:cursor-not-allowed disabled:opacity-50"
						:disabled="operating"
						@click="handleDelete"
					>
						删除
					</button>
				</div>
			</DialogContent>
		</DialogPortal>
	</DialogRoot>

	<DialogRoot v-model:open="showImportDialog">
		<DialogPortal>
			<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
			<DialogContent
				class="fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[min(600px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
			>
				<div class="mb-5 space-y-1">
					<DialogTitle class="text-lg font-semibold text-foreground">导入凭据</DialogTitle>
					<DialogDescription class="text-sm text-muted-foreground">
						确认导入文件中的凭据信息。
					</DialogDescription>
				</div>
				<div class="space-y-4">
					<div class="space-y-1.5">
						<label class="block text-sm font-medium text-foreground">凭据名称</label>
						<input
							v-model="importForm.name"
							type="text"
							placeholder="输入凭据名称"
							class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
							:class="importErrors.name ? 'border-destructive' : 'border-input'"
						/>
						<p v-if="importErrors.name" class="text-xs text-destructive">{{ importErrors.name }}</p>
					</div>
					<div class="space-y-1.5">
						<label class="block text-sm font-medium text-foreground">凭据类型</label>
						<SelectControl
							v-model="importForm.type"
							:options="credentialTypeOptions"
							placeholder="选择凭据类型"
						/>
					</div>
					<div class="space-y-1.5">
						<label class="block text-sm font-medium text-foreground">凭据内容</label>
						<textarea
							v-model="importForm.data"
							rows="8"
							class="w-full rounded-md border bg-background px-3 py-2 font-mono text-sm text-foreground outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
							:class="importErrors.data ? 'border-destructive' : 'border-input'"
						/>
						<p v-if="importErrors.data" class="text-xs text-destructive">{{ importErrors.data }}</p>
					</div>
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
						@click="handleImportOk"
					>
						<span
							v-if="operating"
							class="size-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent"
						/>
						导入
					</button>
				</div>
			</DialogContent>
		</DialogPortal>
	</DialogRoot>
</template>
