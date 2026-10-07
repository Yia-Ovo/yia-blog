import { useCallback, useEffect, useState } from 'react'
import { weatherApi, weatherCities } from '../api/weather'
import type { WeatherCity } from '../api/weather'
import { useResource } from '../hooks/useResource'
import { useI18n } from '../i18n'
import WeatherScene, { weatherDescription } from './WeatherScene'

const cityNames: Record<Exclude<WeatherCity, ''>, string> = { nanning: '南宁', beijing: '北京', shanghai: '上海', guangzhou: '广州', chengdu: '成都', tokyo: '东京', hanoi: '河内', london: '伦敦', 'new-york': '纽约' }
const cityKey = 'yia.weather.city.v1'
function readCity(): WeatherCity {
  try { const city = localStorage.getItem(cityKey); return weatherCities.includes(city as Exclude<WeatherCity, ''>) ? city as WeatherCity : '' } catch { return '' }
}

export default function WeatherCard() {
  const { t, locale } = useI18n()
  const [city, setCity] = useState<WeatherCity>(readCity)
  const result = useResource(useCallback((signal) => weatherApi.current(city, signal), [city]))
  const data = result.data
  const weather = data?.status === 'ready' && !data.stale ? data.weather : null
  const condition = weather ? weatherDescription(weather.code) : null
  const label = city ? t(cityNames[city]) : data?.location?.city
  const number = (value: number) => new Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(value)
  const timestamp = weather ? new Date(weather.updated_at) : null
  const updated = timestamp && Number.isFinite(timestamp.getTime()) ? new Intl.DateTimeFormat(locale, { ...(timestamp.toDateString() !== new Date().toDateString() ? { month: 'short' as const, day: 'numeric' as const } : {}), hour: '2-digit', minute: '2-digit' }).format(timestamp) : ''

  useEffect(() => {
    let refreshedAt = Date.now()
    const refresh = () => {
      if (document.visibilityState === 'visible' && Date.now() - refreshedAt >= 15 * 60_000) {
        refreshedAt = Date.now(); result.reload()
      }
    }
    const timer = window.setInterval(refresh, 60_000)
    document.addEventListener('visibilitychange', refresh)
    return () => { window.clearInterval(timer); document.removeEventListener('visibilitychange', refresh) }
  }, [city, result.reload])

  function selectCity(value: string) {
    if (value !== '' && !weatherCities.includes(value as Exclude<WeatherCity, ''>)) return
    setCity(value as WeatherCity)
    try { localStorage.setItem(cityKey, value) } catch { /* 无法保存偏好时仍可切换城市。 */ }
  }
  return <section className="weather-card daily-card" aria-labelledby="weather-title" data-night={weather ? !weather.is_day : false}>
    <div className="weather-top"><h3 id="weather-title">{t('此刻天气')}</h3><label className="weather-city-picker"><span className="sr-only">{t('切换天气城市')}</span><select value={city} onChange={(event) => selectCity(event.target.value)}><option value="">{t('自动识别城市')}</option>{weatherCities.map((item) => <option value={item} key={item}>{t(cityNames[item])}</option>)}</select></label></div>
    {result.pending ? <div className="weather-empty" role="status"><span className="weather-loading-dot"/><p>{t(city ? '正在获取天气…' : '正在识别大致城市…')}</p></div>
      : weather && condition ? <>
        <div className="weather-location"><strong>{label}</strong><span>{t(city ? '手动选择' : 'IP 估算')}</span></div>
        <div className="weather-current"><div><div className="weather-temperature">{number(weather.temperature)}<span>°C</span></div><p>{t(condition.text)}<span>{number(weather.minimum)}° / {number(weather.maximum)}°</span></p></div><WeatherScene kind={condition.kind} day={weather.is_day}/></div>
        <div className="weather-metrics"><span>{t('湿度')} <strong>{number(weather.humidity)}%</strong></span><span>{t('风速')} <strong>{number(weather.wind_speed)} km/h</strong></span></div>
      </> : <div className="weather-empty"><WeatherScene kind="cloud" day/><p>{t(data?.reason === 'location_unavailable' ? '暂时无法识别城市，请手动选择。' : '天气暂时不可用，可以换个城市或稍后重试。')}</p></div>}
    <div className="weather-actions"><p className="weather-location-hint">{weather ? t(city ? '已使用你选择的城市。' : '按网络出口估算，城市不对可手动切换。') : ''}</p><button type="button" className="link-button" aria-label={t(weather || result.pending ? '刷新天气' : '重试')} aria-disabled={result.pending} onClick={() => { if (!result.pending) result.reload() }}>{t(weather || result.pending ? '刷新' : '重试')}</button></div>
    <span className="sr-only" aria-live="polite" aria-atomic="true">{result.pending ? '' : t(weather ? '天气已更新' : '天气暂时不可用，可以换个城市或稍后重试。')}</span>
    <div className="weather-credit"><span><a href="https://open-meteo.com/" target="_blank" rel="noreferrer">Open-Meteo</a> · {t('模型天气')}{!city && <> · <a href="https://ipwho.is/" target="_blank" rel="noreferrer">IPwho.is</a></>}</span>{updated && <span>{t('{time} 更新', { time: updated })}</span>}</div>
  </section>
}
