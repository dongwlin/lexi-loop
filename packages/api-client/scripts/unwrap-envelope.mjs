// 从 docs/openapi/openapi.json 派生 Orval 输入 spec（packages/api-client/spec/openapi.json）：
// 1) 每个操作的 200 响应去掉 EnvelopeXxx 外壳，直接引用 data 的 schema——
//    运行时的 {code, message, data} 解包由 mutator 统一负责（前端技术栈 §7.2），
//    生成类型的 Promise<T> 与 mutator 返回值才能保持一致；
// 2) 只保留 200 响应、移除 Envelope* / Error / ErrorData / FieldError 组件——
//    错误响应模型由包内 errors.ts 的 ApiError 统一承载，生成代码不重复建模。
// 本文件由 `pnpm -F @lexi-loop/api-client generate` 自动执行；派生产物可随时重建，勿手改。
import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const packageRoot = resolve(dirname(fileURLToPath(import.meta.url)), '..')
const repoRoot = resolve(packageRoot, '../..')

const source = JSON.parse(readFileSync(resolve(repoRoot, 'docs/openapi/openapi.json'), 'utf8'))
const spec = structuredClone(source)
const components = spec.components?.schemas ?? {}

for (const pathItem of Object.values(spec.paths)) {
  for (const [method, operation] of Object.entries(pathItem)) {
    if (method === 'parameters' || typeof operation !== 'object' || operation.responses === undefined) continue
    const ok = operation.responses['200']
    if (ok === undefined) continue

    const schema = ok.content?.['application/json']?.schema
    const envelopeName = schema?.$ref?.split('/').at(-1)
    const envelope = typeof envelopeName === 'string' ? components[envelopeName] : undefined
    if (envelope?.properties?.data === undefined) {
      throw new Error(`200 响应不是 Envelope 引用，无法解包：${method.toUpperCase()} ${JSON.stringify(schema)}`)
    }
    ok.content['application/json'].schema = envelope.properties.data

    operation.responses = { '200': ok }
  }
}

for (const name of Object.keys(components)) {
  if (name.startsWith('Envelope') || name === 'Error' || name === 'ErrorData' || name === 'FieldError') {
    delete components[name]
  }
}

mkdirSync(dirname(resolve(packageRoot, 'spec/openapi.json')), { recursive: true })
writeFileSync(resolve(packageRoot, 'spec/openapi.json'), `${JSON.stringify(spec, null, 2)}\n`)
console.log(`derived spec written: packages/api-client/spec/openapi.json`)
