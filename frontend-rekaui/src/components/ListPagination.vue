<template>
	<div
		v-if="totalPages > 0"
		class="overflow-x-auto"
		:class="standalone ? 'py-1' : 'border-t border-border px-4 py-2 sm:px-6'"
	>
		<div class="flex min-w-max items-center justify-between gap-6">
			<div class="text-sm text-foreground">共 {{ total }} 条</div>
			<div class="flex items-center gap-2">
				<SelectControl
					:model-value="pageSize"
					:options="pageSizeSelectOptions"
					width-class="h-8 w-28"
					@update:model-value="emit('change-page-size', Number($event))"
				/>
				<button
					class="flex size-8 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="current <= 1"
					@click="goPage(current - 1)"
				>
					<ChevronLeft class="size-3.5" />
				</button>
				<button
					v-for="page in totalPages"
					:key="page"
					class="flex size-8 items-center justify-center rounded-md border text-sm transition-colors"
					:class="
						page === current
							? 'border-primary bg-primary text-primary-foreground'
							: 'border-border bg-background text-foreground hover:bg-muted/50'
					"
					@click="goPage(page)"
				>
					{{ page }}
				</button>
				<button
					class="flex size-8 items-center justify-center rounded-md border border-border bg-background text-muted-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="current >= totalPages"
					@click="goPage(current + 1)"
				>
					<ChevronRight class="size-3.5" />
				</button>
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
	import { ChevronLeft, ChevronRight } from 'lucide-vue-next';
	import { computed } from 'vue';
	import SelectControl from '@/components/SelectControl.vue';

	const props = withDefaults(
		defineProps<{
			current: number
			pageSize: number
			total: number
			totalPages: number
			pageSizeOptions?: number[]
			standalone?: boolean
		}>(),
		{
			pageSizeOptions: () => [10, 20, 50],
			standalone: false,
		}
	);

	const emit = defineEmits<{
		'change-page': [page: number]
		'change-page-size': [pageSize: number]
	}>();

	function goPage(page: number) {
		if (page < 1 || page > props.totalPages || page === props.current) {
			return;
		}
		emit('change-page', page);
	}

	const pageSizeSelectOptions = computed(() =>
		props.pageSizeOptions.map((size) => ({
			value: size,
			label: `${size} 条/页`,
		}))
	);
</script>
