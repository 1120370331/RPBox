package api

import (
	"encoding/base64"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

// getUserInfo 获取当前用户信息
func (s *Server) getUserInfo(c *gin.Context) {
	userID := c.GetUint("userID")

	var user model.User
	if err := database.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "用户不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
		"avatar":   user.Avatar,
	})
}

// updateAvatar 更新用户头像
func (s *Server) updateAvatar(c *gin.Context) {
	userID := c.GetUint("userID")

	file, header, err := c.Request.FormFile("avatar")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择头像文件"})
		return
	}
	defer file.Close()

	// 检查文件大小 (最大 2MB)
	if header.Size > 2*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "头像文件不能超过2MB"})
		return
	}

	// 检查文件类型
	contentType := header.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只支持图片格式"})
		return
	}

	// 读取文件内容
	data, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取文件失败"})
		return
	}

	// 转换为 base64
	base64Data := base64.StdEncoding.EncodeToString(data)
	avatarURL := "data:" + contentType + ";base64," + base64Data

	// 更新数据库
	if err := database.DB.Model(&model.User{}).Where("id = ?", userID).Update("avatar", avatarURL).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新头像失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "头像更新成功",
		"avatar":  avatarURL,
	})
}

// updateUserInfo 更新用户信息
func (s *Server) updateUserInfo(c *gin.Context) {
	userID := c.GetUint("userID")

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := make(map[string]interface{})
	if req.Username != "" {
		updates["username"] = req.Username
	}
	if req.Email != "" {
		updates["email"] = req.Email
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "没有要更新的内容"})
		return
	}

	if err := database.DB.Model(&model.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}
