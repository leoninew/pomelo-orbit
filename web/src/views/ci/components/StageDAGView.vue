<template>
  <div class="h-full w-full rounded-lg border border-border bg-background">
    <VueFlow
      :nodes="initialNodes"
      :edges="edges"
      :node-types="nodeTypes"
      :fit-view-on-init="false"
      class="vue-flow-custom"
      @node-click="handleNodeClick"
    >
      <Background pattern-color="hsl(var(--muted-foreground) / 0.2)" :gap="16" />
      <Controls class="vue-flow-controls-custom" />
      <MiniMap v-if="showMinimap" class="vue-flow-minimap-custom" />
    </VueFlow>
  </div>
</template>

<script setup lang="ts">
  import { Background } from '@vue-flow/background';
  import { Controls } from '@vue-flow/controls';
  import type { Edge, Node, NodeComponent, NodeTypesObject } from '@vue-flow/core';
  import { useVueFlow, VueFlow } from '@vue-flow/core';
  import { MiniMap } from '@vue-flow/minimap';
  import dagre from 'dagre';
  import { computed, markRaw, nextTick, ref, watch } from 'vue';
  import type { StageRunResp } from '@/gen/proto/orbit/pipeline_run';
  import type { SnapshotStageResp } from '@/gen/proto/orbit/snapshot';
  import { statusColor } from '@/utils/status';
  import StageNode from './StageNode.vue';

  interface Props {
    stages: SnapshotStageResp[];
    stageRuns?: StageRunResp[];
    showMinimap?: boolean;
    animated?: boolean;
  }

  const props = withDefaults(defineProps<Props>(), { showMinimap: false, animated: false });
  const emit = defineEmits<(e: 'view-stage', stageRun: StageRunResp) => void>();

  const { fitView, updateNodeData } = useVueFlow();
  const isReady = ref(false);
  const nodeTypes: NodeTypesObject = { stage: markRaw(StageNode) as unknown as NodeComponent };

  // ── 布局：只算一次 ────────────────────────────────────────────────────────────

  function buildLayoutedNodes(stages: SnapshotStageResp[]): Node[] {
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
    const stageRunMap = new Map<string, StageRunResp>();
    for (const sr of props.stageRuns ?? []) {
      stageRunMap.set(sr.stage_id, sr);
    }

    const result: Edge[] = [];
    for (const stage of props.stages) {
      for (const dep of stage.depends_on) {
        const targetStatus =
          props.stageRuns !== undefined
            ? (stageRunMap.get(stage.id)?.status ?? 'waiting_to_run')
            : stageRunMap.get(stage.id)?.status;
        const color = statusColor(targetStatus);
        result.push({
          id: `${dep}->${stage.id}`,
          source: dep,
          target: stage.id,
          type: 'smoothstep',
          animated: props.animated,
          style: { stroke: color, strokeWidth: 2 },
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
    const map = new Map<string, StageRunResp>();
    for (const sr of props.stageRuns ?? []) {
      map.set(sr.stage_id, sr);
    }
    for (const stage of props.stages) {
      const sr = map.get(stage.id);
      // 只有传入了 stageRuns（运行详情场景）才显示默认 waiting_to_run
      // 纯展示模式（模板/快照详情）不传 stageRuns，status 保持 undefined
      const status = props.stageRuns !== undefined ? (sr?.status ?? 'waiting_to_run') : sr?.status;
      updateNodeData(stage.id, { stage, status, stageRun: sr }, { replace: true });
    }
  }

  // isReady 变为 true 时同步一次初始状态（此时 Vue Flow store 已就绪）
  watch(isReady, (ready) => {
    if (ready) {
      syncStatus();
    }
  });

  // stageRuns 变化时同步；isReady 前的变化会在 isReady 触发时通过上面的 watch 补齐
  watch(
    () => props.stageRuns,
    () => {
      if (isReady.value) {
        syncStatus();
      }
    },
    { deep: true, immediate: true }
  );

  // ── 交互 ─────────────────────────────────────────────────────────────────────

  function handleNodeClick(event: { node: Node }) {
    const sr = (event.node.data as { stageRun?: StageRunResp }).stageRun;
    if (sr) {
      emit('view-stage', sr);
    }
  }
</script>

<style scoped>
  .vue-flow-custom {
    background-color: hsl(var(--background));
  }

  :deep(.vue-flow__node) {
    background: transparent;
    border: none;
    padding: 0;
  }

  :deep(.vue-flow__edge-path) {
    stroke-width: 2;
  }

  :deep(.vue-flow-controls-custom) {
    background-color: hsl(var(--card));
    border: 1px solid hsl(var(--border));
    border-radius: 0.5rem;
  }

  :deep(.vue-flow-controls-custom button) {
    background-color: hsl(var(--background));
    border-bottom: 1px solid hsl(var(--border));
    color: hsl(var(--foreground));
  }

  :deep(.vue-flow-controls-custom button:hover) {
    background-color: hsl(var(--muted));
  }

  :deep(.vue-flow-minimap-custom) {
    background-color: hsl(var(--card));
    border: 1px solid hsl(var(--border));
    border-radius: 0.5rem;
  }
</style>
