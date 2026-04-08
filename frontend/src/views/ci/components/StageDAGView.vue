<template>
	<div class="stage-dag-view">
		<div v-if="!readonly && stages.length > 0" class="dag-controls mb-2 flex items-center gap-2">
			<button v-if="linkMode" class="btn btn-xs btn-primary gap-1" @click="linkMode = false">
				<X class="size-3.5" />
				退出链接模式
			</button>
			<button v-else class="btn btn-xs btn-ghost gap-1" @click="linkMode = true">
				<Link2 class="size-3.5" />
				链接依赖
			</button>
			<span v-if="linkMode" class="text-xs text-base-content/60">
				点击源 Stage，再点击目标 Stage，创建"源依赖目标"关系（箭头从目标指向源）
			</span>
			<span v-if="selectedStage && linkMode" class="text-xs text-primary">
				已选择: {{ selectedStage }}
			</span>
			<button
				v-if="selectedStage && linkMode"
				class="btn btn-xs btn-secondary gap-1"
				@click="copySelectedStage"
			>
				<Copy class="size-3.5" />
				复制 Stage
			</button>
			<span v-if="selectedEdge" class="text-xs text-primary">
				已选择边: {{ selectedEdge.source }} → {{ selectedEdge.target }}
			</span>
			<button v-if="selectedEdge" class="btn btn-xs btn-error gap-1" @click="deleteSelectedEdge">
				<X class="size-3.5" />
				删除依赖
			</button>
		</div>

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
			:class="{ 'link-mode': linkMode }"
			@node-click="handleNodeClick"
			@edge-click="handleEdgeClick"
			@connect="handleConnect"
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
import { VueFlow } from '@vue-flow/core';
import { MiniMap } from '@vue-flow/minimap';
import dagre from 'dagre';
import { Copy, Link2, X } from 'lucide-vue-next';
import { computed, markRaw, ref } from 'vue';
import type { SnapshotStage } from '@/types/ci/snapshot';
import StageNode from './StageNode.vue';

interface Props {
	stages: SnapshotStage[]
	stageStatuses?: Map<string, 'success' | 'failed' | 'running' | 'waiting' | 'mixed' | 'skipped'>
	showMinimap?: boolean
	readonly?: boolean
}

const props = withDefaults(defineProps<Props>(), {
	showMinimap: false,
	readonly: false,
});

const emit = defineEmits<{
	(e: 'view-stage', stage: SnapshotStage): void
	(e: 'edit-stage', stage: SnapshotStage): void
	(e: 'update-dependencies', stageName: string, dependsOn: string[]): void
	(e: 'delete-dependency', sourceName: string, targetName: string): void
	(e: 'copy-stage', stageName: string): void
}>();

const linkMode = ref(false);
const selectedStage = ref<string | null>(null);
const selectedEdge = ref<{ source: string; target: string } | null>(null);

const nodeTypes = {
	stage: markRaw(StageNode),
};

const nodes = computed<Node[]>(() => {
	return props.stages
		.filter((stage) => stage != null && stage.name)
		.map((stage) => ({
			id: stage.name,
			type: 'stage',
			position: { x: 0, y: 0 },
			data: {
				stage,
				status: props.stageStatuses?.get(stage.name),
				readonly: props.readonly,
				selected: selectedStage.value === stage.name,
			},
		}));
});

const edges = computed<Edge[]>(() => {
	const edges: Edge[] = [];
	for (const stage of props.stages) {
		for (const dep of stage.depends_on) {
			const sourceStatus = props.stageStatuses?.get(dep);
			const targetStatus = props.stageStatuses?.get(stage.name);

			// Enhanced edge animation logic
			const isFlowing =
				sourceStatus === 'running' && (targetStatus === 'waiting' || targetStatus === 'running');
			const isSuccess = sourceStatus === 'success' && targetStatus === 'success';
			const isFailed = sourceStatus === 'failed' || targetStatus === 'failed';
			const isSelected = linkMode.value && selectedStage.value === stage.name;

			// Determine edge style class
			let edgeClass = '';
			if (isFailed) {
				edgeClass = 'edge-failed';
			} else if (isFlowing) {
				edgeClass = 'edge-flowing';
			} else if (isSuccess) {
				edgeClass = 'edge-success';
			} else if (isSelected) {
				edgeClass = 'edge-selected';
			}

			edges.push({
				id: `${dep}->${stage.name}`,
				source: dep,
				target: stage.name,
				type: 'smoothstep',
				animated: isFlowing,
				class: edgeClass,
				style: {
					stroke:
						sourceStatus === 'failed'
							? 'hsl(var(--er))'
							: isSelected
								? 'hsl(var(--p))'
								: isSuccess
									? 'hsl(var(--su))'
									: undefined,
					strokeWidth: isFlowing || isSelected ? 2.5 : 1.5,
					strokeDasharray: isFlowing ? '8 4' : undefined,
				},
			});
		}
	}
	return edges;
});

