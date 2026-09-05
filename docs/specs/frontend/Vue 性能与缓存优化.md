# Vue 3 性能与缓存优化

> 目标：通过缓存、预取、乐观更新、状态保留和渲染优化，让 Web 应用获得接近原生应用的交互体验。
> **相关**: [[Vue 最佳实践]]

## 核心原则

优化"体感性能"比单纯优化 benchmark 更重要。

用户真正能感知到的是：

- 点击后是否立即响应
- 页面切换是否需要等待
- 返回页面后状态是否还在
- 是否频繁出现 Loading
- 列表滚动是否掉帧
- 图片加载是否导致页面抖动

整体思路：

```text
用户操作
→ UI 立即响应
→ 内存缓存
→ 持久化缓存
→ 后台请求服务器
→ 必要时更新 UI
```

核心原则：

> 优先展示已有数据，再在后台判断它是否需要更新。

---

## 1. Server State 与 Client State 分离

不要把所有 API 数据都塞进 Pinia。

推荐：

```text
TanStack Query
→ Server State

Pinia
→ Client State
```

### Pinia

适合：

- Theme
- Sidebar
- 当前 Tab
- 本地配置
- UI 状态
- 用户交互状态

### TanStack Query

适合：

- 用户信息
- 文章
- 商品
- Feed
- 分页数据
- API 查询结果

例如：

```ts
const query = useQuery({
  queryKey: ['user', id],
  queryFn: () => api.getUser(id),

  staleTime: 5 * 60 * 1000,
  gcTime: 30 * 60 * 1000,
})
```

其中：

- `staleTime`：多久以内认为数据仍然新鲜
- `gcTime`：没有组件使用以后，缓存继续保留多久

参考：

| 数据 | staleTime |
| --- | --- |
| 实时状态 | 0 ~ 10 秒 |
| 首页 Feed | 30 ~ 60 秒 |
| 文章列表 | 1 ~ 5 分钟 |
| 用户资料 | 5 ~ 30 分钟 |
| 配置 | 30 分钟 ~ 数小时 |
| Metadata | `Infinity` |

推荐模式：

```text
缓存
→ 立即显示
→ 后台重新验证
→ 有变化再更新 UI
```

而不是：

```text
进入页面
→ Loading
→ API
→ 等待
→ 显示内容
```

---

## 2. Optimistic Update

用户主动操作时，不要默认等服务器响应以后才更新 UI。

例如收藏：

```ts
async function favorite() {
  const previous = item.favorite

  item.favorite = true

  try {
    await api.favorite(id)
  } catch {
    item.favorite = previous
  }
}
```

流程：

```text
点击
→ UI 立即变化
→ 后台 API
→ 成功：保持
→ 失败：回滚
```

适合：

- 点赞
- 收藏
- Toggle
- 标记已读
- 删除
- 排序
- 简单编辑

这是提升"原生感"收益最大的优化之一。

---

## 3. KeepAlive 页面缓存

Vue 可以通过：

```vue
<RouterView v-slot="{ Component }">
  <KeepAlive :max="10">
    <component :is="Component" />
  </KeepAlive>
</RouterView>
```

缓存组件实例。

适合：

- 首页
- 搜索页
- 列表页
- Dashboard
- 高频返回页面

例如：

```text
文章列表
→ 滚动到第 500 条
→ 打开详情
→ 返回
```

可以保留：

- Scroll Position
- 筛选条件
- Tab
- 表单状态
- 组件内部状态

不要无脑缓存所有路由。

通常不需要长期缓存：

- 一次性页面
- 大量详情页
- 编辑页面
- 很少返回的页面

---

## 4. Prefetch

缓存解决：

> 第二次访问快。

Prefetch 解决：

> 第一次访问也快。

例如：

```ts
function prefetchArticle(id: string) {
  queryClient.prefetchQuery({
    queryKey: ['article', id],
    queryFn: () => api.getArticle(id),
  })
}
```

桌面：

```vue
<ArticleCard
  @pointerenter="prefetchArticle(article.id)"
/>
```

移动端：

```vue
<ArticleCard
  @pointerdown="prefetchArticle(article.id)"
/>
```

流程：

```text
pointerdown
→ 请求开始
→ click
→ Router 跳转
→ 数据可能已经进入缓存
```

还可以利用用户行为预测：

```text
进入首页
→ 空闲时预加载 Search

进入文章列表
→ Prefetch 前几篇文章

进入商品详情
→ Prefetch 推荐商品
```

---

## 5. IndexedDB 持久化缓存

TanStack Query 默认主要是内存缓存。

