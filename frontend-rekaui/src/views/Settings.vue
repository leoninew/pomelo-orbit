
<template>
	<div class="space-y-6">
		<ToolbarRoot class="flex items-center justify-between gap-6" aria-label="系统设置工具栏">
			<SearchControl
				v-model="searchText"
				placeholder="搜索配置项"
				:loading="configLoading"
			/>
			<button
				class="flex h-10 items-center gap-2 rounded-md bg-primary px-5 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90"
				@click="passwordModalOpen = true"
			>
				<Key class="size-4" />
				修改密码
			</button>
		</ToolbarRoot>

		<!-- Restart Warning -->
		<div v-if="needsRestart" class="rounded-lg border border-amber-200 bg-amber-50 p-4">
			<p class="text-sm text-amber-800">
				⚠️ 配置已更新，请重启服务以使更改生效
			</p>
		</div>

		<!-- Config Table -->
		<div v-if="configLoading" class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div class="flex justify-center py-16">
				<div class="size-8 animate-spin rounded-full border-4 border-primary/20 border-t-primary" />
			</div>
		</div>
		<div v-else-if="filteredConfig.length === 0" class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<div class="text-center py-16 text-muted-foreground">
				<p class="text-sm">{{ searchText ? '未找到匹配的配置项' : '暂无配置' }}</p>
			</div>
		</div>
		<div v-else class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
			<table class="w-full">
				<thead class="border-b border-border bg-muted/30">
					<tr>
						<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">
							配置项
						</th>
						<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">
							当前值
						</th>
						<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">
							默认值
						</th>
						<th class="px-6 py-4 text-left text-xs font-normal text-muted-foreground">
							更新时间
						</th>
						<th class="px-6 py-4 text-right text-xs font-normal text-muted-foreground">
							操作
						</th>
					</tr>
				</thead>
				<tbody class="divide-y divide-border">
					<tr v-for="item in filteredConfig" :key="item.key" class="transition-colors hover:bg-muted/30">
						<td class="px-6 py-5 text-sm">
							<div class="text-foreground">{{ item.key }}</div>
							<div v-if="item.description" class="text-xs text-muted-foreground mt-1">{{ item.description }}</div>
						</td>
						<td class="px-6 py-5 text-sm">
							<!-- Editing Mode -->
							<div v-if="editingKey === item.key">
								<!-- Boolean -->
								<SelectControl
									v-if="typeof item.default === 'boolean'"
									v-model="editingBool"
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
									:placeholder="secretKeys.has(item.key) ? '留空表示不修改' : ''"
									class="h-9 rounded-md border border-input bg-background px-3 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
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
						<td class="px-6 py-5 text-sm text-muted-foreground">
							<span v-if="typeof item.default === 'boolean'">
								{{ item.default ? 'true' : 'false' }}
							</span>
							<span v-else>{{ item.default || '-' }}</span>
						</td>
						<td class="px-6 py-5 text-sm text-muted-foreground">
							{{ item.updated_at ? formatTime(item.updated_at) : '-' }}
						</td>
						<td class="px-6 py-5 text-sm">
							<div v-if="editingKey === item.key" class="flex justify-end gap-2">
								<button
									:disabled="operating"
									class="text-primary hover:underline disabled:opacity-50"
									@click="handleSave(item)"
								>
									保存
								</button>
								<button
									class="text-muted-foreground hover:text-foreground"
									@click="cancelEdit"
								>
									取消
								</button>
							</div>
							<div v-else class="flex justify-end gap-2">
								<button
									class="text-primary hover:underline"
									@click="startEdit(item)"
								>
									编辑
								</button>
								<button
									class="text-muted-foreground hover:text-foreground"
									@click="confirmReset(item.key)"
								>
									重置
								</button>
							</div>
						</td>
					</tr>
				</tbody>
			</table>
		</div>

		<!-- Change Password Modal -->
		<DialogRoot v-model:open="passwordModalOpen">
			<DialogPortal>
				<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
				<DialogContent
					class="fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[min(520px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
				>
					<div class="mb-5 space-y-1">
						<DialogTitle class="text-lg font-semibold text-foreground">修改密码</DialogTitle>
						<DialogDescription class="text-sm text-muted-foreground">
							请输入当前密码和新密码。
						</DialogDescription>
					</div>

					<div class="space-y-4">
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">当前密码</label>
							<input
								v-model="passwordForm.old_password"
								type="password"
								placeholder="输入当前密码"
								class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
								:class="passwordErrors.old_password ? 'border-destructive' : 'border-input'"
							/>
							<p v-if="passwordErrors.old_password" class="text-xs text-destructive">
								{{ passwordErrors.old_password }}
							</p>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">新密码</label>
							<input
								v-model="passwordForm.new_password"
								type="password"
								placeholder="输入新密码（至少 6 位）"
								class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
								:class="passwordErrors.new_password ? 'border-destructive' : 'border-input'"
							/>
							<p v-if="passwordErrors.new_password" class="text-xs text-destructive">
								{{ passwordErrors.new_password }}
							</p>
						</div>
						<div class="space-y-1.5">
							<label class="block text-sm font-medium text-foreground">确认新密码</label>
							<input
								v-model="passwordForm.confirm_password"
								type="password"
								placeholder="再次输入新密码"
								class="w-full rounded-md border bg-background px-3 py-2 text-sm text-foreground outline-none transition-colors placeholder:text-muted-foreground focus:border-ring focus:ring-2 focus:ring-ring/20"
								:class="passwordErrors.confirm_password ? 'border-destructive' : 'border-input'"
							/>
							<p v-if="passwordErrors.confirm_password" class="text-xs text-destructive">
								{{ passwordErrors.confirm_password }}
							</p>
						</div>
					</div>

					<div class="mt-6 flex justify-end gap-2">
						<DialogClose as-child>
							<button
								class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
							>
								取消
							</button>
						</DialogClose>
						<button
							class="flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
							:disabled="passwordLoading"
							@click="handleChangePassword"
						>
							<span
								v-if="passwordLoading"
								class="size-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent"
							/>
							修改密码
						</button>
					</div>
				</DialogContent>
			</DialogPortal>
		</DialogRoot>

		<!-- Reset Confirmation -->
		<DialogRoot v-model:open="resetModalOpen">
			<DialogPortal>
				<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
				<DialogContent
					class="fixed left-1/2 top-1/2 z-50 w-[min(400px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
				>
					<div class="mb-5 space-y-1">
						<DialogTitle class="text-lg font-semibold text-foreground">确认重置</DialogTitle>
						<DialogDescription class="text-sm text-muted-foreground">
							确定要将此配置项重置为默认值吗？
						</DialogDescription>
					</div>

					<div class="flex justify-end gap-2">
						<DialogClose as-child>
							<button
								class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
							>
								取消
							</button>
						</DialogClose>
						<button
							class="flex items-center gap-2 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
							:disabled="operating"
							@click="handleReset"
						>
							<span
								v-if="operating"
								class="size-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent"
							/>
							重置
						</button>
					</div>
				</DialogContent>
			</DialogPortal>
		</DialogRoot>
	</div>
