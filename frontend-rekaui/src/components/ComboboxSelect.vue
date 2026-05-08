<script setup lang="ts">
import { Check, ChevronDown, X } from 'lucide-vue-next'
import { computed } from 'vue'
import {
	ComboboxAnchor,
	ComboboxContent,
	ComboboxEmpty,
	ComboboxInput,
	ComboboxItem,
	ComboboxItemIndicator,
	ComboboxPortal,
	ComboboxRoot,
	ComboboxTrigger
} from 'reka-ui'

export type ComboboxOptionValue = string | number

export interface ComboboxOption {
	value: ComboboxOptionValue
	label: string
	description?: string
	disabled?: boolean
}

const props = withDefaults(
	defineProps<{
		modelValue?: ComboboxOptionValue
		options: ComboboxOption[]
		placeholder?: string
		disabled?: boolean
		emptyText?: string
		portal?: boolean
		widthClass?: string
	}>(),
	{
		placeholder: '请选择',
		disabled: false,
		emptyText: '暂无数据',
		portal: true,
		widthClass: 'w-full'
	}
)

const emit = defineEmits<{
	'update:modelValue': [value: ComboboxOptionValue]
}>()

const selectableOptions = computed(() => props.options.filter((item) => item.value !== ''))

function displayValue(value: unknown) {
	const option = selectableOptions.value.find((item) => item.value === value)
	return option?.label ?? ''
}

function hasValue(value: unknown) {
	return value !== undefined && value !== null && value !== ''
}
</script>

<template>
	<ComboboxRoot
		:model-value="modelValue"
		:disabled="disabled"
		open-on-click
		open-on-focus
		@update:model-value="emit('update:modelValue', $event as ComboboxOptionValue)"
	>
		<ComboboxAnchor
			class="app-combobox-anchor"
			:class="[widthClass, disabled ? 'cursor-not-allowed opacity-60' : '']"
		>
			<ComboboxInput
				:display-value="displayValue"
				:placeholder="placeholder"
				:disabled="disabled"
				class="min-w-0 grow bg-transparent outline-none placeholder:text-muted-foreground disabled:cursor-not-allowed"
			/>
			<button
				v-if="hasValue(modelValue)"
				type="button"
				aria-label="清空选择"
				class="text-muted-foreground transition-colors hover:text-foreground disabled:cursor-not-allowed"
				:disabled="disabled"
				@click.stop="emit('update:modelValue', '')"
			>
				<X class="size-3.5" />
			</button>
			<ComboboxTrigger as-child>
				<button
					type="button"
					class="text-muted-foreground transition-colors hover:text-foreground disabled:cursor-not-allowed"
					:disabled="disabled"
				>
					<ChevronDown class="size-4" />
				</button>
			</ComboboxTrigger>
		</ComboboxAnchor>
		<ComboboxPortal :disabled="!portal">
			<ComboboxContent
				position="popper"
				align="start"
				class="app-popover-content w-[var(--reka-combobox-trigger-width)] overflow-y-auto"
				:side-offset="4"
			>
				<ComboboxEmpty class="px-3 py-2 text-sm text-muted-foreground">
					{{ emptyText }}
				</ComboboxEmpty>
				<ComboboxItem
					v-for="option in selectableOptions"
					:key="String(option.value)"
					:value="option.value"
					:text-value="option.label"
					:disabled="option.disabled"
					class="app-option-item"
				>
					<span class="min-w-0">
						<span class="block truncate">{{ option.label }}</span>
						<span v-if="option.description" class="block truncate text-xs text-muted-foreground">
							{{ option.description }}
						</span>
					</span>
					<ComboboxItemIndicator>
						<Check class="size-4 text-primary" />
					</ComboboxItemIndicator>
				</ComboboxItem>
			</ComboboxContent>
		</ComboboxPortal>
	</ComboboxRoot>
</template>
