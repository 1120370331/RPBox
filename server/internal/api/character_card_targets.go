package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type characterCardSaveTarget struct {
	BackupID        uint   `json:"backup_id"`
	ProfileID       string `json:"profile_id"`
	ExpectedVersion int    `json:"expected_version"`
}

func validGameFlavor(flavor string) bool {
	switch flavor {
	case "", "_retail_", "_classic_", "_classic_era_", "_classic_anniversary_", "_ptr_", "_beta_", "_classic_ptr_", "_classic_beta_":
		return true
	}
	return false
}

// labelLegacyAccountBackup assigns an explicit edition without merging any archives.
func (s *Server) labelLegacyAccountBackup(c *gin.Context) {
	backup, ok := ownedAccountBackup(c)
	if !ok {
		return
	}
	var req struct {
		GameFlavor string `json:"game_flavor"`
	}
	if c.ShouldBindJSON(&req) != nil || req.GameFlavor == "" || !validGameFlavor(req.GameFlavor) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择游戏版本"})
		return
	}
	if backup.GameFlavor != "" {
		c.JSON(http.StatusConflict, gin.H{"error": "已标记版本的存档不能改到其他版本"})
		return
	}
	result := database.DB.Model(&model.AccountBackup{}).Where("id = ? AND user_id = ? AND game_flavor = ''", backup.ID, backup.UserID).Update("game_flavor", req.GameFlavor)
	if result.Error != nil || result.RowsAffected != 1 {
		c.JSON(http.StatusConflict, gin.H{"error": "该版本已有同名子账号或存档已变化；未合并、覆盖任何数据"})
		return
	}
	backup.GameFlavor = req.GameFlavor
	c.JSON(http.StatusOK, backup)
}

func (s *Server) getCharacterCardTargets(c *gin.Context) {
	id, ok := parseCharacterCardID(c)
	if !ok {
		return
	}
	var card model.CharacterCard
	if err := database.DB.Where("id = ? AND user_id = ?", id, c.GetUint("userID")).First(&card).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		return
	}
	targets := []model.CharacterCardTarget{}
	if err := database.DB.Where("character_card_id = ?", id).Find(&targets).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取保存目标失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"targets": targets})
}

// ensureAccountBackup registers an empty cloud target without overwriting an existing archive.
func (s *Server) ensureAccountBackup(c *gin.Context) {
	var req struct {
		AccountID  string `json:"account_id"`
		GameFlavor string `json:"game_flavor"`
	}
	if c.ShouldBindJSON(&req) != nil || validateCharacterCardSourceAccountID(req.AccountID) != nil || !validGameFlavor(req.GameFlavor) || req.GameFlavor == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择游戏版本并填写有效子账号"})
		return
	}
	backup := model.AccountBackup{UserID: c.GetUint("user_id"), AccountID: req.AccountID, GameFlavor: req.GameFlavor, ProfilesData: "{}", Version: 1}
	backup.Checksum = accountBackupContentHash(backup)
	if err := database.DB.Clauses(clause.OnConflict{DoNothing: true}).Create(&backup).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "添加子账号失败"})
		return
	}
	if err := database.DB.Where("user_id = ? AND account_id = ? AND game_flavor = ?", backup.UserID, req.AccountID, req.GameFlavor).First(&backup).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取子账号失败"})
		return
	}
	c.JSON(http.StatusOK, backup)
}

func validateCloudTargets(targets []characterCardSaveTarget) error {
	if len(targets) == 0 || len(targets) > 20 {
		return errors.New("请选择 1 到 20 个保存子账号")
	}
	seen := map[uint]bool{}
	for _, target := range targets {
		if target.BackupID == 0 || seen[target.BackupID] || target.ExpectedVersion < 1 {
			return errors.New("保存目标重复或版本无效，请刷新后重选")
		}
		if err := validateTRP3ProfileID(target.ProfileID); err != nil {
			return err
		}
		seen[target.BackupID] = true
	}
	return nil
}

// saveCharacterCardTargets runs inside the card update transaction. No local file operation is performed.
func saveCharacterCardTargets(tx *gorm.DB, card model.CharacterCard, impressions []model.CharacterCardImpression, targets []characterCardSaveTarget) error {
	sort.Slice(targets, func(i, j int) bool { return targets[i].BackupID < targets[j].BackupID })
	for _, target := range targets {
		var backup model.AccountBackup
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", target.BackupID, card.UserID).First(&backup).Error; err != nil {
			return &apiError{status: http.StatusNotFound, message: "保存目标不存在或不属于当前账号"}
		}
		if backup.Version != target.ExpectedVersion {
			return &apiError{status: http.StatusConflict, message: "云端存档已变化，请刷新保存目标后重试"}
		}
		var occupied model.CharacterCardTarget
		err := tx.Where("backup_id = ? AND profile_id = ?", backup.ID, target.ProfileID).First(&occupied).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil && occupied.CharacterCardID != card.ID {
			return &apiError{status: http.StatusConflict, message: "目标人物卡已关联其他 RPBox 档案，请选择其他人物卡"}
		}
		profiles, err := decodeTRP3ProfilesObject(backup.ProfilesData)
		if err != nil {
			return &apiError{status: http.StatusBadRequest, message: "目标存档格式损坏，未保存任何修改"}
		}
		existing := profiles[target.ProfileID]
		if existing == nil && card.TRP3ProfileJSON != "" {
			if err := json.Unmarshal([]byte(card.TRP3ProfileJSON), &existing); err != nil {
				return err
			}
		}
		profile, err := buildTRP3ProfileFromCharacterCard(card, impressions, existing)
		if err != nil {
			return err
		}
		profiles[target.ProfileID] = profile
		encoded, err := json.Marshal(profiles)
		if err != nil {
			return err
		}
		if _, err := createAccountBackupSnapshot(tx, backup, "", "before_character_card_writeback"); err != nil {
			return err
		}
		if strings.TrimSpace(backup.RawTrp3Lua) != "" {
			backup.RawTrp3Lua, err = replaceTRP3ProfilesLua(backup.RawTrp3Lua, encodeLuaValue(profiles, 0))
			if err != nil {
				return fmt.Errorf("更新原始 Lua: %w", err)
			}
		}
		backup.ProfilesData = string(encoded)
		backup.ProfilesCount = len(profiles)
		backup.Version++
		backup.Checksum = accountBackupContentHash(backup)
		if err := tx.Save(&backup).Error; err != nil {
			return err
		}
		binding := model.CharacterCardTarget{CharacterCardID: card.ID, BackupID: backup.ID, ProfileID: target.ProfileID}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "character_card_id"}, {Name: "backup_id"}}, DoUpdates: clause.AssignmentColumns([]string{"profile_id", "updated_at"})}).Create(&binding).Error; err != nil {
			return err
		}
	}
	return nil
}
