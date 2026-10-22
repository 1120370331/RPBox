export interface RPTool {
  id: string
  name: string
  subtitle: string
  description: string
  icon: string
  route: string
  features: string[]
  tags: string[]
}

export const rpTools: RPTool[] = [
  {
    id: 'rpbox', name: 'RPBox', subtitle: '连接游戏内外的 RP 工具',
    description: '安装配套插件，备份人物卡，将游戏里的对话整理成可以再次翻阅的剧情。',
    icon: 'ri-box-3-line', route: '/tools/rpbox',
    features: ['插件安装', '人物卡备份', '人物卡档案', '剧情归档'], tags: ['RPBox', '魔兽世界', '桌面工具'],
  },
  {
    id: 'trp3', name: '在线人物卡编辑器', subtitle: '熟悉的 Total RP 3 编辑体验',
    description: '编辑角色特征、第一印象与关于页面。使用原版格式标签和魔兽图标，直接保存到 RPBox 人物卡档案。',
    icon: 'ri-id-card-line', route: '/tools/trp3',
    features: ['人物卡编辑', '档案联动', '图标选择', '排版预览'], tags: ['TRP3', 'RPBox', '魔兽世界', '在线工具'],
  },
]

export function filterRPTools(tools: RPTool[], search: string, feature: string, tags: string[]) {
  const words = search.trim().toLocaleLowerCase().split(/\s+/).filter(Boolean)
  return tools.filter(tool => {
    const text = [tool.name, tool.subtitle, tool.description, ...tool.features, ...tool.tags].join(' ').toLocaleLowerCase()
    return words.every(word => text.includes(word)) && (!feature || tool.features.includes(feature)) && tags.every(tag => tool.tags.includes(tag))
  })
}
