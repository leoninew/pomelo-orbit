<script setup lang="ts">
const colorMode = useColorMode()
const { logout } = useAuth()

function toggleTheme() {
  colorMode.preference = colorMode.preference === 'dark' ? 'light' : 'dark'
}

function handleLogout() {
  logout()
}
</script>

<template>
  <div class="min-h-screen bg-base text-base">
    <!-- Header -->
    <header class="sticky top-0 z-50 w-full border-b border-base bg-base/95 backdrop-blur">
      <div class="container mx-auto flex h-16 items-center justify-between px-4">
        <!-- Left: Logo -->
        <NuxtLink to="/" class="flex items-center gap-2 hover:opacity-80 transition-opacity">
          <div class="i-lucide-orbit w-6 h-6 text-primary" />
          <span class="text-xl font-bold">Pomelo Orbit</span>
        </NuxtLink>

        <!-- Right: Actions -->
        <div class="flex items-center gap-2">
          <!-- Theme Toggle -->
          <NButton
            btn="solid-gray"
            size="sm"
            :label="colorMode.preference === 'dark' ? 'i-lucide-moon' : 'i-lucide-sun'"
            icon
            @click="toggleTheme"
          />

          <!-- User Menu -->
          <NDropdownMenu>
            <NButton btn="solid-gray" size="sm" label="i-lucide-user" icon />

            <template #content>
              <NDropdownMenuGroup>
                <NDropdownMenuItem @click="navigateTo('/settings')">
                  <div class="i-lucide-settings w-4 h-4" />
                  <span>设置</span>
                </NDropdownMenuItem>
                <NDropdownMenuSeparator />
                <NDropdownMenuItem @click="handleLogout">
                  <div class="i-lucide-log-out w-4 h-4" />
                  <span>退出登录</span>
                </NDropdownMenuItem>
              </NDropdownMenuGroup>
            </template>
          </NDropdownMenu>
        </div>
      </div>
    </header>

    <!-- Main Content -->
    <main class="container mx-auto px-4 py-6">
      <slot />
    </main>
  </div>
</template>
