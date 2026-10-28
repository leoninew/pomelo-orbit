<template>
  <span :class="badgeClass">
    <component
      :is="statusIcon"
      v-if="variant === 'status' && statusIcon"
      class="size-4 shrink-0"
      aria-hidden="true"
    />
    <span
      v-else-if="variant === 'status' || variant === 'category'"
      class="size-1.5 shrink-0 rounded-full bg-current"
      aria-hidden="true"
    />
    <slot />
  </span>
</template>

<script lang="ts">
  export type BadgeTone = 'default' | 'primary' | 'success' | 'error' | 'warning' | 'info';
</script>

<script setup lang="ts">
  import { CircleAlert, CircleCheck, CircleX } from '@lucide/vue';
  import { computed } from 'vue';

  interface Props {
    variant?: 'default' | 'status' | 'category' | 'pill';
    tone?: BadgeTone;
  }

  const props = withDefaults(defineProps<Props>(), {
    variant: 'default',
    tone: 'default',
  });

  const statusIcon = computed(() => {
    if (props.tone === 'success') {
      return CircleCheck;
    }
    if (props.tone === 'error') {
      return CircleX;
    }
    if (props.tone === 'warning') {
      return CircleAlert;
    }
    return null;
  });

  const badgeClass = computed(() => {
    const base =
      'inline-flex items-center gap-1.5 whitespace-nowrap rounded py-0.5 text-xs leading-5';

    const variants = {
      default: 'px-2',
      status: '',
      category: '',
      pill: 'rounded-full px-2',
    };

    const tones: Record<BadgeTone, string> = {
      default: 'text-muted-foreground',
      primary: 'text-primary',
      success: 'text-emerald-700 dark:text-emerald-300',
      error: 'text-red-700 dark:text-red-300',
      warning: 'text-amber-700 dark:text-amber-300',
      info: 'text-blue-700 dark:text-blue-300',
    };

    const backgrounds: Record<BadgeTone, string> = {
      default: 'bg-muted',
      primary: 'bg-primary/10',
      success: 'bg-emerald-50 dark:bg-emerald-500/10',
      error: 'bg-red-50 dark:bg-red-500/10',
      warning: 'bg-amber-50 dark:bg-amber-500/10',
      info: 'bg-blue-50 dark:bg-blue-500/10',
    };

    return `${base} ${variants[props.variant]} ${tones[props.tone]} ${props.variant === 'status' || props.variant === 'category' ? '' : backgrounds[props.tone]}`;
  });
</script>
