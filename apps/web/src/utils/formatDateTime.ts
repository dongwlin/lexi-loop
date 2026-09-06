/**
 * 把服务端的 ISO 时间串格式化为「YYYY-MM-DD HH:mm」（review-flow §5 详情页的上次复习）。
 * MVP 面向中文用户，统一按东八区渲染，保证不同机器展示一致、单测确定；无效输入与空值
 * 返回 null，由调用方兜底展示「—」。
 */
const formatter = new Intl.DateTimeFormat('en-CA', {
  timeZone: 'Asia/Shanghai',
  year: 'numeric',
  month: '2-digit',
  day: '2-digit',
  hour: '2-digit',
  minute: '2-digit',
  hourCycle: 'h23',
})

export function formatDateTime(
  value: string | null | undefined,
): string | null {
  if (!value) return null
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return null

  const parts = formatter.formatToParts(date)
  const get = (type: Intl.DateTimeFormatPartTypes) =>
    parts.find((part) => part.type === type)?.value ?? ''
  return `${get('year')}-${get('month')}-${get('day')} ${get('hour')}:${get('minute')}`
}
