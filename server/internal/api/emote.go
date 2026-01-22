package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

type emoteManifest struct {
	Packs []emotePackConfig `json:"packs"`
}

type emotePackConfig struct {
	ID    string            `json:"id"`
	Name  string            `json:"name"`
	Icon  string            `json:"icon"`
	Items []emoteItemConfig `json:"items"`
}

type emoteItemConfig struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Text   string `json:"text"`
	File   string `json:"file"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type emotePackResponse struct {
	ID      string              `json:"id"`
	Name    string              `json:"name"`
	IconURL string              `json:"icon_url"`
	Items   []emoteItemResponse `json:"items"`
}

type emoteItemResponse struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Text   string `json:"text,omitempty"`
	URL    string `json:"url"`
	Width  int    `json:"width,omitempty"`
	Height int    `json:"height,omitempty"`
}

func (s *Server) listEmotePacks(c *gin.Context) {
	manifestPath := filepath.Join(s.cfg.Storage.Path, "emotes", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"packs": []emotePackResponse{}})
		return
	}

	var manifest emoteManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "表情包配置解析失败"})
		return
	}

	packs := make([]emotePackResponse, 0, len(manifest.Packs))
	for _, pack := range manifest.Packs {
		items := make([]emoteItemResponse, 0, len(pack.Items))
		for _, item := range pack.Items {
			width := item.Width
			height := item.Height
			if width == 0 {
				width = 128
			}
			if height == 0 {
				height = 128
			}
			items = append(items, emoteItemResponse{
				ID:     item.ID,
				Name:   item.Name,
				Text:   item.Text,
				URL:    resolveEmoteURL(c, item.File),
				Width:  width,
				Height: height,
			})
		}
		packs = append(packs, emotePackResponse{
			ID:      pack.ID,
			Name:    pack.Name,
			IconURL: resolveEmoteURL(c, pack.Icon),
			Items:   items,
		})
	}

	c.JSON(http.StatusOK, gin.H{"packs": packs})
}

func resolveEmoteURL(c *gin.Context, raw string) string {
	if raw == "" {
		return ""
	}
	if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
		return raw
	}
	return buildPublicURL(c, raw)
}
