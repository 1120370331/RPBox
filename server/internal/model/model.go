package model

import "time"

type User struct {
	ID            uint   `gorm:"primarykey" json:"id"`
	Username      string `gorm:"uniqueIndex;size:50" json:"username"`
	Email         string `gorm:"uniqueIndex;size:100" json:"email"`
	EmailVerified bool   `gorm:"default:false" json:"email_verified"` // 邮箱是否已验证
	Avatar        string `gorm:"type:text" json:"avatar"`             // 头像(base64)
	Role          string `gorm:"size:20;default:user" json:"role"`    // user|moderator|admin
	Password      string `gorm:"-" json:"-"`
	PassHash      string `json:"-"`
	// 个人资料字段
	Bio      string `gorm:"size:500" json:"bio"`      // 个人简介
	Location string `gorm:"size:100" json:"location"` // 地区
	Website  string `gorm:"size:256" json:"website"`  // 个人网站
	// 统计数据
	PostCount    int `gorm:"default:0" json:"post_count"`    // 帖子数
	StoryCount   int `gorm:"default:0" json:"story_count"`   // 剧情数
	ProfileCount int `gorm:"default:0" json:"profile_count"` // 人物卡数
	// 封禁状态
	IsMuted     bool       `gorm:"default:false" json:"is_muted"`  // 禁言状态
	MutedUntil  *time.Time `json:"muted_until"`                    // 禁言截止时间（空=永久）
	MuteReason  string     `gorm:"size:512" json:"mute_reason"`    // 禁言原因
	IsBanned    bool       `gorm:"default:false" json:"is_banned"` // 禁止登录状态
	BannedUntil *time.Time `json:"banned_until"`                   // 禁止登录截止时间（空=永久）
	BanReason   string     `gorm:"size:512" json:"ban_reason"`     // 禁止登录原因
	BannedBy    *uint      `json:"banned_by"`                      // 执行封禁的版主ID
	BannedAt    *time.Time `json:"banned_at"`                      // 封禁时间
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
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
type StoryTagInfo struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type Story struct {
	ID           uint           `gorm:"primarykey" json:"id"`
	UserID       uint           `gorm:"index;not null" json:"user_id"`
	Title        string         `gorm:"size:256" json:"title"`
	Description  string         `gorm:"type:text" json:"description"`
	Participants string         `gorm:"type:text" json:"participants"` // JSON数组
	Tags         string         `gorm:"size:512" json:"tags"`          // 逗号分隔
	StartTime    time.Time      `json:"start_time"`
	EndTime      time.Time      `json:"end_time"`
	Status       string         `gorm:"size:20;default:draft" json:"status"` // draft, published
	IsPublic     bool           `gorm:"default:false" json:"is_public"`      // 是否公开分享
	ShareCode    string         `gorm:"size:16;index" json:"share_code"`     // 分享码
	ViewCount    int            `gorm:"default:0" json:"view_count"`         // 浏览次数
	EntryCount   int            `gorm:"-" json:"entry_count"`                // entry count for list views
	TagList      []StoryTagInfo `gorm:"-" json:"tag_list"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
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
	UserID    uint      `gorm:"index" json:"user_id"`          // 创建者用户ID
	RefID     string    `gorm:"size:64;index" json:"ref_id"`   // TRP3 ref ID
	GameID    string    `gorm:"size:128;index" json:"game_id"` // 游戏内ID (角色名-服务器)
	IsNPC     bool      `gorm:"default:false" json:"is_npc"`   // 是否是NPC
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	// TRP3 characteristics 字段 (1:1对应)
	TRP3Version int    `gorm:"default:1" json:"trp3_version"` // v: 版本
	Race        string `gorm:"size:64" json:"race"`           // RA: 种族
	Class       string `gorm:"size:64" json:"class"`          // CL: 职业
	FirstName   string `gorm:"size:128" json:"first_name"`    // FN: 名字
	LastName    string `gorm:"size:128" json:"last_name"`     // LN: 姓氏
	FullTitle   string `gorm:"size:256" json:"full_title"`    // FT: 全称
	Title       string `gorm:"size:128" json:"title"`         // TI: 称号
	Icon        string `gorm:"size:128" json:"icon"`          // IC: 图标
	Color       string `gorm:"size:8" json:"color"`           // CH: 名字颜色(hex)
	EyeColor    string `gorm:"size:64" json:"eye_color"`      // EC: 眼睛颜色
	Age         string `gorm:"size:64" json:"age"`            // AG: 年龄
	Height      string `gorm:"size:64" json:"height"`         // HE: 身高
	Residence   string `gorm:"size:256" json:"residence"`     // RE: 住所
	Birthplace  string `gorm:"size:256" json:"birthplace"`    // BP: 出生地
	MiscInfo    string `gorm:"type:text" json:"misc_info"`    // MI: 其他信息 (JSON数组)
	Psycho      string `gorm:"type:text" json:"psycho"`       // PS: 性格特征 (JSON数组)

	// TRP3 about 字段
	AboutText string `gorm:"type:text" json:"about_text"` // 关于/描述 (JSON)

	// 用户自定义覆盖字段
	CustomAvatar string `gorm:"size:256" json:"custom_avatar"` // 自定义头像URL
	CustomName   string `gorm:"size:128" json:"custom_name"`   // 自定义显示名
	CustomColor  string `gorm:"size:8" json:"custom_color"`    // 自定义颜色

	// 原始TRP3数据备份
	RawTRP3Data string `gorm:"type:text" json:"raw_trp3_data"` // 完整原始JSON备份
}

// Tag 标签
type Tag struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	Name       string    `gorm:"size:64;not null" json:"name"`
	Color      string    `gorm:"size:8" json:"color"`
	Category   string    `gorm:"size:20;default:story" json:"category"` // story|item|post
	Type       string    `gorm:"size:20;default:custom" json:"type"`    // preset|custom|guild
	GuildID    *uint     `gorm:"index" json:"guild_id"`
	CreatorID  uint      `gorm:"index" json:"creator_id"`
	IsPublic   bool      `gorm:"default:false" json:"is_public"`
	UsageCount int       `gorm:"default:0" json:"usage_count"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// StoryTag 剧情-标签关联
type StoryTag struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	StoryID   uint      `gorm:"uniqueIndex:idx_story_tag;not null" json:"story_id"`
	TagID     uint      `gorm:"uniqueIndex:idx_story_tag;not null" json:"tag_id"`
	AddedBy   uint      `gorm:"index" json:"added_by"`
	CreatedAt time.Time `json:"created_at"`
}

// Guild 公会
type Guild struct {
	ID              uint       `gorm:"primarykey" json:"id"`
	Name            string     `gorm:"size:128;not null" json:"name"`
	Description     string     `gorm:"type:text" json:"description"`
	Icon            string     `gorm:"size:128" json:"icon"`
	Color           string     `gorm:"size:8" json:"color"`
	Banner          string     `gorm:"type:text" json:"banner"` // 头图(base64)
	BannerUpdatedAt *time.Time `json:"banner_updated_at,omitempty"`
	Slogan          string     `gorm:"size:256" json:"slogan"`  // 公会标语
	Lore            string     `gorm:"type:text" json:"lore"`   // 公会设定(富文本HTML)
	Faction         string     `gorm:"size:20" json:"faction"`  // 阵营: alliance|horde|neutral
	Layout          int        `gorm:"default:3" json:"layout"` // 主页布局: 1-4, 默认3
	OwnerID         uint       `gorm:"index;not null" json:"owner_id"`
	MemberCount     int        `gorm:"default:1" json:"member_count"`
	StoryCount      int        `gorm:"default:0" json:"story_count"`
	IsPublic        bool       `gorm:"default:true" json:"is_public"`
	InviteCode      string     `gorm:"size:16;uniqueIndex" json:"invite_code"`
	// 审核相关字段
	Status        string     `gorm:"size:20;default:pending" json:"status"` // pending|approved|rejected
	ReviewerID    *uint      `gorm:"index" json:"reviewer_id"`              // 审核人ID
	ReviewComment string     `gorm:"size:512" json:"review_comment"`        // 审核意见
	ReviewedAt    *time.Time `json:"reviewed_at"`                           // 审核时间
	// 隐私设置（分别控制剧情和帖子的可见性）
	VisitorCanViewStories bool      `gorm:"default:false" json:"visitor_can_view_stories"` // 访客可查看剧情
	VisitorCanViewPosts   bool      `gorm:"default:false" json:"visitor_can_view_posts"`   // 访客可查看帖子
	MemberCanViewStories  bool      `gorm:"default:true" json:"member_can_view_stories"`   // 成员可查看剧情（owner/admin始终可见）
	MemberCanViewPosts    bool      `gorm:"default:true" json:"member_can_view_posts"`     // 成员可查看帖子（owner/admin始终可见）
	AutoApprove           bool      `gorm:"default:false" json:"auto_approve"`             // 自动审核（无需审核直接加入）
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// GuildMember 公会成员
type GuildMember struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	GuildID   uint      `gorm:"uniqueIndex:idx_guild_user;not null" json:"guild_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_guild_user;not null" json:"user_id"`
	Role      string    `gorm:"size:20;default:member" json:"role"` // owner|admin|member
	JoinedAt  time.Time `json:"joined_at"`
	CreatedAt time.Time `json:"created_at"`
}

// GuildApplication 公会申请
type GuildApplication struct {
	ID            uint       `gorm:"primarykey" json:"id"`
	GuildID       uint       `gorm:"uniqueIndex:idx_guild_applicant;not null" json:"guild_id"`
	UserID        uint       `gorm:"uniqueIndex:idx_guild_applicant;not null" json:"user_id"`
	Message       string     `gorm:"size:512" json:"message"`
	Status        string     `gorm:"size:20;default:pending" json:"status"` // pending|approved|rejected
	ReviewerID    *uint      `gorm:"index" json:"reviewer_id"`
	ReviewComment string     `gorm:"size:512" json:"review_comment"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// StoryGuild 剧情-公会归档
type StoryGuild struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	StoryID   uint      `gorm:"uniqueIndex:idx_story_guild;not null" json:"story_id"`
	GuildID   uint      `gorm:"uniqueIndex:idx_story_guild;not null" json:"guild_id"`
	AddedBy   uint      `gorm:"index" json:"added_by"`
	CreatedAt time.Time `json:"created_at"`
}

// Item TRP3道具
type Item struct {
	ID                    uint       `gorm:"primarykey" json:"id"`
	AuthorID              uint       `gorm:"index;not null" json:"author_id"`
	Name                  string     `gorm:"size:256;not null" json:"name"`
	Type                  string     `gorm:"size:20;index" json:"type"` // item|document|campaign|artwork
	Icon                  string     `gorm:"size:128" json:"icon"`
	PreviewImage          string     `gorm:"type:text" json:"preview_image"` // 预览图（URL或base64）
	PreviewImageUpdatedAt *time.Time `json:"preview_image_updated_at,omitempty"`
	Description           string     `gorm:"type:text" json:"description"`
	DetailContent         string     `gorm:"type:text" json:"detail_content"`          // 富文本详情
	ImportCode            string     `gorm:"type:text" json:"import_code"`             // TRP3导入代码（artwork类型可选）
	RawData               string     `gorm:"type:text" json:"raw_data"`                // 原始Lua数据
	RequiresPermission    bool       `gorm:"default:false" json:"requires_permission"` // 是否需要TRP3权限授权
	EnableWatermark       bool       `gorm:"default:true" json:"enable_watermark"`     // 画作是否启用水印
	Downloads             int        `gorm:"default:0" json:"downloads"`
	Rating                float64    `gorm:"default:0" json:"rating"`             // 平均评分
	RatingCount           int        `gorm:"default:0" json:"rating_count"`       // 评分人数
	LikeCount             int        `gorm:"default:0" json:"like_count"`         // 点赞数
	FavoriteCount         int        `gorm:"default:0" json:"favorite_count"`     // 收藏数
	Status                string     `gorm:"size:20;default:draft" json:"status"` // draft|pending|published|removed
	// 审核相关字段
	ReviewStatus  string     `gorm:"size:20;default:pending;index" json:"review_status"` // pending|approved|rejected
	ReviewerID    *uint      `gorm:"index" json:"reviewer_id"`                           // 审核人ID
	ReviewComment string     `gorm:"size:512" json:"review_comment"`                     // 审核意见
	ReviewedAt    *time.Time `json:"reviewed_at"`                                        // 审核时间
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ItemTag 道具-标签关联
type ItemTag struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	ItemID    uint      `gorm:"uniqueIndex:idx_item_tag;not null" json:"item_id"`
	TagID     uint      `gorm:"uniqueIndex:idx_item_tag;not null" json:"tag_id"`
	AddedBy   uint      `gorm:"index" json:"added_by"`
	CreatedAt time.Time `json:"created_at"`
}

// ItemRating 道具评分
type ItemRating struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	ItemID    uint      `gorm:"uniqueIndex:idx_item_user;not null" json:"item_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_item_user;not null" json:"user_id"`
	Rating    int       `gorm:"not null" json:"rating"` // 1-5星
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ItemComment 道具评论（带评分）
type ItemComment struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	ItemID    uint      `gorm:"index;not null" json:"item_id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Rating    int       `gorm:"not null" json:"rating"` // 1-5星评分
	Content   string    `gorm:"type:text;not null" json:"content"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ItemLike 道具点赞
type ItemLike struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	ItemID    uint      `gorm:"uniqueIndex:idx_item_like_user;not null" json:"item_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_item_like_user;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ItemFavorite 道具收藏
type ItemFavorite struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	ItemID    uint      `gorm:"uniqueIndex:idx_item_fav_user;not null" json:"item_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_item_fav_user;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ItemView 道具浏览记录
type ItemView struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	ItemID    uint      `gorm:"uniqueIndex:idx_item_view_user;not null" json:"item_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_item_view_user;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ItemDownload 道具下载记录（每用户每道具最多1次）
type ItemDownload struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	ItemID    uint      `gorm:"uniqueIndex:idx_item_download_user;not null" json:"item_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_item_download_user;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ItemPendingEdit 道具待审核编辑
type ItemPendingEdit struct {
	ID            uint       `gorm:"primarykey" json:"id"`
	ItemID        uint       `gorm:"uniqueIndex;not null" json:"item_id"` // 每个道具只能有一个待审核编辑
	AuthorID      uint       `gorm:"index;not null" json:"author_id"`
	Name          string     `gorm:"size:256" json:"name"`
	Icon          string     `gorm:"size:128" json:"icon"`
	Description   string     `gorm:"type:text" json:"description"`
	ImportCode    string     `gorm:"type:text" json:"import_code"`
	ReviewStatus  string     `gorm:"size:20;default:pending" json:"review_status"` // pending|approved|rejected
	ReviewerID    *uint      `gorm:"index" json:"reviewer_id"`
	ReviewComment string     `gorm:"size:512" json:"review_comment"`
	ReviewedAt    *time.Time `json:"reviewed_at"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// ItemImage 画作图片（用于artwork类型的多图）
type ItemImage struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	ItemID    uint      `gorm:"index;not null" json:"item_id"`
	ImageData string    `gorm:"type:text;not null" json:"-"` // 图片数据（URL或base64，不直接返回）
	SortOrder int       `gorm:"default:0" json:"sort_order"` // 排序顺序
	CreatedAt time.Time `json:"created_at"`
}

// ========== 社区系统 ==========

// Post 社区帖子
type Post struct {
	ID                  uint       `gorm:"primarykey" json:"id"`
	AuthorID            uint       `gorm:"index;not null" json:"author_id"`
	Title               string     `gorm:"size:256;not null" json:"title"`
	Content             string     `gorm:"type:text;not null" json:"content"`
	ContentType         string     `gorm:"size:20;default:markdown" json:"content_type"` // markdown|html
	CoverImage          string     `gorm:"type:text" json:"cover_image"`                 // 封面图（URL或base64）
	CoverImageUpdatedAt *time.Time `json:"cover_image_updated_at,omitempty"`
	Category            string     `gorm:"size:20;default:other;index" json:"category"` // 分区: profile|guild|report|novel|item|event|other
	GuildID             *uint      `gorm:"index" json:"guild_id"`                       // 关联公会（可选）
	StoryID             *uint      `gorm:"index" json:"story_id"`                       // 关联剧情（可选）
	Status              string     `gorm:"size:20;default:draft" json:"status"`         // draft|pending|published
	IsPublic            bool       `gorm:"default:true" json:"is_public"`
	ViewCount           int        `gorm:"default:0" json:"view_count"`
	LikeCount           int        `gorm:"default:0" json:"like_count"`
	CommentCount        int        `gorm:"default:0" json:"comment_count"`
	FavoriteCount       int        `gorm:"default:0" json:"favorite_count"`
	// 版主管理字段
	IsPinned   bool `gorm:"default:false" json:"is_pinned"`   // 置顶
	IsFeatured bool `gorm:"default:false" json:"is_featured"` // 精华
	// 活动相关字段
	EventType      string     `gorm:"size:20" json:"event_type"` // server|guild (服务器活动/公会活动)
	EventStartTime *time.Time `json:"event_start_time"`          // 活动开始时间
	EventEndTime   *time.Time `json:"event_end_time"`            // 活动结束时间
	EventColor     string     `gorm:"size:7" json:"event_color"` // 活动标记颜色（十六进制，如 #FF5733）
	// 审核相关字段
	ReviewStatus  string     `gorm:"size:20;default:pending;index" json:"review_status"` // pending|approved|rejected
	ReviewerID    *uint      `gorm:"index" json:"reviewer_id"`                           // 审核人ID
	ReviewComment string     `gorm:"size:512" json:"review_comment"`                     // 审核意见
	ReviewedAt    *time.Time `json:"reviewed_at"`                                        // 审核时间
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// PostEditRequest 帖子编辑请求（待审核）
type PostEditRequest struct {
	ID          uint       `gorm:"primarykey" json:"id"`
	PostID      uint       `gorm:"uniqueIndex;not null" json:"post_id"`
	AuthorID    uint       `gorm:"index;not null" json:"author_id"`
	Title       string     `gorm:"size:256" json:"title"`
	Content     string     `gorm:"type:text" json:"content"`
	ContentType string     `gorm:"size:20" json:"content_type"`
	Category    string     `gorm:"size:20" json:"category"`
	Status      string     `gorm:"size:20;default:pending" json:"status"`
	ReviewerID  *uint      `gorm:"index" json:"reviewer_id"`
	ReviewedAt  *time.Time `json:"reviewed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// PostTag 帖子-标签关联
type PostTag struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	PostID    uint      `gorm:"uniqueIndex:idx_post_tag;not null" json:"post_id"`
	TagID     uint      `gorm:"uniqueIndex:idx_post_tag;not null" json:"tag_id"`
	AddedBy   uint      `gorm:"index" json:"added_by"`
	CreatedAt time.Time `json:"created_at"`
}

// Comment 评论
type Comment struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	PostID    uint      `gorm:"index;not null" json:"post_id"`
	AuthorID  uint      `gorm:"index;not null" json:"author_id"`
	Content   string    `gorm:"type:text;not null" json:"content"`
	ParentID  *uint     `gorm:"index" json:"parent_id"` // 父评论ID（支持回复）
	LikeCount int       `gorm:"default:0" json:"like_count"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// PostLike 帖子点赞
type PostLike struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	PostID    uint      `gorm:"uniqueIndex:idx_post_user;not null" json:"post_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_post_user;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// PostFavorite 帖子收藏
type PostFavorite struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	PostID    uint      `gorm:"uniqueIndex:idx_post_fav;not null" json:"post_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_post_fav;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// PostView 帖子浏览记录
type PostView struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	PostID    uint      `gorm:"uniqueIndex:idx_post_view_user;not null" json:"post_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_post_view_user;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CommentLike 评论点赞
type CommentLike struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CommentID uint      `gorm:"uniqueIndex:idx_comment_user;not null" json:"comment_id"`
	UserID    uint      `gorm:"uniqueIndex:idx_comment_user;not null" json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

// ========== 管理系统 ==========

// AdminActionLog 管理操作日志
type AdminActionLog struct {
	ID           uint      `gorm:"primarykey" json:"id"`
	OperatorID   uint      `gorm:"index;not null" json:"operator_id"`         // 操作者ID
	OperatorName string    `gorm:"size:50" json:"operator_name"`              // 操作者用户名（快照）
	OperatorRole string    `gorm:"size:20" json:"operator_role"`              // 操作者角色（快照）
	ActionType   string    `gorm:"size:50;index;not null" json:"action_type"` // 操作类型
	TargetType   string    `gorm:"size:20;index" json:"target_type"`          // 目标类型: post|item|guild|user
	TargetID     uint      `gorm:"index" json:"target_id"`                    // 目标ID
	TargetName   string    `gorm:"size:256" json:"target_name"`               // 目标名称（快照）
	Details      string    `gorm:"type:text" json:"details"`                  // 详情（JSON）
	IPAddress    string    `gorm:"size:45" json:"ip_address"`                 // IP地址
	CreatedAt    time.Time `json:"created_at"`
}

// DailyMetrics 每日统计快照（用于历史趋势）
type DailyMetrics struct {
	ID          uint      `gorm:"primarykey" json:"id"`
	Date        time.Time `gorm:"uniqueIndex;not null" json:"date"` // 日期（无时间）
	TotalUsers  int64     `json:"total_users"`                      // 累计用户数
	TotalPosts  int64     `json:"total_posts"`                      // 累计帖子数
	TotalItems  int64     `json:"total_items"`                      // 累计道具数
	TotalGuilds int64     `json:"total_guilds"`                     // 累计公会数
	NewUsers    int64     `json:"new_users"`                        // 当日新增用户
	NewPosts    int64     `json:"new_posts"`                        // 当日新增帖子
	NewItems    int64     `json:"new_items"`                        // 当日新增道具
	NewGuilds   int64     `json:"new_guilds"`                       // 当日新增公会
	CreatedAt   time.Time `json:"created_at"`
}

// ========== 通知系统 ==========

// Notification 通知
type Notification struct {
	ID         uint      `gorm:"primarykey" json:"id"`
	UserID     uint      `gorm:"index;not null" json:"user_id"`      // 接收通知的用户ID
	Type       string    `gorm:"size:20;index;not null" json:"type"` // 通知类型: post_like|post_comment|item_like|item_comment|guild_application|guild_invite|system
	ActorID    *uint     `gorm:"index" json:"actor_id"`              // 触发通知的用户ID（可空，系统通知无actor）
	TargetType string    `gorm:"size:20;index" json:"target_type"`   // 目标类型: post|item|comment|guild
	TargetID   uint      `gorm:"index" json:"target_id"`             // 目标ID
	Content    string    `gorm:"size:512" json:"content"`            // 通知内容
	IsRead     bool      `gorm:"default:false;index" json:"is_read"` // 是否已读
	CreatedAt  time.Time `json:"created_at"`
}
