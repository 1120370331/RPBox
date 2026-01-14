package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
)

// CreateCharacterRequest 创建角色请求 (TRP3字段1:1对应)
type CreateCharacterRequest struct {
	RefID   string `json:"ref_id"`
	GameID  string `json:"game_id"`
	IsNPC   bool   `json:"is_npc"`

	// TRP3 characteristics
	Race       string `json:"race"`        // RA
	Class      string `json:"class"`       // CL
	FirstName  string `json:"first_name"`  // FN
	LastName   string `json:"last_name"`   // LN
	FullTitle  string `json:"full_title"`  // FT
	Title      string `json:"title"`       // TI
	Icon       string `json:"icon"`        // IC
	Color      string `json:"color"`       // CH
	EyeColor   string `json:"eye_color"`   // EC
	Age        string `json:"age"`         // AG
	Height     string `json:"height"`      // HE
	Residence  string `json:"residence"`   // RE
	Birthplace string `json:"birthplace"`  // BP
	MiscInfo   string `json:"misc_info"`   // MI (JSON)
	Psycho     string `json:"psycho"`      // PS (JSON)
	AboutText  string `json:"about_text"`  // about (JSON)

	RawTRP3Data string `json:"raw_trp3_data"` // 原始完整JSON
}

// UpdateCharacterRequest 更新角色请求
type UpdateCharacterRequest struct {
	// TRP3 字段 (可单独更新)
	Race       string `json:"race"`
	Class      string `json:"class"`
	FirstName  string `json:"first_name"`
	LastName   string `json:"last_name"`
	FullTitle  string `json:"full_title"`
	Title      string `json:"title"`
	Icon       string `json:"icon"`
	Color      string `json:"color"`
	EyeColor   string `json:"eye_color"`
	Age        string `json:"age"`
	Height     string `json:"height"`
	Residence  string `json:"residence"`
	Birthplace string `json:"birthplace"`
	MiscInfo   string `json:"misc_info"`
	Psycho     string `json:"psycho"`
	AboutText  string `json:"about_text"`

	// 自定义覆盖
	CustomAvatar string `json:"custom_avatar"`
	CustomName   string `json:"custom_name"`
	CustomColor  string `json:"custom_color"`
}

// listCharacters 获取用户的角色列表
func (s *Server) listCharacters(c *gin.Context) {
	userID := c.GetUint("userID")

	var characters []model.Character
	if err := database.DB.Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&characters).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"characters": characters})
}

// getCharacter 获取单个角色
func (s *Server) getCharacter(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var character model.Character
	if err := database.DB.First(&character, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	c.JSON(http.StatusOK, character)
}

// createOrUpdateCharacter 创建或更新角色（根据RefID或GameID匹配）
func (s *Server) createOrUpdateCharacter(c *gin.Context) {
	userID := c.GetUint("userID")

	var req CreateCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var character model.Character
	var err error

	// 优先用RefID匹配，其次用GameID
	if req.RefID != "" {
		err = database.DB.Where("user_id = ? AND ref_id = ?", userID, req.RefID).First(&character).Error
	} else if req.GameID != "" {
		err = database.DB.Where("user_id = ? AND game_id = ? AND is_npc = ?", userID, req.GameID, req.IsNPC).First(&character).Error
	}

	if err != nil {
		// 不存在，创建新角色
		character = model.Character{
			UserID:      userID,
			RefID:       req.RefID,
			GameID:      req.GameID,
			IsNPC:       req.IsNPC,
			Race:        req.Race,
			Class:       req.Class,
			FirstName:   req.FirstName,
			LastName:    req.LastName,
			FullTitle:   req.FullTitle,
			Title:       req.Title,
			Icon:        req.Icon,
			Color:       req.Color,
			EyeColor:    req.EyeColor,
			Age:         req.Age,
			Height:      req.Height,
			Residence:   req.Residence,
			Birthplace:  req.Birthplace,
			MiscInfo:    req.MiscInfo,
			Psycho:      req.Psycho,
			AboutText:   req.AboutText,
			RawTRP3Data: req.RawTRP3Data,
		}
		if err := database.DB.Create(&character).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "创建失败"})
			return
		}
		c.JSON(http.StatusCreated, character)
	} else {
		// 存在，更新TRP3数据
		updateCharacterFields(&character, &req)
		database.DB.Save(&character)
		c.JSON(http.StatusOK, character)
	}
}

