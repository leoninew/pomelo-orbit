<!--
	进度条组件
	显示带百分比和数值的进度条
	@example
	<ProgressBar :data="{ current: 80, total: 100, percentage: 80 }" />
-->
<template>
  <div class="space-y-1.5">
    <div class="h-1.5 w-full overflow-hidden rounded-full bg-muted">
      <div
        class="h-full rounded-full bg-purple-500 transition-all duration-500 dark:bg-purple-400"
        :style="{ width: `${data.percentage}%` }"
      />
    </div>
    <div class="flex justify-between text-sm text-muted-foreground">
      <span>成功率 {{ data.percentage }}%</span>
      <span>{{ data.current }}/{{ data.total }}</span>
    </div>
  </div>
</template>

<script setup lang="ts">
  defineProps<{
    data: {
      current: number;
      total: number;
      percentage: number;
    };
  }>();
</script>
