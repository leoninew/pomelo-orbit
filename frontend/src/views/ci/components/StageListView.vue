<template>
	<table class="table w-full">
		<thead>
			<tr class="text-base-content/60 text-xs">
				<th v-if="!readonly" class="w-6 pr-0"></th>
				<th class="w-8">#</th>
				<th class="w-36">Stage 名称</th>
				<th class="w-40">Stage Key</th>
				<th>依赖</th>
				<th class="w-16 text-center">制品</th>
				<th v-if="!readonly" class="w-24">操作</th>
				<th v-if="readonly && stageStatuses" class="w-24">状态</th>
				<th v-if="readonly && stageStatuses" class="w-16">日志</th>
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
				<td colspan="7" class="text-center py-8 text-base-content/60">
					暂无 Stage，点击"添加 Stage"开始编排
				</td>
			</tr>
			<tr v-for="(row, idx) in sortableRows" :key="row.orch.stage_key" class="hover">
				<td class="pr-0 w-6">
					<GripVertical
						class="drag-handle size-4 text-base-content/30 hover:text-base-content/60 cursor-grab active:cursor-grabbing transition-colors"
					/>
				</td>
				<td class="text-base-content/40 text-xs">{{ idx + 1 }}</td>
				<td>
					<router-link :to="`/ci/stages/${row.stage.id}`" class="link link-primary text-xs">
						{{ row.stage.name }}
					</router-link>
				</td>
				<td class="text-xs text-base-content/70">{{ row.orch.stage_key }}</td>
				<td>
					<div v-if="row.dependsOnKeys.length > 0" class="flex items-center gap-1 flex-wrap">
						<span
							v-for="dep in row.dependsOnKeys"
							:key="dep"
							class="text-xs bg-base-200 rounded px-2 py-0.5 text-base-content/70"
						>
							{{ dep }}
						</span>
					</div>
					<span v-else class="text-base-content/40 text-xs">—</span>
				</td>
				<td class="text-center text-xs text-base-content/60">
					{{ row.stage.artifacts?.length ?? '—' }}
				</td>
				<td>
					<div class="flex items-center gap-3">
						<button class="link link-primary text-xs" @click="$emit('edit-stage', idx)">编辑</button>
						<button class="link link-error text-xs" @click="$emit('remove-stage', idx)">移除</button>
					</div>
				</td>
			</tr>
		</VueDraggable>
		<tbody v-else>
			<tr v-if="rows.length === 0">
				<td :colspan="stageStatuses ? 7 : 5" class="text-center py-8 text-base-content/60">暂无 Stage</td>
			</tr>
			<tr v-for="(row, idx) in rows" :key="row.orch.stage_key" class="hover">
				<td class="text-base-content/40 text-xs">{{ idx + 1 }}</td>
				<td>
					<router-link :to="`/ci/stages/${row.stage.id}`" class="link link-primary text-xs">
						{{ row.stage.name }}
					</router-link>
				</td>
				<td class="text-xs text-base-content/70">{{ row.orch.stage_key }}</td>
				<td>
					<div v-if="row.dependsOnKeys.length > 0" class="flex items-center gap-1 flex-wrap">
						<span
							v-for="dep in row.dependsOnKeys"
							:key="dep"
							class="text-xs bg-base-200 rounded px-2 py-0.5 text-base-content/70"
						>
							{{ dep }}
						</span>
					</div>
					<span v-else class="text-base-content/40 text-xs">—</span>
				</td>
				<td class="text-center text-xs text-base-content/60">
					{{ row.stage.artifacts?.length ?? '—' }}
				</td>
				<td v-if="stageStatuses">
					<span
						v-if="stageStatuses.get(row.orch.stage_key)"
						class="badge badge-xs"
						:class="statusBadgeClass(stageStatuses.get(row.orch.stage_key)!)"
					>
						{{ statusLabel(stageStatuses.get(row.orch.stage_key)!) }}
					</span>
					<span v-else class="text-base-content/40 text-xs">—</span>
				</td>
				<td v-if="stageStatuses">
					<button
						class="link link-primary text-xs"
						@click="emit('view-log', row.orch.stage_key)"
					>
						日志
					</button>
				</td>
			</tr>
		</tbody>
	</table>
</template>

<script setup lang="ts">
import { GripVertical } from 'lucide-vue-next';
import { computed, ref, watch } from 'vue';
import { VueDraggable } from 'vue-draggable-plus';
import type { PipelineStage, StageOrchestration } from '@/types/ci/template';
import { statusBadgeClass, statusLabel } from '@/utils/status';
import { SnapshotStage } from '@/types/ci/snapshot';
import { TaskStatus } from '@/types/common';

interface Row {
	orch: StageOrchestration
	stage: PipelineStage
	dependsOnKeys: string[]
}

interface Props {
	stages: SnapshotStage[]
	orchestration: StageOrchestration[]
	readonly?: boolean
	stageStatuses?: Map<string, TaskStatus>
	linkable?: boolean
}

const props = withDefaults(defineProps<Props>(), { readonly: false, linkable: true });

const emit = defineEmits<{
	(e: 'remove-stage', idx: number): void
	(e: 'edit-stage', idx: number): void
	(e: 'view-log', stageKey: string): void
	(e: 'reorder', newOrchestration: StageOrchestration[]): void
}>();

const stageMap = computed(() => Object.fromEntries(props.stages.map((s) => [s.id, s])));

const rows = computed(() =>
	[...props.orchestration]
		.sort((a, b) => a.sort_order - b.sort_order)
		.map((orch) => ({
			orch,
			stage: stageMap.value[orch.stage_id],
			dependsOnKeys: orch.depends_on,
		}))
		.filter((r) => r.stage)
);

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
