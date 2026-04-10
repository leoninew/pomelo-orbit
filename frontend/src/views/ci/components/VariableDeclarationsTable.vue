<template>
	<div>
		<div v-if="declarations.length === 0" class="text-sm text-base-content/60 py-4 text-center">
			暂无变量
		</div>
		<table v-else class="table w-full">
			<thead>
				<tr class="text-base-content/60 text-xs">
					<th>变量名</th>
					<th>变量值</th>
					<th>来源</th>
					<th>说明</th>
					<th class="w-24">操作</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="decl in declarations" :key="decl.name" class="hover">
					<td class="font-mono text-xs">{{ decl.name }}</td>
					<td class="font-mono text-xs cell-muted">
						{{ hasDisplayValue(decl.value) ? (decl.secret ? '••••••' : String(decl.value)) : '—' }}
					</td>
					<td>
						<span
							v-if="decl.source"
							class="badge badge-sm"
							:class="getSourceBadgeClass(decl.source)"
						>
							{{ getSourceLabel(decl.source) }}
						</span>
						<span v-else class="text-base-content/40">—</span>
					</td>
					<td class="cell-muted">{{ decl.description || '—' }}</td>
					<td>
						<div v-if="!readonly && canEdit(decl.source)">
							<button class="link link-primary text-xs" @click="emit('edit', decl.name)">
								编辑
							</button>
							<button class="link link-error text-xs ml-2" @click="emit('delete', decl.name)">
								删除
							</button>
						</div>
						<span v-else class="text-base-content/40 text-xs">—</span>
					</td>
				</tr>
			</tbody>
		</table>
	</div>
</template>

<script setup lang="ts">
import type { VariableDeclaration } from '@/types/ci/template';
import { getSourceBadgeClass, getSourceLabel, isVariableEditable } from '@/utils/variableSource';

withDefaults(
	defineProps<{
		declarations: VariableDeclaration[]
		readonly?: boolean
	}>(),
	{
		readonly: false,
	}
);

const emit = defineEmits<{
	(e: 'edit', name: string): void
	(e: 'delete', name: string): void
}>();

function hasDisplayValue(value: unknown) {
	if (value === null || value === undefined) {
		return false;
	}
	if (typeof value === 'string') {
		return value.trim().length > 0;
	}
	return true;
}

function canEdit(source?: string) {
	return source ? isVariableEditable(source) : false;
}
</script>
