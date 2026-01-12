package service

import (
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

const MaxVersions = 10

// GetVersions 获取版本历史列表
func GetVersions(profileID string) ([]model.ProfileVersion, error) {
	var versions []model.ProfileVersion
	err := database.DB.
		Where("profile_id = ?", profileID).
		Order("version DESC").
		Limit(MaxVersions).
		Find(&versions).Error
	return versions, err
}

// CleanOldVersions 清理超过限制的旧版本
func CleanOldVersions(profileID string) error {
	var count int64
	database.DB.Model(&model.ProfileVersion{}).
		Where("profile_id = ?", profileID).
		Count(&count)

	if count > MaxVersions {
		var oldVersions []model.ProfileVersion
		database.DB.
			Where("profile_id = ?", profileID).
			Order("version ASC").
			Limit(int(count - MaxVersions)).
			Find(&oldVersions)

		for _, v := range oldVersions {
			database.DB.Delete(&v)
		}
	}
	return nil
}
