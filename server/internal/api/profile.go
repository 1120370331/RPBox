package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/service"
)

func (s *Server) listProfiles(c *gin.Context) {
	userID := c.GetUint("user_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var profiles []model.Profile
	var total int64

	db := database.DB.Where("user_id = ?", userID)
	db.Model(&model.Profile{}).Count(&total)
	db.Offset((page - 1) * pageSize).Limit(pageSize).Find(&profiles)

	c.JSON(http.StatusOK, gin.H{
		"profiles":  profiles,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func (s *Server) createProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		ID          string `json:"id" binding:"required"`
		AccountID   string `json:"account_id"`
		ProfileName string `json:"profile_name"`
		RawLua      string `json:"raw_lua"`
		Checksum    string `json:"checksum"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	profile := model.Profile{
		ID:          req.ID,
		UserID:      userID,
		AccountID:   req.AccountID,
		ProfileName: req.ProfileName,
		RawLua:      req.RawLua,
		Checksum:    req.Checksum,
		Version:     1,
	}

	if err := database.DB.Create(&profile).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
		return
	}

	c.JSON(http.StatusCreated, profile)
}

func (s *Server) getProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var profile model.Profile
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

func (s *Server) updateProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var profile model.Profile
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		return
	}

	var req struct {
		ProfileName string `json:"profile_name"`
		RawLua      string `json:"raw_lua"`
		Checksum    string `json:"checksum"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 保存版本历史
	version := model.ProfileVersion{
		ProfileID: profile.ID,
		Version:   profile.Version,
		RawLua:    profile.RawLua,
		Checksum:  profile.Checksum,
	}
	database.DB.Create(&version)
	_ = service.CleanOldVersions(profile.ID)

	// 更新 Profile
	profile.ProfileName = req.ProfileName
	profile.RawLua = req.RawLua
	profile.Checksum = req.Checksum
	profile.Version++

	database.DB.Save(&profile)
	c.JSON(http.StatusOK, profile)
}

func (s *Server) deleteProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	result := database.DB.Where("id = ? AND user_id = ?", id, userID).Delete(&model.Profile{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

func (s *Server) getProfileVersions(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	// 验证 Profile 归属
	var profile model.Profile
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		return
	}

	var versions []model.ProfileVersion
	database.DB.Where("profile_id = ?", id).Order("version DESC").Limit(10).Find(&versions)

	c.JSON(http.StatusOK, gin.H{"versions": versions})
}

func (s *Server) rollbackProfile(c *gin.Context) {
	userID := c.GetUint("user_id")
	id := c.Param("id")

	var req struct {
		Version int `json:"version" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证 Profile 归属
	var profile model.Profile
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		return
	}

	// 查找目标版本
	var targetVersion model.ProfileVersion
	if err := database.DB.Where("profile_id = ? AND version = ?", id, req.Version).First(&targetVersion).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "版本不存在"})
		return
	}

	// 保存当前版本
	currentVersion := model.ProfileVersion{
		ProfileID: profile.ID,
		Version:   profile.Version,
		RawLua:    profile.RawLua,
		Checksum:  profile.Checksum,
		ChangeLog: "回滚前备份",
	}
	database.DB.Create(&currentVersion)
	_ = service.CleanOldVersions(profile.ID)

	// 回滚
	profile.RawLua = targetVersion.RawLua
	profile.Checksum = targetVersion.Checksum
	profile.Version++
	database.DB.Save(&profile)

	c.JSON(http.StatusOK, profile)
}
