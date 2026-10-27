import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

export default defineConfig({
  plugins: [react()],
  server: {
    // The browser talks to the Vite server; /api goes to the Go backend. No CORS needed.
    proxy: { '/api': 'http://localhost:8080' },
  },
  test: {
    environment: 'jsdom',
    setupFiles: ['./src/test/setup.ts'],
    coverage: {
      provider: 'v8',
      include: ['src/**/*.{ts,tsx}'],
      exclude: ['src/main.tsx', 'src/api/schema.d.ts', 'src/test/**', 'src/**/*.test.{ts,tsx}'],
      thresholds: { statements: 90, branches: 85, functions: 90, lines: 90 },
    },
  },
})
