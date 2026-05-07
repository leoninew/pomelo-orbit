<script setup lang="ts">
import SelectControl from '@/components/SelectControl.vue'

interface FormData {
	name: string
	code: string
	image_pull_policy: string
	route_managed: boolean
}

const props = defineProps<{
	form: FormData
	errors: { name: string; code: string }
}>();

const emit = defineEmits<{
	'update:form': [value: FormData]
}>();

function updateField<K extends keyof FormData>(field: K, value: FormData[K]) {
	emit('update:form', { ...props.form, [field]: value });
}

const imagePullPolicyOptions = [
	{ value: 'missing', label: '缺失时拉取 (missing)' },
	{ value: 'always', label: '总是拉取 (always)' },
	{ value: 'never', label: '从不拉取 (never)' }
]
</script>

<template>
	<div class="space-y-4">
		<!-- 应用名称 -->
		<div>
			<label class="mb-1.5 block text-sm font-medium text-foreground">
				应用名称
				<span class="text-destructive">*</span>
			</label>
			<input
				:value="form.name"
				type="text"
				placeholder="输入应用名称"
				class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground transition-colors placeholder:text-muted-foreground focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/20"
				:class="{ 'border-destructive': errors.name }"
				@input="updateField('name', ($event.target as HTMLInputElement).value)"
			/>
			<p v-if="errors.name" class="mt-1 text-xs text-destructive">{{ errors.name }}</p>
		</div>

		<!-- 应用代码 -->
		<div>
			<label class="mb-1.5 block text-sm font-medium text-foreground">
				应用代码
				<span class="text-destructive">*</span>
			</label>
			<input
				:value="form.code"
				type="text"
				placeholder="输入应用代码（英文、数字、下划线）"
				class="w-full rounded-md border border-input bg-background px-3 py-2 text-sm text-foreground transition-colors placeholder:text-muted-foreground focus:border-ring focus:outline-none focus:ring-2 focus:ring-ring/20"
				:class="{ 'border-destructive': errors.code }"
				@input="updateField('code', ($event.target as HTMLInputElement).value)"
			/>
			<p v-if="errors.code" class="mt-1 text-xs text-destructive">{{ errors.code }}</p>
			<p class="mt-1 text-xs text-muted-foreground">
				应用代码用于生成工作目录，创建后不可修改
			</p>
		</div>

		<!-- 镜像拉取策略 -->
		<div>
			<label class="mb-1.5 block text-sm font-medium text-foreground">镜像拉取策略</label>
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
				class="h-4 w-4 rounded border-input text-primary transition-colors focus:ring-2 focus:ring-ring/20"
				@change="updateField('route_managed', ($event.target as HTMLInputElement).checked)"
			/>
			<label for="route_managed" class="text-sm font-medium text-foreground">
				启用路由管理
			</label>
		</div>
		<p class="text-xs text-muted-foreground">
			启用后可为应用配置 HTTP 路由，自动生成 Traefik 配置
		</p>
	</div>
</template>
