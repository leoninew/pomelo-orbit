<template>
  <div class="h-dvh min-h-0 bg-background text-foreground">
    <!-- Login page: no layout -->
    <RouterView v-if="isLoginPage" />

    <!-- Main layout -->
    <div v-else class="flex h-full min-h-0 flex-col">
      <AppTopBar :current-module="currentPrimaryModule" />

      <div
        class="flex min-h-0 flex-1 flex-col gap-4 overflow-hidden p-3 md:flex-row md:gap-4 md:p-4"
      >
        <!-- Sidebar -->
        <aside
          v-if="currentScope"
          class="flex shrink-0 flex-col overflow-hidden rounded-lg border border-border bg-card shadow-sm transition-all duration-200"
          :class="collapsed ? 'md:w-16' : 'md:w-60'"
        >
          <AppVerticalNav
            :items="sidebarItems"
            :selected-key="selectedKey"
            :collapsed="collapsed"
          />

          <button
            class="hidden h-11 items-center justify-center border-t border-border text-muted-foreground transition-colors hover:bg-muted/50 hover:text-foreground md:flex"
            @click="collapsed = !collapsed"
          >
            <PanelLeftClose v-if="!collapsed" class="size-4" />
            <PanelLeftOpen v-else class="size-4" />
          </button>
        </aside>

        <!-- Main content -->
        <main
          class="flex min-w-0 flex-1 flex-col gap-2"
          :class="isDeploymentDialogue ? 'overflow-visible' : 'overflow-x-hidden overflow-y-auto'"
        >
          <AppBreadcrumb :items="breadcrumbItems" />
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
  import { PanelLeftClose, PanelLeftOpen } from '@lucide/vue';
  import { computed, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute } from 'vue-router';
  import AppToaster from '@/components/AppToaster.vue';
  import AppBreadcrumb from '@/components/AppBreadcrumb.vue';
  import AppTopBar from '@/components/AppTopBar.vue';
  import AppVerticalNav from '@/components/AppVerticalNav.vue';
  import { useTheme } from '@/composables/useTheme';
  import { provideBreadcrumbItems } from '@/composables/useBreadcrumbs';
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
  const breadcrumbItems = provideBreadcrumbItems();
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

  const isLoginPage = computed(() => route.name === 'Login');
</script>
