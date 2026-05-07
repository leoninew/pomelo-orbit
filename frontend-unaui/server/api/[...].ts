export default defineEventHandler(async (event) => {
  const path = event.path
  const target = 'http://127.0.0.1:10001'

  // 代理 /api/** 和 /hooks/** 请求到后端
  if (path.startsWith('/api/') || path.startsWith('/hooks/')) {
    return proxyRequest(event, `${target}${path}`)
  }
})
