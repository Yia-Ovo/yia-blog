import { createContext } from 'react'

// 编辑器把离开检查交给页面外壳，登出按钮也遵守未保存和保存中的保护。
export const LeaveGuardContext = createContext<(guard: () => boolean) => () => void>(() => () => {})
