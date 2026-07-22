export interface TRP3Info {
  FN?: string
  LN?: string
  TI?: string
  IC?: string
  CH?: string
}

export interface ProfileSnapshot extends TRP3Info {
  ref?: string
  gameID?: string
  n?: string
  pn?: string
  at?: number
  rev?: number
}

export interface Listener {
  gameID: string
  profileID?: string
  snapshot_id?: string
  snapshot?: ProfileSnapshot
}

export interface IdentityEndpoint {
  ref_id?: string
  snapshot_id?: string
  display_name?: string
  profile_name?: string
}

export interface IdentityEvent {
  kind: 'profile_switch' | 'profile_update' | string
  certainty: 'exact' | 'observed' | string
  from?: IdentityEndpoint
  to?: IdentityEndpoint
}

export interface ChatRecord {
  record_key: string
  account_id: string
  record_id?: string
  schema_version?: number
  session_id?: string
  sequence?: number
  timestamp: number
  channel: string
  sender: {
    gameID: string
    trp3?: TRP3Info
  }
  content: string
  mark?: string
  npc?: string
  nt?: string
  ref_id?: string
  profile_snapshot_id?: string
  profile_snapshot?: ProfileSnapshot
  identity_source: 'snapshot' | 'legacy_cache' | 'embedded_legacy' | 'game_id' | string
  event?: IdentityEvent
  raw_profile?: string
  listeners?: Listener[]
}

export interface AccountChatLogs {
  account_id: string
  last_update: number | null
  record_count: number
  records: ChatRecord[]
}
