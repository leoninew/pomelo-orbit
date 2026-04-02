<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold">{{ routeData?.domain ?? '路由详情' }}</h1>
			<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/cd/routes')">
				<ArrowLeft class="size-4" />
				返回
			</button>
		</div>

		<!-- Basic info -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<div class="flex items-center justify-between mb-4">
					<h2 class="font-semibold">基本信息</h2>
					<div v-if="routeData" class="flex items-center gap-2">
						<button class="btn btn-sm btn-ghost" :disabled="operating" @click="openEditModal">
							编辑
						</button>
						<button
							v-if="routeData.enabled"
							class="btn btn-sm btn-warning btn-ghost"
							:disabled="operating"
							@click="handleDisable"
						>
							停用
						</button>
						<button
							v-else
							class="btn btn-sm btn-success btn-ghost"
							:disabled="operating"
							@click="handleEnable"
						>
							启用
						</button>
						<button
							class="btn btn-sm btn-error btn-ghost"
							:disabled="routeData.enabled"
							@click="deleteModalRef?.showModal()"
						>
							删除
						</button>
					</div>
				</div>
				<div v-if="basicInfoLoading" class="flex justify-center py-6">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<dl v-else-if="routeData" class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">路由名称</dt>
						<dd>{{ routeData.name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">域名</dt>
						<dd>
							<a
								:href="`${routeData.https_enabled ? 'https' : 'http'}://${routeData.domain}`"
								target="_blank"
								class="link link-primary flex items-center gap-1"
							>
								{{ routeData.domain }}
								<ExternalLink class="size-3" />
							</a>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">路径前缀</dt>
						<dd>{{ routeData.path_prefix }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">目标地址</dt>
						<dd class="text-xs font-mono">{{ routeData.target_url }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">状态</dt>
						<dd>
							<span
								class="badge badge-sm"
								:class="routeData.enabled ? 'badge-outline badge-success' : 'badge-ghost'"
							>
								{{ routeData.enabled ? '启用' : '停用' }}
							</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">协议</dt>
						<dd>
							<span
								class="badge badge-sm"
								:class="routeData.https_enabled ? 'badge-outline badge-info' : 'badge-ghost'"
							>
								{{ routeData.https_enabled ? 'HTTPS' : 'HTTP' }}
							</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">创建时间</dt>
						<dd>{{ formatTime(routeData.created_at) }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">更新时间</dt>
						<dd>{{ formatTime(routeData.updated_at) }}</dd>
					</div>
				</dl>
			</div>
		</div>

		<!-- SSL card -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<h2 class="font-semibold mb-4">SSL 证书</h2>
				<template v-if="routeData">
					<div v-if="!routeData.https_enabled" class="flex flex-col gap-3">
						<div role="alert" class="alert alert-info text-sm">当前使用 HTTP，未启用 HTTPS</div>
						<div class="flex items-center gap-2 flex-wrap">
							<label class="btn btn-sm btn-ghost gap-1.5 cursor-pointer">
								<Upload class="size-4" />
								上传证书 (PEM)
								<input
									type="file"
									accept=".pem,.crt,.cer"
									class="hidden"
									@change="handleCertUpload"
								/>
							</label>
							<button
								class="btn btn-sm btn-primary"
								:disabled="!canUseLetsencrypt || operating"
								:title="!canUseLetsencrypt ? `Let's Encrypt 不支持内网域名或IP` : ''"
								@click="handleEnableLetsencrypt"
							>
								使用 Let's Encrypt
							</button>
							<button
								class="btn btn-sm btn-ghost"
								:disabled="operating"
								@click="handleEnableMkcert"
							>
								使用 mkcert
							</button>
						</div>
					</div>
					<div v-else class="flex flex-col gap-3">
						<div role="alert" class="alert alert-success text-sm">
							{{
								routeData.cert_type === 'letsencrypt'
									? "使用 Let's Encrypt 自动证书"
									: routeData.cert_type === 'mkcert'
										? '使用 mkcert 本地证书'
										: 'HTTPS 已启用（手动证书）'
							}}
						</div>
						<button
							class="btn btn-sm btn-error btn-ghost w-fit"
							:disabled="operating"
							@click="handleDisableHttps"
						>
							{{ routeData.cert_type === 'manual' ? '删除证书并禁用 HTTPS' : '禁用 HTTPS' }}
						</button>
					</div>
				</template>
			</div>
		</div>

		<!-- Edit modal -->
		<dialog ref="editModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">编辑路由</h3>
				<div class="flex flex-col gap-3">
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">路由名称</span></div>
						<input
							:value="form.name"
							type="text"
							class="input input-bordered input-sm opacity-60"
							disabled
						/>
					</label>
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">域名</span></div>
						<input
							v-model="form.domain"
							type="text"
							class="input input-bordered input-sm"
							:class="{ 'input-error': errors.domain }"
						/>
						<div v-if="errors.domain" class="label pt-1">
							<span class="label-text-alt text-error">{{ errors.domain }}</span>
						</div>
					</label>
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">路径前缀</span></div>
						<input v-model="form.path_prefix" type="text" class="input input-bordered input-sm" />
					</label>
					<label class="form-control w-full">
						<div class="label pb-1"><span class="label-text">目标地址</span></div>
						<input
							v-model="form.target_url"
							type="text"
							class="input input-bordered input-sm"
							:class="{ 'input-error': errors.target_url }"
						/>
						<div v-if="errors.target_url" class="label pt-1">
							<span class="label-text-alt text-error">{{ errors.target_url }}</span>
						</div>
					</label>
					<label class="flex items-center gap-2 cursor-pointer">
						<span class="label-text text-sm">启用</span>
						<input v-model="form.enabled" type="checkbox" class="toggle toggle-sm toggle-primary" />
					</label>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleSave">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						保存
					</button>
					<button class="btn btn-ghost" @click="editModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>

		<!-- Delete modal -->
		<dialog ref="deleteModalRef" class="modal">
			<div class="modal-box">
				<h3 class="font-bold text-lg">删除路由</h3>
				<p class="py-4 text-sm">
					确定删除路由
					<strong>{{ routeData?.domain }}</strong>
					吗？此操作不可恢复。
				</p>
				<div class="modal-action">
					<button class="btn btn-error" :disabled="operating" @click="handleDelete">
						<span v-if="operating" class="loading loading-spinner loading-xs" />
						删除
					</button>
					<button class="btn btn-ghost" @click="deleteModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ArrowLeft, ExternalLink, Upload } from 'lucide-vue-next';
import { routeApi } from '@/api/route';
import type { Route } from '@/api/route';
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
const editModalRef = ref<HTMLDialogElement>();
const deleteModalRef = ref<HTMLDialogElement>();

const form = reactive({ name: '', domain: '', path_prefix: '/', target_url: '', enabled: false });
const errors = reactive({ domain: '', target_url: '' });

const canUseLetsencrypt = computed(() => {
	if (!routeData.value) return false;
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
	editModalRef.value?.showModal();
}

async function handleSave() {
	errors.domain = form.domain.trim() ? '' : '请输入域名';
	errors.target_url = /^https?:\/\/[a-zA-Z0-9.-]+:\d+$/.test(form.target_url)
		? ''
		: '格式应为 http://host:port';
	if (errors.domain || errors.target_url) return;
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
			editModalRef.value?.close();
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
	if (!file) return;
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
