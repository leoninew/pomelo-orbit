<script setup lang="ts">
import { Play, Plus, Trash2 } from 'lucide-vue-next'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { credentialApi, pipelineTemplateApi, repositoryApi, webhookApi } from '@/api/ci'
import ComboboxSelect from '@/components/ComboboxSelect.vue'
import { useStatusAsync } from '@/composables/useStatusAsync'
import { useToast } from '@/composables/useToast'
import type { Credential } from '@/types/ci/credential'
import { credentialTypeLabels } from '@/types/ci/credential'
import type { Repository } from '@/types/ci/repository'
import type { PipelineTemplate, VariableDeclaration } from '@/types/ci/template'
import type { RepositoryWebhook } from '@/types/ci/webhook'
import { formatTime } from '@/utils/time'
import {
	DialogClose,
	DialogContent,
	DialogDescription,
	DialogOverlay,
	DialogPortal,
	DialogRoot,
	DialogTitle
} from 'reka-ui'
import TriggerModal from './components/TriggerModal.vue'
import VariableDeclarationsTable from './components/VariableDeclarationsTable.vue'
import WebhookList from './components/WebhookList.vue'

const route = useRoute()
const router = useRouter()
const repositoryId = route.params.id as string
const toast = useToast()

const { status, execute } = useStatusAsync()
const { loading: operating, execute: executeOp } = useStatusAsync()

const repository = ref<Repository>()
const templates = ref<PipelineTemplate[]>([])
const webhooks = ref<RepositoryWebhook[]>([])
const credentials = ref<Credential[]>([])

const isEditDialogOpen = ref(false)
const isDeleteDialogOpen = ref(false)
const isAddVariableDialogOpen = ref(false)
const isEditVariableDialogOpen = ref(false)
const triggerModalRef = ref<InstanceType<typeof TriggerModal>>()

const editForm = reactive({
	name: '',
	repository_url: '',
	git_credential_id: '',
	default_branch: 'master'
})
const editErrors = reactive({
	name: '',
	repository_url: ''
})
const variableForm = reactive({
	name: '',
	value: '',
	description: ''
})
const editingVariableName = ref('')

const gitCredentials = computed(() =>
	credentials.value.filter(
		(credential) =>
			credential.type === 'git_ssh' ||
			credential.type === 'git_token' ||
			credential.type === 'gitee_token'
	)
)
const gitCredentialOptions = computed(() =>
	gitCredentials.value.map((credential) => ({
		value: credential.id,
		label: credential.name,
		description: credentialTypeLabels[credential.type] ?? credential.type
	}))
)
const repositoryVariables = computed(() => repository.value?.variable_declarations ?? [])
const repositoryCustomVariables = computed(() =>
	repositoryVariables.value.filter((variable) => variable.source === 'repository_custom')
)

function normalizeValue(value: unknown) {
	return value == null ? '' : String(value)
}

function resetEditForm() {
	if (!repository.value) {
		return
	}
	Object.assign(editForm, {
		name: repository.value.name,
		repository_url: repository.value.repository_url,
		git_credential_id: repository.value.git_credential_id ?? '',
		default_branch: repository.value.default_branch || 'master'
	})
	Object.assign(editErrors, { name: '', repository_url: '' })
}

function validateEditForm() {
	editErrors.name = editForm.name.trim() ? '' : '请输入名称'
	editErrors.repository_url = editForm.repository_url.trim() ? '' : '请输入仓库地址'
	return !editErrors.name && !editErrors.repository_url
}

function variableOverridesWith(nextVariable?: VariableDeclaration) {
	const next = repositoryCustomVariables.value.filter(
		(variable) => variable.name !== nextVariable?.name
	)
	return nextVariable ? [...next, nextVariable] : next
}

async function fetchRepository() {
	try {
		await execute(async () => {
			repository.value = await repositoryApi.get(repositoryId)
			resetEditForm()
		})
	} catch {
		toast.error('获取代码仓库信息失败')
		router.push('/ci/repository')
	}
}

async function fetchTemplates() {
	try {
		const res = await pipelineTemplateApi.list({ per_page: 100 })
		templates.value = res.items
	} catch {
		toast.error('获取模板列表失败')
	}
}

async function fetchWebhooks() {
	try {
		webhooks.value = await webhookApi.list(repositoryId)
	} catch {
		toast.error('获取 Webhook 列表失败')
	}
}

async function fetchCredentials() {
	try {
		const res = await credentialApi.list({ per_page: 100 })
		credentials.value = res.items
	} catch {
		toast.error('获取凭据列表失败')
	}
}

