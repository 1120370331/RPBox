package api

import (
	"fmt"

	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
)

// deleteGuildRecords removes guild-owned relationships while preserving user content.
func deleteGuildRecords(tx *gorm.DB, guildIDs []uint) error {
	if len(guildIDs) == 0 {
		return nil
	}

	var tagIDs []uint
	if err := tx.Model(&model.Tag{}).
		Where("guild_id IN ?", guildIDs).
		Pluck("id", &tagIDs).Error; err != nil {
		return fmt.Errorf("failed to collect guild tags: %w", err)
	}

	if err := deleteNotificationsByTarget(tx, "guild", guildIDs); err != nil {
		return fmt.Errorf("failed to delete guild notifications: %w", err)
	}
	if err := tx.Model(&model.Post{}).
		Where("guild_id IN ?", guildIDs).
		Update("guild_id", nil).Error; err != nil {
		return fmt.Errorf("failed to detach guild posts: %w", err)
	}
	if len(tagIDs) > 0 {
		if err := tx.Where("tag_id IN ?", tagIDs).Delete(&model.StoryTag{}).Error; err != nil {
			return fmt.Errorf("failed to delete guild story tags: %w", err)
		}
		if err := tx.Where("tag_id IN ?", tagIDs).Delete(&model.ItemTag{}).Error; err != nil {
			return fmt.Errorf("failed to delete guild item tags: %w", err)
		}
		if err := tx.Where("tag_id IN ?", tagIDs).Delete(&model.PostTag{}).Error; err != nil {
			return fmt.Errorf("failed to delete guild post tags: %w", err)
		}
		if err := tx.Where("tag_id IN ?", tagIDs).Delete(&model.RPDBTag{}).Error; err != nil {
			return fmt.Errorf("failed to delete guild RPDB tags: %w", err)
		}
	}
	if err := detachDeletedGuildsFromRPDBWorks(tx, guildIDs); err != nil {
		return err
	}
	if err := tx.Where("guild_id IN ?", guildIDs).Delete(&model.GuildApplication{}).Error; err != nil {
		return fmt.Errorf("failed to delete guild applications: %w", err)
	}
	if err := tx.Where("guild_id IN ?", guildIDs).Delete(&model.GuildMember{}).Error; err != nil {
		return fmt.Errorf("failed to delete guild members: %w", err)
	}
	if err := tx.Where("guild_id IN ?", guildIDs).Delete(&model.StoryGuild{}).Error; err != nil {
		return fmt.Errorf("failed to delete guild story archives: %w", err)
	}
	if err := tx.Where("guild_id IN ?", guildIDs).Delete(&model.Tag{}).Error; err != nil {
		return fmt.Errorf("failed to delete guild tags: %w", err)
	}
	if err := tx.Where("id IN ?", guildIDs).Delete(&model.Guild{}).Error; err != nil {
		return fmt.Errorf("failed to delete guilds: %w", err)
	}

	return nil
}

func detachDeletedGuildsFromRPDBWorks(tx *gorm.DB, guildIDs []uint) error {
	removedGuildIDs := make(map[uint]struct{}, len(guildIDs))
	for _, guildID := range guildIDs {
		removedGuildIDs[guildID] = struct{}{}
	}

	var works []model.RPDBWork
	if err := tx.Select("id", "guild_id", "guild_ids", "visibility", "is_public").
		Where("visibility = ?", model.RPDBVisibilityGuild).
		Find(&works).Error; err != nil {
		return fmt.Errorf("failed to collect guild RPDB works: %w", err)
	}

	for i := range works {
		work := &works[i]
		currentGuildIDs := normalizeRPDBGuildIDs(work.GuildIDs, work.GuildID)
		remainingGuildIDs := make([]uint, 0, len(currentGuildIDs))
		for _, guildID := range currentGuildIDs {
			if _, removed := removedGuildIDs[guildID]; !removed {
				remainingGuildIDs = append(remainingGuildIDs, guildID)
			}
		}
		if len(remainingGuildIDs) == len(currentGuildIDs) {
			continue
		}

		work.GuildIDs = remainingGuildIDs
		work.GuildID = firstRPDBGuildID(remainingGuildIDs)
		work.IsPublic = false
		if len(remainingGuildIDs) == 0 {
			work.Visibility = model.RPDBVisibilityPrivate
		}
		if err := tx.Model(work).
			Select("guild_id", "guild_ids", "visibility", "is_public").
			Updates(work).Error; err != nil {
			return fmt.Errorf("failed to detach guild RPDB work %d: %w", work.ID, err)
		}
	}

	return nil
}
