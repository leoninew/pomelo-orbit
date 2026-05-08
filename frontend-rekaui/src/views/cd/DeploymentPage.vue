<script setup lang="ts">
import { Search, X } from 'lucide-vue-next'
import { computed, onMounted, reactive, ref } from 'vue'
import { useRouter } from 'vue-router'
import { deploymentApi } from '@/api/cd/deployments'
import ListPagination from '@/components/ListPagination.vue'
import { useStatusAsync } from '@/composables/useStatusAsync'
import { useToast } from '@/composables/useToast'
import type { Deployment } from '@/types/cd/deployment'
import { formatDuration, statusBadgeClass, statusLabel } from '@/utils/status'
import { formatTime } from '@/utils/time'
import { ToolbarRoot } from 'reka-ui'

const router = useRouter()
const toast = useToast()
const { status, execute } = useStatusAsync()

const deployments = ref<Deployment[]>([])
const pagination = reactive({ current: 1, pageSize: 10, total: 0 })
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize))
const searchText = ref('')

function operationTypeLabel(type: string) {
	const map: Record<string, string> = {
		deploy: '部署',
		stop: '停止',
		restart: '重启'
	}
	return map[type] ?? type
}

function triggerTypeLabel(type: string) {
	const map: Record<string, string> = {
		manual: '手动'
	}
	return map[type] ?? type
}

async function fetchDeployments() {
	try {
		await execute(async () => {
			const res = await deploymentApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				search: searchText.value || undefined
			})
			deployments.value = res.items
			pagination.total = res.total
		})
	} catch {
		toast.error('获取部署记录失败')
	}
}

function handleSearch() {
	pagination.current = 1
	fetchDeployments()
}

function goPage(p: number) {
	pagination.current = p
	fetchDeployments()
}

function handlePageSizeChange(pageSize: number) {
	pagination.pageSize = pageSize
	pagination.current = 1
	fetchDeployments()
}

onMounted(fetchDeployments)
</script>

<template>
	<div class="space-y-6">
		<ToolbarRoot class="flex items-center gap-6" aria-label="部署记录工具栏">
			<div class="flex items-center gap-2">
				<div class="relative">
					<Search class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
					<input
						v-model="searchText"
						type="text"
						placeholder="搜索应用/环境"
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
			<div v-else-if="deployments.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[1200px]">
					<thead>
						<tr>
							<th>应用</th>
							<th>操作类型</th>
							<th>触发方式</th>
							<th>环境</th>
							<th>环境文件</th>
							<th>状态</th>
							<th>开始时间</th>
							<th>耗时</th>
							<th>操作</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="deployment in deployments" :key="deployment.id">
							<td>
								<button
									class="text-primary hover:underline"
									@click="router.push(`/cd/applications/${deployment.application_id}`)"
								>
									{{ deployment.application_name || deployment.application_id }}
								</button>
							</td>
							<td class="text-foreground">{{ operationTypeLabel(deployment.operation_type) }}</td>
							<td class="text-foreground">{{ triggerTypeLabel(deployment.trigger_type) }}</td>
							<td class="text-foreground">{{ deployment.environment || '—' }}</td>
							<td class="max-w-48 truncate text-foreground" :title="deployment.env_file || undefined">
								{{ deployment.env_file || '—' }}
							</td>
							<td>
								<span
									class="inline-flex rounded-md px-2 py-0.5 text-sm"
									:class="statusBadgeClass(deployment.status)"
								>
									{{ statusLabel(deployment.status) }}
								</span>
							</td>
							<td class="text-foreground">{{ formatTime(deployment.started_at) }}</td>
							<td class="text-foreground">{{ formatDuration(deployment.duration_ms) }}</td>
							<td>
								<button
									class="text-primary hover:underline"
									@click="router.push(`/cd/deployments/${deployment.id}`)"
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
