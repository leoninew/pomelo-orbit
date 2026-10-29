<template>
  <div
    class="pipeline-stage-node min-w-[160px] cursor-pointer rounded-lg border border-border bg-card p-4 transition-colors hover:border-primary"
    :class="isRunning ? 'animate-pulse motion-reduce:animate-none' : ''"
    :data-selected="data.selected ? '' : undefined"
    @click="handleClick"
  >
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <h4 class="text-base font-normal text-foreground">{{ stage.name }}</h4>
        <span
          v-if="status"
          class="inline-block h-2 w-2 rounded-full"
          :style="{ backgroundColor: textColor }"
        ></span>
      </div>
      <div class="text-sm text-muted-foreground">
        <p>{{ stage.image }}</p>
      </div>
      <div v-if="status" class="text-xs" :style="{ color: textColor }">
        {{ statusText }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';
  import type { SnapshotStageResp } from '@/gen/proto/orbit/v1/pipeline/snapshot';
  import { statusColor } from '@/utils/status';

  interface Props {
    data: {
      stage: SnapshotStageResp;
      status?: string;
      readonly?: boolean;
      selected?: boolean;
    };
  }

  const props = defineProps<Props>();

  const emit = defineEmits(['click']);

  const stage = computed(() => props.data.stage);
  const status = computed(() => props.data.status);

  const statusText = computed(() => (status.value ? status.value : ''));
  const textColor = computed(() => statusColor(status.value));
  const isRunning = computed(() => status.value === 'running');

  function handleClick() {
    emit('click');
  }
</script>
