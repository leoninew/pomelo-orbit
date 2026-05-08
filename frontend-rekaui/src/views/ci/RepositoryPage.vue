<script setup lang="ts">
import { ChevronLeft, ChevronRight, Plus } from 'lucide-vue-next'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { credentialApi, repositoryApi } from '@/api/ci'
import ComboboxSelect from '@/components/ComboboxSelect.vue'
import SearchControl from '@/components/SearchControl.vue'
import SelectControl from '@/components/SelectControl.vue'
import { useStatusAsync } from '@/composables/useStatusAsync'
import { useToast } from '@/composables/useToast'
import type { Credential } from '@/types/ci/credential'
import type { RepositoryListItem } from '@/types/ci/repository'
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

const router = useRouter()
const toast = useToast()
const { status, execute } = useStatusAsync()
const { loading: operating, execute: executeOp } = useStatusAsync()
const { status: modalStatus, execute: executeModal } = useStatusAsync()

const repositories = ref<RepositoryListItem[]>([])
const credentials = ref<Credential[]>([])
const gitCredentials = computed(() =>
	credentials.value.filter(
		(c) => c.type === 'git_ssh' || c.type === 'git_token' || c.type === 'gitee_token'
	)
)
const gitCredentialOptions = computed(() =>
	gitCredentials.value.map((cred) => ({
		value: cred.id,
		label: cred.name
	}))
)
const pageSizeOptions = [
	{ value: 10, label: '10 条/页' },
	{ value: 20, label: '20 条/页' },
	{ value: 50, label: '50 条/页' }
]

const pagination = reactive({ current: 1, pageSize: 10, total: 0 })
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize))
const searchText = ref('')
const showCreateModal = ref(false)

const form = reactive({
	name: '',
	code: '',
	repository_url: '',
	git_credential_id: ''
})
const errors = reactive({
	name: '',
	code: '',
	repository_url: ''
})

function validate() {
	errors.name = form.name.trim() ? '' : '请输入名称'
	errors.code = /^[a-z0-9-]+$/.test(form.code.trim()) ? '' : '编码只能包含小写字母、数字和连字符'
	errors.repository_url = form.repository_url.trim() ? '' : '请输入仓库地址'
	return !errors.name && !errors.code && !errors.repository_url
}

async function fetchProjects() {
	try {
		await execute(async () => {
			const res = await repositoryApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				search: searchText.value || undefined
			})
			repositories.value = res.items
			pagination.total = res.total
		})
	} catch {
		toast.error('获取项目列表失败')
	}
}

function handleSearch() {
	pagination.current = 1
	fetchProjects()
}

function goPage(p: number) {
	if (p < 1 || p > totalPages.value || p === pagination.current) {
		return
	}
	pagination.current = p
	fetchProjects()
}

function handlePageSizeChange(pageSize: number) {
	pagination.pageSize = pageSize
	pagination.current = 1
	fetchProjects()
}

async function openCreateModal() {
	Object.assign(form, {
		name: '',
		code: '',
		repository_url: '',
		git_credential_id: ''
	})
	Object.assign(errors, { name: '', code: '', repository_url: '' })
	showCreateModal.value = true
	try {
		await executeModal(async () => {
			const credRes = await credentialApi.list({ per_page: 100 })
			credentials.value = credRes.items
		})
	} catch {
		toast.error('加载表单数据失败')
	}
}

async function handleCreateOk() {
	if (!validate()) {
		return
	}
	try {
		await executeOp(async () => {
			const repository = await repositoryApi.create({
				name: form.name,
				code: form.code,
				repository_url: form.repository_url,
				git_credential_id: form.git_credential_id || undefined
			})
			toast.success('创建成功')
			showCreateModal.value = false
			router.push(`/ci/repository/${repository.id}`)
		})
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '创建失败')
	}
}

onMounted(fetchProjects)
</script>

