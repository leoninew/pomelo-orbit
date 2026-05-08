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

function effectiveValue(decl: VariableDeclaration) {
	return decl.value ?? decl.default;
}

function canEdit(decl: VariableDeclaration) {
	// 优先使用后端明确设置的 editable 字段，回退到 source 推断
	if (decl.editable !== undefined) {
		return decl.editable;
	}
	return decl.source ? isVariableEditable(decl.source) : false;
}
</script>

<template>
	<div class="overflow-hidden">
		<div v-if="declarations.length === 0" class="px-5 py-10 text-center text-muted-foreground">
			<p class="text-sm">暂无变量</p>
		</div>
		<table v-else class="app-table-detail">
			<thead>
				<tr>
					<th>变量名</th>
					<th>说明</th>
					<th>变量值</th>
					<th>来源</th>
					<th v-if="!readonly">操作</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="decl in declarations" :key="decl.name">
					<td>
						<span class="text-foreground">{{ decl.name }}</span>
					</td>
					<td class="max-w-md truncate" :title="decl.description">
						<span v-if="decl.description" class="text-muted-foreground">
							{{ decl.description }}
						</span>
						<span v-else class="text-muted-foreground">—</span>
					</td>
					<td class="max-w-sm truncate" :title="String(effectiveValue(decl) ?? '')">
						<span v-if="hasDisplayValue(effectiveValue(decl))" class="text-foreground">
							{{ effectiveValue(decl) }}
						</span>
						<span v-else class="text-muted-foreground italic">未设置</span>
					</td>
					<td>
						<span
							class="inline-block rounded px-2 py-0.5 text-sm"
							:class="
								decl.source
									? getSourceBadgeClass(decl.source)
									: 'border border-border bg-muted text-muted-foreground'
							"
						>
							{{ decl.source ? getSourceLabel(decl.source) : '未知' }}
						</span>
					</td>
					<td v-if="!readonly">
						<div class="flex items-center gap-2">
							<button v-if="canEdit(decl)" class="app-link" @click="emit('edit', decl.name)">
								编辑
							</button>
							<button
								v-if="canEdit(decl)"
								class="app-link-danger"
								@click="emit('delete', decl.name)"
							>
								删除
							</button>
							<span v-if="!canEdit(decl)" class="text-muted-foreground">—</span>
						</div>
					</td>
				</tr>
			</tbody>
		</table>
	</div>
</template>
