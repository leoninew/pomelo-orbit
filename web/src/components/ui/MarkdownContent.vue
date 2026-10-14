<template>
  <!-- eslint-disable-next-line vue/no-v-html -->
  <div class="markdown-content" v-html="html" />
</template>

<script setup lang="ts">
  import createDOMPurify from 'dompurify';
  import { marked } from 'marked';
  import { computed } from 'vue';

  const props = defineProps<{ content: string }>();
  const sanitizer = createDOMPurify(window);

  const html = computed(() =>
    sanitizer.sanitize(marked.parse(props.content, { async: false, breaks: true, gfm: true }))
  );
</script>

<style scoped>
  .markdown-content :deep(p) {
    margin: 0;
    white-space: pre-wrap;
  }

  .markdown-content :deep(p + p),
  .markdown-content :deep(ul),
  .markdown-content :deep(ol),
  .markdown-content :deep(pre),
  .markdown-content :deep(table) {
    margin-top: 0.75rem;
  }

  .markdown-content :deep(ul),
  .markdown-content :deep(ol) {
    margin-bottom: 0;
    padding-left: 1.25rem;
  }

  .markdown-content :deep(ul) {
    list-style-type: disc;
  }

  .markdown-content :deep(ol) {
    list-style-type: decimal;
  }

  .markdown-content :deep(a) {
    color: hsl(var(--primary));
    text-decoration: underline;
    text-underline-offset: 0.25rem;
  }

  .markdown-content :deep(code) {
    overflow-wrap: anywhere;
    border-radius: 0.25rem;
    background: hsl(var(--muted));
    padding: 0.125rem 0.25rem;
    font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
    font-size: 0.75rem;
  }

  .markdown-content :deep(pre) {
    max-width: 100%;
    overflow-x: auto;
    border: 1px solid hsl(var(--border));
    border-radius: 0.375rem;
    background: hsl(var(--background));
    padding: 0.75rem;
  }

  .markdown-content :deep(pre code) {
    white-space: pre-wrap;
    background: transparent;
    padding: 0;
  }

  .markdown-content :deep(table) {
    width: 100%;
    table-layout: fixed;
    border-collapse: collapse;
    font-size: 0.75rem;
    text-align: left;
  }

  .markdown-content :deep(th) {
    border: 1px solid hsl(var(--border));
    background: hsl(var(--muted) / 0.5);
    padding: 0.375rem 0.5rem;
    vertical-align: top;
    color: hsl(var(--foreground));
    font-weight: 600;
  }

  .markdown-content :deep(td) {
    overflow-wrap: break-word;
    border: 1px solid hsl(var(--border));
    padding: 0.375rem 0.5rem;
    vertical-align: top;
  }

  .markdown-content :deep(td code) {
    font-size: 0.6875rem;
  }
</style>
