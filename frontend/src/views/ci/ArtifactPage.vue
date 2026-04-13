<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">制品记录</h1>
		</div>

		<div class="card bg-base-100 shadow-sm overflow-x-auto">
			<table class="table min-h-48">
				<thead>
					<tr class="text-base-content/60">
						<th>名称</th>
						<th>类型</th>
						<th>项目</th>
						<th>模板</th>
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
						<td class="font-medium">{{ a.name }}</td>
						<td>
							<span class="badge badge-sm badge-ghost">{{ a.type }}</span>
						</td>
						<td>
							<router-link
								:to="`/ci/repository/${a.repository_id}`"
								class="link link-primary text-xs"
							>
								{{ a.repository_name }}
							</router-link>
						</td>
						<td>
							<router-link :to="`/ci/template/${a.template_id}`" class="link link-primary text-xs">
								{{ a.template_name }}
							</router-link>
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
import { artifactApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { Artifact } from '@/types/ci/stage_run';
import { formatTime } from '@/utils/time';

const { status, error, execute } = useStatusAsync();
const toast = useToast();
const artifacts = ref<Artifact[]>([]);
const totalPages = ref(0);
const pagination = reactive({ current: 1, perPage: 20 });

async function fetchArtifacts() {
	try {
		await execute(async () => {
			const resp = await artifactApi.list({
				page: pagination.current,
				per_page: pagination.perPage,
			});
			artifacts.value = resp.items;
			totalPages.value = resp.pages;
		});
	} catch {
		toast.error('获取制品列表失败');
	}
}

function goPage(p: number) {
	pagination.current = p;
	fetchArtifacts();
}

onMounted(fetchArtifacts);
</script>
