<template>
  <div
    class="min-w-[160px] cursor-pointer rounded-lg border-2 bg-card p-3 shadow-sm transition-all hover:shadow-md"
    :class="[data.selected ? 'border-primary' : 'border-border', isRunning ? 'animate-pulse' : '']"
    @click="handleClick"
  >
    <div class="space-y-2">
      <div class="flex items-center justify-between">
        <h4 class="text-sm font-semibold text-foreground">{{ stage.name }}</h4>
        <span
          v-if="status"
          class="inline-block h-2 w-2 rounded-full"
          :style="{ backgroundColor: textColor }"
        ></span>
      </div>
      <div class="text-xs text-muted-foreground">
        <p>{{ stage.image }}</p>
      </div>
      <div v-if="status" class="text-xs font-medium" :style="{ color: textColor }">
        {{ statusText }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue';
  import type { SnapshotStageResp } from '@/gen/proto/orbit/api/v1/snapshot';
  import { statusColor } from '@/utils/status';
  import type { TaskStatus } from '@/types/common';

  interface Props {
    data: {
      stage: SnapshotStageResp;
      status?: TaskStatus;
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
