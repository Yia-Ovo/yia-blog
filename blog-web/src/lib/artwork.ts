export type AppearanceMode = 'dark' | 'light'
export type RGB = readonly [number, number, number]
export interface LoadedArtwork { color: RGB; objectURL: string }

const fallback: RGB = [184, 137, 104]
const clamp = (value: number, min: number, max: number) => Math.min(max, Math.max(min, value))

function toHSL([red, green, blue]: RGB): RGB {
  const [r, g, b] = [red / 255, green / 255, blue / 255]
  const max = Math.max(r!, g!, b!), min = Math.min(r!, g!, b!)
  const delta = max - min, light = (max + min) / 2
  if (!delta) return [0, 0, light]
  const hue = max === r ? ((g! - b!) / delta + (g! < b! ? 6 : 0)) : max === g ? (b! - r!) / delta + 2 : (r! - g!) / delta + 4
  return [hue * 60, delta / (1 - Math.abs(2 * light - 1)), light]
}

function fromHSL(hue: number, saturation: number, lightness: number): RGB {
  const h = ((hue % 360) + 360) % 360
  const c = (1 - Math.abs(2 * lightness - 1)) * saturation
  const x = c * (1 - Math.abs((h / 60) % 2 - 1)), m = lightness - c / 2
  const rgb = h < 60 ? [c, x, 0] : h < 120 ? [x, c, 0] : h < 180 ? [0, c, x] : h < 240 ? [0, x, c] : h < 300 ? [x, 0, c] : [c, 0, x]
  return rgb.map(value => Math.round((value + m) * 255)) as unknown as RGB
}

function hex(rgb: RGB): string { return '#' + rgb.map(value => clamp(Math.round(value), 0, 255).toString(16).padStart(2, '0')).join('') }

function luminance(rgb: RGB): number {
  const values = rgb.map(value => {
    const channel = value / 255
    return channel <= .04045 ? channel / 12.92 : ((channel + .055) / 1.055) ** 2.4
  })
  return values[0]! * .2126 + values[1]! * .7152 + values[2]! * .0722
}

export function contrastRatio(first: RGB, second: RGB): number {
  const a = luminance(first), b = luminance(second)
  return (Math.max(a, b) + .05) / (Math.min(a, b) + .05)
}

// 按色块面积与饱和度加权，略过透明、近黑与近白像素，避免暗背景吞掉主题色。
export function extractArtworkColor(pixels: ArrayLike<number>): RGB {
  const buckets = new Map<number, { sum: number[]; count: number; score: number }>()
  for (let i = 0; i + 3 < pixels.length; i += 4) {
    if (pixels[i + 3]! < 160) continue
    const rgb: RGB = [pixels[i]!, pixels[i + 1]!, pixels[i + 2]!]
    const [, saturation, light] = toHSL(rgb)
    if (light < .12 || light > .88 || saturation < .12) continue
    const key = (rgb[0] >> 5) * 64 + (rgb[1] >> 5) * 8 + (rgb[2] >> 5)
    const bucket = buckets.get(key) || { sum: [0, 0, 0], count: 0, score: 0 }
    rgb.forEach((value, channel) => { bucket.sum[channel]! += value })
    bucket.count++
    bucket.score += .3 + saturation * (1 - Math.abs(light - .5))
    buckets.set(key, bucket)
  }
  const best = [...buckets.entries()].sort((a, b) => b[1].score - a[1].score || a[0] - b[0])[0]?.[1]
  return best ? best.sum.map(value => Math.round(value / best.count)) as unknown as RGB : fallback
}

