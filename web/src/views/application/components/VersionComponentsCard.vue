<template>
  <DetailInfoCard :title="t('application.detail.fields.components')" actions-class="flex-nowrap">
    <template #actions>
      <SearchControl
        v-model="searchText"
        :placeholder="t('application.detail.searchComponentsPlaceholder')"
        :loading="loading"
        class="min-w-0 flex-1"
        @search="handleSearch"
      />
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
    <AppEmptyState v-if="filteredComponents.length === 0" size="compact" />
    <div v-else class="overflow-x-auto">
      <table class="app-data-table min-w-[960px]">
        <thead>
          <tr>
            <th>{{ t('application.detail.fields.component') }}</th>
            <th>{{ t('application.detail.fields.image') }}</th>
            <th>{{ t('application.componentDetail.fields.imageSha256') }}</th>
            <th>{{ t('application.componentDetail.fields.pullPolicy') }}</th>
            <th>{{ t('application.componentDetail.fields.restartPolicy') }}</th>
            <th>{{ t('common.operation') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="component in filteredComponents" :key="component.id">
            <td>
              <router-link :to="`/version/${versionId}/component/${component.id}`" class="app-link">
                {{ component.name }}
              </router-link>
            </td>
            <td class="max-w-xs truncate text-muted-foreground" :title="component.image">
              {{ component.image }}
            </td>
            <td
              class="max-w-xs truncate text-muted-foreground"
              :title="component.artifact_local_image_sha256 || ''"
            >
              {{ component.artifact_local_image_sha256 || '-' }}
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
  import { Plus } from '@lucide/vue';
  import { computed, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppEmptyState from '@/components/AppEmptyState.vue';
  import DetailInfoCard from '@/components/DetailInfoCard.vue';
  import SearchControl from '@/components/SearchControl.vue';
  import type { VersionComponentResp } from '@/gen/proto/orbit/v1/application/version';

  const props = withDefaults(
    defineProps<{
      versionId: string;
      components: VersionComponentResp[];
      editable: boolean;
      disabled: boolean;
      loading?: boolean;
    }>(),
    {
      loading: false,
    }
  );

  const emit = defineEmits<{
    add: [];
    edit: [component: VersionComponentResp];
    delete: [component: VersionComponentResp];
    search: [];
  }>();

  const { t } = useI18n();
  const searchText = ref('');
  const appliedSearch = ref('');

  const filteredComponents = computed(() => {
    const keyword = appliedSearch.value.trim().toLowerCase();
    if (!keyword) {
      return props.components;
    }
    return props.components.filter(
      (component) =>
        component.name.toLowerCase().includes(keyword) ||
        component.image.toLowerCase().includes(keyword) ||
        (component.artifact_name || '').toLowerCase().includes(keyword)
    );
  });

  function handleSearch() {
    appliedSearch.value = searchText.value;
    emit('search');
  }
</script>
