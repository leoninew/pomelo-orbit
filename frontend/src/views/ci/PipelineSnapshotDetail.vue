<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold flex items-center gap-2">
				快照详情
				<span v-if="snapshot" class="badge badge-sm badge-ghost">v{{ snapshot.version }}</span>
			</h1>
			<button class="btn btn-sm btn-ghost gap-1" @click="$router.back()">
				<ArrowLeft class="size-4" />
				返回
			</button>
		</div>

		<div v-if="status === 'loading'" class="flex justify-center py-16">
			<span class="loading loading-spinner loading-lg text-primary" />
		</div>

		<template v-else-if="snapshot">
			<!-- 基本信息 -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<h2 class="font-semibold mb-4">基本信息</h2>
					<dl class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">模板</dt>
							<dd>
								<router-link
									:to="`/ci/template/${snapshot.template_id}`"
									class="link link-primary text-xs"
								>
									查看
								</router-link>
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">版本</dt>
							<dd>
								<span class="badge badge-sm badge-ghost">v{{ snapshot.version }}</span>
							</dd>
						</div>
						<div class="flex gap-2">
							<dt class="text-base-content/70 w-24 shrink-0">创建时间</dt>
							<dd class="text-base-content/60">{{ formatTime(snapshot.created_at) }}</dd>
						</div>
					</dl>
				</div>
			</div>

			<!-- Stages 编排与变量声明 -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<div class="flex items-center justify-between mb-4">
						<div class="flex items-center gap-3">
							<h2 class="font-semibold">阶段快照</h2>
							<div class="join">
								<button
									class="btn btn-xs join-item"
									:class="stagesView === 'list' ? 'btn-active' : 'btn-ghost'"
									@click="stagesView = 'list'"
								>
									列表
								</button>
								<button
									class="btn btn-xs join-item"
									:class="stagesView === 'dag' ? 'btn-active' : 'btn-ghost'"
									@click="stagesView = 'dag'"
								>
									DAG
								</button>
							</div>
						</div>
					</div>

					<!-- Stages 编排内容 -->
					<table v-if="stagesView === 'list'" class="table w-full">
						<thead>
							<tr class="text-base-content/60 text-xs">
								<th class="w-8">#</th>
								<th>Stage 名称</th>
								<th class="w-20">版本</th>
								<th>依赖</th>
								<th class="w-24 text-center">制品</th>
							</tr>
						</thead>
						<tbody>
							<tr v-if="snapshot.stages_snapshot.length === 0">
								<td colspan="5" class="text-center py-8 text-base-content/60">暂无数据</td>
							</tr>
							<tr v-for="(stage, idx) in snapshot.stages_snapshot" :key="stage.id" class="hover">
								<td class="text-base-content/40 text-xs">{{ idx + 1 }}</td>
								<td>
									<router-link
										:to="`/ci/build-stage/${stage.id}`"
										class="link link-primary text-xs"
									>
										{{ stage.name }}
									</router-link>
								</td>
								<td class="text-center">
									<span class="badge badge-sm badge-ghost">v{{ stage.version }}</span>
								</td>
								<td>
									<div v-if="stage.depends_on.length > 0" class="flex items-center gap-1 flex-wrap">
										<span
											v-for="depId in stage.depends_on"
											:key="depId"
											class="text-xs bg-base-200 rounded px-2 py-0.5 text-base-content/70"
										>
											{{ snapshotStageMap[depId]?.name ?? depId }}
										</span>
									</div>
									<span v-else class="text-base-content/40 text-xs">—</span>
								</td>
								<td class="text-center text-xs text-base-content/60">
									{{ stage.artifacts?.length ?? '—' }}
								</td>
							</tr>
						</tbody>
					</table>

					<div v-else class="min-h-[300px]">
						<p
							v-if="snapshot.stages_snapshot.length === 0"
							class="text-sm text-base-content/60 py-4 text-center"
						>
							暂无数据
						</p>
						<StageDAGView v-else :stages="snapshot.stages_snapshot" />
					</div>

					<!-- 变量声明内容 -->
					<div class="mt-6 pt-6 border-t border-base-300">
						<h3 class="font-semibold mb-4">变量声明</h3>
						<VariableDeclarationsTable
							:declarations="snapshot.variables_snapshot"
							:readonly="true"
							context="template"
						/>
					</div>

					<!-- 制品声明内容 -->
					<div class="mt-6 pt-6 border-t border-base-300">
						<h3 class="font-semibold mb-4">制品声明</h3>
						<div v-if="artifactDeclarations.length === 0" class="text-sm text-base-content/60 py-2">
							暂无数据
						</div>
						<table v-else class="table w-full">
							<thead>
								<tr class="text-base-content/60 text-xs">
									<th>Stage</th>
									<th>类型</th>
									<th>名称</th>
									<th>路径/镜像</th>
								</tr>
							</thead>
							<tbody>
								<tr v-for="(a, idx) in artifactDeclarations" :key="idx" class="hover">
									<td class="text-sm">{{ a.stageName }}</td>
									<td>
										<span class="badge badge-sm badge-ghost">{{ a.type }}</span>
									</td>
									<td class="text-sm">{{ a.name }}</td>
									<td class="text-sm text-base-content/70">{{ a.path }}</td>
								</tr>
							</tbody>
						</table>
					</div>
				</div>
			</div>
		</template>
	</div>
</template>

<script setup lang="ts">
import { ArrowLeft } from 'lucide-vue-next';
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { pipelineTemplateApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { PipelineSnapshot, SnapshotStage } from '@/types/ci/snapshot';
import type { ArtifactDeclaration } from '@/types/ci/template';
import { formatTime } from '@/utils/time';
import StageDAGView from './components/StageDAGView.vue';
import VariableDeclarationsTable from './components/VariableDeclarationsTable.vue';

const route = useRoute();
const router = useRouter();
const snapshotId = route.params.id as string;
const toast = useToast();

const { status, execute } = useStatusAsync();
const snapshot = ref<PipelineSnapshot>();
const stagesView = ref<'list' | 'dag'>('list');

const snapshotStageMap = computed<Record<string, SnapshotStage>>(() => {
	const map: Record<string, SnapshotStage> = {};
	for (const s of snapshot.value?.stages_snapshot ?? []) {
		map[s.id] = s;
	}
	return map;
});

const artifactDeclarations = computed(() => {
	const result: ArtifactDeclaration[] = [];
	for (const s of snapshot.value?.stages_snapshot ?? []) {
		for (const a of s.artifacts ?? []) {
			result.push({ stageName: s.name, type: a.type, name: a.name, path: a.path });
		}
	}
	return result;
});

async function fetchSnapshot() {
	try {
		await execute(async () => {
			snapshot.value = await pipelineTemplateApi.getSnapshot(snapshotId);
		});
	} catch {
		toast.error('获取快照信息失败');
		router.push('/ci/template');
	}
}

onMounted(fetchSnapshot);
</script>
