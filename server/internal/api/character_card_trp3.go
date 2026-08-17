package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type writeBackCharacterCardTRP3Request struct {
	BackupID     *uint  `json:"backup_id"`
	ProfileID    string `json:"profile_id"`
	SnapshotName string `json:"snapshot_name"`
}

var errInvalidCharacterCardTRP3RelationshipStatus = errors.New("relationship_status 必须是 0 到 5 的 TRP3 数值")

func (s *Server) exportCharacterCardTRP3Lua(c *gin.Context) {
	cardID, ok := parseCharacterCardID(c)
	if !ok {
		return
	}
	userID := c.GetUint("userID")
	var card model.CharacterCard
	if err := database.DB.Where("id = ? AND user_id = ?", cardID, userID).First(&card).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "人物卡不存在"})
		return
	}
	profileID := strings.TrimSpace(card.SourceProfileID)
	if profileID == "" {
		profileID = fmt.Sprintf("rpbox-%d", card.ID)
	}
	var existing interface{}
	if card.SourceBackupID != nil && *card.SourceBackupID != 0 && strings.TrimSpace(card.SourceProfileID) != "" {
		var backup model.AccountBackup
		if err := database.DB.Where("id = ? AND user_id = ?", *card.SourceBackupID, userID).First(&backup).Error; err == nil {
			if profiles, decodeErr := decodeTRP3ProfilesObject(backup.ProfilesData); decodeErr == nil {
				existing = profiles[profileID]
			}
		}
	}
	impressionsByCard, err := loadCharacterCardImpressions(database.DB, []uint{card.ID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "读取人物卡第一印象失败"})
		return
	}
	profile, err := buildTRP3ProfileFromCharacterCard(card, impressionsByCard[card.ID], existing)
	if err != nil {
		if errors.Is(err, errInvalidCharacterCardTRP3RelationshipStatus) {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成人物卡 Lua 失败"})
		return
	}
	profiles := map[string]interface{}{profileID: profile}
	c.JSON(http.StatusOK, gin.H{
		"profile_id": profileID,
		"profile":    profile,
		"lua":        "TRP3_Profiles = " + encodeLuaValue(profiles, 0) + "\n",
	})
}

func (s *Server) writeBackCharacterCardTRP3(c *gin.Context) {
	cardID, ok := parseCharacterCardID(c)
	if !ok {
		return
	}
	userID := c.GetUint("userID")
	var req writeBackCharacterCardTRP3Request
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请求参数无效"})
			return
		}
	}
	if _, err := validateAccountBackupVersionName(req.SnapshotName); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var card model.CharacterCard
	var backup model.AccountBackup
	var snapshot model.AccountBackupVersion
	var profile map[string]interface{}
	var profileID string
	var lua string
	err := database.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", cardID, userID).First(&card).Error; err != nil {
			return err
		}
		backupID := uint(0)
		if req.BackupID != nil {
			backupID = *req.BackupID
		} else if card.SourceBackupID != nil {
			backupID = *card.SourceBackupID
		}
		if backupID == 0 {
			return errors.New("请指定要写回的账号备份")
		}
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND user_id = ?", backupID, userID).First(&backup).Error; err != nil {
			return err
		}
		if err := validateCharacterCardSourceAccountID(backup.AccountID); err != nil {
			return errors.New("目标账号 ID 不安全")
		}
		profileID = strings.TrimSpace(req.ProfileID)
		if profileID == "" {
			profileID = strings.TrimSpace(card.SourceProfileID)
		}
		if err := validateTRP3ProfileID(profileID); err != nil {
			return err
		}
		profiles, err := decodeTRP3ProfilesObject(backup.ProfilesData)
		if err != nil {
			return errors.New("目标账号备份的人物卡数据损坏")
		}
		impressionsByCard, loadErr := loadCharacterCardImpressions(tx, []uint{card.ID})
		if loadErr != nil {
			return loadErr
		}
		profile, err = buildTRP3ProfileFromCharacterCard(card, impressionsByCard[card.ID], profiles[profileID])
		if err != nil {
			return err
		}
		profiles[profileID] = profile
		profilesJSON, err := json.Marshal(profiles)
		if err != nil {
			return err
		}
		luaTable := encodeLuaValue(profiles, 0)
		lua = "TRP3_Profiles = " + luaTable + "\n"
		updatedRaw := ""
		if strings.TrimSpace(backup.RawTrp3Lua) != "" {
			updatedRaw, err = replaceTRP3ProfilesLua(backup.RawTrp3Lua, luaTable)
			if err != nil {
				return fmt.Errorf("原始 TRP3 Lua 无法安全更新: %w", err)
			}
		}

		// Mandatory pre-write snapshot. There is deliberately no request flag
		// that can bypass this step.
		snapshot, err = createAccountBackupSnapshot(tx, backup, req.SnapshotName, "before_character_card_writeback")
		if err != nil {
			return err
		}
		backup.ProfilesData = string(profilesJSON)
		backup.ProfilesCount = len(profiles)
		backup.RawTrp3Lua = updatedRaw
		backup.Version++
		backup.Checksum = accountBackupContentHash(backup)
		if err := tx.Save(&backup).Error; err != nil {
			return err
		}
		return tx.Model(&model.CharacterCard{}).Where("id = ? AND user_id = ?", card.ID, userID).Updates(map[string]interface{}{
			"source_backup_id": backup.ID, "source_account_id": backup.AccountID, "source_profile_id": profileID,
		}).Error
	})
	if err != nil {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			c.JSON(http.StatusNotFound, gin.H{"error": "人物卡或目标账号备份不存在"})
		case errors.Is(err, errInvalidCharacterCardTRP3RelationshipStatus),
			strings.Contains(err.Error(), "指定") || strings.Contains(err.Error(), "不安全") || strings.Contains(err.Error(), "损坏") || strings.Contains(err.Error(), "profile_id"):
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "写回 TRP3 失败"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"backup": backup, "snapshot": snapshot,
		"profile_id": profileID, "profile": profile, "lua": lua,
	})
}

