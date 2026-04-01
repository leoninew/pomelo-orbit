<template>
	<a-space direction="vertical" style="width: 100%">
		<div class="page-header">
			<h2>{{ route ? route.domain : '加载中...' }}</h2>
			<a-button @click="() => $router.push('/routes')">返回</a-button>
		</div>

		<!-- 基本信息卡片 -->
		<a-card title="基本信息" :loading="basicInfoLoading">
			<template v-if="route" #extra>
				<a-space>
					<a-button :disabled="operating" @click="showEditModal = true">编辑</a-button>
					<a-button v-if="route.enabled" danger :loading="operating" @click="handleDisable">
						停用
					</a-button>
					<a-button v-else type="primary" :loading="operating" @click="handleEnable">启用</a-button>
					<a-button danger :disabled="route.enabled" @click="showDeleteModal = true">删除</a-button>
				</a-space>
			</template>
			<a-descriptions v-if="route" :column="2" bordered size="small">
				<a-descriptions-item label="路由名称">
					{{ route.name }}
				</a-descriptions-item>
				<a-descriptions-item label="域名">
					<a :href="`${route.https_enabled ? 'https' : 'http'}://${route.domain}`" target="_blank">
						{{ route.domain }}
						<LinkOutlined />
					</a>
				</a-descriptions-item>
				<a-descriptions-item label="路径前缀">
					{{ route.path_prefix }}
				</a-descriptions-item>
				<a-descriptions-item label="目标地址">
					{{ route.target_url }}
				</a-descriptions-item>
				<a-descriptions-item label="状态">
					<a-tag :color="route.enabled ? 'success' : 'default'">
						{{ route.enabled ? '启用' : '停用' }}
					</a-tag>
				</a-descriptions-item>
				<a-descriptions-item label="协议">
					<a-tag :color="route.https_enabled ? 'green' : 'default'">
						{{ route.https_enabled ? 'HTTPS' : 'HTTP' }}
					</a-tag>
				</a-descriptions-item>
				<a-descriptions-item label="创建时间">
					{{ formatTime(route.created_at) }}
				</a-descriptions-item>
				<a-descriptions-item label="更新时间">
					{{ formatTime(route.updated_at) }}
				</a-descriptions-item>
			</a-descriptions>
		</a-card>

		<!-- SSL 证书卡片 -->
		<a-card title="SSL 证书" :loading="basicInfoLoading">
			<template v-if="route">
				<div v-if="!route.https_enabled">
					<a-alert message="当前使用 HTTP" type="info" show-icon style="margin-bottom: 16px" />
					<a-space>
						<a-upload
							:multiple="false"
							accept=".pem,.crt,.cer"
							:before-upload="handleCertUpload"
							:show-upload-list="false"
							:disabled="operating"
						>
							<a-button :loading="operating">
								<template #icon><UploadOutlined /></template>
								上传证书 (PEM)
							</a-button>
						</a-upload>
						<a-button
							type="primary"
							:disabled="!canUseLetsencrypt"
							:title="
								!canUseLetsencrypt
									? 'Let\'s Encrypt 不支持 localhost/.lvh.me 等内网域名或IP 地址'
									: ''
							"
							:loading="operating"
							@click="handleEnableLetsencrypt"
						>
							使用 Let's Encrypt
						</a-button>
						<a-button :loading="operating" @click="handleEnableMkcert">使用 mkcert</a-button>
					</a-space>
				</div>
				<div v-else>
					<a-alert
						:message="
							route.cert_type === 'letsencrypt'
								? '使用 Let\'s Encrypt 自动证书'
								: route.cert_type === 'mkcert'
									? '使用 mkcert 本地证书'
									: 'HTTPS 已启用（手动证书）'
						"
						type="success"
						show-icon
						style="margin-bottom: 16px"
					/>
					<a-button type="primary" danger :loading="operating" @click="handleDisableHttps">
						{{ route.cert_type === 'manual' ? '删除证书并禁用 HTTPS' : '禁用 HTTPS' }}
					</a-button>
				</div>
			</template>
		</a-card>

		<!-- 删除确认弹窗 -->
		<a-modal v-model:open="showDeleteModal" title="删除路由">
			<p>
				确定删除路由
				<strong>{{ route?.domain }}</strong>
				吗？
			</p>
			<p style="color: #ff4d4f">此操作不可恢复，请谨慎操作。</p>
			<template #footer>
				<a-button type="primary" danger :loading="operating" @click="handleDelete">删除</a-button>
				<a-button @click="showDeleteModal = false">取消</a-button>
			</template>
		</a-modal>

		<!-- 编辑弹窗 -->
		<a-modal v-model:open="showEditModal" title="编辑路由">
			<a-form
				ref="formRef"
				:model="form"
				:rules="formRules"
				:label-col="{ span: 6 }"
				:wrapper-col="{ span: 16 }"
			>
				<a-form-item label="路由名称" name="name">
					<a-input :value="form.name" disabled />
				</a-form-item>
				<a-form-item label="域名" name="domain">
					<a-input v-model:value="form.domain" />
				</a-form-item>
				<a-form-item label="路径前缀" name="path_prefix">
					<a-input v-model:value="form.path_prefix" />
				</a-form-item>
				<a-form-item label="目标地址" name="target_url">
					<a-input v-model:value="form.target_url" />
				</a-form-item>
				<a-form-item label="状态" name="enabled">
					<a-switch v-model:checked="form.enabled" />
				</a-form-item>
			</a-form>
			<template #footer>
				<a-button type="primary" :loading="operating" @click="handleSave">保存</a-button>
				<a-button @click="showEditModal = false">取消</a-button>
			</template>
		</a-modal>
	</a-space>
