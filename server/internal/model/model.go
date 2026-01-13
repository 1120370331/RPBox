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
