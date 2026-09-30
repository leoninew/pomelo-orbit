<template>
  <div class="flex h-full min-h-0 min-w-0 flex-1 flex-col">
    <div class="relative min-h-0 flex-1">
      <MonacoEditor
        :model-value="initialText"
        language="plaintext"
        height="100%"
        :readonly="true"
        @mount="mountEditor"
      />
      <div
        v-if="!state?.text"
        class="pointer-events-none absolute inset-0 flex items-center justify-center text-sm text-muted-foreground"
      >
        {{ emptyText }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue';
  import { useI18n } from 'vue-i18n';
  import MonacoEditor from '@/components/MonacoEditor.vue';
  import type { LogState } from '@/composables/useLogStream';
  import type { editor, IDisposable } from 'monaco-editor';

  const props = defineProps<{ state?: LogState }>();
  const { t } = useI18n();
  const initialText = ref(props.state?.text ?? '');
  const emptyText = computed(() =>
    t(
      props.state?.status === 'complete' || props.state?.status === 'error'
        ? 'logs.empty'
        : 'logs.waiting'
    )
  );
  let instance: editor.IStandaloneCodeEditor | undefined;
  let scrollListener: IDisposable | undefined;
  let applying = false;
  let version = props.state?.version ?? 0;

  function bottom() {
    if (instance) {
      instance.setScrollTop(instance.getScrollHeight());
    }
  }
  function save() {
    if (instance) {
      props.state?.saveViewState(instance.saveViewState());
    }
  }
  function follow() {
    const following = !props.state?.follow;
    props.state?.setFollow(following);
    if (following) {
      bottom();
    }
  }
  function find() {
    void instance?.getAction('actions.find')?.run();
  }
  defineExpose({ find, follow });

  function mountEditor(ed: editor.IStandaloneCodeEditor) {
    instance = ed;
    instance.updateOptions({ contextmenu: false, find: { addExtraSpaceOnTop: false } });
    instance.getModel()?.setValue(props.state?.text ?? '');
    version = props.state?.version ?? 0;
    if (props.state?.follow) {
      bottom();
    } else if (props.state?.viewState) {
      instance.restoreViewState(props.state.viewState);
    }
    scrollListener = ed.onDidScrollChange(() => {
      if (applying || !props.state || !instance) {
        return;
      }
      props.state.setFollow(
        instance.getScrollTop() + instance.getLayoutInfo().height >= instance.getScrollHeight() - 8
      );
      save();
    });
  }

  watch(
    () => props.state,
    (state, previous) => {
      if (previous && instance) {
        previous.saveViewState(instance.saveViewState());
      }
      initialText.value = state?.text ?? '';
      instance?.getModel()?.setValue(initialText.value);
      version = state?.version ?? 0;
      if (state?.follow) {
        bottom();
      } else if (state?.viewState) {
        instance?.restoreViewState(state.viewState);
      }
    }
  );

  watch(
    () => props.state?.version,
    () => {
      const state = props.state;
      const model = instance?.getModel();
      if (!state || !instance || !model) {
        initialText.value = state?.text ?? '';
        return;
      }
      applying = true;
      const scrollTop = instance.getScrollTop();
      const visibleLine = instance.getVisibleRanges()[0]?.startLineNumber ?? 1;
      const lineOffset = instance.getTopForLineNumber(visibleLine) - scrollTop;
      if (state.patch && state.version === version + 1) {
        const end = model.getPositionAt(model.getValueLength());
        model.applyEdits([
          {
            range: {
              startLineNumber: end.lineNumber,
              startColumn: end.column,
              endLineNumber: end.lineNumber,
              endColumn: end.column,
            },
            text: state.patch.append,
          },
        ]);
        if (state.patch.removedChars) {
          const end = model.getPositionAt(state.patch.removedChars);
          model.applyEdits([
            {
              range: {
                startLineNumber: 1,
                startColumn: 1,
                endLineNumber: end.lineNumber,
                endColumn: end.column,
              },
              text: '',
            },
          ]);
        }
      } else {
        model.setValue(state.text);
      }
      version = state.version;
      if (state.follow) {
        bottom();
      } else {
        instance.setScrollTop(
          Math.max(
            0,
            instance.getTopForLineNumber(
              Math.max(1, visibleLine - (state.patch?.removedLines ?? 0))
            ) - lineOffset
          )
        );
      }
      save();
      void nextTick(() => {
        applying = false;
      });
    },
    { flush: 'post' }
  );

  onBeforeUnmount(() => {
    save();
    scrollListener?.dispose();
    instance = undefined;
  });
</script>
