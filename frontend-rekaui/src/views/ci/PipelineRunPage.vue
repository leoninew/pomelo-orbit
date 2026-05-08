<script setup lang="ts">
import { Search } from 'lucide-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { pipelineRunApi, pipelineTemplateApi, repositoryApi } from '@/api/ci';
import ComboboxSelect from '@/components/ComboboxSelect.vue';
import ListPagination from '@/components/ListPagination.vue';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { Repository } from '@/types/ci/repository';
import type { PipelineRun } from '@/types/ci/run';
import type { PipelineTemplate } from '@/types/ci/template';
import { statusBadgeClass, statusLabel } from '@/utils/status';
import { formatTime, formatDuration } from '@/utils/time';
import { ToolbarRoot } from 'reka-ui';

const route = useRoute();
const router = useRouter();
const toast = useToast();
const { status, error, execute } = useStatusAsync();

const runs = ref<PipelineRun[]>([]);
const repositories = ref<Repository[]>([]);
const templates = ref<PipelineTemplate[]>([]);
const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));
const query = reactive({ repository_id: '', template_id: '' });

function triggerLabel(trigger: string) {
	const map: Record<string, string> = {
		manual: '手动',
		webhook: 'Webhook',
	};
	return map[trigger] ?? trigger;
}

const repositoryOptions = computed(() =>
	repositories.value.map((repo) => ({
		value: repo.id,
		label: repo.name,
		description: repo.repository_url,
	}))
);
const templateOptions = computed(() =>
	templates.value.map((template) => ({
		value: template.id,
		label: template.name,
		description: `v${template.version}`,
	}))
);

async function fetchRuns() {
	try {
		await execute(async () => {
			const res = await pipelineRunApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				repository_id: query.repository_id || undefined,
				template_id: query.template_id || undefined,
			});
			runs.value = res.items;
			pagination.total = res.total;
		});
	} catch {
		toast.error('获取流水线记录失败');
	}
}

async function loadRepositories() {
	try {
		const res = await repositoryApi.list({ per_page: 100 });
		repositories.value = res.items;
	} catch (err: unknown) {
		toast.error(err instanceof Error ? err.message : '获取项目列表失败');
	}
}

async function loadTemplates() {
	try {
		const res = await pipelineTemplateApi.list({ per_page: 100 });
		templates.value = res.items;
	} catch (err: unknown) {
		toast.error(err instanceof Error ? err.message : '获取模板列表失败');
	}
}

async function ensureRepositoryOption(id: string) {
	if (!id || repositories.value.some((repo) => repo.id === id)) {
		return;
	}
	try {
		const repo = await repositoryApi.get(id);
		repositories.value = [repo, ...repositories.value];
	} catch {
		// The filter still works by id; missing display text should not block the page.
	}
}

async function ensureTemplateOption(id: string) {
	if (!id || templates.value.some((template) => template.id === id)) {
		return;
	}
	try {
		const template = await pipelineTemplateApi.get(id);
		templates.value = [template, ...templates.value];
	} catch {
		// The filter still works by id; missing display text should not block the page.
	}
}

function handleRepositoryChange(value: string | number | boolean) {
	query.repository_id = String(value || '');
}

function handleTemplateChange(value: string | number | boolean) {
	query.template_id = String(value || '');
}

function handleSearch() {
	pagination.current = 1;
	fetchRuns();
}

function goPage(p: number) {
	pagination.current = p;
	fetchRuns();
}

function handlePageSizeChange(pageSize: number) {
	pagination.pageSize = pageSize;
	pagination.current = 1;
	fetchRuns();
}

onMounted(async () => {
	const repositoryId = route.query.repository_id as string | undefined;
	const templateId = route.query.template_id as string | undefined;
	query.repository_id = repositoryId ?? '';
	query.template_id = templateId ?? '';

	await Promise.all([loadRepositories(), loadTemplates()]);
	await Promise.all([
		ensureRepositoryOption(query.repository_id),
		ensureTemplateOption(query.template_id),
	]);
	fetchRuns();
});
</script>

<template>
	<div class="space-y-6">
		<ToolbarRoot class="overflow-x-auto" aria-label="流水线记录工具栏">
			<div class="flex min-w-max items-center gap-2">
				<ComboboxSelect
					:model-value="query.repository_id"
					:options="repositoryOptions"
					placeholder="筛选项目"
					width-class="w-60"
					@update:model-value="handleRepositoryChange"
				/>
				<ComboboxSelect
					:model-value="query.template_id"
					:options="templateOptions"
					placeholder="筛选模板"
					width-class="w-60"
					@update:model-value="handleTemplateChange"
				/>
				<button
					class="app-button-primary shrink-0 px-5"
					:disabled="status === 'loading'"
					@click="handleSearch"
				>
					<span
						v-if="status === 'loading'"
						class="size-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent"
					/>
					<Search v-else class="size-4" />
					搜索
				</button>
			</div>
		</ToolbarRoot>

		<div class="app-surface">
			<div v-if="status === 'loading'" class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error || '加载失败' }}</p>
			</div>
			<div v-else-if="runs.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list table-fixed min-w-[960px]">
					<colgroup>
						<col class="w-[13%]" />
						<col class="w-[13%]" />
						<col class="w-[7%]" />
						<col class="w-[8%]" />
						<col class="w-[10%]" />
						<col class="w-[8%]" />
						<col class="w-[12%]" />
						<col class="w-[12%]" />
						<col class="w-[8%]" />
						<col class="w-[9%]" />
					</colgroup>
					<thead>
						<tr>
							<th>仓库</th>
							<th>模板</th>
							<th>版本</th>
							<th>触发方式</th>
							<th>触发 Ref</th>
							<th>状态</th>
							<th>错误信息</th>
							<th>开始时间</th>
							<th>耗时</th>
							<th>操作</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="run in runs" :key="run.id">
							<td class="overflow-hidden">
								<button
									class="app-link block truncate"
									:title="run.repository_name"
									@click="router.push(`/ci/repository/${run.repository_id}`)"
								>
									{{ run.repository_name }}
								</button>
							</td>
							<td class="overflow-hidden">
								<router-link
									:to="`/ci/template/${run.template_id}`"
									class="app-link block truncate"
									:title="run.template_name"
								>
									{{ run.template_name }}
								</router-link>
							</td>
							<td>
								<span
									class="inline-block whitespace-nowrap rounded bg-muted px-2 py-0.5 text-sm text-muted-foreground"
								>
									v{{ run.template_version }}
								</span>
							</td>
							<td class="whitespace-nowrap text-foreground">{{ triggerLabel(run.trigger) }}</td>
							<td class="overflow-hidden truncate text-foreground" :title="run.trigger_ref">
								{{ run.trigger_ref }}
							</td>
							<td>
								<span
									class="inline-flex whitespace-nowrap rounded-md px-2 py-0.5 text-sm"
									:class="statusBadgeClass(run.status)"
								>
									{{ statusLabel(run.status) }}
								</span>
							</td>
							<td
								class="overflow-hidden truncate text-foreground"
								:title="run.error_message || undefined"
							>
								{{ run.error_message || '—' }}
							</td>
							<td
								class="overflow-hidden truncate text-foreground"
								:title="formatTime(run.started_at)"
							>
								{{ formatTime(run.started_at) }}
							</td>
							<td class="whitespace-nowrap text-foreground">
								{{ formatDuration(run.started_at, run.finished_at) }}
							</td>
							<td>
								<button
									class="app-link whitespace-nowrap"
									@click="router.push(`/ci/run/${run.id}`)"
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