如果希望应用重新启动以后也能马上看到上次的数据，可以把 Query Cache 持久化到 IndexedDB。

第一次：

```text
API
→ TanStack Query
→ UI
→ IndexedDB
```

下一次启动：

```text
IndexedDB
→ 恢复 Query Cache
→ UI 立即显示
→ 后台请求 API
→ 更新最新数据
```

将：

```text
启动
→ Loading
→ API
→ UI
```

变成：

```text
启动
→ UI
→ Background Sync
```

特别适合：

- PWA
- Wails
- Tauri
- Electron
- 移动 Web App

---

## 6. 大列表 Virtualization

不要：

```text
10000 条数据
=
10000 个 DOM
```

可以使用：

```text
@tanstack/vue-virtual
```

实现：

```text
数据：10000
DOM：约 20 ~ 50
```

只渲染当前 Viewport 附近的元素。

适合：

- Feed
- Chat
- Log
- Table
- 文件列表
- 无限滚动

---

## 7. Vue 响应式优化

大型 API Response 如果通常整体替换，可以考虑：

```ts
const data = shallowRef<LargeData>()
```

而不是让所有嵌套对象都进入深层响应式。

更新：

```ts
data.value = newData
```

适合：

- 大型不可变对象
- 大型列表
- 服务端完整 Response
- 很少修改深层属性的数据

---

## 8. v-memo

大型列表或昂贵组件可以考虑：

```vue
<div
  v-for="item in items"
  :key="item.id"
  v-memo="[item.id, item.selected]"
>
```

但不要一开始就大量使用。

推荐顺序：

1. Virtual List
2. 减少状态影响范围
3. 稳定 Props
4. `shallowRef`
5. `v-memo`

---

## 9. Code Splitting

路由级拆分：

```ts
const routes = [
  {
    path: '/settings',
    component: () => import('./pages/Settings.vue'),
  },
]
```

推荐按照：

- 页面
- 大型模块
- Editor
- Admin
- 重型第三方库

拆分。

避免两个极端：

```text
整个 App 一个 Bundle
```

或者：

```text
每个小组件一个 Chunk
```

后者容易造成大量网络 Waterfall。

---

## 10. HTTP / 静态资源缓存

Vite 构建后的资源通常带 Hash：

```text
app-a82c431.js
vendor-813cd23.js
style-f129aac.css
```

适合长期缓存：

```http
Cache-Control: public, max-age=31536000, immutable
```

而：

```text
index.html
```

应避免长期强缓存，例如：

```http
Cache-Control: no-cache
```

推荐：

```text
index.html
→ no-cache

/assets/*
→ public, max-age=31536000, immutable
```

---

## 11. 图片优化

图片往往比 Vue Render 更容易产生明显卡顿。

至少做到：

- 固定 `width` / `height`
- Lazy Loading
- WebP / AVIF
- Thumbnail → 高清图
- 避免 Layout Shift
- 合理利用浏览器缓存

例如：

```vue
<img
  :src="image"
  width="320"
  height="180"
  loading="lazy"
/>
```

---

## 12. 动画优化

优先动画：

```css
transform
opacity
```

谨慎动画：

```css
width
height
top
left
margin
```

避免：

```css
transition: all 300ms;
```

帧预算：

```text
60Hz
≈ 16.67ms / frame

120Hz
≈ 8.33ms / frame
```

一帧里包含：

```text
JavaScript
+
Style
+
Layout
+
Paint
+
Composite
```

---

## 推荐架构

```text
Vue 3
│
├── Vue Router
│   ├── Lazy Route
│   └── KeepAlive
│
├── TanStack Query
│   ├── API Cache
│   ├── staleTime
│   ├── Prefetch
│   ├── Optimistic Update
│   └── Background Refetch
│
├── Pinia
│   └── Client / UI State
│
├── IndexedDB
│   └── Persistent Query Cache
│
├── TanStack Virtual
│   └── Large List
│
└── Vite
    ├── Code Splitting
    └── Immutable Assets
```

## 优化优先级

1. TanStack Query 缓存
2. Optimistic Update
3. KeepAlive
4. Prefetch
5. IndexedDB 持久缓存
6. Virtual List
7. 图片优化
8. Code Splitting
9. 响应式粒度优化
10. v-memo

## 最终目标

不要：

- 等待网络
- 重复请求
- 重复初始化页面
- 渲染不可见内容
- 制造不必要的 Layout
- 频繁给用户展示 Loading

核心思想：

> 让已有内容立即出现，让网络请求尽量发生在用户感知之外。
