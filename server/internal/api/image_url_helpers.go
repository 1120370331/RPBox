package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/rpbox/server/internal/model"
)

func buildAPIURL(apiHost, path string) string {
	if path == "" {
		return ""
	}
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		return path
	}
	host := strings.TrimRight(strings.TrimSpace(apiHost), "/")
	if host == "" {
		return path
	}
	if strings.HasPrefix(path, "/") {
		return host + path
	}
	return host + "/" + path
}

func userAvatarURL(apiHost string, user model.User) string {
	if strings.TrimSpace(user.Avatar) == "" {
		return ""
	}
	return buildAPIURL(apiHost, fmt.Sprintf("/api/v1/images/user-avatar/%d?w=80&q=80", user.ID))
}

func guildBannerURL(guild model.Guild) string {
	if strings.TrimSpace(guild.Banner) == "" {
		return ""
	}
	version := guild.UpdatedAt.Unix()
	if guild.BannerUpdatedAt != nil {
		version = guild.BannerUpdatedAt.Unix()
	}
	return fmt.Sprintf("/api/v1/images/guild-banner/%d?w=600&q=80&v=%d", guild.ID, version)
}

func guildAvatarURL(guild model.Guild) string {
	if strings.TrimSpace(guild.Avatar) == "" {
		return ""
	}
	version := guild.UpdatedAt.Unix()
	if guild.AvatarUpdatedAt != nil {
		version = guild.AvatarUpdatedAt.Unix()
	}
	return fmt.Sprintf("/api/v1/images/guild-avatar/%d?w=200&q=80&v=%d", guild.ID, version)
}

func postCoverURL(post model.Post) string {
	if strings.TrimSpace(post.CoverImage) == "" {
		return ""
	}
	version := post.UpdatedAt.Unix()
	if post.CoverImageUpdatedAt != nil {
		version = post.CoverImageUpdatedAt.Unix()
	}
	return fmt.Sprintf("/api/v1/images/post-cover/%d?w=600&q=80&v=%d", post.ID, version)
}

func ensureItemPreviewUpdatedAt(item *model.Item) {
	if item.PreviewImageUpdatedAt != nil {
		return
	}
	if strings.TrimSpace(item.PreviewImage) == "" {
		return
	}
	t := item.UpdatedAt
	item.PreviewImageUpdatedAt = &t
}

func ensurePostCoverUpdatedAt(post *model.Post) {
	if post.CoverImageUpdatedAt != nil {
		return
	}
	if strings.TrimSpace(post.CoverImage) == "" {
		return
	}
	t := post.UpdatedAt
	post.CoverImageUpdatedAt = &t
}

func ensureGuildBannerUpdatedAt(guild *model.Guild) {
	if guild.BannerUpdatedAt != nil {
		return
	}
	if strings.TrimSpace(guild.Banner) == "" {
		return
	}
	t := guild.UpdatedAt
	guild.BannerUpdatedAt = &t
}

func ensureGuildAvatarUpdatedAt(guild *model.Guild) {
	if guild.AvatarUpdatedAt != nil {
		return
	}
	if strings.TrimSpace(guild.Avatar) == "" {
		return
	}
	t := guild.UpdatedAt
	guild.AvatarUpdatedAt = &t
}

func nowPtr() *time.Time {
	t := time.Now()
	return &t
}
