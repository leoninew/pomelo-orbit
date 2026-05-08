<script setup lang="ts">
import { Check, ChevronDown } from 'lucide-vue-next'
import {
	SelectContent,
	SelectItem,
	SelectItemIndicator,
	SelectItemText,
	SelectPortal,
	SelectRoot,
	SelectTrigger,
	SelectValue,
	SelectViewport
} from 'reka-ui'

export type SelectOptionValue = string | number

export interface SelectOption {
	value: SelectOptionValue
	label: string
	disabled?: boolean
}

withDefaults(
	defineProps<{
		modelValue?: SelectOptionValue
		options: SelectOption[]
		placeholder?: string
		disabled?: boolean
		widthClass?: string
	}>(),
	{
		placeholder: '请选择',
		disabled: false,
		widthClass: 'w-full'
	}
)

const emit = defineEmits<{
	'update:modelValue': [value: SelectOptionValue]
}>()
</script>

<template>
	<SelectRoot
		:model-value="modelValue"
		:disabled="disabled"
		@update:model-value="emit('update:modelValue', $event as SelectOptionValue)"
	>
		<SelectTrigger
			class="flex h-10 items-center justify-between gap-2 rounded-md border border-input bg-background px-3 text-sm text-foreground outline-none transition-colors hover:bg-accent/50 focus:border-ring focus:ring-2 focus:ring-ring/20 disabled:cursor-not-allowed disabled:opacity-60"
			:class="widthClass"
		>
			<SelectValue :placeholder="placeholder" />
			<ChevronDown class="size-4 shrink-0 text-muted-foreground" />
		</SelectTrigger>
		<SelectPortal>
			<SelectContent
				position="popper"
				class="z-[60] max-h-64 min-w-[var(--reka-select-trigger-width)] overflow-hidden rounded-md border border-border bg-popover p-1 text-popover-foreground shadow-lg outline-none"
				:side-offset="4"
			>
				<SelectViewport>
					<SelectItem
						v-for="option in options"
						:key="String(option.value)"
						:value="option.value"
						:text-value="option.label"
						:disabled="option.disabled"
						class="flex cursor-pointer items-center justify-between gap-3 rounded-sm px-3 py-2 text-sm text-foreground outline-none transition-colors hover:bg-accent/50 data-[disabled]:cursor-not-allowed data-[disabled]:opacity-50 data-[highlighted]:bg-accent/50"
					>
						<SelectItemText>{{ option.label }}</SelectItemText>
						<SelectItemIndicator>
							<Check class="size-4 text-primary" />
						</SelectItemIndicator>
					</SelectItem>
				</SelectViewport>
			</SelectContent>
		</SelectPortal>
	</SelectRoot>
</template>
