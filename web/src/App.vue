<template>
  <div class="min-h-screen bg-background text-foreground md:h-screen">
    <!-- Login page: no layout -->
    <RouterView v-if="isLoginPage" />

    <!-- Main layout -->
    <div v-else class="flex min-h-screen flex-col md:h-full md:min-h-0">
      <AppTopBar :current-module="currentPrimaryModule" />

      <div
        class="flex min-h-0 flex-1 flex-col gap-3 overflow-hidden p-3 md:flex-row md:gap-4 md:p-4"
      >
        <!-- Sidebar -->
        <aside
          v-if="currentScope"
          class="flex shrink-0 flex-col overflow-hidden rounded-2xl border border-border bg-card shadow-sm transition-all duration-200"
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
        <main class="min-w-0 flex-1 overflow-x-hidden overflow-y-auto">
          <RouterView />
        </main>
      </div>
    </div>

    <AppToaster />
  </div>
</template>

<script setup lang="ts">
  import { PanelLeftClose, PanelLeftOpen } from 'lucide-vue-next';
  import { computed, ref } from 'vue';
  import { useI18n } from 'vue-i18n';
  import { useRoute } from 'vue-router';
  import AppToaster from '@/components/AppToaster.vue';
  import AppTopBar from '@/components/AppTopBar.vue';
  import AppVerticalNav from '@/components/AppVerticalNav.vue';
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
