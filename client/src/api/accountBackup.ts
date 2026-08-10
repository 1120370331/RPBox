import { request } from './request'

export interface AccountBackup {
  id: number
  user_id: number
  account_id: string
  profiles_data?: string
  profiles_count: number
  tools_data?: string
  tools_count: number
  runtime_data?: string
  runtime_size_kb: number
  config_data?: string
  extra_data?: string
  raw_trp3_lua?: string
  raw_trp3_data_lua?: string
  raw_trp3_extended_lua?: string
  checksum: string
  version: number
  created_at: string
  updated_at: string
}

export interface AccountBackupVersion {
  id: number
  backup_id: number
  version: number
  profiles_count?: number
  tools_count?: number
  runtime_size_kb?: number
  profiles_data?: string
  tools_data?: string
  runtime_data?: string
  config_data?: string
  extra_data?: string
  raw_trp3_lua?: string
  raw_trp3_data_lua?: string
  raw_trp3_extended_lua?: string
  checksum: string
  name?: string
  change_log: string
  created_at: string
}

export interface AccountBackupRestoreResult {
  backup: AccountBackup
  snapshot?: AccountBackupVersion
}

interface DataEnvelope<T> {
  data?: T
}

function unwrapData<T>(response: T | DataEnvelope<T>): T {
  if (response && typeof response === 'object' && 'data' in response) {
    const data = (response as DataEnvelope<T>).data
    if (data !== undefined) return data
  }
  return response as T
}

// 获取所有账号备份列表
export async function listAccountBackups(): Promise<AccountBackup[]> {
  const res = await request.get<{ backups?: AccountBackup[] }>('/account-backups')
  return res.backups || []
}

// 获取单个账号备份详情
export async function getAccountBackup(accountId: string): Promise<AccountBackup> {
  return request.get<AccountBackup>(`/account-backups/${encodeURIComponent(accountId)}`)
}

// 创建或更新账号备份
export async function upsertAccountBackup(data: {
  account_id: string
  profiles_data: string
  profiles_count: number
  tools_data?: string
  tools_count?: number
  runtime_data?: string
  runtime_size_kb?: number
  config_data?: string
  extra_data?: string
  raw_trp3_lua?: string
  raw_trp3_data_lua?: string
  raw_trp3_extended_lua?: string
  checksum: string
}): Promise<AccountBackup> {
  return request.post<AccountBackup>('/account-backups', data)
}

// 删除账号备份
export async function deleteAccountBackup(accountId: string): Promise<void> {
  await request.delete(`/account-backups/${encodeURIComponent(accountId)}`)
}

// 获取版本历史
export async function getAccountBackupVersions(accountId: string): Promise<AccountBackupVersion[]> {
  const res = await request.get<{ versions?: AccountBackupVersion[] } | DataEnvelope<{ versions?: AccountBackupVersion[] }>>(
    `/account-backups/${encodeURIComponent(accountId)}/versions`
  )
  return unwrapData(res).versions || []
}

export async function getAccountBackupVersion(
  accountId: string,
  versionId: number,
): Promise<AccountBackupVersion> {
  const response = await request.get<AccountBackupVersion | DataEnvelope<AccountBackupVersion>>(
    `/account-backups/${encodeURIComponent(accountId)}/versions/${versionId}`,
  )
  return unwrapData(response)
}

export async function renameAccountBackupVersion(
  accountId: string,
  versionId: number,
  name: string,
): Promise<AccountBackupVersion> {
  const response = await request.put<AccountBackupVersion | DataEnvelope<AccountBackupVersion>>(
    `/account-backups/${encodeURIComponent(accountId)}/versions/${versionId}`,
    { name },
  )
  return unwrapData(response)
}

export async function restoreAccountBackupVersion(
  accountId: string,
  versionId: number,
  snapshotName?: string,
): Promise<AccountBackupRestoreResult> {
  const response = await request.post<AccountBackupRestoreResult | DataEnvelope<AccountBackupRestoreResult>>(
    `/account-backups/${encodeURIComponent(accountId)}/versions/${versionId}/restore`,
    { snapshot_name: snapshotName || undefined },
  )
  return unwrapData(response)
}

export async function deleteAccountBackupVersion(accountId: string, versionId: number): Promise<void> {
  await request.delete(`/account-backups/${encodeURIComponent(accountId)}/versions/${versionId}`)
}
