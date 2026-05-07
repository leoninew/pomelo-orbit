<script setup lang="ts">
definePageMeta({
  title: '登录',
  layout: false
})

const { login } = useAuth()
const router = useRouter()

const form = reactive({
  username: '',
  password: ''
})

const errors = reactive({
  username: '',
  password: ''
})

const loading = ref(false)
const showPassword = ref(false)
const errorMessage = ref('')

function validate() {
  errors.username = form.username.trim() ? '' : '请输入用户名'
  errors.password = form.password ? '' : '请输入密码'
  return !errors.username && !errors.password
}

async function handleLogin() {
  if (!validate()) {
    return
  }

  loading.value = true
  errorMessage.value = ''

  try {
    await login(form.username, form.password)
    router.push('/')
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '登录失败'
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div
    class="min-h-screen flex items-center justify-center bg-gradient-to-br from-blue-500 to-purple-600"
  >
    <NCard class="w-96">
      <div class="space-y-6">
        <h1 class="text-2xl font-bold text-center text-gray-900">Pomelo Orbit</h1>

        <form class="space-y-4" @submit.prevent="handleLogin">
          <!-- Username -->
          <div class="space-y-2">
            <label class="text-sm font-medium text-gray-700">用户名</label>
            <NInput
              v-model="form.username"
              type="text"
              placeholder="请输入用户名"
              :class="{ 'border-red-500': errors.username }"
              autocomplete="username"
            />
            <p v-if="errors.username" class="text-sm text-red-600">
              {{ errors.username }}
            </p>
          </div>

          <!-- Password -->
          <div class="space-y-2">
            <label class="text-sm font-medium text-gray-700">密码</label>
            <div class="relative">
              <NInput
                v-model="form.password"
                :type="showPassword ? 'text' : 'password'"
                placeholder="请输入密码"
                :class="{ 'border-red-500': errors.password }"
                autocomplete="current-password"
              />
              <button
                type="button"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-gray-500 hover:text-gray-700"
                @click="showPassword = !showPassword"
              >
                <div v-if="showPassword" class="i-lucide-eye-off w-5 h-5" />
                <div v-else class="i-lucide-eye w-5 h-5" />
              </button>
            </div>
            <p v-if="errors.password" class="text-sm text-red-600">
              {{ errors.password }}
            </p>
          </div>

          <!-- Error Message -->
          <div v-if="errorMessage" class="p-3 bg-red-50 border border-red-200 rounded-md">
            <p class="text-sm text-red-600">
              {{ errorMessage }}
            </p>
          </div>

          <!-- Submit Button -->
          <NButton
            type="submit"
            btn="solid"
            block
            :loading="loading"
            :disabled="loading"
            class="w-full"
          >
            <template v-if="!loading"> 登录 </template>
            <template v-else>
              <div class="i-lucide-loader-2 w-5 h-5 animate-spin" />
              登录中...
            </template>
          </NButton>
        </form>
      </div>
    </NCard>
  </div>
</template>
