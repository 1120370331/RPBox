package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/pkg/validator"
)

// listAccountBackups 获取用户所有账号备份
func (s *Server) listAccountBackups(c *gin.Context) {
	userID := c.GetUint("user_id")

	var backups []model.AccountBackup
	database.DB.Where("user_id = ?", userID).
		Select("id, user_id, account_id, profiles_count, tools_count, runtime_size_kb, checksum, version, created_at, updated_at").
		Find(&backups)

	// 调试日志
	for _, b := range backups {
		log.Printf("[AccountBackup] list - account=%s, checksum=%s, version=%d", b.AccountID, b.Checksum, b.Version)
	}

	c.JSON(http.StatusOK, gin.H{"backups": backups})
}

// getAccountBackup 获取单个账号备份详情
func (s *Server) getAccountBackup(c *gin.Context) {
	userID := c.GetUint("user_id")
	accountID := c.Param("account_id")

	var backup model.AccountBackup
	if err := database.DB.Where("user_id = ? AND account_id = ?", userID, accountID).First(&backup).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "备份不存在"})
		return
	}

	c.JSON(http.StatusOK, backup)
}

// upsertAccountBackup 创建或更新账号备份
func (s *Server) upsertAccountBackup(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		AccountID     string `json:"account_id" binding:"required"`
		ProfilesData  string `json:"profiles_data" binding:"required"`
		ProfilesCount int    `json:"profiles_count"`
		ToolsData     string `json:"tools_data"`
		ToolsCount    int    `json:"tools_count"`
		RuntimeData   string `json:"runtime_data"`
		RuntimeSizeKB int    `json:"runtime_size_kb"`
		ConfigData    string `json:"config_data"`
		ExtraData     string `json:"extra_data"`
		Checksum      string `json:"checksum" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}

	// 调试日志：打印接收到的数据长度
	log.Printf("[AccountBackup] upsert - account=%s, profiles=%d, tools_data_len=%d, tools_count=%d, runtime_data_len=%d, runtime_kb=%d",
		req.AccountID, req.ProfilesCount, len(req.ToolsData), req.ToolsCount, len(req.RuntimeData), req.RuntimeSizeKB)

	checksum := req.Checksum

	var existing model.AccountBackup
	err := database.DB.Where("user_id = ? AND account_id = ?", userID, req.AccountID).First(&existing).Error

	if err != nil {
		// 创建新备份
		backup := model.AccountBackup{
			UserID:        userID,
			AccountID:     req.AccountID,
			ProfilesData:  req.ProfilesData,
			ProfilesCount: req.ProfilesCount,
			ToolsData:     req.ToolsData,
			ToolsCount:    req.ToolsCount,
			RuntimeData:   req.RuntimeData,
			RuntimeSizeKB: req.RuntimeSizeKB,
			ConfigData:    req.ConfigData,
			ExtraData:     req.ExtraData,
			Checksum:      checksum,
			Version:       1,
		}
		if err := database.DB.Create(&backup).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
			return
		}
		c.JSON(http.StatusCreated, backup)
		return
	}

	// 检查是否有变化
	if existing.Checksum == checksum {
		c.JSON(http.StatusOK, existing)
		return
	}

	// 保存版本历史
	version := model.AccountBackupVersion{
		BackupID:     existing.ID,
		Version:      existing.Version,
		ProfilesData: existing.ProfilesData,
		ToolsData:    existing.ToolsData,
		RuntimeData:  existing.RuntimeData,
		ConfigData:   existing.ConfigData,
		ExtraData:    existing.ExtraData,
		Checksum:     existing.Checksum,
	}
	database.DB.Create(&version)
	cleanOldBackupVersions(existing.ID)

	// 更新备份
	existing.ProfilesData = req.ProfilesData
	existing.ProfilesCount = req.ProfilesCount
	existing.ToolsData = req.ToolsData
	existing.ToolsCount = req.ToolsCount
	existing.RuntimeData = req.RuntimeData
	existing.RuntimeSizeKB = req.RuntimeSizeKB
	existing.ConfigData = req.ConfigData
	existing.ExtraData = req.ExtraData
	existing.Checksum = checksum
	existing.Version++
	if err := database.DB.Save(&existing).Error; err != nil {
		log.Printf("[AccountBackup] Save error: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, existing)
}

// deleteAccountBackup 删除账号备份
func (s *Server) deleteAccountBackup(c *gin.Context) {
	userID := c.GetUint("user_id")
	accountID := c.Param("account_id")

	result := database.DB.Where("user_id = ? AND account_id = ?", userID, accountID).Delete(&model.AccountBackup{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "备份不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// getAccountBackupVersions 获取版本历史
func (s *Server) getAccountBackupVersions(c *gin.Context) {
	userID := c.GetUint("user_id")
	accountID := c.Param("account_id")

	var backup model.AccountBackup
	if err := database.DB.Where("user_id = ? AND account_id = ?", userID, accountID).First(&backup).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "备份不存在"})
		return
	}

	var versions []model.AccountBackupVersion
	database.DB.Where("backup_id = ?", backup.ID).Order("version DESC").Limit(10).Find(&versions)

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

// cleanOldBackupVersions 清理旧版本，保留最近10个
func cleanOldBackupVersions(backupID uint) {
	var count int64
	database.DB.Model(&model.AccountBackupVersion{}).Where("backup_id = ?", backupID).Count(&count)
	if count > 10 {
		var oldest model.AccountBackupVersion
		database.DB.Where("backup_id = ?", backupID).Order("version ASC").First(&oldest)
		database.DB.Delete(&oldest)
	}
}