async function openTriggerModal() {
	if (!repository.value?.git_credential_id) {
		toast.error('请先配置 Git 凭据后再触发流水线')
		return
	}
	await fetchTemplates()
	triggerModalRef.value?.open()
}

async function handleTrigger(data: {
	template_id: string
	trigger_ref: string
	variables: Record<string, string>
}) {
	try {
		await executeOp(async () => {
			const run = await repositoryApi.trigger(repositoryId, data)
			toast.success('触发成功')
			router.push(`/ci/run/${run.id}`)
		})
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '触发失败')
	}
}

async function openEditDialog() {
	resetEditForm()
	await fetchCredentials()
	isEditDialogOpen.value = true
}

async function handleEditOk() {
	if (!validateEditForm()) {
		return
	}
	try {
		await executeOp(async () => {
			const updated = await repositoryApi.update(repositoryId, {
				name: editForm.name,
				repository_url: editForm.repository_url,
				git_credential_id: editForm.git_credential_id || null,
				default_branch: editForm.default_branch || 'master'
			})
			repository.value = updated
			resetEditForm()
			toast.success('更新成功')
			isEditDialogOpen.value = false
		})
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '更新失败')
	}
}

function openDeleteDialog() {
	isDeleteDialogOpen.value = true
}

async function handleDeleteOk() {
	try {
		await executeOp(async () => {
			await repositoryApi.delete(repositoryId)
			toast.success('删除成功')
			router.push('/ci/repository')
		})
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '删除失败')
	}
}

function openAddVariableDialog() {
	Object.assign(variableForm, { name: '', value: '', description: '' })
	isAddVariableDialogOpen.value = true
}

function openEditVariableDialog(name: string) {
	const variable = repositoryCustomVariables.value.find((item) => item.name === name)
	if (!variable) {
		return
	}
	editingVariableName.value = variable.name
	Object.assign(variableForm, {
		name: variable.name,
		value: normalizeValue(variable.value ?? variable.default),
		description: variable.description ?? ''
	})
	isEditVariableDialogOpen.value = true
}

async function handleAddVariableOk() {
	if (!variableForm.name.trim()) {
		toast.error('请输入变量名')
		return
	}
	if (repositoryVariables.value.some((variable) => variable.name === variableForm.name.trim())) {
		toast.error('变量名已存在')
		return
	}
	try {
		await executeOp(async () => {
			const nextVariable: VariableDeclaration = {
				name: variableForm.name.trim(),
				value: variableForm.value,
				description: variableForm.description.trim() || undefined,
				secret: false,
				source: 'repository_custom'
			}
			const updated = await repositoryApi.update(repositoryId, {
				variable_overrides: variableOverridesWith(nextVariable)
			})
			repository.value = updated
			toast.success('添加成功')
			isAddVariableDialogOpen.value = false
		})
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '添加失败')
	}
}

async function handleEditVariableOk() {
	if (!editingVariableName.value) {
		return
	}
	try {
		await executeOp(async () => {
			const current = repositoryCustomVariables.value.find(
				(variable) => variable.name === editingVariableName.value
			)
			if (!current) {
				return
			}
			const updated = await repositoryApi.update(repositoryId, {
				variable_overrides: variableOverridesWith({
					...current,
					value: variableForm.value,
					description: variableForm.description.trim() || undefined
				})
			})
			repository.value = updated
			toast.success('更新成功')
			isEditVariableDialogOpen.value = false
		})
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '更新失败')
	}
}

async function deleteVariable(name: string) {
	try {
		await executeOp(async () => {
			const updated = await repositoryApi.update(repositoryId, {
				variable_overrides: repositoryCustomVariables.value.filter(
					(variable) => variable.name !== name
				)
			})
			repository.value = updated
			toast.success('删除成功')
		})
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '删除失败')
	}
}

onMounted(async () => {
	await fetchRepository()
	await Promise.all([fetchTemplates(), fetchWebhooks()])
})
</script>

