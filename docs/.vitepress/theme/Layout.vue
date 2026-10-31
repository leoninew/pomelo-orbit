<script setup lang="ts">
import { useData, useRoute } from 'vitepress';
import { useSidebar } from 'vitepress/theme';
import { watch } from 'vue';
import VPBackdrop from 'vitepress/dist/client/theme-default/components/VPBackdrop.vue';
import VPContent from 'vitepress/dist/client/theme-default/components/VPContent.vue';
import VPFooter from 'vitepress/dist/client/theme-default/components/VPFooter.vue';
import VPLocalNav from 'vitepress/dist/client/theme-default/components/VPLocalNav.vue';
import VPNav from 'vitepress/dist/client/theme-default/components/VPNav.vue';
import VPSkipLink from 'vitepress/dist/client/theme-default/components/VPSkipLink.vue';
import { useCloseSidebarOnEscape } from 'vitepress/dist/client/theme-default/composables/sidebar';
import Sidebar from './Sidebar.vue';

const { frontmatter } = useData();
const route = useRoute();
const { isOpen, open, close } = useSidebar();

watch(() => route.path, close);
useCloseSidebarOnEscape(isOpen, close);
</script>

<template>
  <div v-if="frontmatter.layout !== false" class="Layout" :class="frontmatter.pageClass">
    <VPSkipLink />
    <VPBackdrop :show="isOpen" @click="close" />
    <VPNav />
    <VPLocalNav :open="isOpen" @open-menu="open" />
    <Sidebar :open="isOpen" />
    <VPContent />
    <VPFooter />
  </div>
  <Content v-else />
</template>

<style scoped>
.Layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}
</style>
