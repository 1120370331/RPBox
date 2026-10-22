import request from './request'

export type ContributionKind = 'post' | 'rpdb' | 'item'
export type ContributionSort = 'created_at' | 'popular'

export interface UserContribution {
  id: number
  kind: ContributionKind
  title: string
  summary: string
  has_cover: boolean
  cover_url: string
  cover_updated_at?: string
  updated_at: string
  created_at: string
  like_count: number
  view_count: number
}

export interface UserContributionPage {
  entries: UserContribution[]
  total: number
  page: number
  page_size: number
}

export function listUserContributions(userId: number, params: {
  page: number
  page_size: number
  kind: ContributionKind | 'all'
  sort: ContributionSort
}) {
  return request.get<UserContributionPage>(`/users/${userId}/contributions`, { params })
}
