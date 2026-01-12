export interface ConflictInfo {
  profileId: string
  profileName: string
  localModifiedAt: string
  cloudModifiedAt: string
  localChecksum: string
  cloudChecksum: string
}

export type ConflictResolution = 'local' | 'cloud'
