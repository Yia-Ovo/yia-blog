import { useId } from 'react'

export type WeatherKind = 'clear' | 'cloud' | 'fog' | 'rain' | 'snow' | 'storm' | 'unknown'
export function weatherDescription(code: number): { text: string; kind: WeatherKind } {
  if (code === 0) return { text: '晴朗', kind: 'clear' }
  if (code >= 1 && code <= 3) return { text: '多云', kind: 'cloud' }
  if ([45, 48].includes(code)) return { text: '雾', kind: 'fog' }
  if ([51, 53, 55, 56, 57, 61, 63, 65, 66, 67, 80, 81, 82].includes(code)) return { text: '有雨', kind: 'rain' }
  if ([71, 73, 75, 77, 85, 86].includes(code)) return { text: '有雪', kind: 'snow' }
  if ([95, 96, 99].includes(code)) return { text: '雷雨', kind: 'storm' }
  return { text: '天气变化中', kind: 'unknown' }
}

export default function WeatherScene({ kind, day }: { kind: WeatherKind; day: boolean }) {
  const id = useId()
  return <svg className="weather-scene" viewBox="0 0 160 120" aria-hidden="true">
    <defs><linearGradient id={id} x2="0.7" y2="1"><stop stopColor="#fff8e7" /><stop offset="1" stopColor={day ? '#efbc71' : '#b6c9ed'} /></linearGradient></defs>
    {day ? <><g stroke="#e7b879" strokeWidth="2.5" strokeLinecap="round"><path d="M78 12v7m0 58v7M42 48h7m58 0h7M52 22l5 5m42 42 5 5M52 74l5-5m42-42 5-5" /></g><circle cx="78" cy="48" r="23" fill={`url(#${id})`} /></>
      : <><path d="M91 15a32 32 0 1 0 26 47A28 28 0 0 1 91 15Z" fill={`url(#${id})`} /><g fill="#cfdef7"><circle cx="120" cy="24" r="2"/><circle cx="43" cy="36" r="1.5"/><path d="m116 81 2 5 5 2-5 2-2 5-2-5-5-2 5-2Z"/></g></>}
    {kind !== 'clear' && <path d="M48 85h68a16 16 0 0 0 1-32 24 24 0 0 0-46-7 17 17 0 0 0-23 13 13 13 0 0 0 0 26Z" fill="var(--weather-cloud, #dae3eb)" stroke="white" strokeOpacity=".3" />}
    {kind === 'rain' && <path d="m62 95-4 8m26-8-4 8m26-8-4 8" stroke="#7eafd5" strokeWidth="3" strokeLinecap="round" />}
    {kind === 'snow' && <g fill="#b8d5e8"><circle cx="61" cy="99" r="3"/><circle cx="83" cy="107" r="3"/><circle cx="106" cy="98" r="3"/></g>}
    {kind === 'storm' && <path d="m84 80-9 17h10l-7 16 23-25H88l6-8Z" fill="#edc47d" />}
    {kind === 'fog' && <path d="M44 96h75m-63 9h52" stroke="#aebaca" strokeWidth="3" strokeLinecap="round" />}
  </svg>
}
