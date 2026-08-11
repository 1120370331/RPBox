package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/pkg/validator"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// listAccountBackups 获取用户所有账号备份。
func (s *Server) listAccountBackups(c *gin.Context) {
	userID := c.GetUint("user_id")
	var backups []model.AccountBackup
	if err := database.DB.Where("user_id = ?", userID).
		Select("id, user_id, account_id, profiles_count, tools_count, runtime_size_kb, checksum, version, created_at, updated_at").
		Order("updated_at DESC, id DESC").Find(&backups).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询备份失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"backups": backups})
}

// getAccountBackup 获取单个账号备份详情。
func (s *Server) getAccountBackup(c *gin.Context) {
	backup, ok := ownedAccountBackup(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, backup)
}

// upsertAccountBackup 创建或更新账号备份。每次内容变化都在同一事务中
// 先保存当前版本，避免客户端同步绕过版本保护。
func (s *Server) upsertAccountBackup(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req struct {
		AccountID      string `json:"account_id" binding:"required"`
		ProfilesData   string `json:"profiles_data" binding:"required"`
		ProfilesCount  int    `json:"profiles_count"`
		ToolsData      string `json:"tools_data"`
		ToolsCount     int    `json:"tools_count"`
		RuntimeData    string `json:"runtime_data"`
		RuntimeSizeKB  int    `json:"runtime_size_kb"`
		ConfigData     string `json:"config_data"`
		ExtraData      string `json:"extra_data"`
		RawTrp3Lua     string `json:"raw_trp3_lua"`
		RawTrp3Data    string `json:"raw_trp3_data_lua"`
		RawTrp3Ext     string `json:"raw_trp3_extended_lua"`
		Checksum       string `json:"checksum" binding:"required"`
		SnapshotReason string `json:"snapshot_reason"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": validator.TranslateError(err)})
		return
	}
	if err := validateCharacterCardSourceAccountID(req.AccountID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "account_id 不是安全的账号目录名"})
		return
	}
	if _, err := decodeCharacterCardProfileMap(req.ProfilesData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "profiles_data 必须是 profile 对象"})
		return
	}
	snapshotReason, err := validateAccountBackupSnapshotReason(req.SnapshotReason)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	log.Printf("[AccountBackup] upsert - account=%s, profiles=%d, tools_data_len=%d, tools_count=%d, runtime_data_len=%d, runtime_kb=%d, raw_trp3_len=%d, raw_data_len=%d, raw_ext_len=%d",
		req.AccountID, req.ProfilesCount, len(req.ToolsData), req.ToolsCount, len(req.RuntimeData), req.RuntimeSizeKB, len(req.RawTrp3Lua), len(req.RawTrp3Data), len(req.RawTrp3Ext))

	var existing model.AccountBackup
	err = database.DB.Where("user_id = ? AND account_id = ?", userID, req.AccountID).First(&existing).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		backup := model.AccountBackup{
			UserID: userID, AccountID: req.AccountID,
			ProfilesData: req.ProfilesData, ProfilesCount: req.ProfilesCount,
			ToolsData: req.ToolsData, ToolsCount: req.ToolsCount,
			RuntimeData: req.RuntimeData, RuntimeSizeKB: req.RuntimeSizeKB,
			ConfigData: req.ConfigData, ExtraData: req.ExtraData,
			RawTrp3Lua: req.RawTrp3Lua, RawTrp3Data: req.RawTrp3Data, RawTrp3Ext: req.RawTrp3Ext,
			Checksum: req.Checksum, Version: 1,
		}
		if err := database.DB.Create(&backup).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
			return
		}
		c.JSON(http.StatusCreated, backup)
		return
	}
	if existing.Checksum == req.Checksum {
		c.JSON(http.StatusOK, existing)
		return
	}

	err = database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&existing, existing.ID).Error; err != nil {
			return err
		}
		if _, err := createAccountBackupSnapshot(tx, existing, "", snapshotReason); err != nil {
			return err
		}
		existing.ProfilesData = req.ProfilesData
		existing.ProfilesCount = req.ProfilesCount
		existing.ToolsData = req.ToolsData
		existing.ToolsCount = req.ToolsCount
		existing.RuntimeData = req.RuntimeData
		existing.RuntimeSizeKB = req.RuntimeSizeKB
		existing.ConfigData = req.ConfigData
		existing.ExtraData = req.ExtraData
		existing.RawTrp3Lua = req.RawTrp3Lua
		existing.RawTrp3Data = req.RawTrp3Data
		existing.RawTrp3Ext = req.RawTrp3Ext
		existing.Checksum = req.Checksum
		existing.Version++
		return tx.Save(&existing).Error
	})
	if err != nil {
		log.Printf("[AccountBackup] update error user=%d account=%s: %v", userID, req.AccountID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}
	c.JSON(http.StatusOK, existing)
}

// deleteAccountBackup 删除账号备份及其版本。
func (s *Server) deleteAccountBackup(c *gin.Context) {
	backup, ok := ownedAccountBackup(c)
	if !ok {
		return
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("backup_id = ?", backup.ID).Delete(&model.AccountBackupVersion{}).Error; err != nil {
			return err
		}
		return tx.Delete(&backup).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// getAccountBackupVersions returns all metadata. Large version payloads are
// available only through the owner-only detail endpoint.
func (s *Server) getAccountBackupVersions(c *gin.Context) {
	backup, ok := ownedAccountBackup(c)
	if !ok {
		return
	}
	var versions []model.AccountBackupVersion
	if err := database.DB.Where("backup_id = ?", backup.ID).
		Select("id", "backup_id", "version", "name", "content_hash", "profiles_count", "tools_count", "runtime_size_kb", "checksum", "change_log", "created_at").
		Order("created_at DESC, id DESC").Find(&versions).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询版本失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

func (s *Server) getAccountBackupVersion(c *gin.Context) {
	_, version, ok := ownedAccountBackupVersion(c)
	if !ok {
		return
	}
	c.JSON(http.StatusOK, gin.H{"version": version})
}

func (s *Server) renameAccountBackupVersion(c *gin.Context) {
	backup, version, ok := ownedAccountBackupVersion(c)
	if !ok {
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name 无效"})
		return
	}
	name, err := validateAccountBackupVersionName(req.Name)
	if err != nil || name == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "name 不能为空且不能超过 128 个字符"})
		return
	}
	if err := database.DB.Model(&model.AccountBackupVersion{}).
		Where("id = ? AND backup_id = ?", version.ID, backup.ID).Update("name", name).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "命名版本失败"})
		return
	}
	version.Name = name
	c.JSON(http.StatusOK, gin.H{"version": version})
}

func (s *Server) restoreAccountBackupVersion(c *gin.Context) {
	backup, version, ok := ownedAccountBackupVersion(c)
	if !ok {
		return
	}
	var req struct {
		SnapshotName string `json:"snapshot_name"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}
	}
	var protection model.AccountBackupVersion
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", backup.ID, backup.UserID).First(&backup).Error; err != nil {
			return err
		}
		var err error
		protection, err = createAccountBackupSnapshot(tx, backup, req.SnapshotName, "before_restore")
		if err != nil {
			return err
		}
		backup.ProfilesData = version.ProfilesData
		backup.ProfilesCount = version.ProfilesCount
		backup.ToolsData = version.ToolsData
		backup.ToolsCount = version.ToolsCount
		backup.RuntimeData = version.RuntimeData
		backup.RuntimeSizeKB = version.RuntimeSizeKB
		backup.ConfigData = version.ConfigData
		backup.ExtraData = version.ExtraData
		backup.RawTrp3Lua = version.RawTrp3Lua
		backup.RawTrp3Data = version.RawTrp3Data
		backup.RawTrp3Ext = version.RawTrp3Ext
		backup.Checksum = version.Checksum
		backup.Version++
		return tx.Save(&backup).Error
	})
	if err != nil {
		if _, nameErr := validateAccountBackupVersionName(req.SnapshotName); nameErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": nameErr.Error()})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "回退版本失败"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"backup": backup, "snapshot": protection, "restored_version": version})
}

