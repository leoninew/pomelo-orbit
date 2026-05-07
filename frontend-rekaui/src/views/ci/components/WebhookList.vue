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
		<div v-if="webhooks.length === 0" class="px-5 py-10 text-center">
			<p class="text-sm text-muted-foreground">暂无 Webhook 配置</p>
		</div>
		<div v-else class="divide-y divide-border">
			<div
				v-for="wh in webhooks"
				:key="wh.id"
				class="px-5 py-4 transition-colors hover:bg-muted/30"
			>
				<div class="flex items-start justify-between gap-4">
					<div class="flex-1 min-w-0">
						<div class="flex items-center gap-2 mb-2">
							<h4 class="font-medium text-foreground">{{ wh.name }}</h4>
							<span
								class="inline-block px-2 py-0.5 text-xs rounded border"
								:class="wh.enabled ? 'bg-green-50 text-green-700 border-green-200' : 'bg-muted text-muted-foreground border-border'"
							>
								{{ wh.enabled ? '启用' : '停用' }}
							</span>
						</div>
						<div class="space-y-1 text-sm">
							<div class="flex items-center gap-2">
								<span class="text-muted-foreground">模板:</span>
								<span class="text-foreground">{{ getTemplateName(wh.template_id) }}</span>
							</div>
							<div v-if="wh.branch_filter" class="flex items-center gap-2">
								<span class="text-muted-foreground">分支过滤:</span>
								<span class="text-sm text-foreground">{{ wh.branch_filter }}</span>
							</div>
							<div class="flex items-center gap-2">
								<span class="text-muted-foreground">URL:</span>
								<span class="truncate rounded bg-muted/50 px-2 py-0.5 text-xs text-foreground">
									{{ webhookUrl(wh.id) }}
								</span>
								<button
									class="p-1 hover:bg-muted rounded transition-colors"
									@click="copyUrl(wh.id)"
								>
									<Copy class="size-3.5 text-muted-foreground" />
								</button>
							</div>
						</div>
					</div>
					<div class="flex items-center gap-2">
						<button
							class="px-3 py-1.5 text-sm text-foreground bg-background border border-input rounded-md hover:bg-muted/50 transition-colors"
							@click="openEditModal(wh)"
						>
							编辑
						</button>
						<button
							class="px-3 py-1.5 text-sm text-destructive bg-background border border-destructive/50 rounded-md hover:bg-destructive/10 transition-colors"
							@click="handleDelete(wh)"
						>
							删除
						</button>
					</div>
				</div>
			</div>
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
