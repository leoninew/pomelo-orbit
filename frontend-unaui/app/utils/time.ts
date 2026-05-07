export function formatTime(dateStr: string): string {
  const date = new Date(dateStr)
  const now = new Date()
  const diff = now.getTime() - date.getTime()

  const minutes = Math.floor(diff / 60000)
  const hours = Math.floor(diff / 3600000)
  const days = Math.floor(diff / 86400000)

  if (minutes < 1) return '刚刚'
  if (minutes < 60) return `${minutes} 分钟前`
  if (hours < 24) return `${hours} 小时前`
  if (days < 7) return `${days} 天前`

  return date.toLocaleDateString('zh-CN')
}

export function getTodayStart(): Date {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return today
}

export function getTodayEnd(): Date {
  const today = new Date()
  today.setHours(23, 59, 59, 999)
  return today
}
