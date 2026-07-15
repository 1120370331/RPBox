package api

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
)

type rpdbListEntryResponse struct {
	model.RPDBListEntry
	Work rpdbWorkCard `json:"work"`
}

type rpdbListResponse struct {
	model.RPDBList
	Entries []rpdbListEntryResponse `json:"entries"`
}

func (s *Server) listRPDBLists(c *gin.Context) {
	userID := c.GetUint("userID")
	if _, err := ensureDefaultRPDBList(database.DB, userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载默认收集清单失败"})
		return
	}
	var lists []model.RPDBList
	if err := database.DB.Where("user_id = ?", userID).Order("is_default DESC, updated_at DESC").Find(&lists).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载清单失败"})
		return
	}
	result := make([]rpdbListResponse, 0, len(lists))
	for _, list := range lists {
		detail, err := buildRPDBListResponse(list, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "加载清单内容失败"})
			return
		}
		result = append(result, detail)
	}
	c.JSON(http.StatusOK, gin.H{"lists": result})
}

func (s *Server) createRPDBList(c *gin.Context) {
	userID := c.GetUint("userID")
	var request struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		IsPublic    bool   `json:"is_public"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请求格式无效"})
		return
	}
	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" || len([]rune(request.Name)) > 128 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "清单名称长度必须为 1 到 128 个字符"})
		return
	}
	list := model.RPDBList{
		UserID:      userID,
		Name:        request.Name,
		Description: strings.TrimSpace(request.Description),
		IsPublic:    request.IsPublic,
	}
	if err := database.DB.Create(&list).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "创建清单失败"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"list": list})
}

func (s *Server) addRPDBWorkToDefaultList(c *gin.Context) {
	userID := c.GetUint("userID")
	workID, ok := parseRPDBWorkID(c)
	if !ok {
		return
	}
	if !rpdbPublishedWorkExists(workID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}
	var request struct {
		Status string `json:"status"`
		Note   string `json:"note"`
		ListID uint   `json:"list_id"`
	}
	_ = c.ShouldBindJSON(&request)
	if request.Status == "" {
		request.Status = model.RPDBListStatusWanted
	}
	if !validRPDBListStatus(request.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "清单状态无效"})
		return
	}

	var list model.RPDBList
	var entry model.RPDBListEntry
	created := false
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		var result *gorm.DB
		if request.ListID != 0 {
			result = tx.Where("id = ? AND user_id = ?", request.ListID, userID).First(&list)
			if result.Error != nil {
				return result.Error
			}
		} else {
			result = tx.Where("user_id = ? AND is_default = ?", userID, true).First(&list)
			if result.Error == gorm.ErrRecordNotFound {
				defaultList, err := ensureDefaultRPDBList(tx, userID)
				if err != nil {
					return err
				}
				list = defaultList
			} else if result.Error != nil {
				return result.Error
			}
		}

		entry = model.RPDBListEntry{
			ListID:   list.ID,
			WorkID:   workID,
			Status:   request.Status,
			Note:     strings.TrimSpace(request.Note),
			Quantity: 1,
		}
		result = tx.Where("list_id = ? AND work_id = ?", list.ID, workID).FirstOrCreate(&entry)
		if result.Error != nil {
			return result.Error
		}
		created = result.RowsAffected == 1
		if !created {
			return tx.Model(&entry).Updates(map[string]interface{}{
				"status": request.Status,
				"note":   strings.TrimSpace(request.Note),
			}).Error
		}
		if err := tx.Model(&list).UpdateColumn("item_count", gorm.Expr("item_count + 1")).Error; err != nil {
			return err
		}
		return tx.Model(&model.RPDBWork{}).Where("id = ?", workID).
			UpdateColumn("list_count", gorm.Expr("list_count + 1")).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加入清单失败"})
		return
	}

	status := http.StatusOK
	if created {
		status = http.StatusCreated
	}
	s.bumpRPDBListCache(c.Request.Context())
	c.JSON(status, gin.H{"list": list, "entry": entry})
}

func ensureDefaultRPDBList(tx *gorm.DB, userID uint) (model.RPDBList, error) {
	var list model.RPDBList
	result := tx.Where("user_id = ? AND is_default = ?", userID, true).First(&list)
	if result.Error == nil {
		if list.Name != "默认收集清单" {
			if err := tx.Model(&list).Update("name", "默认收集清单").Error; err != nil {
				return list, err
			}
			list.Name = "默认收集清单"
		}
		return list, nil
	}
	if result.Error != gorm.ErrRecordNotFound {
		return list, result.Error
	}
	list = model.RPDBList{
		UserID:      userID,
		Name:        "默认收集清单",
		Description: "自动创建的 RPDB 收集列表，用来追踪想收集和已收集的玩家作品。",
		IsDefault:   true,
		IsPublic:    false,
	}
	return list, tx.Create(&list).Error
}

func (s *Server) updateRPDBListEntry(c *gin.Context) {
	userID := c.GetUint("userID")
	listID, workID, ok := parseRPDBListWorkIDs(c)
	if !ok {
		return
	}
	var list model.RPDBList
	if err := database.DB.Where("id = ? AND user_id = ?", listID, userID).First(&list).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "清单不存在"})
		return
	}
	var request struct {
		Status      string `json:"status"`
		Note        string `json:"note"`
		Priority    int    `json:"priority"`
		Quantity    int    `json:"quantity"`
		CharacterID *uint  `json:"character_id"`
	}
	if err := c.ShouldBindJSON(&request); err != nil || !validRPDBListStatus(request.Status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "清单条目参数无效"})
		return
	}
	if request.Quantity <= 0 {
		request.Quantity = 1
	}
	result := database.DB.Model(&model.RPDBListEntry{}).
		Where("list_id = ? AND work_id = ?", listID, workID).
		Updates(map[string]interface{}{
			"status":       request.Status,
			"note":         strings.TrimSpace(request.Note),
			"priority":     request.Priority,
			"quantity":     request.Quantity,
			"character_id": request.CharacterID,
		})
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "更新清单条目失败"})
		return
	}
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "清单条目不存在"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (s *Server) removeRPDBListEntry(c *gin.Context) {
	userID := c.GetUint("userID")
	listID, workID, ok := parseRPDBListWorkIDs(c)
	if !ok {
		return
	}
	var list model.RPDBList
	if err := database.DB.Where("id = ? AND user_id = ?", listID, userID).First(&list).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "清单不存在"})
		return
	}
	if err := database.DB.Transaction(func(tx *gorm.DB) error {
		result := tx.Where("list_id = ? AND work_id = ?", listID, workID).Delete(&model.RPDBListEntry{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return nil
		}
		if err := tx.Model(&list).UpdateColumn("item_count", gorm.Expr("CASE WHEN item_count > 0 THEN item_count - 1 ELSE 0 END")).Error; err != nil {
			return err
		}
		return tx.Model(&model.RPDBWork{}).Where("id = ?", workID).
			UpdateColumn("list_count", gorm.Expr("CASE WHEN list_count > 0 THEN list_count - 1 ELSE 0 END")).Error
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "移除清单条目失败"})
		return
	}
	s.bumpRPDBListCache(c.Request.Context())
	c.Status(http.StatusNoContent)
}

func (s *Server) exportRPDBList(c *gin.Context) {
	userID := c.GetUint("userID")
	listIDValue, err := strconv.ParseUint(c.Param("listId"), 10, 64)
	if err != nil || listIDValue == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的清单 ID"})
		return
	}
	var list model.RPDBList
	if err := database.DB.Where("id = ? AND user_id = ?", uint(listIDValue), userID).First(&list).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "清单不存在"})
		return
	}
	detail, err := buildRPDBListResponse(list, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载清单失败"})
		return
	}

	switch c.DefaultQuery("format", "json") {
	case "json":
		c.JSON(http.StatusOK, gin.H{"format": "json", "list": detail})
	case "csv":
		var builder strings.Builder
		writer := csv.NewWriter(&builder)
		_ = writer.Write([]string{"id", "title", "type", "status", "quantity", "note"})
		for _, entry := range detail.Entries {
			_ = writer.Write([]string{
				strconv.FormatUint(uint64(entry.WorkID), 10),
				entry.Work.Title,
				entry.Work.Type,
				entry.Status,
				strconv.Itoa(entry.Quantity),
				entry.Note,
			})
		}
		writer.Flush()
		c.JSON(http.StatusOK, gin.H{"format": "csv", "content": builder.String()})
	case "tomtom":
		content, missing := buildRPDBTomTomExport(detail.Entries)
		c.JSON(http.StatusOK, gin.H{
			"format":              "tomtom",
			"content":             content,
			"missing_coordinates": missing,
		})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "不支持的导出格式"})
	}
}

func buildRPDBListResponse(list model.RPDBList, viewerID uint) (rpdbListResponse, error) {
	var entries []model.RPDBListEntry
	if err := database.DB.Where("list_id = ?", list.ID).Order("priority DESC, sort_order ASC, id ASC").Find(&entries).Error; err != nil {
		return rpdbListResponse{}, err
	}
	workIDs := make([]uint, 0, len(entries))
	for _, entry := range entries {
		workIDs = append(workIDs, entry.WorkID)
	}
	var works []model.RPDBWork
	if len(workIDs) > 0 {
		if err := database.DB.Where("id IN ?", workIDs).Find(&works).Error; err != nil {
			return rpdbListResponse{}, err
		}
	}
	cards, err := buildRPDBWorkCards(works, viewerID)
	if err != nil {
		return rpdbListResponse{}, err
	}
	cardMap := map[uint]rpdbWorkCard{}
	for _, card := range cards {
		cardMap[card.ID] = card
	}
	response := rpdbListResponse{RPDBList: list, Entries: make([]rpdbListEntryResponse, 0, len(entries))}
	for _, entry := range entries {
		response.Entries = append(response.Entries, rpdbListEntryResponse{
			RPDBListEntry: entry,
			Work:          cardMap[entry.WorkID],
		})
	}
	return response, nil
}

func buildRPDBTomTomExport(entries []rpdbListEntryResponse) (string, []gin.H) {
	lines := make([]string, 0)
	missing := make([]gin.H, 0)
	for _, entry := range entries {
		var steps []model.RPDBGuideStep
		database.DB.Where("work_id = ?", entry.WorkID).Order("sort_order ASC").Find(&steps)
		available := make([]model.RPDBGuideStep, 0, len(steps))
		for _, step := range steps {
			if !validRPDBTomTomCoordinates(step) {
				continue
			}
			available = append(available, step)
		}
		for index, step := range available {
			label := cleanRPDBTomTomLabel(step.Label)
			if label == "" {
				label = cleanRPDBTomTomLabel(step.Title)
			}
			workTitle := cleanRPDBTomTomLabel(entry.Work.Title)
			if workTitle != "" && !strings.EqualFold(workTitle, label) {
				if label == "" {
					label = workTitle
				} else {
					label = workTitle + " · " + label
				}
			}
			if len(available) > 1 {
				label = fmt.Sprintf("[%d/%d] %s", index+1, len(available), label)
			}
			parts := []string{"/way"}
			if target := rpdbTomTomTarget(step); target != "" {
				parts = append(parts, target)
			}
			parts = append(parts, fmt.Sprintf("%.2f", step.X), fmt.Sprintf("%.2f", step.Y))
			if label != "" {
				parts = append(parts, label)
			}
			lines = append(lines, strings.Join(parts, " "))
		}
		if len(available) == 0 {
			missing = append(missing, gin.H{"work_id": entry.WorkID, "title": entry.Work.Title})
		}
	}
	return strings.Join(lines, "\n"), missing
}

func validRPDBTomTomCoordinates(step model.RPDBGuideStep) bool {
	return step.X >= 0 && step.X <= 100 && step.Y >= 0 && step.Y <= 100 && (step.X != 0 || step.Y != 0)
}

func rpdbTomTomTarget(step model.RPDBGuideStep) string {
	mapID := strings.TrimPrefix(strings.TrimSpace(step.MapID), "#")
	if mapID != "" {
		if _, err := strconv.ParseUint(mapID, 10, 64); err == nil {
			return "#" + mapID
		}
		return mapID
	}
	return cleanRPDBTomTomLabel(step.Zone)
}

func cleanRPDBTomTomLabel(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func parseRPDBListWorkIDs(c *gin.Context) (uint, uint, bool) {
	listID, err := strconv.ParseUint(c.Param("listId"), 10, 64)
	if err != nil || listID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的清单 ID"})
		return 0, 0, false
	}
	workID, err := strconv.ParseUint(c.Param("workId"), 10, 64)
	if err != nil || workID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品 ID"})
		return 0, 0, false
	}
	return uint(listID), uint(workID), true
}

func validRPDBListStatus(status string) bool {
	switch status {
	case model.RPDBListStatusWanted, model.RPDBListStatusFarming, model.RPDBListStatusOwned, model.RPDBListStatusPaused:
		return true
	default:
		return false
	}
}
