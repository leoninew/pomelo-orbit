<template>
  <div class="@container flex min-w-0 flex-col gap-4">
    <DetailInfoCard
      :title="
        component
          ? t('application.componentDetail.sections.basic')
          : t('application.detail.sections.basicInfo')
      "
      actions-class="gap-4"
    >
      <template #actions>
        <button
          v-if="editable"
          type="button"
          class="app-link"
          :disabled="busy"
          :aria-label="t('application.detail.explorer.editBasic')"
          @click="emit('action', 'edit')"
        >
          {{ t('common.edit') }}
        </button>
        <template v-if="component">
          <router-link :to="`/version/${version.id}/component/${component.id}`" class="app-link">
            {{ t('application.detail.explorer.openDetails') }}
          </router-link>
          <button
            v-if="editable"
            type="button"
            class="app-link-danger"
            :disabled="busy"
            @click="emit('action', 'delete')"
          >
            {{ t('common.delete') }}
          </button>
        </template>
        <template v-else>
          <button
            type="button"
            class="app-link"
            :disabled="busy"
            @click="emit('action', 'preview')"
          >
            {{ t('application.detail.actions.preview') }}
          </button>
          <router-link :to="`/version/${version.id}`" class="app-link">
            {{ t('application.detail.explorer.openDetails') }}
          </router-link>
          <AppActionMenu :options="moreActions" @select="emit('action', $event)">
            <template #trigger>
              <button
                type="button"
                class="app-link inline-flex items-center gap-1"
                :disabled="busy"
              >
                {{ t('application.detail.explorer.more') }}
                <ChevronDown class="size-4" />
              </button>
            </template>
          </AppActionMenu>
        </template>
      </template>
      <dl
        v-if="component"
        class="app-detail-fields grid grid-cols-1 gap-x-8 gap-y-3 @3xl:grid-cols-2"
      >
        <div>
          <dt>{{ t('application.detail.fields.component') }}</dt>
          <dd class="break-all text-foreground">{{ component.name }}</dd>
        </div>
        <div class="col-span-full">
          <dt>{{ t('application.detail.fields.image') }}</dt>
          <dd class="break-all text-foreground">{{ component.image }}</dd>
        </div>
        <div>
          <dt>{{ t('application.componentDetail.fields.pullPolicy') }}</dt>
          <dd class="text-foreground">{{ component.pull_policy }}</dd>
        </div>
        <div>
          <dt>{{ t('application.componentDetail.fields.restartPolicy') }}</dt>
          <dd class="text-foreground">{{ component.restart_policy || t('common.notSet') }}</dd>
        </div>
        <div class="col-span-full">
          <dt>{{ t('application.componentDetail.fields.containerEntrypoint') }}</dt>
          <dd v-if="component.entrypoint" class="app-code-content whitespace-pre-wrap break-all">
            {{ component.entrypoint }}
          </dd>
          <dd v-else>{{ t('common.notSet') }}</dd>
        </div>
        <div class="col-span-full">
          <dt>{{ t('application.componentDetail.fields.command') }}</dt>
          <dd v-if="component.command" class="app-code-content whitespace-pre-wrap break-all">
            {{ component.command }}
          </dd>
          <dd v-else>{{ t('common.notSet') }}</dd>
        </div>
        <div v-if="component.artifact_local_image_sha256" class="col-span-full">
          <dt>{{ t('application.componentDetail.fields.imageSha256') }}</dt>
          <dd class="break-all text-foreground">{{ component.artifact_local_image_sha256 }}</dd>
        </div>
        <div>
          <dt>{{ t('common.createdAt') }}</dt>
          <dd class="text-muted-foreground">{{ formatTime(component.created_at) }}</dd>
        </div>
        <div>
          <dt>{{ t('common.updatedAt') }}</dt>
          <dd class="text-muted-foreground">{{ formatTime(component.updated_at) }}</dd>
        </div>
      </dl>
      <dl v-else class="app-detail-fields grid grid-cols-1 gap-x-8 gap-y-3 @3xl:grid-cols-2">
        <div>
          <dt>{{ t('application.detail.fields.versionLabel') }}</dt>
          <dd class="break-all text-foreground">{{ version.label }}</dd>
        </div>
        <div>
          <dt>{{ t('common.status') }}</dt>
          <dd>
            <AppBadge variant="pill" :tone="versionStatusTone(version.status)">
              {{ version.status }}
            </AppBadge>
          </dd>
        </div>
        <div>
          <dt>{{ t('application.detail.fields.components') }}</dt>
          <dd class="text-foreground">{{ version.components.length }}</dd>
        </div>
        <div class="col-span-full">
          <dt>{{ t('application.detail.fields.note') }}</dt>
          <dd class="whitespace-pre-wrap break-words text-foreground">
            {{ version.note || t('common.notSet') }}
          </dd>
        </div>
        <div>
          <dt>{{ t('common.createdAt') }}</dt>
          <dd class="text-muted-foreground">{{ formatTime(version.created_at) }}</dd>
        </div>
        <div>
          <dt>{{ t('common.updatedAt') }}</dt>
          <dd class="text-muted-foreground">{{ formatTime(version.updated_at) }}</dd>
        </div>
      </dl>
    </DetailInfoCard>
    <DetailInfoCard v-if="component" :title="t('application.componentDetail.sections.identity')">
      <template v-if="editable" #actions>
        <button
          type="button"
          class="app-link"
          :disabled="busy"
          @click="emit('action', 'edit-identity')"
        >
          {{ t('common.edit') }}
        </button>
      </template>
      <dl class="app-detail-fields grid grid-cols-1 gap-x-8 gap-y-3 @3xl:grid-cols-2">
        <div>
          <dt>{{ t('application.componentDetail.fields.user') }}</dt>
          <dd class="break-words text-foreground">{{ component.user || t('common.notSet') }}</dd>
        </div>
        <div>
          <dt>{{ t('application.componentDetail.fields.groupAdd') }}</dt>
          <dd class="break-words text-foreground">
            {{ component.group_add.join(', ') || t('common.notSet') }}
          </dd>
        </div>
      </dl>
    </DetailInfoCard>
  </div>
</template>

<script setup lang="ts">
  import { ChevronDown } from '@lucide/vue';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import AppActionMenu from '@/components/ui/AppActionMenu.vue';
  import AppBadge from '@/components/ui/AppBadge.vue';
  import DetailInfoCard from '@/components/ui/DetailInfoCard.vue';
  import type { VersionComponentResp, VersionResp } from '@/gen/proto/orbit/v1/application/version';
  import { versionStatusTone } from '@/utils/status';
  import { formatTime } from '@/utils/time';

  const props = defineProps<{
    version: VersionResp;
    component?: VersionComponentResp;
    busy: boolean;
  }>();
  const emit = defineEmits<{
    action: [action: string];
  }>();
  const { t } = useI18n();
  const editable = computed(() => props.version.status === 'unpublished');
  const moreActions = computed(() => [
    { value: 'fork', label: t('application.detail.actions.fork'), disabled: props.busy },
    ...(editable.value
      ? [{ value: 'publish', label: t('application.detail.actions.publish'), disabled: props.busy }]
      : props.version.status === 'published'
        ? [
            {
              value: 'unpublish',
              label: t('application.detail.actions.unpublish'),
              disabled: props.busy,
            },
          ]
        : []),
    { value: 'delete', label: t('common.delete'), destructive: true, disabled: props.busy },
  ]);
</script>