// updateCharacterFields 更新角色字段（非空字段才更新）
func updateCharacterFields(character *model.Character, req *CreateCharacterRequest) {
	if req.Race != "" {
		character.Race = req.Race
	}
	if req.Class != "" {
		character.Class = req.Class
	}
	if req.FirstName != "" {
		character.FirstName = req.FirstName
	}
	if req.LastName != "" {
		character.LastName = req.LastName
	}
	if req.FullTitle != "" {
		character.FullTitle = req.FullTitle
	}
	if req.Title != "" {
		character.Title = req.Title
	}
	if req.Icon != "" {
		character.Icon = req.Icon
	}
	if req.Color != "" {
		character.Color = req.Color
	}
	if req.EyeColor != "" {
		character.EyeColor = req.EyeColor
	}
	if req.Age != "" {
		character.Age = req.Age
	}
	if req.Height != "" {
		character.Height = req.Height
	}
	if req.Residence != "" {
		character.Residence = req.Residence
	}
	if req.Birthplace != "" {
		character.Birthplace = req.Birthplace
	}
	if req.MiscInfo != "" {
		character.MiscInfo = req.MiscInfo
	}
	if req.Psycho != "" {
		character.Psycho = req.Psycho
	}
	if req.AboutText != "" {
		character.AboutText = req.AboutText
	}
	if req.RawTRP3Data != "" {
		character.RawTRP3Data = req.RawTRP3Data
	}
	if req.GameID != "" && character.GameID == "" {
		character.GameID = req.GameID
	}
}

// updateCharacter 更新角色自定义信息
func (s *Server) updateCharacter(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var character model.Character
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&character).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	var req UpdateCharacterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 更新TRP3字段
	if req.Race != "" {
		character.Race = req.Race
	}
	if req.Class != "" {
		character.Class = req.Class
	}
	if req.FirstName != "" {
		character.FirstName = req.FirstName
	}
	if req.LastName != "" {
		character.LastName = req.LastName
	}
	if req.FullTitle != "" {
		character.FullTitle = req.FullTitle
	}
	if req.Title != "" {
		character.Title = req.Title
	}
	if req.Icon != "" {
		character.Icon = req.Icon
	}
	if req.Color != "" {
		character.Color = req.Color
	}
	if req.EyeColor != "" {
		character.EyeColor = req.EyeColor
	}
	if req.Age != "" {
		character.Age = req.Age
	}
	if req.Height != "" {
		character.Height = req.Height
	}
	if req.Residence != "" {
		character.Residence = req.Residence
	}
	if req.Birthplace != "" {
		character.Birthplace = req.Birthplace
	}
	if req.MiscInfo != "" {
		character.MiscInfo = req.MiscInfo
	}
	if req.Psycho != "" {
		character.Psycho = req.Psycho
	}
	if req.AboutText != "" {
		character.AboutText = req.AboutText
	}

	// 更新自定义字段
	if req.CustomAvatar != "" {
		character.CustomAvatar = req.CustomAvatar
	}
	if req.CustomName != "" {
		character.CustomName = req.CustomName
	}
	if req.CustomColor != "" {
		character.CustomColor = req.CustomColor
	}

	database.DB.Save(&character)
	c.JSON(http.StatusOK, character)
}

// deleteCharacter 删除角色
func (s *Server) deleteCharacter(c *gin.Context) {
	userID := c.GetUint("userID")
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)

	var character model.Character
	if err := database.DB.Where("id = ? AND user_id = ?", id, userID).First(&character).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "角色不存在"})
		return
	}

	database.DB.Delete(&character)
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

