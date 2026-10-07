import { useEffect, useRef, useState } from 'react'
import type { CSSProperties, KeyboardEvent, PointerEvent } from 'react'

type Position = { x: number; y: number }
const positionKey = 'yia.reading.menu.position.v1'
const margin = 12

/** useReadingMenuPosition 在当前标签页保存位置；缩放窗口后重新约束在可见区域。 */
export function useReadingMenuPosition() {
  const shell = useRef<HTMLDivElement>(null)
  const drag = useRef<{ pointer: number; start: Position; origin: Position } | null>(null)
  const [position, setPosition] = useState<Position | null>(null)
  const [dragging, setDragging] = useState(false)
  const [viewport, setViewport] = useState({ width: window.innerWidth, height: window.innerHeight })

  function clamp(point: Position) {
    const rect = shell.current?.getBoundingClientRect()
    return {
      x: Math.max(margin, Math.min(point.x, window.innerWidth - (rect?.width || 64) - margin)),
      y: Math.max(margin, Math.min(point.y, window.innerHeight - (rect?.height || 252) - margin)),
    }
  }
  function remember(point: Position) {
    try { sessionStorage.setItem(positionKey, JSON.stringify(point)) } catch { /* 禁用存储仍允许拖动。 */ }
  }
  function current() {
    const rect = shell.current?.getBoundingClientRect()
    return { x: rect?.left || 0, y: rect?.top || 0 }
  }
  useEffect(() => {
    let saved: Position | undefined
    try {
      const value = JSON.parse(sessionStorage.getItem(positionKey) || 'null')
      if (value && Number.isFinite(value.x) && Number.isFinite(value.y)) saved = value
    } catch { /* 损坏的位置数据使用默认位置。 */ }
    setPosition(clamp(saved || { x: window.innerWidth - 82, y: (window.innerHeight - (shell.current?.offsetHeight || 252)) / 2 }))
    const resize = () => {
      setViewport({ width: window.innerWidth, height: window.innerHeight })
      setPosition((point) => point ? clamp(point) : null)
    }
    window.addEventListener('resize', resize)
    const observer = typeof ResizeObserver === 'undefined' ? null : new ResizeObserver(() => {
      setPosition((point) => {
        if (!point) return point
        const next = clamp(point)
        return next.x === point.x && next.y === point.y ? point : next
      })
    })
    if (shell.current) observer?.observe(shell.current)
    return () => { window.removeEventListener('resize', resize); observer?.disconnect() }
  }, [])

  function pointerDown(event: PointerEvent<HTMLButtonElement>) {
    if (!event.isPrimary || event.button !== 0) return
    event.preventDefault()
    event.currentTarget.focus({ preventScroll: true })
    drag.current = { pointer: event.pointerId, start: { x: event.clientX, y: event.clientY }, origin: current() }
    event.currentTarget.setPointerCapture(event.pointerId)
    setDragging(true)
  }
  function pointerMove(event: PointerEvent<HTMLButtonElement>) {
    const active = drag.current
    if (!active || active.pointer !== event.pointerId) return
    const point = clamp({ x: active.origin.x + event.clientX - active.start.x, y: active.origin.y + event.clientY - active.start.y })
    setPosition(point)
  }
  function pointerEnd(event: PointerEvent<HTMLButtonElement>) {
    if (drag.current?.pointer !== event.pointerId) return
    drag.current = null
    setDragging(false)
    remember(clamp(current()))
  }
  function keyboardMove(event: KeyboardEvent<HTMLButtonElement>) {
    const moves: Record<string, readonly [number, number]> = { ArrowLeft: [-20, 0], ArrowRight: [20, 0], ArrowUp: [0, -20], ArrowDown: [0, 20] }
    const delta = moves[event.key]
    if (!delta && event.key !== 'Home') return
    event.preventDefault()
    const origin = current()
    const point = clamp(delta ? { x: origin.x + delta[0], y: origin.y + delta[1] } : { x: window.innerWidth - 82, y: window.innerHeight / 2 - 126 })
    setPosition(point)
    remember(point)
  }
  const panelWidth = Math.min(360, viewport.width - 32)
  const panelHeight = Math.min(viewport.height * .6, 460)
  const point = position || { x: viewport.width - 82, y: viewport.height / 2 - 126 }
  const candidate = point.x >= viewport.width / 2 ? point.x - panelWidth - 12 : point.x + 76
  const panelStyle: CSSProperties = {
    width: panelWidth,
    left: Math.max(16, Math.min(candidate, viewport.width - panelWidth - 16)),
    top: Math.max(16, Math.min(point.y, viewport.height - panelHeight - 16)),
  }
  return { shell, dragging, panelStyle, style: position ? { left: position.x, top: position.y, right: 'auto' } : undefined,
    handle: { onPointerDown: pointerDown, onPointerMove: pointerMove, onPointerUp: pointerEnd, onPointerCancel: pointerEnd, onLostPointerCapture: pointerEnd, onKeyDown: keyboardMove } }
}
