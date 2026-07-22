<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { invoke } from '@tauri-apps/api/core'
import RButton from '@/components/RButton.vue'
import RCheckbox from '@/components/RCheckbox.vue'
import REmpty from '@/components/REmpty.vue'
import WowIcon from '@/components/WowIcon.vue'
import type {
  AccountChatLogs,
  ChatRecord,
  IdentityEndpoint,
  Listener,
  ProfileSnapshot,
} from '@/types/chatLog'

const { t } = useI18n()

interface Props {
  active?: boolean
}

const props = withDefaults(defineProps<Props>(), {
  active: true,
})

interface ProfileOption {
  key: string
  name: string
  detail: string
  legacy: boolean
  count: number
}

type RecordKind = 'dialogue' | 'npc' | 'background' | 'identity'
type SortDirection = 'newest' | 'oldest'

const recordKinds: RecordKind[] = ['dialogue', 'npc', 'background', 'identity']

const emit = defineEmits<{
  archive: [records: ChatRecord[]]
}>()

const loading = ref(false)
const syncError = ref('')
const accounts = ref<AccountChatLogs[]>([])
const selectedRecords = ref<Set<string>>(new Set())
const expandedDates = ref<Set<string>>(new Set())
const expandedHours = ref<Set<string>>(new Set())
const hourPageStarts = ref<Map<string, number>>(new Map())

const filterSearch = ref('')
const filterStartDate = ref('')
const filterEndDate = ref('')
const filterAccounts = ref<Set<string>>(new Set())
const filterRecordKinds = ref<Set<RecordKind>>(new Set())
const filterChannels = ref<Set<string>>(new Set())
const filterSenderProfiles = ref<Set<string>>(new Set())
const filterListenerProfiles = ref<Set<string>>(new Set())
const senderProfileSearch = ref('')
const listenerProfileSearch = ref('')
const sortDirection = ref<SortDirection>('newest')
const selectionAnchor = ref('')

const ARCHIVED_KEYS_STORAGE = 'rpbox_archived_record_keys_v2'
const LEGACY_ARCHIVED_STORAGE = 'archived_timestamps'
const HOUR_RECORD_BATCH_SIZE = 120
const PROFILE_OPTION_RENDER_LIMIT = 160

function readStoredSet<T>(key: string): Set<T> {
  try {
    const value = JSON.parse(localStorage.getItem(key) || '[]')
    return new Set(Array.isArray(value) ? value : [])
  } catch {
    return new Set()
  }
}

const archivedRecordKeys = ref<Set<string>>(readStoredSet<string>(ARCHIVED_KEYS_STORAGE))
const legacyArchivedTimestamps = readStoredSet<number>(LEGACY_ARCHIVED_STORAGE)

function saveArchivedKeys() {
  localStorage.setItem(ARCHIVED_KEYS_STORAGE, JSON.stringify([...archivedRecordKeys.value]))
}

function fallbackRecordKey(accountID: string, record: ChatRecord, ordinal: number): string {
  const seed = [
    accountID,
    record.timestamp,
    ordinal,
    record.channel,
    record.sender?.gameID || '',
    record.content || '',
  ].join('|')
  let hash = 2166136261
  for (let i = 0; i < seed.length; i += 1) {
    hash ^= seed.charCodeAt(i)
    hash = Math.imul(hash, 16777619)
  }
  return `legacy-client-${(hash >>> 0).toString(16).padStart(8, '0')}`
}

function normalizeAccounts(logs: AccountChatLogs[]): AccountChatLogs[] {
  let migrated = false
  const normalized = logs.map(account => ({
    ...account,
    records: account.records.map((record, ordinal) => {
      const normalizedRecord: ChatRecord = {
        ...record,
        account_id: record.account_id || account.account_id,
        record_key: record.record_key || fallbackRecordKey(account.account_id, record, ordinal),
        identity_source: record.identity_source || (record.profile_snapshot ? 'snapshot' : 'game_id'),
      }
      if (legacyArchivedTimestamps.has(normalizedRecord.timestamp)) {
        archivedRecordKeys.value.add(normalizedRecord.record_key)
        migrated = true
      }
      return normalizedRecord
    }),
  }))
  if (migrated) {
    saveArchivedKeys()
    legacyArchivedTimestamps.clear()
    localStorage.removeItem(LEGACY_ARCHIVED_STORAGE)
  }
  return normalized
}

const allRecords = computed(() => accounts.value.flatMap(account => account.records))
const unarchivedRecords = computed(() => allRecords.value
  .filter(record => !archivedRecordKeys.value.has(record.record_key)))

const availableAccounts = computed(() => accounts.value
  .map(account => ({
    id: account.account_id,
    count: account.records.filter(record => !archivedRecordKeys.value.has(record.record_key)).length,
  }))
  .filter(account => account.count > 0)
  .sort((a, b) => a.id.localeCompare(b.id)))

function getSelectedRecords(): ChatRecord[] {
  return visibleRecords.value
    .filter(record => selectedRecords.value.has(record.record_key))
    .sort((a, b) => (
      a.timestamp - b.timestamp
      || (a.sequence ?? 0) - (b.sequence ?? 0)
      || a.record_key.localeCompare(b.record_key)
    ))
}

function clearSelection() {
  selectedRecords.value = new Set()
  selectionAnchor.value = ''
}

function removeArchivedRecords(recordKeys: string[]) {
  for (const key of recordKeys) {
    archivedRecordKeys.value.add(key)
    selectedRecords.value.delete(key)
  }
  archivedRecordKeys.value = new Set(archivedRecordKeys.value)
  selectedRecords.value = new Set(selectedRecords.value)
  reconcileExpandedRecordWindow()
  saveArchivedKeys()
}

function cleanWowText(text: string): string {
  return text
    .replace(/\{[^}]+\}/g, '')
    .replace(/\|c[0-9a-fA-F]{8}/g, '')
    .replace(/\|r/g, '')
    .replace(/\|T[^|]+\|t/g, '')
    .replace(/\|H[^|]+\|h/g, '')
    .replace(/\|h/g, '')
    .replace(/[\uE000-\uF8FF]/g, '')
    .replace(/\uFFFD/g, '')
    .replace(/[\u0000-\u001F]/g, '')
    .trim()
}

function snapshotDisplayName(snapshot?: ProfileSnapshot): string {
  if (!snapshot) return ''
  if (snapshot.n) return cleanWowText(snapshot.n)
  if (snapshot.FN) return cleanWowText(snapshot.LN ? `${snapshot.FN} ${snapshot.LN}` : snapshot.FN)
  return ''
}

function endpointName(endpoint?: IdentityEndpoint): string {
  if (!endpoint) return t('archives.staging.unknownProfile')
  return cleanWowText(
    endpoint.display_name
    || endpoint.profile_name
    || endpoint.ref_id
    || t('archives.staging.unknownProfile'),
  )
}

function senderProfileKeys(record: ChatRecord): string[] {
  const keys: string[] = []
  if (record.ref_id) keys.push(`ref:${record.ref_id}`)
  else if (record.profile_snapshot_id) keys.push(`snapshot:${record.profile_snapshot_id}`)
  else if (record.sender.gameID && record.mark !== 'N' && record.mark !== 'B') {
    keys.push(`game:${record.sender.gameID}`)
  }
  for (const endpoint of [record.event?.from, record.event?.to]) {
    const key = endpointProfileKey(endpoint)
    if (key) keys.push(key)
  }
  return [...new Set(keys)]
}

function endpointProfileKey(endpoint?: IdentityEndpoint): string {
  if (endpoint?.ref_id) return `ref:${endpoint.ref_id}`
  if (endpoint?.snapshot_id) return `snapshot:${endpoint.snapshot_id}`
  return ''
}

function listenerProfileKey(listener: Listener): string {
  if (listener.profileID) return `ref:${listener.profileID}`
  if (listener.snapshot_id) return `snapshot:${listener.snapshot_id}`
  return `game:${listener.gameID}`
}

