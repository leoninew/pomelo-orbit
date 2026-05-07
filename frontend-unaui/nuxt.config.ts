export default defineNuxtConfig({
  modules: ['@una-ui/nuxt'],

  devtools: {
    enabled: true,

    timeline: {
      enabled: true
    }
  },

  colorMode: {
    preference: 'light',
    fallback: 'light'
  },

  app: {
    head: {
      title: 'Pomelo Orbit',
      meta: [
        { charset: 'utf-8' },
        { name: 'viewport', content: 'width=device-width, initial-scale=1' },
        { name: 'description', content: 'Pomelo Orbit - CI/CD 管理平台' }
      ],
      link: [{ rel: 'icon', type: 'image/x-icon', href: '/favicon.ico' }],
      htmlAttrs: {
        lang: 'zh-CN'
      }
    }
  },

  devServer: {
    port: 10004
  },

  compatibilityDate: '2024-04-03'
})
