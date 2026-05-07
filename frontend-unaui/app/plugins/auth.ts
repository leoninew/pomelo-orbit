export default defineNuxtPlugin(() => {
  const { initAuth } = useAuth()
  initAuth()
})
