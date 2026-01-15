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
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return err
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
		&model.StoryGuild{},
		&model.Item{},
		&model.ItemTag{},
		&model.ItemRating{},
		&model.ItemComment{},
		&model.ItemLike{},
		&model.ItemFavorite{},
		&model.ItemDownload{},
		&model.ItemPendingEdit{},
		&model.Post{},
		&model.PostEditRequest{},
		&model.PostTag{},
		&model.Comment{},
		&model.PostLike{},
		&model.PostFavorite{},
		&model.CommentLike{},
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

	DB = db

	// 初始化预设标签
	initPresetTags()

	return nil
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
