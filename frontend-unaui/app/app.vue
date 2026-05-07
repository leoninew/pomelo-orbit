<script setup>
const colorMode = useColorMode()

// 设置默认为浅色主题
if (!colorMode.value) {
  colorMode.preference = 'light'
}

useHead({
  htmlAttrs: {
    lang: 'zh-CN'
  }
})
</script>

<template>
  <NuxtLayout>
    <NuxtPage />
  </NuxtLayout>
</template>
