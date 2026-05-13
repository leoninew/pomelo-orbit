<template>
	<span :class="badgeClass">
		<slot />
	</span>
</template>

<script setup lang="ts">
	import { computed } from 'vue';
	import type { BadgeTone } from '@/utils/status';

	interface Props {
		variant?: 'default' | 'status' | 'pill'
		tone?: BadgeTone
	}

	const props = withDefaults(defineProps<Props>(), {
		variant: 'default',
		tone: 'default',
	});

	const badgeClass = computed(() => {
		const base =
			'inline-flex items-center whitespace-nowrap rounded border px-2 py-0.5 text-xs font-medium';

		const variants = {
			default: '',
			status: '',
			pill: 'rounded-full',
		};

		const tones: Record<BadgeTone, string> = {
			default: 'border-border bg-muted text-muted-foreground',
			primary: 'border-primary/30 bg-primary/10 text-primary',
			success:
				'border-green-200 bg-green-50 text-green-700 dark:border-green-500/30 dark:bg-green-500/10 dark:text-green-300',
			error:
				'border-red-200 bg-red-50 text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-300',
			warning:
				'border-amber-200 bg-amber-50 text-amber-700 dark:border-amber-500/30 dark:bg-amber-500/10 dark:text-amber-300',
			info: 'border-blue-200 bg-blue-50 text-blue-700 dark:border-blue-500/30 dark:bg-blue-500/10 dark:text-blue-300',
		};

		return `${base} ${variants[props.variant]} ${tones[props.tone]}`;
	});
</script>
