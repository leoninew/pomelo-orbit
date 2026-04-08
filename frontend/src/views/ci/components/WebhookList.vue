<template>
	<div class="flex flex-col gap-4">
		<!-- Header -->
		<div class="flex items-center justify-between">
			<h3 class="font-semibold">Webhook 配置</h3>
			<button class="btn btn-sm btn-primary gap-1" @click="openCreateModal">
				<Plus class="size-3.5" />
				添加 Webhook
			</button>
		</div>

		<!-- Webhook list -->
		<div v-if="webhooks.length === 0" class="text-sm text-base-content/60 py-4 text-center">
			未配置 Webhook
		</div>
		<div v-else class="overflow-x-auto">
			<table class="table">
				<thead>
					<tr class="text-base-content/60">
						<th>名称</th>
						<th>模板</th>
						<th>分支过滤</th>
						<th>Webhook URL</th>
						<th>状态</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					<tr v-for="wh in webhooks" :key="wh.id" class="hover">
						<td>{{ wh.name }}</td>
						<td>
							<router-link
								:to="`/ci/templates/${wh.template_id}`"
								class="link link-primary text-xs"
							>
								{{ getTemplateName(wh.template_id) }}
							</router-link>
						</td>
						<td>
							<code v-if="wh.branch_filter" class="text-xs">{{ wh.branch_filter }}</code>
							<span v-else class="text-error text-xs">拒绝所有分支</span>
						</td>
						<td>
							<div class="flex items-center gap-1">
								<code class="text-xs truncate max-w-48">{{ webhookUrl(wh.id) }}</code>
								<button class="btn btn-xs btn-ghost" title="复制 URL" @click="copyUrl(wh.id)">
									<Copy class="size-3" />
								</button>
							</div>
						</td>
						<td>
							<div class="badge" :class="wh.enabled ? 'badge-success' : 'badge-ghost'">
								{{ wh.enabled ? '已启用' : '已禁用' }}
							</div>
						</td>
						<td>
							<div class="flex items-center gap-2">
								<button class="link link-primary text-xs" @click="openEditModal(wh)">编辑</button>
								<button class="link link-error text-xs" @click="handleDelete(wh)">删除</button>
							</div>
						</td>
					</tr>
				</tbody>
			</table>
		</div>

		<!-- Create/Edit modal -->
		<dialog ref="modalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg mb-4">
					{{ editingWebhook ? '编辑 Webhook' : '添加 Webhook' }}
				</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">名称</legend>
						<input
							v-model="form.name"
							type="text"
							class="input w-full"
							placeholder="如：push → 测试"
						/>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">模板</legend>
						<select v-model="form.template_id" class="select w-full">
							<option value="">选择模板</option>
							<option v-for="tpl in templates" :key="tpl.id" :value="tpl.id">
								{{ tpl.name }}
							</option>
						</select>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">分支过滤（可选）</legend>
						<input
							v-model="form.branch_filter"
							type="text"
							class="input w-full"
							placeholder="如：main 或 release/*"
						/>
						<p class="fieldset-label text-base-content/50">
							留空则拒绝所有分支，支持 glob 模式（如 main、release/*）
						</p>
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">签名密钥</legend>
						<input
							v-model="form.secret"
							type="password"
							class="input w-full"
							placeholder="输入 HMAC 签名密钥"
						/>
						<p class="fieldset-label text-base-content/50">
							{{ editingWebhook ? '留空则不修改' : '用于验证 Git 平台推送的签名' }}
						</p>
					</fieldset>
					<fieldset v-if="editingWebhook" class="fieldset">
						<legend class="fieldset-legend">状态</legend>
						<label class="flex items-center gap-2 cursor-pointer">
							<input v-model="form.enabled" type="checkbox" class="checkbox checkbox-sm" />
							<span class="text-sm">启用此 Webhook</span>
						</label>
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						保存
					</button>
					<button class="btn btn-ghost" @click="modalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Delete confirmation modal -->
		<dialog ref="deleteModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">删除 Webhook</h3>
				<p class="py-4 text-sm">
					确定要删除 Webhook「
					<strong>{{ deletingWebhook?.name }}</strong>
					」吗？
				</p>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="operating" @click="confirmDelete">删除</button>
					<button class="btn btn-ghost" @click="deleteModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { Copy, Plus } from 'lucide-vue-next';
import { reactive, ref } from 'vue';
import { webhookApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { PipelineTemplate, ProjectWebhook } from '@/types/api';

const props = defineProps<{
	projectId: string
	webhooks: ProjectWebhook[]
	templates: PipelineTemplate[]
}>();

const emit = defineEmits<{
	refresh: []
}>();

const toast = useToast();
const { operating, execute: executeOp } = useStatusAsync();

const modalRef = ref<HTMLDialogElement>();
const deleteModalRef = ref<HTMLDialogElement>();
const editingWebhook = ref<ProjectWebhook>();
const deletingWebhook = ref<ProjectWebhook>();

const form = reactive({
	name: '',
	template_id: '',
	branch_filter: '',
	secret: '',
	enabled: true,
});

const getTemplateName = (templateId: string) => {
	const tpl = props.templates.find((t) => t.id === templateId);
	return tpl?.name || templateId;
};

const webhookUrl = (webhookId: string) => `${window.location.origin}/api/webhooks/${webhookId}`;

async function copyUrl(webhookId: string) {
	try {
		await navigator.clipboard.writeText(webhookUrl(webhookId));
		toast.success('URL 已复制');
	} catch {
		toast.error('复制失败，请手动复制');
	}
}

function openCreateModal() {
	editingWebhook.value = undefined;
	Object.assign(form, {
		name: '',
		template_id: '',
		branch_filter: '',
		secret: '',
		enabled: true,
	});
	modalRef.value?.showModal();
}

function openEditModal(wh: ProjectWebhook) {
	editingWebhook.value = wh;
	Object.assign(form, {
		name: wh.name,
		template_id: wh.template_id,
		branch_filter: wh.branch_filter || '',
		secret: '',
		enabled: wh.enabled,
	});
	modalRef.value?.showModal();
}

async function handleOk() {
	if (!form.name.trim()) {
		toast.error('请输入名称');
		return;
	}
	if (!form.template_id) {
		toast.error('请选择模板');
		return;
	}
	if (!editingWebhook.value && !form.secret) {
		toast.error('请输入签名密钥');
		return;
	}

	try {
		await executeOp(async () => {
			if (editingWebhook.value) {
				await webhookApi.update(props.projectId, editingWebhook.value.id, {
					name: form.name,
					template_id: form.template_id,
					branch_filter: form.branch_filter || null,
					secret: form.secret || undefined,
					enabled: form.enabled,
				});
				toast.success('更新成功');
			} else {
				await webhookApi.create(props.projectId, {
					name: form.name,
					template_id: form.template_id,
					secret: form.secret,
					branch_filter: form.branch_filter || null,
				});
				toast.success('创建成功');
			}
			modalRef.value?.close();
			emit('refresh');
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '操作失败');
	}
}

function handleDelete(wh: ProjectWebhook) {
	deletingWebhook.value = wh;
	deleteModalRef.value?.showModal();
}

async function confirmDelete() {
	if (!deletingWebhook.value) {
		return;
	}

	try {
		await executeOp(async () => {
			await webhookApi.delete(props.projectId, deletingWebhook.value?.id ?? '');
			toast.success('删除成功');
			deleteModalRef.value?.close();
			emit('refresh');
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '删除失败');
	}
}
</script>
