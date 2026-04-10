<template>
	<div class="vue-flow-wrapper" :class="{ 'is-loading': !isReady }">
		<div v-if="!isReady" class="loading-overlay">
			<span class="loading loading-spinner loading-md text-primary" />
		</div>
		<VueFlow
			:nodes="layoutedNodes"
			:edges="edges"
			:default-viewport="{ zoom: 1, x: 0, y: 0 }"
			:min-zoom="0.2"
			:max-zoom="2"
			:fit-view-on-init="false"
			:node-types="nodeTypes"
			:zoom-on-scroll="true"
			:pan-on-scroll="true"
			class="vue-flow-container"
			@node-click="handleNodeClick"
		>
			<Background />
			<Controls />
			<MiniMap v-if="showMinimap" node-color="#9ca3af" />
		</VueFlow>
	</div>
</template>

<script setup lang="ts">
import { Background } from '@vue-flow/background';
import { Controls } from '@vue-flow/controls';
import type { Edge, Node, NodeClickEvent } from '@vue-flow/core';
import { useVueFlow, VueFlow } from '@vue-flow/core';
import { MiniMap } from '@vue-flow/minimap';
import dagre from 'dagre';
import { computed, markRaw, nextTick, ref, watch } from 'vue';
import type { SnapshotStage } from '@/types/ci/snapshot';
import type { TaskStatus } from '@/types/common';
import StageNode from './StageNode.vue';

interface StageRunLike {
	id: string
	pipeline_run_id: string
	stage_id: string
	stage_name: string
	status: TaskStatus
}

interface Props {
	stages: SnapshotStage[]
	stageRuns?: StageRunLike[]
	showMinimap?: boolean
}

const props = withDefaults(defineProps<Props>(), { showMinimap: false });

const emit = defineEmits<(e: 'view-stage', stageRun: StageRunLike) => void>();

const { fitView } = useVueFlow();

// 缓存 stages，避免每次 props 变化都重新布局
const cachedStages = ref(props.stages);
const isReady = ref(false);

watch(
	() => props.stages,
	(newStages) => {
		const newIds = newStages.map((s) => s.id).join(',');
		const cachedIds = cachedStages.value.map((s) => s.id).join(',');
		if (newIds !== cachedIds) {
			isReady.value = false;
			cachedStages.value = newStages;
		} else {
			cachedStages.value = cachedStages.value.map((cached) => {
				const updated = newStages.find((s) => s.id === cached.id);
				return updated || cached;
			});
		}
	}
);

// 在布局计算完成后执行 fitView
watch(
	() => cachedStages.value.length,
	async (newLength) => {
		if (newLength > 0 && !isReady.value) {
			await nextTick();
			await nextTick();
			// 给 dagre 和 Vue Flow 足够的时间完成布局
			setTimeout(() => {
				fitView({ padding: 0.15, duration: 0 });
				// 再等一帧确保 fitView 完成
				requestAnimationFrame(() => {
					isReady.value = true;
				});
			}, 150);
		}
	},
	{ immediate: true }
);

// stage.id -> StageRun 映射
const stageRunByIdMap = computed(() => {
	const map = new Map<string, StageRunLike>();
	for (const sr of props.stageRuns ?? []) {
		map.set(sr.stage_id, sr);
	}
	return map;
});

const nodeTypes = { stage: markRaw(StageNode) };

const nodes = computed<Node[]>(() =>
	cachedStages.value
		.filter((s) => s != null && s.name)
		.map((stage) => ({
			id: stage.id,
			type: 'stage',
			position: { x: 0, y: 0 },
			data: {
				stage,
				status:
					(stage as { status?: string }).status || stageRunByIdMap.value.get(stage.id)?.status,
				readonly: true,
				selected: false,
			},
		}))
);

const edges = computed<Edge[]>(() => {
	const result: Edge[] = [];
	for (const stage of cachedStages.value) {
		for (const dep of stage.depends_on) {
			result.push({
				id: `${dep}->${stage.id}`,
				source: dep,
				target: stage.id,
				type: 'smoothstep',
				animated: false,
				markerEnd: {
					type: 'arrowclosed',
					color: '#e2e8f0',
				},
				style: {
					stroke: '#e2e8f0',
					strokeWidth: 2,
				},
			});
		}
	}
	return result;
});

const layoutedNodes = computed(() => {
	const g = new dagre.graphlib.Graph();
	g.setDefaultEdgeLabel(() => ({}));
	g.setGraph({ rankdir: 'TB', nodesep: 40, ranksep: 60 });
	nodes.value.forEach((node) => g.setNode(node.id, { width: 160, height: 80 }));
	edges.value.forEach((edge) => g.setEdge(edge.source, edge.target));
	dagre.layout(g);
	return nodes.value.map((node) => {
		const n = g.node(node.id);
		return {
			...node,
			position: { x: n.x - n.width / 2, y: n.y - n.height / 2 },
		};
	});
});

function handleNodeClick(event: NodeClickEvent) {
	// 从 cachedStages 中获取 stage 信息
	const stage = cachedStages.value.find((s) => s.id === event.node.id);
	if (!stage) {
		return;
	}

	// 优先从 stage 对象获取 stageRun，如果没有则从 stageRunByIdMap 获取
	const stageRun =
		(stage as { stageRun?: StageRunLike }).stageRun || stageRunByIdMap.value.get(event.node.id);

	if (stageRun) {
		emit('view-stage', stageRun);
	}
}
</script>

<style scoped>
.vue-flow-wrapper {
	position: relative;
	width: 100%;
	height: 400px;
}

.loading-overlay {
	position: absolute;
	top: 0;
	left: 0;
	right: 0;
	bottom: 0;
	display: flex;
	justify-content: center;
	align-items: center;
	background-color: #1a202c;
	z-index: 10;
}

.vue-flow-wrapper.is-loading .vue-flow-container {
	opacity: 0;
	pointer-events: none;
}

.vue-flow-container {
	width: 100%;
	height: 400px;
	border: 1px solid hsl(var(--b2));
	border-radius: 0.5rem;
	overflow: hidden;
	background-color: #1a202c;
	transition: opacity 0.2s ease;
}

/* 统一的浅色箭头 */
.vue-flow-container :deep(.vue-flow__edge-path) {
	stroke: #e2e8f0;
	stroke-width: 2;
}

.vue-flow-container :deep(.vue-flow__edge marker) {
	fill: #e2e8f0;
	stroke: #e2e8f0;
}

.vue-flow-container :deep(.vue-flow__minimap) {
	border: 1px solid hsl(var(--b2));
	border-radius: 0.25rem;
}

.vue-flow-container :deep(.vue-flow__minimap-node) {
	fill: hsl(var(--bc) / 0.5);
}

.vue-flow-container :deep(.vue-flow__node) {
	animation: node-fade-in 0.4s ease-out backwards;
}

.vue-flow-container :deep(.vue-flow__node .stage-node) {
	border: 2px solid #4a5568;
}

.vue-flow-container :deep(.vue-flow__edge) {
	animation: edge-draw-in 0.5s ease-out backwards;
}

@keyframes node-fade-in {
	from {
		opacity: 0;
		transform: scale(0.9);
	}
	to {
		opacity: 1;
		transform: scale(1);
	}
}

@keyframes edge-draw-in {
	from {
		opacity: 0;
	}
	to {
		opacity: 1;
	}
}
</style>
