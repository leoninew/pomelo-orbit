<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">制品记录</h1>

			<!-- 过滤控件组 -->
			<div class="flex items-center gap-2 flex-wrap">
				<!-- 项目选择 -->
				<div class="dropdown">
					<label class="input input-sm flex items-center gap-1 w-40">
						<Search class="size-3.5 text-base-content/40 shrink-0" />
						<input
							v-model="repoInput"
							tabindex="0"
							type="text"
							class="grow"
							placeholder="筛选项目"
							@input="onRepoInput"
						/>
						<button
							v-if="repoInput"
							class="text-base-content/40 hover:text-base-content/70"
							@click.prevent="clearRepo"
						>
							<X class="size-3" />
						</button>
					</label>
					<ul tabindex="0" class="dropdown-content menu bg-base-100 rounded-box border border-base-200 shadow-lg z-50 w-44 max-h-48 overflow-y-auto flex-nowrap p-0 mt-1">
						<li v-if="repoOptions.length === 0">
							<span class="text-xs text-base-content/50 px-3 py-2">暂无数据</span>
						</li>
						<li v-for="r in repoOptions" :key="r.id">
							<a
								class="text-xs px-3 py-1.5 rounded-none block truncate"
								:class="{ 'bg-primary/10 font-medium': r.id === query.repository_id }"
								@mousedown.prevent="selectRepo(r)"
							>{{ r.name }}</a>
						</li>
					</ul>
				</div>

				<!-- 模板选择 -->
				<div class="dropdown">
					<label class="input input-sm flex items-center gap-1 w-40">
						<Search class="size-3.5 text-base-content/40 shrink-0" />
						<input
							v-model="templateInput"
							tabindex="0"
							type="text"
							class="grow"
							placeholder="筛选模板"
							@input="onTemplateInput"
						/>
						<button
							v-if="templateInput"
							class="text-base-content/40 hover:text-base-content/70"
							@click.prevent="clearTemplate"
						>
							<X class="size-3" />
						</button>
					</label>
					<ul tabindex="0" class="dropdown-content menu bg-base-100 rounded-box border border-base-200 shadow-lg z-50 w-44 max-h-48 overflow-y-auto flex-nowrap p-0 mt-1">
						<li v-if="templateOptions.length === 0">
							<span class="text-xs text-base-content/50 px-3 py-2">暂无数据</span>
						</li>
						<li v-for="t in templateOptions" :key="t.id">
							<a
								class="text-xs px-3 py-1.5 rounded-none block truncate"
								:class="{ 'bg-primary/10 font-medium': t.id === query.template_id }"
								@mousedown.prevent="selectTemplate(t)"
							>{{ t.name }}</a>
						</li>
					</ul>
				</div>

				<!-- 文本搜索 -->
				<label class="input input-sm flex items-center gap-1 w-48">
					<Search class="size-3.5 text-base-content/40 shrink-0" />
					<input
						v-model="query.search"
						type="text"
						class="grow"
						placeholder="搜索名称/路径"
						@keydown.enter="doSearch"
					/>
					<button
						v-if="query.search"
						class="text-base-content/40 hover:text-base-content/70"
						@click="query.search = ''"
					>
						<X class="size-3" />
					</button>
				</label>

				<!-- 搜索按钮 -->
				<button class="btn btn-sm btn-primary" :disabled="status === 'loading'" @click="doSearch">
					<span v-if="status === 'loading'" class="loading loading-spinner loading-xs" />
					搜索
				</button>
			</div>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table min-h-48">
				<thead>
					<tr class="text-base-content/60">
						<th>项目</th>
						<th>模板</th>
						<th>名称</th>
						<th>类型</th>
						<th>Stage</th>
						<th>路径/镜像</th>
						<th>时间</th>
						<th>运行记录</th>
					</tr>
				</thead>
				<tbody>
					<tr v-if="status === 'loading'">
						<td colspan="8" class="text-center py-8">
							<span class="loading loading-spinner loading-md text-primary" />
						</td>
					</tr>
					<tr v-else-if="status === 'error'">
						<td colspan="8" class="text-center py-8 text-error">{{ error }}</td>
					</tr>
					<tr v-else-if="artifacts.length === 0">
						<td colspan="8" class="text-center py-8 text-base-content/60">暂无数据</td>
					</tr>
					<tr v-for="a in artifacts" :key="a.id" class="hover">
						<td>
							<router-link :to="`/ci/repository/${a.repository_id}`" class="link link-primary text-xs">
								{{ a.repository_name }}
							</router-link>
						</td>
						<td>
							<router-link :to="`/ci/template/${a.template_id}`" class="link link-primary text-xs">
								{{ a.template_name }}
							</router-link>
						</td>
						<td class="font-medium">{{ a.name }}</td>
						<td>
							<span class="badge badge-sm badge-ghost">{{ a.type }}</span>
						</td>
						<td class="text-sm text-base-content/70">{{ a.stage_name }}</td>
						<td class="text-sm text-base-content/70">{{ a.path ?? '—' }}</td>
						<td class="cell-muted">{{ formatTime(a.created_at) }}</td>
						<td>
							<router-link :to="`/ci/run/${a.pipeline_run_id}`" class="link link-primary text-xs">
								查看
							</router-link>
						</td>
					</tr>
				</tbody>
			</table>
			<div v-if="totalPages > 1" class="flex justify-end p-3 border-t border-base-200">
				<div class="join">
					<button
						v-for="p in totalPages"
						:key="p"
						class="join-item btn btn-sm"
						:class="p === pagination.current ? 'btn-primary' : 'btn-ghost'"
						@click="goPage(p)"
					>
						{{ p }}
					</button>
				</div>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { Search, X } from 'lucide-vue-next';
