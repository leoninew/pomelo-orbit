<template>
	<div>
		<div v-if="declarations.length === 0" class="text-base-content/60 py-4 text-center">
			暂无变量，Stage 脚本中使用
			<code class="text-xs bg-base-200 px-1 rounded">&#123;&#123; VAR_NAME &#125;&#125;</code>
			占位符后自动提取
		</div>
		<table v-else class="table table-sm w-full">
			<thead>
				<tr class="text-base-content/60">
					<th>变量名</th>
					<th class="w-16">内置</th>
					<th>变量值</th>
					<th class="w-12">敏感</th>
					<th class="w-20">操作</th>
				</tr>
			</thead>
			<tbody>
				<tr v-for="decl in declarations" :key="decl.name" class="hover">
					<td>{{ decl.name }}</td>
					<td>
						<span v-if="decl.builtin" class="badge badge-xs badge-ghost">内置</span>
					</td>
					<td>
						<template v-if="editingName === decl.name">
							<input
								:ref="
									(el) => {
										if (el) editInputRef = el as HTMLInputElement;
									}
								"
								v-model="editValue"
								type="text"
								class="input input-xs w-full"
								placeholder="变量值"
								@keyup.enter="commitEdit(decl.name)"
								@keyup.esc="cancelEdit"
							/>
						</template>
						<span v-else-if="decl.default">
							{{ decl.secret ? '••••••' : decl.default }}
						</span>
						<span v-else class="text-base-content/30">未设置</span>
					</td>
					<td>
						<input
							type="checkbox"
							class="checkbox checkbox-xs"
							:checked="decl.secret"
							@change="update(decl.name, 'secret', ($event.target as HTMLInputElement).checked)"
						/>
					</td>
					<td>
						<button
							v-if="editingName !== decl.name"
							class="link link-primary"
							@click="startEdit(decl)"
						>
							编辑
						</button>
						<div v-else class="flex items-center gap-1">
							<button class="link link-primary" @click="commitEdit(decl.name)">确定</button>
							<button class="link link-ghost" @click="cancelEdit">取消</button>
						</div>
					</td>
				</tr>
			</tbody>
		</table>
	</div>
</template>

<script setup lang="ts">
import { nextTick, ref } from 'vue';
import type { VariableDeclaration } from '@/types/ci';

const props = defineProps<{ declarations: VariableDeclaration[] }>();

const emit = defineEmits<{
	(e: 'update:declarations', value: VariableDeclaration[]): void
}>();

const editingName = ref<string | null>(null);
const editValue = ref('');
const editInputRef = ref<HTMLInputElement | null>(null);

function update(name: string, field: keyof VariableDeclaration, value: unknown) {
	emit(
		'update:declarations',
		props.declarations.map((d) => (d.name === name ? { ...d, [field]: value } : d))
	);
}

function startEdit(decl: VariableDeclaration) {
	editingName.value = decl.name;
	editValue.value = decl.default ?? '';
	nextTick(() => editInputRef.value?.focus());
}

function cancelEdit() {
	editingName.value = null;
}

function commitEdit(name: string) {
	update(name, 'default', editValue.value || null);
	editingName.value = null;
}
</script>
