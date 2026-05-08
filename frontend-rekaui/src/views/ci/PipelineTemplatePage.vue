<script setup lang="ts">
import { Inbox, LayoutGrid, List, Plus } from 'lucide-vue-next'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { pipelineTemplateApi } from '@/api/ci'
import ListPagination from '@/components/ListPagination.vue'
import SearchControl from '@/components/SearchControl.vue'
import { useStatusAsync } from '@/composables/useStatusAsync'
import { useToast } from '@/composables/useToast'
import type { PipelineTemplate } from '@/types/ci/template'
import { formatTime } from '@/utils/time'
import {
	DialogClose,
	DialogContent,
	DialogDescription,
	DialogOverlay,
	DialogPortal,
	DialogRoot,
	DialogTitle,
	ToggleGroupItem,
	ToggleGroupRoot,
	ToolbarRoot
} from 'reka-ui'

const router = useRouter()
const toast = useToast()
const { status, error, execute } = useStatusAsync()
const { loading: operating, execute: executeOp } = useStatusAsync()
const { loading: duplicating, execute: executeDuplicate } = useStatusAsync()

const templates = ref<PipelineTemplate[]>([])
const viewMode = ref<'card' | 'table'>('card')
const pagination = reactive({ current: 1, pageSize: 12, total: 0 })
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize))
const searchText = ref('')
const showCreateDialog = ref(false)

const form = reactive({ name: '', description: '' })
const errors = reactive({ name: '' })

async function fetchTemplates() {
	try {
		await execute(async () => {
			const res = await pipelineTemplateApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				search: searchText.value || undefined
			})
			templates.value = res.items
			pagination.total = res.total
		})
	} catch {
		toast.error('获取模板列表失败')
	}
}

function handleSearch() {
	if (status.value === 'loading') {
		return
	}
	pagination.current = 1
	fetchTemplates()
}

function goPage(p: number) {
	pagination.current = p
	fetchTemplates()
}

function handlePageSizeChange(pageSize: number) {
	pagination.pageSize = pageSize
	pagination.current = 1
	fetchTemplates()
}

function openCreateModal() {
	Object.assign(form, { name: '', description: '' })
	Object.assign(errors, { name: '' })
	showCreateDialog.value = true
}

async function handleCreateOk() {
	errors.name = form.name.trim() ? '' : '请输入模板名称'
	if (errors.name) {
		return
	}
	try {
		await executeOp(async () => {
			const tpl = await pipelineTemplateApi.create({
				name: form.name,
				description: form.description || undefined,
				variable_declarations: []
			})
			toast.success('创建成功')
			showCreateDialog.value = false
			router.push(`/ci/template/${tpl.id}`)
		})
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '创建失败')
	}
}

async function handleDuplicate(id: string) {
	try {
		await executeDuplicate(async () => {
			const newTemplate = await pipelineTemplateApi.duplicate(id)
			toast.success('复制成功')
			router.push(`/ci/template/${newTemplate.id}`)
		})
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '复制失败')
	}
}

onMounted(fetchTemplates)
</script>

