<template>
	<div class="space-y-4">
		<!-- 应用名称 -->
		<div class="space-y-1.5">
			<label class="app-field-label block">
				应用名称
				<span class="text-destructive">*</span>
			</label>
			<input
				:value="form.name"
				type="text"
				placeholder="输入应用名称"
				class="app-input"
				:class="errors.name ? 'app-input-error' : ''"
				@input="updateField('name', ($event.target as HTMLInputElement).value)"
			/>
			<p v-if="errors.name" class="app-field-error mt-1 text-xs">{{ errors.name }}</p>
		</div>

		<!-- 应用代码 -->
		<div class="space-y-1.5">
			<label class="app-field-label block">
				应用代码
				<span class="text-destructive">*</span>
			</label>
			<input
				:value="form.code"
				type="text"
				placeholder="输入应用代码（以小写字母开头，可含数字和连字符）"
				class="app-input"
				:class="errors.code ? 'app-input-error' : ''"
				@input="updateField('code', ($event.target as HTMLInputElement).value)"
			/>
			<p v-if="errors.code" class="app-field-error mt-1 text-xs">{{ errors.code }}</p>
			<p class="app-field-hint">应用代码用于生成工作目录，创建后不可修改</p>
		</div>

		<!-- 镜像拉取策略 -->
		<div class="space-y-1.5">
			<label class="app-field-label block">镜像拉取策略</label>
			<SelectControl
				:model-value="form.image_pull_policy"
				:options="imagePullPolicyOptions"
				placeholder="选择镜像拉取策略"
				@update:model-value="updateField('image_pull_policy', String($event))"
			/>
		</div>

		<!-- 路由管理 -->
		<div class="flex items-center gap-2">
			<input
				id="route_managed"
				:checked="form.route_managed"
				type="checkbox"
				class="app-checkbox"
				@change="updateField('route_managed', ($event.target as HTMLInputElement).checked)"
			/>
			<label for="route_managed" class="text-sm font-medium text-foreground">启用路由管理</label>
		</div>
		<p class="app-field-hint">启用后可为应用配置 HTTP 路由，自动生成 Traefik 配置</p>
	</div>
</template>

<script setup lang="ts">
	import SelectControl from '@/components/SelectControl.vue';
	import type { ApplicationFormState } from '@/types/cd/application';

	const props = defineProps<{
		form: ApplicationFormState
		errors: { name: string; code: string }
	}>();

	const emit = defineEmits<{
		'update:form': [value: ApplicationFormState]
	}>();

	function updateField<K extends keyof ApplicationFormState>(
		field: K,
		value: ApplicationFormState[K]
	) {
		emit('update:form', { ...props.form, [field]: value });
	}

	const imagePullPolicyOptions = [
		{ value: 'missing', label: '缺失时拉取 (missing)' },
		{ value: 'always', label: '总是拉取 (always)' },
		{ value: 'never', label: '从不拉取 (never)' },
	];
</script>
