import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'

dayjs.extend(relativeTime)
dayjs.locale('zh-cn')

export const formatDateTime = (value?: string) =>
  value ? dayjs(value).format('YYYY.MM.DD HH:mm') : '—'

export const formatTime = (value?: string) =>
  value ? dayjs(value).format('HH:mm:ss') : '—'

export const formatRelative = (value?: string) => {
  if (!value) return '—'
  const diffSeconds = dayjs().diff(dayjs(value), 'second')
  if (diffSeconds < 60) return '刚刚'
  return dayjs(value).fromNow()
}

export const formatCount = (value: number) => {
  if (value >= 10000) return `${(value / 10000).toFixed(1)}w`
  if (value >= 1000) return `${(value / 1000).toFixed(1)}k`
  return String(value)
}

export const minutesAgo = (minutes: number) =>
  dayjs().subtract(minutes, 'minute').toISOString()

export const hoursAgo = (hours: number) =>
  dayjs().subtract(hours, 'hour').toISOString()

export const daysAgo = (days: number) =>
  dayjs().subtract(days, 'day').toISOString()
