package model

import "time"

type User struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:50" json:"username"`
	Email     string    `gorm:"uniqueIndex;size:100" json:"email"`
	Password  string    `gorm:"-" json:"-"`
	PassHash  string    `json:"-"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Profile struct {
	ID          string    `gorm:"primarykey;size:64" json:"id"`
	UserID      uint      `gorm:"index;not null" json:"user_id"`
	AccountID   string    `gorm:"size:32;index" json:"account_id"`
	ProfileName string    `gorm:"size:128" json:"profile_name"`
	RawLua      string    `gorm:"type:text" json:"raw_lua,omitempty"`
	Checksum    string    `gorm:"size:32" json:"checksum"`
	Version     int       `gorm:"default:1" json:"version"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type ProfileVersion struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	ProfileID string    `gorm:"size:64;index" json:"profile_id"`
	Version   int       `json:"version"`
	RawLua    string    `gorm:"type:text" json:"raw_lua,omitempty"`
	Checksum  string    `gorm:"size:32" json:"checksum"`
	ChangeLog string    `gorm:"type:text" json:"change_log"`
	CreatedAt time.Time `json:"created_at"`
}

// AccountBackup 账号备份（以账号为单位）
type AccountBackup struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	UserID        uint      `gorm:"index;not null" json:"user_id"`
	AccountID     string    `gorm:"size:32;uniqueIndex:idx_user_account" json:"account_id"`
	ProfilesData  string    `gorm:"type:text" json:"profiles_data,omitempty"` // JSON: 所有人物卡数据
	ProfilesCount int       `json:"profiles_count"`
	ToolsData     string    `gorm:"type:text" json:"tools_data,omitempty"` // JSON: TRP3 Extended 道具数据库
	ToolsCount    int       `json:"tools_count"`
	RuntimeData   string    `gorm:"type:text" json:"runtime_data,omitempty"` // JSON: TRP3 运行时数据
	RuntimeSizeKB int       `json:"runtime_size_kb"`
	ConfigData    string    `gorm:"type:text" json:"config_data,omitempty"` // JSON: TRP3 配置数据
	ExtraData     string    `gorm:"type:text" json:"extra_data,omitempty"`  // JSON: TRP3 额外数据(角色绑定、伙伴等)
	Checksum      string    `gorm:"type:text" json:"checksum"`
	Version       int       `gorm:"default:1" json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// AccountBackupVersion 账号备份版本历史
type AccountBackupVersion struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	BackupID     uint      `gorm:"index" json:"backup_id"`
	Version      int       `json:"version"`
	ProfilesData string    `gorm:"type:text" json:"profiles_data,omitempty"`
	ToolsData    string    `gorm:"type:text" json:"tools_data,omitempty"`
	RuntimeData  string    `gorm:"type:text" json:"runtime_data,omitempty"`
	ConfigData   string    `gorm:"type:text" json:"config_data,omitempty"`
	ExtraData    string    `gorm:"type:text" json:"extra_data,omitempty"`
	Checksum     string    `gorm:"type:text" json:"checksum"`
	ChangeLog    string    `gorm:"type:text" json:"change_log"`
	CreatedAt    time.Time `json:"created_at"`
}

// Story 剧情
type Story struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	UserID       uint      `gorm:"index;not null" json:"user_id"`
	Title        string    `gorm:"size:256" json:"title"`
	Description  string    `gorm:"type:text" json:"description"`
	Participants string    `gorm:"type:text" json:"participants"` // JSON数组
	Tags         string    `gorm:"size:512" json:"tags"`          // 逗号分隔
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Status       string    `gorm:"size:20;default:draft" json:"status"` // draft, published
	IsPublic     bool      `gorm:"default:false" json:"is_public"`      // 是否公开分享
	ShareCode    string    `gorm:"size:16;index" json:"share_code"`     // 分享码
	ViewCount    int       `gorm:"default:0" json:"view_count"`         // 浏览次数
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// StoryEntry 剧情条目
type StoryEntry struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	StoryID     uint      `gorm:"index;not null" json:"story_id"`
	SourceID    string    `gorm:"size:64" json:"source_id"`             // 来源聊天记录ID
	Type        string    `gorm:"size:20;default:dialogue" json:"type"` // dialogue, narration, image
	CharacterID *uint     `gorm:"index" json:"character_id"`            // 关联角色ID（可空，旁白无角色）
	Speaker     string    `gorm:"size:128" json:"speaker"`              // 说话者名字快照
	Content     string    `gorm:"type:text" json:"content"`
	Channel     string    `gorm:"size:32" json:"channel"`
	Timestamp   time.Time `json:"timestamp"`
	SortOrder   int       `gorm:"default:0" json:"sort_order"`
	CreatedAt   time.Time `json:"created_at"`
}

// Character 全局人物卡模型 (与TRP3 characteristics 1:1对应)
type Character struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	UserID    uint      `gorm:"index" json:"user_id"`                // 创建者用户ID
	RefID     string    `gorm:"size:64;index" json:"ref_id"`         // TRP3 ref ID
	GameID    string    `gorm:"size:128;index" json:"game_id"`       // 游戏内ID (角色名-服务器)
	IsNPC     bool      `gorm:"default:false" json:"is_npc"`         // 是否是NPC
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// TRP3 characteristics 字段 (1:1对应)
	TRP3Version int    `gorm:"default:1" json:"trp3_version"`       // v: 版本
	Race        string `gorm:"size:64" json:"race"`                 // RA: 种族
	Class       string `gorm:"size:64" json:"class"`                // CL: 职业
	FirstName   string `gorm:"size:128" json:"first_name"`          // FN: 名字
	LastName    string `gorm:"size:128" json:"last_name"`           // LN: 姓氏
	FullTitle   string `gorm:"size:256" json:"full_title"`          // FT: 全称
	Title       string `gorm:"size:128" json:"title"`               // TI: 称号
	Icon        string `gorm:"size:128" json:"icon"`                // IC: 图标
	Color       string `gorm:"size:8" json:"color"`                 // CH: 名字颜色(hex)
	EyeColor    string `gorm:"size:64" json:"eye_color"`            // EC: 眼睛颜色
	Age         string `gorm:"size:64" json:"age"`                  // AG: 年龄
	Height      string `gorm:"size:64" json:"height"`               // HE: 身高
	Residence   string `gorm:"size:256" json:"residence"`           // RE: 住所
	Birthplace  string `gorm:"size:256" json:"birthplace"`          // BP: 出生地
	MiscInfo    string `gorm:"type:text" json:"misc_info"`          // MI: 其他信息 (JSON数组)
	Psycho      string `gorm:"type:text" json:"psycho"`             // PS: 性格特征 (JSON数组)

	// TRP3 about 字段
	AboutText string `gorm:"type:text" json:"about_text"` // 关于/描述 (JSON)

	// 用户自定义覆盖字段
	CustomAvatar string `gorm:"size:256" json:"custom_avatar"` // 自定义头像URL
	CustomName   string `gorm:"size:128" json:"custom_name"`   // 自定义显示名
	CustomColor  string `gorm:"size:8" json:"custom_color"`    // 自定义颜色

	// 原始TRP3数据备份
	RawTRP3Data string `gorm:"type:text" json:"raw_trp3_data"` // 完整原始JSON备份
}
