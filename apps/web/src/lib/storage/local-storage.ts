// 浏览器 localStorage 薄适配（前端应用架构规范 §4.7）：集中封装访问与异常处理。
// 隐私模式、存储被禁用或配额失败时，调用方只失去持久化、功能不受影响
// （设计系统方案 §11.3「持久化失败时仍能使用」）；数据的版本与迁移策略由各 Store 自持。

function nativeStorage(): Storage | null {
  try {
    return globalThis.localStorage ?? null
  } catch {
    // 部分隐私模式下访问 localStorage 属性本身就抛 SecurityError。
    return null
  }
}

export const localStorageAdapter = {
  get(key: string): string | null {
    try {
      return nativeStorage()?.getItem(key) ?? null
    } catch {
      return null
    }
  },
  set(key: string, value: string): void {
    try {
      nativeStorage()?.setItem(key, value)
    } catch {
      // 写入失败（配额 / 禁用）：放弃持久化。
    }
  },
  remove(key: string): void {
    try {
      nativeStorage()?.removeItem(key)
    } catch {
      // 同写入失败。
    }
  },
}
