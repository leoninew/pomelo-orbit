<template>
  <LogDrawer
    :open="open"
    :title="target.title"
    :state="state"
    @update:open="emit('update:open', $event)"
  />
</template>

<script setup lang="ts">
  import { computed } from 'vue';
  import LogDrawer from '@/components/LogDrawer.vue';
  import { useLogStream } from '@/composables/useLogStream';
  import { useProjectStore } from '@/stores/project';
  import type { RuntimeContainerLogTarget } from './runtimeContainerLogs';
  const props = defineProps<{ open: boolean; target: RuntimeContainerLogTarget }>();
  const emit = defineEmits<{ 'update:open': [open: boolean] }>();
  const project = useProjectStore();
  const resource = computed(() =>
    project.activeProjectId
      ? {
          projectId: project.activeProjectId,
          path: `/api/application/${props.target.applicationId}/log/stream`,
          params: {
            service_id: props.target.serviceId,
            ...(props.target.component ? { component: props.target.component } : {}),
          },
          kind: 'container' as const,
        }
      : undefined
  );
  const { state } = useLogStream(resource, () => props.open);
</script>
