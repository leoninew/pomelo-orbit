<script setup lang="ts">
import { Download } from 'lucide-vue-next';
import { onMounted, reactive, ref } from 'vue';
import { useRouter } from 'vue-router';
import {
	DialogClose,
	DialogContent,
	DialogDescription,
	DialogOverlay,
	DialogPortal,
	DialogRoot,
	DialogTitle,
} from 'reka-ui';
import { credentialApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { Credential } from '@/types/ci/credential';
import { credentialTypeLabels } from '@/types/ci/credential';
import { formatTime } from '@/utils/time';

const props = defineProps<{ id: string }>();
const $router = useRouter();
const toast = useToast();
const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const credential = ref<Credential>();
const isEditModalOpen = ref(false);
const isDeleteModalOpen = ref(false);
const form = reactive({ name: '', data: '' });
const errors = reactive({ name: '' });

async function fetchCredential() {
	try {
		await execute(async () => {
			credential.value = await credentialApi.get(props.id);
		});
	} catch {
		toast.error('获取凭据详情失败');
	}
}

function openEditModal() {
	Object.assign(form, { name: credential.value?.name ?? '', data: '' });
	Object.assign(errors, { name: '' });
	isEditModalOpen.value = true;
}

async function handleEditOk() {
	errors.name = form.name.trim() ? '' : '请输入凭据名称';
	if (errors.name) {
		return;
	}
	try {
		await executeOp(async () => {
			await credentialApi.update(props.id, {
				name: form.name,
				...(form.data ? { data: form.data } : {}),
			});
			toast.success('更新成功');
			isEditModalOpen.value = false;
			fetchCredential();
		});
	} catch (err) {
		toast.error(err instanceof Error ? err.message : '操作失败');
	}
}

function openDeleteModal() {
	isDeleteModalOpen.value = true;
}

async function handleDelete() {
	try {
		await executeOp(async () => {
			await credentialApi.delete(props.id);
			toast.success('删除成功');
			$router.push('/ci/credential');
		});
	} catch (err) {
		toast.error(err instanceof Error ? err.message : '删除失败');
	}
}

async function handleExport() {
	try {
		const data = await credentialApi.exportCredential(props.id);
		const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `${credential.value?.name ?? 'credential'}.json`;
		a.click();
		URL.revokeObjectURL(url);
	} catch (err) {
		toast.error(err instanceof Error ? err.message : '导出失败');
	}
}

function getDataPlaceholder(type: string) {
	if (type === 'git_ssh') {
		return '-----BEGIN OPENSSH PRIVATE KEY-----\n...';
	}
	if (type === 'git_token') {
		return 'ghp_xxxxxxxxxxxxxxxxxxxx';
	}
	if (type === 'gitee_token') {
		return 'your_username:your_gitee_token';
	}
	return 'registry_token_here';
}

onMounted(fetchCredential);
</script>

<template>
	<div class="flex flex-col gap-4">
		<div class="flex flex-wrap items-center justify-between gap-3">
			<h1 class="text-xl font-semibold text-foreground">{{ credential?.name ?? '凭据详情' }}</h1>
			<div class="flex flex-wrap items-center gap-2">
				<button
					v-if="credential"
					class="h-9 rounded-md border border-input bg-background px-3 text-sm font-medium text-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="operating"
					@click="openEditModal"
				>
					编辑
				</button>
				<button
					v-if="credential"
					class="inline-flex h-9 items-center gap-2 rounded-md border border-input bg-background px-3 text-sm font-medium text-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="operating"
					@click="handleExport"
				>
					<Download class="size-4" />
					导出
				</button>
				<button
					v-if="credential"
					class="h-9 rounded-md border border-destructive/50 bg-background px-3 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="operating"
					@click="openDeleteModal"
				>
					删除
				</button>
				<button
					class="h-9 rounded-md border border-input bg-background px-4 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="$router.push('/ci/credential')"
				>
					返回
				</button>
			</div>
		</div>

		<div class="rounded-lg border border-border bg-card shadow-sm">
			<div class="border-b border-border px-5 py-4">
				<h2 class="font-semibold text-foreground">基本信息</h2>
			</div>

			<div v-if="loading" class="flex justify-center px-5 py-10">
				<span class="inline-block size-8 border-4 border-primary/20 border-t-primary rounded-full animate-spin" />
			</div>
			<dl v-else-if="credential" class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
				<div class="flex gap-2">
					<dt class="text-muted-foreground w-24 shrink-0">凭据名称</dt>
					<dd class="text-foreground">{{ credential.name }}</dd>
				</div>
				<div class="flex gap-2">
					<dt class="text-muted-foreground w-24 shrink-0">类型</dt>
					<dd>
						<span class="inline-block px-2 py-0.5 text-xs bg-muted text-muted-foreground rounded">
							{{ credentialTypeLabels[credential.type] ?? credential.type }}
						</span>
					</dd>
				</div>
				<div class="flex gap-2">
					<dt class="text-muted-foreground w-24 shrink-0">创建时间</dt>
					<dd class="text-muted-foreground">{{ formatTime(credential.created_at) }}</dd>
				</div>
			</dl>
		</div>

		<!-- Edit modal -->
		<DialogRoot v-model:open="isEditModalOpen">
			<DialogPortal>
				<DialogOverlay class="fixed inset-0 bg-black/50 z-50 data-[state=open]:animate-overlayShow" />
				<DialogContent class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 bg-card border border-border rounded-lg shadow-xl w-full max-w-lg max-h-[90vh] overflow-y-auto z-50 p-6 data-[state=open]:animate-contentShow">
					<DialogTitle class="mb-4 text-lg font-semibold text-foreground">编辑凭据</DialogTitle>
					<DialogDescription class="sr-only">编辑凭据信息</DialogDescription>
					
					<div class="flex flex-col gap-3">
						<div class="flex flex-col gap-1.5">
							<label class="text-sm font-medium text-foreground">凭据名称</label>
							<input
								v-model="form.name"
								type="text"
								class="px-3 py-2 text-sm bg-background border rounded-md outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
								:class="errors.name ? 'border-destructive' : 'border-input'"
							/>
							<p v-if="errors.name" class="text-xs text-destructive">{{ errors.name }}</p>
						</div>
						<div class="flex flex-col gap-1.5">
							<label class="text-sm font-medium text-foreground">
								凭据内容
								<span class="text-muted-foreground font-normal">（留空则不修改）</span>
							</label>
							<textarea
								v-model="form.data"
								class="px-3 py-2 text-xs font-mono bg-background border border-input rounded-md outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
								rows="8"
								:placeholder="credential ? getDataPlaceholder(credential.type) : ''"
							/>
						</div>
					</div>
					
					<div class="flex justify-end gap-2 mt-6">
						<DialogClose as-child>
							<button class="px-4 py-2 text-sm font-medium text-foreground bg-background border border-input rounded-md hover:bg-muted/50 transition-colors">
								取消
							</button>
						</DialogClose>
						<button
							class="px-4 py-2 text-sm font-medium bg-primary text-primary-foreground rounded-md hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5 transition-colors"
							:disabled="operating"
							@click="handleEditOk"
						>
							<span v-if="operating" class="inline-block size-3.5 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin" />
							保存
						</button>
					</div>
				</DialogContent>
			</DialogPortal>
		</DialogRoot>

		<!-- Delete confirm modal -->
		<DialogRoot v-model:open="isDeleteModalOpen">
			<DialogPortal>
				<DialogOverlay class="fixed inset-0 bg-black/50 z-50 data-[state=open]:animate-overlayShow" />
				<DialogContent class="fixed left-1/2 top-1/2 -translate-x-1/2 -translate-y-1/2 bg-card border border-border rounded-lg shadow-xl w-full max-w-md z-50 p-6 data-[state=open]:animate-contentShow">
					<DialogTitle class="text-lg font-semibold text-foreground">删除凭据</DialogTitle>
					<DialogDescription class="py-4 text-sm text-muted-foreground">确定删除此凭据？</DialogDescription>
					
					<div class="flex justify-end gap-2">
						<DialogClose as-child>
							<button class="px-4 py-2 text-sm font-medium text-foreground bg-background border border-input rounded-md hover:bg-muted/50 transition-colors">
								取消
							</button>
						</DialogClose>
						<button
							class="px-4 py-2 text-sm font-medium bg-destructive text-destructive-foreground rounded-md hover:bg-destructive/90 disabled:opacity-50 disabled:cursor-not-allowed flex items-center gap-1.5 transition-colors"
							:disabled="operating"
							@click="handleDelete"
						>
							<span v-if="operating" class="inline-block size-3.5 border-2 border-destructive-foreground/30 border-t-destructive-foreground rounded-full animate-spin" />
							删除
						</button>
					</div>
				</DialogContent>
			</DialogPortal>
		</DialogRoot>
	</div>
</template>
