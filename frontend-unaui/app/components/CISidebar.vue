<script setup lang="ts">
const navItems = [
  {
    title: 'CI 管理',
    icon: 'i-lucide-folder-git-2',
    items: [
      { title: '仓库列表', url: '/ci/repository' },
      { title: '构建历史', url: '/ci/builds' },
      { title: '构建配置', url: '/ci/config' }
    ]
  },
  {
    title: 'CD 管理',
    icon: 'i-lucide-rocket',
    items: [
      { title: '应用列表', url: '/cd/apps' },
      { title: '部署历史', url: '/cd/deployments' },
      { title: '环境配置', url: '/cd/environments' }
    ]
  }
]
</script>

<template>
  <NSidebar collapsible="icon" sidebar="sidebar" sheet="left" rail>
    <template #header>
      <div class="p-4">
        <p class="text-sm font-medium text-gray-900">项目管理</p>
      </div>
    </template>

    <template #content>
      <NSidebarGroup v-for="nav in navItems" :key="nav.title">
        <NSidebarGroupLabel>{{ nav.title }}</NSidebarGroupLabel>
        <NSidebarGroupContent>
          <NSidebarMenu>
            <NSidebarMenuItem v-for="item in nav.items" :key="item.title">
              <NSidebarMenuButton :to="item.url">
                {{ item.title }}
              </NSidebarMenuButton>
            </NSidebarMenuItem>
          </NSidebarMenu>
        </NSidebarGroupContent>
      </NSidebarGroup>
    </template>
  </NSidebar>
</template>
