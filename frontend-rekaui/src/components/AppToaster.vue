<script setup lang="ts">
import { AlertCircle, AlertTriangle, CheckCircle2, Info, X } from 'lucide-vue-next'
import {
	ToastClose,
	ToastDescription,
	ToastProvider,
	ToastRoot,
	ToastTitle,
	ToastViewport
} from 'reka-ui'
import type { Component } from 'vue'
import { useToast, type ToastType } from '@/composables/useToast'

const { toasts, remove } = useToast()

const toastConfig: Record<ToastType, {
	title: string
	icon: Component
	className: string
}> = {
	success: {
		title: '成功',
		icon: CheckCircle2,
		className: 'border-green-200 bg-green-50 text-green-800'
	},
	error: {
		title: '错误',
		icon: AlertCircle,
		className: 'border-red-200 bg-red-50 text-red-800'
	},
	warning: {
		title: '提醒',
		icon: AlertTriangle,
		className: 'border-amber-200 bg-amber-50 text-amber-800'
	},
	info: {
		title: '通知',
		icon: Info,
		className: 'border-blue-200 bg-blue-50 text-blue-800'
	}
}

function handleOpenChange(id: number, open: boolean) {
	if (!open) {
		remove(id)
	}
}
</script>

<template>
	<ToastProvider label="通知" swipe-direction="right" :duration="3000">
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
				<button type="button" class="-mr-1 rounded p-1 opacity-70 transition-opacity hover:opacity-100" aria-label="关闭通知">
					<X class="size-4" />
				</button>
			</ToastClose>
		</ToastRoot>
		<ToastViewport
			class="fixed bottom-4 left-4 right-4 z-50 flex flex-col gap-2 outline-none md:left-auto md:right-4 md:w-96"
		/>
	</ToastProvider>
</template>