<template>
	<div class="flex flex-col gap-4">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<div class="flex items-center gap-3">
				<div>
					<h1 class="text-xl font-semibold text-foreground">
						{{ repository?.name ?? '仓库详情' }}
					</h1>
				</div>
			</div>
			<div class="flex flex-wrap items-center gap-2">
				<button
					v-if="repository"
					class="inline-flex h-9 items-center gap-2 rounded-md bg-primary px-3 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="operating"
					@click="openTriggerModal"
				>
					<Play class="size-4" />
					触发
				</button>
				<button
					v-if="repository"
					class="h-9 rounded-md border border-input bg-background px-3 text-sm font-medium text-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="operating"
					@click="openEditDialog"
				>
					编辑
				</button>
				<button
					v-if="repository"
					class="inline-flex h-9 items-center gap-2 rounded-md border border-destructive/50 bg-background px-3 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="operating"
					@click="openDeleteDialog"
				>
					<Trash2 class="size-4" />
					删除
				</button>
				<button
					class="h-9 rounded-md border border-input bg-background px-4 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="router.push('/ci/repository')"
				>
					返回
				</button>
			</div>
		</div>

		<div v-if="status === 'loading'" class="flex justify-center py-12">
			<span class="inline-block size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
		</div>

		<template v-else-if="repository">
			<div class="rounded-lg border border-border bg-card shadow-sm">
				<div class="flex items-center justify-between border-b border-border px-5 py-4">
					<h2 class="font-semibold text-foreground">基本信息</h2>
					<router-link
						:to="`/ci/run?repository_id=${repository.id}`"
						class="text-sm font-medium text-primary hover:underline"
					>
						查看流水线记录
					</router-link>
				</div>
				<dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">名称</dt>
						<dd class="min-w-0 text-foreground">{{ repository.name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">编码</dt>
						<dd class="min-w-0 text-foreground">{{ repository.code }}</dd>
					</div>
					<div class="flex gap-2 sm:col-span-2">
						<dt class="w-24 shrink-0 text-muted-foreground">仓库地址</dt>
						<dd class="min-w-0 truncate text-foreground" :title="repository.repository_url">
							{{ repository.repository_url }}
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">默认分支</dt>
						<dd class="text-foreground">{{ repository.default_branch || 'master' }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">Git 凭据</dt>
						<dd>
							<router-link
								v-if="repository.git_credential_id"
								:to="`/ci/credential/${repository.git_credential_id}`"
								class="text-primary hover:underline"
							>
								{{ repository.git_credential_name || repository.git_credential_id }}
							</router-link>
							<span v-else class="text-muted-foreground">未配置</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">创建时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(repository.created_at) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">更新时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(repository.updated_at) }}</dd>
					</div>
				</dl>
			</div>

			<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
				<div class="flex flex-wrap items-center justify-between gap-3 border-b border-border px-5 py-4">
					<h2 class="font-semibold text-foreground">变量配置</h2>
					<button
						class="inline-flex h-8 items-center gap-1.5 rounded-md bg-primary px-3 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
						@click="openAddVariableDialog"
					>
						<Plus class="size-4" />
						添加自定义变量
					</button>
				</div>
				<VariableDeclarationsTable
					:declarations="repositoryVariables"
					:readonly="false"
					@edit="openEditVariableDialog"
					@delete="deleteVariable"
				/>
			</div>

			<WebhookList
				:repository-id="repositoryId"
				:webhooks="webhooks"
				:templates="templates"
				@refresh="fetchWebhooks"
			/>
		</template>

		<TriggerModal
			ref="triggerModalRef"
			:repository-id="repositoryId"
			:templates="templates"
			:default-branch="repository?.default_branch"
			:project-variables="repositoryCustomVariables"
			:repository="repository"
			@trigger="handleTrigger"
		/>

		<DialogRoot v-model:open="isEditDialogOpen">
			<DialogPortal>
				<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
				<DialogContent
					class="fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[min(520px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
				>
					<DialogTitle class="text-lg font-semibold text-foreground">编辑仓库</DialogTitle>
					<DialogDescription class="mt-1 text-sm text-muted-foreground">
						更新仓库地址、默认分支和 Git 凭据配置。
					</DialogDescription>

					<div class="mt-5 space-y-4">
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">名称</label>
							<input
								v-model="editForm.name"
								type="text"
								class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
								:class="editErrors.name ? 'border-destructive' : 'border-input'"
							/>
							<p v-if="editErrors.name" class="text-xs text-destructive">{{ editErrors.name }}</p>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">编码</label>
							<input
								:value="repository?.code"
								type="text"
								disabled
								class="w-full cursor-not-allowed rounded-md border border-input bg-muted/40 px-3 py-2 text-sm text-muted-foreground outline-none"
							/>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">仓库地址</label>
							<input
								v-model="editForm.repository_url"
								type="text"
								class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
								:class="editErrors.repository_url ? 'border-destructive' : 'border-input'"
							/>
							<p v-if="editErrors.repository_url" class="text-xs text-destructive">
								{{ editErrors.repository_url }}
							</p>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">Git 凭据</label>
							<ComboboxSelect
								v-model="editForm.git_credential_id"
								:options="gitCredentialOptions"
								placeholder="不使用凭据"
							/>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">默认分支</label>
							<input
								v-model="editForm.default_branch"
								type="text"
								placeholder="master"
								class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
							/>
						</div>
					</div>

					<div class="mt-6 flex justify-end gap-2">
						<DialogClose as-child>
							<button
								class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
							>
								取消
							</button>
						</DialogClose>
						<button
							class="inline-flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
							:disabled="operating"
							@click="handleEditOk"
						>
							<span
								v-if="operating"
								class="size-4 animate-spin rounded-full border-2 border-primary-foreground/30 border-t-primary-foreground"
							/>
							保存
						</button>
					</div>
				</DialogContent>
			</DialogPortal>
		</DialogRoot>

		<DialogRoot v-model:open="isDeleteDialogOpen">
			<DialogPortal>
				<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
				<DialogContent
					class="fixed left-1/2 top-1/2 z-50 w-[min(420px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
				>
					<DialogTitle class="text-lg font-semibold text-foreground">删除仓库</DialogTitle>
					<DialogDescription class="mt-2 text-sm text-muted-foreground">
						确定要删除仓库「{{ repository?.name }}」吗？此操作不可撤销。
					</DialogDescription>
					<div class="mt-6 flex justify-end gap-2">
						<DialogClose as-child>
							<button
								class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
							>
								取消
							</button>
						</DialogClose>
						<button
							class="inline-flex items-center gap-2 rounded-md bg-destructive px-4 py-2 text-sm font-medium text-destructive-foreground transition-colors hover:bg-destructive/90 disabled:cursor-not-allowed disabled:opacity-50"
							:disabled="operating"
							@click="handleDeleteOk"
						>
							<span
								v-if="operating"
								class="size-4 animate-spin rounded-full border-2 border-destructive-foreground/30 border-t-destructive-foreground"
							/>
							删除
						</button>
					</div>
				</DialogContent>
			</DialogPortal>
		</DialogRoot>

		<DialogRoot v-model:open="isAddVariableDialogOpen">
			<DialogPortal>
				<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
				<DialogContent
					class="fixed left-1/2 top-1/2 z-50 w-[min(480px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
				>
					<DialogTitle class="text-lg font-semibold text-foreground">添加变量</DialogTitle>
					<DialogDescription class="sr-only">添加仓库自定义变量</DialogDescription>
					<div class="mt-5 space-y-4">
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">变量名</label>
							<input
								v-model="variableForm.name"
								type="text"
								placeholder="例如: DEPLOY_ENV"
								class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
							/>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">变量值</label>
							<input
								v-model="variableForm.value"
								type="text"
								class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
							/>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">说明</label>
							<input
								v-model="variableForm.description"
								type="text"
								placeholder="可选"
								class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
							/>
						</div>
					</div>
					<div class="mt-6 flex justify-end gap-2">
						<DialogClose as-child>
							<button
								class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
							>
								取消
							</button>
						</DialogClose>
						<button
							class="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
							:disabled="operating"
							@click="handleAddVariableOk"
						>
							保存
						</button>
					</div>
				</DialogContent>
			</DialogPortal>
		</DialogRoot>

		<DialogRoot v-model:open="isEditVariableDialogOpen">
			<DialogPortal>
				<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
				<DialogContent
					class="fixed left-1/2 top-1/2 z-50 w-[min(480px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
				>
					<DialogTitle class="text-lg font-semibold text-foreground">编辑变量</DialogTitle>
					<DialogDescription class="sr-only">编辑仓库自定义变量</DialogDescription>
					<div class="mt-5 space-y-4">
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">变量名</label>
							<input
								:value="editingVariableName"
								type="text"
								disabled
								class="w-full cursor-not-allowed rounded-md border border-input bg-muted/40 px-3 py-2 text-sm text-muted-foreground outline-none"
							/>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">变量值</label>
							<input
								v-model="variableForm.value"
								type="text"
								class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
							/>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">说明</label>
							<input
								v-model="variableForm.description"
								type="text"
								placeholder="可选"
								class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
							/>
						</div>
					</div>
					<div class="mt-6 flex justify-end gap-2">
						<DialogClose as-child>
							<button
								class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
							>
								取消
							</button>
						</DialogClose>
						<button
							class="rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
							:disabled="operating"
							@click="handleEditVariableOk"
						>
							保存
						</button>
					</div>
				</DialogContent>
			</DialogPortal>
		</DialogRoot>
	</div>
</template>
