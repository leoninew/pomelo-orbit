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
					class="app-button h-9 px-3"
					:disabled="operating"
					@click="handleEnable"
				>
					启用
				</button>
				<button
					v-else-if="routeData"
					class="app-button h-9 px-3"
					:disabled="operating"
					@click="handleDisable"
				>
					停用
				</button>
				<button v-if="routeData" class="app-button h-9 px-3" @click="openEditModal">编辑</button>
				<button
					v-if="routeData"
					class="app-button-danger h-9 px-3"
					:disabled="operating"
					@click="isDeleteDialogOpen = true"
				>
					删除
				</button>
				<button class="app-button h-9 px-4" @click="router.push('/cd/routes')">返回</button>
			</div>
		</div>

		<!-- Loading State -->
		<AppSpinner v-if="basicInfoLoading" class="py-12" />

		<!-- Content -->
		<template v-else-if="routeData">
			<!-- Basic Info Card -->
			<div class="app-surface">
				<div class="app-section-header">
					<h2 class="font-semibold text-foreground">基本信息</h2>
				</div>
				<dl class="grid grid-cols-1 gap-x-8 gap-y-3 px-5 py-4 text-sm sm:grid-cols-2">
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">名称</dt>
						<dd class="text-foreground">{{ routeData.name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">域名</dt>
						<dd>
							<a
								:href="`${routeData.https_enabled ? 'https' : 'http'}://${routeData.domain}`"
								target="_blank"
								class="app-link inline-flex items-center gap-1"
							>
								{{ routeData.domain }}
								<ExternalLink class="size-3" />
							</a>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">路径前缀</dt>
						<dd class="text-foreground">{{ routeData.path_prefix }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">目标地址</dt>
						<dd class="text-foreground">{{ routeData.target_url }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">状态</dt>
						<dd>
							<span
								class="app-badge-status"
								:class="
									routeData.enabled
										? 'bg-green-50 text-green-700 border-green-200'
										: 'bg-muted text-muted-foreground border-border'
								"
							>
								{{ routeData.enabled ? '启用' : '停用' }}
							</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">创建时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(routeData.created_at) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="w-24 shrink-0 text-muted-foreground">更新时间</dt>
						<dd class="text-muted-foreground">{{ formatTime(routeData.updated_at) }}</dd>
					</div>
				</dl>
			</div>

			<!-- HTTPS Config Card -->
			<div class="app-surface">
				<div class="app-section-header">
					<h2 class="font-semibold text-foreground">HTTPS 配置</h2>
				</div>
				<div class="space-y-4 px-5 py-4">
					<div class="flex items-center justify-between rounded-md bg-muted/30 p-3">
						<div>
							<p class="text-sm text-foreground">当前状态</p>
							<p class="text-sm text-muted-foreground">
								{{ routeData.https_enabled ? 'HTTPS 已启用' : 'HTTPS 未启用' }}
							</p>
						</div>
						<span
							class="app-badge-status"
							:class="
								routeData.https_enabled
									? 'bg-blue-50 text-blue-700 border-blue-200'
									: 'bg-muted text-muted-foreground border-border'
							"
						>
							{{ routeData.https_enabled ? 'HTTPS' : 'HTTP' }}
						</span>
					</div>

					<div v-if="!routeData.https_enabled" class="space-y-2">
						<button
							v-if="canUseLetsencrypt"
							class="app-action-item"
							:disabled="operating"
							@click="handleEnableLetsencrypt"
						>
							<p class="text-sm text-foreground">Let's Encrypt 自动证书</p>
							<p class="text-sm text-muted-foreground">自动申请并续期免费 SSL 证书</p>
						</button>
						<button class="app-action-item" :disabled="operating" @click="handleEnableMkcert">
							<p class="text-sm text-foreground">mkcert 本地证书</p>
							<p class="text-sm text-muted-foreground">生成本地开发用的自签名证书</p>
						</button>
						<label class="app-action-item cursor-pointer">
							<p class="text-sm text-foreground">上传自定义证书</p>
							<p class="text-sm text-muted-foreground">上传 .pem 格式的证书文件</p>
							<input type="file" accept=".pem" class="hidden" @change="handleCertUpload" />
						</label>
					</div>

					<div v-else class="flex justify-end">
						<button class="app-button-danger" :disabled="operating" @click="handleDisableHttps">
							禁用 HTTPS
						</button>
					</div>
				</div>
			</div>
		</template>

		<AppDialog v-model:open="isEditDialogOpen" title="编辑路由">
			<div class="space-y-4">
				<div class="space-y-1.5">
					<label class="app-field-label block">域名</label>
					<input
						v-model="form.domain"
						type="text"
						class="app-input"
						:class="errors.domain ? 'app-input-error' : ''"
						placeholder="example.com"
					/>
					<p v-if="errors.domain" class="app-field-error text-xs">{{ errors.domain }}</p>
				</div>
				<div class="space-y-1.5">
					<label class="app-field-label block">路径前缀</label>
					<input v-model="form.path_prefix" type="text" class="app-input" placeholder="/" />
				</div>
				<div class="space-y-1.5">
					<label class="app-field-label block">目标地址</label>
					<input
						v-model="form.target_url"
						type="text"
						class="app-input"
						:class="errors.target_url ? 'app-input-error' : ''"
						placeholder="http://host:port"
					/>
					<p v-if="errors.target_url" class="app-field-error text-xs">{{ errors.target_url }}</p>
				</div>
			</div>
			<template #footer>
				<button class="app-button" @click="isEditDialogOpen = false">取消</button>
				<button class="app-button-primary" :disabled="operating" @click="handleSave">保存</button>
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
				<button class="app-button" @click="isDeleteDialogOpen = false">取消</button>
				<button class="app-button-destructive" :disabled="operating" @click="handleDelete">
					删除
				</button>
			</template>
		</AppDialog>
	</div>
</template>

<script setup lang="ts">
	import { ExternalLink } from 'lucide-vue-next';
	import { computed, onMounted, reactive, ref } from 'vue';
	import { useRoute, useRouter } from 'vue-router';
	import type { Route } from '@/api/cd/route';
	import { routeApi } from '@/api/cd/route';
	import AppDialog from '@/components/AppDialog.vue';
	import AppSpinner from '@/components/AppSpinner.vue';
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
