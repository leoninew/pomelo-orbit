<template>
  <div class="flex shrink-0 items-center gap-1">
    <span
      v-if="state?.gap || state?.truncated"
      class="inline-flex size-8 items-center justify-center text-muted-foreground"
      :title="historyMessage"
      role="img"
      :aria-label="historyMessage"
    >
      <Info class="size-4" />
    </span>
    <button
      type="button"
      class="app-icon-button size-8"
      :title="t('logs.find')"
      :aria-label="t('logs.find')"
      @click="emit('find')"
    >
      <Search class="size-4" />
    </button>
    <button
      type="button"
      class="app-icon-button size-8"
      :class="state?.follow ? 'text-primary' : 'text-muted-foreground'"
      :title="t(state?.follow ? 'logs.stopFollowing' : 'logs.follow')"
      :aria-label="t(state?.follow ? 'logs.stopFollowing' : 'logs.follow')"
      :aria-pressed="state?.follow ?? true"
      :disabled="!state"
      @click="emit('follow')"
    >
      <ArrowDownToLine v-if="state?.follow" class="size-4" />
      <Pause v-else class="size-4" />
    </button>
  </div>
</template>

<script setup lang="ts">
  import { ArrowDownToLine, Info, Pause, Search } from '@lucide/vue';
  import { computed } from 'vue';
  import { useI18n } from 'vue-i18n';
  import type { LogState } from '@/composables/useLogStream';
  const props = defineProps<{ state?: LogState }>();
  const emit = defineEmits<{ find: []; follow: [] }>();
  const { t } = useI18n();
  const historyMessage = computed(() => (props.state?.gap ? t('logs.gap') : t('logs.truncated')));
</script>
