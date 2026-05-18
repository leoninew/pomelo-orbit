<template>
	<div class="space-y-6">
		<div class="app-toolbar-simple" aria-label="角色工具栏">
			<SearchControl
				v-model="searchText"
				class="shrink-0"
				:placeholder="t('roleManagement.searchPlaceholder')"
				:loading="status === 'loading'"
				@search="handleSearch"
			/>
			<div class="flex items-center gap-3">
				<button class="app-button-primary px-5" @click="openCreateDialog">
					<Plus class="size-4" />
					{{ t('roleManagement.create') }}
				</button>
			</div>
		</div>

		<div class="app-surface">
			<AppSpinner v-if="status === 'loading'" class="py-16" />
			<div v-else-if="status === 'error'" class="text-center py-16 text-destructive">
				<p class="text-sm">{{ error || t('roleManagement.loadFailed') }}</p>
			</div>
			<div v-else-if="roles.length === 0" class="text-center py-16 text-muted-foreground">
				<p class="text-sm">{{ t('common.noData') }}</p>
			</div>
			<div v-else class="overflow-x-auto">
				<table class="app-table-list min-w-[760px]">
					<colgroup>
						<col class="w-[20%]" />
						<col class="w-[20%]" />
						<col class="w-[32%]" />
						<col class="w-[16%]" />
						<col class="w-[12%]" />
					</colgroup>
					<thead>
						<tr>
							<th>{{ t('roleManagement.code') }}</th>
							<th>{{ t('common.name') }}</th>
							<th>{{ t('common.description') }}</th>
							<th>{{ t('common.createdAt') }}</th>
							<th>{{ t('common.operation') }}</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="role in roles" :key="role.id">
							<td class="max-w-0 truncate text-foreground" :title="role.code">
								{{ role.code }}
							</td>
							<td class="max-w-0 truncate text-foreground" :title="role.name">
								{{ role.name }}
							</td>
							<td class="max-w-0 truncate text-foreground" :title="role.description || undefined">
								{{ role.description || '-' }}
							</td>
							<td class="whitespace-nowrap text-foreground">
								{{ formatTime(role.created_at) }}
							</td>
							<td class="whitespace-nowrap">
								<div class="flex items-center gap-3">
									<button class="app-link" @click="openEditDialog(role)">
										{{ t('common.edit') }}
									</button>
									<button class="app-link-danger" @click="openConfirmDialog(role)">
										{{ t('common.delete') }}
									</button>
								</div>
							</td>
						</tr>
					</tbody>
				</table>
			</div>

			<ListPagination
				:current="pagination.current"
				:page-size="pagination.pageSize"
				:total="pagination.total"
				:total-pages="totalPages"
				@change-page="goPage"
				@change-page-size="handlePageSizeChange"
			/>
		</div>

		<AppDialog
			v-model:open="isDialogOpen"
			:title="editingRole ? t('roleManagement.edit') : t('roleManagement.create')"
		>
			<form id="role-form" class="space-y-4" @submit.prevent="handleSave">
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
					/>
				</div>
			</form>
			<template #footer>
				<button class="app-button" @click="isDialogOpen = false">{{ t('common.cancel') }}</button>
				<button class="app-button-primary" type="submit" form="role-form" :disabled="operating">
					{{ editingRole ? t('common.save') : t('roleManagement.create') }}
				</button>
			</template>
		</AppDialog>

		<AppDialog v-model:open="confirmDialogOpen" :title="t('roleManagement.delete')">
			<p class="text-sm text-foreground">
				{{ t('roleManagement.deleteConfirm') }}
				<strong>{{ confirmAction?.role.name }}</strong>
			</p>
			<template #footer>
				<button class="app-button" @click="confirmAction = null">{{ t('common.cancel') }}</button>
				<button class="app-button-danger" :disabled="operating" @click="handleConfirm">
					{{ t('common.delete') }}
				</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { Plus } from 'lucide-vue-next';
	import { computed, nextTick, onMounted, reactive, ref } from 'vue';
	import { useI18n } from 'vue-i18n';
	import { roleApi } from '@/api/role';
	import AppDialog from '@/components/AppDialog.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import ListPagination from '@/components/ListPagination.vue';
	import SearchControl from '@/components/SearchControl.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import type { RoleResp } from '@/types/role';
	import { formatTime } from '@/utils/time';

	const { t } = useI18n();
	const toast = useToast();
	const { status, error, execute } = useStatusAsync();
	const { loading: operating, execute: executeOp } = useStatusAsync();

	const roles = ref<RoleResp[]>([]);
	const searchText = ref('');
	const pagination = reactive({ current: 1, pageSize: 10, total: 0 });
	type ConfirmAction = { role: RoleResp };

	const editingRole = ref<RoleResp | null>(null);
	const confirmAction = ref<ConfirmAction | null>(null);
	const isDialogOpen = ref(false);
	const form = reactive({ code: '', name: '', description: '' });
	const totalPages = computed(() => Math.max(1, Math.ceil(pagination.total / pagination.pageSize)));
	const confirmDialogOpen = computed({
		get: () => confirmAction.value !== null,
		set: (open) => {
			if (!open) {
				confirmAction.value = null;
			}
		},
	});

	function resetForm(role?: RoleResp) {
		form.code = role?.code ?? '';
		form.name = role?.name ?? '';
		form.description = role?.description ?? '';
	}

	async function fetchRoles() {
		try {
			await execute(async () => {
				const res = await roleApi.list({
					page: pagination.current,
					per_page: pagination.pageSize,
					search: searchText.value || undefined,
				});
				roles.value = res.items;
				pagination.total = res.total;
			});
		} catch {
			toast.error(t('roleManagement.loadFailed'));
		}
	}

	function handleSearch() {
		pagination.current = 1;
		fetchRoles();
	}

	function goPage(page: number) {
		pagination.current = page;
		fetchRoles();
	}

	function handlePageSizeChange(pageSize: number) {
		pagination.pageSize = pageSize;
		pagination.current = 1;
		fetchRoles();
	}

	function openCreateDialog() {
		editingRole.value = null;
		resetForm();
		isDialogOpen.value = true;
	}

	function openEditDialog(role: RoleResp) {
		editingRole.value = role;
		resetForm(role);
		isDialogOpen.value = true;
	}

	async function openConfirmDialog(role: RoleResp) {
		confirmAction.value = { role };
		(document.activeElement as HTMLElement)?.blur();
		await nextTick();
	}

	async function handleSave() {
		try {
			await executeOp(async () => {
				const payload = {
					code: form.code.trim(),
					name: form.name.trim(),
					description: form.description.trim() || null,
				};
				if (editingRole.value) {
					await roleApi.update(editingRole.value.id, payload);
					toast.success(t('roleManagement.updated'));
				} else {
					await roleApi.create(payload);
					toast.success(t('roleManagement.created'));
				}
				isDialogOpen.value = false;
				await fetchRoles();
			});
		} catch (e: unknown) {
			toast.error(e instanceof Error ? e.message : t('roleManagement.saveFailed'));
		}
	}

	async function handleConfirm() {
		const action = confirmAction.value;
		if (!action) {
			return;
		}
		try {
			await executeOp(async () => {
				await roleApi.delete(action.role.id);
				toast.success(t('roleManagement.deleted'));
				confirmAction.value = null;
				await fetchRoles();
			});
		} catch (e: unknown) {
			toast.error(e instanceof Error ? e.message : t('roleManagement.deleteFailed'));
		}
	}

	onMounted(fetchRoles);
</script>
