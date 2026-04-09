<template>
	<VueFlow
		:nodes="layoutedNodes"
		:edges="edges"
		:default-viewport="{ zoom: 1, x: 0, y: 0 }"
		:min-zoom="0.2"
		:max-zoom="2"
		:fit-view-on-init="true"
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
</template>

<script setup lang="ts">
import { Background } from '@vue-flow/background';
import { Controls } from '@vue-flow/controls';
import type { Edge, Node, NodeClickEvent } from '@vue-flow/core';
import { useVueFlow, VueFlow } from '@vue-flow/core';
import { MiniMap } from '@vue-flow/minimap';
import dagre from 'dagre';
import { computed, markRaw, nextTick, watch } from 'vue';
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

watch(
	() => props.stages,
	async () => {
		await nextTick();
		fitView({ padding: 0.2 });
	}
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
	props.stages
		.filter((s) => s != null && s.name)
		.map((stage) => ({
			id: stage.id,
			type: 'stage',
			position: { x: 0, y: 0 },
			data: {
				stage,
				status: stageRunByIdMap.value.get(stage.id)?.status,
				readonly: true,
				selected: false,
			},
		}))
);

const edges = computed<Edge[]>(() => {
	const result: Edge[] = [];
	for (const stage of props.stages) {
		for (const dep of stage.depends_on) {
			const sourceStatus = stageRunByIdMap.value.get(dep)?.status;
			const targetStatus = stageRunByIdMap.value.get(stage.id)?.status;

			const isFlowing =
				sourceStatus === 'running' &&
				(targetStatus === 'waiting_to_run' || targetStatus === 'running');
			const isSuccess =
				sourceStatus === 'ran_to_completion' && targetStatus === 'ran_to_completion';
			const isFailed = sourceStatus === 'faulted' || targetStatus === 'faulted';

			let edgeClass = '';
			if (isFailed) {
				edgeClass = 'edge-failed';
			} else if (isFlowing) {
				edgeClass = 'edge-flowing';
			} else if (isSuccess) {
				edgeClass = 'edge-success';
			}

			result.push({
				id: `${dep}->${stage.id}`,
				source: dep,
				target: stage.id,
				type: 'smoothstep',
				animated: isFlowing,
				class: edgeClass,
				markerEnd: {
					type: 'arrowclosed',
					color: isFailed
						? 'hsl(var(--er))'
						: isSuccess
							? 'hsl(var(--su))'
							: 'hsl(var(--bc) / 0.4)',
				},
				style: {
					stroke: isFailed ? 'hsl(var(--er))' : isSuccess ? 'hsl(var(--su))' : undefined,
					strokeWidth: isFlowing ? 2.5 : 1.5,
					strokeDasharray: isFlowing ? '8 4' : undefined,
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
	const stageRun = stageRunByIdMap.value.get(event.node.id);
	if (stageRun) {
		emit('view-stage', stageRun);
	}
}
</script>

<style scoped>
.vue-flow-container {
	width: 100%;
	height: 400px;
	border: 1px solid hsl(var(--b2));
	border-radius: 0.5rem;
	overflow: hidden;
	background-color: hsl(var(--b1) / 0.5);
}

.vue-flow-container :deep(.vue-flow__minimap) {
	border: 1px solid hsl(var(--b2));
	border-radius: 0.25rem;
}

.vue-flow-container :deep(.vue-flow__minimap-node) {
	fill: hsl(var(--bc) / 0.5);
}

.vue-flow-container :deep(.edge-flowing) {
	stroke: hsl(var(--in));
	animation: edge-flow 1s linear infinite;
}

@keyframes edge-flow {
	0% {
		stroke-dashoffset: 12;
	}
	100% {
		stroke-dashoffset: 0;
	}
}

.vue-flow-container :deep(.edge-success) {
	stroke: hsl(var(--su)) !important;
	transition: stroke 0.5s ease;
}

.vue-flow-container :deep(.edge-failed) {
	stroke: hsl(var(--er)) !important;
	stroke-width: 2 !important;
}

.vue-flow-container :deep(.vue-flow__node) {
	animation: node-fade-in 0.4s ease-out backwards;
}

.vue-flow-container :deep(.vue-flow__node .stage-node) {
	border: 2px solid hsl(var(--b3));
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
