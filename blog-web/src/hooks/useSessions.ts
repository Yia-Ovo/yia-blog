import { useSyncExternalStore } from 'react'
import { getSessions, subscribeSessions } from '../stores/session'

export function useSessions() {
  return useSyncExternalStore(subscribeSessions, getSessions)
}
