import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

const devApiTarget = process.env.RPBOX_DEV_API_TARGET || 'http://127.0.0.1:9010'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
    },
  },
  clearScreen: false,
  server: {
    host: '127.0.0.1',
    port: 3102,
    strictPort: true,
    proxy: {
      '/api': {
        target: devApiTarget,
        changeOrigin: true,
      },
      '/uploads': {
        target: devApiTarget,
        changeOrigin: true,
      },
      '/releases': {
        target: devApiTarget,
        changeOrigin: true,
      },
      '/emotes': {
        target: devApiTarget,
        changeOrigin: true,
      },
    },
  },
  envPrefix: ['VITE_', 'TAURI_'],
  test: {
    globals: true,
    environment: 'happy-dom',
  },
})
