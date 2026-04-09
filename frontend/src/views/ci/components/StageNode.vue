<template>
	<div
		v-if="stage"
		class="stage-node"
		:class="[
			statusClass,
			{ selected: selected, readonly: readonly && !status, clickable: !readonly || !!status },
		]"
	>
		<div class="stage-node-header">
			<div class="stage-type-icon">
				<CheckCircle v-if="stage.builtin" class="size-4" />
				<Cog v-else class="size-4" />
			</div>
			<span class="stage-name">{{ stage.name }}</span>
		</div>
		<div class="stage-node-footer">
			<span v-if="stage.builtin" class="stage-type-badge badge-info">内置</span>
			<span v-else class="stage-type-badge">自定义</span>
			<span class="commands-count">
				{{ (stage.script || '').split('\n').filter((l) => l.trim()).length }} 行
			</span>
			<div v-if="status" class="status-icon">
				<CheckCircle v-if="status === 'ran_to_completion'" class="size-3 text-success" />
				<XCircle v-else-if="status === 'faulted'" class="size-3 text-error" />
				<Loader2 v-else-if="status === 'running'" class="size-3 text-info animate-spin" />
				<Clock v-else-if="status === 'waiting_to_run'" class="size-3 text-warning" />
			</div>
		</div>
	</div>
</template>

<script setup lang="ts">
import { CheckCircle, Clock, Cog, Loader2, XCircle } from 'lucide-vue-next';
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

const stage = computed(() => props.data.stage);
const status = computed(() => props.data.status);
const readonly = computed(() => props.data.readonly ?? false);
const selected = computed(() => props.data.selected ?? false);

const statusClass = computed(() => {
	if (selected.value) {
		return 'selected';
	}
	if (!status.value) {
		return '';
	}
	const statusBorderMap: Record<string, string> = {
		ran_to_completion: 'border-success',
		faulted: 'border-error',
		running: 'border-info',
		waiting_to_run: 'border-warning',
		canceled: 'border-base-300',
	};
	return statusBorderMap[status.value] ?? '';
});
</script>

<style scoped>
.stage-node {
	background-color: hsl(var(--b1));
	border: 2px solid hsl(var(--bc) / 0.3);
	border-radius: 0.5rem;
	box-shadow: 0 2px 4px 0 rgb(0 0 0 / 0.1);
	transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
	min-width: 140px;
	padding: 0.5rem 0.75rem;
	animation: node-enter 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
}

@keyframes node-enter {
	from {
		opacity: 0;
		transform: scale(0.8) translateY(10px);
	}
	to {
		opacity: 1;
		transform: scale(1) translateY(0);
	}
}

.stage-node.clickable {
	cursor: pointer;
}

.stage-node.clickable:hover {
	transform: translateY(-2px);
	box-shadow: 0 4px 12px 0 rgb(0 0 0 / 0.1);
	border-color: hsl(var(--in));
}

.stage-node.readonly {
	cursor: default;
}

.stage-node.selected {
	border: 2px solid hsl(var(--p));
	box-shadow: 0 0 0 3px hsl(var(--p) / 0.2);
	animation: pulse-select 1.5s ease-in-out infinite;
}

@keyframes pulse-select {
	0%,
	100% {
		box-shadow: 0 0 0 3px hsl(var(--p) / 0.2);
	}
	50% {
		box-shadow: 0 0 0 5px hsl(var(--p) / 0.3);
	}
}

.stage-node.border-success {
	border: 2px solid hsl(var(--su));
	animation: status-success 0.5s ease-out;
}

@keyframes status-success {
	from {
		border-color: hsl(var(--wa));
	}
	to {
		border-color: hsl(var(--su));
	}
}

.stage-node.border-error {
	border: 2px solid hsl(var(--er));
	animation: shake 0.5s cubic-bezier(0.36, 0.07, 0.19, 0.97);
}

@keyframes shake {
	10%,
	90% {
		transform: translate3d(-1px, 0, 0);
	}
	20%,
	80% {
		transform: translate3d(2px, 0, 0);
	}
	30%,
	50%,
	70% {
		transform: translate3d(-4px, 0, 0);
	}
	40%,
	60% {
		transform: translate3d(4px, 0, 0);
	}
}

.stage-node.border-info {
	border: 2px solid hsl(var(--in));
	animation: pulse-running 2s ease-in-out infinite;
}

@keyframes pulse-running {
	0%,
	100% {
		border-color: hsl(var(--in));
		box-shadow: 0 0 0 0 hsl(var(--in) / 0.4);
	}
	50% {
		border-color: hsl(var(--in) / 0.7);
		box-shadow: 0 0 0 8px hsl(var(--in) / 0);
	}
}

.stage-node.border-warning {
	border: 2px solid hsl(var(--wa));
}

.stage-node-header {
	display: flex;
	align-items: center;
	gap: 0.5rem;
	margin-bottom: 0.25rem;
}

.stage-type-icon {
	flex-shrink: 0;
	color: hsl(var(--in));
}

.stage-name {
	font-weight: 500;
	font-size: 0.75rem;
	overflow: hidden;
	text-overflow: ellipsis;
	white-space: nowrap;
}

.stage-node-footer {
	display: flex;
	align-items: center;
	justify-content: space-between;
	gap: 0.5rem;
}

.stage-type-badge {
	font-size: 0.6875rem;
	padding-left: 0.375rem;
	padding-right: 0.375rem;
	padding-top: 0.125rem;
	padding-bottom: 0.125rem;
	border-radius: 0.25rem;
	background-color: hsl(var(--b2));
	color: hsl(var(--bc) / 0.7);
}

.badge-info {
	background-color: hsl(var(--in) / 0.15);
	color: hsl(var(--in));
}

.commands-count {
	font-size: 0.6875rem;
	color: hsl(var(--bc) / 0.6);
}

.status-icon {
	flex-shrink: 0;
}
</style>