// 使用 dagre 自动布局
const layoutedNodes = computed(() => {
	const g = new dagre.graphlib.Graph();
	g.setDefaultEdgeLabel(() => ({}));
	g.setGraph({
		rankdir: 'TB', // 从上到下的布局
		nodesep: 80, // 节点水平间距
		ranksep: 120, // 层级垂直间距
	});

	// 添加节点到图
	nodes.value.forEach((node) => {
		g.setNode(node.id, { width: 160, height: 80 }); // 假设节点尺寸
	});

	// 添加边到图
	edges.value.forEach((edge) => {
		g.setEdge(edge.source, edge.target);
	});

	// 计算布局
	dagre.layout(g);

	// 应用布局位置到节点
	return nodes.value.map((node) => {
		const nodeWithPosition = g.node(node.id);
		return {
			...node,
			position: {
				x: nodeWithPosition.x - nodeWithPosition.width / 2,
				y: nodeWithPosition.y - nodeWithPosition.height / 2,
			},
		};
	});
});

function handleNodeClick(event: NodeClickEvent) {
	if (props.readonly) {
		// 只读模式（运行详情）：点击节点查看日志
		const stageName = event.node.id;
		const stage = props.stages.find((s) => s.name === stageName);
		if (stage) {
			emit('view-stage', stage);
		}
		return;
	}

	const stageName = event.node.id;
	const stage = props.stages.find((s) => s.name === stageName);
	if (!stage) {
		return;
	}

	if (linkMode.value) {
		if (selectedStage.value === null) {
			selectedStage.value = stageName;
		} else if (selectedStage.value === stageName) {
			selectedStage.value = null;
		} else {
			// Create dependency from selected to current
			const targetStage = props.stages.find((s) => s.name === selectedStage.value);
			if (targetStage) {
				const newDependsOn = [...targetStage.depends_on, stageName];
				emit('update-dependencies', targetStage.name, newDependsOn);
			}
			selectedStage.value = null;
			linkMode.value = false;
		}
	} else {
		emit('edit-stage', stage);
	}
}

function handleEdgeClick(event: { edge: Edge }) {
	if (props.readonly || linkMode.value) {
		return;
	}

	const edge = event.edge;
	selectedEdge.value = { source: edge.source, target: edge.target };
}

function deleteSelectedEdge() {
	if (!selectedEdge.value) {
		return;
	}

	emit('delete-dependency', selectedEdge.value.source, selectedEdge.value.target);
	selectedEdge.value = null;
}

function copySelectedStage() {
	if (!selectedStage.value) {
		return;
	}
	emit('copy-stage', selectedStage.value);
	selectedStage.value = null;
	linkMode.value = false;
}

function handleConnect() {
	// Prevent creating edges by dragging (use link mode instead)
	return false;
}
</script>

<style scoped>
.stage-dag-view {
	width: 100%;
}

.dag-controls {
	padding: 0.5rem;
	background: hsl(var(--b2));
	border-radius: 0.5rem;
	border: 1px solid hsl(var(--b3));
}

.vue-flow-container {
	width: 100%;
	height: 400px;
	border: 1px solid hsl(var(--b2));
	border-radius: 0.5rem;
	overflow: hidden;
	background-color: hsl(var(--b1) / 0.5);
}

.vue-flow-container.link-mode {
	cursor: crosshair;
}

.vue-flow-container :deep(.vue-flow__minimap) {
	border: 1px solid hsl(var(--b2));
	border-radius: 0.25rem;
}

.vue-flow-container :deep(.vue-flow__minimap-node) {
	fill: hsl(var(--bc) / 0.5);
}

/* Enhanced edge animations */
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
	animation: edge-pulse-error 1s ease-in-out;
}

@keyframes edge-pulse-error {
	0%,
	100% {
		opacity: 1;
	}
	50% {
		opacity: 0.6;
	}
}

.vue-flow-container :deep(.edge-selected) {
	stroke: hsl(var(--p)) !important;
	stroke-width: 2.5 !important;
	filter: drop-shadow(0 0 4px hsl(var(--p) / 0.5));
}

/* Node entrance animation coordination */
.vue-flow-container :deep(.vue-flow__node) {
	animation: node-fade-in 0.4s ease-out backwards;
}

/* 确保节点边框可见 */
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
