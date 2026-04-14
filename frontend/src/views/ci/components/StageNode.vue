<template>
	<div v-if="stage" class="stage-node" :class="{ 'is-running': isRunning }" @click="emit('click')">
		<div class="stage-node-content">
			<span class="stage-name" :style="{ color: textColor }">{{ stage.name }}</span>
		</div>
		<span v-if="status" class="stage-status" :style="{ color: textColor }">{{ statusText }}</span>
	</div>
</template>

<script setup lang="ts">
import { computed } from 'vue';
import type { SnapshotStage } from '@/types/ci/snapshot';
import { statusColor, statusLabel } from '@/utils/status';
import type { TaskStatus } from '@/types/common';

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

const statusText = computed(() => statusLabel(status.value ?? ''));

const textColor = computed(() => statusColor(status.value));
const isRunning = computed(() => status.value === 'running');
</script>

<style scoped>
.stage-node {
	background: #ffffff;
	border: 2px solid #cbd5e0;
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
	text-align: center;
	word-break: break-word;
}

.stage-status {
	font-size: 0.65rem;
	font-weight: 500;
}

.is-running {
	animation: pulse-border 1.8s ease-in-out infinite;
}

@keyframes pulse-border {
	0%,
	100% {
		opacity: 1;
	}
	50% {
		opacity: 0.6;
	}
}
</style>
