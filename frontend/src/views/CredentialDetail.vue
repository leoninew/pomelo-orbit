<template>
	<a-space direction="vertical" style="width: 100%">
		<div v-if="credential" class="page-header">
			<h2>{{ credential.name }}</h2>
			<a-button @click="() => $router.push('/credentials')">返回</a-button>
		</div>

		<a-card title="基本信息" :loading="loading">
			<a-descriptions v-if="credential" :column="2" bordered size="small">
				<a-descriptions-item label="凭据名称">{{ credential.name }}</a-descriptions-item>
				<a-descriptions-item label="凭据类型">
					<a-tag :color="getTypeColor(credential.type)">{{ credential.type }}</a-tag>
				</a-descriptions-item>
				<a-descriptions-item label="所属应用">
					<a @click="$router.push(`/applications/${credential.application_id}`)">查看应用</a>
				</a-descriptions-item>
				<a-descriptions-item label="创建时间">
					{{ formatTime(credential.created_at) }}
				</a-descriptions-item>
				<a-descriptions-item label="凭据值" :span="2">
					<pre
						v-if="credential.value"
						style="
							margin: 0;
							white-space: pre-wrap;
							word-break: break-all;
							cursor: pointer;
							user-select: all;
						"
						:title="'点击复制'"
						@click="copyValue"
						>{{ credential.value }}</pre
					>
					<span v-else>-</span>
				</a-descriptions-item>
				<a-descriptions-item v-if="credential.extra_data" label="附加信息" :span="2">
					<pre style="margin: 0; white-space: pre-wrap; word-break: break-all">{{
						credential.extra_data
					}}</pre>
				</a-descriptions-item>
			</a-descriptions>
		</a-card>
	</a-space>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue';
import { formatTime } from '@/utils/time';
import { onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { credentialApi } from '@/api/credential';
import { useStatusAsync } from '@/composables/useStatusAsync';
import type { Credential } from '@/types/api';

const route = useRoute();
const router = useRouter();
const credentialId = route.params.id as string;

const { loading, execute } = useStatusAsync();
const credential = ref<Credential>();

function getTypeColor(type: string) {
	const colors: Record<string, string> = {
		github_token: 'blue',
		gitlab_token: 'orange',
		docker_registry: 'cyan',
		ssh_key: 'purple',
	};
	return colors[type] || 'default';
}

async function copyValue() {
	if (!credential.value) return;
	if (!credential.value.value) return;
	try {
		await navigator.clipboard.writeText(credential.value.value);
		message.success('已复制到剪贴板');
	} catch (error) {
		message.error('复制失败\n' + error);
	}
}

async function fetchCredential() {
	try {
		await execute(async () => {
			const data = await credentialApi.get(credentialId);
			credential.value = data;
		});
	} catch (error) {
		message.error('获取凭据信息失败\n' + error);
		router.push('/credentials');
	}
}

onMounted(() => {
	fetchCredential();
});
</script>

<style scoped>
.page-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.page-header h2 {
	margin: 0;
}

.sub-title {
	color: rgba(0, 0, 0, 0.45);
	font-size: 14px;
	margin-top: 4px;
}
</style>
