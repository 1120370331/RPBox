import { defineConfig, loadEnv } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, __dirname, '')
  const proxyTarget = env.VITE_API_PROXY_TARGET || 'http://localhost:8080'

  return {
    plugins: [vue()],
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src'),
        '@shared': resolve(__dirname, '../shared'),
        '@client': resolve(__dirname, '../client/src'),
      },
    },
    server: {
      port: 3102,
      fs: {
        allow: [resolve(__dirname, '..')],
      },
      proxy: {
        '/api': {
          target: proxyTarget,
          changeOrigin: true,
        },
      },
    },
    envPrefix: ['VITE_'],
    test: {
      environment: 'jsdom',
      include: ['src/**/*.test.ts'],
    },
  }
})