func validateTRP3ProfileID(raw string) error {
	if raw == "" || strings.TrimSpace(raw) != raw || raw == "." || raw == ".." || strings.ContainsAny(raw, `/\`) {
		return errors.New("profile_id 不是安全的档案标识")
	}
	if err := validatePlainCharacterCardField("profile_id", raw, 128); err != nil {
		return err
	}
	// TRP3 replaces the final generated ID character with "*" for its default
	// profile. Keep the marker scoped to that official trailing position so an
	// imported default profile can be synced without broadening the ID grammar.
	idBody := strings.TrimSuffix(raw, "*")
	if idBody == "" {
		return errors.New("profile_id 不是安全的档案标识")
	}
	for _, char := range idBody {
		if unicode.IsLetter(char) || unicode.IsDigit(char) || char == '-' || char == '_' || char == '.' || char == '#' {
			continue
		}
		return errors.New("profile_id 不是安全的档案标识")
	}
	return nil
}

func decodeTRP3ProfilesObject(raw string) (map[string]interface{}, error) {
	rawProfiles, err := decodeCharacterCardProfileMap(raw)
	if err != nil {
		return nil, err
	}

	// decodeCharacterCardProfileMap is the canonical container decoder used by
	// imports and write-back. It unwraps the legacy {"profiles": {...}} shape.
	// Normalize each RawMessage before it reaches the merge builder so unknown
	// profile/player sections remain ordinary maps instead of being discarded by
	// a failed type assertion.
	profiles := make(map[string]interface{}, len(rawProfiles))
	for profileID, rawProfile := range rawProfiles {
		profile, err := normalizeTRP3ProfileObject(rawProfile)
		if err != nil {
			return nil, fmt.Errorf("profile %q is invalid: %w", profileID, err)
		}
		profiles[profileID] = profile
	}
	return profiles, nil
}

func buildTRP3ProfileFromCharacterCard(card model.CharacterCard, impressions []model.CharacterCardImpression, existing interface{}) (map[string]interface{}, error) {
	profile, err := normalizeTRP3ProfileObject(existing)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(card.DisplayName) != "" {
		profile["profileName"] = strings.TrimSpace(card.DisplayName)
	} else if _, exists := profile["profileName"]; !exists {
		profile["profileName"] = joinedCharacterCardName(card.FirstName, card.LastName)
	}
	player, _ := profile["player"].(map[string]interface{})
	if player == nil {
		player = make(map[string]interface{})
	}
	characteristics, _ := player["characteristics"].(map[string]interface{})
	if characteristics == nil {
		characteristics = make(map[string]interface{})
	}
	fields := map[string]string{
		"FN": card.FirstName, "LN": card.LastName, "TI": card.Title, "FT": card.FullTitle,
		"RA": card.Race, "CL": card.Class, "EC": card.EyeColor, "EH": card.EyeColorHex,
		"AG": card.Age, "HE": card.Height, "WE": card.Weight, "BP": card.Birthplace,
		"RE": card.Residence, "IC": card.Icon,
		"CH": canonicalCharacterCardTRP3Color(card.ClassColor, card.NameColor),
	}
	for key, value := range fields {
		value = strings.TrimSpace(value)
		if value == "" {
			delete(characteristics, key)
		} else {
			characteristics[key] = value
		}
	}
	relationshipStatus, hasRelationshipStatus, err := characterCardTRP3RelationshipStatus(card.RelationshipStatus)
	if err != nil {
		return nil, err
	}
	if hasRelationshipStatus {
		characteristics["RS"] = relationshipStatus
	} else {
		delete(characteristics, "RS")
	}
	additionalInfo := characterCardTRP3AdditionalInfoFromCard(card)
	characteristics["MI"] = characterCardTRP3AdditionalInfoForWrite(additionalInfo)
	personalityTraits := characterCardTRP3PersonalityTraitsFromCard(card)
	characteristics["PS"] = characterCardTRP3PersonalityTraitsForWrite(personalityTraits)
	characteristics["v"] = nextCharacterCardTRP3Version(characteristics["v"])
	player["characteristics"] = characteristics

	misc, _ := player["misc"].(map[string]interface{})
	if misc == nil {
		misc = make(map[string]interface{})
	}
	pe := make(map[string]interface{}, characterCardImpressionSlotCount)
	for _, row := range fixedCharacterCardImpressions(card.ID, impressions) {
		pe[strconv.Itoa(int(row.Slot))] = map[string]interface{}{
			"AC": row.Active,
			"IC": strings.TrimSpace(row.TRP3Icon),
			"TI": row.Title,
			"TX": row.Text,
		}
	}
	misc["PE"] = pe
	misc["v"] = nextCharacterCardTRP3Version(misc["v"])
	player["misc"] = misc
	profile["player"] = player
	return profile, nil
}

func characterCardTRP3AdditionalInfoForWrite(items []characterCardTRP3AdditionalInfo) []interface{} {
	result := make([]interface{}, 0, len(items))
	for _, item := range items {
		result = append(result, map[string]interface{}{
			"ID": item.ID, "NA": item.Name, "VA": item.Value, "IC": item.Icon,
		})
	}
	return result
}

func characterCardTRP3PersonalityTraitsForWrite(items []characterCardTRP3PersonalityTrait) []interface{} {
	result := make([]interface{}, 0, len(items))
	for _, item := range items {
		trait := map[string]interface{}{"V2": item.Value}
		if item.PresetID != nil {
			trait["ID"] = *item.PresetID
		} else {
			trait["LT"] = item.LeftText
			trait["RT"] = item.RightText
			trait["LI"] = item.LeftIcon
			trait["RI"] = item.RightIcon
			if item.LeftColor != nil {
				trait["LC"] = map[string]interface{}{"r": item.LeftColor.R, "g": item.LeftColor.G, "b": item.LeftColor.B}
			}
			if item.RightColor != nil {
				trait["RC"] = map[string]interface{}{"r": item.RightColor.R, "g": item.RightColor.G, "b": item.RightColor.B}
			}
		}
		result = append(result, trait)
	}
	return result
}

func nextCharacterCardTRP3Version(raw interface{}) json.Number {
	current := int64(0)
	switch value := raw.(type) {
	case json.Number:
		current, _ = value.Int64()
	case float64:
		current = int64(value)
	case int:
		current = int64(value)
	case int64:
		current = value
	}
	next := current + 1
	if next <= 0 || next >= 100 {
		next = 1
	}
	return json.Number(strconv.FormatInt(next, 10))
}

func normalizeTRP3ProfileObject(value interface{}) (map[string]interface{}, error) {
	if value == nil {
		return make(map[string]interface{}), nil
	}
	if profile, ok := value.(map[string]interface{}); ok && profile != nil {
		return profile, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(encoded))
	decoder.UseNumber()
	var profile map[string]interface{}
	if err := decoder.Decode(&profile); err != nil || profile == nil {
		if err == nil {
			err = errors.New("profile is not an object")
		}
		return nil, err
	}
	return profile, nil
}

func characterCardTRP3RelationshipStatus(raw string) (json.Number, bool, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return "", false, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 8)
	if err != nil || parsed > 5 {
		return "", false, errInvalidCharacterCardTRP3RelationshipStatus
	}
	return json.Number(strconv.FormatUint(parsed, 10)), true, nil
}

func encodeLuaValue(value interface{}, indent int) string {
	switch typed := value.(type) {
	case nil:
		return "nil"
	case bool:
		if typed {
			return "true"
		}
		return "false"
	case json.Number:
		return typed.String()
	case float64:
		return strconv.FormatFloat(typed, 'g', -1, 64)
	case string:
		return `"` + escapeLuaString(typed) + `"`
	case []interface{}:
		if len(typed) == 0 {
			return "{}"
		}
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			parts = append(parts, strings.Repeat(" ", indent+2)+encodeLuaValue(item, indent+2))
		}
		return "{\n" + strings.Join(parts, ",\n") + "\n" + strings.Repeat(" ", indent) + "}"
	case map[string]interface{}:
		if len(typed) == 0 {
			return "{}"
		}
		keys := make([]string, 0, len(typed))
		for key := range typed {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		parts := make([]string, 0, len(keys))
		for _, key := range keys {
			luaKey := key
			if !isLuaIdentifier(key) || isLuaKeyword(key) {
				luaKey = `[` + `"` + escapeLuaString(key) + `"` + `]`
			}
			parts = append(parts, strings.Repeat(" ", indent+2)+luaKey+" = "+encodeLuaValue(typed[key], indent+2))
		}
		return "{\n" + strings.Join(parts, ",\n") + "\n" + strings.Repeat(" ", indent) + "}"
	default:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return "nil"
		}
		decoder := json.NewDecoder(bytes.NewReader(encoded))
		decoder.UseNumber()
		var normalized interface{}
		if err := decoder.Decode(&normalized); err != nil {
			return "nil"
		}
		return encodeLuaValue(normalized, indent)
	}
}

func escapeLuaString(value string) string {
	var builder strings.Builder
	for _, char := range value {
		switch char {
		case '\\':
			builder.WriteString(`\\`)
		case '"':
			builder.WriteString(`\"`)
		case '\n':
			builder.WriteString(`\n`)
		case '\r':
			builder.WriteString(`\r`)
		case '\t':
			builder.WriteString(`\t`)
		case '\b':
			builder.WriteString(`\b`)
		case '\f':
			builder.WriteString(`\f`)
		default:
			if char <= 0x1f || char == 0x7f {
				builder.WriteString(fmt.Sprintf(`\%03d`, char))
			} else {
				builder.WriteRune(char)
			}
		}
	}
	return builder.String()
}

func isLuaIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for index, char := range value {
		if index == 0 {
			if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || char == '_') {
				return false
			}
		} else if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_') {
			return false
		}
	}
	return true
}

func isLuaKeyword(value string) bool {
	switch value {
	case "and", "break", "do", "else", "elseif", "end", "false", "for", "function", "goto", "if", "in", "local", "nil", "not", "or", "repeat", "return", "then", "true", "until", "while":
		return true
	default:
		return false
	}
}

func replaceTRP3ProfilesLua(raw, table string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "TRP3_Profiles = " + table + "\n", nil
	}
	braceStart, found, err := findLuaTableAssignment(raw, "TRP3_Profiles")
	if err != nil {
		return "", err
	}
	if !found {
		separator := "\n"
		if strings.HasSuffix(raw, "\n") {
			separator = ""
		}
		return raw + separator + "TRP3_Profiles = " + table + "\n", nil
	}
	braceEnd, err := findLuaTableEnd(raw, braceStart)
	if err != nil {
		return "", err
	}
	return raw[:braceStart] + table + raw[braceEnd:], nil
}

func findLuaTableAssignment(raw, identifier string) (int, bool, error) {
	for index := 0; index < len(raw); {
		if raw[index] == '"' || raw[index] == '\'' {
			next, err := skipLuaQuoted(raw, index)
			if err != nil {
				return 0, false, err
			}
			index = next
			continue
		}
		if strings.HasPrefix(raw[index:], "--") {
			index = skipLuaComment(raw, index)
			continue
		}
		if strings.HasPrefix(raw[index:], identifier) && luaIdentifierBoundary(raw, index-1) && luaIdentifierBoundary(raw, index+len(identifier)) {
			cursor := skipLuaWhitespaceAndComments(raw, index+len(identifier))
			if cursor >= len(raw) || raw[cursor] != '=' {
				index += len(identifier)
				continue
			}
			cursor = skipLuaWhitespaceAndComments(raw, cursor+1)
			if cursor >= len(raw) || raw[cursor] != '{' {
				return 0, false, errors.New("TRP3_Profiles assignment is not a table")
			}
			return cursor, true, nil
		}
		index++
	}
	return 0, false, nil
}

