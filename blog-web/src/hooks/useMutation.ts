import { useCallback, useEffect, useRef, useState } from 'react'

/** 防止重复提交；离开页面后，旧写请求不能清空新表单或触发跳转。 */
export function useMutation(scope?: unknown) {
  const [busy, setBusy] = useState(false)
  const running = useRef(false)
  const generation = useRef(0)

  useEffect(() => {
    setBusy(false)
    return () => { generation.current++; running.current = false }
  }, [scope])

  const run = useCallback(async <T,>(
    action: () => Promise<T>,
    onSuccess: (value: T) => void,
    onError: (cause: unknown) => void,
  ) => {
    if (running.current) return
    running.current = true
    const current = ++generation.current
    setBusy(true)
    try {
      const value = await action()
      if (current === generation.current) onSuccess(value)
    } catch (cause) {
      if (current === generation.current) onError(cause)
    } finally {
      if (current === generation.current) { running.current = false; setBusy(false) }
    }
  }, [])

  return { busy, run }
}
