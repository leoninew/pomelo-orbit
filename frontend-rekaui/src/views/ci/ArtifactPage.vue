<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { ToolbarRoot } from 'reka-ui';
import { artifactApi, repositoryApi, pipelineTemplateApi } from '@/api/ci';
import ComboboxSelect from '@/components/ComboboxSelect.vue';
import ListPagination from '@/components/ListPagination.vue';
import SearchControl from '@/components/SearchControl.vue';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { Artifact } from '@/types/ci/stage_run';
import type { Repository } from '@/types/ci/repository';
import type { PipelineTemplate } from '@/types/ci/template';
import { formatTime } from '@/utils/time';

const { status, error, execute } = useStatusAsync();
const toast = useToast();
const artifacts = ref<Artifact[]>([]);
const pagination = reactive({ current: 1, pageSize: 20, total: 0 });
const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

const query = reactive({ search: '', repository_id: '', template_id: '' });

const repoOptions = ref<Repository[]>([]);
const templateOptions = ref<PipelineTemplate[]>([]);

const repoSelectOptions = computed(() =>
	repoOptions.value.map((repo) => ({
		value: repo.id,
		label: repo.name,
	}))
);

const templateSelectOptions = computed(() =>
	templateOptions.value.map((template) => ({
		value: template.id,
		label: template.name,
	}))
);

async function loadRepos() {
	try {
		const resp = await repositoryApi.list({ per_page: 100 });
		repoOptions.value = resp.items;
	} catch (err: unknown) {
		toast.error(err instanceof Error ? err.message : '获取项目列表失败');
	}
}

async function loadTemplates() {
	try {
		const resp = await pipelineTemplateApi.list({ per_page: 100 });
		templateOptions.value = resp.items;
	} catch (err: unknown) {
		toast.error(err instanceof Error ? err.message : '获取模板列表失败');
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
	fetchArtifacts();
}

async function fetchArtifacts() {
	try {
		await execute(async () => {
			const resp = await artifactApi.list({
				page: pagination.current,
				per_page: pagination.pageSize,
				search: query.search || undefined,
				repository_id: query.repository_id || undefined,
				template_id: query.template_id || undefined,
			});
			artifacts.value = resp.items;
			pagination.total = resp.total;
		});
	} catch (err: unknown) {
		toast.error(err instanceof Error ? err.message : '获取制品列表失败');
	}
}

function goPage(p: number) {
	pagination.current = p;
	fetchArtifacts();
}

function handlePageSizeChange(pageSize: number) {
	pagination.pageSize = pageSize;
	pagination.current = 1;
	fetchArtifacts();
}

onMounted(async () => {
	fetchArtifacts();
	await loadRepos();
	await loadTemplates();
});
</script>

<template>
	<div class="space-y-6">
		<ToolbarRoot class="flex flex-wrap items-center justify-between gap-3" aria-label="制品工具栏">
			<div class="flex flex-wrap items-center gap-2">
				<ComboboxSelect
					:model-value="query.repository_id"
					:options="repoSelectOptions"
					placeholder="筛选项目"
					width-class="w-60"
					@update:model-value="handleRepositoryChange"
				/>

				<ComboboxSelect
					:model-value="query.template_id"
					:options="templateSelectOptions"
					placeholder="筛选模板"
					width-class="w-60"
					@update:model-value="handleTemplateChange"
				/>

				<SearchControl
					v-model="query.search"
					placeholder="搜索名称/路径"
					:loading="status === 'loading'"
					@search="handleSearch"
				/>
			</div>
		</ToolbarRoot>

		<!-- Table Card -->
		<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div v-if="status === 'loading'" class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error }}</p>
			</div>
			<div v-else-if="artifacts.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="w-full">
					<thead class="border-b border-border bg-muted/30">
						<tr>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">项目</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">模板</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">名称</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">类型</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">Stage</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">路径/镜像</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">时间</th>
							<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">运行记录</th>
						</tr>
					</thead>
					<tbody class="divide-y divide-border">
						<tr v-for="a in artifacts" :key="a.id" class="transition-colors hover:bg-muted/30">
							<td class="px-6 py-5 text-sm">
								<router-link
									:to="`/ci/repository/${a.repository_id}`"
									class="text-primary hover:underline"
								>
									{{ a.repository_name }}
								</router-link>
							</td>
							<td class="px-6 py-5 text-sm">
								<router-link :to="`/ci/template/${a.template_id}`" class="text-primary hover:underline">
									{{ a.template_name }}
								</router-link>
							</td>
							<td class="px-6 py-5 text-sm text-foreground">{{ a.name }}</td>
							<td class="px-6 py-5 text-sm">
								<span class="inline-flex items-center rounded-md bg-secondary px-2 py-0.5 text-sm text-secondary-foreground">{{ a.type }}</span>
							</td>
							<td class="px-6 py-5 text-sm text-foreground">{{ a.stage_name }}</td>
							<td class="px-6 py-5 text-sm text-foreground">{{ a.path ?? '—' }}</td>
							<td class="px-6 py-5 text-sm text-foreground">{{ formatTime(a.created_at) }}</td>
							<td class="px-6 py-5 text-sm">
								<router-link :to="`/ci/run/${a.pipeline_run_id}`" class="text-primary hover:underline">
									查看
								</router-link>
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
