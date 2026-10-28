<template>
  <div class="h-dvh min-h-0 bg-background text-foreground">
    <!-- Standalone auth handoff pages: no application chrome -->
    <RouterView v-if="isStandalonePage" />

    <!-- Main layout -->
    <div v-else class="flex h-full min-h-0 flex-col">
      <AppTopBar :current-module="currentPrimaryModule" />

      <div class="flex min-h-0 flex-1 flex-col overflow-hidden md:flex-row">
        <!-- Sidebar -->
        <aside
          v-if="currentScope"
          class="app-sidebar flex shrink-0 flex-col overflow-hidden border-b border-border bg-background transition-all duration-200 motion-reduce:transition-none md:border-b-0 md:border-r"
          :class="collapsed ? 'md:w-16' : 'md:w-60'"
          :data-collapsed="collapsed ? '' : undefined"
        >
          <AppVerticalNav
            :items="sidebarItems"
            :selected-key="selectedKey"
            :collapsed="collapsed"
          />

          <button
            class="app-icon-button m-3 hidden w-auto shrink-0 md:flex"
            @click="collapsed = !collapsed"
          >
            <PanelLeftClose v-if="!collapsed" class="size-4" />
            <PanelLeftOpen v-else class="size-4" />
          </button>
        </aside>

        <!-- Main content -->
        <main
          class="app-workspace-main flex min-h-0 min-w-0 flex-1 flex-col p-4 md:p-6 xl:p-10"
          :class="isDeploymentDialogue ? 'overflow-visible' : 'overflow-x-hidden overflow-y-auto'"
        >
          <div class="min-h-0 flex-1">
            <RouterView />
          </div>
        </main>
      </div>
    </div>

    <AppToaster />
  </div>
</template>

<script setup lang="ts">
  import '@/styles/workspace.css';
  import { PanelLeftClose, PanelLeftOpen } from '@lucide/vue';
  import { computed, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute } from 'vue-router';
  import AppToaster from '@/components/ui/AppToaster.vue';
  import AppTopBar from '@/components/layout/AppTopBar.vue';
  import AppVerticalNav from '@/components/layout/AppVerticalNav.vue';
  import { useTheme } from '@/composables/useTheme';
  import {
    getNavigationScope,
    getPrimaryNavigationKey,
    resolveSecondaryNavigation,
  } from '@/navigation';
  import { useAuthStore } from '@/stores/auth';

  const route = useRoute();
  const authStore = useAuthStore();
  const { t } = useI18n({ useScope: 'global' });
  useTheme();

  const collapsed = ref(false);
  const currentScope = computed(() => getNavigationScope(route.path));
  const selectedKey = computed(() => (route.meta.menuKey as string) ?? '');
  const currentPrimaryModule = computed(() => getPrimaryNavigationKey(route.path));
  const isDeploymentDialogue = computed(() => route.name === 'DeploymentDialogue');

  const sidebarItems = computed(() => {
    if (!currentScope.value) {
      return [];
    }
    return resolveSecondaryNavigation(currentScope.value, {
      hasPermission: (permission) => authStore.hasPermission(permission),
      t: (key) => t(key),
    });
  });

  const isStandalonePage = computed(() => route.meta.standalone === true);
</script>
