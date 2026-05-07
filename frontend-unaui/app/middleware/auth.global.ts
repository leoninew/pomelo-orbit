export default defineNuxtRouteMiddleware((to) => {
  // 只在客户端检查认证状态
  if (import.meta.server) {
    return
  }

  // 公开页面列表
  const publicPages = ['/login']

  // 如果是公开页面，直接放行
  if (publicPages.includes(to.path)) {
    return
  }

  // 检查 localStorage 中的 token
  const token = typeof window !== 'undefined' ? localStorage.getItem('token') : null

  // 如果未认证，跳转到登录页
  if (!token) {
    return navigateTo('/login')
  }
})
