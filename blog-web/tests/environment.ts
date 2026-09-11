import { builtinEnvironments } from 'vitest/environments'
import type { Environment } from 'vitest/environments'

const jsdom = builtinEnvironments.jsdom!

export default {
  ...jsdom,
  name: 'jsdom-with-node-fetch',
  async setup(global, options) {
    // Node 24 的原生 Request 校验 AbortSignal 品牌；取消和 Fetch 使用同一套原生实现。
    const native = { AbortController: global.AbortController, AbortSignal: global.AbortSignal }
    const environment = await jsdom.setup(global, options)
    global.AbortController = native.AbortController
    global.AbortSignal = native.AbortSignal
    return environment
  },
} satisfies Environment
