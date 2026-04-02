<template>
	<label class="form-control w-full">
		<div class="label pb-1"><span class="label-text">应用名称</span></div>
		<input :value="form.name" type="text" class="input input-bordered input-sm" :class="{ 'input-error': errors.name }" placeholder="例如: my-app" @input="emit('update:form', { ...form, name: ($event.target as HTMLInputElement).value })" />
		<div v-if="errors.name" class="label pt-1"><span class="label-text-alt text-error">{{ errors.name }}</span></div>
	</label>

	<label class="form-control w-full">
		<div class="label pb-1"><span class="label-text">应用编码</span></div>
		<input :value="form.code" type="text" class="input input-bordered input-sm" :class="{ 'input-error': errors.code }" placeholder="小写字母、数字和连字符" @input="emit('update:form', { ...form, code: ($event.target as HTMLInputElement).value })" />
		<div v-if="errors.code" class="label pt-1"><span class="label-text-alt text-error">{{ errors.code }}</span></div>
	</label>

	<label class="form-control w-full">
		<div class="label pb-1"><span class="label-text">仓库地址</span></div>
		<input :value="form.repository_url" type="text" class="input input-bordered input-sm" placeholder="https://github.com/..." @input="emit('update:form', { ...form, repository_url: ($event.target as HTMLInputElement).value })" />
	</label>

	<label class="form-control w-full">
		<div class="label pb-1"><span class="label-text">部署分支</span></div>
		<input :value="form.deploy_branches" type="text" class="input input-bordered input-sm" placeholder="master,develop" @input="emit('update:form', { ...form, deploy_branches: ($event.target as HTMLInputElement).value })" />
	</label>

	<label class="form-control w-full">
		<div class="label pb-1"><span class="label-text">镜像拉取策略</span></div>
		<select :value="form.image_pull_policy" class="select select-bordered select-sm" @change="emit('update:form', { ...form, image_pull_policy: ($event.target as HTMLSelectElement).value })">
			<option value="always">always</option>
			<option value="missing">missing</option>
			<option value="never">never</option>
		</select>
	</label>

	<div class="flex items-center gap-6">
		<label class="flex items-center gap-2 cursor-pointer">
			<span class="label-text text-sm">自动部署</span>
			<input :checked="form.auto_deploy" type="checkbox" class="toggle toggle-sm toggle-primary" @change="emit('update:form', { ...form, auto_deploy: ($event.target as HTMLInputElement).checked })" />
		</label>
		<label class="flex items-center gap-2 cursor-pointer">
			<span class="label-text text-sm">启用</span>
			<input :checked="form.enabled" type="checkbox" class="toggle toggle-sm toggle-primary" @change="emit('update:form', { ...form, enabled: ($event.target as HTMLInputElement).checked })" />
		</label>
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
