<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
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

interface ProfileOption {
  key: string
  name: string
  detail: string
  legacy: boolean
}

const emit = defineEmits<{
  archive: [records: ChatRecord[]]
}>()

const loading = ref(false)
const syncError = ref('')
const accounts = ref<AccountChatLogs[]>([])
const selectedRecords = ref<Set<string>>(new Set())
const expandedDates = ref<Set<string>>(new Set())
const expandedHours = ref<Set<string>>(new Set())

const filterSearch = ref('')
const filterStartDate = ref('')
const filterEndDate = ref('')
const filterChannels = ref<Set<string>>(new Set())
const filterSenderProfiles = ref<Set<string>>(new Set())
const filterListenerProfiles = ref<Set<string>>(new Set())

const ARCHIVED_KEYS_STORAGE = 'rpbox_archived_record_keys_v2'
const LEGACY_ARCHIVED_STORAGE = 'archived_timestamps'

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
}

function removeArchivedRecords(recordKeys: string[]) {
  for (const key of recordKeys) {
    archivedRecordKeys.value.add(key)
    selectedRecords.value.delete(key)
  }
  archivedRecordKeys.value = new Set(archivedRecordKeys.value)
  selectedRecords.value = new Set(selectedRecords.value)
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
  }
}

function mergeProfileOption(map: Map<string, ProfileOption>, option: ProfileOption) {
  const current = map.get(option.key)
  if (!current || (current.legacy && !option.legacy) || (!current.detail && option.detail)) {
    map.set(option.key, option)
  }
}

const availableSenderProfiles = computed(() => {
  const profiles = new Map<string, ProfileOption>()
  for (const record of allRecords.value) {
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
  for (const record of allRecords.value) {
    for (const listener of record.listeners || []) {
      const key = listenerProfileKey(listener)
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
      })
    }
  }
  return [...profiles.values()].sort((a, b) => a.name.localeCompare(b.name))
})

function localDateKey(timestamp: number): string {
  const date = new Date(timestamp * 1000)
  const year = date.getFullYear()
  const month = String(date.getMonth() + 1).padStart(2, '0')
  const day = String(date.getDate()).padStart(2, '0')
  return `${year}-${month}-${day}`
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
    const dateStr = localDateKey(record.timestamp)
    if (filterStartDate.value && dateStr < filterStartDate.value) continue
    if (filterEndDate.value && dateStr > filterEndDate.value) continue
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
        b.timestamp - a.timestamp
        || (b.sequence ?? 0) - (a.sequence ?? 0)
        || b.record_key.localeCompare(a.record_key)
      ))
    }
  }
  return groups
})

const totalRecords = computed(() => Object.values(groupedRecords.value)
  .reduce((total, date) => total + Object.values(date).flat().length, 0))
const visibleRecords = computed(() => Object.values(groupedRecords.value)
  .flatMap(date => Object.values(date).flat()))
const unarchivedRecordCount = computed(() => allRecords.value
  .filter(record => !archivedRecordKeys.value.has(record.record_key)).length)
const selectedCount = computed(() => getSelectedRecords().length)
const activeFilterCount = computed(() => (
  Number(Boolean(filterSearch.value.trim()))
  + Number(Boolean(filterStartDate.value))
  + Number(Boolean(filterEndDate.value))
  + filterChannels.value.size
  + filterSenderProfiles.value.size
  + filterListenerProfiles.value.size
))

function toggledSet(current: Set<string>, value: string): Set<string> {
  const next = new Set(current)
  if (next.has(value)) next.delete(value)
  else next.add(value)
  return next
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
  expandedDates.value = toggledSet(expandedDates.value, key)
}

function toggleExpandedHour(key: string) {
  expandedHours.value = toggledSet(expandedHours.value, key)
}

function archiveSelected() {
  emit('archive', getSelectedRecords())
}

function clearFilters() {
  filterSearch.value = ''
  filterStartDate.value = ''
  filterEndDate.value = ''
  filterChannels.value = new Set()
  filterSenderProfiles.value = new Set()
  filterListenerProfiles.value = new Set()
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
    const dates = Object.keys(groupedRecords.value).sort()
    if (dates.length > 0) {
      const lastDate = dates[dates.length - 1]
      expandedDates.value = new Set([lastDate])
      const hours = Object.keys(groupedRecords.value[lastDate]).sort()
      if (hours.length > 0) expandedHours.value = new Set([`${lastDate}-${hours[hours.length - 1]}`])
    }
  } catch (error) {
    syncError.value = error instanceof Error ? error.message : String(error)
    console.error('同步失败:', error)
  } finally {
    loading.value = false
  }
}

