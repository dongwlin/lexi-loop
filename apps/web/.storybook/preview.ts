// Story 预览加载应用唯一样式入口，保证 Design Token / Tailwind 与应用渲染一致。
import type { Preview } from '@storybook/vue3-vite'

import '../src/styles/main.css'

const preview: Preview = {
  parameters: {
    // axe 检查默认作为测试门禁（前端测试规范 §10.5）：违规即失败。
    // 个别 Story 需要放宽时须显式标注 'todo'（已登记待修）或 'off'（反例展示）。
    a11y: {
      test: 'error',
    },
  },
}

export default preview