<template>
	<div class="space-y-6">
		<ToolbarRoot class="flex items-center justify-between gap-6" aria-label="仓库工具栏">
			<SearchControl
				v-model="searchText"
				placeholder="搜索名称/地址"
				:loading="status === 'loading'"
				@search="handleSearch"
			/>
			<button
				class="flex h-10 items-center gap-2 rounded-md bg-primary px-5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
				@click="openCreateModal"
			>
				<Plus class="size-4" />
				新建仓库
			</button>
		</ToolbarRoot>

		<!-- Table Card -->
		<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div v-if="status === 'loading'" class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
			<div v-else-if="repositories.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[1120px]">
					<thead>
						<tr>
							<th>名称</th>
							<th>编码</th>
							<th>默认分支</th>
							<th>地址</th>
							<th>Git 凭据</th>
							<th>创建时间</th>
							<th>操作</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="p in repositories" :key="p.id">
							<td>
								<button
									class="text-primary hover:underline"
									@click="router.push(`/ci/repository/${p.id}`)"
								>
									{{ p.name }}
								</button>
							</td>
							<td class="text-foreground">{{ p.code }}</td>
							<td class="text-foreground">{{ p.default_branch || '—' }}</td>
							<td class="max-w-md truncate text-foreground" :title="p.repository_url">
								{{ p.repository_url }}
							</td>
							<td class="text-foreground">
								<router-link
									v-if="p.git_credential_id"
									:to="`/ci/credential/${p.git_credential_id}`"
									class="text-primary hover:underline"
								>
									已配置
								</router-link>
								<span v-else-if="p.has_credential">已配置</span>
								<span v-else class="text-muted-foreground">—</span>
							</td>
							<td class="text-foreground">{{ formatTime(p.created_at) }}</td>
							<td>
								<button
									class="text-primary hover:underline"
									@click="router.push(`/ci/repository/${p.id}`)"
								>
									查看
								</button>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>

		<!-- Pagination -->
		<div v-if="totalPages > 0" class="rounded-lg border border-border bg-card px-6 py-4 shadow-sm">
			<div class="flex items-center justify-between">
				<div class="text-sm text-foreground">
					共 {{ pagination.total }} 条
				</div>
				<div class="flex items-center gap-3">
					<SelectControl
						:model-value="pagination.pageSize"
						:options="pageSizeOptions"
						width-class="w-28"
						@update:model-value="handlePageSizeChange(Number($event))"
					/>
					<button
						class="flex size-9 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
						:disabled="pagination.current <= 1"
						@click="goPage(pagination.current - 1)"
					>
						<ChevronLeft class="size-4" />
					</button>
					<button
						v-for="p in totalPages"
						:key="p"
						class="flex size-9 items-center justify-center rounded-md border text-sm transition-colors"
						:class="p === pagination.current ? 'border-primary bg-primary text-primary-foreground' : 'border-border bg-background text-foreground hover:bg-muted/50'"
						@click="goPage(p)"
					>
						{{ p }}
					</button>
					<button
						class="flex size-9 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
						:disabled="pagination.current >= totalPages"
						@click="goPage(pagination.current + 1)"
					>
						<ChevronRight class="size-4" />
					</button>
				</div>
			</div>
		</div>
	</div>

	<DialogRoot v-model:open="showCreateModal">
		<DialogPortal>
			<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
			<DialogContent
				class="fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[min(520px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
			>
				<div class="mb-5 space-y-1">
					<DialogTitle class="text-lg font-semibold text-foreground">新建仓库</DialogTitle>
					<DialogDescription class="text-sm text-muted-foreground">
						添加一个可用于流水线触发的 Git 仓库。
					</DialogDescription>
				</div>

				<div v-if="modalStatus === 'loading'" class="flex justify-center py-8">
					<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
				</div>

				<div v-else class="space-y-4">
					<div class="space-y-1.5">
						<label class="block text-sm font-medium text-foreground">名称</label>
						<input
							v-model="form.name"
							type="text"
							placeholder="例如: my-backend"
							class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
							:class="errors.name ? 'border-destructive' : 'border-input'"
						/>
						<p v-if="errors.name" class="text-xs text-destructive">{{ errors.name }}</p>
					</div>

					<div class="space-y-1.5">
						<label class="block text-sm font-medium text-foreground">仓库编码</label>
						<input
							v-model="form.code"
							type="text"
							placeholder="例如: my-backend（固化工作目录，创建后不可修改）"
							class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
							:class="errors.code ? 'border-destructive' : 'border-input'"
						/>
						<p v-if="errors.code" class="text-xs text-destructive">{{ errors.code }}</p>
					</div>

					<div class="space-y-1.5">
						<label class="block text-sm font-medium text-foreground">仓库地址</label>
						<input
							v-model="form.repository_url"
							type="text"
							placeholder="git@github.com:user/repo.git"
							class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
							:class="errors.repository_url ? 'border-destructive' : 'border-input'"
						/>
						<p v-if="errors.repository_url" class="text-xs text-destructive">
							{{ errors.repository_url }}
						</p>
					</div>

					<div class="space-y-1.5">
						<label class="block text-sm font-medium text-foreground">Git 凭据（可选）</label>
						<ComboboxSelect
							v-model="form.git_credential_id"
							:options="gitCredentialOptions"
							placeholder="不使用凭据"
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
						class="flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
						:disabled="operating || modalStatus === 'loading'"
						@click="handleCreateOk"
					>
						<span
							v-if="operating"
							class="size-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent"
						/>
						创建
					</button>
				</div>
			</DialogContent>
		</DialogPortal>
	</DialogRoot>
</template>