func (s *Server) deleteAccountBackupVersion(c *gin.Context) {
	backup, version, ok := ownedAccountBackupVersion(c)
	if !ok {
		return
	}
	result := database.DB.Where("id = ? AND backup_id = ?", version.ID, backup.ID).Delete(&model.AccountBackupVersion{})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "删除版本失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func ownedAccountBackup(c *gin.Context) (model.AccountBackup, bool) {
	userID := c.GetUint("user_id")
	accountID := c.Param("account_id")
	if err := validateCharacterCardSourceAccountID(accountID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "备份不存在"})
		return model.AccountBackup{}, false
	}
	var backup model.AccountBackup
	if err := database.DB.Where("user_id = ? AND account_id = ?", userID, accountID).First(&backup).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "备份不存在"})
		return model.AccountBackup{}, false
	}
	return backup, true
}

func ownedAccountBackupVersion(c *gin.Context) (model.AccountBackup, model.AccountBackupVersion, bool) {
	backup, ok := ownedAccountBackup(c)
	if !ok {
		return model.AccountBackup{}, model.AccountBackupVersion{}, false
	}
	versionID, err := strconv.ParseUint(c.Param("version_id"), 10, 32)
	if err != nil || versionID == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "版本不存在"})
		return model.AccountBackup{}, model.AccountBackupVersion{}, false
	}
	var version model.AccountBackupVersion
	if err := database.DB.Where("id = ? AND backup_id = ?", uint(versionID), backup.ID).First(&version).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "版本不存在"})
		return model.AccountBackup{}, model.AccountBackupVersion{}, false
	}
	return backup, version, true
}

