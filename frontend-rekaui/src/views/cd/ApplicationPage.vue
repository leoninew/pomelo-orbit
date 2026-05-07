<script setup lang="ts">
import { Search, X } from 'lucide-vue-next'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { applicationApi } from '@/api/cd/application'
import ListPagination from '@/components/ListPagination.vue'
import { useStatusAsync } from '@/composables/useStatusAsync'
import { useToast } from '@/composables/useToast'
import type { Application } from '@/types/cd/application'
import { appStatusLabel } from '@/utils/status'
import { formatTime } from '@/utils/time'
import { ToolbarRoot } from 'reka-ui'

const router = useRouter()
const toast = useToast()
const { status, execute } = useStatusAsync()

const applications = ref<Application[]>([])
const searchText = ref('')
const pagination = reactive({ current: 1, pageSize: 20, total: 0 })
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize))

const badgeMap: Record<string, string> = {
	deployed: 'border-green-200 bg-green-50 text-green-700',
	deploy_failed: 'border-red-200 bg-red-50 text-red-700',
	deploying: 'border-blue-200 bg-blue-50 text-blue-700',
	undeployed: 'border-border bg-muted text-muted-foreground'
}

function appBadgeClass(s: string) {
	return badgeMap[s] ?? 'border-border bg-muted text-muted-foreground'
}

async function fetchApplications() {
	try {
		await execute(async () => {
			const res = await applicationApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				search: searchText.value || undefined
			})
			applications.value = res.items
			pagination.total = res.total
		})
	} catch {
		toast.error('获取应用列表失败')
	}
}

function handleSearch() {
	pagination.current = 1
	fetchApplications()
}

function goPage(p: number) {
	pagination.current = p
	fetchApplications()
}

function handlePageSizeChange(pageSize: number) {
	pagination.pageSize = pageSize
	pagination.current = 1
	fetchApplications()
}

onMounted(fetchApplications)
</script>

<template>
	<div class="space-y-6">
		<ToolbarRoot class="flex items-center gap-6" aria-label="应用工具栏">
			<div class="flex items-center gap-2">
				<div class="relative">
					<Search class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
					<input
						v-model="searchText"
						type="text"
						placeholder="搜索应用名称"
						class="h-10 w-80 rounded-l-md border border-r-0 border-input bg-background py-2 pl-10 pr-9 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
						@keydown.enter="handleSearch"
					/>
					<button
						v-if="searchText"
						class="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
						@click="searchText = ''; handleSearch()"
					>
						<X class="size-4" />
					</button>
				</div>
				<button
					class="h-10 rounded-r-md border border-border bg-background px-5 text-sm font-medium text-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="status === 'loading'"
					@click="handleSearch"
				>
					搜索
				</button>
			</div>
		</ToolbarRoot>

		<!-- Table Card -->
		<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div v-if="status === 'loading'" class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
			<div v-else-if="applications.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="w-full">
					<thead class="border-b border-border">
						<tr>
							<th class="px-6 py-4 text-left text-sm font-medium text-foreground">名称</th>
							<th class="px-6 py-4 text-left text-sm font-medium text-foreground">编码</th>
							<th class="px-6 py-4 text-left text-sm font-medium text-foreground">状态</th>
							<th class="px-6 py-4 text-left text-sm font-medium text-foreground">创建时间</th>
							<th class="px-6 py-4 text-left text-sm font-medium text-foreground">操作</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border">
						<tr
							v-for="app in applications"
							:key="app.id"
							class="transition-colors hover:bg-muted/30"
						>
							<td class="px-6 py-5 text-sm">
								<button
									class="text-primary hover:underline"
									@click="router.push(`/cd/applications/${app.id}`)"
								>
									{{ app.name }}
								</button>
							</td>
							<td class="px-6 py-5 text-sm text-foreground">{{ app.code }}</td>
							<td class="px-6 py-5 text-sm">
								<span
									class="inline-flex rounded-md border px-2 py-0.5 text-sm"
									:class="appBadgeClass(app.status)"
								>
									{{ appStatusLabel(app.status) }}
								</span>
							</td>
							<td class="px-6 py-5 text-sm text-foreground">{{ formatTime(app.created_at) }}</td>
							<td class="px-6 py-5 text-sm">
								<button
									class="text-primary hover:underline"
									@click="router.push(`/cd/applications/${app.id}`)"
								>
									查看
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
	</div>
</template>
