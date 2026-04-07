<template>
	<table class="table w-full">
		<thead>
			<tr class="text-base-content/60">
				<th v-if="!readonly" class="w-6"></th>
				<th class="w-8">#</th>
				<th class="w-48">名称 / 镜像</th>
				<th>依赖</th>
				<th>制品</th>
				<th v-if="!readonly" class="w-16">操作</th>
			</tr>
		</thead>
		<VueDraggable
			v-if="!readonly"
			v-model="sortableRows"
			tag="tbody"
			handle=".drag-handle"
			:animation="150"
			ghost-class="opacity-30"
			drag-class="shadow-lg"
			chosen-class="bg-base-200"
			@end="onDragEnd"
		>
			<tr v-if="sortableRows.length === 0">
				<td colspan="6" class="text-center py-8 text-base-content/60">
					暂无 Stage，点击"添加 Stage"开始编排
				</td>
			</tr>
			<tr v-for="(row, idx) in sortableRows" :key="row.orch.stage_id" class="hover">
				<td class="pr-0">
					<GripVertical
						class="drag-handle size-4 text-base-content/30 hover:text-base-content/60 cursor-grab active:cursor-grabbing transition-colors"
					/>
				</td>
				<td class="text-base-content/40">{{ idx + 1 }}</td>
				<td>
					<router-link :to="`/ci/stages/${row.stage.id}`" class="font-medium link link-primary">
						{{ row.stage.name }}
					</router-link>
					<div class="text-xs text-base-content/50 font-mono mt-0.5">{{ row.stage.image }}</div>
				</td>
				<td>
					<div v-if="row.dependsOnNames.length > 0" class="flex items-center gap-1 flex-wrap">
						<span
							v-for="dep in row.dependsOnNames"
							:key="dep"
							class="inline-flex items-center gap-1 text-xs bg-base-200 rounded px-2 py-0.5"
						>
							<span class="text-base-content/70">← {{ dep }}</span>
							<button
								class="btn btn-xs btn-ghost btn-circle text-error p-0 w-4 h-4 min-w-4"
								@click="$emit('remove-dependency', idx, row.orch.stage_id, dep)"
							>
								<X class="size-3.5" />
							</button>
						</span>
					</div>
					<span v-else class="text-base-content/40">—</span>
				</td>
				<td>
					<span v-if="row.stage.artifacts && row.stage.artifacts.length > 0">
						{{ row.stage.artifacts.length }} 个
					</span>
					<span v-else class="text-base-content/40">—</span>
				</td>
				<td>
					<button class="link link-error text-xs" @click="$emit('remove-stage', idx)">
						移除
					</button>
				</td>
			</tr>
		</VueDraggable>
		<tbody v-else>
			<tr v-if="rows.length === 0">
				<td colspan="4" class="text-center py-8 text-base-content/60">暂无 Stage</td>
			</tr>
			<tr v-for="(row, idx) in rows" :key="row.orch.stage_id" class="hover">
				<td class="text-base-content/40">{{ idx + 1 }}</td>
				<td>
					<router-link :to="`/ci/stages/${row.stage.id}`" class="font-medium link link-primary">
						{{ row.stage.name }}
					</router-link>
					<div class="text-xs text-base-content/50 font-mono mt-0.5">{{ row.stage.image }}</div>
				</td>
				<td>
					<div v-if="row.dependsOnNames.length > 0" class="flex items-center gap-1 flex-wrap">
						<span
							v-for="dep in row.dependsOnNames"
							:key="dep"
							class="inline-flex items-center gap-1 text-xs bg-base-200 rounded px-2 py-0.5"
						>
							<span class="text-base-content/70">← {{ dep }}</span>
						</span>
					</div>
					<span v-else class="text-base-content/40">—</span>
				</td>
				<td>
					<span v-if="row.stage.artifacts && row.stage.artifacts.length > 0">
						{{ row.stage.artifacts.length }} 个
					</span>
					<span v-else class="text-base-content/40">—</span>
				</td>
			</tr>
		</tbody>
	</table>
</template>

<script setup lang="ts">
import { GripVertical, X } from 'lucide-vue-next';
import { computed, ref, watch } from 'vue';
import { VueDraggable } from 'vue-draggable-plus';
import type { PipelineStage, StageOrchestration } from '@/types/ci/template';

interface Row {
	orch: StageOrchestration
	stage: PipelineStage
	dependsOnNames: string[]
}

interface Props {
	stages: PipelineStage[]
	orchestration: StageOrchestration[]
	readonly?: boolean
}

const props = withDefaults(defineProps<Props>(), { readonly: false });

const emit = defineEmits<{
	(e: 'remove-stage', idx: number): void
	(e: 'remove-dependency', orchIdx: number, stageId: string, depStageId: string): void
	(e: 'reorder', newOrchestration: StageOrchestration[]): void
}>();

const stageMap = computed(() => Object.fromEntries(props.stages.map((s) => [s.id, s])));

const rows = computed(() =>
	[...props.orchestration]
		.sort((a, b) => a.sort_order - b.sort_order)
		.map((orch) => ({
			orch,
			stage: stageMap.value[orch.stage_id],
			dependsOnNames: orch.depends_on.map((id) => stageMap.value[id]?.name ?? id),
		}))
		.filter((r) => r.stage)
);

// 可拖拽的本地副本，拖拽结束后 emit reorder
const sortableRows = ref<Row[]>([]);

watch(
	rows,
	(val) => {
		sortableRows.value = val.map((r) => ({ ...r }));
	},
	{ immediate: true }
);

function onDragEnd() {
	const newOrch = sortableRows.value.map((r, i) => ({
		...r.orch,
		sort_order: i,
	}));
	emit('reorder', newOrch);
}
</script>
