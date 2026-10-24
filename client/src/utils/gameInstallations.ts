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
export function getGameInstallations(): GameInstallation[] {
  let items: GameInstallation[] = []
  try {
    const raw: unknown = JSON.parse(localStorage.getItem(storageKey) || '[]')
    if (Array.isArray(raw)) items = raw.filter(item => item && typeof item.path === 'string' && typeof item.flavor === 'string')
  } catch { /* Retain the existing single-directory setting during migration. */ }
  const legacy = localStorage.getItem('wow_path') || ''
  if (legacy && !items.some(item => sameGamePath(item.path, legacy))) items.push({ path: legacy, flavor: inferGameFlavor(legacy) })
  return items
}
export function sameGamePath(a: string, b: string) { return a.replace(/\\/g, '/').replace(/\/+$/, '').toLowerCase() === b.replace(/\\/g, '/').replace(/\/+$/, '').toLowerCase() }
export function selectGameInstallation(item: GameInstallation) {
  const items = getGameInstallations().filter(existing => !sameGamePath(existing.path, item.path))
  items.push(item)
  localStorage.setItem(storageKey, JSON.stringify(items))
  localStorage.setItem('wow_path', item.path)
  localStorage.setItem('selected_flavor', item.flavor)
}
export function removeGameInstallation(path: string) {
  const items = getGameInstallations().filter(item => !sameGamePath(item.path, path))
  if (!items.length) throw new Error('至少保留一个游戏目录；可以先添加新目录再移除旧目录。')
  localStorage.setItem(storageKey, JSON.stringify(items))
  if (sameGamePath(localStorage.getItem('wow_path') || '', path)) selectGameInstallation(items[0]!)
}

export function configuredGameFlavor(path: string) {
  return inferGameFlavor(path) || getGameInstallations().find(item => sameGamePath(item.path, path))?.flavor || ''
}
