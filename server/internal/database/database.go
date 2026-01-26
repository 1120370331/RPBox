package database

import (
	"fmt"
	"log"

	"github.com/rpbox/server/internal/config"
	"github.com/rpbox/server/internal/model"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(cfg *config.DatabaseConfig) error {
	sslmode := cfg.SSLMode
	if sslmode == "" {
		sslmode = "require"
	}

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, sslmode,
	)
	if cfg.SSLRootCert != "" {
		dsn += fmt.Sprintf(" sslrootcert=%s", cfg.SSLRootCert)
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("connect database with sslmode=%s: %w", sslmode, err)
	}

	// 自动迁移
	if err := db.AutoMigrate(
		&model.User{},
		&model.Profile{},
		&model.ProfileVersion{},
		&model.AccountBackup{},
		&model.AccountBackupVersion{},
		&model.Story{},
		&model.StoryEntry{},
		&model.Character{},
		&model.Tag{},
		&model.StoryTag{},
		&model.Guild{},
		&model.GuildMember{},
		&model.GuildApplication{},
		&model.StoryGuild{},
		&model.Item{},
		&model.ItemTag{},
		&model.ItemRating{},
		&model.ItemComment{},
		&model.ItemLike{},
		&model.ItemFavorite{},
		&model.ItemView{},
		&model.ItemDownload{},
		&model.ItemPendingEdit{},
		&model.ItemImage{},
		&model.Post{},
		&model.PostEditRequest{},
		&model.PostTag{},
		&model.Comment{},
		&model.PostLike{},
		&model.PostFavorite{},
		&model.PostView{},
		&model.CommentLike{},
		&model.AdminActionLog{},
		&model.DailyMetrics{},
		&model.Notification{},
		&model.Collection{},
		&model.CollectionPost{},
		&model.CollectionItem{},
		&model.CollectionFavorite{},
	); err != nil {
		return err
	}

	// 手动迁移：修改 checksum 列类型为 text
	migrations := []string{
		"ALTER TABLE account_backups ALTER COLUMN checksum TYPE text",
		"ALTER TABLE account_backup_versions ALTER COLUMN checksum TYPE text",
	}
	for _, sql := range migrations {
		if err := db.Exec(sql).Error; err != nil {
			log.Printf("[DB Migration] %s - %v", sql, err)
		}
	}
	if err := db.Exec("UPDATE users SET sponsor_level = 2 WHERE is_sponsor = true AND (sponsor_level IS NULL OR sponsor_level = 0)").Error; err != nil {
		log.Printf("[DB Migration] update sponsor_level from is_sponsor - %v", err)
	}

	// 添加性能优化索引
	indexMigrations := []string{
		// guild_members 表添加 user_id 单独索引，优化按用户查询公会成员
		"CREATE INDEX IF NOT EXISTS idx_guild_members_user_id ON guild_members(user_id)",
		// posts 表添加复合索引，优化活动列表查询
		"CREATE INDEX IF NOT EXISTS idx_posts_event_list ON posts(category, status, review_status, is_public) WHERE category = 'event'",
		// posts 表添加 status 索引
		"CREATE INDEX IF NOT EXISTS idx_posts_status ON posts(status)",
		// posts 表添加 is_public 索引
		"CREATE INDEX IF NOT EXISTS idx_posts_is_public ON posts(is_public)",
	}
	for _, sql := range indexMigrations {
		if err := db.Exec(sql).Error; err != nil {
			// 索引可能已存在，忽略错误
			log.Printf("[DB Index] %s - %v", sql, err)
		}
	}

	// 修复旧预设标签的 category 字段
	fixPresetTagCategories(db)

	DB = db

	// 初始化预设标签
	initPresetTags()

	return nil
}

// fixPresetTagCategories 修复旧预设标签的 category 字段
func fixPresetTagCategories(db *gorm.DB) {
	// 道具标签名称列表
	itemTagNames := []string{"普通道具", "可使用道具", "消耗品", "书籍", "多道具", "画作"}

	// 将道具标签的 category 设置为 item
	db.Model(&model.Tag{}).
		Where("name IN ? AND type = ? AND (category = '' OR category IS NULL OR category = 'story')", itemTagNames, "preset").
		Update("category", "item")

	// 将剧情标签的 category 设置为 story（如果为空）
	storyTagNames := []string{"主线剧情", "日常互动", "战斗场景", "社交活动"}
	db.Model(&model.Tag{}).
		Where("name IN ? AND type = ? AND (category = '' OR category IS NULL)", storyTagNames, "preset").
		Update("category", "story")
}

// initPresetTags 初始化预设标签
func initPresetTags() {
	// 剧情标签
	storyTags := []model.Tag{
		{Name: "主线剧情", Color: "B87333", Type: "preset", Category: "story", IsPublic: true},
		{Name: "日常互动", Color: "4682B4", Type: "preset", Category: "story", IsPublic: true},
		{Name: "战斗场景", Color: "DC143C", Type: "preset", Category: "story", IsPublic: true},
		{Name: "社交活动", Color: "9370DB", Type: "preset", Category: "story", IsPublic: true},
	}

	// 道具细分标签
	itemTags := []model.Tag{
		{Name: "普通道具", Color: "A08060", Type: "preset", Category: "item", IsPublic: true},
		{Name: "可使用道具", Color: "6B9B6B", Type: "preset", Category: "item", IsPublic: true},
		{Name: "消耗品", Color: "C98B7B", Type: "preset", Category: "item", IsPublic: true},
		{Name: "书籍", Color: "7B9BC7", Type: "preset", Category: "item", IsPublic: true},
		{Name: "多道具", Color: "A88BC7", Type: "preset", Category: "item", IsPublic: true},
		{Name: "画作", Color: "C9B370", Type: "preset", Category: "item", IsPublic: true},
	}

	allTags := append(storyTags, itemTags...)

	for _, tag := range allTags {
		var existing model.Tag
		if err := DB.Where("name = ? AND type = ? AND category = ?", tag.Name, "preset", tag.Category).First(&existing).Error; err != nil {
			DB.Create(&tag)
		}
	}
}
