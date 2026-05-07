<script setup lang="ts">
import type { Repository } from '~/types/ci/repository'
import { useRepositoryApi } from '~/composables/api'

definePageMeta({
  layout: 'sidebar',
  title: '仓库详情'
})

const route = useRoute()
const repositoryId = route.params.id as string

const repositoryApi = useRepositoryApi()

const loading = ref(false)
const repository = ref<Repository>()

async function fetchRepository() {
  loading.value = true
  try {
    repository.value = await repositoryApi.get(repositoryId)
  } catch (error) {
    console.error('获取仓库信息失败:', error)
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await fetchRepository()
})
</script>

<template>
  <div class="space-y-6">
    <!-- Header -->
    <div class="flex items-center justify-between">
      <h1 class="text-xl font-semibold text-gray-900">
        {{ repository?.name ?? '仓库详情' }}
      </h1>
      <NButton
        label="返回"
        btn="ghost"
        leading="i-lucide-arrow-left"
        @click="$router.push('/ci/repository')"
      />
    </div>

    <!-- Basic Info -->
    <NCard title="基本信息">
      <div v-if="loading" class="grid grid-cols-2 gap-4">
        <div v-for="i in 6" :key="i" class="space-y-1">
          <div class="h-4 w-20 bg-gray-200 rounded animate-pulse" />
          <div class="h-4 w-32 bg-gray-100 rounded animate-pulse" />
        </div>
      </div>
      <dl v-else-if="repository" class="grid grid-cols-1 sm:grid-cols-2 gap-x-8 gap-y-4 text-sm">
        <div class="space-y-1">
          <dt class="text-gray-600">名称</dt>
          <dd class="text-gray-900">{{ repository.name }}</dd>
        </div>
        <div class="space-y-1">
          <dt class="text-gray-600">编码</dt>
          <dd class="text-gray-900">{{ repository.code }}</dd>
        </div>
        <div class="space-y-1">
          <dt class="text-gray-600">地址</dt>
          <dd class="text-gray-900 truncate">{{ repository.repository_url }}</dd>
        </div>
        <div class="space-y-1">
          <dt class="text-gray-600">默认分支</dt>
          <dd class="text-gray-600">{{ repository.default_branch }}</dd>
        </div>
        <div class="space-y-1">
          <dt class="text-gray-600">Git 凭据</dt>
          <dd class="text-gray-900">
            <span v-if="repository.git_credential_id">{{ repository.git_credential_name }}</span>
            <span v-else class="text-gray-400">未配置</span>
          </dd>
        </div>
        <div class="space-y-1">
          <dt class="text-gray-600">流水线记录</dt>
          <dd>
            <NuxtLink
              :to="`/ci/run?repository_id=${repository.id}`"
              class="text-primary hover:underline"
            >
              查看所有记录
            </NuxtLink>
          </dd>
        </div>
      </dl>
    </NCard>
  </div>
</template>
