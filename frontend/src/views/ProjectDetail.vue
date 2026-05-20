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

		<div class="app-surface">
			<div class="app-section-header flex flex-wrap items-center justify-between gap-3">
				<h2 class="font-semibold text-foreground">项目成员</h2>
				<button class="app-button-primary h-8 px-3" :disabled="operating" @click="openMemberModal">
					<UserPlus class="size-4" />
					添加
				</button>
			</div>

			<AppSpinner v-if="loadingMembers" class="px-5 py-10" />
			<div v-else-if="members.length === 0" class="px-5 py-4">
				<p class="text-sm text-muted-foreground">暂无成员</p>
			</div>
			<div v-else class="px-5 py-4">
				<table class="app-table-detail">
					<thead>
						<tr>
							<th>用户名</th>
							<th>邮箱</th>
							<th>状态</th>
							<th>来源</th>
							<th>上次登录</th>
							<th class="text-right">操作</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="member in members" :key="member.id">
							<td>{{ member.username }}</td>
							<td>{{ member.email || '—' }}</td>
							<td>
								<AppBadge v-if="member.status === 'enabled'" variant="status" tone="success">
									{{ t('userManagement.enabled') }}
								</AppBadge>
								<AppBadge v-else variant="status" tone="default">
									{{ t('userManagement.disabled') }}
								</AppBadge>
							</td>
							<td>{{ formatAuthSource(member.auth_source) }}</td>
							<td>{{ member.last_login_at ? formatTime(member.last_login_at) : '—' }}</td>
							<td class="text-right">
								<button
									class="app-link-danger"
									:disabled="operating"
									@click="handleRemoveMember(member.id)"
								>
									移除
								</button>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
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
				<button class="app-button" :disabled="operating" @click="isEditModalOpen = false">
					取消
				</button>
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

		<AppDialog v-model:open="isMemberModalOpen" title="添加成员">
			<div class="space-y-4">
				<div class="space-y-1.5">
					<label class="app-field-label block">选择用户</label>
					<ComboboxSelect
						v-model="selectedUserId"
						:options="userOptions"
						placeholder="搜索用户"
						empty-text="暂无可用用户"
						:disabled="operating"
					/>
				</div>
			</div>
			<template #footer>
				<button class="app-button" :disabled="operating" @click="isMemberModalOpen = false">
					取消
				</button>
				<button
					class="app-button-primary"
					:disabled="operating || !selectedUserId"
					@click="handleAddMember"
				>
					添加
				</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { ArrowLeft, Pencil, UserPlus } from 'lucide-vue-next';
	import { onMounted, reactive, ref, computed } from 'vue';
	import { useI18n } from 'vue-i18n';
	import { useRouter } from 'vue-router';
	import { projectApi } from '@/api/project';
	import { userApi } from '@/api/user';
	import AppBadge from '@/components/AppBadge.vue';
	import AppDialog from '@/components/AppDialog.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import ComboboxSelect from '@/components/ComboboxSelect.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import { useProjectStore } from '@/stores/project';
	import type { AuthSource } from '@/types/auth';
	import type { Project, ProjectMember } from '@/types/project';
	import type { UserListResp } from '@/types/user';
	import { formatTime } from '@/utils/time';

	const props = defineProps<{ id: string }>();
	const { t } = useI18n();
	const router = useRouter();
	const toast = useToast();
	const projectStore = useProjectStore();
	const { loading, execute } = useStatusAsync();
	const { loading: operating, execute: executeOp } = useStatusAsync();

	const project = ref<Project>();
	const isEditModalOpen = ref(false);
	const isMemberModalOpen = ref(false);
	const members = ref<ProjectMember[]>([]);
	const users = ref<UserListResp[]>([]);
	const selectedUserId = ref('');
	const form = reactive({ name: '', code: '' });
	const errors = reactive({ name: '', code: '' });
	const { loading: loadingMembers, execute: executeMembers } = useStatusAsync();

	const availableUsers = computed(() => {
		const memberIds = new Set(members.value.map((m) => m.id));
		return users.value.filter((u) => u.status === 'enabled' && !memberIds.has(u.id));
	});

	const userOptions = computed(() =>
		availableUsers.value.map((u) => ({
			value: u.id,
			label: u.username,
			description: u.email || undefined,
		}))
	);

	function formatAuthSource(authSource: AuthSource) {
		switch (authSource) {
		case 'oauth':
			return t('userManagement.authSourceOAuth');
		case 'password':
			return t('userManagement.authSourcePassword');
		default:
			throw new Error(`Unsupported auth source: ${authSource}`);
		}
	}

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

	async function fetchMembers() {
		try {
			await executeMembers(async () => {
				const [memberList, userPage] = await Promise.all([
					projectApi.listMembers(props.id),
					userApi.list({ page: 1, per_page: 100 }),
				]);
				members.value = memberList;
				users.value = userPage.items;
			});
		} catch {
			toast.error('加载成员失败');
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

	function openMemberModal() {
		selectedUserId.value = '';
		isMemberModalOpen.value = true;
	}

	async function handleAddMember() {
		if (!selectedUserId.value) {
			return;
		}
		try {
			await executeOp(async () => {
				members.value = await projectApi.addMember(props.id, { user_id: selectedUserId.value });
				selectedUserId.value = '';
				toast.success('成员已添加');
				isMemberModalOpen.value = false;
			});
		} catch (e: unknown) {
			toast.error(e instanceof Error ? e.message : '添加成员失败');
		}
	}

	async function handleRemoveMember(userId: string) {
		try {
			await executeOp(async () => {
				members.value = await projectApi.removeMember(props.id, userId);
				toast.success('成员已移除');
			});
		} catch (e: unknown) {
			toast.error(e instanceof Error ? e.message : '移除成员失败');
		}
	}

	onMounted(() => {
		fetchProject();
		fetchMembers();
	});
</script>
