<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ChevronLeft, ChevronRight } from 'lucide-vue-next'
import { Button } from '@/components/ui'
import { useWordsListQuery } from '@/features/words/api/queries'
import { buildPageItems } from '@/utils/buildPageItems'
import { formatMeanings } from '@/utils/formatMeanings'
import { parsePositiveInt } from '@/utils/parsePositiveInt'

// review-flow.md §4：生词库（搜索 / 分页 / 语义表格）。
// 可分享状态写入 URL（前端应用架构规范 §6.2）：page 与 search 的权威来源是路由
// query，输入框里未提交的值留在本地；默认值不写入 URL，保持分享链接干净。
// 掌握度 / 优先级列依赖服务端契约扩展（README 待办），落地前不展示（D011）。

const PAGE_SIZE = 20

const route = useRoute()
const router = useRouter()

const page = computed(() => parsePositiveInt(route.query.page, 1))
const search = computed(() =>
  typeof route.query.search === 'string' ? route.query.search : '',
)

// 浏览器后退 / 前进恢复 URL 时，把已提交的搜索词同步回输入框。
const searchInput = ref(search.value)
watch(search, (value) => {
  searchInput.value = value
})

const queryParams = computed(() => ({
  page: page.value,
  pageSize: PAGE_SIZE,
  search: search.value || undefined,
}))

const {
  data,
  isPending,
  isError,
  error,
  isFetching,
  refetch,
} = useWordsListQuery(queryParams)

const rows = computed(() =>
  (data.value?.list ?? []).map((item) => ({
    ...item,
    meaning: formatMeanings(item.effectiveReviewMeaning),
  })),
)
const pagination = computed(() => data.value?.pagination)
const summary = computed(() => {
  const value = pagination.value
  return value && value.total > 0
    ? { total: value.total, totalPages: value.totalPages }
    : null
})
const pageItems = computed(() =>
  buildPageItems(page.value, pagination.value?.totalPages ?? 0),
)

const errorMessage = computed(() =>
  error.value instanceof Error ? error.value.message : '生词库加载失败，请稍后重试',
)

// 页码与搜索变化推入历史，后退 / 前进可回退到之前的列表状态（§6.2）。
function pushQuery(next: { page: number; search: string }) {
  const query: Record<string, string> = {}
  if (next.page > 1) query.page = String(next.page)
  if (next.search) query.search = next.search
  void router.push({ query })
}

function handleSearch() {
  const trimmed = searchInput.value.trim()
  searchInput.value = trimmed
  pushQuery({ page: 1, search: trimmed })
}

function clearSearch() {
  searchInput.value = ''
  pushQuery({ page: 1, search: '' })
}

function goToPage(target: number) {
  pushQuery({ page: target, search: search.value })
}

</script>

