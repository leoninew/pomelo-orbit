<template>
	<div class="space-y-6">
		<ToolbarRoot class="app-toolbar-scroll" aria-label="制品工具栏">
			<div class="app-toolbar-row">
				<ComboboxSelect
					:model-value="query.repository_id"
					:options="repoSelectOptions"
					placeholder="筛选项目"
					width-class="app-toolbar-select"
					@update:model-value="handleRepositoryChange"
				/>

				<ComboboxSelect
					:model-value="query.template_id"
					:options="templateSelectOptions"
					placeholder="筛选模板"
					width-class="app-toolbar-select"
					@update:model-value="handleTemplateChange"
				/>

				<SearchControl
					v-model="query.search"
					placeholder="搜索名称/路径"
					:loading="status === 'loading'"
					class="shrink-0"
					@search="handleSearch"
				/>
			</div>
		</ToolbarRoot>

		<div class="app-surface">
			<AppSpinner v-if="status === 'loading'" class="py-16" />
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error || '加载失败' }}</p>
			</div>
			<div v-else-if="artifacts.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">暂无数据</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list table-fixed">
					<colgroup>
						<col class="w-[17%]" />
						<col class="w-[17%]" />
						<col class="w-[12%]" />
						<col class="w-[13%]" />
						<col class="w-[18%]" />
						<col class="w-[16%]" />
						<col class="w-[7%]" />
					</colgroup>
					<thead>
						<tr>
							<th>项目</th>
							<th>模板</th>
							<th>类型</th>
							<th>Stage</th>
							<th>路径/镜像</th>
							<th>创建时间</th>
							<th>运行记录</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="a in artifacts" :key="a.id">
							<td class="overflow-hidden">
								<router-link
									:to="`/ci/repository/${a.repository_id}`"
									class="app-link block truncate"
									:title="a.repository_name"
								>
									{{ a.repository_name }}
								</router-link>
							</td>
							<td class="overflow-hidden">
								<router-link
									:to="`/ci/template/${a.template_id}`"
									class="app-link block truncate"
									:title="a.template_name"
								>
									{{ a.template_name }}
								</router-link>
							</td>
							<td>
								<AppBadge>
									{{ artifactTypeLabel(a.type) }}
								</AppBadge>
							</td>
							<td class="overflow-hidden truncate text-foreground" :title="a.stage_name">
								{{ a.stage_name }}
							</td>
							<td class="overflow-hidden truncate text-foreground" :title="a.path || undefined">
								{{ a.path ?? '—' }}
							</td>
							<td
								class="overflow-hidden truncate text-foreground"
								:title="formatTime(a.created_at)"
							>
								{{ formatTime(a.created_at) }}
							</td>
							<td>
								<router-link :to="`/ci/run/${a.pipeline_run_id}`" class="app-link">
									查看
								</router-link>
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
	</div>
</template>

<script setup lang="ts">
	import { computed, onMounted, reactive, ref } from 'vue';
	import { ToolbarRoot } from 'reka-ui';
	import { artifactApi, repositoryApi, pipelineTemplateApi } from '@/api/ci';
	import AppBadge from '@/components/AppBadge.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
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
	const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
	const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize));

	const query = reactive({ search: '', repository_id: '', template_id: '' });

	const repoOptions = ref<Repository[]>([]);
	const templateOptions = ref<PipelineTemplate[]>([]);

	function artifactTypeLabel(type: Artifact['type']) {
		const map: Record<Artifact['type'], string> = {
			docker_image: 'Docker 镜像',
			binary: '二进制文件',
		};
		return map[type] ?? type;
	}

	const repoSelectOptions = computed(() =>
		repoOptions.value.map((repo) => ({
			value: repo.id,
			label: repo.name,
			description: repo.repository_url,
		}))
	);

	const templateSelectOptions = computed(() =>
		templateOptions.value.map((template) => ({
			value: template.id,
			label: template.name,
			description: `v${template.version}`,
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
		const nextValue = String(value || '');
		if (query.repository_id === nextValue) {
			return;
		}
		query.repository_id = nextValue;
		handleSearch();
	}

	function handleTemplateChange(value: string | number | boolean) {
		const nextValue = String(value || '');
		if (query.template_id === nextValue) {
			return;
		}
		query.template_id = nextValue;
		handleSearch();
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
