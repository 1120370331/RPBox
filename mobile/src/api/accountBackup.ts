import { request } from '@shared/api/request'
import { createAccountBackupApi } from '../../../shared/api/accountBackup'
export type { AccountBackup, AccountBackupVersion, AccountBackupRestoreResult } from '../../../shared/api/accountBackup'
export const { listAccountBackups, getAccountBackup, upsertAccountBackup, deleteAccountBackup, getAccountBackupVersions, getAccountBackupVersion, renameAccountBackupVersion, restoreAccountBackupVersion, deleteAccountBackupVersion, ensureAccountBackup, labelLegacyAccountBackup } = createAccountBackupApi(request)