function palette(color: RGB, mode: AppearanceMode): Record<string, string> {
  const [hue, originalSaturation] = toHSL(color)
  const dark = mode === 'dark'
  const saturation = clamp(originalSaturation * .25, .07, .18)
  const tone = (light: number, sat = saturation) => fromHSL(hue, sat, light)
  const surfaces = dark ? [tone(.055), tone(.085), tone(.105), tone(.14), tone(.18)] : [tone(.91), tone(.87), tone(.95), tone(.985), tone(.88)]
  const readable = (start: RGB) => {
    let result = start
    for (let step = 0; step < 100 && surfaces.some(surface => contrastRatio(result, surface) < 4.5); step++) {
      result = result.map(value => dark ? Math.min(255, value + 2) : Math.max(0, value - 2)) as unknown as RGB
    }
    return result
  }
  const accent = readable(tone(dark ? .65 : .40, clamp(originalSaturation, .28, .55)))
  const strong = readable(tone(dark ? .75 : .32, clamp(originalSaturation, .28, .55)))
  const ink: RGB = contrastRatio(accent, [10, 10, 10]) >= contrastRatio(accent, [255, 255, 255]) ? [10, 10, 10] : [255, 255, 255]
  return {
    '--bg': hex(surfaces[0]!), '--bg-elevated': hex(surfaces[1]!), '--surface': hex(surfaces[2]!),
    '--surface-raised': hex(surfaces[3]!), '--surface-soft': hex(surfaces[4]!),
    '--text': hex(readable(tone(dark ? .92 : .15))),
    '--muted': hex(readable(tone(dark ? .64 : .42))),
    '--muted-strong': hex(readable(tone(dark ? .79 : .28))),
    '--border': hex(tone(dark ? .24 : .76)), '--border-strong': hex(tone(dark ? .33 : .64)),
    '--accent': hex(accent), '--accent-strong': hex(strong), '--accent-ink': hex(ink),
    '--violet': hex(readable(fromHSL(hue + 45, .22, dark ? .67 : .38))),
    '--glass': 'color-mix(in srgb, var(--bg) 94%, transparent)',
  }
}

// 所有普通文字在各层面板上保持至少 4.5:1；图片区域始终使用深色遮罩和独立文字色。
export function artworkTheme(color: RGB, mode: AppearanceMode): Record<string, string> {
  const dark = palette(color, 'dark')
  return {
    ...palette(color, mode),
    '--art-text': dark['--text']!, '--art-muted': dark['--muted-strong']!,
    '--art-accent': dark['--accent-strong']!, '--art-button': dark['--accent']!,
    '--art-button-ink': dark['--accent-ink']!,
    '--art-base': dark['--bg']!,
    '--art-overlay': dark['--bg']! + 'f7',
    '--art-overlay-mid': dark['--bg']! + 'df',
    '--art-overlay-soft': dark['--bg']! + '45',
    '--art-overlay-bottom': dark['--surface']! + 'b3',
    '--art-overlay-mobile': dark['--bg']! + 'e8',
    '--art-overlay-mobile-mid': dark['--bg']! + 'bf',
    '--art-overlay-mobile-soft': dark['--bg']! + '85',
  }
}

export async function loadArtwork(url: string, signal: AbortSignal): Promise<LoadedArtwork> {
  const response = await fetch(url, { signal, credentials: 'omit', mode: 'cors' })
  if (!response.ok) throw new Error('Artwork unavailable')
  const blob = await response.blob()
  if (!blob.type.startsWith('image/') || blob.size > 4 * 1024 * 1024) throw new Error('Invalid artwork')
  const objectURL = URL.createObjectURL(blob)
  try {
    const image = new Image()
    await new Promise<void>((resolve, reject) => {
      const cleanup = () => { signal.removeEventListener('abort', abort); image.onload = null; image.onerror = null }
      const abort = () => { cleanup(); image.src = ''; reject(new DOMException('Cancelled', 'AbortError')) }
      image.onload = () => { cleanup(); resolve() }
      image.onerror = () => { cleanup(); reject(new Error('Artwork decode failed')) }
      signal.addEventListener('abort', abort, { once: true })
      if (signal.aborted) { abort(); return }
      image.src = objectURL
    })
    if (signal.aborted) throw new DOMException('Cancelled', 'AbortError')
    const canvas = document.createElement('canvas')
    canvas.width = 96; canvas.height = 64
    const context = canvas.getContext('2d', { willReadFrequently: true })
    if (!context) throw new Error('Canvas unavailable')
    context.drawImage(image, 0, 0, canvas.width, canvas.height)
    return { objectURL, color: extractArtworkColor(context.getImageData(0, 0, canvas.width, canvas.height).data) }
  } catch (error) {
    URL.revokeObjectURL(objectURL)
    throw error
  }
}

// 使用服务器给出的剩余时长，容忍设备时区、时钟偏差以及请求跨过零点。
export function appearanceRefreshDelay(serverTime: string, nextChangeAt: string, elapsed = 0): number {
  const remaining = Date.parse(nextChangeAt) - Date.parse(serverTime)
  if (!Number.isFinite(remaining) || remaining <= 0 || remaining > 86_400_000) throw new Error('Invalid appearance schedule')
  return clamp(remaining - elapsed, 1000, 86_400_000)
}
