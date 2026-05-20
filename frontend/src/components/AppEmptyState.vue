<template>
	<div class="flex flex-col items-center justify-center text-muted-foreground" :class="paddingClass">
		<Inbox :class="iconClass" />
		<p class="mt-2 text-sm">{{ message }}</p>
	</div>
</template>

<script setup lang="ts">
	import { computed } from 'vue';
	import { Inbox } from 'lucide-vue-next';

	const props = withDefaults(
		defineProps<{
			message?: string
			size?: 'default' | 'compact'
		}>(),
		{
			message: '暂无数据',
			size: 'default',
		}
	);

	const paddingClass = computed(() => (props.size === 'compact' ? 'py-8' : 'py-16'));
	const iconClass = computed(() => (props.size === 'compact' ? 'size-8' : 'size-12'));
</script>
