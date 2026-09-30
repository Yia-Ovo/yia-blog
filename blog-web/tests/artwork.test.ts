import { describe, expect, it } from 'vitest'
import { appearanceRefreshDelay, artworkTheme, contrastRatio, extractArtworkColor } from '../src/lib/artwork'
import type { RGB } from '../src/lib/artwork'

const rgb = (hex: string): RGB => [parseInt(hex.slice(1, 3), 16), parseInt(hex.slice(3, 5), 16), parseInt(hex.slice(5, 7), 16)]

describe('image-derived appearance', () => {
  it('finds a colored area without letting black, white or transparent pixels dominate', () => {
    const pixels = [...Array(100).fill([0, 0, 0, 255]).flat(), ...Array(100).fill([255, 255, 255, 255]).flat(), ...Array(100).fill([255, 0, 0, 0]).flat(), ...Array(20).fill([40, 100, 180, 255]).flat()]
    expect(extractArtworkColor(pixels)).toEqual([40, 100, 180])
    expect(extractArtworkColor([])).toEqual(extractArtworkColor([128, 128, 128, 255]))
  })

  it('keeps text and primary buttons readable for different source hues and both modes', () => {
    const colors: RGB[] = [[240, 20, 20], [20, 220, 20], [20, 30, 230], [255, 220, 0], [180, 40, 180], [0, 220, 240], [0, 0, 0], [255, 255, 255], [184, 137, 104]]
    for (const color of colors) for (const mode of ['dark', 'light'] as const) {
      const theme = artworkTheme(color, mode)
      for (const foreground of ['--text', '--muted', '--muted-strong', '--accent', '--accent-strong']) {
        for (const surface of ['--bg', '--bg-elevated', '--surface', '--surface-raised', '--surface-soft']) {
          expect(contrastRatio(rgb(theme[foreground]!), rgb(theme[surface]!)), `${color} ${mode} ${foreground} on ${surface}`).toBeGreaterThanOrEqual(4.5)
        }
      }
      expect(contrastRatio(rgb(theme['--accent']!), rgb(theme['--accent-ink']!))).toBeGreaterThanOrEqual(4.5)
      expect(contrastRatio(rgb(theme['--accent-strong']!), rgb(theme['--accent-ink']!))).toBeGreaterThanOrEqual(4.5)
    }
    expect(artworkTheme([30, 80, 200], 'dark')['--accent']).not.toEqual(artworkTheme([220, 70, 20], 'dark')['--accent'])
  })

  it('schedules the server midnight independently of client timezone and clock', () => {
    expect(appearanceRefreshDelay('2026-09-29T23:59:58+08:00', '2026-09-30T00:00:00+08:00', 200)).toBe(1800)
    expect(appearanceRefreshDelay('2026-09-29T15:59:58Z', '2026-09-29T16:00:00Z', 3000)).toBe(1000)
    expect(() => appearanceRefreshDelay('broken', 'broken')).toThrow()
    expect(() => appearanceRefreshDelay('2026-09-29T00:00:00Z', '2026-10-02T00:00:00Z')).toThrow()
  })
})
