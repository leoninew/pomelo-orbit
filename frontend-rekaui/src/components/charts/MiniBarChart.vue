<!--
	迷你条形图组件
	用于显示简单的条形图数据可视化
	@example
	<MiniBarChart :data="[10, 20, 15, 30]" />
-->
<template>
	<div
		v-for="(value, index) in data"
		:key="index"
		class="flex-1 rounded-t-sm bg-orange-500 transition-all duration-300 hover:bg-orange-600 dark:bg-orange-400 dark:hover:bg-orange-300"
		:style="{ height: `${(value / maxValue) * 100}%` }"
	/>
</template>

<script setup lang="ts">
	import { computed } from 'vue';

	const props = defineProps<{
		data: number[]
	}>();

	const maxValue = computed(() => Math.max(...props.data) || 1);
</script>
