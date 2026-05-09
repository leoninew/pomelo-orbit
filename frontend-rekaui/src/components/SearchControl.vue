<template>
	<div class="flex w-full min-w-0 max-w-md items-center">
		<div class="relative min-w-0 flex-1">
			<Search class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
			<input
				:value="modelValue"
				type="text"
				:placeholder="placeholder"
				class="app-input-search rounded-r-none border-r-0"
				:disabled="disabled || loading"
				@input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
				@keydown.enter="emit('search')"
			/>
			<button
				v-if="modelValue"
				type="button"
				aria-label="清空搜索"
				class="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground transition-colors hover:text-foreground disabled:cursor-not-allowed disabled:opacity-50"
				:disabled="disabled || loading"
				@click="clearSearch"
			>
				<X class="size-4" />
			</button>
		</div>
		<button
			type="button"
			class="app-button-primary min-w-20 rounded-l-none px-5"
			:disabled="disabled || loading"
			@click="emit('search')"
		>
			搜索
		</button>
	</div>
</template>

<script setup lang="ts">
	import { Search, X } from 'lucide-vue-next';

	withDefaults(
		defineProps<{
			modelValue: string
			placeholder?: string
			disabled?: boolean
			loading?: boolean
		}>(),
		{
			placeholder: '搜索',
			disabled: false,
			loading: false,
		}
	);

	const emit = defineEmits<{
		'update:modelValue': [value: string]
		search: []
	}>();

	function clearSearch() {
		emit('update:modelValue', '');
		emit('search');
	}
</script>
