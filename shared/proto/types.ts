// 用户相关
export interface User {
  id: number
  username: string
  email: string
  created_at: string
}

// 人物卡
export interface Profile {
  id: number
  user_id: number
  name: string
  race: string
  class: string
  description: string
  data: string
  created_at: string
}