func accountBackupContentHash(backup model.AccountBackup) string {
	payload, _ := json.Marshal(struct {
		ProfilesData string `json:"profiles_data"`
		ToolsData    string `json:"tools_data"`
		RuntimeData  string `json:"runtime_data"`
		ConfigData   string `json:"config_data"`
		ExtraData    string `json:"extra_data"`
		RawTrp3Lua   string `json:"raw_trp3_lua"`
		RawTrp3Data  string `json:"raw_trp3_data_lua"`
		RawTrp3Ext   string `json:"raw_trp3_extended_lua"`
	}{backup.ProfilesData, backup.ToolsData, backup.RuntimeData, backup.ConfigData, backup.ExtraData, backup.RawTrp3Lua, backup.RawTrp3Data, backup.RawTrp3Ext})
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func validateAccountBackupVersionName(raw string) (string, error) {
	name := strings.TrimSpace(raw)
	if name == "" {
		return "", nil
	}
	if err := validatePlainCharacterCardField("name", name, 128); err != nil {
		return "", err
	}
	return name, nil
}

func validateAccountBackupSnapshotReason(raw string) (string, error) {
	reason := strings.TrimSpace(raw)
	if reason == "" {
		return "sync", nil
	}
	switch reason {
	case "before_manual_backup", "restore_sync", "sync":
		return reason, nil
	default:
		return "", errors.New("snapshot_reason 无效")
	}
}

func createAccountBackupSnapshot(tx *gorm.DB, backup model.AccountBackup, requestedName, reason string) (model.AccountBackupVersion, error) {
	name, err := validateAccountBackupVersionName(requestedName)
	if err != nil {
		return model.AccountBackupVersion{}, err
	}
	hash := accountBackupContentHash(backup)
	if name == "" {
		name = fmt.Sprintf("%s %s", time.Now().Format("2006-01-02 15:04:05"), hash[:10])
	}
	version := model.AccountBackupVersion{
		BackupID: backup.ID, Version: backup.Version, Name: name, ContentHash: hash,
		ProfilesData: backup.ProfilesData, ProfilesCount: backup.ProfilesCount,
		ToolsData: backup.ToolsData, ToolsCount: backup.ToolsCount,
		RuntimeData: backup.RuntimeData, RuntimeSizeKB: backup.RuntimeSizeKB,
		ConfigData: backup.ConfigData, ExtraData: backup.ExtraData,
		RawTrp3Lua: backup.RawTrp3Lua, RawTrp3Data: backup.RawTrp3Data, RawTrp3Ext: backup.RawTrp3Ext,
		Checksum: backup.Checksum, ChangeLog: strings.TrimSpace(reason),
	}
	if err := tx.Create(&version).Error; err != nil {
		return model.AccountBackupVersion{}, err
	}
	return version, nil
}
