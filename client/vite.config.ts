import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  clearScreen: false,
  server: {
    port: 3101,
    strictPort: true,
  },
  envPrefix: ['VITE_', 'TAURI_'],
  test: {
    globals: true,
    environment: 'happy-dom',
  },
})