<template>
	<div class="space-y-6">
		<ToolbarRoot class="flex flex-wrap items-center justify-between gap-3" aria-label="模板工具栏">
			<SearchControl
				v-model="searchText"
				placeholder="搜索模板名称"
				:loading="status === 'loading'"
				@search="handleSearch"
			/>
			<div class="flex items-center gap-3">
				<ToggleGroupRoot
					v-model="viewMode"
					type="single"
					class="flex h-10 overflow-hidden rounded-md border border-border bg-background"
					aria-label="模板视图"
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
					class="flex h-10 items-center gap-2 rounded-md bg-primary px-5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
					@click="openCreateModal"
				>
					<Plus class="size-4" />
					新建模板
				</button>
			</div>
		</ToolbarRoot>

		<div v-if="status === 'loading'" class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
		</div>

		<div v-else-if="status === 'error'" class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error || '加载失败' }}</p>
			</div>
		</div>

		<div v-else-if="templates.length === 0" class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div class="flex flex-col items-center justify-center py-16">
				<Inbox class="size-12 text-muted-foreground" />
				<p class="mt-2 text-sm text-muted-foreground">暂无模板</p>
			</div>
		</div>

		<div v-else-if="viewMode === 'card'" class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-3">
			<div
				v-for="tpl in templates"
				:key="tpl.id"
				class="group cursor-pointer rounded-lg border border-border bg-card p-4 shadow-sm transition-colors hover:border-primary"
				@click="router.push(`/ci/template/${tpl.id}`)"
			>
				<div class="mb-3 flex items-start justify-between gap-4">
					<h3 class="text-sm font-medium text-foreground group-hover:text-primary">{{ tpl.name }}</h3>
					<button
						:disabled="duplicating"
						class="shrink-0 text-sm text-primary hover:underline disabled:opacity-50"
						@click.stop="handleDuplicate(tpl.id)"
					>
						复制
					</button>
				</div>
				<p class="mb-4 min-h-10 text-sm text-muted-foreground">
					{{ tpl.description || '—' }}
				</p>
				<div class="mb-4 flex flex-wrap gap-2 text-xs text-muted-foreground">
					<span class="rounded bg-muted px-2 py-0.5">v{{ tpl.version }}</span>
					<span class="rounded bg-muted px-2 py-0.5">{{ tpl.orchestration.length }} 阶段</span>
					<span class="rounded bg-muted px-2 py-0.5">{{ tpl.variable_declarations.length }} 变量</span>
				</div>
				<div class="text-sm text-muted-foreground">
					{{ formatTime(tpl.updated_at) }}
				</div>
			</div>
		</div>

		<div v-else class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div class="overflow-x-auto">
				<table class="app-table-list min-w-[1120px]">
					<thead>
						<tr>
							<th>名称</th>
							<th>版本</th>
							<th>阶段</th>
							<th>变量</th>
							<th>描述</th>
							<th>创建时间</th>
							<th class="text-right">操作</th>
						</tr>
					</thead>
					<tbody>
						<tr
							v-for="tpl in templates"
							:key="tpl.id"
							class="cursor-pointer"
							@click="router.push(`/ci/template/${tpl.id}`)"
						>
							<td class="text-foreground">{{ tpl.name }}</td>
							<td>
								<span class="inline-block rounded bg-muted px-2 py-0.5 text-sm text-muted-foreground">v{{ tpl.version }}</span>
							</td>
							<td class="text-foreground">{{ tpl.orchestration.length }}</td>
							<td class="text-foreground">{{ tpl.variable_declarations.length }}</td>
							<td class="max-w-sm truncate text-foreground" :title="tpl.description || undefined">
								{{ tpl.description || '—' }}
							</td>
							<td class="text-foreground">{{ formatTime(tpl.created_at) }}</td>
							<td class="text-right">
								<router-link
									:to="`/ci/template/${tpl.id}`"
									class="text-primary hover:underline"
									@click.stop
								>
									查看
								</router-link>
								<button
									:disabled="duplicating"
									class="ml-3 text-primary hover:underline disabled:opacity-50"
									@click.stop="handleDuplicate(tpl.id)"
								>
									复制
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
			:page-size-options="[12, 24, 48]"
			@change-page="goPage"
			@change-page-size="handlePageSizeChange"
		/>
	</div>

	<DialogRoot v-model:open="showCreateDialog">
		<DialogPortal>
			<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
			<DialogContent
				class="fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[min(520px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
			>
				<div class="mb-5 space-y-1">
					<DialogTitle class="text-lg font-semibold text-foreground">新建流水线模板</DialogTitle>
					<DialogDescription class="text-sm text-muted-foreground">
						创建一个可被仓库流水线复用的模板。
					</DialogDescription>
				</div>
				<div class="space-y-4">
					<div class="space-y-1.5">
						<label class="block text-sm font-medium text-foreground">模板名称</label>
						<input
							v-model="form.name"
							type="text"
							placeholder="输入模板名称"
							class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
							:class="errors.name ? 'border-destructive' : 'border-input'"
						/>
						<p v-if="errors.name" class="text-xs text-destructive">{{ errors.name }}</p>
					</div>
					<div class="space-y-1.5">
						<label class="block text-sm font-medium text-foreground">描述</label>
						<textarea
							v-model="form.description"
							rows="3"
							placeholder="输入模板描述"
							class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
						/>
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
