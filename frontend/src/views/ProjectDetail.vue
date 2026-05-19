<template>
	<div class="flex flex-col gap-4">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h1 class="text-xl font-semibold text-foreground">{{ project?.name ?? '项目详情' }}</h1>
			<div class="flex flex-wrap items-center gap-2">
				<button
					v-if="project"
					class="app-button-primary h-9 px-3"
					:disabled="operating"
					@click="openEditModal"
				>
					<Pencil class="size-4" />
					编辑
				</button>
				<button class="app-button h-9 px-4" @click="router.push('/projects')">
					<ArrowLeft class="size-4" />
					返回
				</button>
			</div>
		</div>

		<div class="app-surface">
			<div class="app-section-header">
				<h2 class="font-semibold text-foreground">基本信息</h2>
			</div>

			<AppSpinner v-if="loading" class="px-5 py-10" />
			<dl
				v-else-if="project"
				class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2"
			>
				<div class="flex gap-2">
					<dt class="w-24 shrink-0 text-muted-foreground">项目名称</dt>
					<dd class="text-foreground">{{ project.name }}</dd>
				</div>
				<div class="flex gap-2">
					<dt class="w-24 shrink-0 text-muted-foreground">项目编码</dt>
					<dd class="text-foreground">{{ project.code }}</dd>
				</div>
				<div class="flex gap-2">
					<dt class="w-24 shrink-0 text-muted-foreground">状态</dt>
					<dd>
						<AppBadge v-if="project.is_active" variant="status" tone="success">活跃</AppBadge>
						<AppBadge v-else variant="status" tone="default">已废弃</AppBadge>
					</dd>
				</div>
				<div class="flex gap-2">
					<dt class="w-24 shrink-0 text-muted-foreground">创建时间</dt>
					<dd class="text-muted-foreground">{{ formatTime(project.created_at) }}</dd>
				</div>
				<div class="flex gap-2">
					<dt class="w-24 shrink-0 text-muted-foreground">更新时间</dt>
					<dd class="text-muted-foreground">{{ formatTime(project.updated_at) }}</dd>
				</div>
			</dl>
		</div>

		<AppDialog
			v-model:open="isEditModalOpen"
			title="编辑项目"
			width-class="w-[min(600px,calc(100vw-32px))]"
		>
			<form id="project-edit-form" class="space-y-4" @submit.prevent="handleEditOk">
				<div class="space-y-1.5">
					<label class="app-field-label block" for="project-name">项目名称</label>
					<input
						id="project-name"
						v-model="form.name"
						type="text"
						class="app-input"
						:class="errors.name ? 'app-input-error' : ''"
						maxlength="100"
						required
						:disabled="operating"
					/>
					<p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
				</div>
				<div class="space-y-1.5">
					<label class="app-field-label block" for="project-code">项目编码</label>
					<input
						id="project-code"
						v-model="form.code"
						type="text"
						class="app-input"
						:class="errors.code ? 'app-input-error' : ''"
						maxlength="50"
						pattern="[a-z0-9_-]+"
						required
						:disabled="operating"
					/>
					<p v-if="errors.code" class="app-field-error text-xs">{{ errors.code }}</p>
					<p v-else class="app-field-hint">只能包含小写字母、数字、下划线和连字符</p>
				</div>
			</form>
			<template #footer>
				<button class="app-button" :disabled="operating" @click="isEditModalOpen = false">取消</button>
				<button
					class="app-button-primary"
					type="submit"
					form="project-edit-form"
					:disabled="operating"
				>
					保存
				</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { ArrowLeft, Pencil } from 'lucide-vue-next';
	import { onMounted, reactive, ref } from 'vue';
	import { useRouter } from 'vue-router';
	import { projectApi } from '@/api/project';
	import AppBadge from '@/components/AppBadge.vue';
	import AppDialog from '@/components/AppDialog.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import { useProjectStore } from '@/stores/project';
	import type { Project } from '@/types/project';
	import { formatTime } from '@/utils/time';

	const props = defineProps<{ id: string }>();
	const router = useRouter();
	const toast = useToast();
	const projectStore = useProjectStore();
	const { loading, execute } = useStatusAsync();
	const { loading: operating, execute: executeOp } = useStatusAsync();

	const project = ref<Project>();
	const isEditModalOpen = ref(false);
	const form = reactive({ name: '', code: '' });
	const errors = reactive({ name: '', code: '' });

	function resetForm() {
		form.name = project.value?.name ?? '';
		form.code = project.value?.code ?? '';
		errors.name = '';
		errors.code = '';
	}

	function validate() {
		errors.name = form.name.trim() ? '' : '请输入项目名称';
		errors.code = /^[a-z0-9_-]+$/.test(form.code) ? '' : '只能包含小写字母、数字、下划线和连字符';
		return !errors.name && !errors.code;
	}

	async function fetchProject() {
		try {
			await execute(async () => {
				project.value = await projectApi.get(props.id);
			});
		} catch {
			toast.error('加载项目失败');
		}
	}

	function openEditModal() {
		resetForm();
		isEditModalOpen.value = true;
	}

	async function handleEditOk() {
		if (!validate()) {
			return;
		}
		try {
			await executeOp(async () => {
				project.value = await projectStore.updateProject(props.id, {
					name: form.name.trim(),
					code: form.code.trim(),
				});
				toast.success('项目已更新');
				isEditModalOpen.value = false;
			});
		} catch (e: unknown) {
			toast.error(e instanceof Error ? e.message : '保存失败');
		}
	}

	onMounted(fetchProject);
</script>
