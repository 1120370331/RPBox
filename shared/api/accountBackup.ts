
export interface AccountBackup {
  id: number
  user_id: number
  game_flavor?: string
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

interface VersionEnvelope {
  version?: AccountBackupVersion
}

export interface AccountBackupTransport {
  get<T>(url: string): Promise<T>
  post<T>(url: string, data: unknown): Promise<T>
  put<T>(url: string, data: unknown): Promise<T>
  delete<T>(url: string): Promise<T>
}

export function createAccountBackupApi(request: AccountBackupTransport) {
function unwrapData<T>(response: T | DataEnvelope<T>): T {
  if (response && typeof response === 'object' && 'data' in response) {
    const data = (response as DataEnvelope<T>).data
    if (data !== undefined) return data
  }
  return response as T
}

// 获取所有账号备份列表
async function listAccountBackups(): Promise<AccountBackup[]> {
  const res = await request.get<{ backups?: AccountBackup[] }>('/account-backups')
  return res.backups || []
}

// 获取单个账号备份详情
async function getAccountBackup(accountId: string, gameFlavor = ''): Promise<AccountBackup> {
  return request.get<AccountBackup>(`/account-backups/${encodeURIComponent(accountId)}?game_flavor=${encodeURIComponent(gameFlavor)}`)
}

// 创建或更新账号备份
async function upsertAccountBackup(data: {
  game_flavor?: string
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
  snapshot_reason?: 'before_manual_backup' | 'restore_sync' | 'sync'
}): Promise<AccountBackup> {
  return request.post<AccountBackup>('/account-backups', data)
}

// 删除账号备份
async function deleteAccountBackup(accountId: string, gameFlavor = ''): Promise<void> {
  await request.delete(`/account-backups/${encodeURIComponent(accountId)}?game_flavor=${encodeURIComponent(gameFlavor)}`)
}

// 获取版本历史
async function getAccountBackupVersions(accountId: string, gameFlavor = ''): Promise<AccountBackupVersion[]> {
  const res = await request.get<{ versions?: AccountBackupVersion[] } | DataEnvelope<{ versions?: AccountBackupVersion[] }>>(
    `/account-backups/${encodeURIComponent(accountId)}/versions?game_flavor=${encodeURIComponent(gameFlavor)}`
  )
  return unwrapData(res).versions || []
}

async function getAccountBackupVersion(
  accountId: string,
  versionId: number,
  gameFlavor = '',
): Promise<AccountBackupVersion> {
  const response = await request.get<AccountBackupVersion | VersionEnvelope | DataEnvelope<AccountBackupVersion | VersionEnvelope>>(
    `/account-backups/${encodeURIComponent(accountId)}/versions/${versionId}?game_flavor=${encodeURIComponent(gameFlavor)}`,
  )
  const unwrapped = unwrapData(response)
  if (unwrapped && typeof unwrapped === 'object' && 'version' in unwrapped && typeof unwrapped.version === 'object' && unwrapped.version) {
    return unwrapped.version
  }
  return unwrapped as AccountBackupVersion
}

async function renameAccountBackupVersion(
  accountId: string,
  versionId: number,
  name: string,
  gameFlavor = '',
): Promise<AccountBackupVersion> {
  const response = await request.put<AccountBackupVersion | VersionEnvelope | DataEnvelope<AccountBackupVersion | VersionEnvelope>>(
    `/account-backups/${encodeURIComponent(accountId)}/versions/${versionId}?game_flavor=${encodeURIComponent(gameFlavor)}`,
    { name },
  )
  const unwrapped = unwrapData(response)
  if (unwrapped && typeof unwrapped === 'object' && 'version' in unwrapped && typeof unwrapped.version === 'object' && unwrapped.version) {
    return unwrapped.version
  }
  return unwrapped as AccountBackupVersion
}

async function restoreAccountBackupVersion(
  accountId: string,
  versionId: number,
  snapshotName?: string,
  gameFlavor = '',
): Promise<AccountBackupRestoreResult> {
  const response = await request.post<AccountBackupRestoreResult | DataEnvelope<AccountBackupRestoreResult>>(
    `/account-backups/${encodeURIComponent(accountId)}/versions/${versionId}/restore?game_flavor=${encodeURIComponent(gameFlavor)}`,
    { snapshot_name: snapshotName || undefined },
  )
  return unwrapData(response)
}

async function deleteAccountBackupVersion(accountId: string, versionId: number, gameFlavor = ''): Promise<void> {
  await request.delete(`/account-backups/${encodeURIComponent(accountId)}/versions/${versionId}?game_flavor=${encodeURIComponent(gameFlavor)}`)
}

async function ensureAccountBackup(accountId: string, gameFlavor: string): Promise<AccountBackup> {
  return request.post<AccountBackup>('/account-backup-targets', { account_id: accountId, game_flavor: gameFlavor })
}

async function labelLegacyAccountBackup(accountId: string, gameFlavor: string): Promise<AccountBackup> {
  return request.put(`/account-backups/${encodeURIComponent(accountId)}/game-flavor`, { game_flavor: gameFlavor })
}

 return { listAccountBackups, getAccountBackup, upsertAccountBackup, deleteAccountBackup, getAccountBackupVersions, getAccountBackupVersion, renameAccountBackupVersion, restoreAccountBackupVersion, deleteAccountBackupVersion, ensureAccountBackup, labelLegacyAccountBackup }
}
