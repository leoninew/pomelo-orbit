<script setup lang="ts">
import { computed } from 'vue';
import {
	DialogContent,
	DialogDescription,
	DialogOverlay,
	DialogPortal,
	DialogRoot,
	DialogTitle,
} from 'reka-ui';

const props = withDefaults(
	defineProps<{
		open: boolean
		title: string
		description?: string
		widthClass?: string
		bodyClass?: string
		contentClass?: string
	}>(),
	{
		description: '',
		widthClass: 'w-[min(520px,calc(100vw-32px))]',
		bodyClass: 'space-y-4 px-6 py-4',
		contentClass: '',
	}
);

const emit = defineEmits<{
	'update:open': [open: boolean]
}>();

const openModel = computed({
	get: () => props.open,
	set: (value) => emit('update:open', value),
});
</script>

<template>
	<DialogRoot v-model:open="openModel">
		<DialogPortal>
			<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
			<DialogContent
				:class="[
					'fixed left-1/2 top-1/2 z-50 max-h-[90vh] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-border bg-card shadow-xl outline-none data-[state=open]:animate-contentShow',
					widthClass,
					contentClass,
				]"
			>
				<div class="border-b border-border px-6 py-4">
					<DialogTitle class="text-lg font-semibold text-foreground">
						{{ title }}
					</DialogTitle>
					<DialogDescription v-if="description" class="mt-1 text-sm text-muted-foreground">
						{{ description }}
					</DialogDescription>
					<slot name="description" />
				</div>
				<div :class="bodyClass">
					<slot />
				</div>
				<div v-if="$slots.footer" class="flex justify-end gap-2 border-t border-border px-6 py-4">
					<slot name="footer" />
				</div>
			</DialogContent>
		</DialogPortal>
	</DialogRoot>
</template>
