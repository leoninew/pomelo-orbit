<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>系统设置</h2>
		</div>

		<a-card title="用户信息">
			<template #extra>
				<a-button type="primary" @click="showChangePasswordModal = true">修改密码</a-button>
			</template>
			<a-descriptions v-if="authStore.user" :column="1" bordered size="small">
				<a-descriptions-item label="用户名">
					{{ authStore.user.username }}
				</a-descriptions-item>
				<a-descriptions-item label="创建时间">
					{{ formatTime(authStore.user.created_at) }}
				</a-descriptions-item>
				<a-descriptions-item label="上次登录">
					{{ formatTime(authStore.user.last_login_at) }}
				</a-descriptions-item>
			</a-descriptions>
		</a-card>

		<a-card title="运行时配置" :loading="configLoading">
			<a-alert
				v-if="needsRestart"
				message="配置已变更，请重启服务以生效"
				type="warning"
				show-icon
				style="margin-bottom: 12px"
			/>
			<a-table
				v-if="config"
				:columns="configColumns"
				:data-source="config.items"
				:pagination="false"
				size="small"
				row-key="key"
			>
				<template #bodyCell="{ column, record }">
					<template v-if="column.key === 'key'">
						<span>{{ record.key }}</span>
					</template>

					<template v-else-if="column.key === 'default'">
						<a-tag
							v-if="typeof record.default === 'boolean'"
							:color="record.default ? 'green' : 'default'"
						>
							{{ record.default ? '已启用' : '未启用' }}
						</a-tag>
						<span v-else>
							{{ record.default === '' || record.default == null ? '-' : record.default }}
						</span>
					</template>

					<template v-else-if="column.key === 'value'">
						<template v-if="editingKey === record.key">
							<a-switch v-if="typeof record.default === 'boolean'" v-model:checked="editingBool" />
							<a-select
								v-else-if="selectOptions[record.key]"
								v-model:value="editingStr"
								style="width: 120px"
							>
								<a-select-option v-for="opt in selectOptions[record.key]" :key="opt" :value="opt">
									{{ opt }}
								</a-select-option>
							</a-select>
							<a-input-password
								v-else-if="secretKeys.has(record.key)"
								v-model:value="editingStr"
								style="width: 260px"
								placeholder="留空则不修改"
							/>
							<a-input v-else v-model:value="editingStr" style="width: 260px" />
						</template>
						<template v-else>
							<a-tag
								v-if="typeof record.value === 'boolean'"
								:color="record.is_overridden ? 'warning' : record.value ? 'green' : 'default'"
							>
								{{ record.value ? '已启用' : '未启用' }}
							</a-tag>
							<a-typography-text v-else :type="record.is_overridden ? 'warning' : undefined">
								{{ record.value === '' || record.value == null ? '-' : record.value }}
							</a-typography-text>
						</template>
					</template>

					<template v-else-if="column.key === 'action'">
						<template v-if="editingKey === record.key">
							<a-button type="link" size="small" :loading="operating" @click="handleSave(record)">
								保存
							</a-button>
							<a-button type="link" size="small" :disabled="operating" @click="cancelEdit">
								取消
							</a-button>
						</template>
						<template v-else>
							<a-button type="link" size="small" @click="startEdit(record)">编辑</a-button>
							<a-popconfirm
								v-if="record.is_overridden"
								:title="`重置「${record.key}」为默认值？`"
								ok-text="确认"
								cancel-text="取消"
								@confirm="handleReset(record.key)"
							>
								<a-button type="link" size="small" danger>重置</a-button>
							</a-popconfirm>
						</template>
					</template>
				</template>
			</a-table>
		</a-card>

		<!-- 修改密码 -->
		<a-modal v-model:open="showChangePasswordModal" title="修改密码">
			<a-form
				ref="passwordFormRef"
				:model="passwordForm"
				:rules="passwordFormRules"
				:label-col="{ span: 6 }"
				:wrapper-col="{ span: 16 }"
			>
				<a-form-item label="当前密码" name="old_password">
					<a-input-password
						v-model:value="passwordForm.old_password"
						placeholder="请输入当前密码"
					/>
				</a-form-item>
				<a-form-item label="新密码" name="new_password">
					<a-input-password
						v-model:value="passwordForm.new_password"
						placeholder="请输入新密码（至少6位）"
					/>
				</a-form-item>
				<a-form-item label="确认密码" name="confirm_password">
					<a-input-password
						v-model:value="passwordForm.confirm_password"
						placeholder="请再次输入新密码"
					/>
				</a-form-item>
			</a-form>
			<template #footer>
				<a-button type="primary" :loading="passwordLoading" @click="handleChangePassword">
					确定
				</a-button>
				<a-button @click="showChangePasswordModal = false">取消</a-button>
			</template>
		</a-modal>
	</a-space>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';
