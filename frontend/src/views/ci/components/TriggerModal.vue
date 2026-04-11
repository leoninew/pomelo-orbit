<template>
	<dialog ref="modalRef" class="modal">
		<div class="modal-box">
			<h3 class="font-bold text-lg mb-4">手动触发流水线</h3>
			<div class="flex flex-col gap-3">
				<fieldset class="fieldset">
					<legend class="fieldset-legend">分支 / Ref</legend>
					<input
						v-model="form.trigger_ref"
						type="text"
						class="input w-full"
						placeholder="如：main 或 commit SHA"
					/>
				</fieldset>

				<fieldset class="fieldset">
					<legend class="fieldset-legend">模板</legend>
					<select v-model="form.template_id" class="select w-full">
						<option value="">选择模板</option>
						<option v-for="tpl in templates" :key="tpl.id" :value="tpl.id">
							{{ tpl.name }}
						</option>
					</select>
				</fieldset>

				<div v-if="currentTemplate" class="flex flex-col gap-2">
					<div class="text-sm font-medium">变量</div>
					<div v-if="variableList.length === 0" class="text-sm text-base-content/60">暂无数据</div>
					<div v-else class="flex flex-col gap-2">
						<div v-for="variable in variableList" :key="variable.name" class="flex flex-col gap-1">
							<label class="text-sm flex items-center gap-2">
								<span>{{ variable.name }}</span>
								<span v-if="isBuiltinVariable(variable)" class="badge badge-xs badge-ghost">
									内置
								</span>
							</label>
							<input
								v-model="form.variables[variable.name]"
								:type="variable.secret ? 'password' : 'text'"
								class="input input-sm w-full"
								:class="{
									'opacity-70 cursor-not-allowed': isBuiltinVariable(variable),
									'input-error': !hasVariableValue(variable.name) && !isBuiltinVariable(variable),
								}"
								:disabled="isBuiltinVariable(variable)"
								:placeholder="getVariablePlaceholder(variable.name)"
							/>
							<p v-if="variable.description" class="fieldset-label text-base-content/50 text-xs">
								{{ variable.description }}
							</p>
						</div>
					</div>
				</div>
			</div>

			<div class="modal-action">
				<button class="btn btn-primary" :disabled="loading || !canSubmit" @click="handleOk">
					<span v-if="loading" class="loading loading-spinner loading-xs" />
					触发
				</button>
				<button class="btn btn-ghost" @click="modalRef?.close()">取消</button>
			</div>
		</div>
		<form method="dialog" class="modal-backdrop"><button>close</button></form>
	</dialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { PipelineTemplate, VariableDeclaration } from '@/types/ci/template';
import type { Repository } from '@/types/ci/repository';

const props = defineProps<{
	repositoryId: string
	templates: PipelineTemplate[]
	defaultBranch?: string
	projectVariables?: VariableDeclaration[]
	repository?: Repository
}>();

const emit = defineEmits<{
	trigger: [
		data: {
			template_id: string
			trigger_ref: string
			variables: Record<string, string>
		},
	]
}>();

const toast = useToast();
const { loading, execute: executeOp } = useStatusAsync();

const modalRef = ref<HTMLDialogElement>();
const initialVariableValues = ref<Record<string, string>>({});

const form = reactive({
	template_id: '',
	trigger_ref: props.defaultBranch || 'main',
	variables: {} as Record<string, string>,
});

const currentTemplate = computed(() =>
	props.templates.find((template) => template.id === form.template_id)
);
const variableList = computed(() => currentTemplate.value?.variable_declarations || []);
const projectVariableMap = computed(() => {
	const result: Record<string, string> = {};
	for (const variable of props.projectVariables || []) {
		if (variable.value == null) {
			continue;
		}
		result[variable.name] = String(variable.value);
	}
	return result;
});

function isBuiltinVariable(variable: VariableDeclaration): boolean {
	// 内置变量包括：运行时变量（template、repository）和从 Stage 发现的变量
	return (
		variable.source === 'template' ||
		variable.source === 'repository' ||
		variable.source === 'template_stage'
	);
}

function stringifyValue(value: unknown): string {
	if (value == null) {
		return '';
	}
	return String(value);
}

function hasValue(value: unknown): boolean {
	return typeof value === 'string' ? value.trim().length > 0 : value != null;
}

function hasVariableValue(name: string): boolean {
	return hasValue(form.variables[name]);
}

function getBuiltinDisplayValue(variable: VariableDeclaration): string {
	// 内置变量显示描述文案，实际值由后端运行时注入
	if (variable.description) {
		return variable.description;
	}
	if (variable.value != null) {
		return stringifyValue(variable.value);
	}
	return '';
}

function syncBuiltinVariables() {
	for (const variable of variableList.value) {
		if (!isBuiltinVariable(variable)) {
			continue;
		}
		form.variables[variable.name] = getBuiltinDisplayValue(variable);
	}
}

function getVariablePlaceholder(name: string): string {
	const variable = variableList.value.find((item) => item.name === name);
	if (!variable) {
		return '';
	}
	// 如果项目有覆盖值，显示项目值
	const projectValue = projectVariableMap.value[name];
	if (projectValue !== undefined) {
		return `项目值: ${projectValue}`;
	}
	// 否则显示模板默认值
	if (variable.value != null) {
		return stringifyValue(variable.value);
	}
	return `输入${name}`;
}

function initializeVariables() {
	const nextValues: Record<string, string> = {};
	for (const variable of variableList.value) {
		if (isBuiltinVariable(variable)) {
			nextValues[variable.name] = getBuiltinDisplayValue(variable);
			continue;
		}
		// 优先使用项目变量值
		const projectValue = projectVariableMap.value[variable.name];
		if (projectValue !== undefined) {
			nextValues[variable.name] = projectValue;
			continue;
		}
		// 其次使用模板默认值
		nextValues[variable.name] = stringifyValue(variable.value);
	}
	form.variables = nextValues;
	initialVariableValues.value = { ...nextValues };
}

const canSubmit = computed(() => {
	if (!form.template_id) {
		return false;
	}
	// 只检查非内置变量是否有值
	return variableList.value
		.filter((variable) => !isBuiltinVariable(variable))
		.every((variable) => hasVariableValue(variable.name));
});

watch(
	() => form.template_id,
	() => {
		initializeVariables();
	}
);

watch(
	() => form.trigger_ref,
	() => {
		syncBuiltinVariables();
	}
);

watch(
	() => props.repository,
	() => {
		syncBuiltinVariables();
	},
	{ deep: true }
);

function buildRuntimeOverrides(): Record<string, string> {
	const overrides: Record<string, string> = {};
	for (const variable of variableList.value) {
		// 跳过内置变量
		if (isBuiltinVariable(variable)) {
			continue;
		}
		const currentValue = form.variables[variable.name] ?? '';
		const initialValue = initialVariableValues.value[variable.name] ?? '';
		// 只发送用户修改过的变量
		if (currentValue !== initialValue) {
			overrides[variable.name] = currentValue;
		}
	}
	return overrides;
}

function open() {
	form.template_id = '';
	form.trigger_ref = props.defaultBranch || 'main';
	form.variables = {};
	initialVariableValues.value = {};
	modalRef.value?.showModal();
}

async function handleOk() {
	if (!canSubmit.value) {
		toast.error('请为所有变量提供值');
		return;
	}

	await executeOp(async () => {
		emit('trigger', {
			template_id: form.template_id,
			trigger_ref: form.trigger_ref,
			variables: buildRuntimeOverrides(),
		});
		modalRef.value?.close();
	});
}

defineExpose({ open });
</script>
