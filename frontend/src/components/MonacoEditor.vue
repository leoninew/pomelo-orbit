<template>
	<div class="monaco-editor-wrapper" :class="wrapperClass">
		<vue-monaco-editor
			v-model:value="content"
			:language="language"
			:theme="editorTheme"
			:options="editorOptions"
			:height="height"
			@mount="handleMount"
		/>
	</div>
</template>

<script setup lang="ts">
	import { computed, ref, watch } from 'vue';
	import { useTheme } from '@/composables/useTheme';
	import type { editor } from 'monaco-editor';

	interface Props {
		modelValue: string
		language?: string
		height?: string | number
		readonly?: boolean
		/** 是否显示错误状态边框 */
		hasError?: boolean
		/** 占位符文本（仅在空值时显示） */
		placeholder?: string
	}

	const props = withDefaults(defineProps<Props>(), {
		language: 'shell',
		height: '400px',
		readonly: false,
		hasError: false,
		placeholder: '',
	});

	const emit = defineEmits<{
		'update:modelValue': [value: string]
		mount: [editor: editor.IStandaloneCodeEditor]
	}>();

	const { theme } = useTheme();
	const editorInstance = ref<editor.IStandaloneCodeEditor>();

	const content = computed({
		get: () => props.modelValue,
		set: (value: string) => emit('update:modelValue', value),
	});

	const editorTheme = computed(() => (theme.value === 'dark' ? 'vs-dark' : 'vs'));

	const wrapperClass = computed(() => ({
		'monaco-editor-error': props.hasError,
		'monaco-editor-readonly': props.readonly,
	}));

	const editorOptions = computed<editor.IStandaloneEditorConstructionOptions>(() => ({
		minimap: { enabled: false },
		fontSize: 13,
		lineNumbers: 'on',
		scrollBeyondLastLine: false,
		automaticLayout: true,
		tabSize: 2,
		wordWrap: 'on',
		readOnly: props.readonly,
		scrollbar: {
			verticalScrollbarSize: 8,
			horizontalScrollbarSize: 8,
		},
		padding: {
			top: 8,
			bottom: 8,
		},
		renderLineHighlight: props.readonly ? 'none' : 'line',
		overviewRulerBorder: false,
		hideCursorInOverviewRuler: true,
		overviewRulerLanes: 0,
	}));

	function handleMount(editor: editor.IStandaloneCodeEditor) {
		editorInstance.value = editor;
		emit('mount', editor);
	}

	// 监听主题变化
	watch(
		() => theme.value,
		() => {
			if (editorInstance.value) {
				editorInstance.value.updateOptions({
					theme: editorTheme.value,
				});
			}
		}
	);
</script>

<style scoped>
	.monaco-editor-wrapper {
		height: 100%;
		overflow: hidden;
		border: 1px solid hsl(var(--border) / 0.5);
		border-radius: 20px;
		background: hsl(var(--card));
	}

	.monaco-editor-error {
		border-color: hsl(var(--destructive));
	}

	.monaco-editor-readonly {
		background: hsl(var(--muted) / 0.3);
	}

	/* 确保编辑器填充容器 */
	.monaco-editor-wrapper :deep(.monaco-editor) {
		border-radius: 20px;
	}
</style>
