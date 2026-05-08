<script setup lang="ts">
import { Search, X } from 'lucide-vue-next'

withDefaults(defineProps<{
	modelValue: string
	placeholder?: string
	disabled?: boolean
	loading?: boolean
}>(), {
	placeholder: '搜索',
	disabled: false,
	loading: false
})

const emit = defineEmits<{
	'update:modelValue': [value: string]
	search: []
}>()

function clearSearch() {
	emit('update:modelValue', '')
	emit('search')
}
</script>

<template>
	<div class="flex w-full min-w-0 max-w-md items-center">
		<div class="relative min-w-0 flex-1">
			<Search class="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
			<input
				:value="modelValue"
				type="text"
				:placeholder="placeholder"
				class="h-10 w-full rounded-l-md border border-r-0 border-input bg-background py-2 pl-10 pr-9 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
				@input="emit('update:modelValue', ($event.target as HTMLInputElement).value)"
				@keydown.enter="emit('search')"
			/>
			<button
				v-if="modelValue"
				type="button"
				aria-label="清空搜索"
				class="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
				@click="clearSearch"
			>
				<X class="size-4" />
			</button>
		</div>
		<button
			type="button"
			class="flex h-10 min-w-20 items-center justify-center rounded-r-md border border-primary bg-primary px-5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
			:disabled="disabled || loading"
			@click="emit('search')"
		>
			<span
				v-if="loading"
				class="inline-block size-4 animate-spin rounded-full border-2 border-primary-foreground/30 border-t-primary-foreground"
			/>
			<span v-else>搜索</span>
		</button>
	</div>
</template>
