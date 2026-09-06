<script setup lang="ts">
import { computed } from 'vue'

// 基础按钮（设计系统方案 §8.1）：变体配方、pending 态与焦点环见 §8.1，类名保持完整字符串供
// Tailwind 静态发现（§12）。Pending 用原生 disabled 阻止重复激活——仅 aria-disabled 或
// pointer-events-none 挡不住键盘触发；焦点环在 outline-hidden 之后必须在 focus-visible 下
// 恢复 outline-solid，否则整个配方渲染不出焦点环。
type ButtonVariant =
  | 'primary'
  | 'secondary'
  | 'tertiary'
  | 'outline'
  | 'ghost'
  | 'danger'
  | 'danger-soft'

interface ButtonProps {
  variant?: ButtonVariant
  /** 原生 type，默认 button，避免放在表单内时隐式提交。 */
  type?: 'button' | 'submit' | 'reset'
  disabled?: boolean
  /** 异步处理中：阻止重复激活、标记 aria-busy，并展示处理中指示。 */
  pending?: boolean
}

const props = withDefaults(defineProps<ButtonProps>(), {
  variant: 'primary',
  type: 'button',
  disabled: false,
  pending: false,
})

// Hover / Pressed 配方逐项对应 §8.1 变体表；enabled: 前缀让 Disabled 态不响应悬停与按压。
const variantClasses: Record<ButtonVariant, string> = {
  primary:
    'bg-primary text-primary-foreground enabled:hover:bg-primary-hover enabled:active:bg-primary-active',
  secondary:
    'bg-default text-primary-text enabled:hover:bg-default-hover enabled:active:bg-default-hover',
  tertiary:
    'bg-default text-default-foreground enabled:hover:bg-default-hover enabled:active:bg-default-hover',
  outline:
    'border border-border-strong bg-transparent text-foreground enabled:hover:bg-default enabled:active:bg-default',
  ghost: 'bg-transparent text-foreground enabled:hover:bg-default enabled:active:bg-default',
  danger:
    'bg-danger text-danger-foreground enabled:hover:bg-danger-hover enabled:active:bg-danger-hover',
  'danger-soft':
    'bg-danger-subtle text-danger-text enabled:hover:bg-danger-subtle-hover enabled:active:bg-danger-subtle-hover',
}

const isDisabled = computed(() => props.disabled || props.pending)
</script>

<template>
  <button
    :type="type"
    :disabled="isDisabled"
    :aria-busy="pending || undefined"
    class="inline-flex min-h-11 items-center justify-center gap-2 rounded-control px-4 py-2 text-sm font-medium transition-[background-color,scale] duration-150 ease-out motion-safe:enabled:active:scale-97 motion-reduce:transition-none outline-hidden focus-visible:outline-solid focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-ring disabled:cursor-not-allowed disabled:opacity-50"
    :class="variantClasses[variant]"
  >
    <svg
      v-if="pending"
      aria-hidden="true"
      viewBox="0 0 24 24"
      fill="none"
      class="size-4 shrink-0 motion-safe:animate-spin"
    >
      <circle cx="12" cy="12" r="9" stroke="currentColor" stroke-width="3" class="opacity-25" />
      <path
        d="M21 12a9 9 0 0 0-9-9"
        stroke="currentColor"
        stroke-width="3"
        stroke-linecap="round"
      />
    </svg>
    <slot />
  </button>
</template>
