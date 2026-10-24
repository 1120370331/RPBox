export const gameFlavors = [
  { value: '_retail_', label: '正式服' },
  { value: '_classic_', label: '怀旧服（进度服）' },
  { value: '_classic_era_', label: '经典怀旧服' },
  { value: '_classic_anniversary_', label: '周年怀旧服' },
  { value: '_ptr_', label: '正式服测试服' },
  { value: '_beta_', label: '正式服 Beta' },
  { value: '_classic_ptr_', label: '怀旧测试服' },
  { value: '_classic_beta_', label: '怀旧 Beta' },
]
export function gameFlavorLabel(flavor?: string) { return gameFlavors.find(item => item.value === flavor)?.label || '旧存档（版本未标记）' }
export function inferGameFlavor(path: string) {
  const parts = path.replace(/\\/g, '/').toLowerCase().split('/')
  return gameFlavors.find(item => parts.includes(item.value))?.value || ''
}
export interface GameInstallation { path: string; flavor: string }
const storageKey = 'rpbox_game_installations'

export function gamePathKey(path: string) {
  return path.trim().replace(/\\/g, '/').replace(/\/+$/, '').replace(/\/wtf(?:\/account)?$/i, '').toLowerCase()
}

export function getGameInstallations(): GameInstallation[] {
  let items: GameInstallation[] = []
  try {
    const raw: unknown = JSON.parse(localStorage.getItem(storageKey) || '[]')
    if (Array.isArray(raw)) items = raw.filter(item => item && typeof item.path === 'string' && item.path.trim() && typeof item.flavor === 'string')
  } catch { /* Retain the existing single-directory setting during migration. */ }
  const legacy = localStorage.getItem('wow_path') || ''
  if (legacy && !items.some(item => sameGamePath(item.path, legacy))) {
    const savedFlavor = localStorage.getItem('selected_flavor') || ''
    items.push({ path: legacy, flavor: gameFlavors.some(item => item.value === savedFlavor) ? savedFlavor : inferGameFlavor(legacy) })
  }
  return [...new Map(items.map(item => [gamePathKey(item.path), item])).values()]
}
export function sameGamePath(a: string, b: string) { return gamePathKey(a) === gamePathKey(b) }

function saveActiveInstallation(item: GameInstallation) {
  localStorage.setItem('wow_path', item.path)
  localStorage.setItem('selected_flavor', item.flavor)
}

// Registration and activation are separate: adding another client preserves the active one.
export function registerGameInstallation(item: GameInstallation) {
  if (!item.path.trim()) throw new Error('请选择游戏目录')
  if (!gameFlavors.some(flavor => flavor.value === item.flavor)) throw new Error('请选择游戏版本')
  const items = getGameInstallations().filter(existing => !sameGamePath(existing.path, item.path))
  const saved = { path: item.path.trim(), flavor: item.flavor }
  items.push(saved)
  localStorage.setItem(storageKey, JSON.stringify(items))
  const activePath = localStorage.getItem('wow_path') || ''
  if (!activePath || sameGamePath(activePath, saved.path)) saveActiveInstallation(saved)
}

export function selectGameInstallation(item: GameInstallation) {
  registerGameInstallation(item)
  saveActiveInstallation({ path: item.path.trim(), flavor: item.flavor })
}
export function removeGameInstallation(path: string) {
  const items = getGameInstallations().filter(item => !sameGamePath(item.path, path))
  localStorage.setItem(storageKey, JSON.stringify(items))
  if (sameGamePath(localStorage.getItem('wow_path') || '', path)) {
    if (items[0]) saveActiveInstallation(items[0])
    else {
      localStorage.removeItem('wow_path')
      localStorage.removeItem('selected_flavor')
    }
  }
}

export function configuredGameFlavor(path: string) {
  return getGameInstallations().find(item => sameGamePath(item.path, path))?.flavor || inferGameFlavor(path)
}
