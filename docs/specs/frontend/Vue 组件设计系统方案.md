# Vue 组件设计系统方案

> [!summary] 当前方案
>
> **Reka UI 管行为，Tailwind CSS v4 管样式，Radix Colors 提供色阶，Semantic Tokens 承载向 HeroUI v3 对齐的视觉语言。**
>
> - 视觉基准：HeroUI v3 默认组件，而不是官网营销页或 v2 示例
> - 中性色：Radix Gray；品牌方向：明亮蓝色，实心主色单独校准
> - 功能色：Green / Amber / Red
> - 形态：胶囊感按钮、12px 输入框圆角、24px 卡片与对话框圆角
> - 层次：浅灰画布、白色表面、柔和阴影；深色主题以表面明度差为主
> - 主题：Light / Dark / System

---

## 1. 对齐目标与边界

本方案用于 Vue 组件系统，与《[[前端技术栈]]》中的 React HeroUI v3 Profile 保持视觉接近，但不共用组件实现。

“向 HeroUI 对齐”包括颜色关系、表面层级、圆角、尺寸、字体、组件变体和交互反馈，不只是把主按钮改成蓝色。具体来说：不安装 React 组件，不复制 HeroUI 的 DOM、React Aria 状态属性或全部主题变量。

| 层 | 本方案的职责 |
| --- | --- |
| Reka UI | 复杂控件的状态、键盘、焦点、ARIA、Portal 与定位 |
| Tailwind CSS v4 | 布局、间距、响应式和状态样式表达 |
| Semantic Tokens | 产品的颜色、表面、圆角和阴影语义 |
| Radix Colors + 少量校准色 | 底层色阶、浅深主题与实心品牌色 |
| 项目 UI 组件 | 组合以上能力，提供稳定的 Variant、Size 与业务无关接口 |

