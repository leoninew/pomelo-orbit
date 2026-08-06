<template>
  <div class="flex flex-col gap-4">
    <AppLoadingState v-if="status === 'loading'" size="section" />

    <div v-else-if="snapshot" class="flex flex-col gap-4">
      <div class="flex flex-wrap items-center justify-between gap-3">
        <h1 class="app-detail-page-title">快照详情</h1>
        <button class="app-button h-9 px-4" @click="router.back()">
          <ArrowLeft class="size-4" />
          返回
        </button>
      </div>

      <!-- 基本信息 -->
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">基本信息</h2>
        </div>
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>快照 ID</dt>
            <dd class="min-w-0 break-all text-foreground">{{ snapshot.id }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>模板</dt>
            <dd>
              <router-link :to="`/pipeline/template/${snapshot.template_id}`" class="app-link">
                {{ snapshot.template_name }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>快照版本</dt>
            <dd class="text-foreground">v{{ snapshot.version }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>模板版本</dt>
            <dd class="text-foreground">v{{ snapshot.template_version }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>创建时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(snapshot.created_at) }}</dd>
          </div>
        </dl>
      </div>

      <!-- Stage 快照 -->
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">Stage 快照</h2>
          <ViewModeToggle v-if="snapshot.stages_snapshot.length > 0" v-model="stagesView" />
        </div>

        <AppEmptyState v-if="snapshot.stages_snapshot.length === 0" size="compact" />

        <!-- 列表视图 -->
        <div v-else-if="stagesView === 'list'" class="overflow-x-auto">
          <table class="app-data-table min-w-[760px]">
            <thead>
              <tr>
                <th>#</th>
                <th>阶段</th>
                <th>版本</th>
                <th>镜像</th>
                <th>依赖</th>
                <th>制品</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(stage, idx) in snapshot.stages_snapshot" :key="stage.id">
                <td class="text-muted-foreground">{{ idx + 1 }}</td>
                <td>
                  <router-link :to="`/pipeline/stage/${stage.id}`" class="app-link">
                    {{ stage.name }}
                  </router-link>
                </td>
                <td class="text-foreground">v{{ stage.version }}</td>
                <td class="text-muted-foreground">{{ stage.image }}</td>
                <td>
                  <div v-if="stage.depends_on.length > 0" class="flex flex-wrap gap-1">
                    <AppBadge v-for="depId in stage.depends_on" :key="depId">
                      {{ snapshotStageMap[depId]?.name }}
                    </AppBadge>
                  </div>
                </td>
                <td class="text-foreground">
                  {{ stage.artifacts?.length }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- DAG 视图 -->
        <div v-else class="p-6">
          <div class="h-[500px]">
            <StageDAGView :stages="snapshot.stages_snapshot" />
          </div>
        </div>
      </div>

      <!-- 变量声明 -->
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">变量声明</h2>
        </div>
        <VariableDeclarationsTable
          :declarations="snapshot.variables_snapshot"
          :readonly="true"
          context="template"
        />
      </div>

      <!-- 制品声明 -->
      <div class="app-surface app-detail-card">
        <div class="app-section-header app-detail-section-header">
          <h2 class="app-detail-section-title">制品声明</h2>
        </div>
        <AppEmptyState v-if="artifactDeclarations.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[720px]">
            <thead>
              <tr>
                <th>Stage</th>
                <th>Collector</th>
                <th>名称</th>
                <th>引用/命令</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="(artifact, idx) in artifactDeclarations" :key="idx">
                <td>
                  <router-link :to="`/pipeline/stage/${artifact.stageId}`" class="app-link">
                    {{ artifact.stageName }}
                  </router-link>
                </td>
                <td>
                  <AppBadge variant="pill">
                    {{ artifact.collector }}
                  </AppBadge>
                </td>
                <td class="text-foreground">{{ artifact.name }}</td>
                <td class="text-muted-foreground">{{ artifact.detail }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft } from 'lucide-vue-next';
  import { computed, onMounted, ref } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { pipelineTemplateApi } from '@/api/pipeline/template';
  import AppBadge from '@/components/AppBadge.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import ViewModeToggle from '@/components/ViewModeToggle.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type {
    PipelineSnapshotResp,
    SnapshotStageResp,
  } from '@/gen/proto/orbit/v1/pipeline/snapshot';
  import { formatTime } from '@/utils/time';
  import StageDAGView from '@/views/pipeline/components/StageDAGView.vue';
  import VariableDeclarationsTable from '@/views/pipeline/components/VariableDeclarationsTable.vue';

  interface ArtifactDeclaration {
    stageId: string;
    stageName: string;
    collector: string;
    name: string;
    detail: string;
  }

  const route = useRoute();
  const router = useRouter();
  const snapshotId = route.params.id as string;
  const toast = useToast();

  const { status, execute } = useStatusAsync();
  const snapshot = ref<PipelineSnapshotResp>();
  const stagesView = ref<'list' | 'dag'>('list');

  const snapshotStageMap = computed<Record<string, SnapshotStageResp>>(() => {
    const map: Record<string, SnapshotStageResp> = {};
    for (const s of snapshot.value?.stages_snapshot ?? []) {
      map[s.id] = s;
    }
    return map;
  });

  const artifactDeclarations = computed(() => {
    const result: ArtifactDeclaration[] = [];
    for (const s of snapshot.value?.stages_snapshot ?? []) {
      for (const a of s.artifacts ?? []) {
        result.push({
          stageId: s.id,
          stageName: s.name,
          collector: a.collector,
          name: a.name,
          detail: a.collector === 'command' ? a.command : a.reference,
        });
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
      router.push('/pipeline/template');
    }
  }

  onMounted(fetchSnapshot);
</script>
