package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) listProfiles(c *gin.Context) {
	// TODO: 获取人物卡列表
	c.JSON(http.StatusOK, gin.H{"profiles": []any{}})
}

func (s *Server) createProfile(c *gin.Context) {
	// TODO: 创建人物卡
	c.JSON(http.StatusCreated, gin.H{"message": "created"})
}

func (s *Server) getProfile(c *gin.Context) {
	id := c.Param("id")
	// TODO: 获取单个人物卡
	c.JSON(http.StatusOK, gin.H{"id": id})
}
