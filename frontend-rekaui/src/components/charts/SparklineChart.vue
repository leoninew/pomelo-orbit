<!--
	迷你折线图组件
	用于显示趋势数据的简单折线图
	@example
	<SparklineChart :data="[10, 20, 15, 30, 25]" />
-->
<template>
	<svg class="h-full w-full" viewBox="0 0 100 30" preserveAspectRatio="none">
		<path
			:d="pathData"
			fill="none"
			stroke="currentColor"
			stroke-width="2"
			stroke-linecap="round"
			stroke-linejoin="round"
			class="text-blue-500 dark:text-blue-400"
		/>
	</svg>
</template>

<script setup lang="ts">
	import { computed } from 'vue';

	const props = defineProps<{
		data: number[]
	}>();

	const pathData = computed(() => {
		if (!props.data || props.data.length === 0) {return '';}
		
		// 单点数据居中显示
		if (props.data.length === 1) {
			return 'M50,15';
		}

		const max = Math.max(...props.data);
		const min = Math.min(...props.data);
		const range = max - min || 1;

		const points = props.data.map((value, index) => {
			const x = (index / (props.data.length - 1)) * 100;
			const y = 30 - ((value - min) / range) * 25;
			return `${x},${y}`;
		});

		return `M${points.join(' L')}`;
	});
</script>
