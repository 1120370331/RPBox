package api

import (
	"bytes"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/nfnt/resize"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

// getImage 获取图片（支持缩略图）
// GET /api/v1/images/:type/:id?w=300&q=80
// type: item-preview, post-cover, user-avatar, guild-banner
func (s *Server) getImage(c *gin.Context) {
	imageType := c.Param("type")
	id := c.Param("id")
	widthStr := c.DefaultQuery("w", "0")
	qualityStr := c.DefaultQuery("q", "85")
	version := c.Query("v")

	width, _ := strconv.Atoi(widthStr)
	quality, _ := strconv.Atoi(qualityStr)
	if quality <= 0 || quality > 100 {
		quality = 85
	}

	cacheControl := "public, max-age=86400"
	if width == 0 {
		cacheControl = "public, max-age=3600"
	}
	if version != "" {
		cacheControl = "public, max-age=31536000, immutable"
	}

	// 构造缓存路径
	cacheDir := filepath.Join(s.cfg.Storage.Path, "cache", "images")
	versionKey := ""
	if version != "" {
		versionKey = fmt.Sprintf("_v%x", md5.Sum([]byte(version)))
	}
	cacheKey := fmt.Sprintf("%s_%s_w%d_q%d%s.jpg", imageType, id, width, quality, versionKey)
	cachePath := filepath.Join(cacheDir, cacheKey)

	// 检查缓存
	if data, err := os.ReadFile(cachePath); err == nil {
		etag := fmt.Sprintf(`"%x"`, md5.Sum(data))
		if c.GetHeader("If-None-Match") == etag {
			c.Status(http.StatusNotModified)
			return
		}
		c.Header("Content-Type", "image/jpeg")
		c.Header("Cache-Control", cacheControl)
		c.Header("ETag", etag)
		c.Data(http.StatusOK, "image/jpeg", data)
		return
	}

	// 从数据库获取原图 Base64
	base64Data, err := s.getOriginalImageBase64(imageType, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image not found"})
		return
	}

	if base64Data == "" {
		c.JSON(http.StatusNotFound, gin.H{"error": "Image data is empty"})
		return
	}

	// 解析 data URI (data:image/jpeg;base64,xxx)
	var imgData []byte
	if strings.HasPrefix(base64Data, "data:") {
		parts := strings.SplitN(base64Data, ",", 2)
		if len(parts) != 2 {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid image data"})
			return
		}
		imgData, err = base64.StdEncoding.DecodeString(parts[1])
	} else {
		// 纯 Base64
		imgData, err = base64.StdEncoding.DecodeString(base64Data)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode image"})
		return
	}

	// 如果不需要缩放，直接返回
	if width == 0 {
		etag := fmt.Sprintf(`"%x"`, md5.Sum(imgData))
		if c.GetHeader("If-None-Match") == etag {
			c.Status(http.StatusNotModified)
			return
		}
		c.Header("Content-Type", http.DetectContentType(imgData))
		c.Header("Cache-Control", cacheControl)
		c.Header("ETag", etag)
		c.Data(http.StatusOK, http.DetectContentType(imgData), imgData)
		return
	}

	// 解码图片
	img, _, err := image.Decode(bytes.NewReader(imgData))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode image: " + err.Error()})
		return
	}

	// 计算缩放尺寸（保持比例）
	originalWidth := img.Bounds().Dx()
	originalHeight := img.Bounds().Dy()

	// 如果原图比请求的小，不放大
	if originalWidth <= width {
		etag := fmt.Sprintf(`"%x"`, md5.Sum(imgData))
		if c.GetHeader("If-None-Match") == etag {
			c.Status(http.StatusNotModified)
			return
		}
		c.Header("Content-Type", http.DetectContentType(imgData))
		c.Header("Cache-Control", cacheControl)
		c.Header("ETag", etag)
		c.Data(http.StatusOK, http.DetectContentType(imgData), imgData)
		return
	}

	// 等比例缩放
	newHeight := uint(float64(originalHeight) * float64(width) / float64(originalWidth))
	resized := resize.Resize(uint(width), newHeight, img, resize.Lanczos3)

	// 编码为 JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, resized, &jpeg.Options{Quality: quality}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to encode image"})
		return
	}

	result := buf.Bytes()

	// 写入缓存
	os.MkdirAll(cacheDir, 0755)
	os.WriteFile(cachePath, result, 0644)

	// 返回
	etag := fmt.Sprintf(`"%x"`, md5.Sum(result))
	if c.GetHeader("If-None-Match") == etag {
		c.Status(http.StatusNotModified)
		return
	}
	c.Header("Content-Type", "image/jpeg")
	c.Header("Cache-Control", cacheControl)
	c.Header("ETag", etag)
	c.Data(http.StatusOK, "image/jpeg", result)
}

// getOriginalImageBase64 从数据库获取原图 Base64
func (s *Server) getOriginalImageBase64(imageType string, id string) (string, error) {
	idNum, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return "", err
	}

	switch imageType {
	case "item-preview":
		var item model.Item
		if err := database.DB.Select("preview_image").First(&item, idNum).Error; err != nil {
			return "", err
		}
		return item.PreviewImage, nil

	case "post-cover":
		var post model.Post
		if err := database.DB.Select("cover_image").First(&post, idNum).Error; err != nil {
			return "", err
		}
		return post.CoverImage, nil

	case "user-avatar":
		var user model.User
		if err := database.DB.Select("avatar").First(&user, idNum).Error; err != nil {
			return "", err
		}
		return user.Avatar, nil

	case "guild-banner":
		var guild model.Guild
		if err := database.DB.Select("banner").First(&guild, idNum).Error; err != nil {
			return "", err
		}
		return guild.Banner, nil

	default:
		return "", fmt.Errorf("unknown image type: %s", imageType)
	}
}
