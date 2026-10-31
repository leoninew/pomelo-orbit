<script setup lang="ts">
import { useScrollLock } from '@vueuse/core';
import { inBrowser, useData } from 'vitepress';
import { useSidebar, VPLink } from 'vitepress/theme';
import { ref, watch } from 'vue';

const props = defineProps<{ open: boolean }>();
const { page } = useData();
const { sidebar, hasSidebar } = useSidebar();
const openGroup = ref<number | null>(null);
const navEl = ref<HTMLElement | null>(null);
const scrollLocked = useScrollLock(inBrowser ? document.body : null);

function isCurrent(link?: string) {
  return link !== undefined && page.value.relativePath === `${link.slice(1)}.md`;
}

watch(
  [() => page.value.relativePath, sidebar],
  () => {
    const active = sidebar.value.findIndex((group) => group.items?.some((item) => isCurrent(item.link)));
    openGroup.value = active === -1 ? 0 : active;
  },
  { immediate: true },
);

watch(
  [() => props.open, navEl],
  ([open]) => {
    scrollLocked.value = open;
    if (open) navEl.value?.focus();
  },
  { immediate: true, flush: 'post' },
);
</script>

<template>
  <aside v-if="hasSidebar" class="VPSidebar" :class="{ open }" @click.stop>
    <div class="curtain" />
    <nav id="VPSidebarNav" ref="navEl" aria-label="文档目录" tabindex="-1">
      <ul class="sidebar-tree">
        <li v-for="(group, index) in sidebar" :key="group.text" class="sidebar-section">
          <h2>
            <button
              type="button"
              class="section-toggle"
              :aria-expanded="openGroup === index"
              :aria-controls="`sidebar-group-${index}`"
              @click="openGroup = openGroup === index ? null : index"
            >
              <span>{{ group.text }}</span>
              <span class="vpi-chevron-right caret" aria-hidden="true" />
            </button>
          </h2>
          <ul
            :id="`sidebar-group-${index}`"
            class="section-links"
            :hidden="openGroup !== index"
          >
            <li v-for="item in group.items" :key="item.link">
              <VPLink
                class="sidebar-link"
                :href="item.link"
                :aria-current="isCurrent(item.link) ? 'page' : undefined"
              >
                {{ item.text }}
              </VPLink>
            </li>
          </ul>
        </li>
      </ul>
    </nav>
  </aside>
</template>

<style scoped>
.VPSidebar {
  position: fixed;
  top: var(--vp-layout-top-height, 0px);
  bottom: 0;
  left: 0;
  z-index: var(--vp-z-index-sidebar);
  padding: 32px 32px 96px;
  width: calc(100vw - 64px);
  max-width: 320px;
  background-color: var(--vp-sidebar-bg-color);
  visibility: hidden;
  opacity: 0;
  box-shadow: var(--vp-c-shadow-3);
  overflow-x: hidden;
  overflow-y: auto;
  transform: translateX(-100%);
  transition: opacity 0.25s, transform 0.25s, visibility 0.25s;
  overscroll-behavior: contain;
}

.VPSidebar.open {
  visibility: visible;
  opacity: 1;
  transform: translateX(0);
}

@media (min-width: 960px) {
  .VPSidebar {
    padding-top: var(--vp-nav-height);
    width: var(--vp-sidebar-width);
    max-width: 100%;
    visibility: visible;
    opacity: 1;
    box-shadow: none;
    transform: translateX(0);
  }

  .curtain {
    position: sticky;
    top: -64px;
    left: 0;
    z-index: 1;
    margin-top: calc(var(--vp-nav-height) * -1);
    margin-right: -32px;
    margin-left: -32px;
    height: var(--vp-nav-height);
    background-color: var(--vp-sidebar-bg-color);
  }
}

@media (min-width: 1440px) {
  .VPSidebar {
    padding-left: max(32px, calc((100% - (var(--vp-layout-max-width) - 64px)) / 2));
    width: calc((100% - (var(--vp-layout-max-width) - 64px)) / 2 + var(--vp-sidebar-width) - 32px);
  }
}

nav {
  outline: none;
}

.sidebar-section {
  padding-bottom: 4px;
}

.section-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 4px 0;
  font-size: 14px;
  font-weight: 700;
  line-height: 24px;
  text-align: left;
  color: var(--vp-c-text-1);
  cursor: pointer;
}

.caret {
  flex-shrink: 0;
  font-size: 18px;
  color: var(--vp-c-text-3);
  transition: transform 0.2s;
}

.section-toggle[aria-expanded='true'] .caret {
  transform: rotate(90deg);
}

.section-links {
  border-left: 1px solid var(--vp-c-divider);
  padding-left: 16px;
}

.section-links[hidden] {
  display: none;
}

.sidebar-link {
  position: relative;
  display: block;
  padding: 4px 0;
  font-size: 14px;
  font-weight: 500;
  line-height: 24px;
  color: var(--vp-c-text-2);
}

.sidebar-link:hover,
.sidebar-link[aria-current='page'] {
  color: var(--vp-c-brand-1);
}

.sidebar-link[aria-current='page']::before {
  position: absolute;
  top: 6px;
  bottom: 6px;
  left: -17px;
  width: 2px;
  border-radius: 2px;
  background: var(--vp-c-brand-1);
  content: '';
}

.section-toggle:focus-visible,
.sidebar-link:focus-visible {
  outline: 2px solid var(--vp-c-brand-1);
  outline-offset: 2px;
  border-radius: 2px;
}
</style>
