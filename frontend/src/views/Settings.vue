<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between">
			<h1 class="text-xl font-semibold">系统设置</h1>
		</div>

		<!-- User info -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<div class="flex items-center justify-between mb-3">
					<h2 class="font-semibold">用户信息</h2>
					<button class="btn btn-sm btn-primary" @click="passwordModalRef?.showModal()">修改密码</button>
				</div>
				<dl v-if="authStore.user" class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
					<div class="flex gap-2"><dt class="text-base-content/50 w-24 shrink-0">用户名</dt><dd class="font-medium">{{ authStore.user.username }}</dd></div>
					<div class="flex gap-2"><dt class="text-base-content/50 w-24 shrink-0">创建时间</dt><dd class="text-xs text-base-content/60">{{ formatTime(authStore.user.created_at) }}</dd></div>
					<div class="flex gap-2"><dt class="text-base-content/50 w-24 shrink-0">上次登录</dt><dd class="text-xs text-base-content/60">{{ formatTime(authStore.user.last_login_at) }}</dd></div>
				</dl>
			</div>
		</div>

		<!-- Runtime config -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<h2 class="font-semibold mb-3">运行时配置</h2>
				<div v-if="needsRestart" role="alert" class="alert alert-warning text-sm mb-4">
					<span>配置已变更，请重启服务以生效</span>
				</div>
				<div v-if="configLoading" class="flex justify-center py-8"><span class="loading loading-spinner loading-md text-primary" /></div>
				<div v-else-if="config" class="overflow-x-auto">
					<table class="table">
						<thead>
							<tr class="text-base-content/60">
								<th>配置项</th><th>默认值</th><th>当前值</th><th>操作</th>
							</tr>
						</thead>
						<tbody>
							<tr v-for="item in config.items" :key="item.key" class="hover">
								<td><code class="text-xs">{{ item.key }}</code></td>
								<td class="text-xs text-base-content/60">
									<span v-if="typeof item.default === 'boolean'">
										<span class="badge badge-xs" :class="item.default ? 'badge-outline badge-success' : 'badge-ghost'">{{ item.default ? '已启用' : '未启用' }}</span>
									</span>
									<span v-else>{{ item.default === '' || item.default == null ? '—' : item.default }}</span>
								</td>
								<td>
									<template v-if="editingKey === item.key">
										<input v-if="typeof item.default === 'boolean'" type="checkbox" class="toggle toggle-sm toggle-primary" :checked="editingBool" @change="editingBool = ($event.target as HTMLInputElement).checked" />
										<select v-else-if="selectOptions[item.key]" v-model="editingStr" class="select select-bordered select-xs w-32">
											<option v-for="opt in selectOptions[item.key]" :key="opt" :value="opt">{{ opt }}</option>
										</select>
										<input v-else-if="secretKeys.has(item.key)" v-model="editingStr" type="password" class="input input-bordered input-xs w-64" placeholder="留空则不修改" />
										<input v-else v-model="editingStr" type="text" class="input input-bordered input-xs w-64" />
									</template>
									<template v-else>
										<span v-if="typeof item.value === 'boolean'">
											<span class="badge badge-xs" :class="item.is_overridden ? 'badge-outline badge-warning' : item.value ? 'badge-outline badge-success' : 'badge-ghost'">{{ item.value ? '已启用' : '未启用' }}</span>
										</span>
										<span v-else class="text-xs" :class="item.is_overridden ? 'text-warning' : 'text-base-content/70'">
											{{ item.value === '' || item.value == null ? '—' : item.value }}
										</span>
									</template>
								</td>
								<td>
									<template v-if="editingKey === item.key">
										<div class="flex items-center gap-2 text-xs">
											<button class="link link-primary" :class="{ 'opacity-50': operating }" @click="handleSave(item)">保存</button>
											<button class="link" @click="cancelEdit">取消</button>
										</div>
									</template>
									<template v-else>
										<div class="flex items-center gap-2 text-xs">
											<button class="link link-primary" @click="startEdit(item)">编辑</button>
											<button v-if="item.is_overridden" class="link link-error" @click="handleReset(item.key)">重置</button>
										</div>
									</template>
								</td>
							</tr>
						</tbody>
					</table>
				</div>
			</div>
		</div>

		<!-- Change password modal -->
		<dialog ref="passwordModalRef" class="modal">
			<div class="modal-box w-full max-w-md">
				<h3 class="font-bold text-lg mb-4">修改密码</h3>
				<div class="flex flex-col gap-3">
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">当前密码</span></div>
						<input v-model="passwordForm.old_password" type="password" class="input input-bordered input-sm" :class="{ 'input-error': passwordErrors.old_password }" />
						<div v-if="passwordErrors.old_password" class="label pt-1"><span class="label-text-alt text-error">{{ passwordErrors.old_password }}</span></div>
					</label>
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">新密码</span></div>
						<input v-model="passwordForm.new_password" type="password" class="input input-bordered input-sm" :class="{ 'input-error': passwordErrors.new_password }" placeholder="至少 6 位" />
						<div v-if="passwordErrors.new_password" class="label pt-1"><span class="label-text-alt text-error">{{ passwordErrors.new_password }}</span></div>
					</label>
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">确认密码</span></div>
						<input v-model="passwordForm.confirm_password" type="password" class="input input-bordered input-sm" :class="{ 'input-error': passwordErrors.confirm_password }" />
						<div v-if="passwordErrors.confirm_password" class="label pt-1"><span class="label-text-alt text-error">{{ passwordErrors.confirm_password }}</span></div>
					</label>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="passwordLoading" @click="handleChangePassword">
						<span v-if="passwordLoading" class="loading loading-spinner loading-xs" />确定
					</button>
					<button class="btn btn-ghost" @click="passwordModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { useAuthStore } from '@/stores/auth';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { settingApi } from '@/api/settings';
