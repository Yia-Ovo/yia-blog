import { defineConfig } from 'vitest/config'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  test: { environment: './tests/environment.ts', include: ['tests/**/*.test.{ts,tsx}'], restoreMocks: true },
})
