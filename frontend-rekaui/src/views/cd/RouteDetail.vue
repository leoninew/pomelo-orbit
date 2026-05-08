

<script setup lang="ts">
import { ExternalLink } from 'lucide-vue-next';
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import type { Route } from '@/api/cd/route';
import { routeApi } from '@/api/cd/route';
import AppDialog from '@/components/AppDialog.vue';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { formatTime } from '@/utils/time';

const currentRoute = useRoute();
const router = useRouter();
const routeId = currentRoute.params.id as string;
const toast = useToast();

const { loading: basicInfoLoading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const routeData = ref<Route>();
const isEditDialogOpen = ref(false);
const isDeleteDialogOpen = ref(false);

const form = reactive({
	name: '',
	domain: '',
	path_prefix: '/',
	target_url: '',
	enabled: false,
});
const errors = reactive({ domain: '', target_url: '' });

const canUseLetsencrypt = computed(() => {
	if (!routeData.value) {
		return false;
	}
	const d = routeData.value.domain;
	return (
		d !== 'localhost' &&
		!d.endsWith('.localhost') &&
		!d.endsWith('.lvh.me') &&
		!/^\d+\.\d+\.\d+\.\d+$/.test(d)
	);
});

async function fetchRoute() {
	try {
		await execute(async () => {
			const data = await routeApi.get(routeId);
			routeData.value = data;
			Object.assign(form, {
				name: data.name,
				domain: data.domain,
				path_prefix: data.path_prefix,
				target_url: data.target_url,
				enabled: data.enabled,
			});
		});
	} catch {
		toast.error('获取路由详情失败');
		router.push('/cd/routes');
	}
}

function openEditModal() {
	Object.assign(errors, { domain: '', target_url: '' });
	isEditDialogOpen.value = true;
}

async function handleSave() {
	errors.domain = form.domain.trim() ? '' : '请输入域名';
	errors.target_url = /^https?:\/\/[a-zA-Z0-9.-]+:\d+$/.test(form.target_url)
		? ''
		: '格式应为 http://host:port';
	if (errors.domain || errors.target_url) {
		return;
	}
	try {
		await executeOp(async () => {
			const updateData = {
				domain: form.domain,
				path_prefix: form.path_prefix,
				target_url: form.target_url,
				enabled: form.enabled,
			};
			await routeApi.update(routeId, updateData);
			toast.success('更新成功');
			isEditDialogOpen.value = false;
			fetchRoute();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '更新失败');
	}
}

async function handleEnable() {
	try {
		await executeOp(async () => {
			await routeApi.enable(routeId);
			toast.success('路由已启用');
			fetchRoute();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '启用失败');
	}
}

async function handleDisable() {
	try {
		await executeOp(async () => {
			await routeApi.disable(routeId);
			toast.success('路由已停用');
			fetchRoute();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '停用失败');
	}
}

async function handleDelete() {
	try {
		await executeOp(async () => {
			await routeApi.delete(routeId);
			toast.success('删除成功');
			router.push('/cd/routes');
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '删除失败');
	}
}

async function handleCertUpload(event: Event) {
	const file = (event.target as HTMLInputElement).files?.[0];
	if (!file) {
		return;
	}
	try {
		await executeOp(async () => {
			await routeApi.uploadCert(routeId, file);
			toast.success('证书上传成功，HTTPS 已启用');
			fetchRoute();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '证书上传失败');
	}
}

async function handleDisableHttps() {
	try {
		await executeOp(async () => {
			await routeApi.disableHttps(routeId);
			toast.success('HTTPS 已禁用');
			fetchRoute();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '操作失败');
	}
}

async function handleEnableLetsencrypt() {
	try {
		await executeOp(async () => {
			await routeApi.enableLetsencrypt(routeId);
			toast.success("Let's Encrypt 证书已启用");
			fetchRoute();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '操作失败');
	}
}

async function handleEnableMkcert() {
	try {
		await executeOp(async () => {
			await routeApi.enableMkcert(routeId);
			toast.success('mkcert 证书已生成并启用');
			fetchRoute();
		});
	} catch (error) {
		toast.error(error instanceof Error ? error.message : '操作失败');
	}
}

onMounted(fetchRoute);
</script>

<template>
	<div class="flex flex-col gap-4">
		<!-- Header -->
		<div class="flex flex-wrap items-center justify-between gap-3">
			<div class="flex items-center gap-3">
				<h1 class="text-xl font-semibold text-foreground">{{ routeData?.name ?? '路由详情' }}</h1>
			</div>
			<div class="flex flex-wrap items-center gap-2">
				<button
					v-if="routeData && !routeData.enabled"
					class="h-9 rounded-md border border-input bg-background px-3 text-sm font-medium text-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="operating"
					@click="handleEnable"
				>
					启用
				</button>
				<button
					v-else-if="routeData"
					class="h-9 rounded-md border border-input bg-background px-3 text-sm font-medium text-foreground transition-colors hover:bg-muted/50 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="operating"
					@click="handleDisable"
				>
					停用
				</button>
				<button
					v-if="routeData"
					class="h-9 rounded-md border border-input bg-background px-3 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="openEditModal"
				>
					编辑
				</button>
				<button
					v-if="routeData"
					class="h-9 rounded-md border border-destructive/50 bg-background px-3 text-sm font-medium text-destructive transition-colors hover:bg-destructive/10 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="operating"
					@click="isDeleteDialogOpen = true"
				>
					删除
				</button>
				<button
					class="h-9 rounded-md border border-input bg-background px-4 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="router.push('/cd/routes')"
				>
					返回
				</button>
			</div>
		</div>

		<!-- Loading State -->
		<div v-if="basicInfoLoading" class="flex justify-center py-12">
			<span class="inline-block size-8 border-4 border-primary/20 border-t-primary rounded-full animate-spin" />
		</div>

		<!-- Content -->
		<template v-else-if="routeData">
			<!-- Basic Info Card -->
			<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
				<div class="border-b border-border px-5 py-4">
					<h2 class="font-semibold text-foreground">基本信息</h2>
				</div>
				<dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
					<div class="flex gap-2">
						<dt class="text-muted-foreground w-24 shrink-0">名称</dt>
						<dd class="text-foreground">{{ routeData.name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-muted-foreground w-24 shrink-0">域名</dt>
						<dd>
							<a
								:href="`${routeData.https_enabled ? 'https' : 'http'}://${routeData.domain}`"
								target="_blank"
								class="text-primary hover:underline flex items-center gap-1"
							>
								{{ routeData.domain }}
								<ExternalLink class="size-3" />
							</a>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-muted-foreground w-24 shrink-0">路径前缀</dt>
						<dd class="text-foreground">{{ routeData.path_prefix }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-muted-foreground w-24 shrink-0">目标地址</dt>
						<dd class="text-foreground">{{ routeData.target_url }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-muted-foreground w-24 shrink-0">状态</dt>
						<dd>
							<span
								class="inline-block px-2 py-0.5 text-xs rounded border"
								:class="routeData.enabled ? 'bg-green-50 text-green-700 border-green-200' : 'bg-muted text-muted-foreground border-border'"
							>
								{{ routeData.enabled ? '启用' : '停用' }}
							</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-muted-foreground w-24 shrink-0">创建时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(routeData.created_at) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-muted-foreground w-24 shrink-0">更新时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(routeData.updated_at) }}</dd>
					</div>
				</dl>
			</div>

			<!-- HTTPS Config Card -->
			<div class="overflow-hidden rounded-lg border border-border bg-card shadow-sm">
				<div class="border-b border-border px-5 py-4">
					<h2 class="font-semibold text-foreground">HTTPS 配置</h2>
				</div>
				<div class="space-y-4 px-5 py-4">
					<div class="flex items-center justify-between p-3 bg-muted/30 rounded-md">
						<div>
							<p class="text-sm text-foreground">当前状态</p>
							<p class="text-sm text-muted-foreground">
								{{ routeData.https_enabled ? 'HTTPS 已启用' : 'HTTPS 未启用' }}
							</p>
						</div>
						<span
							class="inline-block px-2 py-0.5 text-xs rounded border"
							:class="routeData.https_enabled ? 'bg-blue-50 text-blue-700 border-blue-200' : 'bg-muted text-muted-foreground border-border'"
						>
							{{ routeData.https_enabled ? 'HTTPS' : 'HTTP' }}
						</span>
					</div>

					<div v-if="!routeData.https_enabled" class="space-y-2">
						<button
							v-if="canUseLetsencrypt"
							class="w-full p-3 text-left bg-muted/30 rounded-md border border-border hover:border-primary/50 transition-colors"
							:disabled="operating"
							@click="handleEnableLetsencrypt"
						>
							<p class="text-sm text-foreground">Let's Encrypt 自动证书</p>
							<p class="text-sm text-muted-foreground">自动申请并续期免费 SSL 证书</p>
						</button>
						<button
							class="w-full p-3 text-left bg-muted/30 rounded-md border border-border hover:border-primary/50 transition-colors"
							:disabled="operating"
							@click="handleEnableMkcert"
						>
							<p class="text-sm text-foreground">mkcert 本地证书</p>
							<p class="text-sm text-muted-foreground">生成本地开发用的自签名证书</p>
						</button>
						<label class="block w-full p-3 bg-muted/30 rounded-md border border-border hover:border-primary/50 transition-colors cursor-pointer">
							<p class="text-sm text-foreground">上传自定义证书</p>
							<p class="text-sm text-muted-foreground">上传 .pem 格式的证书文件</p>
							<input
								type="file"
								accept=".pem"
								class="hidden"
								@change="handleCertUpload"
							/>
						</label>
					</div>

					<div v-else class="flex justify-end">
						<button
							class="px-4 py-2 text-sm font-medium text-destructive bg-background border border-destructive/50 rounded-md hover:bg-destructive/10 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
							:disabled="operating"
							@click="handleDisableHttps"
						>
							禁用 HTTPS
						</button>
					</div>
				</div>
			</div>
		</template>

		<AppDialog v-model:open="isEditDialogOpen" title="编辑路由">
			<div class="flex flex-col gap-3">
				<div class="flex flex-col gap-1.5">
					<label class="text-sm font-medium text-foreground">域名</label>
					<input
						v-model="form.domain"
						type="text"
						class="px-3 py-2 text-sm bg-background border rounded-md outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
						:class="errors.domain ? 'border-destructive' : 'border-input'"
						placeholder="example.com"
					/>
					<p v-if="errors.domain" class="text-xs text-destructive">{{ errors.domain }}</p>
				</div>
				<div class="flex flex-col gap-1.5">
					<label class="text-sm font-medium text-foreground">路径前缀</label>
					<input
						v-model="form.path_prefix"
						type="text"
						class="px-3 py-2 text-sm bg-background border border-input rounded-md outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
						placeholder="/"
					/>
				</div>
				<div class="flex flex-col gap-1.5">
					<label class="text-sm font-medium text-foreground">目标地址</label>
					<input
						v-model="form.target_url"
						type="text"
						class="px-3 py-2 text-sm bg-background border rounded-md outline-none transition-colors focus:border-ring focus:ring-2 focus:ring-ring/20"
						:class="errors.target_url ? 'border-destructive' : 'border-input'"
						placeholder="http://host:port"
					/>
					<p v-if="errors.target_url" class="text-xs text-destructive">{{ errors.target_url }}</p>
				</div>
			</div>
			<template #footer>
				<button
					class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="isEditDialogOpen = false"
				>
					取消
				</button>
				<button
					class="flex items-center gap-1.5 rounded-md bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-colors hover:bg-primary/90 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="operating"
					@click="handleSave"
				>
					<span v-if="operating" class="inline-block size-3.5 border-2 border-primary-foreground/30 border-t-primary-foreground rounded-full animate-spin" />
					保存
				</button>
			</template>
		</AppDialog>

		<AppDialog
			v-model:open="isDeleteDialogOpen"
			title="删除路由"
			description="确定删除此路由？此操作不可恢复。"
			width-class="w-[min(420px,calc(100vw-32px))]"
			body-class="hidden"
		>
			<template #footer>
				<button
					class="rounded-md border border-input bg-background px-4 py-2 text-sm font-medium text-foreground transition-colors hover:bg-muted/50"
					@click="isDeleteDialogOpen = false"
				>
					取消
				</button>
				<button
					class="flex items-center gap-1.5 rounded-md bg-destructive px-4 py-2 text-sm font-medium text-destructive-foreground transition-colors hover:bg-destructive/90 disabled:cursor-not-allowed disabled:opacity-50"
					:disabled="operating"
					@click="handleDelete"
				>
					<span v-if="operating" class="inline-block size-3.5 border-2 border-destructive-foreground/30 border-t-destructive-foreground rounded-full animate-spin" />
					删除
				</button>
			</template>
		</AppDialog>
	</div>
</template>
