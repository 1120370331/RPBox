import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src'),
      // 替换 client 的 request.ts 为 web 端版本（使用 web 端 router）
      [resolve(__dirname, '../client/src/api/request.ts')]: resolve(__dirname, 'src/api/request.ts'),
      [resolve(__dirname, '../client/src/api/request')]: resolve(__dirname, 'src/api/request.ts'),
      // 引用 client 的共享代码
      '@shared/api': resolve(__dirname, '../client/src/api'),
      '@shared/components': resolve(__dirname, '../client/src/components'),
      '@shared/stores': resolve(__dirname, '../client/src/stores'),
      '@shared/utils': resolve(__dirname, '../client/src/utils'),
      '@shared/composables': resolve(__dirname, '../client/src/composables'),
      '@shared/styles': resolve(__dirname, '../client/src/styles'),
    }
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8081',
        changeOrigin: true
      }
    }
  }
})