function toggleRecord(recordKey: string) {
  const next = new Set(selectedRecords.value)
  if (next.has(recordKey)) next.delete(recordKey)
  else next.add(recordKey)
  selectedRecords.value = next
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
}

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
    <aside class="filter-rail" aria-label="剧情回溯筛选">
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
        <div v-if="availableSenderProfiles.length" class="profile-options">
          <button
            v-for="profile in availableSenderProfiles"
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
          </button>
        </div>
        <span v-else class="filter-empty-hint">{{ t('archives.staging.noProfileData') }}</span>
      </fieldset>

      <fieldset class="filter-section">
        <legend>
          {{ t('archives.staging.listenerProfile') }}
          <small>{{ t('archives.staging.multiSelectOr') }}</small>
        </legend>
        <div v-if="availableListenerProfiles.length" class="profile-options">
          <button
            v-for="profile in availableListenerProfiles"
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
          </button>
        </div>
        <span v-else class="filter-empty-hint">{{ t('archives.staging.noListenerData') }}</span>
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
          <p>{{ t('archives.staging.timelineSubtitle') }}</p>
        </div>
        <RButton :loading="loading" @click="syncFromPlugin">
          <i class="ri-refresh-line" aria-hidden="true"></i>
          {{ t('archives.staging.sync') }}
        </RButton>
      </header>

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
          v-for="date in Object.keys(groupedRecords).sort().reverse()"
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
              v-for="hour in Object.keys(groupedRecords[date]).sort().reverse()"
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
                  v-for="record in groupedRecords[date][hour]"
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
                  @click="toggleRecord(record.record_key)"
                  @keydown.enter.prevent="toggleRecord(record.record_key)"
                  @keydown.space.prevent="toggleRecord(record.record_key)"
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
              </div>
            </section>
          </div>
        </section>
      </div>

      <footer v-if="selectedCount > 0" class="staging-footer">
        <span>{{ t('archives.staging.selectedCount', { count: selectedCount }) }}</span>
        <RButton type="primary" @click="archiveSelected">
          {{ t('archives.staging.archiveSelected') }}
        </RButton>
      </footer>
    </section>
  </div>
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
.date-grid input {
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

.profile-option {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
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

.filter-chip.active,
.profile-option.active {
  color: var(--btn-primary-text, var(--color-text-light, #fff));
  background: var(--archive-copper);
  border-color: var(--archive-copper);
}

.profile-option.active span,
.profile-option.active em {
  color: var(--btn-primary-text, var(--color-text-light, #fff));
}

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

.ledger-header p {
  margin: 3px 0 0;
  color: var(--archive-muted);
  font-size: 12px;
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
  position: absolute;
  right: 20px;
  bottom: 18px;
  min-width: 310px;
  padding: 10px 12px 10px 16px;
  color: var(--archive-ink);
  background: color-mix(in srgb, var(--archive-parchment) 88%, var(--color-card-bg, #fff));
  border: 1px solid rgba(184, 115, 51, 0.35);
  border-radius: var(--radius-md);
  box-shadow: 0 10px 30px rgba(44, 24, 16, 0.14);
  font-size: 12px;
}

.ledger { position: relative; }

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
}

@media (max-width: 640px) {
  .ledger-header,
  .rail-heading { align-items: flex-start; }
  .date-grid { grid-template-columns: 1fr; }
  .staging-content { padding-inline: 10px; }
  .hour-header { margin-left: 6px; }
  .records { margin-left: 10px; padding-left: 9px; }
  .record-item,
  .identity-event {
    grid-template-columns: 20px 58px minmax(0, 1fr);
  }
  .record-channel,
  .identity-glyph { grid-column: 3; }
  .message-copy,
  .identity-copy { grid-column: 3; }
  .staging-footer {
    right: 10px;
    bottom: 10px;
    left: 10px;
    min-width: 0;
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
