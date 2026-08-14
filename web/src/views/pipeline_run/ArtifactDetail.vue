<template>
  <div class="flex flex-col gap-4">
    <div class="flex flex-wrap items-center justify-between gap-3">
      <div class="flex min-w-0 flex-wrap items-center gap-2">
        <h1 class="app-detail-page-title min-w-0 break-words">制品详情</h1>
        <DetailHeaderMeta v-if="artifact">
          <AppBadge variant="pill">{{ artifact.collector }}</AppBadge>
        </DetailHeaderMeta>
      </div>
      <button class="app-button h-9 px-4" @click="router.push('/pipeline-run/artifact')">
        <ArrowLeft class="size-4" />
        返回
      </button>
    </div>

    <AppLoadingState v-if="loading" size="section" />

    <template v-else-if="artifact">
      <DetailInfoCard title="制品信息">
        <dl class="app-detail-info-grid">
          <div class="flex gap-2 sm:col-span-2">
            <dt>ID</dt>
            <dd class="min-w-0 break-all font-mono text-foreground">{{ artifact.id }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>名称</dt>
            <dd class="text-foreground">{{ artifact.name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>Collector</dt>
            <dd>
              <AppBadge variant="pill">{{ artifact.collector }}</AppBadge>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>流水线运行</dt>
            <dd class="min-w-0">
              <router-link :to="`/pipeline-run/${artifact.pipeline_run_id}`" class="app-link">
                查看运行
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>构建阶段</dt>
            <dd class="min-w-0 text-foreground">{{ artifact.stage_name }}</dd>
          </div>
          <div class="flex gap-2">
            <dt>仓库</dt>
            <dd class="min-w-0">
              <router-link :to="`/repository/${artifact.repository_id}`" class="app-link">
                {{ artifact.repository_name }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>流水线</dt>
            <dd class="min-w-0">
              <router-link :to="`/pipeline/${artifact.pipeline_id}`" class="app-link">
                {{ artifact.pipeline_name }}
              </router-link>
            </dd>
          </div>
          <div class="flex gap-2">
            <dt>创建时间</dt>
            <dd class="text-muted-foreground">{{ formatTime(artifact.created_at) }}</dd>
          </div>
        </dl>
      </DetailInfoCard>

      <DetailInfoCard v-if="hasContent" title="制品内容">
        <dl class="app-detail-info-grid">
          <div v-if="artifact.location !== undefined" class="flex gap-2 sm:col-span-2">
            <dt>位置</dt>
            <dd class="min-w-0 break-all text-foreground">{{ artifact.location }}</dd>
          </div>
          <div v-if="artifact.value_format !== undefined" class="flex gap-2">
            <dt>值格式</dt>
            <dd class="text-foreground">{{ artifact.value_format }}</dd>
          </div>
          <div v-if="artifact.value !== undefined" class="flex gap-2 sm:col-span-2">
            <dt>值</dt>
            <dd class="min-w-0 flex-1">
              <pre class="whitespace-pre-wrap break-all text-sm text-foreground">{{
                artifact.value
              }}</pre>
            </dd>
          </div>
        </dl>
      </DetailInfoCard>

      <DetailInfoCard v-if="hasLineage" title="关联">
        <dl class="app-detail-info-grid">
          <div v-if="artifact.application_id && artifact.application_name" class="flex gap-2">
            <dt>应用</dt>
            <dd>
              <router-link :to="`/application/${artifact.application_id}`" class="app-link">
                {{ artifact.application_name }}
              </router-link>
            </dd>
          </div>
          <div
            v-if="artifact.source_version_id && artifact.source_version_label"
            class="flex gap-2"
          >
            <dt>来源版本</dt>
            <dd>
              <router-link :to="`/version/${artifact.source_version_id}`" class="app-link">
                {{ artifact.source_version_label }}
              </router-link>
            </dd>
          </div>
          <div v-if="artifact.image_ref !== undefined" class="flex gap-2 sm:col-span-2">
            <dt>镜像引用</dt>
            <dd class="min-w-0 break-all text-foreground">{{ artifact.image_ref }}</dd>
          </div>
          <div v-if="artifact.local_image_sha256 !== undefined" class="flex gap-2 sm:col-span-2">
            <dt>本地镜像 SHA256</dt>
            <dd class="min-w-0 break-all text-foreground">
              {{ artifact.local_image_sha256 }}
            </dd>
          </div>
          <div v-if="artifact.source_commit_sha !== undefined" class="flex gap-2 sm:col-span-2">
            <dt>源码提交 SHA</dt>
            <dd class="min-w-0 break-all text-foreground">
              {{ artifact.source_commit_sha }}
            </dd>
          </div>
          <div
            v-if="artifact.generated_version_id && artifact.generated_version_label"
            class="flex gap-2"
          >
            <dt>生成版本</dt>
            <dd>
              <router-link :to="`/version/${artifact.generated_version_id}`" class="app-link">
                {{ artifact.generated_version_label }}
              </router-link>
            </dd>
          </div>
          <div
            v-if="
              artifact.generated_version_id &&
              artifact.version_component_id &&
              artifact.version_component_name
            "
            class="flex gap-2"
          >
            <dt>版本组件</dt>
            <dd>
              <router-link
                :to="`/version/${artifact.generated_version_id}/component/${artifact.version_component_id}`"
                class="app-link"
              >
                {{ artifact.version_component_name }}
              </router-link>
            </dd>
          </div>
        </dl>
      </DetailInfoCard>
    </template>
  </div>
</template>

<script setup lang="ts">
  import { ArrowLeft } from '@lucide/vue';
  import { computed, onMounted, ref, watch } from 'vue';
  import { useRoute, useRouter } from 'vue-router';
  import { artifactApi } from '@/api/pipeline_run/artifact';
  import AppBadge from '@/components/AppBadge.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import AppLoadingState from '@/components/AppLoadingState.vue';
  import DetailHeaderMeta from '@/components/DetailHeaderMeta.vue';
  import { usePageBreadcrumbs } from '@/composables/useBreadcrumbs';
  import { useStatusAsync } from '@/composables/useStatusAsync';
  import { useToast } from '@/composables/useToast';
  import type { ArtifactResp } from '@/gen/proto/orbit/v1/pipeline_run/artifact';
  import { formatTime } from '@/utils/time';

  const route = useRoute();
  const router = useRouter();
  const toast = useToast();
  usePageBreadcrumbs([]);
  const artifactId = computed(() => route.params.id as string);
  const artifact = ref<ArtifactResp>();
  const { loading, execute } = useStatusAsync();

  const hasContent = computed(
    () =>
      artifact.value?.location !== undefined ||
      artifact.value?.value !== undefined ||
      artifact.value?.value_format !== undefined
  );
  const hasLineage = computed(
    () =>
      artifact.value?.image_ref !== undefined ||
      artifact.value?.local_image_sha256 !== undefined ||
      artifact.value?.source_commit_sha !== undefined ||
      artifact.value?.application_id !== undefined ||
      artifact.value?.source_version_id !== undefined ||
      artifact.value?.generated_version_id !== undefined
  );

  async function fetchArtifact() {
    artifact.value = undefined;
    try {
      await execute(async () => {
        artifact.value = await artifactApi.get(artifactId.value);
      });
    } catch (error) {
      toast.error(error instanceof Error ? error.message : '加载制品详情失败');
      router.push('/pipeline-run/artifact');
    }
  }

  watch(artifactId, fetchArtifact);
  onMounted(fetchArtifact);
</script>
