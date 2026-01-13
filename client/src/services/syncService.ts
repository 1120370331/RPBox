import * as profileApi from '../api/profile'
import type { ProfileData, CloudProfile } from '../api/profile'

export interface SyncProgress {
  total: number
  completed: number
  current?: string
  failed: string[]
}

export type ProgressCallback = (progress: SyncProgress) => void

const MAX_RETRIES = 3
const CONCURRENT_LIMIT = 3

async function retry<T>(fn: () => Promise<T>, retries = MAX_RETRIES): Promise<T> {
  for (let i = 0; i < retries; i++) {
    try {
      return await fn()
    } catch (e) {
      if (i === retries - 1) throw e
      await new Promise(r => setTimeout(r, 1000 * (i + 1)))
    }
  }
  throw new Error('Max retries exceeded')
}

export async function uploadProfile(data: ProfileData): Promise<CloudProfile> {
  console.log(`[Upload] 上传 ${data.profile_name}(${data.id}), checksum: ${data.checksum}`)
  return retry(async () => {
    try {
      const result = await profileApi.updateProfile(data.id, data)
      console.log(`[Upload] ${data.id} 更新成功, 返回checksum: ${result.checksum}`)
      return result
    } catch (e) {
      console.log(`[Upload] ${data.id} 更新失败，尝试创建...`, e)
      const result = await profileApi.createProfile(data)
      console.log(`[Upload] ${data.id} 创建成功, 返回checksum: ${result.checksum}`)
      return result
    }
  })
}

export async function uploadProfiles(
  profiles: ProfileData[],
  onProgress?: ProgressCallback
): Promise<{ success: CloudProfile[]; failed: string[] }> {
  const progress: SyncProgress = {
    total: profiles.length,
    completed: 0,
    failed: [],
  }

  const success: CloudProfile[] = []
  const queue = [...profiles]

  async function processOne(): Promise<void> {
    const item = queue.shift()
    if (!item) return

    progress.current = item.profile_name
    onProgress?.(progress)

    try {
      const result = await uploadProfile(item)
      success.push(result)
    } catch {
      progress.failed.push(item.id)
    }

    progress.completed++
    onProgress?.(progress)
    await processOne()
  }

  const workers = Array(Math.min(CONCURRENT_LIMIT, profiles.length))
    .fill(null)
    .map(() => processOne())

  await Promise.all(workers)

  return { success, failed: progress.failed }
}
