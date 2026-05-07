<script setup lang="ts">
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

<template>
	<div class="flex flex-col gap-4">
		<div v-if="status === 'loading'" class="flex items-center justify-center py-12">
			<div class="h-8 w-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary"></div>
		</div>

		<div v-else-if="snapshot" class="flex flex-col gap-4">
			<div class="flex flex-wrap items-center justify-between gap-3">
				<h1 class="text-xl font-semibold text-foreground">快照详情</h1>
				<button
					class="h-9 rounded-md border border-input bg-background px-4 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="router.back()"
				>
					返回
				</button>
			</div>

			<!-- 基本信息 -->
			<div class="rounded-lg border border-border bg-card shadow-sm">
				<div class="border-b border-border px-5 py-4">
					<h2 class="font-semibold text-foreground">基本信息</h2>
				</div>
				<dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">快照 ID</dt>
						<dd class="min-w-0 text-foreground">{{ snapshot.id }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">模板名称</dt>
						<dd class="text-foreground">{{ snapshot.template_name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">版本</dt>
						<dd class="text-foreground">v{{ snapshot.template_version }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">创建时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(snapshot.created_at) }}</dd>
					</div>
				</dl>
			</div>

			<!-- Stage 快照 -->
			<div class="rounded-lg border border-border bg-card shadow-sm">
				<div class="flex items-center justify-between border-b border-border px-5 py-4">
					<h2 class="font-semibold text-foreground">Stage 快照</h2>
					<div v-if="snapshot.stages_snapshot.length > 0" class="flex gap-1 rounded-md border border-border bg-background p-1">
						<button
							class="rounded px-3 py-1 text-xs font-medium transition-colors"
							:class="stagesView === 'list' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'"
							@click="stagesView = 'list'"
						>
							列表
						</button>
						<button
							class="rounded px-3 py-1 text-xs font-medium transition-colors"
							:class="stagesView === 'dag' ? 'bg-primary text-primary-foreground' : 'text-muted-foreground hover:bg-muted/50 hover:text-foreground'"
							@click="stagesView = 'dag'"
						>
							DAG
						</button>
					</div>
				</div>

				<!-- 列表视图 -->
				<div v-if="stagesView === 'list'">
					<div v-if="snapshot.stages_snapshot.length === 0" class="px-6 py-12 text-center text-sm text-muted-foreground">
						暂无 Stage
					</div>
					<div v-else class="overflow-x-auto">
						<table class="w-full">
							<thead class="border-b border-border bg-muted/30">
								<tr>
									<th class="px-6 py-3 text-left text-xs font-normal text-muted-foreground">#</th>
									<th class="px-6 py-3 text-left text-xs font-normal text-muted-foreground">Stage 名称</th>
									<th class="px-6 py-3 text-left text-xs font-normal text-muted-foreground">版本</th>
									<th class="px-6 py-3 text-left text-xs font-normal text-muted-foreground">镜像</th>
									<th class="px-6 py-3 text-left text-xs font-normal text-muted-foreground">依赖</th>
									<th class="px-6 py-3 text-center text-xs font-normal text-muted-foreground">制品</th>
								</tr>
							</thead>
							<tbody class="divide-y divide-border">
								<tr
									v-for="(stage, idx) in snapshot.stages_snapshot"
									:key="stage.id"
									class="transition-colors hover:bg-muted/30"
								>
									<td class="px-6 py-4 text-sm text-muted-foreground">{{ idx + 1 }}</td>
									<td class="px-6 py-4 text-sm">
										<router-link
											:to="`/ci/build-stage/${stage.id}`"
											class="text-sm text-primary transition-colors hover:text-primary/80"
										>
											{{ stage.name }}
										</router-link>
									</td>
									<td class="px-6 py-4 text-sm">
										<span class="inline-block rounded-full border border-border bg-muted/30 px-2 py-0.5 text-sm text-foreground">
											v{{ stage.version }}
										</span>
									</td>
									<td class="px-6 py-4 text-sm text-muted-foreground">{{ stage.image }}</td>
									<td class="px-6 py-4 text-sm">
										<div v-if="stage.depends_on.length > 0" class="flex flex-wrap gap-1">
											<span
												v-for="depId in stage.depends_on"
												:key="depId"
												class="inline-block rounded border border-border bg-muted/30 px-2 py-0.5 text-sm text-foreground"
											>
												{{ snapshotStageMap[depId]?.name ?? depId }}
											</span>
										</div>
										<span v-else class="text-sm text-muted-foreground">—</span>
									</td>
									<td class="px-6 py-4 text-center text-sm text-muted-foreground">
										{{ stage.artifacts?.length ?? 0 }}
									</td>
								</tr>
							</tbody>
						</table>
					</div>
				</div>

				<!-- DAG 视图 -->
				<div v-else-if="snapshot.stages_snapshot.length > 0" class="p-6">
					<div class="h-[500px]">
						<StageDAGView :stages="snapshot.stages_snapshot" />
					</div>
				</div>
			</div>

			<!-- 变量声明 -->
			<div class="rounded-lg border border-border bg-card shadow-sm">
				<div class="border-b border-border px-5 py-4">
					<h2 class="font-semibold text-foreground">变量声明</h2>
				</div>
				<VariableDeclarationsTable
					:declarations="snapshot.variables_snapshot"
					:readonly="true"
					context="template"
				/>
			</div>

			<!-- 制品声明 -->
			<div class="rounded-lg border border-border bg-card shadow-sm">
				<div class="border-b border-border px-5 py-4">
					<h2 class="font-semibold text-foreground">制品声明</h2>
				</div>
				<div v-if="artifactDeclarations.length === 0" class="px-5 py-12 text-center text-sm text-muted-foreground">
					暂无制品
				</div>
				<div v-else class="overflow-x-auto">
					<table class="w-full">
						<thead class="border-b border-border bg-muted/30">
							<tr>
								<th class="px-6 py-3 text-left text-xs font-normal text-muted-foreground">Stage</th>
								<th class="px-6 py-3 text-left text-xs font-normal text-muted-foreground">类型</th>
								<th class="px-6 py-3 text-left text-xs font-normal text-muted-foreground">名称</th>
								<th class="px-6 py-3 text-left text-xs font-normal text-muted-foreground">路径</th>
							</tr>
						</thead>
						<tbody class="divide-y divide-border">
							<tr
								v-for="(artifact, idx) in artifactDeclarations"
								:key="idx"
								class="transition-colors hover:bg-muted/30"
							>
								<td class="px-6 py-4 text-sm text-foreground">{{ artifact.stageName }}</td>
								<td class="px-6 py-4 text-sm">
									<span class="inline-block rounded-full border border-border bg-primary/10 px-2 py-0.5 text-xs text-primary">
										{{ artifact.type }}
									</span>
								</td>
								<td class="px-6 py-4 text-sm text-foreground">{{ artifact.name }}</td>
								<td class="px-6 py-4 text-sm text-muted-foreground">{{ artifact.path }}</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</div>
	</div>
</template>
