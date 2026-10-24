import request from './request'

export interface HomeContent {
  id: number
  kind: 'post' | 'rpdb' | 'item'
  title: string
  author_name: string
  view_count?: number
  like_count?: number
  favorite_count?: number
  comment_count?: number
  list_count?: number
  downloads?: number
  has_cover: boolean
  cover_url: string
  cover_updated_at?: string
  updated_at: string
  weekly_score: number
  reason: 'popular' | 'interest'
}

export interface HomeFeedResponse {
  entries: HomeContent[]
  personalized: boolean
  has_more: boolean
}

export function getHomeFeatured() {
  return request.get<HomeFeedResponse>('/home/featured')
}

export function getHomeRecommendations(exclude: string[] = []) {
  return request.get<HomeFeedResponse>('/home/recommendations', { params: { exclude: exclude.join(',') } })
}

export function homeContentKey(content: HomeContent) {
  return `${content.kind}:${content.id}`
}
