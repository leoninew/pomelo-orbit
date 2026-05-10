<template>
	<div class="space-y-6">
		<!-- 用户信息 -->
		<div class="app-surface">
			<div class="app-section-header flex items-center justify-between">
				<h2 class="font-semibold text-foreground">{{ t('settings.userInfo') }}</h2>
				<button class="app-button-primary h-8 px-3" @click="isPasswordDialogOpen = true">
					<Key class="size-4" />
					{{ t('settings.changePassword') }}
				</button>
			</div>
			<dl
				v-if="authStore.user"
				class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2"
			>
				<div class="flex gap-2">
					<dt class="w-24 shrink-0 text-muted-foreground">{{ t('settings.username') }}</dt>
					<dd class="text-foreground">{{ authStore.user.username }}</dd>
				</div>
				<div class="flex gap-2">
					<dt class="w-24 shrink-0 text-muted-foreground">{{ t('settings.createdAt') }}</dt>
					<dd class="text-muted-foreground">{{ formatTime(authStore.user.created_at) }}</dd>
				</div>
				<div class="flex gap-2">
					<dt class="w-24 shrink-0 text-muted-foreground">{{ t('settings.lastLogin') }}</dt>
					<dd class="text-muted-foreground">{{ formatTime(authStore.user.last_login_at) }}</dd>
				</div>
			</dl>
		</div>

		<ToolbarRoot class="flex items-center justify-between gap-6" aria-label="系统设置工具栏">
			<SearchControl
				v-model="searchText"
				:placeholder="t('settings.searchPlaceholder')"
				:loading="configLoading"
			/>
		</ToolbarRoot>

		<!-- Restart Warning -->
		<div v-if="needsRestart" class="app-tip border-amber-200 bg-amber-50">
			<p class="text-sm text-amber-800">{{ t('settings.restartWarning') }}</p>
		</div>

		<!-- Config Table -->
		<div v-if="configLoading" class="app-surface">
			<AppSpinner class="py-16" />
		</div>
		<div v-else-if="filteredConfig.length === 0" class="app-surface">
			<div class="text-center py-16 text-muted-foreground">
				<p class="text-sm">{{ searchText ? t('settings.noMatch') : t('settings.noConfig') }}</p>
			</div>
		</div>
		<div v-else class="app-surface">
			<div class="overflow-x-auto">
				<table class="app-table-list min-w-[1120px]">
					<thead>
						<tr>
							<th>{{ t('settings.configKey') }}</th>
							<th>{{ t('settings.currentValue') }}</th>
							<th>{{ t('settings.defaultValue') }}</th>
							<th>{{ t('settings.updatedAt') }}</th>
							<th>{{ t('common.operation') }}</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="item in filteredConfig" :key="item.key">
							<td>
								<div class="text-foreground">{{ item.key }}</div>
								<div v-if="item.description" class="mt-1 text-xs text-muted-foreground">
									{{ item.description }}
								</div>
							</td>
							<td>
								<!-- Editing Mode -->
								<div v-if="editingKey === item.key">
									<!-- Boolean -->
									<SelectControl
										v-if="typeof item.default === 'boolean'"
										v-model="editingBoolStr"
										:options="booleanOptions"
										width-class="w-28"
									/>
									<!-- Select -->
									<SelectControl
										v-else-if="selectOptions[item.key]"
										v-model="editingStr"
										:options="getSettingOptions(item.key)"
										width-class="w-40"
									/>
									<!-- Text -->
									<input
										v-else
										v-model="editingStr"
										:type="secretKeys.has(item.key) ? 'password' : 'text'"
										:placeholder="
											secretKeys.has(item.key)
												? t('settings.passwordDialog.emptyKeepUnchanged')
												: ''
										"
										class="app-input h-9"
									/>
								</div>
								<!-- Display Mode -->
								<div v-else class="text-foreground">
									<span v-if="secretKeys.has(item.key)">••••••••</span>
									<span v-else-if="typeof item.value === 'boolean'">
										{{ item.value ? 'true' : 'false' }}
									</span>
									<span v-else>{{ item.value || '-' }}</span>
								</div>
							</td>
							<td class="text-muted-foreground">
								<span v-if="typeof item.default === 'boolean'">
									{{ item.default ? 'true' : 'false' }}
								</span>
								<span v-else>{{ item.default || '-' }}</span>
							</td>
							<td class="text-muted-foreground">
								{{ item.updated_at ? formatTime(item.updated_at) : '-' }}
							</td>
							<td>
								<div v-if="editingKey === item.key" class="flex justify-end gap-2">
									<button :disabled="operating" class="app-link" @click="handleSave(item)">
										{{ t('common.save') }}
									</button>
									<button class="text-muted-foreground hover:text-foreground" @click="cancelEdit">
										{{ t('common.cancel') }}
									</button>
								</div>
								<div v-else class="flex justify-end gap-2">
									<button class="app-link" @click="startEdit(item)">{{ t('common.edit') }}</button>
									<button
										class="text-muted-foreground hover:text-foreground"
										@click="confirmReset(item.key)"
									>
										{{ t('common.reset') }}
									</button>
								</div>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>

		<AppDialog
			v-model:open="isPasswordDialogOpen"
			:title="t('settings.passwordDialog.title')"
		>
			<div class="space-y-4">
				<div class="space-y-1.5">
					<label class="app-field-label block">
						{{ t('settings.passwordDialog.oldPassword') }}
					</label>
					<input
						v-model="passwordForm.old_password"
						type="password"
						:placeholder="t('settings.passwordDialog.oldPasswordPlaceholder')"
						class="app-input"
						:class="passwordErrors.old_password ? 'app-input-error' : ''"
					/>
					<p v-if="passwordErrors.old_password" class="app-field-error text-xs">
						{{ passwordErrors.old_password }}
					</p>
				</div>
				<div class="space-y-1.5">
					<label class="app-field-label block">
						{{ t('settings.passwordDialog.newPassword') }}
					</label>
					<input
						v-model="passwordForm.new_password"
						type="password"
						:placeholder="t('settings.passwordDialog.newPasswordPlaceholder')"
						class="app-input"
						:class="passwordErrors.new_password ? 'app-input-error' : ''"
					/>
					<p v-if="passwordErrors.new_password" class="app-field-error text-xs">
						{{ passwordErrors.new_password }}
					</p>
				</div>
				<div class="space-y-1.5">
					<label class="app-field-label block">
						{{ t('settings.passwordDialog.confirmPassword') }}
					</label>
					<input
						v-model="passwordForm.confirm_password"
						type="password"
						:placeholder="t('settings.passwordDialog.confirmPasswordPlaceholder')"
						class="app-input"
						:class="passwordErrors.confirm_password ? 'app-input-error' : ''"
					/>
					<p v-if="passwordErrors.confirm_password" class="app-field-error text-xs">
						{{ passwordErrors.confirm_password }}
					</p>
				</div>
			</div>
			<template #footer>
				<button class="app-button" @click="isPasswordDialogOpen = false">
					{{ t('common.cancel') }}
				</button>
				<button
					class="app-button-primary"
					:disabled="passwordLoading"
					@click="handleChangePassword"
				>
					{{ t('settings.changePassword') }}
				</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isResetDialogOpen"
			:title="t('settings.resetDialog.title')"
			width-class="w-[min(400px,calc(100vw-32px))]"
		>
			<p class="text-sm text-muted-foreground">{{ t('settings.resetDialog.description') }}</p>
			<template #footer>
				<button class="app-button" @click="isResetDialogOpen = false">
					{{ t('common.cancel') }}
				</button>
				<button class="app-button-primary" :disabled="operating" @click="handleReset">
					{{ t('common.reset') }}
				</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { Key } from 'lucide-vue-next';
	import { computed, onMounted, reactive, ref } from 'vue';
	import { ToolbarRoot } from 'reka-ui';
	import { useI18n } from 'vue-i18n';
	import { settingApi } from '@/api/settings';
	import AppDialog from '@/components/AppDialog.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
	import SearchControl from '@/components/SearchControl.vue';
	import SelectControl from '@/components/SelectControl.vue';
	import { useStatusAsync } from '@/composables/useStatusAsync';
	import { useToast } from '@/composables/useToast';
	import { useAuthStore } from '@/stores/auth';
	import type { ConfigItemResp, SystemConfigResp } from '@/types/cd/settings';
	import { formatTime } from '@/utils/time';

	const { t } = useI18n();
	const authStore = useAuthStore();
	const toast = useToast();

	const config = ref<SystemConfigResp>();
	const searchText = ref('');
	const { loading: configLoading, execute } = useStatusAsync();
	const { loading: operating, execute: executeOp } = useStatusAsync();
	const { loading: passwordLoading, execute: executeChangePassword } = useStatusAsync();
	const needsRestart = ref(false);

	const filteredConfig = computed(() => {
		if (!config.value?.items) {
			return [];
		}
		if (!searchText.value.trim()) {
			return config.value.items;
		}
		const search = searchText.value.toLowerCase();
		return config.value.items.filter(
			(item) =>
				item.key.toLowerCase().includes(search) || item.description?.toLowerCase().includes(search)
		);
	});

	const selectOptions: Record<string, string[]> = {
		cert__letsencrypt__challenge: ['http', 'dns'],
	};
	const booleanOptions = [
		{ value: 'true', label: 'true' },
		{ value: 'false', label: 'false' },
	];
	const secretKeys = new Set(['jwt__secret_key']);

	const editingKey = ref<string>();
	const editingStr = ref('');
	const editingBoolStr = ref('false');

	const isPasswordDialogOpen = ref(false);
	const passwordForm = reactive({
		old_password: '',
		new_password: '',
		confirm_password: '',
	});

	function getSettingOptions(key: string) {
		return (selectOptions[key] ?? []).map((option) => ({
			value: option,
			label: option,
		}));
	}
	const passwordErrors = reactive({
		old_password: '',
		new_password: '',
		confirm_password: '',
	});

	async function fetchConfig() {
		try {
			await execute(async () => {
				config.value = await settingApi.getConfig();
			});
		} catch {
			toast.error(t('settings.loadFailed'));
		}
	}

	function startEdit(record: ConfigItemResp) {
		editingKey.value = record.key;
		if (typeof record.default === 'boolean') {
			editingBoolStr.value = String(record.value);
		} else {
			editingStr.value = secretKeys.has(record.key) ? '' : String(record.value ?? '');
		}
	}

	function cancelEdit() {
		editingKey.value = undefined;
	}

	async function handleSave(record: ConfigItemResp) {
		if (secretKeys.has(record.key) && !editingStr.value) {
			cancelEdit();
			return;
		}
		try {
			await executeOp(async () => {
				const value =
					typeof record.default === 'boolean' ? editingBoolStr.value === 'true' : editingStr.value;
				config.value = await settingApi.updateConfig({ key: record.key, value });
				needsRestart.value = true;
				editingKey.value = undefined;
				toast.warning(t('settings.saveSuccess'));
			});
		} catch {
			toast.error(t('settings.saveFailed'));
		}
	}

	const isResetDialogOpen = ref(false);
	const pendingResetKey = ref('');

	function confirmReset(key: string) {
		pendingResetKey.value = key;
		isResetDialogOpen.value = true;
	}

	async function handleReset() {
		try {
			await executeOp(async () => {
				config.value = await settingApi.resetConfig({
					keys: [pendingResetKey.value],
				});
				needsRestart.value = true;
				isResetDialogOpen.value = false;
				toast.warning(t('settings.resetSuccess'));
			});
		} catch {
			toast.error(t('settings.resetFailed'));
		}
	}

	function validatePassword() {
		passwordErrors.old_password = passwordForm.old_password
			? ''
			: t('settings.passwordDialog.oldPasswordRequired');
		passwordErrors.new_password =
			passwordForm.new_password.length >= 6 ? '' : t('settings.passwordDialog.newPasswordTooShort');
		passwordErrors.confirm_password =
			passwordForm.confirm_password === passwordForm.new_password
				? ''
				: t('settings.passwordDialog.passwordMismatch');
		return (
			!passwordErrors.old_password &&
			!passwordErrors.new_password &&
			!passwordErrors.confirm_password
		);
	}

	async function handleChangePassword() {
		if (!validatePassword()) {
			return;
		}
		try {
			await executeChangePassword(async () => {
				await authStore.changePassword(passwordForm.old_password, passwordForm.new_password);
				toast.success(t('settings.passwordDialog.changeSuccess'));
				isPasswordDialogOpen.value = false;
				Object.assign(passwordForm, {
					old_password: '',
					new_password: '',
					confirm_password: '',
				});
			});
		} catch (error) {
			toast.error(
				error instanceof Error ? error.message : t('settings.passwordDialog.changeFailed')
			);
		}
	}

	onMounted(fetchConfig);
</script>
