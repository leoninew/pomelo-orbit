<script setup lang="ts">
import type { RepositoryListItem } from '~/types/ci/repository'
import { formatTime } from '~/utils/time'
import { useRepositoryApi } from '~/composables/api'

definePageMeta({
  layout: 'sidebar',
  title: 'CI 仓库'
})

const repositoryApi = useRepositoryApi()

const loading = ref(false)
const repositories = ref<RepositoryListItem[]>([])
const searchText = ref('')
const pagination = reactive({ current: 1, pageSize: 20, total: 0 })

const totalPages = computed(() => Math.ceil(pagination.total / pagination.pageSize))

async function fetchRepositories() {
  loading.value = true
  try {
    const res = await repositoryApi.list({
      page: pagination.current,
      per_page: pagination.pageSize,
      search: searchText.value || undefined
    })
    repositories.value = res.items
    pagination.total = res.total
  } catch (error) {
    console.error('获取仓库列表失败:', error)
  } finally {
    loading.value = false
  }
}

function handleSearch() {
  pagination.current = 1
  fetchRepositories()
}

function goPage(p: number) {
  pagination.current = p
  fetchRepositories()
}

onMounted(() => {
  fetchRepositories()
})
</script>

<template>
  <div class="space-y-4">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-semibold text-gray-900">代码仓库</h1>
      <div class="flex items-center gap-2">
        <NInput
          v-model="searchText"
          placeholder="搜索名称/地址"
          class="w-52"
          @keydown.enter="handleSearch"
        />
        <NButton label="搜索" btn="solid-gray" :loading="loading" @click="handleSearch" />
      </div>
    </div>

    <!-- Table -->
    <NCard>
      <div v-if="loading" class="flex justify-center py-12">
        <div class="i-lucide-loader-2 w-8 h-8 animate-spin text-primary" />
      </div>
      <div
        v-else-if="repositories.length === 0"
        class="flex flex-col items-center gap-2 py-12 text-gray-600"
      >
        <div class="i-lucide-inbox w-10 h-10" />
        <span class="text-sm">暂无数据</span>
      </div>
      <div v-else class="overflow-x-auto">
        <table class="w-full">
          <thead>
            <tr class="border-b border-gray-200">
              <th class="text-left py-3 px-4 text-sm font-medium text-gray-600">名称</th>
              <th class="text-left py-3 px-4 text-sm font-medium text-gray-600">编码</th>
              <th class="text-left py-3 px-4 text-sm font-medium text-gray-600">地址</th>
              <th class="text-left py-3 px-4 text-sm font-medium text-gray-600">Git 凭据</th>
              <th class="text-left py-3 px-4 text-sm font-medium text-gray-600">创建时间</th>
              <th class="text-left py-3 px-4 text-sm font-medium text-gray-600">操作</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="repo in repositories"
              :key="repo.id"
              class="border-b border-gray-100 hover:bg-gray-50"
            >
              <td class="py-3 px-4">
                <NuxtLink
                  :to="`/ci/repository/${repo.id}`"
                  class="text-primary hover:underline font-medium"
                >
                  {{ repo.name }}
                </NuxtLink>
              </td>
              <td class="py-3 px-4 text-gray-600">{{ repo.code }}</td>
              <td class="py-3 px-4 text-gray-600 max-w-xs truncate">{{ repo.repository_url }}</td>
              <td class="py-3 px-4 text-gray-600">
                <span v-if="repo.git_credential_id">已配置</span>
                <span v-else class="text-gray-400">—</span>
              </td>
              <td class="py-3 px-4 text-gray-600">{{ formatTime(repo.created_at) }}</td>
              <td class="py-3 px-4">
                <NuxtLink :to="`/ci/repository/${repo.id}`" class="text-primary hover:underline">
                  查看
                </NuxtLink>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div v-if="totalPages > 0" class="flex justify-end pt-4 border-t border-gray-200 mt-4">
        <div class="flex gap-1">
          <NButton
            v-for="p in totalPages"
            :key="p"
            :label="String(p)"
            :btn="p === pagination.current ? 'solid' : 'ghost'"
            size="sm"
            @click="goPage(p)"
          />
        </div>
      </div>
    </NCard>
  </div>
</template>