</template>

<script setup lang="ts">
import { Key } from 'lucide-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';
import {
	DialogClose,
	DialogContent,
	DialogDescription,
	DialogOverlay,
	DialogPortal,
	DialogRoot,
	DialogTitle,
	ToolbarRoot,
} from 'reka-ui';
import { settingApi } from '@/api/settings';
import SearchControl from '@/components/SearchControl.vue';
import SelectControl from '@/components/SelectControl.vue';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { useAuthStore } from '@/stores/auth';
import type { ConfigItemResp, SystemConfigResp } from '@/types/cd/settings';
import { formatTime } from '@/utils/time';

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
			item.key.toLowerCase().includes(search) ||
			item.description?.toLowerCase().includes(search)
	);
});

const selectOptions: Record<string, string[]> = {
	cert__letsencrypt__challenge: ['http', 'dns'],
};
const booleanOptions = [
	{ value: true, label: 'true' },
	{ value: false, label: 'false' },
];
const secretKeys = new Set(['jwt__secret_key']);

const editingKey = ref<string>();
const editingStr = ref('');
const editingBool = ref(false);

const passwordModalOpen = ref(false);
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
		toast.error('加载配置失败');
	}
}

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
			toast.warning('已保存，请重启服务以生效');
		});
	} catch {
		toast.error('保存失败');
	}
}

const resetModalOpen = ref(false);
const pendingResetKey = ref('');

function confirmReset(key: string) {
	pendingResetKey.value = key;
	resetModalOpen.value = true;
}

async function handleReset() {
	try {
		await executeOp(async () => {
			config.value = await settingApi.resetConfig({
				keys: [pendingResetKey.value],
			});
			needsRestart.value = true;
			resetModalOpen.value = false;
			toast.warning('已重置，请重启服务以生效');
		});
	} catch {
		toast.error('重置失败');
	}
}

function validatePassword() {
	passwordErrors.old_password = passwordForm.old_password ? '' : '请输入当前密码';
	passwordErrors.new_password = passwordForm.new_password.length >= 6 ? '' : '密码至少 6 位';
	passwordErrors.confirm_password =
		passwordForm.confirm_password === passwordForm.new_password ? '' : '两次输入的密码不一致';
	return (
		!passwordErrors.old_password && !passwordErrors.new_password && !passwordErrors.confirm_password
	);
}

async function handleChangePassword() {
	if (!validatePassword()) {
		return;
	}
	try {
		await executeChangePassword(async () => {
			await authStore.changePassword(passwordForm.old_password, passwordForm.new_password);
			toast.success('密码修改成功');
			passwordModalOpen.value = false;
			Object.assign(passwordForm, {
				old_password: '',
				new_password: '',
				confirm_password: '',
			});
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '密码修改失败');
	}
}

onMounted(fetchConfig);
</script>