import { onMounted, reactive, ref } from 'vue';
import { formatTime } from '@/utils/time';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useAuthStore } from '@/stores/auth';
import { settingApi } from '@/api/settings';
import type { ConfigItemResp, SystemConfigResp } from '@/types/api';

const authStore = useAuthStore();

const configColumns = [
	{ title: '配置项', key: 'key', width: 280 },
	{ title: '默认值', key: 'default', width: 160 },
	{ title: '当前值', key: 'value', width: 280 },
	{ title: '操作', key: 'action', width: 120 },
];

// key -> 可选值列表（前端维护，后端不感知）
const selectOptions: Record<string, string[]> = {
	cert__letsencrypt__challenge: ['http', 'dns'],
};

// 脱敏字段集合（前端维护）
const secretKeys = new Set(['jwt__secret_key']);

// ── 配置加载 ────────────────────────────────────────────
const config = ref<SystemConfigResp>();
const { loading: configLoading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const needsRestart = ref(false);

async function fetchConfig() {
	try {
		await execute(async () => {
			config.value = await settingApi.getConfig();
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '加载配置失败');
	}
}

onMounted(() => {
	fetchConfig();
});

// ── 行内编辑 ────────────────────────────────────────────
const editingKey = ref<string>();
const editingStr = ref('');
const editingBool = ref(false);

function startEdit(record: ConfigItemResp) {
	editingKey.value = record.key;
	if (typeof record.default === 'boolean') {
		editingBool.value = record.value as boolean;
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
			const value = typeof record.default === 'boolean' ? editingBool.value : editingStr.value;
			config.value = await settingApi.updateConfig({ key: record.key, value });
			needsRestart.value = true;
			editingKey.value = undefined;
			message.warning('已保存，请重启服务以生效');
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '保存失败');
	}
}

async function handleReset(key: string) {
	try {
		await executeOp(async () => {
			config.value = await settingApi.resetConfig({ keys: [key] });
			needsRestart.value = true;
			message.warning('已重置，请重启服务以生效');
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '重置失败');
	}
}

// ── 修改密码 ────────────────────────────────────────────
const showChangePasswordModal = ref(false);
const passwordFormRef = ref<FormInstance>();
const { loading: passwordLoading, execute: executeChangePassword } = useStatusAsync();

const passwordForm = reactive({
	old_password: '',
	new_password: '',
	confirm_password: '',
});

const passwordFormRules = {
	old_password: [{ required: true, message: '请输入当前密码' }],
	new_password: [
		{ required: true, message: '请输入新密码' },
		{ min: 6, message: '密码至少6位' },
	],
	confirm_password: [
		{ required: true, message: '请确认新密码' },
		{
			validator: (_rule: unknown, value: string) => {
				if (value !== passwordForm.new_password) return Promise.reject('两次输入的密码不一致');
				return Promise.resolve();
			},
		},
	],
};

async function handleChangePassword() {
	try {
		await passwordFormRef.value?.validate();
	} catch {
		return;
	}
	try {
		await executeChangePassword(async () => {
			await authStore.changePassword(passwordForm.old_password, passwordForm.new_password);
			message.success('密码修改成功');
			showChangePasswordModal.value = false;
			passwordForm.old_password = '';
			passwordForm.new_password = '';
			passwordForm.confirm_password = '';
		});
	} catch (error) {
		message.error(error instanceof Error ? error.message : '密码修改失败');
	}
}
</script>

<style scoped>
.page-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	min-height: 32px;
}

.page-header h2 {
	margin: 0;
}
</style>
