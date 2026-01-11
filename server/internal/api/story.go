package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) listStories(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"stories": []any{}})
}

func (s *Server) createStory(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "created"})
}
