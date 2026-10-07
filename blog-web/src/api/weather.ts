import { request } from './http'

export const weatherCities = ['nanning', 'beijing', 'shanghai', 'guangzhou', 'chengdu', 'tokyo', 'hanoi', 'london', 'new-york'] as const
export type WeatherCity = '' | typeof weatherCities[number]
export interface WeatherResult {
  status: 'ready' | 'unavailable'
  location: { city: string; region: string; country: string; source: 'ip' | 'manual' } | null
  weather: { temperature: number; code: number; is_day: boolean; humidity: number; wind_speed: number; minimum: number; maximum: number; updated_at: string } | null
  stale: boolean
  reason?: 'location_unavailable' | 'weather_unavailable' | 'busy'
}
export const weatherApi = {
  current: (city: WeatherCity, signal?: AbortSignal) => request<WeatherResult>('/weather/current', { query: { city: city || undefined }, signal }),
}
