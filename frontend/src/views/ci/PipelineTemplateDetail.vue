<template>
	<div class="flex flex-col gap-4">
		<div class="flex items-center justify-between flex-wrap gap-2">
			<h1 class="text-xl font-semibold flex items-center gap-2">
				{{ template?.name ?? '模板详情' }}
				<span v-if="template?.is_builtin" class="badge badge-sm badge-outline badge-info">
					内置
				</span>
			</h1>
			<div class="flex items-center gap-2">
				<button class="btn btn-sm btn-ghost gap-1" @click="$router.push('/ci/templates')">
					<ArrowLeft class="size-4" />
					返回
				</button>
				<button
					v-if="template && !template.is_builtin"
					class="btn btn-sm btn-ghost"
					@click="editModalRef?.showModal()"
				>
					编辑
				</button>
			</div>
		</div>

		<!-- Basic info -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<h2 class="font-semibold mb-3">基本信息</h2>
				<div v-if="loading" class="flex justify-center py-6">
					<span class="loading loading-spinner loading-md text-primary" />
				</div>
				<dl v-else-if="template" class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-3 text-sm">
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">模板名称</dt>
						<dd>{{ template.name }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">类型</dt>
						<dd>
							<span
								class="badge badge-sm"
								:class="template.is_builtin ? 'badge-outline badge-info' : 'badge-ghost'"
							>
								{{ template.is_builtin ? '内置模板' : '自定义模板' }}
							</span>
						</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">描述</dt>
						<dd class="text-base-content/60">{{ template.description || '—' }}</dd>
					</div>
					<div class="flex gap-2">
						<dt class="text-base-content/70 w-24 shrink-0">创建时间</dt>
						<dd>{{ formatTime(template.created_at) }}</dd>
					</div>
				</dl>
			</div>
		</div>

		<!-- Variable declarations -->
		<div class="card bg-base-100 shadow-sm">
			<div class="card-body p-5">
				<h2 class="font-semibold mb-3">变量声明</h2>
				<div
					v-if="!template || (template.variable_declarations ?? []).length === 0"
					class="text-sm text-base-content/60 py-4 text-center"
				>
					无变量声明
				</div>
				<table v-else class="table">
					<thead>
						<tr class="text-base-content/60">
							<th>变量名</th>
							<th>描述</th>
							<th>必填</th>
							<th>默认值</th>
						</tr>
					</thead>
					<tbody>
						<tr v-for="v in template.variable_declarations" :key="v.name" class="hover">
							<td>
								<code class="text-xs">{{ v.name }}</code>
							</td>
							<td class="cell-muted">{{ v.description || '—' }}</td>
							<td>
								<span
									class="badge badge-sm"
									:class="v.required ? 'badge-outline badge-error' : 'badge-ghost'"
								>
									{{ v.required ? '必填' : '可选' }}
								</span>
							</td>
							<td>
								<code v-if="v.default" class="text-xs">{{ v.default }}</code>
								<span v-else class="text-base-content/60">—</span>
							</td>
						</tr>
					</tbody>
				</table>
			</div>
		</div>

		<!-- Pipeline YAML -->
		<div class="card bg-base-100 shadow-sm flex-1">
			<div class="card-body p-5 flex flex-col">
				<div class="flex items-center justify-between mb-3">
					<h2 class="font-semibold">Pipeline YAML</h2>
					<div v-if="template && !template.is_builtin" class="flex items-center gap-2">
						<template v-if="editingYaml">
							<button class="btn btn-sm btn-primary" :disabled="operating" @click="handleSaveYaml">
								<span v-if="operating" class="loading loading-spinner loading-xs" />保存
							</button>
							<button class="btn btn-sm btn-ghost" @click="cancelEditYaml">取消</button>
						</template>
						<button v-else class="btn btn-sm btn-ghost" @click="editingYaml = true">编辑</button>
					</div>
				</div>
				<div style="height: 500px">
					<CodeEditor
						v-if="template"
						v-model:value="template.content"
						:style="{ height: '100%' }"
						theme="vs"
						language="yaml"
						:options="{
							readOnly: !editingYaml,
							minimap: { enabled: false },
							fontSize: 14,
							automaticLayout: true,
						}"
					/>
				</div>
			</div>
		</div>

		<!-- Edit info modal -->
		<dialog ref="editModalRef" class="modal">
			<div class="modal-box w-full max-w-lg">
				<h3 class="font-bold text-lg mb-4">编辑模板</h3>
				<div class="flex flex-col gap-3">
					<fieldset class="fieldset">
						<legend class="fieldset-legend">模板名称</legend>
						<input v-model="editForm.name" type="text" class="input w-full" />
					</fieldset>
					<fieldset class="fieldset">
						<legend class="fieldset-legend">描述</legend>
						<textarea v-model="editForm.description" class="textarea w-full" rows="2" />
					</fieldset>
				</div>
				<div class="modal-action">
					<button class="btn btn-primary" :disabled="operating" @click="handleEditOk">
						<span v-if="operating" class="loading loading-spinner loading-xs" />保存
					</button>
					<button class="btn btn-ghost" @click="editModalRef?.close()">取消</button>
				</div>
			</div>
			<form method="dialog" class="modal-backdrop"><button>close</button></form>
		</dialog>
	</div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue';
import { useRoute, useRouter } from 'vue-router';
import { ArrowLeft } from 'lucide-vue-next';
import { CodeEditor } from 'monaco-editor-vue3';
import { pipelineTemplateApi } from '@/api/ci';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import { formatTime } from '@/utils/time';
import type { PipelineTemplate } from '@/types/api';

const route = useRoute();
const router = useRouter();
const templateId = route.params.id as string;
const toast = useToast();

const { loading, execute } = useStatusAsync();
const { loading: operating, execute: executeOp } = useStatusAsync();

const template = ref<PipelineTemplate>();
const editModalRef = ref<HTMLDialogElement>();
const editingYaml = ref(false);
const editForm = reactive({ name: '', description: '' });
const originalContent = ref('');

async function fetchTemplate() {
	try {
		await execute(async () => {
			const data = await pipelineTemplateApi.get(templateId);
			template.value = data;
			Object.assign(editForm, { name: data.name, description: data.description ?? '' });
			originalContent.value = data.content;
		});
	} catch {
		toast.error('获取模板信息失败');
		router.push('/ci/templates');
	}
}

async function handleEditOk() {
	try {
		await executeOp(async () => {
			const data = await pipelineTemplateApi.update(templateId, {
				name: editForm.name,
				description: editForm.description || undefined,
			});
			template.value = data;
			toast.success('更新成功');
			editModalRef.value?.close();
		});
	} catch (error) { toast.error(error instanceof Error ? error.message : '更新失败'); }
}

function cancelEditYaml() {
	if (template.value) template.value.content = originalContent.value;
	editingYaml.value = false;
}

async function handleSaveYaml() {
	try {
		await executeOp(async () => {
			const data = await pipelineTemplateApi.update(templateId, { content: template.value?.content ?? '' });
			template.value = data;
			originalContent.value = data.content;
			editingYaml.value = false;
			toast.success('保存成功');
		});
	} catch (error) { toast.error(error instanceof Error ? error.message : '保存失败'); }
}

onMounted(fetchTemplate);
</script>
