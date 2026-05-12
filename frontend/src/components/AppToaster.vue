<template>
	<ToastProvider :label="t('toast.providerLabel')" swipe-direction="right" :duration="3000">
		<ToastRoot
			v-for="toast in toasts"
			:key="toast.id"
			class="app-toast-root"
			:class="toastConfig[toast.type].className"
			:duration="toast.duration"
			@update:open="handleOpenChange(toast.id, $event)"
		>
			<component :is="toastConfig[toast.type].icon" class="mt-0.5 size-5 shrink-0" />
			<div class="min-w-0 flex-1">
				<ToastTitle class="text-sm font-medium">
					{{ toastConfig[toast.type].title }}
				</ToastTitle>
				<ToastDescription class="mt-1 text-sm leading-5 opacity-90">
					{{ toast.text }}
				</ToastDescription>
			</div>
			<ToastClose as-child>
				<button
					type="button"
					class="-mr-1 rounded p-1 opacity-70 transition-opacity hover:opacity-100"
					:aria-label="t('toast.close')"
				>
					<X class="size-4" />
				</button>
			</ToastClose>
		</ToastRoot>
		<ToastViewport
			class="fixed bottom-4 left-4 right-4 z-50 flex flex-col gap-2 outline-none md:left-auto md:right-4 md:w-96"
		/>
	</ToastProvider>
</template>

<script setup lang="ts">
	import { AlertCircle, AlertTriangle, CheckCircle2, Info, X } from 'lucide-vue-next';
	import {
		ToastClose,
		ToastDescription,
		ToastProvider,
		ToastRoot,
		ToastTitle,
		ToastViewport,
	} from 'reka-ui';
	import { computed, type Component } from 'vue';
	import { useI18n } from 'vue-i18n';
	import { useToast, type ToastType } from '@/composables/useToast';

	const { toasts, remove } = useToast();
	const { t } = useI18n({ useScope: 'global' });

	const toastStyles = {
		success: {
			light: 'border-green-200 bg-green-50 text-green-800',
			dark: 'dark:border-green-500/30 dark:bg-green-950/60 dark:text-green-100',
		},
		error: {
			light: 'border-red-200 bg-red-50 text-red-800',
			dark: 'dark:border-red-500/30 dark:bg-red-950/60 dark:text-red-100',
		},
		warning: {
			light: 'border-amber-200 bg-amber-50 text-amber-800',
			dark: 'dark:border-amber-500/30 dark:bg-amber-950/60 dark:text-amber-100',
		},
		info: {
			light: 'border-blue-200 bg-blue-50 text-blue-800',
			dark: 'dark:border-blue-500/30 dark:bg-blue-950/60 dark:text-blue-100',
		},
	} as const;

	const toastConfig = computed<
		Record<
			ToastType,
			{
				title: string
				icon: Component
				className: string
			}
		>
	>(() => ({
		success: {
			title: t('toast.success'),
			icon: CheckCircle2,
			className: `${toastStyles.success.light} ${toastStyles.success.dark}`,
		},
		error: {
			title: t('toast.error'),
			icon: AlertCircle,
			className: `${toastStyles.error.light} ${toastStyles.error.dark}`,
		},
		warning: {
			title: t('toast.warning'),
			icon: AlertTriangle,
			className: `${toastStyles.warning.light} ${toastStyles.warning.dark}`,
		},
		info: {
			title: t('toast.info'),
			icon: Info,
			className: `${toastStyles.info.light} ${toastStyles.info.dark}`,
		},
	}));

	function handleOpenChange(id: number, open: boolean) {
		if (!open) {
			remove(id);
		}
	}
</script>
