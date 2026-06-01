export interface UserActivityInfo {
  activity_points?: number
  activity_experience?: number
  forum_level?: number
  forum_level_name?: string
  forum_level_color?: string
  forum_level_bold?: boolean
  current_level_exp?: number
  next_level_exp?: number
  level_progress_percent?: number
  signed_in_today?: boolean
  total_sign_in_days?: number
  consecutive_sign_in_days?: number
  name_style_preference?: 'default' | 'sponsor'
  avatar_change_count?: number
  username_change_count?: number
  next_avatar_change_cost?: number
  next_username_change_cost?: number
  post_count?: number
  guild_count?: number
  item_count?: number
  story_count?: number
  story_entry_count?: number
  profile_count?: number
  max_post_views?: number
  max_item_downloads?: number
  total_likes?: number
  total_item_downloads?: number
}

export interface UserData extends UserActivityInfo {
  id: number
  username: string
  email?: string
  email_verified?: boolean
  avatar?: string
  avatar_review_status?: string
  role?: string
  is_sponsor?: boolean
  sponsor_level?: number
  sponsor_expires_at?: string | null
  sponsor_color?: string
  sponsor_bold?: boolean
  name_color?: string
  name_bold?: boolean
}
