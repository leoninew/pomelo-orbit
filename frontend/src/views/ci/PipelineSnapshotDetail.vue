<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold flex items-center gap-2">
				快照详情
				<span class="badge badge-sm badge-ghost">v{{ snapshot?.version }}</span>
			</h1>
			<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/ci/templates')">
				<ArrowLeft class="size-4" />
				返回
			</button>
		</div>

		<!-- Loading -->
		<div v-if="status === 'loading'" class="flex justify-center py-16">
			<span class="loading loading-spinner loading-lg text-primary" />
		</div>

		<template v-else-if="snapshot">
			<!-- Snapshot Info -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<h2 class="font-semibold mb-3">快照信息</h2>
					<div class="grid grid-cols-2 gap-4 text-sm">
						<div>
							<span class="text-base-content/60">快照 ID:</span>
							<code class="ml-2">{{ snapshot.id }}</code>
						</div>
						<div>
							<span class="text-base-content/60">模板 ID:</span>
							<code class="ml-2">{{ snapshot.template_id }}</code>
						</div>
						<div>
							<span class="text-base-content/60">版本:</span>
							<span class="ml-2 font-medium">v{{ snapshot.version }}</span>
						</div>
						<div>
							<span class="text-base-content/60">创建时间:</span>
							<span class="ml-2">{{ formatTime(snapshot.created_at) }}</span>
						</div>
						<div>
							<span class="text-base-content/60">Stage 数量:</span>
							<span class="ml-2">{{ snapshot.stages_snapshot.length }}</span>
						</div>
						<div>
							<span class="text-base-content/60">变量数量:</span>
							<span class="ml-2">{{ snapshot.variable_declarations_snapshot.length }}</span>
						</div>
					</div>
					<div class="mt-4 flex gap-2">
						<RouterLink
							:to="`/ci/templates/${snapshot.template_id}`"
							class="btn btn-sm btn-primary"
						>
							查看模板
						</RouterLink>
					</div>
				</div>
			</div>

			<!-- DAG View -->
			<div class="card bg-base-100 shadow-sm">
				<div class="card-body p-5">
					<h2 class="font-semibold mb-3">Stage 流程图</h2>
					<div
						v-if="snapshot.stages_snapshot.length === 0"
						class="text-sm text-base-content/60 py-4 text-center"
					>
						暂无 Stage
					</div>
					<StageDAGView v-else :stages="snapshot.stages_snapshot" />
				</div>
			</div>

			<!-- Variable Declarations -->
			<VariableDeclarationsTable
				:declarations="snapshot.variable_declarations_snapshot"
				readonly
				hint="快照中的变量声明不可修改"
			/>
		</template>
	</div>
</template>

<script setup lang="ts">
import { ArrowLeft } from 'lucide-vue-next';
import { onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { pipelineTemplateApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { PipelineSnapshot } from '@/types/api';
import { formatTime } from '@/utils/time';
import StageDAGView from './components/StageDAGView.vue';
import VariableDeclarationsTable from './components/VariableDeclarationsTable.vue';

const route = useRoute();
const router = useRouter();
const snapshotId = route.params.id as string;
const toast = useToast();

const { status, execute } = useStatusAsync();

const snapshot = ref<PipelineSnapshot>();

async function fetchSnapshot() {
	try {
		await execute(async () => {
			snapshot.value = await pipelineTemplateApi.getSnapshot(snapshotId);
		});
	} catch {
		toast.error('获取快照信息失败');
		router.push('/ci/templates');
	}
}

onMounted(fetchSnapshot);
</script>
