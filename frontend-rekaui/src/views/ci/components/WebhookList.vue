<script setup lang="ts">
import { Copy, Plus } from 'lucide-vue-next';
import { computed, reactive, ref } from 'vue';
import {
	DialogClose,
	DialogContent,
	DialogDescription,
	DialogOverlay,
	DialogPortal,
	DialogRoot,
	DialogTitle,
} from 'reka-ui';
import { webhookApi } from '@/api/ci';
import ComboboxSelect from '@/components/ComboboxSelect.vue';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { PipelineTemplate } from '@/types/ci/template';
import type { RepositoryWebhook } from '@/types/ci/webhook';

const props = defineProps<{
	repositoryId: string
	webhooks: RepositoryWebhook[]
	templates: PipelineTemplate[]
}>();

const emit = defineEmits<{
	refresh: []
}>();

const toast = useToast();
const { loading: operating, execute: executeOp } = useStatusAsync();

const isDialogOpen = ref(false);
const isDeleteDialogOpen = ref(false);
const editingWebhook = ref<RepositoryWebhook>();
const deletingWebhook = ref<RepositoryWebhook>();

const form = reactive({
	name: '',
	template_id: '',
	branch_filter: '',
	secret: '',
	enabled: true,
});
const templateOptions = computed(() =>
	props.templates.map((template) => ({
		value: template.id,
		label: template.name,
	}))
);

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
	isDialogOpen.value = true;
}

function openEditModal(wh: RepositoryWebhook) {
	editingWebhook.value = wh;
	Object.assign(form, {
		name: wh.name,
		template_id: wh.template_id,
		branch_filter: wh.branch_filter || '',
		secret: '',
		enabled: wh.enabled,
	});
	isDialogOpen.value = true;
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
				await webhookApi.update(props.repositoryId, editingWebhook.value.id, {
					name: form.name,
					template_id: form.template_id,
					branch_filter: form.branch_filter || null,
					secret: form.secret || undefined,
					enabled: form.enabled,
				});
				toast.success('更新成功');
			} else {
				await webhookApi.create(props.repositoryId, {
					name: form.name,
					template_id: form.template_id,
					secret: form.secret,
					branch_filter: form.branch_filter || null,
				});
				toast.success('创建成功');
			}
			isDialogOpen.value = false;
			emit('refresh');
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '操作失败');
	}
}

function handleDelete(wh: RepositoryWebhook) {
	deletingWebhook.value = wh;
	isDeleteDialogOpen.value = true;
}

