/**
 * 把 URL query 参数解析为正整数。vue-router 的 query 值是 `string | string[] | null`，
 * 取数组首项；缺失、非数字、小数与小于 1 一律回退到默认值，调用方无需再判空。
 */
export function parsePositiveInt(raw: unknown, fallback: number): number {
  const value = Array.isArray(raw) ? raw[0] : raw
  const parsed = typeof value === 'string' ? Number(value) : Number.NaN
  return Number.isSafeInteger(parsed) && parsed >= 1 ? parsed : fallback
}