func luaIdentifierBoundary(raw string, index int) bool {
	if index < 0 || index >= len(raw) {
		return true
	}
	char := raw[index]
	return !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == '_')
}

func skipLuaWhitespaceAndComments(raw string, index int) int {
	for index < len(raw) {
		if raw[index] == ' ' || raw[index] == '\t' || raw[index] == '\r' || raw[index] == '\n' {
			index++
			continue
		}
		if strings.HasPrefix(raw[index:], "--") {
			index = skipLuaComment(raw, index)
			continue
		}
		break
	}
	return index
}

func skipLuaComment(raw string, index int) int {
	if openEnd, equals, ok := luaLongBracketOpen(raw, index+2); ok {
		closing := "]" + strings.Repeat("=", equals) + "]"
		if offset := strings.Index(raw[openEnd:], closing); offset >= 0 {
			return openEnd + offset + len(closing)
		}
		return len(raw)
	}
	if newline := strings.IndexByte(raw[index:], '\n'); newline >= 0 {
		return index + newline + 1
	}
	return len(raw)
}

func findLuaTableEnd(raw string, braceStart int) (int, error) {
	depth := 0
	for index := braceStart; index < len(raw); {
		switch raw[index] {
		case '"', '\'':
			next, err := skipLuaQuoted(raw, index)
			if err != nil {
				return 0, err
			}
			index = next
			continue
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return index + 1, nil
			}
		case '-':
			if strings.HasPrefix(raw[index:], "--") {
				index = skipLuaComment(raw, index)
				continue
			}
		case '[':
			if openEnd, equals, ok := luaLongBracketOpen(raw, index); ok {
				closing := "]" + strings.Repeat("=", equals) + "]"
				offset := strings.Index(raw[openEnd:], closing)
				if offset < 0 {
					return 0, errors.New("unterminated Lua long string")
				}
				index = openEnd + offset + len(closing)
				continue
			}
		}
		index++
	}
	return 0, errors.New("unterminated TRP3_Profiles table")
}

func skipLuaQuoted(raw string, start int) (int, error) {
	quote := raw[start]
	for index := start + 1; index < len(raw); index++ {
		if raw[index] == '\\' {
			index++
			continue
		}
		if raw[index] == quote {
			return index + 1, nil
		}
	}
	return 0, errors.New("unterminated Lua string")
}

func luaLongBracketOpen(raw string, start int) (int, int, bool) {
	if start >= len(raw) || raw[start] != '[' {
		return 0, 0, false
	}
	index := start + 1
	for index < len(raw) && raw[index] == '=' {
		index++
	}
	if index >= len(raw) || raw[index] != '[' {
		return 0, 0, false
	}
	return index + 1, index - start - 1, true
}