import { artifactApi, repositoryApi, pipelineTemplateApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { Artifact } from '@/types/ci/stage_run';
import type { Repository } from '@/types/ci/repository';
import type { PipelineTemplate } from '@/types/ci/template';
import { formatTime } from '@/utils/time';

const { status, error, execute } = useStatusAsync();
const toast = useToast();
const artifacts = ref<Artifact[]>([]);
const totalPages = ref(0);
const pagination = reactive({ current: 1, perPage: 20 });

// 查询参数
const query = reactive({ search: '', repository_id: '', template_id: '' });

// 下拉输入（展示层，不参与查询）
const repoInput = ref('');
const templateInput = ref('');

const repoOptions = ref<Repository[]>([]);
const templateOptions = ref<PipelineTemplate[]>([]);

let repoDebounce: ReturnType<typeof setTimeout> | null = null;
let templateDebounce: ReturnType<typeof setTimeout> | null = null;

async function searchRepos() {
	try {
		const resp = await repositoryApi.list({ search: repoInput.value, per_page: 20 });
		repoOptions.value = resp.items;
	} catch (err: unknown) {
		toast.error(err instanceof Error ? err.message : '获取项目列表失败');
	}
}

async function searchTemplates() {
	try {
		const resp = await pipelineTemplateApi.list({ search: templateInput.value, per_page: 20 });
		templateOptions.value = resp.items;
	} catch (err: unknown) {
		toast.error(err instanceof Error ? err.message : '获取模板列表失败');
	}
}

function onRepoInput() {
	if (repoDebounce) {
		clearTimeout(repoDebounce);
	}
	repoDebounce = setTimeout(searchRepos, 300);
}

function onTemplateInput() {
	if (templateDebounce) {
		clearTimeout(templateDebounce);
	}
	templateDebounce = setTimeout(searchTemplates, 300);
}

function selectRepo(r: Repository) {
	query.repository_id = r.id;
	repoInput.value = r.name;
}

function clearRepo() {
	query.repository_id = '';
	repoInput.value = '';
	doSearch();
}

function selectTemplate(t: PipelineTemplate) {
	query.template_id = t.id;
	templateInput.value = t.name;
}

function clearTemplate() {
	query.template_id = '';
	templateInput.value = '';
	doSearch();
}

function doSearch() {
	pagination.current = 1;
	fetchArtifacts();
}

async function fetchArtifacts() {
	try {
		await execute(async () => {
			const resp = await artifactApi.list({
				page: pagination.current,
				per_page: pagination.perPage,
				search: query.search || undefined,
				repository_id: query.repository_id || undefined,
				template_id: query.template_id || undefined,
			});
			artifacts.value = resp.items;
			totalPages.value = resp.pages;
		});
	} catch (err: unknown) {
		toast.error(err instanceof Error ? err.message : '获取制品列表失败');
	}
}

function goPage(p: number) {
	pagination.current = p;
	fetchArtifacts();
}

onMounted(async () => {
	fetchArtifacts();
	await searchRepos();
	await searchTemplates();
});
</script>