<template>
  <section class="rounded-card bg-surface p-6 shadow-surface">
    <h1 class="text-lg font-semibold text-foreground">生词库</h1>

    <!-- 搜索：提交后才写入 URL，输入过程不触发请求 -->
    <form
      role="search"
      aria-label="搜索生词"
      class="mt-4"
      @submit.prevent="handleSearch"
    >
      <label for="word-search" class="block text-sm text-muted-foreground">
        搜索单词或释义
      </label>
      <div class="mt-2 flex gap-2">
        <input
          id="word-search"
          v-model="searchInput"
          name="search"
          type="search"
          placeholder="例如：ambiguous 或 模棱两可"
          class="min-h-11 w-full min-w-0 flex-1 rounded-field border border-field-border bg-field px-3 py-2 text-base text-foreground shadow-field placeholder:text-muted-foreground enabled:hover:bg-field-hover outline-hidden focus-visible:outline-solid focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
        />
        <Button type="submit" class="shrink-0">搜索</Button>
      </div>
      <p v-if="search" class="mt-2 text-sm text-muted-foreground">
        正在筛选「{{ search }}」的结果
        <button
          type="button"
          class="rounded-item text-primary-text underline-offset-2 hover:underline outline-hidden focus-visible:outline-solid focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
          @click="clearSearch"
        >
          清除搜索
        </button>
      </p>
    </form>

    <!-- 页面级 Error State：错误说明 + 重试（交互与可访问性规范 §9.3） -->
    <template v-if="isError">
      <p role="alert" class="mt-4 text-sm text-danger-text">
        {{ errorMessage }}
      </p>
      <Button variant="outline" class="mt-3" :pending="isFetching" @click="refetch()">
        重试
      </Button>
    </template>

    <template v-else>
      <!-- 状态播报区预先存在于 DOM，随阶段更新文本（交互与可访问性规范 §7.5） -->
      <p
        v-if="isPending"
        role="status"
        class="mt-4 text-sm text-muted-foreground"
      >
        正在加载生词…
      </p>
      <p
        v-else-if="summary"
        role="status"
        class="mt-4 text-sm text-muted-foreground"
      >
        共 {{ summary.total }} 个单词 · 第 {{ page }} / {{ summary.totalPages }} 页
      </p>

      <div class="mt-3" :aria-busy="isFetching || undefined">
        <!-- Skeleton 形状接近最终表格结构（交互与可访问性规范 §9.1），仅首次加载展示 -->
        <div
          v-if="isPending"
          aria-hidden="true"
          class="space-y-4 py-2"
        >
          <div v-for="index in 5" :key="index" class="flex items-center gap-4">
            <div
              class="h-5 w-24 rounded-field bg-surface-tertiary motion-safe:animate-pulse"
            />
            <div
              class="h-5 flex-1 rounded-field bg-surface-tertiary motion-safe:animate-pulse"
            />
            <div
              class="h-5 w-10 rounded-field bg-surface-tertiary motion-safe:animate-pulse"
            />
          </div>
        </div>

        <!-- 语义表格（review-flow §4；交互与可访问性规范 §7.6） -->
        <table
          v-else-if="rows.length > 0"
          class="w-full border-collapse text-sm"
        >
          <thead>
            <tr class="border-b border-border-strong text-left">
              <th
                scope="col"
                class="py-2.5 pr-3 font-medium text-muted-foreground"
              >
                单词
              </th>
              <th
                scope="col"
                class="px-3 py-2.5 font-medium text-muted-foreground"
              >
                释义
              </th>
              <th
                scope="col"
                class="whitespace-nowrap px-3 py-2.5 text-right font-medium text-muted-foreground"
              >
                遇到
              </th>
              <th
                scope="col"
                class="whitespace-nowrap px-3 py-2.5 text-right font-medium text-muted-foreground"
              >
                复习
              </th>
              <th
                scope="col"
                class="whitespace-nowrap px-3 py-2.5 text-right font-medium text-muted-foreground"
              >
                记得
              </th>
              <th
                scope="col"
                class="whitespace-nowrap py-2.5 pl-3 text-right font-medium text-muted-foreground"
              >
                忘记
              </th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="row in rows"
              :key="row.id"
              class="border-b border-border last:border-b-0"
            >
              <td class="py-3 pr-3 align-top">
                <!-- review-flow §4：点击单词打开详情页；真实链接保留中键 / 新标签页（§6.2） -->
                <RouterLink
                  :to="{ name: 'word-detail', params: { id: row.id } }"
                  class="rounded-item font-medium text-foreground underline-offset-2 transition-colors duration-150 ease-out hover:text-primary-text hover:underline outline-hidden focus-visible:outline-solid focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring"
                >
                  {{ row.word }}
                  <span
                    v-if="row.phonetic"
                    class="block text-xs font-normal text-muted-foreground"
                  >
                    {{ row.phonetic }}
                  </span>
                </RouterLink>
              </td>
              <td
                class="px-3 py-3 align-top"
                :class="row.meaning ? 'text-foreground' : 'text-muted-foreground'"
              >
                {{ row.meaning || '—' }}
              </td>
              <td class="px-3 py-3 text-right align-top tabular-nums text-muted-foreground">
                {{ row.encounterCount }}
              </td>
              <td class="px-3 py-3 text-right align-top tabular-nums text-muted-foreground">
                {{ row.reviewCount }}
              </td>
              <td class="px-3 py-3 text-right align-top tabular-nums text-muted-foreground">
                {{ row.rememberCount }}
              </td>
              <td class="py-3 pl-3 text-right align-top tabular-nums text-muted-foreground">
                {{ row.forgetCount }}
              </td>
            </tr>
          </tbody>
        </table>

        <!-- Empty：区分「词库为空」与「搜索无结果」（交互与可访问性规范 §3.5 统一模式） -->
        <div
          v-else
          class="rounded-field border border-dashed border-border px-6 py-10 text-center"
        >
          <p class="text-sm text-muted-foreground">
            <template v-if="search">
              没有找到与「{{ search }}」匹配的单词，试试其他关键词。
            </template>
            <template v-else>
              词库还是空的，先把阅读中遇到的生词粘贴进来吧。
            </template>
          </p>
          <div class="mt-4 flex justify-center gap-3">
            <Button v-if="search" variant="outline" @click="clearSearch">
              清除搜索
            </Button>
            <RouterLink
              v-else
              to="/import"
              class="inline-flex min-h-11 items-center justify-center rounded-control bg-primary px-4 py-2 text-sm font-medium text-primary-foreground transition-[background-color] duration-150 ease-out hover:bg-primary-hover"
            >
              导入生词
            </RouterLink>
          </div>
        </div>
      </div>

      <!-- 分页控件：Navigation Landmark + 当前页 aria-current（交互与可访问性规范 §7.6） -->
      <nav
        v-if="summary && summary.totalPages > 1"
        aria-label="生词库分页"
        class="mt-4 flex flex-wrap items-center justify-center gap-1"
      >
        <Button
          variant="ghost"
          :disabled="page <= 1"
          aria-label="上一页"
          @click="goToPage(page - 1)"
        >
          <ChevronLeft class="size-4 shrink-0" aria-hidden="true" />
        </Button>
        <template
          v-for="(item, index) in pageItems"
          :key="item.kind === 'page' ? `p${item.page}` : `e${index}`"
        >
          <span
            v-if="item.kind === 'ellipsis'"
            aria-hidden="true"
            class="px-1 text-muted-foreground"
          >
            …
          </span>
          <Button
            v-else
            :variant="item.page === page ? 'secondary' : 'ghost'"
            :aria-current="item.page === page ? 'page' : undefined"
            @click="goToPage(item.page)"
          >
            {{ item.page }}
          </Button>
        </template>
        <Button
          variant="ghost"
          :disabled="page >= summary.totalPages"
          aria-label="下一页"
          @click="goToPage(page + 1)"
        >
          <ChevronRight class="size-4 shrink-0" aria-hidden="true" />
        </Button>
      </nav>
    </template>
  </section>
</template>