</template>

<script setup lang="ts">
import { UploadOutlined } from '@ant-design/icons-vue';
import { ref, onMounted, reactive, computed } from 'vue';
import { useRouter, useRoute } from 'vue-router';
import { message } from 'ant-design-vue';
import type { FormInstance } from 'ant-design-vue';
import { routeApi } from '@/api/route';
import type { Route } from '@/api/route';
import { formatTime } from '@/utils/time';
import { useStatusAsync } from '@/composables/useStatusAsync';

const router = useRouter();
const currentRoute = useRoute();
const routeId = currentRoute.params.id as string;

const { loading: basicInfoLoading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();
const route = ref<Route>();
const showDeleteModal = ref(false);
const showEditModal = ref(false);
const formRef = ref<FormInstance>();

const form = reactive({
	name: '',
	domain: '',
	path_prefix: '/',
	target_url: '',
	enabled: false,
});

const formRules = {
	name: [{ required: true, message: '请输入路由名称' }],
	domain: [{ required: true, message: '请输入域名' }],
	path_prefix: [{ required: true, message: '请输入路径前缀' }],
	target_url: [
		{ required: true, message: '请输入目标地址' },
		{
			pattern: /^https?:\/\/[a-zA-Z0-9.-]+:\d+$/,
			message: '格式应为 http://host:port',
		},
	],
};

// 检测域名是否支持 Let's Encrypt
const canUseLetsencrypt = computed(() => {
	if (!route.value) return false;
	const domain = route.value.domain;
	// 不支持 localhost、内网域名、IP 地址
	if (domain === 'localhost' || domain.endsWith('.localhost') || domain.endsWith('.lvh.me')) {
		return false;
	}
	// 检测 IP 地址
	if (/^\d+\.\d+\.\d+\.\d+$/.test(domain)) {
		return false;
	}
	return true;
});

async function fetchRoute() {
	try {
		await execute(async () => {
			const data = await routeApi.get(routeId);
			route.value = data;
			form.name = data.name;
			form.domain = data.domain;
			form.path_prefix = data.path_prefix;
			form.target_url = data.target_url;
			form.enabled = data.enabled;
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '获取路由详情失败');
		router.push('/routes');
	}
}

async function handleSave() {
	try {
		await formRef.value?.validate();
	} catch {
		return;
	}

	try {
		await executeOp(async () => {
			// eslint-disable-next-line @typescript-eslint/no-unused-vars
			const { name, ...updateData } = form;
			await routeApi.update(routeId, updateData);
			message.success('更新成功');
			showEditModal.value = false;
			await fetchRoute();
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '更新失败');
	}
}

async function handleEnable() {
	try {
		await executeOp(async () => {
			await routeApi.enable(routeId);
			message.success('路由已启用');
			await fetchRoute();
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '启用失败');
	}
}

async function handleDisable() {
	try {
		await executeOp(async () => {
			await routeApi.disable(routeId);
			message.success('路由已停用');
			await fetchRoute();
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '停用失败');
	}
}

async function handleDelete() {
	try {
		await executeOp(async () => {
			await routeApi.delete(routeId);
			message.success('路由删除成功');
			router.push('/routes');
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '删除失败');
	}
}

async function handleCertUpload(file: File) {
	try {
		await executeOp(async () => {
			await routeApi.uploadCert(routeId, file);
			message.success('证书上传成功，HTTPS 已启用');
			await fetchRoute();
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '证书上传失败');
	}
	return false;
}

async function handleDisableHttps() {
	try {
		await executeOp(async () => {
			await routeApi.disableHttps(routeId);
			message.success('HTTPS 已禁用');
			await fetchRoute();
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '操作失败');
	}
}

async function handleEnableLetsencrypt() {
	try {
		await executeOp(async () => {
			await routeApi.enableLetsencrypt(routeId);
			message.success("Let's Encrypt 证书已启用，Traefik 将自动获取证书");
			await fetchRoute();
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '操作失败');
	}
}

async function handleEnableMkcert() {
	try {
		await executeOp(async () => {
			await routeApi.enableMkcert(routeId);
			message.success('mkcert 证书已生成并启用');
			await fetchRoute();
		});
	} catch (error: unknown) {
		message.error(error instanceof Error ? error.message : '操作失败');
	}
}

onMounted(() => {
	fetchRoute();
});
</script>

<style scoped>
.page-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	min-height: 32px;
}
</style>