async function confirmDelete() {
	if (!deletingWebhook.value) {
		return;
	}

	try {
		await executeOp(async () => {
			await webhookApi.delete(props.repositoryId, deletingWebhook.value?.id ?? '');
			toast.success('删除成功');
			isDeleteDialogOpen.value = false;
			emit('refresh');
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '删除失败');
	}
}
</script>

<template>
	<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
		<!-- Header -->
		<div class="flex items-center justify-between border-b border-border px-5 py-4">
			<h3 class="font-semibold text-foreground">Webhook 配置</h3>
			<button
				class="h-8 px-3 text-sm font-medium bg-primary text-primary-foreground rounded-md hover:bg-primary/90 flex items-center gap-1.5 transition-colors"
				@click="openCreateModal"
			>
				<Plus class="size-4" />
				添加 Webhook
			</button>
		</div>

		<!-- Webhook List -->
		<div class="overflow-x-auto">
			<table class="app-table-detail min-w-[960px]">
				<thead>
					<tr>
						<th>名称</th>
						<th>模板</th>
						<th>分支过滤</th>
						<th>Webhook URL</th>
						<th>状态</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					<tr v-if="webhooks.length === 0">
						<td colspan="6" class="text-center text-muted-foreground">
							暂无 Webhook 配置
						</td>
					</tr>
					<tr
						v-for="wh in webhooks"
						:key="wh.id"
					>
						<td class="text-foreground">{{ wh.name }}</td>
						<td>
							<router-link
								:to="`/ci/template/${wh.template_id}`"
								class="text-primary hover:underline"
							>
								{{ getTemplateName(wh.template_id) }}
							</router-link>
						</td>
						<td>
							<span v-if="wh.branch_filter" class="text-foreground">{{ wh.branch_filter }}</span>
							<span v-else class="text-destructive">拒绝所有分支</span>
						</td>
						<td>
							<div class="flex min-w-0 items-center gap-2">
								<span class="max-w-72 truncate rounded bg-muted px-2 py-0.5 text-xs text-foreground">
									{{ webhookUrl(wh.id) }}
								</span>
								<button
									class="rounded p-1 transition-colors hover:bg-muted"
									title="复制 URL"
									@click="copyUrl(wh.id)"
								>
									<Copy class="size-3.5 text-muted-foreground" />
								</button>
							</div>
						</td>
						<td>
							<span
								class="inline-block rounded border px-2 py-0.5 text-xs"
								:class="wh.enabled ? 'border-green-200 bg-green-50 text-green-700' : 'border-border bg-muted text-muted-foreground'"
							>
								{{ wh.enabled ? '启用' : '停用' }}
							</span>
						</td>
						<td>
							<div class="flex items-center gap-3">
								<button
									class="text-primary hover:underline"
									@click="openEditModal(wh)"
								>
									编辑
								</button>
								<button
									class="text-destructive hover:underline"
									@click="handleDelete(wh)"
								>
									删除
								</button>
							</div>
						</td>
					</tr>
				</tbody>
			</table>
		</div>

		<DialogRoot v-model:open="isDialogOpen">
			<DialogPortal>
				<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
				<DialogContent
					class="fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[min(520px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
				>
					<DialogTitle class="text-lg font-semibold text-foreground">
						{{ editingWebhook ? '编辑 Webhook' : '添加 Webhook' }}
					</DialogTitle>
					<DialogDescription class="sr-only">配置仓库 Webhook</DialogDescription>
					<div class="mt-5 flex flex-col gap-3">
						<div class="flex flex-col gap-1.5">
							<label class="text-sm font-medium text-foreground">名称</label>
							<input
								v-model="form.name"
								type="text"
								class="rounded-md border border-input bg-background px-3 py-2 text-sm outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
								placeholder="例如: main-branch-webhook"
							/>
						</div>
						<div class="flex flex-col gap-1.5">
							<label class="text-sm font-medium text-foreground">流水线模板</label>
							<ComboboxSelect
								v-model="form.template_id"
								:options="templateOptions"
								placeholder="请选择模板"
							/>
						</div>
						<div class="flex flex-col gap-1.5">
							<label class="text-sm font-medium text-foreground">
								分支过滤
								<span class="font-normal text-muted-foreground">（可选，支持正则）</span>
							</label>
							<input
								v-model="form.branch_filter"
								type="text"
								class="rounded-md border border-input bg-background px-3 py-2 text-sm outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
								placeholder="例如: ^main$"
							/>
						</div>
						<div class="flex flex-col gap-1.5">
							<label class="text-sm font-medium text-foreground">
								签名密钥
								<span v-if="editingWebhook" class="font-normal text-muted-foreground">（留空则不修改）</span>
							</label>
							<input
								v-model="form.secret"
								type="password"
								class="rounded-md border border-input bg-background px-3 py-2 text-sm outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
								placeholder="用于验证 Webhook 请求"
							/>
						</div>
						<label v-if="editingWebhook" class="flex cursor-pointer items-center gap-2">
							<input
								v-model="form.enabled"
								type="checkbox"
								class="size-4 rounded border-input text-primary focus:ring-2 focus:ring-ring/20"
							/>
							<span class="text-sm text-foreground">启用</span>
						</label>
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
							class="flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
							:disabled="operating"
							@click="handleOk"
						>
							<span v-if="operating" class="inline-block size-3.5 animate-spin rounded-full border-2 border-primary-foreground/30 border-t-primary-foreground" />
							保存
						</button>
					</div>
				</DialogContent>
			</DialogPortal>
		</DialogRoot>

		<DialogRoot v-model:open="isDeleteDialogOpen">
			<DialogPortal>
				<DialogOverlay class="fixed inset-0 z-50 bg-black/50 data-[state=open]:animate-overlayShow" />
				<DialogContent
					class="fixed left-1/2 top-1/2 z-50 w-[min(420px,calc(100vw-32px))] -translate-x-1/2 -translate-y-1/2 rounded-lg border border-border bg-card p-6 shadow-xl outline-none data-[state=open]:animate-contentShow"
				>
					<DialogTitle class="text-lg font-semibold text-foreground">删除 Webhook</DialogTitle>
					<DialogDescription class="mt-2 text-sm text-muted-foreground">
						确定删除此 Webhook？此操作不可撤销。
					</DialogDescription>
					<div class="mt-6 flex justify-end gap-2">
						<DialogClose as-child>
							<button
								class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
							>
								取消
							</button>
						</DialogClose>
						<button
							class="flex items-center gap-1.5 rounded-md bg-destructive px-4 py-2 text-sm font-medium text-destructive-foreground transition-colors hover:bg-destructive/90 disabled:cursor-not-allowed disabled:opacity-50"
							:disabled="operating"
							@click="confirmDelete"
						>
							<span v-if="operating" class="inline-block size-3.5 animate-spin rounded-full border-2 border-destructive-foreground/30 border-t-destructive-foreground" />
							删除
						</button>
					</div>
				</DialogContent>
			</DialogPortal>
		</DialogRoot>
	</div>
</template>
