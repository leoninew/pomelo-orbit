<template>
	<div class="vue-flow-wrapper" :class="{ 'is-loading': !isReady }">
		<div v-if="!isReady" class="loading-overlay">
			<span class="loading loading-spinner loading-md text-primary" />
		</div>
		<VueFlow
			:nodes="initialNodes"
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
import type { StageRun } from '@/types/ci/stage_run';
import type { SnapshotStage } from '@/types/ci/snapshot';
import StageNode from './StageNode.vue';

interface Props {
	stages: SnapshotStage[]
	stageRuns?: StageRun[]
	showMinimap?: boolean
}

const props = withDefaults(defineProps<Props>(), { showMinimap: false });
const emit = defineEmits<(e: 'view-stage', stageRun: StageRun) => void>();

const { fitView, updateNodeData } = useVueFlow();
const isReady = ref(false);
const nodeTypes = { stage: markRaw(StageNode) };

// ── 布局：只算一次 ────────────────────────────────────────────────────────────

function buildLayoutedNodes(stages: SnapshotStage[]): Node[] {
	const g = new dagre.graphlib.Graph();
	g.setDefaultEdgeLabel(() => ({}));
	g.setGraph({ rankdir: 'TB', nodesep: 40, ranksep: 60 });
	stages.forEach((s) => g.setNode(s.id, { width: 160, height: 80 }));
	stages.forEach((s) => s.depends_on.forEach((dep) => g.setEdge(dep, s.id)));
	dagre.layout(g);
	return stages
		.filter((s) => s.id && s.name)
		.map((stage) => {
			const n = g.node(stage.id);
			return {
				id: stage.id,
				type: 'stage',
				position: { x: n.x - n.width / 2, y: n.y - n.height / 2 },
				data: { stage, status: undefined, stageRun: undefined },
			};
		});
}

const initialNodes = ref<Node[]>(buildLayoutedNodes(props.stages));

const edges = computed<Edge[]>(() => {
	const result: Edge[] = [];
	for (const stage of props.stages) {
		for (const dep of stage.depends_on) {
			result.push({
				id: `${dep}->${stage.id}`,
				source: dep,
				target: stage.id,
				type: 'smoothstep',
				animated: false,
				markerEnd: { type: 'arrowclosed', color: '#e2e8f0' },
				style: { stroke: '#e2e8f0', strokeWidth: 2 },
			});
		}
	}
	return result;
});

// fitView 在节点挂载后执行一次
watch(
	() => initialNodes.value.length,
	async (len) => {
		if (len > 0 && !isReady.value) {
			await nextTick();
			await nextTick();
			setTimeout(() => {
				fitView({ padding: 0.15, duration: 0 });
				requestAnimationFrame(() => {
					isReady.value = true;
				});
			}, 150);
		}
	},
	{ immediate: true }
);

// ── 状态更新：用 updateNodeData，不重新布局 ───────────────────────────────────

function syncStatus() {
	const map = new Map<string, StageRun>();
	for (const sr of props.stageRuns ?? []) {
		map.set(sr.stage_id, sr);
	}
	for (const stage of props.stages) {
		const sr = map.get(stage.id);
		updateNodeData(stage.id, { stage, status: sr?.status, stageRun: sr }, { replace: true });
	}
}

// isReady 变为 true 时同步一次初始状态（此时 Vue Flow store 已就绪）
watch(isReady, (ready) => {
	if (ready) {
		syncStatus();
	}
});

// stageRuns 变化时同步（轮询场景，此时节点已挂载）
watch(
	() => props.stageRuns,
	() => {
		if (isReady.value) {
			syncStatus();
		}
	},
	{ deep: true }
);

// ── 交互 ─────────────────────────────────────────────────────────────────────

function handleNodeClick(event: NodeClickEvent) {
	const sr = (event.node.data as { stageRun?: StageRun }).stageRun;
	if (sr) {
		emit('view-stage', sr);
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
	inset: 0;
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

.vue-flow-container :deep(.stage-node) {
	background: #ffffff;
	border: 2px solid #4a5568;
}

.vue-flow-container :deep(.status-waiting) {
	border-color: #d69e2e;
	background: #fffbeb;
}
.vue-flow-container :deep(.status-running) {
	border-color: #3182ce;
	background: #ebf8ff;
}
.vue-flow-container :deep(.status-success) {
	border-color: #38a169;
	background: #f0fff4;
}
.vue-flow-container :deep(.status-error) {
	border-color: #e53e3e;
	background: #fff5f5;
}
.vue-flow-container :deep(.status-canceled) {
	border-color: #a0aec0;
	background: #f7fafc;
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
