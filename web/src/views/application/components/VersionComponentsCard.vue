<template>
  <DetailInfoCard :title="t('application.detail.fields.components')">
    <template #actions>
      <button
        v-if="editable"
        class="app-button-primary h-9 px-3"
        :disabled="disabled"
        @click="emit('add')"
      >
        <Plus class="size-4" />
        {{ t('common.add') }}
      </button>
    </template>
    <AppEmptyState v-if="components.length === 0" size="compact" />
    <div v-else class="overflow-x-auto">
      <table class="app-data-table min-w-[840px]">
        <thead>
          <tr>
            <th>{{ t('application.detail.fields.component') }}</th>
            <th>{{ t('application.detail.fields.image') }}</th>
            <th>{{ t('application.componentDetail.fields.pullPolicy') }}</th>
            <th>{{ t('application.componentDetail.fields.restartPolicy') }}</th>
            <th>{{ t('common.operation') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="component in components" :key="component.id">
            <td>
              <router-link :to="`/version/${versionId}/component/${component.id}`" class="app-link">
                {{ component.name }}
              </router-link>
            </td>
            <td class="max-w-md whitespace-normal break-all text-muted-foreground">
              <div>{{ component.image }}</div>
              <div v-if="component.artifact_name" class="mt-1 text-xs">
                {{ component.artifact_name }}
              </div>
              <div
                v-if="component.artifact_local_image_sha256"
                class="mt-1 font-mono text-xs text-muted-foreground"
              >
                {{ component.artifact_local_image_sha256 }}
              </div>
              <div
                v-if="component.artifact_source_commit_sha"
                class="font-mono text-xs text-muted-foreground"
              >
                {{ component.artifact_source_commit_sha }}
              </div>
            </td>
            <td class="text-muted-foreground">{{ component.pull_policy }}</td>
            <td class="text-muted-foreground">{{ component.restart_policy }}</td>
            <td>
              <div v-if="editable" class="flex items-center gap-2">
                <button class="app-link" :disabled="disabled" @click="emit('edit', component)">
                  {{ t('common.edit') }}
                </button>
                <button
                  class="app-link-danger"
                  :disabled="disabled"
                  @click="emit('delete', component)"
                >
                  {{ t('common.delete') }}
                </button>
              </div>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </DetailInfoCard>
</template>

<script setup lang="ts">
  import { Plus } from 'lucide-vue-next';
  import { useI18n } from 'vue-i18n';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import type { VersionComponentResp } from '@/gen/proto/orbit/v1/application/version';

  defineProps<{
    versionId: string;
    components: VersionComponentResp[];
    editable: boolean;
    disabled: boolean;
  }>();

  const emit = defineEmits<{
    add: [];
    edit: [component: VersionComponentResp];
    delete: [component: VersionComponentResp];
  }>();

  const { t } = useI18n();
</script>