function optionFromRecord(record: ChatRecord, key: string): ProfileOption {
  const snapshotName = snapshotDisplayName(record.profile_snapshot)
  const legacyTRPName = record.sender.trp3?.FN
    ? cleanWowText(record.sender.trp3.LN
      ? `${record.sender.trp3.FN} ${record.sender.trp3.LN}`
      : record.sender.trp3.FN)
    : ''
  const gameName = cleanWowText(record.sender.gameID.split('-')[0] || record.sender.gameID)
  const profileName = cleanWowText(record.profile_snapshot?.pn || '')
  return {
    key,
    name: profileName || snapshotName || legacyTRPName || gameName || t('archives.staging.unknownProfile'),
    detail: [
      profileName && snapshotName !== profileName ? snapshotName : '',
      record.ref_id
        || (record.identity_source === 'game_id' ? t('archives.staging.noProfileCard') : ''),
    ].filter(Boolean).join(' · '),
    legacy: record.identity_source !== 'snapshot',
    count: 1,
  }
}

function mergeProfileOption(map: Map<string, ProfileOption>, option: ProfileOption) {
  const current = map.get(option.key)
  if (!current) {
    map.set(option.key, option)
    return
  }
  const count = current.count + option.count
  if ((current.legacy && !option.legacy) || (!current.detail && option.detail)) {
    map.set(option.key, { ...option, count })
  } else {
    map.set(option.key, { ...current, count })
  }
}

const availableSenderProfiles = computed(() => {
  const profiles = new Map<string, ProfileOption>()
  for (const record of unarchivedRecords.value) {
    for (const key of senderProfileKeys(record)) {
      if (record.event) {
        const endpoint = [record.event.from, record.event.to]
          .find(item => endpointProfileKey(item) === key)
        if (endpoint) {
          mergeProfileOption(profiles, {
            key,
            name: cleanWowText(endpoint.profile_name || '') || endpointName(endpoint),
            detail: [
              endpoint.profile_name ? endpoint.display_name : '',
              endpoint.ref_id,
            ].filter(Boolean).join(' · '),
            legacy: false,
            count: 1,
          })
        } else {
          mergeProfileOption(profiles, optionFromRecord(record, key))
        }
      } else {
        mergeProfileOption(profiles, optionFromRecord(record, key))
      }
    }
  }
  return [...profiles.values()].sort((a, b) => a.name.localeCompare(b.name))
})

const availableListenerProfiles = computed(() => {
  const profiles = new Map<string, ProfileOption>()
  for (const record of unarchivedRecords.value) {
    const seen = new Set<string>()
    for (const listener of record.listeners || []) {
      const key = listenerProfileKey(listener)
      if (seen.has(key)) continue
      seen.add(key)
      const name = snapshotDisplayName(listener.snapshot)
        || cleanWowText(listener.gameID.split('-')[0] || listener.gameID)
      const profileName = cleanWowText(listener.snapshot?.pn || '')
      mergeProfileOption(profiles, {
        key,
        name: profileName || name || t('archives.staging.unknownProfile'),
        detail: [
          profileName && name !== profileName ? name : '',
          listener.profileID || '',
        ].filter(Boolean).join(' · '),
        legacy: !listener.snapshot_id,
        count: 1,
      })
    }
  }
  return [...profiles.values()].sort((a, b) => a.name.localeCompare(b.name))
})

function profileMatchesSearch(profile: ProfileOption, search: string): boolean {
  const query = search.trim().toLocaleLowerCase()
  if (!query) return true
  return [profile.name, profile.detail, profile.key]
    .filter(Boolean)
    .join(' ')
    .toLocaleLowerCase()
    .includes(query)
}

const filteredSenderProfiles = computed(() => availableSenderProfiles.value
  .filter(profile => profileMatchesSearch(profile, senderProfileSearch.value)))
const filteredListenerProfiles = computed(() => availableListenerProfiles.value
  .filter(profile => profileMatchesSearch(profile, listenerProfileSearch.value)))

function boundedProfileOptions(profiles: ProfileOption[], selected: Set<string>): ProfileOption[] {
  return [...profiles]
    .sort((left, right) => Number(selected.has(right.key)) - Number(selected.has(left.key)))
    .slice(0, PROFILE_OPTION_RENDER_LIMIT)
}

const visibleSenderProfiles = computed(() => boundedProfileOptions(filteredSenderProfiles.value, filterSenderProfiles.value))
const visibleListenerProfiles = computed(() => boundedProfileOptions(filteredListenerProfiles.value, filterListenerProfiles.value))
const hiddenSenderProfileCount = computed(() => Math.max(0, filteredSenderProfiles.value.length - visibleSenderProfiles.value.length))
const hiddenListenerProfileCount = computed(() => Math.max(0, filteredListenerProfiles.value.length - visibleListenerProfiles.value.length))

