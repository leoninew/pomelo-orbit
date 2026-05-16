<template>
	<div class="space-y-6">
		<ToolbarRoot class="app-toolbar-simple" aria-label="项目工具栏">
			<SearchControl
				v-model="searchText"
				class="shrink-0"
				placeholder="搜索项目名称/编码"
				:loading="status === 'loading'"
				@search="handleSearch"
			/>
			<div class="flex items-center gap-3">
				<button class="app-button-primary px-5" @click="openCreateDialog">
					<Plus class="size-4" />
					新建项目
				</button>
			</div>
		</ToolbarRoot>

		<div class="app-surface">
			<AppSpinner v-if="status === 'loading'" class="py-16" />
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error || '加载失败' }}</p>
			</div>
			<div
				v-else-if="filteredProjects.length === 0"
				class="text-center py-16 text-muted-foreground"
			>
				<p class="text-sm">{{ searchText ? '未找到匹配的项目' : '暂无项目' }}</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[900px]">
					<colgroup>
						<col class="w-[25%]" />
						<col class="w-[20%]" />
						<col class="w-[20%]" />
						<col class="w-[20%]" />
						<col class="w-[15%]" />
					</colgroup>
					<thead>
						<tr>
							<th>项目名称</th>
							<th>项目编码</th>
							<th>创建时间</th>
							<th>更新时间</th>
							<th>操作</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="project in pagedProjects" :key="project.id">
							<td class="max-w-0 truncate text-foreground" :title="project.name">
								{{ project.name }}
							</td>
							<td class="whitespace-nowrap text-foreground">{{ project.code }}</td>
							<td class="whitespace-nowrap text-foreground">
								{{ formatTime(project.created_at) }}
							</td>
							<td class="whitespace-nowrap text-foreground">
								{{ formatTime(project.updated_at) }}
							</td>
							<td class="whitespace-nowrap">
								<button class="app-link" @click="openEditDialog(project)">编辑</button>
							</td>
						</tr>
					</tbody>
				</table>
			</div>

			<ListPagination
				:current="pagination.current"
				:page-size="pagination.pageSize"
				:total="filteredProjects.length"
				:total-pages="totalPages"
				@change-page="goPage"
				@change-page-size="handlePageSizeChange"
			/>
		</div>

		<AppDialog v-model:open="isDialogOpen" :title="editingProject ? '编辑项目' : '新建项目'">
			<div class="space-y-4">
				<div class="space-y-1.5">
					<label class="app-field-label block">项目名称</label>
					<input
						v-model="form.name"
						type="text"
						class="app-input"
						:class="errors.name ? 'app-input-error' : ''"
						placeholder="Default Project"
					/>
					<p v-if="errors.name" class="app-field-error text-xs">{{ errors.name }}</p>
				</div>
				<div class="space-y-1.5">
					<label class="app-field-label block">项目编码</label>
					<input
						v-model="form.code"
						type="text"
						class="app-input"
						:class="errors.code ? 'app-input-error' : ''"
						placeholder="default"
					/>
					<p v-if="errors.code" class="app-field-error text-xs">{{ errors.code }}</p>
					<p v-else class="app-field-hint">只能包含小写字母、数字、下划线和连字符</p>
				</div>
			</div>

			<template #footer>
				<button class="app-button" @click="isDialogOpen = false">取消</button>
				<button class="app-button-primary" :disabled="operating" @click="handleSave">
					{{ editingProject ? '保存' : '创建' }}
				</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { Plus } from 'lucide-vue-next';
	import { computed, onMounted, reactive, ref } from 'vue';
	import { ToolbarRoot } from 'reka-ui';
	import type { Project } from '@/types/project';
	import AppDialog from '@/components/AppDialog.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import ListPagination from '@/components/ListPagination.vue';
	import SearchControl from '@/components/SearchControl.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import { useProjectStore } from '@/stores/project';
	import { formatTime } from '@/utils/time';

	const toast = useToast();
	const projectStore = useProjectStore();
	const { status, error, execute } = useStatusAsync();
	const { loading: operating, execute: executeOp } = useStatusAsync();

	const searchText = ref('');
	const isDialogOpen = ref(false);
	const editingProject = ref<Project | null>(null);
	const pagination = reactive({ current: 1, pageSize: 10 });
	const form = reactive({ name: '', code: '' });
	const errors = reactive({ name: '', code: '' });

	const filteredProjects = computed(() => {
		const keyword = searchText.value.trim().toLowerCase();
		if (!keyword) {
			return projectStore.projects;
		}
		return projectStore.projects.filter(
			(project) =>
				project.name.toLowerCase().includes(keyword) || project.code.toLowerCase().includes(keyword)
		);
	});
	const totalPages = computed(() =>
		Math.max(1, Math.ceil(filteredProjects.value.length / pagination.pageSize))
	);
	const pagedProjects = computed(() => {
		const start = (pagination.current - 1) * pagination.pageSize;
		return filteredProjects.value.slice(start, start + pagination.pageSize);
	});

	function resetForm(project?: Project) {
		form.name = project?.name ?? '';
		form.code = project?.code ?? '';
		errors.name = '';
		errors.code = '';
	}

	function validate() {
		errors.name = form.name.trim() ? '' : '请输入项目名称';
		errors.code = /^[a-z0-9_-]+$/.test(form.code) ? '' : '只能包含小写字母、数字、下划线和连字符';
		return !errors.name && !errors.code;
	}

	function openCreateDialog() {
		editingProject.value = null;
		resetForm();
		isDialogOpen.value = true;
	}

	function openEditDialog(project: Project) {
		editingProject.value = project;
		resetForm(project);
		isDialogOpen.value = true;
	}

	function handleSearch() {
		pagination.current = 1;
	}

	function goPage(page: number) {
		pagination.current = page;
	}

	function handlePageSizeChange(pageSize: number) {
		pagination.pageSize = pageSize;
		pagination.current = 1;
	}

	async function fetchProjects() {
		await execute(async () => {
			await projectStore.fetchProjects();
		});
	}

	async function handleSave() {
		if (!validate()) {
			return;
		}
		await executeOp(async () => {
			if (editingProject.value) {
				await projectStore.updateProject(editingProject.value.id, {
					name: form.name.trim(),
					code: form.code.trim(),
				});
				toast.success('项目已更新');
			} else {
				await projectStore.createProject({ name: form.name.trim(), code: form.code.trim() });
				toast.success('项目已创建');
			}
			isDialogOpen.value = false;
		});
	}

	onMounted(fetchProjects);
</script>
