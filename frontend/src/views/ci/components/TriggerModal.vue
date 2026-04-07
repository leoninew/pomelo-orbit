<template>
	<dialog ref="modalRef" class="modal">
		<div class="modal-box">
			<h3 class="font-bold text-lg mb-4">手动触发流水线</h3>
			<div class="flex flex-col gap-3">
				<fieldset class="fieldset">
					<legend class="fieldset-legend">模板</legend>
					<select v-model="form.template_id" class="select w-full">
						<option value="">选择模板</option>
						<option v-for="tpl in templates" :key="tpl.id" :value="tpl.id">
							{{ tpl.name }}
						</option>
					</select>
				</fieldset>

				<fieldset class="fieldset">
					<legend class="fieldset-legend">分支 / Ref</legend>
					<input
						v-model="form.trigger_ref"
						type="text"
						class="input w-full"
						placeholder="如：main 或 commit SHA"
					/>
				</fieldset>

				<!-- Variables -->
				<div v-if="currentTemplate" class="flex flex-col gap-2">
					<div class="text-sm font-medium">变量</div>
					<div v-if="variableList.length === 0" class="text-sm text-base-content/60">
						此模板无需配置变量
					</div>
					<div v-else class="flex flex-col gap-2">
						<div v-for="v in variableList" :key="v.name" class="flex flex-col gap-1">
							<label class="text-sm flex items-center gap-1">
								{{ v.name }}
								<span v-if="v.required" class="text-error">*</span>
							</label>
							<input
								v-model="form.variables[v.name]"
								:type="v.secret ? 'password' : 'text'"
								class="input input-sm w-full"
								:placeholder="v.default || `输入${v.name}`"
							/>
							<p v-if="v.description" class="fieldset-label text-base-content/50 text-xs">
								{{ v.description }}
							</p>
						</div>
					</div>
				</div>
			</div>

			<div class="modal-action">
				<button class="btn btn-primary" :disabled="operating || !canSubmit" @click="handleOk">
					<span v-if="operating" class="loading loading-spinner loading-xs" />
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
import type { PipelineTemplate } from '@/types/api';

const props = defineProps<{
	projectId: string
	templates: PipelineTemplate[]
	defaultBranch?: string
	projectVariables?: Record<string, string>
}>();

const emit = defineEmits<{
	trigger: [data: { template_id: string; trigger_ref: string; variables: Record<string, string> }]
}>();

const toast = useToast();
const { operating, execute: executeOp } = useStatusAsync();

const modalRef = ref<HTMLDialogElement>();

const form = reactive({
	template_id: '',
	trigger_ref: props.defaultBranch || 'main',
	variables: {} as Record<string, string>,
});

// 当前选择的模板
const currentTemplate = computed(() => {
	return props.templates.find((t) => t.id === form.template_id);
});

// 模板变量声明
const variableList = computed(() => {
	return currentTemplate.value?.variable_declarations || [];
});

// 是否可以提交
const canSubmit = computed(() => {
	if (!form.template_id) return false;
	// 检查必填变量是否都已填写
	const required = variableList.value.filter((v) => v.required);
	return required.every((v) => form.variables[v.name]?.trim());
});

// 监听模板变化，重置变量
watch(
	() => form.template_id,
	() => {
		form.variables = {};
		// 预填项目级变量
		if (props.projectVariables) {
			variableList.value.forEach((v) => {
				const projectVar = props.projectVariables?.[v.name];
				if (projectVar !== undefined) {
					form.variables[v.name] = projectVar;
				}
			});
		}
		// 预填默认值
		variableList.value.forEach((v) => {
			if (v.default && !form.variables[v.name]) {
				form.variables[v.name] = v.default;
			}
		});
	}
);

function open() {
	form.template_id = '';
	form.trigger_ref = props.defaultBranch || 'main';
	form.variables = {};
	modalRef.value?.showModal();
}

async function handleOk() {
	if (!canSubmit.value) {
		toast.error('请填写所有必填字段');
		return;
	}

	await executeOp(async () => {
		emit('trigger', {
			template_id: form.template_id,
			trigger_ref: form.trigger_ref,
			variables: form.variables,
		});
		modalRef.value?.close();
	});
}

defineExpose({ open });
</script>
