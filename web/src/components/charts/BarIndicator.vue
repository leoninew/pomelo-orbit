<!--
	条形指示器组件
	用于显示多个条形的状态指示器
	@example
	<BarIndicator :data="[1, 2, 3, 4, 5]" color-theme="indigo" />
-->
<template>
  <div
    v-for="(_value, index) in data"
    :key="index"
    class="h-2 flex-1 rounded-sm"
    :class="getBarClass(index)"
  />
</template>

<script setup lang="ts">
  type ColorTheme = 'indigo' | 'blue' | 'purple' | 'orange';

  const props = withDefaults(
    defineProps<{
      data: number[];
      colorTheme?: ColorTheme;
    }>(),
    {
      colorTheme: 'indigo',
    }
  );

  function getBarClass(index: number) {
    const themeClasses: Record<ColorTheme, string[]> = {
      indigo: [
        'bg-indigo-500 dark:bg-indigo-400',
        'bg-indigo-400 dark:bg-indigo-500',
        'bg-indigo-300 dark:bg-indigo-600',
        'bg-indigo-200 dark:bg-indigo-700',
        'bg-muted',
      ],
      blue: [
        'bg-blue-500 dark:bg-blue-400',
        'bg-blue-400 dark:bg-blue-500',
        'bg-blue-300 dark:bg-blue-600',
        'bg-blue-200 dark:bg-blue-700',
        'bg-muted',
      ],
      purple: [
        'bg-purple-500 dark:bg-purple-400',
        'bg-purple-400 dark:bg-purple-500',
        'bg-purple-300 dark:bg-purple-600',
        'bg-purple-200 dark:bg-purple-700',
        'bg-muted',
      ],
      orange: [
        'bg-orange-500 dark:bg-orange-400',
        'bg-orange-400 dark:bg-orange-500',
        'bg-orange-300 dark:bg-orange-600',
        'bg-orange-200 dark:bg-orange-700',
        'bg-muted',
      ],
    };

    const classes = themeClasses[props.colorTheme];
    return classes[index] || 'bg-muted';
  }
</script>
