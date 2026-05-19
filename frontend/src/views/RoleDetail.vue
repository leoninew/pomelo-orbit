<template>
	<div class="flex flex-col gap-4">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h1 class="text-xl font-semibold text-foreground">{{ role?.name ?? '角色详情' }}</h1>
			<div class="flex flex-wrap items-center gap-2">
				<button
					v-if="role && canWriteRoles"
					class="app-button-primary h-9 px-3"
					:disabled="operating"
					@click="openEditModal"
				>
					<Pencil class="size-4" />
					{{ t('common.edit') }}
				</button>
				<button
					v-if="role && canWriteRoles"
					class="app-button-danger h-9 px-3"
					:disabled="operating"
					@click="openDeleteModal"
				>
					<Trash2 class="size-4" />
					{{ t('common.delete') }}
				</button>
				<button class="app-button h-9 px-4" @click="$router.push('/roles')">
					<ArrowLeft class="size-4" />
					{{ t('common.back') }}
				</button>
			</div>
		</div>

		<div class="app-surface">
			<div class="app-section-header">
				<h2 class="font-semibold text-foreground">{{ t('roleManagement.basicInfo') }}</h2>
			</div>

			<AppSpinner v-if="loading" class="px-5 py-10" />
			<dl
				v-else-if="role"
				class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2"
			>
				<div class="flex gap-2">
					<dt class="w-24 shrink-0 text-muted-foreground">{{ t('roleManagement.code') }}</dt>
					<dd class="text-foreground">{{ role.code }}</dd>
				</div>
				<div class="flex gap-2">
					<dt class="w-24 shrink-0 text-muted-foreground">{{ t('common.name') }}</dt>
					<dd class="text-foreground">{{ role.name }}</dd>
				</div>
				<div class="flex gap-2">
					<dt class="w-24 shrink-0 text-muted-foreground">{{ t('common.description') }}</dt>
					<dd class="text-foreground">{{ role.description || '-' }}</dd>
				</div>
				<div class="flex gap-2">
					<dt class="w-24 shrink-0 text-muted-foreground">{{ t('common.createdAt') }}</dt>
					<dd class="text-muted-foreground">{{ formatTime(role.created_at) }}</dd>
				</div>
				<div class="flex gap-2">
					<dt class="w-24 shrink-0 text-muted-foreground">{{ t('common.updatedAt') }}</dt>
					<dd class="text-muted-foreground">{{ formatTime(role.updated_at) }}</dd>
				</div>
			</dl>
		</div>

		<div class="app-surface">
			<div class="app-section-header">
				<h2 class="font-semibold text-foreground">{{ t('roleManagement.permissions') }}</h2>
			</div>

			<div class="px-5 py-4">
				<div v-if="role && role.permission_codes.length > 0" class="flex flex-wrap gap-2">
					<AppBadge v-for="code in role.permission_codes" :key="code">
						{{ getPermissionName(code) }}
					</AppBadge>
				</div>
				<p v-else class="text-sm text-muted-foreground">{{ t('common.noData') }}</p>
			</div>
		</div>

		<AppDialog
			v-model:open="isEditModalOpen"
			:title="t('roleManagement.edit')"
			width-class="w-[min(600px,calc(100vw-32px))]"
		>
			<form id="role-edit-form" class="space-y-4" @submit.prevent="handleEditOk">
				<div class="space-y-1.5">
					<label class="app-field-label block" for="role-code">
						{{ t('roleManagement.code') }}
					</label>
					<input
						id="role-code"
						v-model="form.code"
						type="text"
						class="app-input"
						maxlength="50"
						pattern="[A-Za-z0-9_-]+"
						required
						:disabled="operating"
					/>
				</div>
				<div class="space-y-1.5">
					<label class="app-field-label block" for="role-name">{{ t('common.name') }}</label>
					<input
						id="role-name"
						v-model="form.name"
						type="text"
						class="app-input"
						maxlength="100"
						required
						:disabled="operating"
					/>
				</div>
				<div class="space-y-1.5">
					<label class="app-field-label block" for="role-description">
						{{ t('common.description') }}
					</label>
					<textarea
						id="role-description"
						v-model="form.description"
						class="app-input min-h-24"
						maxlength="500"
						:disabled="operating"
					/>
				</div>
				<div class="space-y-2">
					<span class="app-field-label block">{{ t('roleManagement.permissions') }}</span>
					<div class="grid gap-2 sm:grid-cols-2">
						<label
							v-for="permission in permissions"
							:key="permission.code"
							class="flex items-start gap-2 rounded-md border border-border px-3 py-2 text-sm"
						>
							<input v-model="form.permissionCodes" type="checkbox" :value="permission.code" :disabled="operating" />
							<span>
								<span class="block text-foreground">{{ permission.name }}</span>
								<span class="block text-xs text-muted-foreground">{{ permission.code }}</span>
							</span>
						</label>
					</div>
				</div>
			</form>
			<template #footer>
				<button class="app-button" :disabled="operating" @click="isEditModalOpen = false">
					{{ t('common.cancel') }}
				</button>
				<button
					class="app-button-primary"
					type="submit"
					form="role-edit-form"
					:disabled="operating"
				>
					{{ t('common.save') }}
				</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isDeleteModalOpen"
			:title="t('common.delete')"
			width-class="w-[min(420px,calc(100vw-32px))]"
		>
			<p class="text-sm text-foreground">
				{{ t('roleManagement.deleteConfirm') }}
				<strong>{{ role?.name }}</strong>
			</p>
			<template #footer>
				<button class="app-button" :disabled="operating" @click="isDeleteModalOpen = false">
					{{ t('common.cancel') }}
				</button>
				<button class="app-button-danger" :disabled="operating" @click="handleDelete">
					{{ t('common.delete') }}
				</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { ArrowLeft, Pencil, Trash2 } from 'lucide-vue-next';
	import { computed, onMounted, reactive, ref } from 'vue';
	import { useI18n } from 'vue-i18n';
	import { useRouter } from 'vue-router';
	import { roleApi } from '@/api/role';
	import AppBadge from '@/components/AppBadge.vue';
	import AppDialog from '@/components/AppDialog.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import { useAuthStore } from '@/stores/auth';
	import { PERMISSIONS } from '@/constants/permissions';
	import type { PermissionResp, RoleResp } from '@/types/role';
	import { formatTime } from '@/utils/time';

	const props = defineProps<{ id: string }>();
	const { t } = useI18n();
	const $router = useRouter();
	const toast = useToast();
	const authStore = useAuthStore();
	const { loading, execute } = useStatusAsync();
	const { loading: operating, execute: executeOp } = useStatusAsync();

	const role = ref<RoleResp>();
	const permissions = ref<PermissionResp[]>([]);
	const isEditModalOpen = ref(false);
	const isDeleteModalOpen = ref(false);
	const form = reactive({ code: '', name: '', description: '', permissionCodes: [] as string[] });

	const canWriteRoles = computed(() => authStore.hasPermission(PERMISSIONS.ROLE_WRITE));

	function getPermissionName(code: string): string {
		const permission = permissions.value.find((p) => p.code === code);
		return permission ? `${permission.name} (${code})` : code;
	}

	async function fetchRole() {
		try {
			await execute(async () => {
				role.value = await roleApi.get(props.id);
			});
		} catch {
			toast.error(t('roleManagement.loadFailed'));
		}
	}

	async function fetchPermissions() {
		if (!canWriteRoles.value) {
			return;
		}
		try {
			permissions.value = await roleApi.listPermissions();
		} catch {
			toast.error(t('roleManagement.loadPermissionsFailed'));
		}
	}

	function openEditModal() {
		form.code = role.value?.code ?? '';
		form.name = role.value?.name ?? '';
		form.description = role.value?.description ?? '';
		form.permissionCodes = role.value?.permission_codes ? [...role.value.permission_codes] : [];
		isEditModalOpen.value = true;
	}

	async function handleEditOk() {
		try {
			await executeOp(async () => {
				await roleApi.update(props.id, {
					code: form.code.trim(),
					name: form.name.trim(),
					description: form.description.trim() || null,
					permission_codes: form.permissionCodes,
				});
				await authStore.fetchUser();
				toast.success(t('roleManagement.updated'));
				isEditModalOpen.value = false;
				await fetchRole();
			});
		} catch (e: unknown) {
			toast.error(e instanceof Error ? e.message : t('roleManagement.saveFailed'));
		}
	}

	function openDeleteModal() {
		isDeleteModalOpen.value = true;
	}

	async function handleDelete() {
		try {
			await executeOp(async () => {
				await roleApi.delete(props.id);
				await authStore.fetchUser();
				toast.success(t('roleManagement.deleted'));
				$router.push('/roles');
			});
		} catch (e: unknown) {
			toast.error(e instanceof Error ? e.message : t('roleManagement.deleteFailed'));
		}
	}

	onMounted(() => {
		fetchRole();
		fetchPermissions();
	});
</script>
