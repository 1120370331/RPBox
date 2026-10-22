export interface NavigationStep { path: string; label: string }

const roots: Record<string, string> = {
  '/': '首页', '/tools': '工具箱', '/warcraft': '魔兽功能', '/market': '创意市场',
  '/rpdb': 'RP 数据库', '/community': '社区广场', '/guild': '公会', '/settings': '系统设置', '/moderator': '版主中心',
}
const labels: Record<string, string> = {
  ...roots, '/tools/rpbox': 'RPBox', '/tools/rpbox/install': '插件安装', '/tools/trp3': '在线人物卡编辑器',
  '/sync': '人物卡备份', '/sync/setup': '设置游戏目录', '/archives': '剧情归档', '/character-cards/new': '新建人物卡',
  '/market/upload': '上传道具', '/market/my-items': '我的道具', '/market/preview': '道具预览',
  '/community/create': '发布帖子', '/community/my-posts': '我的帖子', '/community/preview': '帖子预览',
  '/rpdb/create': '发布作品', '/rpdb/my-uploads': '我的作品', '/rpdb/preview': '作品预览', '/rpdb/lists': '收藏清单',
  '/guild/create': '创建公会', '/library/favorites': '我的收藏', '/library/history': '浏览历史',
  '/notifications': '消息中心', '/thanks': '鸣谢', '/guide': '使用指南', '/settings/blocked-users': '屏蔽用户',
}

export function navigationPath(path: string) { return path.split(/[?#]/)[0] || '/' }
export function safeNavigationPath(value: unknown): value is string {
  return typeof value === 'string' && value.startsWith('/') && !/^\/\//.test(value)
    && !/[\\\u0000-\u001f]/.test(value) && !/^\/(login|register|forgot-password)(?:[/?#]|$)/.test(value)
}
export function navigationLabel(fullPath: string) {
  const path = navigationPath(fullPath)
  if (labels[path]) return labels[path]
  if (/^\/tools\/trp3\/[^/]+$/.test(path)) return '编辑人物卡'
  if (/^\/character-cards\/[^/]+\/edit$/.test(path)) return '编辑 RPBox 人物卡'
  if (path.startsWith('/character-cards/')) return 'RPBox 人物卡'
  if (/^\/sync\/profile\/[^/]+\/edit$/.test(path)) return '编辑备份'
  if (path.startsWith('/sync/profile/')) return '备份详情'
  if (path.startsWith('/archives/story/')) return '剧情详情'
  if (/^\/user\/[^/]+\/achievements$/.test(path)) return '成就墙'
  if (path.startsWith('/user/')) return '个人主页'
  if (path.endsWith('/edit')) return '编辑内容'
  if (path.startsWith('/community/post/')) return '帖子详情'
  if (path.startsWith('/rpdb/')) return '作品详情'
  if (path.startsWith('/market/')) return '道具详情'
  if (path.startsWith('/guild/')) return path.endsWith('/manage') ? '管理公会' : path.endsWith('/posts') ? '公会帖子' : path.endsWith('/stories') ? '公会剧情' : '公会详情'
  if (path.startsWith('/collection/')) return '合集详情'
  return '当前页面'
}
export function defaultNavigationTrail(fullPath: string): NavigationStep[] {
  const path = navigationPath(fullPath)
  const parents = ['/']
  if (path.startsWith('/tools/')) {
    parents.push('/tools')
    if (path.startsWith('/tools/rpbox/')) parents.push('/tools/rpbox')
    if (path.startsWith('/tools/trp3/')) parents.push('/tools/trp3')
  } else if (path.startsWith('/sync') || path.startsWith('/archives')) {
    parents.push('/tools', '/tools/rpbox')
    if (path.startsWith('/sync/') && path !== '/sync/setup') parents.push('/sync')
    if (path.startsWith('/archives/')) parents.push('/archives')
  } else if (path.startsWith('/character-cards/')) parents.push('/tools', '/tools/trp3')
  else {
    const root = '/' + path.split('/')[1]
    if (roots[root] && root !== path) parents.push(root)
  }
  if (path.endsWith('/edit') || /^\/user\/[^/]+\/achievements$/.test(path) || /^\/guild\/[^/]+\/(manage|posts|stories)$/.test(path)) parents.push(path.slice(0, path.lastIndexOf('/')))
  return [...new Set([...parents.filter(parent => parent !== path), fullPath])].map(value => ({ path: value, label: navigationLabel(value) }))
}

/** Successful navigation extends a branch; returning to an ancestor trims it. */
export function advanceNavigationTrail(previous: NavigationStep[], fullPath: string): NavigationStep[] {
  const path = navigationPath(fullPath)
  if (!safeNavigationPath(fullPath)) return []
  const current = { path: fullPath, label: navigationLabel(fullPath) }
  if (path === '/sync/setup' && new URLSearchParams(fullPath.split('?')[1]).get('required') === '1') {
    previous = previous.filter(step => navigationPath(step.path) !== '/sync')
  }
  const last = previous.at(-1)?.path || ''
  if (navigationPath(last) === '/sync/setup') {
    const destination = new URLSearchParams(last.split('?')[1]).get('redirect')
    if (destination && navigationPath(destination) === path) previous = previous.slice(0, -1)
  }
  if (roots[path]) return defaultNavigationTrail(fullPath)
  const index = previous.findIndex(step => navigationPath(step.path) === path)
  if (index >= 0) return [...previous.slice(0, index), current]
  if (!previous.length || previous.at(-1)?.path === '/') return defaultNavigationTrail(fullPath)
  // Switching between sibling editor records is not another hierarchy level.
  const leaf = previous.at(-1)!
  if (/^\/tools\/trp3\/[^/]+$/.test(path) && /^\/tools\/trp3\/[^/]+$/.test(navigationPath(leaf.path))) return [...previous.slice(0, -1), current]
  const next = [...previous, current]
  return next.length > 10 ? [next[0]!, ...next.slice(-9)] : next
}
