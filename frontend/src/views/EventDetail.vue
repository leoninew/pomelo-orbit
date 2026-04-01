<template>
	<a-space direction="vertical" style="width: 100%">
		<div v-if="event" class="page-header">
			<h2>事件 #{{ eventId }}</h2>
			<a-button @click="() => $router.push('/events')">返回</a-button>
		</div>

		<a-card title="基本信息" :loading="loading">
			<a-descriptions v-if="event" :column="2" bordered size="small">
				<a-descriptions-item label="来源">
					<a-tag :color="event.source === 'github' ? 'blue' : 'orange'">
						{{ event.source }}
					</a-tag>
				</a-descriptions-item>
				<a-descriptions-item label="状态">
					<a-tag :color="getStatusColor(event.status)">{{ event.status }}</a-tag>
				</a-descriptions-item>
				<a-descriptions-item label="仓库">
					<span v-if="event.repository_name">{{ event.repository_name }}</span>
					<span v-else>-</span>
				</a-descriptions-item>
				<a-descriptions-item label="分支">
					<span v-if="event.branch">{{ event.branch }}</span>
					<span v-else>-</span>
				</a-descriptions-item>
				<a-descriptions-item label="触发者">
					<span v-if="event.sender">{{ event.sender }}</span>
					<span v-else>-</span>
				</a-descriptions-item>
				<a-descriptions-item label="签名验证">
					<a-tag v-if="event.signature_valid === null" color="default">未验证</a-tag>
					<a-tag v-else-if="event.signature_valid" color="success">有效</a-tag>
					<a-tag v-else color="error">无效</a-tag>
				</a-descriptions-item>
				<a-descriptions-item label="接收时间">
					{{ formatTime(event.received_at) }}
				</a-descriptions-item>
				<a-descriptions-item label="处理时间">
					{{ formatTime(event.processed_at) }}
				</a-descriptions-item>
				<a-descriptions-item label="匹配应用">
					<router-link
						v-if="event.matched_application_id"
						:to="`/applications/${event.matched_application_id}`"
					>
						Application {{ event.matched_application_id }}
					</router-link>
					<span v-else>-</span>
				</a-descriptions-item>
				<a-descriptions-item label="触发部署">
					<router-link
						v-if="event.triggered_deploy_record_id"
						:to="`/deploys/${event.triggered_deploy_record_id}`"
					>
						Deploy {{ event.triggered_deploy_record_id }}
					</router-link>
					<span v-else>-</span>
				</a-descriptions-item>
			</a-descriptions>
			<div v-if="event?.error_message" style="margin-top: 12px">
				<a-alert type="error" :message="event.error_message" show-icon />
			</div>
		</a-card>

		<a-card title="Payload" :loading="loading">
			<div v-if="event" class="payload-container">
				<pre class="payload-content">{{ formattedPayload }}</pre>
			</div>
		</a-card>
	</a-space>
</template>

<script setup lang="ts">
import { message } from 'ant-design-vue';
import { formatTime } from '@/utils/time';
import { computed, onMounted, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { eventApi } from '@/api/event';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { eventStatusColors, type WebhookEventDetail } from '@/types/api';

const route = useRoute();
const router = useRouter();
const eventId = route.params.id as string;

const { loading, execute } = useStatusAsync();
const event = ref<WebhookEventDetail>();

const formattedPayload = computed(() => {
	if (!event.value) return '无数据';
	if (!event.value.payload) return '无数据';
	try {
		return JSON.stringify(JSON.parse(event.value.payload), null, 2);
	} catch {
		return event.value.payload;
	}
});

function getStatusColor(status: string) {
	return eventStatusColors[status] || 'default';
}

async function fetchEvent() {
	try {
		await execute(async () => {
			const data = await eventApi.get(eventId);
			event.value = data;
		});
	} catch (error) {
		message.error('获取事件详情失败\n' + error);
		router.push('/events');
	}
}

onMounted(() => {
	fetchEvent();
});
</script>

<style scoped>
.page-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	min-height: 32px;
}

.page-header h2 {
	margin: 0;
}

.sub-title {
	color: rgba(0, 0, 0, 0.45);
	font-size: 14px;
	margin-top: 4px;
}

.payload-container {
	background: #f5f5f5;
	border-radius: 4px;
	padding: 12px;
	max-height: 500px;
	overflow: auto;
}

.payload-content {
	font-family: 'Consolas', 'Monaco', monospace;
	font-size: 13px;
	line-height: 1.5;
	margin: 0;
	white-space: pre-wrap;
	word-break: break-all;
}
</style>
