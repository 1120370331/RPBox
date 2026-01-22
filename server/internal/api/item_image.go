package api

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

// uploadItemImages 上传画作图片（支持多图）
func (s *Server) uploadItemImages(c *gin.Context) {
	userID := c.GetUint("user_id")
	itemID := c.Param("id")

	var item model.Item
	if err := database.DB.First(&item, itemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}

	// 验证权限
	if item.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此作品"})
		return
	}

	// 验证类型
	if item.Type != "artwork" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "只有画作类型支持多图上传"})
		return
	}

	// 获取现有图片数量
	var count int64
	database.DB.Model(&model.ItemImage{}).Where("item_id = ?", itemID).Count(&count)
	if count >= 20 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "最多只能上传20张图片"})
		return
	}

	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择图片文件"})
		return
	}

	files := form.File["images"]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请选择图片文件"})
		return
	}

	type ImageResponse struct {
		ID        uint   `json:"id"`
		ImageURL  string `json:"image_url"`
		SortOrder int    `json:"sort_order"`
	}

	var uploaded []ImageResponse
	for _, file := range files {
		if file.Size > 10*1024*1024 { // 单张10MB
			continue
		}

		contentType := file.Header.Get("Content-Type")
		if contentType != "" && !strings.HasPrefix(contentType, "image/") {
			continue
		}

		imageURL, err := s.saveUploadedImage(c, file, path.Join("items", itemID))
		if err != nil {
			continue
		}

		imgRecord := model.ItemImage{
			ItemID:    item.ID,
			ImageData: imageURL,
			SortOrder: int(count) + len(uploaded),
		}
		database.DB.Create(&imgRecord)

		uploaded = append(uploaded, ImageResponse{
			ID:        imgRecord.ID,
			ImageURL:  buildPublicURL(c, "/api/v1/items/"+itemID+"/images/"+strconv.Itoa(int(imgRecord.ID))),
			SortOrder: imgRecord.SortOrder,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": gin.H{"images": uploaded},
	})
}

// listItemImages 获取画作图片列表
func (s *Server) listItemImages(c *gin.Context) {
	itemID := c.Param("id")

	var item model.Item
	if err := database.DB.First(&item, itemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}

	var images []model.ItemImage
	database.DB.Where("item_id = ?", itemID).Order("sort_order asc").Find(&images)

	type ImageResponse struct {
		ID        uint   `json:"id"`
		ImageURL  string `json:"image_url"`
		SortOrder int    `json:"sort_order"`
	}

	var result []ImageResponse
	for _, img := range images {
		result = append(result, ImageResponse{
			ID:        img.ID,
			ImageURL:  buildPublicURL(c, "/api/v1/items/"+itemID+"/images/"+strconv.Itoa(int(img.ID))),
			SortOrder: img.SortOrder,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": result,
	})
}

// getItemImage 获取画作图片（用于显示）
func (s *Server) getItemImage(c *gin.Context) {
	itemID := c.Param("id")
	imageID := c.Param("imageId")

	var item model.Item
	if err := database.DB.First(&item, itemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}

	var img model.ItemImage
	if err := database.DB.Where("id = ? AND item_id = ?", imageID, itemID).First(&img).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "图片不存在"})
		return
	}

	data, contentType, err := s.loadImageBytes(c, img.ImageData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "图片读取失败"})
		return
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	c.Header("Content-Type", contentType)
	c.Header("Cache-Control", "public, max-age=3600")
	c.Data(http.StatusOK, contentType, data)
}

// downloadItemImage 下载画作图片（带可选水印）
func (s *Server) downloadItemImage(c *gin.Context) {
	itemID := c.Param("id")
	imageID := c.Param("imageId")
	withWatermark := c.Query("watermark") != "false"

	var item model.Item
	if err := database.DB.First(&item, itemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}

	var img model.ItemImage
	if err := database.DB.Where("id = ? AND item_id = ?", imageID, itemID).First(&img).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "图片不存在"})
		return
	}

	data, contentType, err := s.loadImageBytes(c, img.ImageData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "图片读取失败"})
		return
	}
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}

	ext := imageExtension(contentType, "")
	if ext == "" {
		ext = ".jpg"
	}

	// 如果需要水印且作者开启了水印
	if withWatermark && item.EnableWatermark && (contentType == "image/jpeg" || contentType == "image/png") {
		// 获取作者用户名
		var author model.User
		database.DB.Select("username").First(&author, item.AuthorID)
		watermarkText := "@" + author.Username
		data = applyWatermark(data, watermarkText, contentType)
	}

	filename := item.Name + "_" + imageID + ext
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("Content-Type", contentType)
	c.Data(http.StatusOK, contentType, data)
}

// deleteItemImage 删除画作图片
func (s *Server) deleteItemImage(c *gin.Context) {
	userID := c.GetUint("user_id")
	itemID := c.Param("id")
	imageID := c.Param("imageId")

	var item model.Item
	if err := database.DB.First(&item, itemID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}

	// 验证权限
	if item.AuthorID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "无权操作此作品"})
		return
	}

	var img model.ItemImage
	if err := database.DB.Where("id = ? AND item_id = ?", imageID, itemID).First(&img).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "图片不存在"})
		return
	}

	keys := make(map[string]struct{})
	collectUploadKeysFromValue(c, img.ImageData, keys)
	s.deleteUploadKeys(keys)
	database.DB.Delete(&img)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "删除成功",
	})
}

// applyWatermark 应用水印到图片
func applyWatermark(imageData []byte, text string, contentType string) []byte {
	// 解码图片
	var img image.Image
	var err error

	reader := bytes.NewReader(imageData)
	if contentType == "image/png" {
		img, err = png.Decode(reader)
	} else {
		img, err = jpeg.Decode(reader)
	}

	if err != nil {
		return imageData // 解码失败返回原图
	}

	// 创建可绘制的图片
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	// 绘制水印文字（右下角）
	col := color.RGBA{255, 255, 255, 180} // 半透明白色
	x := bounds.Max.X - len(text)*7 - 20  // 右边距20px
	y := bounds.Max.Y - 20                // 底部距20px

	if x < 20 {
		x = 20
	}

	// 使用基础字体绘制文字
	point := fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y)}
	d := &font.Drawer{
		Dst:  rgba,
		Src:  image.NewUniform(col),
		Face: basicfont.Face7x13,
		Dot:  point,
	}
	d.DrawString(text)

	// 编码回字节
	var buf bytes.Buffer
	if contentType == "image/png" {
		png.Encode(&buf, rgba)
	} else {
		jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: 95})
	}

	return buf.Bytes()
}
