// meta Feature 的 Query Key 工厂（前端应用架构规范 §8.1）：集中定义缓存键层级，
// 查询与失效共用，避免页面散落字符串键。层级约定：['meta'] → ['meta', 'version']。
export const metaKeys = {
  all: ['meta'] as const,
  version: () => [...metaKeys.all, 'version'] as const,
}
