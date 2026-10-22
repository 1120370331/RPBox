package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
)

// createLocalCharacterCard imports only the explicitly selected local profile.
// It never changes an account backup or writes to the user's game files.
func createLocalCharacterCard(db *gorm.DB, userID uint, req createCharacterCardRequest) (model.CharacterCard, []model.CharacterCardImpression, error) {
	card := model.CharacterCard{UserID: userID, Status: model.CharacterCardStatusDraft, Visibility: model.CharacterCardVisibilityPrivate}
	if req.SourceBackupID != nil || len(req.LocalProfile) == 0 || len(req.LocalProfile) > 1<<20 {
		return card, nil, errors.New("请提供单个人物卡的数据，本地导入不能指定云备份")
	}
	if err := validateCharacterCardSourceAccountID(req.SourceAccountID); err != nil {
		return card, nil, err
	}
	if err := validateTRP3ProfileID(req.SourceProfileID); err != nil {
		return card, nil, err
	}
	profile, err := parseTRP3CharacterCardProfile(req.LocalProfile)
	if err != nil {
		return card, nil, fmt.Errorf("本地人物卡格式无效: %w", err)
	}
	card.SourceAccountID = req.SourceAccountID
	card.SourceProfileID = req.SourceProfileID
	applyTRP3CharacterCardFields(&card, profile.Fields)
	card.DisplayName = importedCharacterCardDisplayName(profile)
	if err := setCharacterCardTRP3Details(&card, profile.AdditionalInfo, profile.PersonalityTraits); err != nil {
		return card, nil, err
	}
	var envelope struct {
		Player struct {
			About map[string]json.RawMessage `json:"about"`
		} `json:"player"`
	}
	if err := json.Unmarshal(req.LocalProfile, &envelope); err != nil {
		return card, nil, errors.New("本地人物卡关于信息无效")
	}
	if len(envelope.Player.About) > 0 {
		if _, exists := envelope.Player.About["TE"]; !exists {
			envelope.Player.About["TE"] = json.RawMessage("1")
		}
		encoded, err := json.Marshal(envelope.Player.About)
		if err != nil {
			return card, nil, err
		}
		card.TRP3AboutJSON, err = normalizeCharacterCardTRP3About(encoded)
		if err != nil {
			return card, nil, err
		}
	}
	if err := validateCharacterCard(card); err != nil {
		return card, nil, err
	}
	var impressions []model.CharacterCardImpression
	err = db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&card).Error; err != nil {
			return err
		}
		impressions = characterCardImpressionsFromRequests(card.ID, profile.Impressions)
		return tx.Omit("CharacterCard").Create(&impressions).Error
	})
	if err != nil {
		return card, nil, &apiError{status: http.StatusInternalServerError, message: "导入本地人物卡失败"}
	}
	return card, impressions, nil
}
