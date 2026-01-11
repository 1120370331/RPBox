package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (s *Server) listItems(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"items": []any{}})
}

func (s *Server) createItem(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "created"})
}
