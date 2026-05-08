<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { applicationApi } from '@/api/cd/application'
import ListPagination from '@/components/ListPagination.vue'
import SearchControl from '@/components/SearchControl.vue'
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
const pagination = reactive({ current: 1, pageSize: 10, total: 0 })
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
			<SearchControl
				v-model="searchText"
				placeholder="搜索应用名称"
				:loading="status === 'loading'"
				@search="handleSearch"
			/>
		</ToolbarRoot>

		<!-- Table Card -->
		<div class="app-surface">
			<div v-if="status === 'loading'" class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
			<div v-else-if="applications.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[900px]">
					<thead>
						<tr>
							<th>名称</th>
							<th>编码</th>
							<th>状态</th>
							<th>路由管理</th>
							<th>创建时间</th>
							<th>操作</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="app in applications" :key="app.id">
							<td>
								<button
									class="app-link"
									@click="router.push(`/cd/applications/${app.id}`)"
								>
									{{ app.name }}
								</button>
							</td>
							<td class="text-foreground">{{ app.code }}</td>
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
								<button
									class="app-link"
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
