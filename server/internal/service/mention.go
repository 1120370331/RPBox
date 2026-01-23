package service

import (
	"net/url"
	"regexp"
	"strconv"

	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

var (
	mentionTokenRe = regexp.MustCompile(`\[\[mention:(\d+):([^\]]+)\]\]`)
	mentionHTMLRe  = regexp.MustCompile(`data-mention-id=['"](\d+)['"]`)
)

// ExtractMentionIDs extracts unique mention user IDs from text/HTML content.
func ExtractMentionIDs(contents ...string) []uint {
	ids := make([]uint, 0)
	seen := make(map[uint]struct{})

	for _, content := range contents {
		if content == "" {
			continue
		}

		for _, match := range mentionTokenRe.FindAllStringSubmatch(content, -1) {
			if len(match) < 2 {
				continue
			}
			id, err := strconv.ParseUint(match[1], 10, 64)
			if err != nil || id == 0 {
				continue
			}
			if _, ok := seen[uint(id)]; ok {
				continue
			}
			seen[uint(id)] = struct{}{}
			ids = append(ids, uint(id))
		}

		for _, match := range mentionHTMLRe.FindAllStringSubmatch(content, -1) {
			if len(match) < 2 {
				continue
			}
			id, err := strconv.ParseUint(match[1], 10, 64)
			if err != nil || id == 0 {
				continue
			}
			if _, ok := seen[uint(id)]; ok {
				continue
			}
			seen[uint(id)] = struct{}{}
			ids = append(ids, uint(id))
		}
	}

	return ids
}

// NormalizeMentionPreview replaces mention tokens with @label for notifications.
func NormalizeMentionPreview(input string) string {
	if input == "" {
		return input
	}
	return mentionTokenRe.ReplaceAllStringFunc(input, func(match string) string {
		sub := mentionTokenRe.FindStringSubmatch(match)
		if len(sub) < 3 {
			return match
		}
		label, err := url.QueryUnescape(sub[2])
		if err != nil || label == "" {
			return "@" + sub[2]
		}
		return "@" + label
	})
}

// CreateMentionNotifications sends mention notifications for the provided content.
func CreateMentionNotifications(actorID uint, targetType string, targetID uint, message string, contents ...string) {
	mentionIDs := ExtractMentionIDs(contents...)
	if len(mentionIDs) == 0 {
		return
	}

	filtered := make([]uint, 0, len(mentionIDs))
	for _, id := range mentionIDs {
		if id == 0 || id == actorID {
			continue
		}
		filtered = append(filtered, id)
	}
	if len(filtered) == 0 {
		return
	}

	// Ensure users exist.
	var validIDs []uint
	database.DB.Model(&model.User{}).Select("id").Where("id IN ?", filtered).Pluck("id", &validIDs)
	if len(validIDs) == 0 {
		return
	}

	// Skip duplicates from the same actor and target.
	var existing []model.Notification
	database.DB.
		Select("user_id").
		Where("type = ? AND target_type = ? AND target_id = ? AND actor_id = ? AND user_id IN ?",
			"mention", targetType, targetID, actorID, validIDs).
		Find(&existing)

	existingMap := make(map[uint]struct{}, len(existing))
	for _, notif := range existing {
		existingMap[notif.UserID] = struct{}{}
	}

	for _, userID := range validIDs {
		if _, ok := existingMap[userID]; ok {
			continue
		}
		actor := actorID
		notification := model.Notification{
			UserID:     userID,
			Type:       "mention",
			ActorID:    &actor,
			TargetType: targetType,
			TargetID:   targetID,
			Content:    message,
		}
		_ = CreateNotification(&notification)
	}
}
