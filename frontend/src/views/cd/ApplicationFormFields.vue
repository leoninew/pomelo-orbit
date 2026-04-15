<template>
	<div class="flex flex-col gap-3">
		<fieldset class="fieldset">
			<legend class="fieldset-legend">应用名称</legend>
			<input
				:value="form.name"
				type="text"
				class="input w-full"
				:class="{ 'input-error': errors.name }"
				placeholder="例如: my-app"
				@input="emit('update:form', { ...form, name: ($event.target as HTMLInputElement).value })"
			/>
			<p v-if="errors.name" class="fieldset-label text-error">{{ errors.name }}</p>
		</fieldset>

		<fieldset class="fieldset">
			<legend class="fieldset-legend">应用编码</legend>
			<input
				:value="form.code"
				type="text"
				class="input w-full"
				:class="{ 'input-error': errors.code }"
				placeholder="小写字母、数字和连字符"
				@input="emit('update:form', { ...form, code: ($event.target as HTMLInputElement).value })"
			/>
			<p v-if="errors.code" class="fieldset-label text-error">{{ errors.code }}</p>
		</fieldset>

		<fieldset class="fieldset">
			<legend class="fieldset-legend">镜像拉取策略</legend>
			<select
				:value="form.image_pull_policy"
				class="select w-full"
				@change="
					emit('update:form', {
						...form,
						image_pull_policy: ($event.target as HTMLSelectElement).value,
					})
				"
			>
				<option value="always">always</option>
				<option value="missing">missing</option>
				<option value="never">never</option>
			</select>
		</fieldset>

		<fieldset class="fieldset">
			<legend class="fieldset-legend">路由托管</legend>
			<label class="flex items-center gap-3 cursor-pointer">
				<input
					type="checkbox"
					class="toggle toggle-sm"
					:checked="form.route_managed"
					@change="
						emit('update:form', {
							...form,
							route_managed: ($event.target as HTMLInputElement).checked,
						})
					"
				/>
				<span class="text-sm text-base-content/70">启用后，部署时将自动生成 Traefik 路由配置</span>
			</label>
		</fieldset>
	</div>
</template>

<script setup lang="ts">
interface FormData {
	name: string
	code: string
	image_pull_policy: string
	route_managed: boolean
}

defineProps<{
	form: FormData
	errors: { name: string; code: string }
}>();

const emit = defineEmits<{
	'update:form': [value: FormData]
}>();
</script>
