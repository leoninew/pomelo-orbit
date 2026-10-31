<script setup lang="ts">
import { onMounted, ref, useId, watch } from 'vue';
import { useData } from 'vitepress';

const props = defineProps<{ source: string }>();
const { isDark } = useData();
const svg = ref('');
const error = ref('');
const id = `diagram-${useId().replace(/[^a-zA-Z0-9-]/g, '-')}`;
let revision = 0;

async function render() {
  const current = ++revision;
  try {
    const { default: mermaid } = await import('mermaid');
    mermaid.initialize({
      startOnLoad: false,
      securityLevel: 'strict',
      theme: isDark.value ? 'dark' : 'default',
      fontFamily: 'inherit',
      flowchart: { htmlLabels: false, useMaxWidth: true },
    });
    const result = await mermaid.render(`${id}-${current}`, decodeURIComponent(props.source));
    if (current === revision) {
      svg.value = result.svg;
      error.value = '';
    }
  } catch (cause) {
    if (current === revision) {
      error.value = '图示加载失败，请刷新页面重试。';
      console.error('Failed to render Mermaid diagram', cause);
    }
  }
}

onMounted(() => {
  render();
  watch([() => props.source, isDark], render);
});
</script>

<template>
  <figure class="mermaid-diagram" aria-label="流程与关系图" :aria-busy="!svg && !error">
    <p v-if="error" role="alert">{{ error }}</p>
    <div v-else-if="svg" v-html="svg" />
    <p v-else>正在加载图示…</p>
    <noscript>启用 JavaScript 后可查看图示。</noscript>
  </figure>
</template>
