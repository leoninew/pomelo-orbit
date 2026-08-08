<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <h1 class="app-detail-page-title">执行快照</h1>
      <button class="app-button h-9 px-3" @click="router.back()">
        <ArrowLeft class="size-4" />
        返回
      </button>
    </div>
    <AppLoadingState v-if="status === 'loading'" size="section" />
    <template v-else-if="snapshot">
      <DetailInfoCard title="冻结的流水线配置">
        <dl class="app-detail-info-grid">
          <div class="flex gap-2">
            <dt>流水线</dt>
            <dd>
              <router-link :to="`/pipeline/${snapshot.pipeline_id}`" class="app-link">
                {{ snapshot.pipeline_name }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>配置版本</dt>
            <dd class="text-foreground">v{{ snapshot.pipeline_version }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>来源模板</dt>
            <dd class="text-foreground">
              {{ snapshot.source_template_name }} v{{ snapshot.source_template_version }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>应用</dt>
            <dd class="text-foreground">{{ snapshot.application_name || '未绑定' }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>代码仓库</dt>
            <dd class="text-foreground">{{ snapshot.repository_name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>来源版本策略</dt>
            <dd class="text-foreground">
              {{
                snapshot.version_fork_strategy === 'fixed'
                  ? `固定版本 ${snapshot.fixed_version_label || ''}`
                  : snapshot.version_fork_strategy === 'latest'
                    ? '最新版本'
                    : '不生成应用版本'
              }}
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>创建时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(snapshot.created_at) }}</dd>
          </div>
        </dl>
      </DetailInfoCard>
      <DetailInfoCard title="阶段快照">
        <AppEmptyState v-if="snapshot.stages_snapshot.length === 0" size="compact" />
        <div v-else class="overflow-x-auto">
          <table class="app-data-table min-w-[760px]">
            <thead>
              <tr>
                <th>#</th>
                <th>名称</th>
                <th>镜像</th>
                <th>依赖</th>
                <th>制品</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="stage in snapshot.stages_snapshot" :key="stage.id">
                <td>{{ stage.sort_order }}</td>
                <td>{{ stage.name }}</td>
                <td class="max-w-xs truncate">{{ stage.image }}</td>
                <td>
                  <div class="flex flex-wrap gap-1">
                    <AppBadge v-for="dependency in stage.depends_on" :key="dependency">
                      {{ stageName(dependency) }}
                    </AppBadge>
                  </div>
                </td>
                <td>{{ stage.artifacts.length }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </DetailInfoCard>
      <DetailInfoCard title="变量快照">
        <VariableDeclarationsTable :declarations="snapshot.variables_snapshot" readonly />
      </DetailInfoCard>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft } from '@lucide/vue';
  import { computed, onMounted, ref, watch } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { pipelineApi } from '@/api/pipeline/pipeline';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { PipelineSnapshotResp } from '@/gen/proto/orbit/v1/pipeline/snapshot';
  import { formatTime } from '@/utils/time';
  import VariableDeclarationsTable from '@/views/pipeline/components/VariableDeclarationsTable.vue';

  const route = useRoute();
  const router = useRouter();
  const toast = useToast();
  const { status, execute } = useStatusAsync();
  const snapshot = ref<PipelineSnapshotResp>();
  const snapshotId = computed(() => String(route.params.id));
  function stageName(id: string) {
    return snapshot.value?.stages_snapshot.find((stage) => stage.id === id)?.name || id;
  }
  async function fetchSnapshot() {
    try {
      await execute(async () => {
        snapshot.value = await pipelineApi.getSnapshot(snapshotId.value);
      });
    } catch (reason) {
      toast.error(reason instanceof Error ? reason.message : '加载快照失败');
      await router.push('/pipeline');
    }
  }
  watch(snapshotId, fetchSnapshot);
  onMounted(fetchSnapshot);
</script>
