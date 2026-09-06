/*
 * ESLint Flat Config（前端技术栈 §11.1）：ESLint 负责正确性，格式化交给 Prettier，
 * 相互冲突的格式规则由末尾的 eslint-config-prettier 关闭。
 */
import { globalIgnores } from 'eslint/config'
import {
  defineConfigWithVueTs,
  vueTsConfigs,
} from '@vue/eslint-config-typescript'
import pluginVue from 'eslint-plugin-vue'
import vuejsAccessibility from 'eslint-plugin-vuejs-accessibility'
import tanstackQuery from '@tanstack/eslint-plugin-query'
import importX from 'eslint-plugin-import-x'
import prettier from 'eslint-config-prettier'

export default defineConfigWithVueTs(
  {
    name: 'lexi-loop/web/files',
    files: ['**/*.{ts,tsx,vue}'],
  },
  globalIgnores([
    '**/dist/**',
    '**/coverage/**',
    '**/storybook-static/**',
    '**/node_modules/**',
  ]),
  // Vue 官方规则（前端技术栈 §11.1）。
  pluginVue.configs['flat/recommended'],
  // TypeScript 类型感知规则（前端技术栈 §11.1）。
  vueTsConfigs.recommendedTypeChecked,
  // vuejs-accessibility 模板可访问性规则（前端技术栈 §11.1）。
  vuejsAccessibility.configs['flat/recommended'],
  {
    name: 'lexi-loop/web/label-has-for',
    files: ['**/*.vue'],
    rules: {
      // 默认要求 label 同时「嵌套控件 + 带 id」才放行，对 for 指向兄弟控件的
      // 可访问写法误报；二者满足其一即可（规则文档的 some 语义）。
      'vuejs-accessibility/label-has-for': [
        'error',
        { required: { some: ['nesting', 'id'] } },
      ],
    },
  },
  {
    name: 'lexi-loop/web/ui-components',
    files: ['src/components/ui/**/*.vue'],
    rules: {
      // 设计系统基础组件沿用 Button / Dialog 等单词命名（设计系统方案 §8），
      // multi-word 约束只对业务组件有意义。
      'vue/multi-word-component-names': 'off',
    },
  },
  // TanStack Query 官方规则（前端技术栈 §11.1）。
  tanstackQuery.configs['flat/recommended'],
  // Import 与无用代码检查（前端技术栈 §11.1）；未使用变量由 TS 规则承担。
  {
    name: 'lexi-loop/web/import-x',
    files: ['**/*.{ts,tsx,vue}'],
    plugins: { 'import-x': importX },
    settings: {
      'import-x/parsers': {
        // .vue 单文件组件的 <script> 由 vue-eslint-parser 解析（eslint-plugin-vue 依赖）。
        'vue-eslint-parser': ['.vue'],
      },
      'import-x/resolver': {
        typescript: {
          projectService: true,
          alwaysTryTypes: true,
        },
      },
    },
    rules: {
      'import-x/no-unresolved': 'error',
      'import-x/no-duplicates': 'error',
      'import-x/no-self-import': 'error',
      'import-x/no-useless-path-segments': 'error',
    },
  },
  // Prettier 只负责格式化：放在最后关闭与格式相关的 Lint 规则，避免两者争夺职责。
  prettier,
)
