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
			<legend class="fieldset-legend">仓库地址</legend>
			<input
				:value="form.repository_url"
				type="text"
				class="input w-full"
				placeholder="https://github.com/..."
				@input="emit('update:form', { ...form, repository_url: ($event.target as HTMLInputElement).value })"
			/>
		</fieldset>

		<fieldset class="fieldset">
			<legend class="fieldset-legend">部署分支</legend>
			<input
				:value="form.deploy_branches"
				type="text"
				class="input w-full"
				placeholder="master,develop"
				@input="emit('update:form', { ...form, deploy_branches: ($event.target as HTMLInputElement).value })"
			/>
		</fieldset>

		<fieldset class="fieldset">
			<legend class="fieldset-legend">镜像拉取策略</legend>
			<select
				:value="form.image_pull_policy"
				class="select w-full"
				@change="emit('update:form', { ...form, image_pull_policy: ($event.target as HTMLSelectElement).value })"
			>
				<option value="always">always</option>
				<option value="missing">missing</option>
				<option value="never">never</option>
			</select>
		</fieldset>

		<div class="flex flex-col gap-2">
			<label class="flex items-center gap-3 cursor-pointer">
				<input
					:checked="form.auto_deploy"
					type="checkbox"
					class="toggle toggle-primary"
					@change="emit('update:form', { ...form, auto_deploy: ($event.target as HTMLInputElement).checked })"
				/>
				<span class="text-sm">自动部署</span>
			</label>

			<label class="flex items-center gap-3 cursor-pointer">
				<input
					:checked="form.enabled"
					type="checkbox"
					class="toggle toggle-primary"
					@change="emit('update:form', { ...form, enabled: ($event.target as HTMLInputElement).checked })"
				/>
				<span class="text-sm">启用</span>
			</label>
		</div>
	</div>
</template>

<script setup lang="ts">
interface FormData {
	name: string
	code: string
	repository_url: string
	deploy_branches: string
	auto_deploy: boolean
	image_pull_policy: string
	enabled: boolean
}

defineProps<{
	form: FormData
	errors: { name: string; code: string }
}>();

const emit = defineEmits<{
	'update:form': [value: FormData]
}>();
</script>