// findOrCreateCharacter 查找或创建角色（内部使用，用于归档时）
func findOrCreateCharacter(userID uint, refID, gameID, rawTRP3Data string, isNPC bool) *model.Character {
	var character model.Character
	var err error

	// NPC消息：优先用 game_id + is_npc 查找，创建独立的NPC角色
	// 玩家消息：优先用 ref_id 查找
	if isNPC {
		// NPC：用 game_id + is_npc 查找
		if gameID != "" {
			err = database.DB.Where("user_id = ? AND game_id = ? AND is_npc = ?", userID, gameID, true).First(&character).Error
		} else {
			return nil
		}
	} else {
		// 玩家：优先用 ref_id 查找
		if refID != "" {
			err = database.DB.Where("user_id = ? AND ref_id = ?", userID, refID).First(&character).Error
		} else if gameID != "" {
			err = database.DB.Where("user_id = ? AND game_id = ? AND is_npc = ?", userID, gameID, false).First(&character).Error
		} else {
			return nil
		}
	}

	if err != nil {
		// 不存在，创建新角色
		character = model.Character{
			UserID:      userID,
			RefID:       refID,
			GameID:      gameID,
			IsNPC:       isNPC,
			RawTRP3Data: rawTRP3Data,
		}

		// 从原始JSON解析字段
		if rawTRP3Data != "" {
			parseTRP3DataToCharacter(&character, rawTRP3Data)
		}

		database.DB.Create(&character)
	} else {
		// 存在，更新原始数据
		if rawTRP3Data != "" && rawTRP3Data != character.RawTRP3Data {
			character.RawTRP3Data = rawTRP3Data
			parseTRP3DataToCharacter(&character, rawTRP3Data)
			database.DB.Save(&character)
		}
	}

	return &character
}

// parseTRP3DataToCharacter 从原始TRP3 JSON解析字段到Character
func parseTRP3DataToCharacter(character *model.Character, rawData string) {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(rawData), &data); err != nil {
		return
	}

	if v, ok := data["RA"].(string); ok {
		character.Race = v
	}
	if v, ok := data["CL"].(string); ok {
		character.Class = v
	}
	if v, ok := data["FN"].(string); ok {
		character.FirstName = v
	}
	if v, ok := data["LN"].(string); ok {
		character.LastName = v
	}
	if v, ok := data["FT"].(string); ok {
		character.FullTitle = v
	}
	if v, ok := data["TI"].(string); ok {
		character.Title = v
	}
	if v, ok := data["IC"].(string); ok {
		character.Icon = v
	}
	if v, ok := data["CH"].(string); ok {
		character.Color = v
	}
	if v, ok := data["EC"].(string); ok {
		character.EyeColor = v
	}
	if v, ok := data["AG"].(string); ok {
		character.Age = v
	}
	if v, ok := data["HE"].(string); ok {
		character.Height = v
	}
	if v, ok := data["RE"].(string); ok {
		character.Residence = v
	}
	if v, ok := data["BP"].(string); ok {
		character.Birthplace = v
	}

	// MI是characteristics中的其他信息数组
	if mi, ok := data["MI"]; ok {
		if miBytes, err := json.Marshal(mi); err == nil {
			character.MiscInfo = string(miBytes)
		}
	}

	// misc是杂项对象，包含PE（第一印象）和ST（RP风格）
	if misc, ok := data["misc"].(map[string]interface{}); ok {
		// 将整个misc对象存入MiscInfo
		if miscBytes, err := json.Marshal(misc); err == nil {
			character.MiscInfo = string(miscBytes)
		}
	}

	// PS是个性特征
	if ps, ok := data["PS"]; ok {
		if psBytes, err := json.Marshal(ps); err == nil {
			character.Psycho = string(psBytes)
		}
	}

	// about是关于信息
	if about, ok := data["about"]; ok {
		if aboutBytes, err := json.Marshal(about); err == nil {
			character.AboutText = string(aboutBytes)
		}
	}
}
