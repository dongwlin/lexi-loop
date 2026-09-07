<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { AlertCircle, CheckCircle2, LoaderCircle, X } from 'lucide-vue-next'
import { Popover } from '@/components/ui'
import { useDictImportStatusQuery } from '../api/queries'

// 词典导入全局指示（issue #3，docs/api/meta.md §3）：AppShell 顶栏右上角
// 常驻徽标（变体 A · 紧凑徽标），轮询 Meta 进度端点并按状态收敛。
// - checking / importing：显示轻量检查 / 「词典导入中 N%」，点击展开 popover
//   详情（progressbar + 行数统计）；
// - completed：徽标消失，弹一次性「词典已就绪」toast（约 4.5s 自动消失）；
// - failed：常驻错误徽标 + 详情指引（重启容器自动续传）；
// - idle：不渲染任何内容。轮询收敛见 features/meta/api/queries.ts。
const { data } = useDictImportStatusQuery()

const status = computed(() => data.value ?? null)
const state = computed(() => status.value?.state ?? 'idle')
const badgeVisible = computed(
  () =>
    state.value === 'checking' ||
    state.value === 'importing' ||
    state.value === 'failed',
)

const numberFormat = new Intl.NumberFormat()
const rowsTotal = computed(() => status.value?.rowsTotal ?? 0)
const rowsProcessed = computed(() => status.value?.rowsProcessed ?? 0)
const entriesWritten = computed(() => status.value?.entriesWritten ?? 0)
// 进度百分比取 rowsProcessed / rowsTotal（importing 时有值；checking /
// rowsTotal 未就绪时为 null，徽标退化为纯状态短语）。四舍五入与原型
// 展示一致，封顶 100%。
const percent = computed(() => {
  const s = status.value
  if (!s || s.state !== 'importing' || s.rowsTotal <= 0) return null
  return Math.min(100, Math.round((s.rowsProcessed / s.rowsTotal) * 100))
})

const badgeLabel = computed(() => {
  if (state.value === 'failed') return '词典导入失败，查看详情'
  if (state.value === 'checking') return '词典数据检查中，查看详情'
  return `词典导入进度 ${percent.value ?? 0}%，查看详情`
})

// 「词典已就绪」toast 只在状态迁移到 completed 时演示一次（首次挂载即
// completed 不打扰——页面晚于导入完成加载属于常态）。
const READY_TOAST_DURATION_MS = 4500
const readyToastVisible = ref(false)
let readyToastTimer: number | undefined

watch(state, (next, prev) => {
  if (next === 'completed' && (prev === 'checking' || prev === 'importing')) {
    readyToastVisible.value = true
    window.clearTimeout(readyToastTimer)
    readyToastTimer = window.setTimeout(hideReadyToast, READY_TOAST_DURATION_MS)
  }
})

function hideReadyToast(): void {
  readyToastVisible.value = false
  window.clearTimeout(readyToastTimer)
}

onBeforeUnmount(() => window.clearTimeout(readyToastTimer))
</script>

<template>
  <Popover v-if="badgeVisible">
    <template #trigger>
      <button
        type="button"
        :aria-label="badgeLabel"
        class="inline-flex size-11 items-center justify-center rounded-item bg-transparent outline-hidden transition-colors duration-150 ease-out focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring focus-visible:outline-solid"
      >
        <span
          class="inline-flex items-center gap-1 rounded-item px-2 py-1 text-xs font-medium"
          :class="
            state === 'failed'
              ? 'bg-danger-subtle text-danger-text'
              : state === 'checking'
                ? 'bg-surface-hover text-muted-foreground'
                : 'bg-primary-subtle text-primary-text'
          "
        >
          <LoaderCircle
            v-if="state !== 'failed'"
            class="size-3.5 shrink-0 motion-safe:animate-spin"
            aria-hidden="true"
          />
          <AlertCircle v-else class="size-3.5 shrink-0" aria-hidden="true" />
          <span v-if="percent !== null" class="tabular-nums">{{ percent }}%</span>
          <span v-else>{{ state === 'failed' ? '导入失败' : '检查中' }}</span>
        </span>
      </button>
    </template>

    <div class="w-72 space-y-3">
      <div class="flex items-center gap-2 text-sm font-medium">
        <LoaderCircle
          v-if="state === 'checking' || state === 'importing'"
          class="size-4 shrink-0 text-primary-text motion-safe:animate-spin"
          aria-hidden="true"
        />
        <AlertCircle
          v-else-if="state === 'failed'"
          class="size-4 shrink-0 text-danger-text"
          aria-hidden="true"
        />
        <span>{{ state === 'failed' ? '词典导入失败' : '词典导入中' }}</span>
      </div>

      <div
        v-if="state === 'importing'"
        role="progressbar"
        aria-label="词典导入进度"
        aria-valuemin="0"
        :aria-valuemax="rowsTotal"
        :aria-valuenow="rowsProcessed"
        :aria-valuetext="`${percent ?? 0}%`"
        class="h-1.5 overflow-hidden rounded-full bg-primary/15"
      >
        <div
          class="h-full rounded-full bg-primary transition-[width] duration-500 ease-out"
          :style="{ width: `${percent ?? 0}%` }"
        />
      </div>

      <dl
        class="space-y-1 text-xs text-muted-foreground [&>div]:flex [&>div]:items-center [&>div]:justify-between [&>dt]:normal-case [&>dd]:tabular-nums [&>dd]:font-medium [&>dd]:text-foreground"
      >
        <div>
          <dt>已处理行数</dt>
          <dd>{{ numberFormat.format(rowsProcessed) }}</dd>
        </div>
        <div v-if="rowsTotal > 0">
          <dt>总行数</dt>
          <dd>{{ numberFormat.format(rowsTotal) }}</dd>
        </div>
        <div>
          <dt>已写入词条</dt>
          <dd>{{ numberFormat.format(entriesWritten) }}</dd>
        </div>
      </dl>

      <p v-if="state === 'failed'" class="text-xs text-danger-text">
        导入失败不影响生词流程；重启容器将自动续传完成导入。
      </p>
      <p
        v-if="state === 'failed' && status?.error"
        class="break-all font-mono text-xs text-muted-foreground"
      >
        {{ status.error }}
      </p>
    </div>
  </Popover>

  <!-- 一次性就绪提示：role=status 即 polite live region（前端交互与可访问性规范 §7.5） -->
  <div
    v-if="readyToastVisible"
    role="status"
    class="fixed right-4 top-16 z-50 flex w-80 max-w-[calc(100vw-2rem)] items-start gap-3 rounded-item bg-overlay p-4 shadow-overlay"
  >
    <CheckCircle2 class="size-5 shrink-0 text-success" aria-hidden="true" />
    <div class="min-w-0 flex-1">
      <p class="text-sm font-medium text-foreground">词典已就绪</p>
      <p class="mt-0.5 text-sm text-muted-foreground">
        词典导入完成，现在可以正常添加生词了。
      </p>
    </div>
    <button
      type="button"
      aria-label="关闭通知"
      class="-me-1 -mt-1 inline-flex size-8 items-center justify-center rounded-item text-muted-foreground outline-hidden transition-colors duration-150 ease-out hover:bg-surface-hover hover:text-foreground focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring focus-visible:outline-solid"
      @click="hideReadyToast"
    >
      <X class="size-4 shrink-0" aria-hidden="true" />
    </button>
  </div>
</template>