HeroUI v3 参考核对日期为 **2026-09-03**。其默认主题和组件样式是视觉参考，不是运行时依赖。后续落地应锁定实际参考版本，避免跟随上游分支静默改变外观。[HeroUI Theming](https://heroui.com/docs/react/getting-started/theming)

---

## 2. 从原方案到 HeroUI 风格

| 维度 | 原方案 | 调整后的方向 |
| --- | --- | --- |
| 整体气质 | 冷色、偏工程工具、边框感较强 | 中性、柔和、圆润，蓝色突出关键操作 |
| 中性色 | Slate 的冷灰 | Gray 为主，减少大面积蓝灰染色 |
| 品牌色 | Indigo | 接近 HeroUI 的明亮蓝色，按对比度校准 |
| 浅色表面 | 背景与卡片直接使用相邻色阶 | 浅灰画布上的白色卡片，明确前后关系 |
| 深色表面 | 沿用浅色 Step 映射 | 独立调整画布、卡片、嵌套区域与弹层 |
| 按钮 | 8–12px 圆角 | 24px 圆角，常规高度下呈胶囊感 |
| 卡片、对话框 | 16px 圆角 | 24px 圆角，留出合理内边距 |
| 层级表达 | Border > Shadow | 表面明度差 + 柔和阴影，边框按用途出现 |
| 次级操作 | 缺少固定配方 | 区分中性填充、品牌文字、透明与描边变体 |
| 状态色 | 主要提供 Solid | 同时提供 Soft 底色与可读的状态文字 |
| 动效 | 未形成统一规则 | 轻微按压、短时过渡，不默认引入涟漪或弹簧动画 |

圆润不意味着所有区域都套圆角卡片；官网中的渐变大字、光晕和玻璃背景也不作为业务界面默认样式。表格、长列表和普通正文以信息密度与阅读顺序为先。

---

## 3. Token 分层与命名

依赖关系：

```text
Radix 色阶 / 品牌校准值
          ↓
Semantic Tokens
          ↓
Tailwind Utilities
          ↓
项目 UI 组件
```

组件只使用 `bg-surface`、`text-foreground`、`bg-primary` 等语义名称，不直接依赖 `gray-*`、`blue-*` 或主题中的具体色值。

### 3.1 与 HeroUI 的语义对应

保留已有的 `primary`、`muted-foreground` 和 `ring` 命名，不为了视觉对齐强制重命名所有组件。

| HeroUI v3 语义 | 本方案 | 用途 |
| --- | --- | --- |
| `background` | `background` | 页面画布 |
| `surface` | `surface` | 卡片等常驻表面 |
| `surface-secondary / tertiary` | 同名 | 嵌套或次级内容区域，不表示悬浮高度 |
| `overlay` | `overlay` | Menu、Popover、Dialog 的实体表面 |
| `backdrop` | `backdrop` | 弹层背后的半透明遮罩 |
| `default` | `default` | 中性填充控件 |
| `accent` | `primary` | 实心品牌操作 |
| `accent-soft` | `primary-subtle` | 轻量品牌提示、选中背景 |
| `accent-soft-foreground` | `primary-text` | 品牌文字，不直接使用实心底色作文字 |
| `muted` | `muted-foreground` | 次要文字 |
| `focus` | `ring` | 焦点指示 |
| `field-background` | `field` | 输入区域背景 |

HeroUI 明确区分常驻表面、弹层表面和遮罩。本方案沿用同样的角色区分。[HeroUI 默认主题变量](https://github.com/heroui-inc/heroui/blob/v3/packages/styles/themes/default/variables.css)

> [!important] Overlay 不再兼指遮罩
>
> `bg-overlay` 用于弹层内容，`bg-backdrop` 用于背景遮罩。已有实现如果用 `overlay` 表示黑色遮罩，迁移时必须检查使用位置，不能直接覆盖变量后继续沿用旧类名。

### 3.2 控制复杂度的方式

不限制 Token 数量，只保留实际需要的视觉角色：

- 颜色至少区分画布、表面、控件填充、文字、焦点和状态。
- `primary-subtle + primary-text` 与 `primary + primary-foreground` 是不同配方，不要求共用同一个颜色。
- 输入框需要 `field`，弹层需要 `overlay`；这些是跨组件角色，不是过早抽象。
- 暂不为每个 Button、Badge、Menu Item 建独立颜色变量。
- 页面通过组件 Variant 选择外观，不自行拼接新的状态规则。

---

## 4. 颜色系统

### 4.1 Radix 的角色

Radix 的 12 个 Step 提供参考起点：1–2 为背景、3–5 为交互填充、6–8 为边界、9–10 为实心颜色、11–12 为文字。这是用途建议，不要求所有主题都把 Card 固定映射到 Step 2，也不保证任意前景与背景组合满足项目对比度要求。[Radix 色阶用途](https://www.radix-ui.com/colors/docs/palette-composition/understanding-the-scale)

默认色板调整为：

| 角色 | 色板或来源 | 选择理由 |
| --- | --- | --- |
| Neutral | Gray | 让页面更中性，蓝色集中在品牌与交互位置 |
| Primary Solid | 校准后的品牌蓝 | 接近 HeroUI 的蓝色方向，同时照顾小字号白字 |
| Primary Soft / Text | Blue | 提供浅深主题中的柔和填充与品牌文字 |
| Success | Green | 比原来的 Jade 更接近明确的绿色成功语义 |
| Warning | Amber | 保留暖色警告，与蓝色品牌区分 |
| Danger | Red | 错误、风险与破坏性操作 |

这是一套 **HeroUI 风格的 Radix 适配主题**，不是逐像素复刻。Gray 与 HeroUI 的中性色不完全相同，Blue 也不等于 HeroUI 的 Accent。

### 4.2 品牌蓝必须校准

HeroUI v3 参考主色为 `oklch(0.6204 0.195 253.83)`。本方案保留其蓝色色相方向，但为常用小字号白字按钮降低亮度，采用以下项目值。参考色来自 [HeroUI 默认主题变量](https://github.com/heroui-inc/heroui/blob/v3/packages/styles/themes/default/variables.css)。

| Token | 项目值 |
| --- | --- |
| `primary` | `#0072e2` |
| `primary-hover` | `#0068d8` |
| `primary-active` | `#005fcd` |
| `primary-foreground` | `#ffffff` |

这些值是本方案的校准结果，不是 HeroUI 官方默认值。默认蓝底与白字的 sRGB 对比度约为 `4.66:1`，Hover 与 Active 更深；不得再整体降低按钮透明度作为 Hover，否则会改变文字的实际对比度。

小字号品牌文字使用 `primary-text`，不使用 `text-primary`。尤其在深色主题中，实心按钮的蓝色与文字需要的蓝色不能混为一谈。

### 4.3 Canonical Color Mapping

下表中的 Radix 变量在当前主题作用域内解析。相同变量名不代表浅深模式下的实际色值相同。

| Semantic Token | Light | Dark | 用途 |
| --- | --- | --- | --- |
| `background` | `#f5f5f5` | `#0b0b0d` | 接近 HeroUI 的浅灰 / 近黑画布 |
| `surface` | `#ffffff` | `gray-2` | 主卡片 |
| `surface-secondary` | `gray-3` | `gray-3` | 卡片内的分组 |
| `surface-tertiary` | `gray-4` | `gray-4` | 更明显的嵌套区域 |
| `overlay` | `#ffffff` | `gray-3` | 悬浮内容 |
| `surface-hover / active` | `gray-3 / gray-4` | 同名暗色阶 | 透明列表项的 Hover / Pressed |
| `foreground` | `gray-12` | `gray-12` | 正文 |
| `muted-foreground` | `gray-11` | `gray-11` | 次要文字 |
| `default / default-hover` | `gray-3 / gray-4` | 同名暗色阶 | 中性填充控件 |
| `default-foreground` | `foreground` | `foreground` | 中性控件文字 |
| `border / border-hover` | `gray-6 / gray-7` | 同名暗色阶 | 装饰边界与分隔线 |
| `border-strong` | `gray-10` | `gray-10` | 需要明确可辨识的控件边界 |
| `field / field-hover` | 白色 / `gray-2` | `gray-3 / gray-4` | 默认输入区域 |
| `field-border` | `border-strong` | `border-strong` | 显式描边输入框 |
| `primary / hover / active` | 上述品牌校准值 | 同 Light | 实心主操作 |
| `primary-foreground` | 白色 | 白色 | 实心主操作文字 |
| `primary-subtle / subtle-hover` | `blue-3 / blue-4` | 同名暗色阶 | 轻量品牌填充 |
| `primary-text` | `blue-11 / blue-12` 按 80:20 混合 | 同名暗色阶按相同比例混合 | 品牌文字与 Soft 变体文字 |
| `ring` | `blue-11` | `blue-11` | 与当前表面有清晰对比的焦点指示 |
| `backdrop` | 黑色 40% | 黑色 60% | 对话框背景遮罩 |

功能色分别定义 Solid、Solid Foreground、Subtle 与 Text：

| 状态 | Solid | Solid Foreground | Subtle | Text |
| --- | --- | --- | --- | --- |
| Success | `green-9` | 固定深色 `#111111` | `green-3` | `green-11 / green-12` 按 80:20 混合 |
| Warning | `amber-9` | 固定深色 `#111111` | `amber-3` | `amber-11 / amber-12` 按 80:20 混合 |
| Danger | `red-9` 与黑色按 85:15 混合 | 白色 | `red-3` | `red-11 / red-12` 按 80:20 混合 |

以上混合均使用 `color-mix(in srgb, ...)`。Text 向同色系 Step 12 混合，提升浅色填充与 Hover 背景上的可读性；不能假设 Step 11 放在任意 Soft 背景上都足够清晰。

边界的对比度分工也需要按实测对待：`border`（gray-6）对表面的对比度约 1.4:1（深色约 1.55:1），只能承担装饰性分隔；需要靠边界识别的控件描边必须使用 `border-strong`（gray-10），浅深主题下约 3.8:1，满足非文本对比要求（WCAG 1.4.11 的 3:1）。数值为 2026-09-06 按 sRGB 计算的实测结果。

Danger Solid 有意加深，以适配小字号白字；Hover 进一步加深。Success 与 Warning 的实心前景不随主题变成白色。普通状态标签优先使用 Subtle + Text，实心颜色保留给确实需要高强调的场景。

以上映射不等于组件已经通过可访问性验收。Soft 背景、正文、边界、焦点以及所有交互状态仍需在真实组合中验证，要求见《[[前端交互与可访问性规范]]》。

---

## 5. Tailwind CSS v4 主题实现

### 5.1 样式入口

应用入口只引入一次 `src/styles/main.css`：

```css
@import "tailwindcss";
@import "./theme.css";

@custom-variant dark (&:where(.dark, .dark *));

@layer base {
  body {
    background: var(--background);
    color: var(--foreground);
  }
}
```

Vue 侧不导入 `@heroui/styles`，避免同时维护 HeroUI 与自有组件的两套状态选择器。主题变量和 CSS-first 配置放在下面的 `theme.css` 中。

### 5.2 Primitive 与 Semantic Tokens

Radix 的 CSS 文件为 `:root / .light` 和 `.dark` 提供对应色阶，需要同时导入浅深两套。应用使用根元素上的 `.light` / `.dark` 切换，不设置 Radix CSS 不识别的 `data-theme`。[Radix CSS 用法](https://www.radix-ui.com/colors/docs/overview/usage)

浅色文件必须写在深色文件之前：浅色的 `:root` 与深色的 `.dark` 对同一个根元素同时匹配且优先级相同，覆盖关系完全由源顺序决定。

```css
@import "@radix-ui/colors/gray.css";
@import "@radix-ui/colors/gray-dark.css";
@import "@radix-ui/colors/blue.css";
@import "@radix-ui/colors/blue-dark.css";
@import "@radix-ui/colors/green.css";
@import "@radix-ui/colors/green-dark.css";
@import "@radix-ui/colors/amber.css";
@import "@radix-ui/colors/amber-dark.css";
@import "@radix-ui/colors/red.css";
@import "@radix-ui/colors/red-dark.css";

@layer base {
  :root,
  .light,
  .dark {
    --foreground: var(--gray-12);
    --muted-foreground: var(--gray-11);

    --surface-secondary: var(--gray-3);
    --surface-tertiary: var(--gray-4);
    --surface-hover: var(--gray-3);
    --surface-active: var(--gray-4);

    --default: var(--gray-3);
    --default-hover: var(--gray-4);
    --default-foreground: var(--foreground);

    --border: var(--gray-6);
    --border-hover: var(--gray-7);
    --border-strong: var(--gray-10);
    --field-border: var(--border-strong);

    --primary: #0072e2;
    --primary-hover: #0068d8;
    --primary-active: #005fcd;
    --primary-foreground: #ffffff;
    --primary-subtle: var(--blue-3);
    --primary-subtle-hover: var(--blue-4);
    --primary-text: color-mix(in srgb, var(--blue-11) 80%, var(--blue-12) 20%);

    --success: var(--green-9);
    --success-foreground: #111111;
    --success-subtle: var(--green-3);
    --success-text: color-mix(in srgb, var(--green-11) 80%, var(--green-12) 20%);

    --warning: var(--amber-9);
    --warning-foreground: #111111;
    --warning-subtle: var(--amber-3);
    --warning-text: color-mix(in srgb, var(--amber-11) 80%, var(--amber-12) 20%);

    --danger: color-mix(in srgb, var(--red-9) 85%, black 15%);
    --danger-hover: color-mix(in srgb, var(--red-9) 78%, black 22%);
    --danger-foreground: #ffffff;
    --danger-subtle: var(--red-3);
    --danger-subtle-hover: var(--red-4);
    --danger-text: color-mix(in srgb, var(--red-11) 80%, var(--red-12) 20%);

    --ring: var(--blue-11);
  }

  :root,
  .light {
    color-scheme: light;

    --background: #f5f5f5;
    --surface: #ffffff;
    --overlay: #ffffff;
    --field: #ffffff;
    --field-hover: var(--gray-2);
    --backdrop: rgb(0 0 0 / 40%);

    --surface-shadow: 0 1px 2px rgb(0 0 0 / 6%),
      0 2px 6px rgb(0 0 0 / 4%);
    --overlay-shadow: 0 4px 12px rgb(0 0 0 / 8%),
      0 12px 28px rgb(0 0 0 / 10%);
    --field-shadow: 0 1px 2px rgb(0 0 0 / 6%);
  }

  .dark {
    color-scheme: dark;

    --background: #0b0b0d;
    --surface: var(--gray-2);
    --overlay: var(--gray-3);
    --field: var(--gray-3);
    --field-hover: var(--gray-4);
    --backdrop: rgb(0 0 0 / 60%);

    --surface-shadow: 0 0 0 0 transparent;
    --overlay-shadow: inset 0 0 0 1px rgb(255 255 255 / 10%);
    --field-shadow: 0 0 0 0 transparent;
  }
}
```

语义别名在每个主题作用域重新声明，避免局部主题继承到已在父级解析好的颜色。当前默认是根级统一主题；局部主题与 Portal 的组合需要额外处理继承边界。

### 5.3 暴露给 Tailwind

引用其他 CSS 变量时使用 `@theme inline`，让 Utility 在使用位置读取当前主题语义变量。[Tailwind Theme Variables](https://tailwindcss.com/docs/theme)

以下配置接在同一份 `theme.css` 后：

```css
@theme inline {
  --color-background: var(--background);
  --color-surface: var(--surface);
  --color-surface-secondary: var(--surface-secondary);
  --color-surface-tertiary: var(--surface-tertiary);
  --color-surface-hover: var(--surface-hover);
  --color-surface-active: var(--surface-active);
  --color-overlay: var(--overlay);
  --color-backdrop: var(--backdrop);

  --color-foreground: var(--foreground);
  --color-muted-foreground: var(--muted-foreground);

  --color-default: var(--default);
  --color-default-hover: var(--default-hover);
  --color-default-foreground: var(--default-foreground);

  --color-border: var(--border);
  --color-border-hover: var(--border-hover);
  --color-border-strong: var(--border-strong);
  --color-field: var(--field);
  --color-field-hover: var(--field-hover);
  --color-field-border: var(--field-border);

  --color-primary: var(--primary);
  --color-primary-hover: var(--primary-hover);
  --color-primary-active: var(--primary-active);
  --color-primary-foreground: var(--primary-foreground);
  --color-primary-subtle: var(--primary-subtle);
  --color-primary-subtle-hover: var(--primary-subtle-hover);
  --color-primary-text: var(--primary-text);

  --color-success: var(--success);
  --color-success-foreground: var(--success-foreground);
  --color-success-subtle: var(--success-subtle);
  --color-success-text: var(--success-text);

  --color-warning: var(--warning);
  --color-warning-foreground: var(--warning-foreground);
  --color-warning-subtle: var(--warning-subtle);
  --color-warning-text: var(--warning-text);

  --color-danger: var(--danger);
  --color-danger-hover: var(--danger-hover);
  --color-danger-foreground: var(--danger-foreground);
  --color-danger-subtle: var(--danger-subtle);
  --color-danger-subtle-hover: var(--danger-subtle-hover);
  --color-danger-text: var(--danger-text);

  --color-ring: var(--ring);

  --shadow-surface: var(--surface-shadow);
  --shadow-overlay: var(--overlay-shadow);
  --shadow-field: var(--field-shadow);
}

@theme {
  --radius-control: 1.5rem;
  --radius-field: 0.75rem;
  --radius-item: 0.5rem;
  --radius-popover: 1rem;
  --radius-card: 1.5rem;
  --radius-dialog: 1.5rem;
}
```

使用 `rounded-control / rounded-field / rounded-card` 等角色名，避免直接覆盖 Tailwind 的 `rounded-sm / md / lg`，导致现有布局中的圆角含义一起变化。颜色、圆角和阴影集中在主题层；Spacing 继续使用 Tailwind 原生刻度。

---

## 6. 圆角、尺寸与排版

### 6.1 圆角

HeroUI v3 参考源码中，默认基础圆角为 8px，Button 使用 24px 圆角级别，Input 使用 12px 的 Field 圆角，Card 默认对应 24px。本方案将这些形态转成稳定的组件角色 Token，不照搬类名。[HeroUI 圆角映射](https://github.com/heroui-inc/heroui/blob/v3/packages/styles/themes/shared/theme.css)、[Button 样式](https://github.com/heroui-inc/heroui/blob/v3/packages/styles/components/button.css)、[Card 样式](https://github.com/heroui-inc/heroui/blob/v3/packages/styles/components/card.css)

| 组件 | 默认圆角 | 使用方式 |
| --- | --- | --- |
| Button、图标按钮 | 24px | `rounded-control`；常规高度下近似胶囊或圆形 |
| Input、Select Trigger、Textarea | 12px | `rounded-field` |
| Menu Item、紧凑选项 | 8px | `rounded-item` |
| Dropdown、Popover | 16px | `rounded-popover` |
| Card、Dialog | 24px | `rounded-card / rounded-dialog` |
| Avatar、状态 Chip、Switch 轨道 | 全圆 | `rounded-full` |

Popover 与 Menu Item 的数值是本方案的配套选择，不保证与所有 HeroUI 组件一一相同。不要把 Textarea、表格容器和大型面板统一做成胶囊形。

### 6.2 尺寸与密度

HeroUI 参考样式的按钮按屏幕宽度调整高度：桌面常见 `sm / md / lg` 为 32 / 36 / 40px，较窄屏幕为 36 / 40 / 44px。**本项目不复制按断点缩小触摸目标的规则**，默认交互目标遵循《[[前端交互与可访问性规范]]》的 44px 基线。

| 场景 | 项目尺寸建议 |
| --- | --- |
| 默认 Button、Input、Select Trigger | `min-h-11`，即至少 44px |
| 明确需要紧凑密度的桌面工具栏 | 可选 32 / 36 / 40px，但触摸环境恢复至少 44px |
| 常规按钮水平内边距 | `px-4`；紧凑按钮 `px-3` |
| 按钮图标与文字 | `gap-2`，图标 16–20px |
| 菜单项 | 默认至少 44px；密集桌面菜单按实际场景收紧 |
| 标准卡片 | `p-4 gap-3`；内容较多的表单卡片可用 `p-6` |
| 表单字段之间 | `gap-4`；字段 Label 与控件之间 `gap-2` |
| 页面主要分区 | `gap-6` 或 `gap-8` |

用最小高度而不是固定高度承载可能换行的内容。长中文、错误提示和字体放大后，增加高度而不是截断。

### 6.3 排版

- 默认使用系统无衬线字体栈并补齐中文回退，不额外引入装饰字体。
- 正文和输入文字以 14–16px 为主；移动输入框优先 16px。
- 按钮、Label 和常规卡片标题使用 Medium；分区标题可用 Semibold。
- 卡片标题不必统一放大到 20–24px，通过层级和间距体现重要性。
- 次要文字使用 `text-muted-foreground`，不给整段内容设置低透明度。
- 金额、时间、计数等需要纵向对齐时用 `tabular-nums`。

---

## 7. 表面、阴影与边框

### 7.1 浅色主题

采用”浅灰画布 → 白色表面 → 柔和阴影”的关系。普通 Card 默认 `bg-surface shadow-surface`，不统一附加 `border border-border`。

阴影只负责轻量分离，不用品牌色发光阴影，也不让普通信息卡片在 Hover 时全部上浮。只有可点击的卡片提供交互反馈。

### 7.2 深色主题

深色主题主要依靠画布、卡片和弹层的明度差。Card 与 Field 取消外投影；Overlay 保留轻微内描边，避免浮层与下方内容融为一体。

`surface-secondary`、`surface-tertiary` 表达嵌套背景，不是”颜色越亮 z-index 越高”。遮挡顺序由组件定位与统一层级策略控制。

### 7.3 边框按用途出现

| 用途 | 处理 |
| --- | --- |
| 普通信息卡片 | 默认无边框，表面与阴影分层 |
| 表格分隔线、区块分隔 | `border-border`，保持轻量 |
| Outline Button | `border-strong` 显式边框，不能与 Filled 混成同一种变体 |
| 必须靠轮廓识别的输入框 | `border-field-border`，确保实际边界清晰 |
| Focus | 独立的 `ring` 色，不用普通装饰边框代替 |
| Invalid | 错误文字 + 错误指示，不能只有红色轮廓 |
| Forced Colors | 阴影与表面分层被引擎移除：弹层容器与卡片补系统色 outline，菜单高亮改用 Highlight 轮廓环 |

HeroUI 默认 Input 使用白色 Field、轻阴影和无可见边框，也提供中性填充的 Secondary 变体。本方案允许这两种外观，但不能以”官方默认无边框”为由豁免控件可辨识性检查——白色卡片上的白色输入框尤其要检查，必要时用显式描边变体。[HeroUI Input 样式](https://github.com/heroui-inc/heroui/blob/v3/packages/styles/components/input.css)

Forced Colors 下（2026-09-06 Chromium 实测）阴影与表面明度差全部失效，轮廓成为唯一层级通道。引擎会把 `outline-hidden` 的透明轮廓渲染为可见系统色，按钮、输入框等交互元素因此天然可辨识；但弹层容器与卡片没有轮廓，会完全失去边界，必须补 `forced-colors:outline-1!`、`forced-colors:outline-solid!` 与 `forced-colors:outline-[CanvasText]!`——`!` 后缀用于压制 Reka 弹层内容上内联的 `outline: none`，普通规则会被它覆盖。菜单高亮不能依赖背景色：作者自定义背景上的文字会被引擎垫上不透明的 Canvas 背板，`Highlight` 背景 + `HighlightText` 文字会因白字落在白色背板上整体不可读，高亮应改用 `Highlight` 色轮廓环表达，文字色保持与 Canvas（背板）对比。

---

## 8. 组件视觉配方

### 8.1 Button

业务 API 使用语义 Variant，不沿用 HeroUI v2 的 `flat / light / bordered / shadow` 分类再与 v3 名称混用。

| Variant | 默认配方 | Hover / Pressed |
| --- | --- | --- |
| `primary` | `bg-primary text-primary-foreground` | `primary-hover / primary-active` |
| `secondary` | `bg-default text-primary-text` | `default-hover` |
| `tertiary` | `bg-default text-default-foreground` | `default-hover` |
| `outline` | 透明底、中性文字、`border-border-strong` 显式边框 | 轻量中性填充 |
| `ghost` | 透明底、中性文字、无边框 | `default` 填充 |
| `danger` | `bg-danger text-danger-foreground` | `danger-hover` |
| `danger-soft` | `bg-danger-subtle text-danger-text` | `danger-subtle-hover` |

这里的 `secondary` 是中性底上的品牌文字，不是浅蓝色实心块；与 `tertiary` 的区分参考当前 HeroUI v3 Button 样式。`danger-soft` 用于低强调危险入口，最终不可逆确认可升级为 `danger`。[HeroUI Button](https://heroui.com/docs/react/components/button)

原生按钮的视觉骨架：

```html
<button
  type="button"
  class="
    inline-flex min-h-11 items-center justify-center gap-2
    rounded-control px-4 py-2 text-sm font-medium
    bg-primary text-primary-foreground
    enabled:hover:bg-primary-hover enabled:active:bg-primary-active
    motion-safe:enabled:active:scale-97
    transition-[background-color,scale] duration-150 ease-out
    motion-reduce:transition-none
    outline-hidden focus-visible:outline-solid
    focus-visible:outline-2
    focus-visible:outline-offset-2 focus-visible:outline-ring
    disabled:cursor-not-allowed disabled:opacity-50
  "
>
  保存修改
</button>
```

这是样式示例，实际由共享 Button 组件封装。Disabled 使用原生 `disabled`；Pending 保留按钮宽度与可读文案、表达 `aria-busy` 并阻止重复激活。只设置 `aria-disabled` 或 `pointer-events-none` 不会阻止键盘触发。

> [!important] `outline-hidden` 之后必须恢复 `outline-solid`
>
> `outline-hidden` 会把 `--tw-outline-style` 固定为 `none`，而 `outline-2` 只声明宽度（`outline-style: var(--tw-outline-style)`）。缺少 `focus-visible:outline-solid` 时，整个配方渲染不出任何焦点环（已在 Tailwind CSS 4.3 构建产物中实测确认）。所有使用 `outline-hidden` 的焦点配方，都要在 `focus-visible:` 下显式写回 `outline-solid`。

### 8.2 Field

Label 放在控件外部并保持可见；Placeholder 只提供示例。默认 Field 与 Secondary Filled Field 共享圆角、尺寸、焦点和错误规范，不由每个表单自行选择风格。

以下使用显式描边，适合需要清晰输入边界的表单：

```html
<div class="flex flex-col gap-2">
  <label for="project-name" class="text-sm font-medium text-foreground">
    项目名称
  </label>
  <input
    id="project-name"
    name="projectName"
    type="text"
    aria-describedby="project-name-help"
    placeholder="例如：个人知识库"
    class="
      min-h-11 w-full rounded-field border border-field-border
      bg-field px-3 py-2 text-base text-foreground shadow-field
      placeholder:text-muted-foreground
      enabled:hover:bg-field-hover
      outline-hidden focus-visible:outline-solid
      focus-visible:outline-2
      focus-visible:outline-offset-2 focus-visible:outline-ring
      disabled:cursor-not-allowed disabled:opacity-50
    "
  />
  <p id="project-name-help" class="text-sm text-muted-foreground">
    使用容易识别的名称。
  </p>
</div>
```

Invalid 状态设置真实的 `aria-invalid`、关联错误信息，并保留可见焦点；不让错误色覆盖键盘 Focus。Secondary Filled 变体可采用 `bg-default` 与无阴影，是否保留边框由实际背景中的可辨识性决定。

### 8.3 其他组件

| 组件 | 视觉配方与注意点 |
| --- | --- |
| Card | `rounded-card bg-surface p-4 shadow-surface`；不默认可点击；Forced Colors 下补 §7.3 的容器 outline |
| Dialog | `rounded-dialog bg-overlay p-6 shadow-overlay`；Backdrop 独立；标题、说明、关闭与底部操作位置一致；Forced Colors 下补容器 outline |
| Dropdown / Popover | `rounded-popover bg-overlay p-1 shadow-overlay`；内容保持不透明，避免文字叠影；Forced Colors 下补容器 outline（Reka 有内联 `outline: none`，必须带 `!` 后缀） |
| Menu Item | `rounded-item`；高亮用 `default-hover`，与深色 Overlay 区分；危险项用 `danger-text` 与 `danger-subtle`；Forced Colors 下高亮补 `Highlight` 色轮廓环 |
| Chip / Badge | `rounded-full`；默认 Subtle + Text；成功、警告、错误有文字或图标辅助 |
| Tabs | 中性圆角轨道 + 清晰的选中块；选中态同时由语义与位置表达，不只改颜色 |
| Checkbox / Radio | 使用 `primary` 表达选中，并保留勾选或圆点形状；标签区域一起可操作 |
| Switch | 全圆轨道 + 清晰滑块位移；同时保留 Checked 与 Focus 状态 |
| Table / List | 优先清晰行列、分隔和选中标识；不把每一行包装成独立悬浮卡片 |

普通按钮、输入框和卡片可用原生 HTML 封装，不为每一种基础元素寻找 Reka Primitive。Dialog、Menu、Select、Tabs 等复杂交互交给 Reka。

---

## 9. Reka UI 状态接入

样式读取 Reka 实际暴露的状态，不自行维护 Hover / Selected 状态，也不复制 React Aria 的属性名称。

| 状态来源 | Tailwind 表达 |
| --- | --- |
| 原生按钮 Hover / Active | `enabled:hover:` / `enabled:active:` |
| 原生键盘 Focus | `focus-visible:` |
| Reka Menu 高亮 | `data-highlighted:` |
| Reka 禁用 | `data-disabled:` |
| Reka 打开、选中、勾选 | `data-[state=open]:`、`data-[state=active]:`、`data-[state=checked]:`，以具体组件文档为准 |
| 表单错误 | `aria-invalid:`，由真实校验结果设置 |

DropdownMenu 的 `data-highlighted` 和 `data-disabled` 是“属性存在即成立”，不是 `="true"`；不同组件的 `data-state` 值不能混用。[Reka DropdownMenu](https://reka-ui.com/docs/components/dropdown-menu)

一个只负责”发出重命名意图”的小型 SFC，完整保留 Trigger、Portal、Content 与 Item 结构：

```vue
<script setup lang="ts">
import {
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuPortal,
  DropdownMenuRoot,
  DropdownMenuTrigger,
} from 'reka-ui'

const emit = defineEmits<{
  rename: []
}>()
</script>

<template>
  <DropdownMenuRoot>
    <DropdownMenuTrigger
      class="
        inline-flex min-h-11 items-center justify-center rounded-control
        bg-default px-4 py-2 text-sm font-medium text-default-foreground
        hover:bg-default-hover data-[state=open]:bg-default-hover
        outline-hidden focus-visible:outline-solid
        focus-visible:outline-2
        focus-visible:outline-offset-2 focus-visible:outline-ring
      "
    >
      更多操作
    </DropdownMenuTrigger>

    <DropdownMenuPortal>
      <DropdownMenuContent
        align="end"
        :side-offset="8"
        class="
          z-50 min-w-44 max-w-[calc(100vw-2rem)]
          max-h-(--reka-dropdown-menu-content-available-height)
          overflow-y-auto rounded-popover bg-overlay p-1
          text-foreground shadow-overlay
        "
      >
        <DropdownMenuItem
          class="
            flex min-h-11 cursor-default items-center rounded-item px-3 py-2 text-sm
            data-highlighted:bg-default-hover
            outline-hidden focus-visible:outline-solid
            focus-visible:outline-2
            focus-visible:-outline-offset-2 focus-visible:outline-ring
            data-disabled:pointer-events-none data-disabled:opacity-50
          "
          @select="emit('rename')"
        >
          重命名
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenuPortal>
  </DropdownMenuRoot>
</template>
```

菜单动作通过 Reka 的 Select 事件处理，避免仅监听鼠标点击。生产封装仍需透传必要的 Props、Attrs 与事件；不覆盖 Reka 的键盘、焦点恢复或关闭逻辑。Dialog 需要 Title、Description 等语义组成，不能用一张外观相似的 Card 替代。

---

## 10. 状态与动效

动效采用《[[前端交互与可访问性规范]]》的时间基线：

- 按钮、输入框等微反馈：100–200ms。
- Menu、Popover、Dialog：150–250ms，退出不长于进入。
- 普通按钮可在按压时缩小到约 0.97；不让整个卡片、表格行或页面持续缩放。
- 不使用 `transition-all`；只声明实际变化的颜色、透明度、位移或缩放。
- Tailwind v4 的 `scale-*` 使用独立 `scale` 属性，Transition 列表需要包含 `scale`，不能只写 `transform`。
- Reduced Motion 下移除缩放、位移和循环装饰动画，保留必要的状态反馈。
- 不为视觉对齐引入 Framer Motion、涟漪依赖或持续 `will-change`。

弹层可采用短暂淡入与轻微缩放，缩放原点使用 Reka 提供的定位变量。退出动画必须与组件的卸载生命周期配合，不能只给 `data-[state=open]` 写一个看起来有效的入场类。

所有交互组件至少覆盖 Default、Hover、Pressed、Focus、Disabled；支持异步或校验的组件补齐 Pending、Invalid。Selected / Checked 是持久状态，不能用短暂 Hover 样式代替。

---

## 11. Light / Dark / System

```ts
type Theme = 'light' | 'dark' | 'system'
type ResolvedTheme = 'light' | 'dark'
```

保存用户偏好 `Theme`，应用到 DOM 的是解析后的 `ResolvedTheme`：

```html
<html class="light">
<html class="dark">
```

两行表示互斥的主题状态，不同时添加两个类。实现时：

1. 在首屏绘制前解析偏好，尽量避免先亮后暗。
2. `system` 根据 `prefers-color-scheme` 解析，并响应系统主题变化。
3. 显式 `light / dark` 不被系统变化覆盖；持久化失败时仍能使用。
4. 主题类放在 `html`，保证挂到 `body` 的 Reka Portal 继承同一主题。
5. CSS 的 `color-scheme` 与当前主题同步，让原生控件和滚动条保持一致。

业务组件使用同一组语义类：

```html
<section class="rounded-card bg-surface p-4 text-foreground shadow-surface">
  内容
</section>
```

不在每个组件重复写 `bg-white dark:bg-...`。阴影也通过主题变量切换，不需要四处增加 `dark:shadow-none`。`dark:` 只保留给图片亮度、主题专属资产等确实不同的行为。

---

## 12. 项目组织与落地顺序

沿用《[[前端应用架构规范]]》的边界：

```text
src/
├── components/
│   └── ui/
│       ├── Button.vue
│       ├── Input.vue
│       ├── Card.vue
│       └── ...
└── styles/
    ├── main.css
    └── theme.css
```

这是应用代码中的职责示意，不要求预建全部组件目录。

- `main.css`：样式入口与应用级基础样式。
- `theme.css`：Primitive 导入、浅深语义值、圆角、阴影与 Tailwind 映射。
- `components/ui/`：共享组件外观与状态封装，不直接访问业务 API。
- 复杂组件按实际需要拆文件；先不建独立 Design Token 包或跨框架组件仓库。
- Variant 类名保持完整、可静态发现，不拼接 `bg-${color}-500` 字符串。

建议按以下顺序落地：

1. 校准画布、表面、文字与主按钮，检查浅深两套主题。
2. 完成 Button、Field、Card 三个基础配方。
3. 接入 Menu、Select、Dialog，检查 Portal、Focus 与状态样式。
4. 补充 Chip、Tabs、选择控件，放进真实表单与列表页面验证。
5. 收敛间距、字体和动效；不在孤立色块上验收设计系统。

---

## 13. 视觉验收与迁移检查

在实际应用的组件工作台中，用相同文案、相近尺寸并排对照已锁定版本的 HeroUI v3 默认示例。验收目标是视觉语言一致，不要求 Vue 与 React 的 DOM 或 Props 相同。

- [ ] 浅色页面有清楚的灰底 / 白色表面关系，不是白底上堆灰框。
- [ ] 深色 Card、嵌套区域和 Overlay 能区分，不是删除所有阴影就结束。
- [ ] Button 的胶囊感、Field 的 12px 圆角、Card 的 24px 圆角保持一致。
- [ ] Secondary、Tertiary、Ghost 与 Outline 的强调程度不同。
- [ ] 主色、状态色同时具备正确的前景文字，不默认全部白字。
- [ ] Primary Subtle、Danger Soft、菜单高亮和持久选中态均已验证。
- [ ] 键盘焦点可见；Disabled、Pending、Invalid 不误触发操作。
- [ ] `overlay` 与 `backdrop` 已按真实用途迁移。
- [ ] 若旧组件用到 `primary-border`，已按用途改为 `primary-text` 或独立的语义边界，不保留未定义 Token。
- [ ] 所有示例依赖的 Token 都有浅深主题值，组件未残留 Slate / Indigo 的直接颜色绑定。
- [ ] 默认触摸目标、长中文、文字缩放和窄视口满足项目规范。
- [ ] 已在实际前景 / 背景组合下验证对比度，不用 Radix 或 HeroUI 的名称代替检查。
- [ ] 已验证 Reduced Motion、Forced Colors 与 Portal 主题继承。

具体键盘、表单、弹层、对比度和测试门槛分别见《[[前端交互与可访问性规范]]》与《[[前端测试规范]]》。