import { formatTime } from '@/utils/time';
import type { ConfigItemResp, SystemConfigResp } from '@/types/api';

const authStore = useAuthStore();
const toast = useToast();

const config = ref<SystemConfigResp>();
const { loading: configLoading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const { loading: passwordLoading, execute: executeChangePassword } = useStatusAsync();
const needsRestart = ref(false);

const selectOptions: Record<string, string[]> = { 'cert__letsencrypt__challenge': ['http', 'dns'] };
const secretKeys = new Set(['jwt__secret_key']);

const editingKey = ref<string>();
const editingStr = ref('');
const editingBool = ref(false);

const passwordModalRef = ref<HTMLDialogElement>();
const passwordForm = reactive({ old_password: '', new_password: '', confirm_password: '' });
const passwordErrors = reactive({ old_password: '', new_password: '', confirm_password: '' });

async function fetchConfig() {
	try {
		await execute(async () => { config.value = await settingApi.getConfig(); });
	} catch { toast.error('加载配置失败'); }
}

function startEdit(record: ConfigItemResp) {
	editingKey.value = record.key;
	if (typeof record.default === 'boolean') editingBool.value = record.value as boolean;
	else editingStr.value = secretKeys.has(record.key) ? '' : String(record.value ?? '');
}

function cancelEdit() { editingKey.value = undefined; }

async function handleSave(record: ConfigItemResp) {
	if (secretKeys.has(record.key) && !editingStr.value) { cancelEdit(); return; }
	try {
		await executeOp(async () => {
			const value = typeof record.default === 'boolean' ? editingBool.value : editingStr.value;
			config.value = await settingApi.updateConfig({ key: record.key, value });
			needsRestart.value = true; editingKey.value = undefined;
			toast.warning('已保存，请重启服务以生效');
		});
	} catch { toast.error('保存失败'); }
}

async function handleReset(key: string) {
	try {
		await executeOp(async () => {
			config.value = await settingApi.resetConfig({ keys: [key] });
			needsRestart.value = true;
			toast.warning('已重置，请重启服务以生效');
		});
	} catch { toast.error('重置失败'); }
}

function validatePassword() {
	passwordErrors.old_password = passwordForm.old_password ? '' : '请输入当前密码';
	passwordErrors.new_password = passwordForm.new_password.length >= 6 ? '' : '密码至少 6 位';
	passwordErrors.confirm_password = passwordForm.confirm_password === passwordForm.new_password ? '' : '两次输入的密码不一致';
	return !passwordErrors.old_password && !passwordErrors.new_password && !passwordErrors.confirm_password;
}

async function handleChangePassword() {
	if (!validatePassword()) return;
	try {
		await executeChangePassword(async () => {
			await authStore.changePassword(passwordForm.old_password, passwordForm.new_password);
			toast.success('密码修改成功');
			passwordModalRef.value?.close();
			Object.assign(passwordForm, { old_password: '', new_password: '', confirm_password: '' });
		});
	} catch (error) { toast.error(error instanceof Error ? error.message : '密码修改失败'); }
}

onMounted(fetchConfig);
</script>
