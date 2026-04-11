<template>
	<div
		v-if="stage"
		class="stage-node"
		:class="[statusClass, { clickable: !!status }]"
		@click="emit('click')"
	>
		<div class="stage-node-content">
			<span class="stage-name">{{ stage.name }}</span>
		</div>
		<span v-if="status" class="stage-status">{{ statusText }}</span>
	</div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { TaskStatus } from '@/types/api';
import type { SnapshotStage } from '@/types/ci/snapshot';

interface Props {
	data: {
		stage: SnapshotStage
		status?: TaskStatus
		readonly?: boolean
		selected?: boolean
	}
}

const props = withDefaults(defineProps<Props>(), {
	data: () => ({
		readonly: false,
		selected: false,
	}),
});

const emit = defineEmits(['click']);

const stage = computed(() => props.data.stage);
const status = computed(() => props.data.status);

const STATUS_LABEL: Record<string, string> = {
	waiting_to_run: '待运行',
	running: '运行中',
	ran_to_completion: '成功',
	faulted: '异常',
	canceled: '已取消',
};

const statusText = computed(() =>
	status.value ? (STATUS_LABEL[status.value] ?? status.value) : ''
);

const statusClass = computed(() => {
	if (!status.value) {
		return '';
	}
	const map: Record<string, string> = {
		ran_to_completion: 'status-success',
		faulted: 'status-error',
		running: 'status-running',
		waiting_to_run: 'status-waiting',
		canceled: 'status-canceled',
	};
	return map[status.value] ?? '';
});
</script>

<style>
.stage-node {
	background: hsl(var(--b2));
	border: 2px solid hsl(var(--b3));
	border-radius: 0.5rem;
	min-width: 140px;
	min-height: 60px;
	padding: 0.75rem;
	display: flex;
	flex-direction: column;
	align-items: center;
	justify-content: center;
	gap: 4px;
	cursor: pointer;
	transition: filter 0.15s;
}

.stage-node:hover {
	filter: brightness(0.95);
}

.stage-name {
	font-weight: 500;
	font-size: 0.75rem;
	color: hsl(var(--bc));
	text-align: center;
	word-break: break-word;
}

.stage-status {
	font-size: 0.65rem;
	font-weight: 500;
	color: hsl(var(--bc) / 0.6);
}

.status-waiting {
	background: hsl(var(--wa) / 0.12);
	border-color: hsl(var(--wa));
}

.status-running {
	background: hsl(var(--in) / 0.12);
	border-color: hsl(var(--in));
	animation: pulse-border 1.8s ease-in-out infinite;
}

.status-success {
	background: hsl(var(--su) / 0.12);
	border-color: hsl(var(--su));
}

.status-error {
	background: hsl(var(--er) / 0.12);
	border-color: hsl(var(--er));
}

.status-canceled {
	background: hsl(var(--b2));
	border-color: hsl(var(--bc) / 0.25);
	color: hsl(var(--bc) / 0.45);
}

@keyframes pulse-border {
	0%,
	100% {
		border-color: hsl(var(--in));
	}
	50% {
		border-color: hsl(var(--in) / 0.4);
	}
}
</style>
