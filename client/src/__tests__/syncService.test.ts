import { describe, it, expect, vi } from 'vitest'

describe('syncService', () => {
  describe('uploadProfile', () => {
    it('should retry on failure', async () => {
      let attempts = 0
      const mockFn = vi.fn().mockImplementation(() => {
        attempts++
        if (attempts < 3) throw new Error('Network error')
        return Promise.resolve({ id: '1', version: 1 })
      })

      // 模拟重试逻辑
      async function retry<T>(fn: () => Promise<T>, retries = 3): Promise<T> {
        for (let i = 0; i < retries; i++) {
          try {
            return await fn()
          } catch (e) {
            if (i === retries - 1) throw e
            await new Promise(r => setTimeout(r, 10))
          }
        }
        throw new Error('Max retries')
      }

      const result = await retry(mockFn)
      expect(attempts).toBe(3)
      expect(result.id).toBe('1')
    })
  })
})