function localDateKey(timestamp: number): string {
  const date = new Date(timestamp * 1000)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function inputDateKey(date: Date): string {
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
}

function recordKind(record: ChatRecord): RecordKind {
  if (record.event || record.mark === 'S') return 'identity'
  if (record.mark === 'N' && record.npc) return 'npc'
  if (record.mark === 'B' || record.mark === 'N') return 'background'
  return 'dialogue'
}

const dateRangeInvalid = computed(() => Boolean(
  filterStartDate.value
  && filterEndDate.value
  && filterStartDate.value > filterEndDate.value,
))

const activeDatePreset = computed<'today' | 'week' | 'all' | 'custom'>(() => {
  const today = new Date()
  const todayKey = inputDateKey(today)
  const weekStart = new Date(today)
  weekStart.setDate(weekStart.getDate() - 6)
  if (!filterStartDate.value && !filterEndDate.value) return 'all'
  if (filterStartDate.value === todayKey && filterEndDate.value === todayKey) return 'today'
  if (filterStartDate.value === inputDateKey(weekStart) && filterEndDate.value === todayKey) return 'week'
  return 'custom'
})

function setDatePreset(preset: 'today' | 'week' | 'all') {
  if (preset === 'all') {
    filterStartDate.value = ''
    filterEndDate.value = ''
    return
  }
  const today = new Date()
  filterEndDate.value = inputDateKey(today)
  if (preset === 'today') {
    filterStartDate.value = filterEndDate.value
    return
  }
  const weekStart = new Date(today)
  weekStart.setDate(weekStart.getDate() - 6)
  filterStartDate.value = inputDateKey(weekStart)
}

function recordSearchText(record: ChatRecord): string {
  const pieces = [
    record.content,
    record.npc,
    record.sender.gameID,
    snapshotDisplayName(record.profile_snapshot),
    record.profile_snapshot?.pn,
    record.ref_id,
    record.event?.from ? endpointName(record.event.from) : '',
    record.event?.to ? endpointName(record.event.to) : '',
  ]
  for (const listener of record.listeners || []) {
    pieces.push(listener.gameID, snapshotDisplayName(listener.snapshot), listener.snapshot?.pn, listener.profileID)
  }
  return pieces.filter(Boolean).join(' ').toLocaleLowerCase()
}

function matchesSet(keys: string[], selected: Set<string>): boolean {
  return selected.size === 0 || keys.some(key => selected.has(key))
}

const groupedRecords = computed(() => {
  const groups: Record<string, Record<string, ChatRecord[]>> = {}
  const query = filterSearch.value.trim().toLocaleLowerCase()

  for (const record of allRecords.value) {
    if (archivedRecordKeys.value.has(record.record_key)) continue
    if (dateRangeInvalid.value) continue
    const dateStr = localDateKey(record.timestamp)
    if (filterStartDate.value && dateStr < filterStartDate.value) continue
    if (filterEndDate.value && dateStr > filterEndDate.value) continue
    if (filterAccounts.value.size > 0 && !filterAccounts.value.has(record.account_id)) continue
    if (filterRecordKinds.value.size > 0 && !filterRecordKinds.value.has(recordKind(record))) continue
    if (filterChannels.value.size > 0 && !filterChannels.value.has(normalizeChannel(record.channel))) continue
    if (!matchesSet(senderProfileKeys(record), filterSenderProfiles.value)) continue

    if (filterListenerProfiles.value.size > 0) {
      const listenerKeys = (record.listeners || []).map(listenerProfileKey)
      if (!matchesSet(listenerKeys, filterListenerProfiles.value)) continue
    }
    if (query && !recordSearchText(record).includes(query)) continue

    const hourStr = new Date(record.timestamp * 1000).getHours().toString().padStart(2, '0')
    if (!groups[dateStr]) groups[dateStr] = {}
    if (!groups[dateStr][hourStr]) groups[dateStr][hourStr] = []
    groups[dateStr][hourStr].push(record)
  }

  for (const date of Object.values(groups)) {
    for (const records of Object.values(date)) {
      records.sort((a, b) => (
        (sortDirection.value === 'newest' ? b.timestamp - a.timestamp : a.timestamp - b.timestamp)
        || (sortDirection.value === 'newest'
          ? (b.sequence ?? 0) - (a.sequence ?? 0)
          : (a.sequence ?? 0) - (b.sequence ?? 0))
        || (sortDirection.value === 'newest'
          ? b.record_key.localeCompare(a.record_key)
          : a.record_key.localeCompare(b.record_key))
      ))
    }
  }
  return groups
})

const totalRecords = computed(() => Object.values(groupedRecords.value)
  .reduce((total, date) => total + Object.values(date).flat().length, 0))
const orderedDates = computed(() => Object.keys(groupedRecords.value)
  .sort((a, b) => sortDirection.value === 'newest' ? b.localeCompare(a) : a.localeCompare(b)))

function orderedHours(date: string): string[] {
  return Object.keys(groupedRecords.value[date] || {})
    .sort((a, b) => sortDirection.value === 'newest' ? b.localeCompare(a) : a.localeCompare(b))
}

const visibleRecords = computed(() => orderedDates.value.flatMap(date => (
  orderedHours(date).flatMap(hour => groupedRecords.value[date][hour])
)))
const renderedVisibleRecords = computed(() => orderedDates.value.flatMap((date) => {
  if (!expandedDates.value.has(date)) return []
  return orderedHours(date).flatMap((hour) => {
    if (!expandedHours.value.has(`${date}-${hour}`)) return []
    return renderedHourRecords(date, hour)
  })
}))
const unarchivedRecordCount = computed(() => unarchivedRecords.value.length)
const selectedCount = computed(() => getSelectedRecords().length)
const activeFilterCount = computed(() => (
  Number(Boolean(filterSearch.value.trim()))
  + Number(Boolean(filterStartDate.value))
  + Number(Boolean(filterEndDate.value))
  + filterAccounts.value.size
  + filterRecordKinds.value.size
  + filterChannels.value.size
  + filterSenderProfiles.value.size
  + filterListenerProfiles.value.size
))

function toggledSet<T>(current: Set<T>, value: T): Set<T> {
  const next = new Set(current)
  if (next.has(value)) next.delete(value)
  else next.add(value)
  return next
}

function toggleAccount(value: string) {
  filterAccounts.value = toggledSet(filterAccounts.value, value)
}

function toggleRecordKind(value: RecordKind) {
  filterRecordKinds.value = toggledSet(filterRecordKinds.value, value)
}

function toggleChannel(value: string) {
  filterChannels.value = toggledSet(filterChannels.value, value)
}

function toggleSenderProfile(value: string) {
  filterSenderProfiles.value = toggledSet(filterSenderProfiles.value, value)
}

function toggleListenerProfile(value: string) {
  filterListenerProfiles.value = toggledSet(filterListenerProfiles.value, value)
}

function toggleExpandedDate(key: string) {
  if (expandedDates.value.has(key)) {
    expandedDates.value = new Set()
  } else {
    expandedDates.value = new Set([key])
  }
  expandedHours.value = new Set()
  selectionAnchor.value = ''
}

function toggleExpandedHour(key: string) {
  if (expandedHours.value.has(key)) {
    expandedHours.value = new Set()
    selectionAnchor.value = ''
    return
  }
  expandedHours.value = new Set([key])
  if (!hourPageStarts.value.has(key)) {
    const next = new Map(hourPageStarts.value)
    next.set(key, 0)
    hourPageStarts.value = next
  }
  selectionAnchor.value = ''
}

function collapseAll() {
  expandedDates.value = new Set()
  expandedHours.value = new Set()
  hourPageStarts.value = new Map()
}

function expandFirstMatch() {
  const date = orderedDates.value[0]
  if (!date) {
    collapseAll()
    return
  }
  const hour = orderedHours(date)[0]
  expandedDates.value = new Set([date])
  expandedHours.value = hour ? new Set([`${date}-${hour}`]) : new Set()
  hourPageStarts.value = hour ? new Map([[`${date}-${hour}`, 0]]) : new Map()
}

function reconcileExpandedRecordWindow() {
  const expandedDate = [...expandedDates.value][0]
  if (!expandedDate) return
  if (!groupedRecords.value[expandedDate]) {
    expandFirstMatch()
    return
  }

  const hours = orderedHours(expandedDate)
  const expandedHourKey = [...expandedHours.value][0]
  const expandedHour = expandedHourKey?.startsWith(`${expandedDate}-`)
    ? expandedHourKey.slice(expandedDate.length + 1)
    : ''
  const hour = hours.includes(expandedHour) ? expandedHour : hours[0]
  if (!hour) {
    expandedHours.value = new Set()
    hourPageStarts.value = new Map()
    return
  }

  const key = `${expandedDate}-${hour}`
  const total = getHourRecords(expandedDate, hour).length
  const maxStart = Math.max(0, Math.floor((total - 1) / HOUR_RECORD_BATCH_SIZE) * HOUR_RECORD_BATCH_SIZE)
  expandedHours.value = new Set([key])
  hourPageStarts.value = new Map([[key, Math.min(hourPageStarts.value.get(key) || 0, maxStart)]])
  selectionAnchor.value = ''
}

function renderedHourRecords(date: string, hour: string): ChatRecord[] {
  const key = `${date}-${hour}`
  const start = hourPageStarts.value.get(key) || 0
  return getHourRecords(date, hour).slice(start, start + HOUR_RECORD_BATCH_SIZE)
}

function hourPageStart(date: string, hour: string): number {
  return (hourPageStarts.value.get(`${date}-${hour}`) || 0) + 1
}

function hourPageEnd(date: string, hour: string): number {
  return (hourPageStarts.value.get(`${date}-${hour}`) || 0) + renderedHourRecords(date, hour).length
}

function hasPreviousHourPage(date: string, hour: string): boolean {
  return (hourPageStarts.value.get(`${date}-${hour}`) || 0) > 0
}

function hasNextHourPage(date: string, hour: string): boolean {
  return hourPageEnd(date, hour) < getHourRecords(date, hour).length
}

function changeHourPage(date: string, hour: string, direction: -1 | 1) {
  const key = `${date}-${hour}`
  const current = hourPageStarts.value.get(key) || 0
  const maxStart = Math.max(0, Math.floor((getHourRecords(date, hour).length - 1) / HOUR_RECORD_BATCH_SIZE) * HOUR_RECORD_BATCH_SIZE)
  const nextStart = Math.min(maxStart, Math.max(0, current + direction * HOUR_RECORD_BATCH_SIZE))
  const next = new Map(hourPageStarts.value)
  next.set(key, nextStart)
  hourPageStarts.value = next
  selectionAnchor.value = ''
}

function selectAllMatches() {
  selectedRecords.value = new Set(visibleRecords.value.map(record => record.record_key))
  selectionAnchor.value = renderedVisibleRecords.value.at(-1)?.record_key || visibleRecords.value.at(-1)?.record_key || ''
}

function invertMatchSelection() {
  const next = new Set<string>()
  for (const record of visibleRecords.value) {
    if (!selectedRecords.value.has(record.record_key)) next.add(record.record_key)
  }
  selectedRecords.value = next
  selectionAnchor.value = ''
}

function archiveSelected() {
  emit('archive', getSelectedRecords())
}

function clearFilters() {
  filterSearch.value = ''
  filterStartDate.value = ''
  filterEndDate.value = ''
  filterAccounts.value = new Set()
  filterRecordKinds.value = new Set()
  filterChannels.value = new Set()
  filterSenderProfiles.value = new Set()
  filterListenerProfiles.value = new Set()
  senderProfileSearch.value = ''
  listenerProfileSearch.value = ''
}

async function syncFromPlugin() {
  const wowPath = localStorage.getItem('wow_path') || ''
  if (!wowPath) return
  loading.value = true
  syncError.value = ''
  try {
    const logs = await invoke<AccountChatLogs[]>('scan_chat_logs', { wowPath })
    accounts.value = normalizeAccounts(logs)
    clearSelection()
    expandFirstMatch()
  } catch (error) {
    syncError.value = error instanceof Error ? error.message : String(error)
    console.error('同步失败:', error)
  } finally {
    loading.value = false
  }
}

function toggleRecord(recordKey: string, extendRange = false) {
  if (extendRange && selectionAnchor.value) {
    const keys = renderedVisibleRecords.value.map(record => record.record_key)
    const anchorIndex = keys.indexOf(selectionAnchor.value)
    const targetIndex = keys.indexOf(recordKey)
    if (anchorIndex >= 0 && targetIndex >= 0) {
      const next = new Set(selectedRecords.value)
      const [start, end] = anchorIndex < targetIndex
        ? [anchorIndex, targetIndex]
        : [targetIndex, anchorIndex]
      for (const key of keys.slice(start, end + 1)) next.add(key)
      selectedRecords.value = next
      return
    }
  }
  const next = new Set(selectedRecords.value)
  if (next.has(recordKey)) next.delete(recordKey)
  else next.add(recordKey)
  selectedRecords.value = next
  selectionAnchor.value = recordKey
}

function getDateRecords(date: string): ChatRecord[] {
  return Object.values(groupedRecords.value[date] || {}).flat()
}

function getHourRecords(date: string, hour: string): ChatRecord[] {
  return groupedRecords.value[date]?.[hour] || []
}

function areAllSelected(records: ChatRecord[]): boolean {
  return records.length > 0 && records.every(record => selectedRecords.value.has(record.record_key))
}

function areSomeSelected(records: ChatRecord[]): boolean {
  const count = records.filter(record => selectedRecords.value.has(record.record_key)).length
  return count > 0 && count < records.length
}

function toggleRecords(records: ChatRecord[]) {
  const next = new Set(selectedRecords.value)
  const shouldRemove = areAllSelected(records)
  for (const record of records) {
    if (shouldRemove) next.delete(record.record_key)
    else next.add(record.record_key)
  }
  selectedRecords.value = next
  selectionAnchor.value = ''
}

const filterSignature = computed(() => JSON.stringify({
  search: filterSearch.value.trim(),
  start: filterStartDate.value,
  end: filterEndDate.value,
  accounts: [...filterAccounts.value].sort(),
  kinds: [...filterRecordKinds.value].sort(),
  channels: [...filterChannels.value].sort(),
  senders: [...filterSenderProfiles.value].sort(),
  listeners: [...filterListenerProfiles.value].sort(),
  sort: sortDirection.value,
}))

watch(filterSignature, () => {
  const visibleKeys = new Set(visibleRecords.value.map(record => record.record_key))
  if ([...selectedRecords.value].some(key => !visibleKeys.has(key))) {
    selectedRecords.value = new Set([...selectedRecords.value].filter(key => visibleKeys.has(key)))
  }
  if (selectionAnchor.value && !visibleKeys.has(selectionAnchor.value)) selectionAnchor.value = ''
  expandFirstMatch()
}, { flush: 'post' })

watch(
  () => availableAccounts.value.map(account => account.id).join('\u0000'),
  () => {
    const available = new Set(availableAccounts.value.map(account => account.id))
    if ([...filterAccounts.value].some(account => !available.has(account))) {
      filterAccounts.value = new Set([...filterAccounts.value].filter(account => available.has(account)))
    }
  },
)

watch(
  () => availableSenderProfiles.value.map(profile => profile.key).join('\u0000'),
  () => {
    const available = new Set(availableSenderProfiles.value.map(profile => profile.key))
    if ([...filterSenderProfiles.value].some(key => !available.has(key))) {
      filterSenderProfiles.value = new Set([...filterSenderProfiles.value].filter(key => available.has(key)))
    }
  },
)

watch(
  () => availableListenerProfiles.value.map(profile => profile.key).join('\u0000'),
  () => {
    const available = new Set(availableListenerProfiles.value.map(profile => profile.key))
    if ([...filterListenerProfiles.value].some(key => !available.has(key))) {
      filterListenerProfiles.value = new Set([...filterListenerProfiles.value].filter(key => available.has(key)))
    }
  },
)

function stripNpcPrefix(text: string): string {
  if (!text || text.startsWith('|c')) return text
  return text.replace(/^\|+\s*/, '')
}

function getRecordContent(record: ChatRecord): string {
  if (record.mark === 'B' || (record.mark === 'N' && !record.npc)) return stripNpcPrefix(record.content)
  return record.content
}

function getSenderName(record: ChatRecord): string {
  if (record.mark === 'N' && record.npc) return cleanWowText(record.npc)
  const snapshotName = snapshotDisplayName(record.profile_snapshot)
  if (snapshotName) return snapshotName
  const trp3 = record.sender.trp3
  if (trp3?.FN) return cleanWowText(trp3.LN ? `${trp3.FN} ${trp3.LN}` : trp3.FN)
  return cleanWowText(record.sender.gameID.split('-')[0] || record.sender.gameID)
}

function getSenderIcon(record: ChatRecord): string {
  return record.profile_snapshot?.IC || record.sender.trp3?.IC || ''
}

function getSenderColor(record: ChatRecord): string {
  return record.profile_snapshot?.CH || record.sender.trp3?.CH || ''
}

function identityEventTitle(record: ChatRecord): string {
  return record.event?.kind === 'profile_update'
    ? t('archives.staging.identityUpdated')
    : t('archives.staging.identitySwitched')
}

function identityEventCertainty(record: ChatRecord): string {
  return record.event?.certainty === 'exact'
    ? t('archives.staging.identityExact')
    : t('archives.staging.identityObserved')
}

function formatTime(timestamp: number): string {
  return new Date(timestamp * 1000).toLocaleTimeString(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

function normalizeChannel(channel: string): string {
  const normalized = channel.replace(/^CHAT_MSG_/, '')
  if (normalized === 'WHISPER_IN' || normalized === 'WHISPER_OUT') return 'WHISPER'
  return normalized === 'TEXT_EMOTE' ? 'EMOTE' : normalized
}

function getChannelLabel(channel: string): string {
  return t(`archives.staging.channel.${normalizeChannel(channel).toLowerCase()}`)
}

function getChannelClass(channel: string): string {
  const normalized = normalizeChannel(channel)
  if (normalized === 'YELL') return 'channel-yell'
  if (normalized === 'WHISPER') return 'channel-whisper'
  return ''
}

function getNpcTalkLabel(nt?: string): string {
  return nt ? t(`archives.staging.npcTalk.${nt}`) : ''
}

function getNpcTalkClass(nt?: string): string {
  return nt === 'yell' ? 'npc-yell' : nt === 'whisper' ? 'npc-whisper' : 'npc-say'
}

function isLegacyIdentity(record: ChatRecord): boolean {
  return record.mark !== 'N'
    && record.mark !== 'B'
    && record.mark !== 'S'
    && record.identity_source !== 'snapshot'
}

onMounted(() => {
  if (localStorage.getItem('wow_path')) syncFromPlugin()
})

defineExpose({
  getSelectedRecords,
  clearSelection,
  removeArchivedRecords,
  syncFromPlugin,
})
</script>

<template>
  <div class="staging-pool">
    <aside class="filter-rail" :aria-label="t('archives.staging.filterAriaLabel')">
      <div class="rail-heading">
        <div>
          <span class="eyebrow">{{ t('archives.staging.archiveDesk') }}</span>
          <h2>{{ t('archives.staging.filters') }}</h2>
        </div>
        <button
          v-if="activeFilterCount > 0"
          type="button"
          class="clear-button"
          @click="clearFilters"
        >
          {{ t('archives.filter.clearFilter') }}
        </button>
      </div>

      <label class="search-field">
        <span>{{ t('archives.staging.searchLabel') }}</span>
        <span class="search-input-wrap">
          <i class="ri-search-line" aria-hidden="true"></i>
          <input
            v-model="filterSearch"
            type="search"
            :placeholder="t('archives.staging.searchPlaceholder')"
          />
        </span>
      </label>

      <fieldset class="filter-section">
        <legend>{{ t('archives.staging.dateRange') }}</legend>
        <div class="date-presets" :aria-label="t('archives.staging.quickDateRange')">
          <button
            type="button"
            :class="{ active: activeDatePreset === 'today' }"
            :aria-pressed="activeDatePreset === 'today'"
            @click="setDatePreset('today')"
          >
            {{ t('archives.staging.today') }}
          </button>
          <button
            type="button"
            :class="{ active: activeDatePreset === 'week' }"
            :aria-pressed="activeDatePreset === 'week'"
            @click="setDatePreset('week')"
          >
            {{ t('archives.staging.recentWeek') }}
          </button>
          <button
            type="button"
            :class="{ active: activeDatePreset === 'all' }"
            :aria-pressed="activeDatePreset === 'all'"
            @click="setDatePreset('all')"
          >
            {{ t('archives.staging.allDates') }}
          </button>
        </div>
        <div class="date-grid">
          <label>
            <span>{{ t('archives.staging.startDate') }}</span>
            <input v-model="filterStartDate" type="date" />
          </label>
          <label>
            <span>{{ t('archives.staging.endDate') }}</span>
            <input v-model="filterEndDate" type="date" />
          </label>
        </div>
        <p v-if="dateRangeInvalid" class="filter-error" role="alert">
          {{ t('archives.staging.invalidDateRange') }}
        </p>
      </fieldset>

      <fieldset v-if="availableAccounts.length" class="filter-section">
        <legend>
          {{ t('archives.staging.gameAccount') }}
          <small>{{ t('archives.staging.multiSelectOr') }}</small>
        </legend>
        <div class="filter-chips account-chips">
          <button
            v-for="account in availableAccounts"
            :key="account.id"
            type="button"
            class="filter-chip account-chip"
            :class="{ active: filterAccounts.has(account.id) }"
            :aria-pressed="filterAccounts.has(account.id)"
            @click="toggleAccount(account.id)"
          >
            <span>{{ account.id }}</span>
            <small>{{ account.count }}</small>
          </button>
        </div>
      </fieldset>

      <fieldset class="filter-section">
        <legend>{{ t('archives.staging.recordType') }}</legend>
        <div class="filter-chips">
          <button
            v-for="kind in recordKinds"
            :key="kind"
            type="button"
            class="filter-chip"
            :class="{ active: filterRecordKinds.has(kind) }"
            :aria-pressed="filterRecordKinds.has(kind)"
            @click="toggleRecordKind(kind)"
          >
            {{ t(`archives.staging.recordKind.${kind}`) }}
          </button>
        </div>
      </fieldset>

      <fieldset class="filter-section">
        <legend>{{ t('archives.staging.chatType') }}</legend>
        <div class="filter-chips">
          <button
            v-for="channel in ['SAY', 'YELL', 'EMOTE', 'PARTY', 'RAID', 'GUILD', 'WHISPER']"
            :key="channel"
            type="button"
            class="filter-chip"
            :class="{ active: filterChannels.has(channel) }"
            :aria-pressed="filterChannels.has(channel)"
            @click="toggleChannel(channel)"
          >
            {{ getChannelLabel(channel) }}
          </button>
        </div>
      </fieldset>

      <fieldset class="filter-section">
        <legend>
          {{ t('archives.staging.senderProfile') }}
          <small>{{ t('archives.staging.multiSelectOr') }}</small>
        </legend>
        <label v-if="availableSenderProfiles.length" class="profile-search">
          <i class="ri-search-line" aria-hidden="true"></i>
          <input
            v-model="senderProfileSearch"
            type="search"
            :aria-label="t('archives.staging.searchSenderProfiles')"
            :placeholder="t('archives.staging.searchProfiles')"
          />
        </label>
        <div v-if="visibleSenderProfiles.length" class="profile-options">
          <button
            v-for="profile in visibleSenderProfiles"
            :key="profile.key"
            type="button"
            class="profile-option"
            :class="{ active: filterSenderProfiles.has(profile.key) }"
            :aria-pressed="filterSenderProfiles.has(profile.key)"
            @click="toggleSenderProfile(profile.key)"
          >
            <span>{{ profile.name }}</span>
            <small v-if="profile.detail">{{ profile.detail }}</small>
            <em v-if="profile.legacy">{{ t('archives.staging.legacyShort') }}</em>
            <b :title="t('archives.staging.profileRecordCount', { count: profile.count })">
              {{ profile.count }}
            </b>
          </button>
        </div>
        <p v-if="hiddenSenderProfileCount > 0" class="filter-empty-hint">
          {{ t('archives.staging.profileOptionsLimited', { count: hiddenSenderProfileCount }) }}
        </p>
        <span v-if="!visibleSenderProfiles.length" class="filter-empty-hint">
          {{ availableSenderProfiles.length
            ? t('archives.staging.noProfileMatches')
            : t('archives.staging.noProfileData') }}
        </span>
      </fieldset>

      <fieldset class="filter-section">
        <legend>
          {{ t('archives.staging.listenerProfile') }}
          <small>{{ t('archives.staging.multiSelectOr') }}</small>
        </legend>
        <label v-if="availableListenerProfiles.length" class="profile-search">
          <i class="ri-search-line" aria-hidden="true"></i>
          <input
            v-model="listenerProfileSearch"
            type="search"
            :aria-label="t('archives.staging.searchListenerProfiles')"
            :placeholder="t('archives.staging.searchProfiles')"
          />
        </label>
        <div v-if="visibleListenerProfiles.length" class="profile-options">
          <button
            v-for="profile in visibleListenerProfiles"
            :key="profile.key"
            type="button"
            class="profile-option"
            :class="{ active: filterListenerProfiles.has(profile.key) }"
            :aria-pressed="filterListenerProfiles.has(profile.key)"
            @click="toggleListenerProfile(profile.key)"
          >
            <span>{{ profile.name }}</span>
            <small v-if="profile.detail">{{ profile.detail }}</small>
            <em v-if="profile.legacy">{{ t('archives.staging.legacyShort') }}</em>
            <b :title="t('archives.staging.profileRecordCount', { count: profile.count })">
              {{ profile.count }}
            </b>
          </button>
        </div>
        <p v-if="hiddenListenerProfileCount > 0" class="filter-empty-hint">
          {{ t('archives.staging.profileOptionsLimited', { count: hiddenListenerProfileCount }) }}
        </p>
        <span v-if="!visibleListenerProfiles.length" class="filter-empty-hint">
          {{ availableListenerProfiles.length
            ? t('archives.staging.noProfileMatches')
            : t('archives.staging.noListenerData') }}
        </span>
        <p v-if="filterListenerProfiles.size" class="strict-filter-note">
          {{ t('archives.staging.legacyListenerExcluded') }}
        </p>
      </fieldset>

      <div class="filter-summary" aria-live="polite">
        <strong>{{ totalRecords }}</strong>
        <span>{{ t('archives.staging.matchesFrom', { total: unarchivedRecordCount }) }}</span>
        <small v-if="activeFilterCount">
          {{ t('archives.staging.activeFilterCount', { count: activeFilterCount }) }}
        </small>
      </div>
    </aside>

    <section class="ledger">
      <header class="ledger-header">
        <div>
          <span class="eyebrow">RPBOX · TRP3</span>
          <h2>{{ t('archives.staging.timelineTitle') }}</h2>
        </div>
        <RButton :loading="loading" @click="syncFromPlugin">
          <i class="ri-refresh-line" aria-hidden="true"></i>
          {{ t('archives.staging.sync') }}
        </RButton>
      </header>

      <nav class="ledger-tools" :aria-label="t('archives.staging.batchTools')">
        <div class="selection-ruler" aria-live="polite">
          <span>{{ t('archives.staging.matchingScope', { count: totalRecords }) }}</span>
          <strong>{{ t('archives.staging.selectedCount', { count: selectedCount }) }}</strong>
        </div>
        <div class="bulk-actions">
          <button type="button" :disabled="totalRecords === 0" @click="selectAllMatches">
            <i class="ri-checkbox-multiple-line" aria-hidden="true"></i>
            {{ t('archives.staging.selectAllMatches') }}
          </button>
          <button type="button" :disabled="totalRecords === 0" @click="invertMatchSelection">
            <i class="ri-checkbox-indeterminate-line" aria-hidden="true"></i>
            {{ t('archives.staging.invertMatches') }}
          </button>
          <button type="button" :disabled="selectedCount === 0" @click="clearSelection">
            {{ t('archives.staging.clearSelection') }}
          </button>
        </div>
        <div class="view-actions">
          <button
            type="button"
            :title="t('archives.staging.sortOrder')"
            @click="sortDirection = sortDirection === 'newest' ? 'oldest' : 'newest'"
          >
            <i :class="sortDirection === 'newest' ? 'ri-sort-desc' : 'ri-sort-asc'" aria-hidden="true"></i>
            {{ sortDirection === 'newest'
              ? t('archives.staging.newestFirst')
              : t('archives.staging.oldestFirst') }}
          </button>
          <button type="button" @click="collapseAll">
            <i class="ri-collapse-diagonal-line" aria-hidden="true"></i>
            {{ t('archives.staging.collapseAll') }}
          </button>
        </div>
      </nav>

      <p v-if="syncError" class="sync-error" role="alert">{{ syncError }}</p>

      <REmpty
        v-if="!loading && totalRecords === 0"
        :description="unarchivedRecordCount > 0
          ? t('archives.staging.noMatches')
          : t('archives.staging.emptyHint')"
      >
        <button
          v-if="unarchivedRecordCount > 0"
          type="button"
          class="clear-button"
          @click="clearFilters"
        >
          {{ t('archives.filter.clearFilter') }}
        </button>
        <router-link v-else class="tutorial-link" :to="{ name: 'guide' }">
          <i class="ri-book-open-line" aria-hidden="true"></i>
          {{ t('archives.staging.viewTutorial') }}
        </router-link>
      </REmpty>

      <div v-else class="staging-content">
        <section
          v-for="date in orderedDates"
          :key="date"
          class="date-group"
        >
          <div class="group-header date-header">
            <button
              type="button"
              class="header-checkbox"
              :aria-label="t('archives.staging.selectDateRecords', { date })"
              @click.stop="toggleRecords(getDateRecords(date))"
            >
              <RCheckbox
                :model-value="areAllSelected(getDateRecords(date))"
                :indeterminate="areSomeSelected(getDateRecords(date))"
              />
            </button>
            <button type="button" @click="toggleExpandedDate(date)">
              <i :class="expandedDates.has(date) ? 'ri-arrow-down-s-line' : 'ri-arrow-right-s-line'"></i>
              <span>{{ date }}</span>
              <small>{{ t('archives.staging.recordCount', { count: getDateRecords(date).length }) }}</small>
            </button>
          </div>

          <div v-if="expandedDates.has(date)" class="hour-groups">
            <section
              v-for="hour in orderedHours(date)"
              :key="`${date}-${hour}`"
              class="hour-group"
            >
              <div class="group-header hour-header">
                <button
                  type="button"
                  class="header-checkbox"
                  :aria-label="t('archives.staging.selectHourRecords', { hour })"
                  @click.stop="toggleRecords(getHourRecords(date, hour))"
                >
                  <RCheckbox
                    :model-value="areAllSelected(getHourRecords(date, hour))"
                    :indeterminate="areSomeSelected(getHourRecords(date, hour))"
                  />
                </button>
                <button type="button" @click="toggleExpandedHour(`${date}-${hour}`)">
                  <i :class="expandedHours.has(`${date}-${hour}`) ? 'ri-arrow-down-s-line' : 'ri-arrow-right-s-line'"></i>
                  <span>{{ hour }}:00—{{ hour }}:59</span>
                  <small>{{ t('archives.staging.recordCount', { count: getHourRecords(date, hour).length }) }}</small>
                </button>
              </div>

              <div v-if="expandedHours.has(`${date}-${hour}`)" class="records">
                <article
                  v-for="record in renderedHourRecords(date, hour)"
                  :key="record.record_key"
                  class="record-item"
                  :class="{
                    selected: selectedRecords.has(record.record_key),
                    'mark-npc': record.mark === 'N',
                    'mark-background': record.mark === 'B',
                    'identity-event': Boolean(record.event) || record.mark === 'S',
                  }"
                  role="checkbox"
                  :aria-checked="selectedRecords.has(record.record_key)"
                  tabindex="0"
                  @click="toggleRecord(record.record_key, $event.shiftKey)"
                  @keydown.enter.prevent="toggleRecord(record.record_key, $event.shiftKey)"
                  @keydown.space.prevent="toggleRecord(record.record_key, $event.shiftKey)"
                >
                  <RCheckbox :model-value="selectedRecords.has(record.record_key)" />
                  <time :datetime="new Date(record.timestamp * 1000).toISOString()">
                    {{ formatTime(record.timestamp) }}
                  </time>

                  <template v-if="record.event || record.mark === 'S'">
                    <span class="identity-glyph" aria-hidden="true"><i class="ri-swap-2-line"></i></span>
                    <div class="identity-copy">
                      <div>
                        <strong>{{ identityEventTitle(record) }}</strong>
                        <span class="certainty">{{ identityEventCertainty(record) }}</span>
                      </div>
                      <p>
                        <span>{{ endpointName(record.event?.from) }}</span>
                        <i class="ri-arrow-right-line" aria-hidden="true"></i>
                        <span>{{ endpointName(record.event?.to) }}</span>
                      </p>
                    </div>
                  </template>

                  <template v-else>
                    <span
                      v-if="record.mark === 'N' && record.nt"
                      class="record-channel"
                      :class="getNpcTalkClass(record.nt)"
                    >
                      NPC{{ getNpcTalkLabel(record.nt) }}
                    </span>
                    <span v-else class="record-channel" :class="getChannelClass(record.channel)">
                      {{ getChannelLabel(record.channel) }}
                    </span>
                    <div class="message-copy">
                      <div v-if="record.mark !== 'B'" class="speaker-line">
                        <WowIcon
                          v-if="getSenderIcon(record)"
                          :icon="getSenderIcon(record)"
                          :size="18"
                          class="record-avatar"
                        />
                        <strong :style="getSenderColor(record) ? { color: `#${getSenderColor(record)}` } : {}">
                          {{ getSenderName(record) }}
                        </strong>
                        <span v-if="record.profile_snapshot?.pn" class="profile-name">
                          {{ record.profile_snapshot.pn }}
                        </span>
                        <span v-if="isLegacyIdentity(record)" class="legacy-badge">
                          {{ t('archives.staging.legacyInferred') }}
                        </span>
                      </div>
                      <p>{{ getRecordContent(record) }}</p>
                    </div>
                  </template>
                </article>
                <nav
                  v-if="getHourRecords(date, hour).length > HOUR_RECORD_BATCH_SIZE"
                  class="hour-pagination"
                  :aria-label="t('archives.staging.hourPagination')"
                >
                  <span>{{ t('archives.staging.showingHourRecords', {
                    start: hourPageStart(date, hour),
                    end: hourPageEnd(date, hour),
                    total: getHourRecords(date, hour).length,
                  }) }}</span>
                  <div>
                    <button
                      type="button"
                      class="previous-hour-page"
                      :disabled="!hasPreviousHourPage(date, hour)"
                      @click="changeHourPage(date, hour, -1)"
                    >
                      <i class="ri-arrow-left-line" aria-hidden="true"></i>
                      {{ t('archives.staging.previousRecordBatch') }}
                    </button>
                    <button
                      type="button"
                      class="next-hour-page"
                      :disabled="!hasNextHourPage(date, hour)"
                      @click="changeHourPage(date, hour, 1)"
                    >
                      {{ t('archives.staging.nextRecordBatch') }}
                      <i class="ri-arrow-right-line" aria-hidden="true"></i>
                    </button>
                  </div>
                </nav>
              </div>
            </section>
          </div>
        </section>
      </div>

    </section>
  </div>

  <Teleport to="body">
    <footer v-if="props.active && selectedCount > 0" class="staging-footer">
      <span>{{ t('archives.staging.selectedCount', { count: selectedCount }) }}</span>
      <div>
        <button type="button" class="clear-button" @click="clearSelection">
          {{ t('archives.staging.clearSelection') }}
        </button>
        <RButton type="primary" @click="archiveSelected">
          {{ t('archives.staging.archiveSelected') }}
        </RButton>
      </div>
    </footer>
  </Teleport>
</template>

<style scoped>
.staging-pool {
  --archive-ink: var(--color-text-main, #2c1810);
  --archive-parchment: var(--color-panel-bg, #f4e8d8);
  --archive-copper: var(--color-accent, #b87333);
  --archive-muted: var(--color-text-secondary, #6f6258);
  --archive-identity: #3f7d7a;
  --archive-identity-text: color-mix(in srgb, var(--archive-identity) 68%, var(--archive-ink));
  display: grid;
  grid-template-columns: minmax(230px, 280px) minmax(0, 1fr);
  min-height: 560px;
  height: 100%;
  overflow: hidden;
  color: var(--archive-ink);
  background: color-mix(in srgb, var(--archive-parchment) 38%, var(--color-main-bg, #eed9c4));
  border: 1px solid color-mix(in srgb, var(--archive-copper) 24%, var(--color-border));
  border-radius: var(--radius-md);
}

.filter-rail {
  overflow-y: auto;
  padding: 18px 16px;
  background:
    linear-gradient(180deg, rgba(184, 115, 51, 0.08), transparent 140px),
    color-mix(in srgb, var(--archive-parchment) 65%, var(--color-main-bg, #eed9c4));
  border-right: 1px solid color-mix(in srgb, var(--archive-copper) 28%, var(--color-border));
}

.rail-heading,
.ledger-header,
.staging-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.rail-heading h2,
.ledger-header h2 {
  margin: 2px 0 0;
  color: var(--archive-ink);
  font-family: Georgia, 'Noto Serif SC', serif;
  font-size: 19px;
}

.eyebrow {
  color: var(--archive-copper);
  font-size: 10px;
  font-weight: 700;
  letter-spacing: 0.14em;
  text-transform: uppercase;
}

.clear-button {
  padding: 4px 0;
  color: var(--archive-copper);
  background: transparent;
  border: 0;
  cursor: pointer;
  font-size: 12px;
}

.search-field {
  display: grid;
  gap: 6px;
  margin: 18px 0;
  color: var(--archive-muted);
  font-size: 12px;
  font-weight: 600;
}

.search-input-wrap {
  display: flex;
  align-items: center;
  gap: 7px;
  padding: 0 9px;
  background: color-mix(in srgb, var(--archive-parchment) 76%, var(--color-card-bg, #fff));
  border: 1px solid color-mix(in srgb, var(--archive-copper) 25%, var(--color-border));
  border-radius: var(--radius-sm);
}

.search-input-wrap input,
.date-grid input,
.profile-search input {
  min-width: 0;
  width: 100%;
  padding: 8px 0;
  color: var(--archive-ink);
  background: transparent;
  border: 0;
  outline: 0;
}

.filter-section {
  min-width: 0;
  margin: 0;
  padding: 14px 0;
  border: 0;
  border-top: 1px solid color-mix(in srgb, var(--archive-copper) 18%, transparent);
}

.filter-section legend {
  display: flex;
  align-items: baseline;
  gap: 6px;
  width: 100%;
  margin-bottom: 9px;
  padding: 0;
  color: var(--archive-ink);
  font-size: 12px;
  font-weight: 700;
}

.filter-section legend small {
  color: var(--archive-muted);
  font-size: 10px;
  font-weight: 400;
}

.date-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.date-presets {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 5px;
  margin-bottom: 8px;
}

.date-presets button {
  padding: 5px 4px;
  color: var(--archive-muted);
  background: transparent;
  border: 1px solid color-mix(in srgb, var(--archive-copper) 22%, var(--color-border));
  cursor: pointer;
  font-size: 10px;
}

.date-presets button:first-child { border-radius: var(--radius-sm) 0 0 var(--radius-sm); }
.date-presets button:last-child { border-radius: 0 var(--radius-sm) var(--radius-sm) 0; }

.date-presets button.active {
  color: var(--btn-primary-text, var(--color-text-light, #fff));
  background: var(--archive-copper);
  border-color: var(--archive-copper);
}

.date-grid label {
  display: grid;
  gap: 4px;
  color: var(--archive-muted);
  font-size: 10px;
}

.date-grid input {
  padding: 7px;
  background: color-mix(in srgb, var(--archive-parchment) 76%, var(--color-card-bg, #fff));
  border: 1px solid color-mix(in srgb, var(--archive-copper) 22%, var(--color-border));
  border-radius: var(--radius-sm);
  font-size: 11px;
}

.filter-error {
  margin: 7px 0 0;
  color: #9d3429;
  font-size: 10px;
  line-height: 1.35;
}

.filter-chips,
.profile-options {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.filter-chip,
.profile-option {
  color: var(--archive-muted);
  background: color-mix(in srgb, var(--archive-parchment) 76%, var(--color-card-bg, #fff));
  border: 1px solid color-mix(in srgb, var(--archive-copper) 20%, var(--color-border));
  border-radius: 999px;
  cursor: pointer;
}

.filter-chip {
  padding: 5px 9px;
  font-size: 11px;
}

.account-chips {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
}

.account-chip {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  border-radius: var(--radius-sm);
  text-align: left;
}

.account-chip span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.account-chip small {
  min-width: 20px;
  padding: 1px 5px;
  background: color-mix(in srgb, currentColor 9%, transparent);
  border-radius: 999px;
  text-align: center;
}

.profile-search {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
  padding: 0 8px;
  color: var(--archive-muted);
  background: color-mix(in srgb, var(--archive-parchment) 76%, var(--color-card-bg, #fff));
  border: 1px solid color-mix(in srgb, var(--archive-copper) 20%, var(--color-border));
  border-radius: var(--radius-sm);
}

.profile-search input {
  padding: 7px 0;
  font-size: 11px;
}

.profile-options {
  max-height: 230px;
  overflow-y: auto;
  padding: 1px 4px 1px 1px;
  scrollbar-color: color-mix(in srgb, var(--archive-copper) 45%, transparent) transparent;
  scrollbar-width: thin;
}

.profile-option {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  gap: 1px 6px;
  max-width: 100%;
  padding: 6px 9px;
  border-radius: var(--radius-sm);
  text-align: left;
}

.profile-option span {
  overflow: hidden;
  color: var(--archive-ink);
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-option small {
  grid-column: 1;
  overflow: hidden;
  font-size: 10px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.profile-option em {
  grid-column: 2;
  grid-row: 1 / span 2;
  align-self: center;
  color: var(--archive-muted);
  font-size: 9px;
  font-style: normal;
}

.profile-option b {
  grid-column: 3;
  grid-row: 1 / span 2;
  align-self: center;
  min-width: 20px;
  padding: 1px 5px;
  color: var(--archive-copper);
  background: rgba(184, 115, 51, 0.08);
  border-radius: 999px;
  font-size: 9px;
  font-weight: 700;
  text-align: center;
}

.filter-chip.active,
.profile-option.active {
  color: var(--btn-primary-text, var(--color-text-light, #fff));
  background: var(--archive-copper);
  border-color: var(--archive-copper);
}

.profile-option.active span,
.profile-option.active em,
.profile-option.active b {
  color: var(--btn-primary-text, var(--color-text-light, #fff));
}

.profile-option.active b { background: rgba(255, 255, 255, 0.16); }

.filter-empty-hint,
.strict-filter-note {
  color: var(--archive-muted);
  font-size: 11px;
}

.strict-filter-note {
  margin: 8px 0 0;
  line-height: 1.4;
}

.filter-summary {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: baseline;
  gap: 1px 6px;
  margin-top: 4px;
  padding: 12px;
  background: rgba(184, 115, 51, 0.09);
  border-left: 2px solid var(--archive-copper);
}

.filter-summary strong {
  color: var(--archive-copper);
  font-family: Georgia, serif;
  font-size: 22px;
}

.filter-summary span,
.filter-summary small {
  color: var(--archive-muted);
  font-size: 11px;
}

.filter-summary small {
  grid-column: 1 / -1;
}

.ledger {
  display: flex;
  min-width: 0;
  flex-direction: column;
  overflow: hidden;
  background: color-mix(in srgb, var(--archive-parchment) 22%, var(--color-main-bg, #eed9c4));
}

.ledger-header {
  padding: 16px 20px;
  border-bottom: 1px solid color-mix(in srgb, var(--archive-copper) 20%, var(--color-border));
}

.ledger-tools {
  display: grid;
  grid-template-columns: minmax(145px, auto) minmax(0, 1fr) auto;
  align-items: center;
  gap: 10px;
  padding: 8px 20px;
  background:
    linear-gradient(90deg, rgba(184, 115, 51, 0.09), transparent 42%),
    color-mix(in srgb, var(--archive-parchment) 50%, var(--color-main-bg, #eed9c4));
  border-bottom: 1px solid color-mix(in srgb, var(--archive-copper) 22%, var(--color-border));
}

.selection-ruler {
  display: grid;
  gap: 1px;
  padding-left: 9px;
  border-left: 2px solid var(--archive-copper);
  font-size: 10px;
}

.selection-ruler span { color: var(--archive-muted); }
.selection-ruler strong { color: var(--archive-ink); font-size: 11px; }

.bulk-actions,
.view-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.view-actions { justify-content: flex-end; }

.ledger-tools button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 7px;
  color: var(--archive-muted);
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 10px;
  white-space: nowrap;
}

.ledger-tools button:hover:not(:disabled) {
  color: var(--archive-ink);
  background: rgba(184, 115, 51, 0.08);
  border-color: rgba(184, 115, 51, 0.16);
}

.ledger-tools button:disabled {
  cursor: not-allowed;
  opacity: 0.4;
}

.sync-error {
  margin: 10px 20px 0;
  padding: 8px 10px;
  color: #8d2d22;
  background: rgba(180, 50, 40, 0.08);
  border-left: 2px solid #a33a2c;
  font-size: 12px;
}

.staging-content {
  flex: 1;
  overflow-y: auto;
  padding: 12px 20px 80px;
}

.date-group {
  margin-bottom: 8px;
}

.group-header {
  display: flex;
  align-items: center;
  gap: 8px;
}

.group-header > button:not(.header-checkbox) {
  display: flex;
  flex: 1;
  align-items: center;
  gap: 7px;
  padding: 7px 5px;
  color: var(--archive-ink);
  background: transparent;
  border: 0;
  cursor: pointer;
  text-align: left;
}

.group-header button small {
  color: var(--archive-muted);
  font-weight: 400;
}

.date-header button {
  font-family: Georgia, 'Noto Serif SC', serif;
  font-weight: 700;
}

.hour-header {
  margin-left: 18px;
}

.hour-header button {
  color: var(--archive-muted);
  font-size: 12px;
}

.header-checkbox {
  display: flex;
  flex: 0 0 auto;
  padding: 0;
  background: transparent;
  border: 0;
  cursor: pointer;
}

.header-checkbox :deep(.r-checkbox) {
  pointer-events: none;
}

.records {
  position: relative;
  margin-left: 31px;
  padding-left: 18px;
}

.records::before {
  position: absolute;
  inset: 0 auto 0 4px;
  width: 1px;
  background: color-mix(in srgb, var(--archive-copper) 24%, var(--color-border));
  content: '';
}

.hour-pagination {
  position: relative;
  z-index: 1;
  width: calc(100% - 10px);
  margin: 8px 0 4px 10px;
  padding: 9px 12px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  border: 1px dashed color-mix(in srgb, var(--archive-copper) 38%, var(--color-border));
  border-radius: var(--radius-sm);
  background: color-mix(in srgb, var(--archive-parchment) 88%, var(--archive-copper));
  color: var(--archive-muted);
  font-size: 11px;
}

.hour-pagination > div {
  display: flex;
  gap: 6px;
}

.hour-pagination button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 8px;
  border: 1px solid color-mix(in srgb, var(--archive-copper) 32%, var(--color-border));
  border-radius: 5px;
  background: var(--archive-parchment);
  color: var(--archive-copper);
  cursor: pointer;
  font-size: 11px;
  white-space: nowrap;
}

.hour-pagination button:hover:not(:disabled) {
  border-color: var(--archive-copper);
  color: var(--archive-ink);
}

.hour-pagination button:disabled {
  opacity: 0.42;
  cursor: not-allowed;
}

.record-item {
  position: relative;
  display: grid;
  grid-template-columns: 20px 66px auto minmax(0, 1fr);
  align-items: start;
  gap: 8px;
  margin: 2px 0;
  padding: 8px 10px;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
  line-height: 1.45;
}

.record-item:hover,
.record-item.selected {
  background: rgba(184, 115, 51, 0.07);
  border-color: rgba(184, 115, 51, 0.18);
}

.record-item:focus-visible,
button:focus-visible,
input:focus-visible {
  outline: 2px solid var(--archive-copper);
  outline-offset: 2px;
}

.record-item time {
  padding-top: 2px;
  color: var(--archive-muted);
  font-variant-numeric: tabular-nums;
  font-size: 11px;
}

.record-channel {
  min-width: 36px;
  margin-top: 1px;
  padding: 2px 6px;
  color: var(--archive-copper);
  background: rgba(184, 115, 51, 0.08);
  border-radius: 999px;
  font-size: 10px;
  text-align: center;
}

.channel-yell,
.npc-yell { color: #a33127; }
.channel-whisper,
.npc-whisper { color: #775592; }
.npc-say { color: #76518d; }

.speaker-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}

.record-avatar {
  flex: 0 0 auto;
  border-radius: 3px;
}

.message-copy {
  min-width: 0;
}

.message-copy strong {
  color: var(--archive-ink);
  font-weight: 650;
}

.message-copy p {
  margin: 2px 0 0;
  color: color-mix(in srgb, var(--archive-ink) 88%, var(--archive-muted));
  overflow-wrap: anywhere;
  white-space: pre-wrap;
}

.profile-name,
.legacy-badge,
.certainty {
  color: var(--archive-muted);
  font-size: 10px;
}

.profile-name::before { content: '· '; }

.legacy-badge {
  padding: 1px 5px;
  background: rgba(111, 98, 88, 0.08);
  border: 1px solid rgba(111, 98, 88, 0.2);
  border-radius: 999px;
}

.mark-background .message-copy p {
  color: var(--archive-muted);
  font-family: Georgia, 'Noto Serif SC', serif;
  font-style: italic;
}

.identity-event {
  grid-template-columns: 20px 66px 32px minmax(0, 1fr);
  margin: 8px 0;
  color: var(--archive-identity-text);
  background: color-mix(in srgb, var(--archive-identity) 8%, transparent);
  border-color: color-mix(in srgb, var(--archive-identity) 28%, transparent);
}

.identity-event:hover,
.identity-event.selected {
  background: color-mix(in srgb, var(--archive-identity) 13%, transparent);
  border-color: color-mix(in srgb, var(--archive-identity) 44%, transparent);
}

.identity-glyph {
  display: grid;
  width: 28px;
  height: 28px;
  place-items: center;
  color: white;
  background: var(--archive-identity);
  border-radius: 50%;
}

.identity-copy strong {
  color: var(--archive-identity-text);
  font-family: Georgia, 'Noto Serif SC', serif;
}

.identity-copy > div {
  display: flex;
  align-items: baseline;
  gap: 8px;
}

.identity-copy p {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 7px;
  margin: 2px 0 0;
  color: var(--archive-identity-text);
}

.staging-footer {
  --archive-ink: var(--color-text-main, #2c1810);
  --archive-parchment: var(--color-panel-bg, #f4e8d8);
  --archive-copper: var(--color-accent, #b87333);
  position: fixed;
  right: 24px;
  bottom: 24px;
  z-index: 120;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  min-width: 310px;
  max-width: calc(100vw - 48px);
  padding: 10px 12px 10px 16px;
  color: var(--archive-ink);
  background: color-mix(in srgb, var(--archive-parchment) 88%, var(--color-card-bg, #fff));
  border: 1px solid rgba(184, 115, 51, 0.35);
  border-radius: var(--radius-md);
  box-shadow: 0 10px 30px rgba(44, 24, 16, 0.14);
  font-size: 12px;
}

.staging-footer > div {
  display: flex;
  align-items: center;
  gap: 12px;
}

.tutorial-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 7px 12px;
  color: var(--archive-copper);
  text-decoration: none;
}

@media (max-width: 980px) {
  .staging-pool {
    grid-template-columns: 1fr;
    height: auto;
    overflow: visible;
  }

  .filter-rail {
    max-height: 360px;
    border-right: 0;
    border-bottom: 1px solid color-mix(in srgb, var(--archive-copper) 28%, var(--color-border));
  }

  .ledger { min-height: 520px; }

  .ledger-tools {
    grid-template-columns: minmax(140px, auto) minmax(0, 1fr);
  }

  .view-actions {
    grid-column: 1 / -1;
    justify-content: flex-start;
  }
}

@media (max-width: 640px) {
  .ledger-header,
  .rail-heading { align-items: flex-start; }
  .ledger-tools {
    grid-template-columns: 1fr;
    padding-inline: 10px;
  }
  .bulk-actions,
  .view-actions {
    grid-column: 1;
    flex-wrap: wrap;
  }
  .date-grid { grid-template-columns: 1fr; }
  .staging-content { padding-inline: 10px; }
  .hour-header { margin-left: 6px; }
  .records { margin-left: 10px; padding-left: 9px; }
  .hour-pagination { align-items: stretch; flex-direction: column; }
  .hour-pagination > div { justify-content: space-between; }
  .record-item,
  .identity-event {
    grid-template-columns: 20px 58px minmax(0, 1fr);
  }
  .record-channel,
  .identity-glyph { grid-column: 3; }
  .message-copy,
  .identity-copy { grid-column: 3; }
  .staging-footer {
    right: 12px;
    bottom: 12px;
    left: 12px;
    flex-wrap: wrap;
    min-width: 0;
    max-width: none;
  }
}

@media (prefers-reduced-motion: reduce) {
  *,
  *::before,
  *::after {
    scroll-behavior: auto !important;
    transition: none !important;
  }
}
</style>
