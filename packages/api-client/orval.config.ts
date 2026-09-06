import { defineConfig } from 'orval'

// 输入是 scripts/unwrap-envelope.mjs 从 docs/openapi/openapi.json 派生的 spec
// （200 响应已剥掉 Envelope 外壳）。先跑 `pnpm -F @lexi-loop/api-client generate`。
export default defineConfig({
  'lexi-loop': {
    input: './spec/openapi.json',
    output: {
      target: './src/generated/client.ts',
      schemas: './src/generated/model',
      client: 'fetch',
      mode: 'single',
      clean: true,
      override: {
        fetch: {
          // 统一解包：mutator 校验 code 后直接返回 data（前端 API 与认证集成规范 §5.2），
          // 不生成 { data, status, headers } 包装类型。
          includeHttpResponseReturnType: false,
        },
        mutator: {
          path: './src/mutator.ts',
          name: 'customFetch',
        },
      },
    },
  },
})
