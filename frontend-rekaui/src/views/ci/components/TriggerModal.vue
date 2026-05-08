<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue';
import AppDialog from '@/components/AppDialog.vue';
import ComboboxSelect from '@/components/ComboboxSelect.vue';
import { useStatusAsync } from '@/composables/useStatusAsync';
import { useToast } from '@/composables/useToast';
import type { Repository } from '@/types/ci/repository';
import type { PipelineTemplate, VariableDeclaration } from '@/types/ci/template';

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

const isOpen = ref(false);
const initialVariableValues = ref<Record<string, string>>({});

const form = reactive({
	template_id: '',
	trigger_ref: props.defaultBranch || 'main',
	variables: {} as Record<string, string>,
});

const currentTemplate = computed(() =>
	props.templates.find((template) => template.id === form.template_id)
);
const templateOptions = computed(() =>
	props.templates.map((template) => ({
		value: template.id,
		label: template.name,
	}))
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
	return variable.source === 'template' || variable.source === 'repository';
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
	const projectValue = projectVariableMap.value[name];
	if (projectValue !== undefined) {
		return `项目值: ${projectValue}`;
	}
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
		const projectValue = projectVariableMap.value[variable.name];
		if (projectValue !== undefined) {
			nextValues[variable.name] = projectValue;
			continue;
		}
		nextValues[variable.name] = stringifyValue(variable.value);
	}
	form.variables = nextValues;
	initialVariableValues.value = { ...nextValues };
}

const canSubmit = computed(() => {
	if (!form.template_id) {
		return false;
	}
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
		if (isBuiltinVariable(variable)) {
			continue;
		}
		const currentValue = form.variables[variable.name] ?? '';
		const initialValue = initialVariableValues.value[variable.name] ?? '';
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
	isOpen.value = true;
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
		isOpen.value = false;
	});
}

defineExpose({ open });
</script>

<template>
	<AppDialog
		v-model:open="isOpen"
		title="触发流水线"
		description="选择流水线模板并配置运行变量。"
		width-class="w-[min(600px,calc(100vw-32px))]"
		content-class="overflow-hidden"
		body-class="max-h-[70vh] space-y-4 overflow-y-auto px-6 py-4"
	>
		<div class="space-y-1.5">
			<label class="app-field-label block">流水线模板</label>
			<ComboboxSelect
				v-model="form.template_id"
				:options="templateOptions"
				:open-on-focus="false"
				placeholder="请选择模板"
			/>
		</div>

		<div class="space-y-1.5">
			<label class="app-field-label block">触发分支/标签</label>
			<input
				v-model="form.trigger_ref"
				type="text"
				:placeholder="defaultBranch || 'main'"
				class="app-input"
			/>
		</div>

		<div v-if="variableList.length > 0" class="space-y-3">
			<h4 class="text-sm font-medium text-foreground">变量配置</h4>
			<div v-for="variable in variableList" :key="variable.name" class="space-y-1.5">
				<label class="app-field-label block">
					{{ variable.name }}
					<span v-if="variable.description" class="text-xs font-normal text-muted-foreground">
						- {{ variable.description }}
					</span>
				</label>
				<input
					v-model="form.variables[variable.name]"
					:type="variable.secret ? 'password' : 'text'"
					:placeholder="getVariablePlaceholder(variable.name)"
					:disabled="isBuiltinVariable(variable)"
					class="app-input"
				/>
			</div>
		</div>

		<template #footer>
			<button type="button" class="app-button" @click="isOpen = false">取消</button>
			<button
				type="button"
				:disabled="!canSubmit || loading"
				class="app-button-primary"
				@click="handleOk"
			>
				<span
					v-if="loading"
					class="size-4 animate-spin rounded-full border-2 border-primary-foreground border-t-transparent"
				/>
				{{ loading ? '触发中...' : '触发' }}
			</button>
		</template>
	</AppDialog>
</template>
