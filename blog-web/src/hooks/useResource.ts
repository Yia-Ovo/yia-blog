import { useCallback, useEffect, useState } from 'react'
import { errorMessage } from '../api/http'

type Loader<T> = (signal: AbortSignal) => Promise<T>
interface Result<T> { load: Loader<T>; version: number; data?: T; error: string }

/** load 使用 useCallback 声明；筛选条件或身份变化时立即隐藏旧数据。 */
export function useResource<T>(load: Loader<T>) {
  const [version, setVersion] = useState(0)
  const [result, setResult] = useState<Result<T>>()
  const reload = useCallback(() => { setVersion((value) => value + 1) }, [])

  useEffect(() => {
    const controller = new AbortController()
    let active = true
    void (async () => {
      try {
        const data = await load(controller.signal)
        if (active) setResult({ load, version, data, error: '' })
      } catch (cause) {
        if (active && !controller.signal.aborted) setResult({ load, version, error: errorMessage(cause) })
      }
    })()
    // 除了取消网络请求，还要拒绝忽略 AbortSignal 的旧 Promise，兼容 StrictMode。
    return () => { active = false; controller.abort() }
  }, [load, version])

  const current = result?.load === load && result.version === version ? result : undefined
  return { data: current?.data, pending: !current, error: current?.error || '', reload }
}
