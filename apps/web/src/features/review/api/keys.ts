// review Feature 的 Query Key 工厂（前端应用架构规范 §8.1）。
// 层级约定：['review'] → ['review', 'sessions', sessionId]。
export const reviewKeys = {
  all: ['review'] as const,
  sessions: () => [...reviewKeys.all, 'sessions'] as const,
  session: (id: string) => [...reviewKeys.sessions(), id] as const,
}
