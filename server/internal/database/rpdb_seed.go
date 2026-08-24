package database

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/rpbox/server/internal/model"
	authpkg "github.com/rpbox/server/pkg/auth"
	"gorm.io/gorm"
)

const (
	rpdbDemoUsername       = "rpdb_demo"
	rpdbDemoEmail          = "rpdb-demo@local.invalid"
	rpdbDemoDisabledReason = "System-managed RPDB demo identity; interactive sign-in is disabled."
)

type rpdbDemoAccountSpec struct {
	Username string
	Email    string
}

type rpdbDemoWorkSpec struct {
	Slug          string
	Type          string
	Title         string
	Summary       string
	Content       string
	RPUseCases    string
	Effect        string
	Expansion     string
	Faction       string
	ArmorType     string
	Zone          string
	MapID         string
	X             float64
	Y             float64
	ImageURL      string
	ReferenceIDs  []string
	ReferenceType string
	Tags          []string
	Server        string
	Region        string
	HomeStyle     string
	ShareCode     string
	VisitNotes    string
	CopyStatus    string
	VisitStatus   string
	SpaceType     string
}

type rpdbDemoTagSpec struct {
	Name  string
	Color string
}

// SeedRPDBDemo creates or refreshes the stable RPDB demonstration dataset.
// It only upserts records identified by stable demo keys and never clears tables.
func SeedRPDBDemo(db *gorm.DB) error {
	if db == nil {
		return errors.New("seed RPDB demo: database is nil")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		accountSpecs := rpdbDemoAccountSpecs()
		author, err := ensureRPDBDemoAccount(tx, accountSpecs[0])
		if err != nil {
			return err
		}
		curator, err := ensureRPDBDemoAccount(tx, accountSpecs[1])
		if err != nil {
			return err
		}
		explorer, err := ensureRPDBDemoAccount(tx, accountSpecs[2])
		if err != nil {
			return err
		}

		tags, err := ensureRPDBDemoTags(tx, author.ID)
		if err != nil {
			return err
		}

		now := time.Now().UTC().Truncate(time.Second)
		works := make([]model.RPDBWork, 0, 12)
		for _, spec := range rpdbDemoWorkSpecs() {
			work, err := ensureRPDBDemoWork(tx, author, curator, spec, now)
			if err != nil {
				return err
			}
			if err := ensureRPDBDemoWorkChildren(tx, work, author, curator, spec, tags, now); err != nil {
				return err
			}
			if err := ensureRPDBDemoInteractions(tx, work, curator, explorer); err != nil {
				return err
			}
			works = append(works, work)
		}

		if err := ensureRPDBDemoList(tx, author, works); err != nil {
			return err
		}
		if err := ensureRPDBDemoSet(tx, curator, works); err != nil {
			return err
		}
		if err := refreshRPDBDemoTagCounts(tx, tags); err != nil {
			return err
		}
		for _, work := range works {
			if err := refreshRPDBDemoWorkCounts(tx, work.ID, now); err != nil {
				return err
			}
		}

		return nil
	})
}

func ensureRPDBDemoAccount(tx *gorm.DB, spec rpdbDemoAccountSpec) (model.User, error) {
	var user model.User
	err := tx.Where("email = ?", spec.Email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = tx.Where("username = ?", spec.Username).First(&user).Error
	}
	if err == nil {
		if err := hardenRPDBDemoAccount(tx, &user, spec); err != nil {
			return model.User{}, err
		}
		if err := tx.First(&user, user.ID).Error; err != nil {
			return model.User{}, fmt.Errorf("reload demo user %s: %w", spec.Username, err)
		}
		return user, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return model.User{}, fmt.Errorf("find demo user %s: %w", spec.Username, err)
	}

	hash, err := newRPDBDemoPasswordHash()
	if err != nil {
		return model.User{}, fmt.Errorf("create disabled credential for demo user %s: %w", spec.Username, err)
	}
	user = model.User{
		Username:           spec.Username,
		Email:              spec.Email,
		EmailVerified:      false,
		PassHash:           hash,
		Role:               "user",
		Bio:                "RPBox RP 数据库演示账号",
		Location:           "艾泽拉斯",
		AgreementVersion:   "demo-seed",
		AvatarReviewStatus: model.RPDBReviewNone,
		IsMuted:            true,
		MuteReason:         rpdbDemoDisabledReason,
		IsBanned:           true,
		BanReason:          rpdbDemoDisabledReason,
	}
	if err := tx.Create(&user).Error; err != nil {
		return model.User{}, fmt.Errorf("create demo user %s: %w", spec.Username, err)
	}
	return user, nil
}

func rpdbDemoAccountSpecs() [3]rpdbDemoAccountSpec {
	return [3]rpdbDemoAccountSpec{
		{Username: rpdbDemoUsername, Email: rpdbDemoEmail},
		{Username: "rpdb_demo_curator", Email: "rpdb-demo-curator@local.invalid"},
		{Username: "rpdb_demo_explorer", Email: "rpdb-demo-explorer@local.invalid"},
	}
}

// hardenRPDBDemoAccounts disables any stable demo identities that already exist.
// It is intentionally called during normal database initialization, not only by
// the manual demo-data seeder.
func hardenRPDBDemoAccounts(db *gorm.DB) error {
	if db == nil {
		return errors.New("harden RPDB demo accounts: database is nil")
	}

	return db.Transaction(func(tx *gorm.DB) error {
		for _, spec := range rpdbDemoAccountSpecs() {
			var users []model.User
			if err := tx.Where("username = ? OR email = ?", spec.Username, spec.Email).Find(&users).Error; err != nil {
				return fmt.Errorf("find demo user %s for hardening: %w", spec.Username, err)
			}
			for index := range users {
				if err := hardenRPDBDemoAccount(tx, &users[index], spec); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func hardenRPDBDemoAccount(tx *gorm.DB, user *model.User, spec rpdbDemoAccountSpec) error {
	if isRPDBDemoAccountHardened(user, spec) {
		return nil
	}

	hash, err := newRPDBDemoPasswordHash()
	if err != nil {
		return fmt.Errorf("rotate disabled credential for demo user %s: %w", spec.Username, err)
	}
	updates := map[string]interface{}{
		"email_verified": false,
		"pass_hash":      hash,
		"role":           "user",
		"is_muted":       true,
		"muted_until":    nil,
		"mute_reason":    rpdbDemoDisabledReason,
		"is_banned":      true,
		"banned_until":   nil,
		"ban_reason":     rpdbDemoDisabledReason,
		"banned_by":      nil,
		"banned_at":      nil,
	}
	if err := tx.Model(user).Updates(updates).Error; err != nil {
		return fmt.Errorf("harden demo user %s: %w", spec.Username, err)
	}
	return nil
}

func isRPDBDemoAccountHardened(user *model.User, spec rpdbDemoAccountSpec) bool {
	return user != nil &&
		!user.EmailVerified &&
		user.PassHash != "" &&
		user.Role == "user" &&
		user.IsMuted &&
		user.MutedUntil == nil &&
		user.MuteReason == rpdbDemoDisabledReason &&
		user.IsBanned &&
		user.BannedUntil == nil &&
		user.BanReason == rpdbDemoDisabledReason &&
		user.BannedBy == nil &&
		user.BannedAt == nil
}

func newRPDBDemoPasswordHash() (string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", fmt.Errorf("generate random credential: %w", err)
	}
	return authpkg.HashPassword(base64.RawURLEncoding.EncodeToString(secret))
}

func ensureRPDBDemoTags(tx *gorm.DB, creatorID uint) (map[string]model.Tag, error) {
	specs := []rpdbDemoTagSpec{
		{Name: "联盟风格", Color: "2F66C8"},
		{Name: "部落风格", Color: "B83030"},
		{Name: "库尔提拉斯风格", Color: "356A8A"},
		{Name: "洛丹伦风格", Color: "6E6A85"},
		{Name: "暴风城风格", Color: "356AB8"},
		{Name: "银月城风格", Color: "C08A2C"},
		{Name: "暗夜精灵风格", Color: "6D5DB8"},
		{Name: "矮人风格", Color: "8A6448"},
		{Name: "侏儒工程风格", Color: "C46B3A"},
		{Name: "地精工程风格", Color: "5D8F3A"},
		{Name: "被遗忘者风格", Color: "5E6E5A"},
		{Name: "熊猫人风格", Color: "4F8C62"},
		{Name: "德鲁斯瓦风格", Color: "5A5B68"},
		{Name: "达拉然风格", Color: "8A6DCC"},
		{Name: "海盗风格", Color: "9A5B38"},
		{Name: "泰坦遗迹风格", Color: "C2A15A"},
		{Name: "龙族风格", Color: "B35C42"},
		{Name: "荒野游侠风格", Color: "557A45"},
		{Name: "圣光教会风格", Color: "C7A95B"},
		{Name: "暗影诅咒风格", Color: "57456F"},
		{Name: "贵族沙龙风格", Color: "8B6F96"},
		{Name: "海港酒馆风格", Color: "4B7991"},
		{Name: "炼金工坊风格", Color: "6F8F46"},
		{Name: "军旅哨站风格", Color: "727A54"},
	}
	tags := make(map[string]model.Tag, len(specs))
	for _, spec := range specs {
		var tag model.Tag
		err := tx.Where("name = ? AND category = ? AND type = ?", spec.Name, "rpdb", "preset").First(&tag).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			tag = model.Tag{
				Name:      spec.Name,
				Color:     spec.Color,
				Category:  "rpdb",
				Type:      "preset",
				CreatorID: creatorID,
				IsPublic:  true,
			}
			if err := tx.Create(&tag).Error; err != nil {
				return nil, fmt.Errorf("create RPDB demo tag %s: %w", spec.Name, err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("find RPDB demo tag %s: %w", spec.Name, err)
		}
		tags[spec.Name] = tag
	}
	return tags, nil
}

func ensureRPDBDemoWork(tx *gorm.DB, author, reviewer model.User, spec rpdbDemoWorkSpec, now time.Time) (model.RPDBWork, error) {
	var work model.RPDBWork
	findErr := tx.Where("slug = ?", spec.Slug).First(&work).Error
	if findErr != nil && !errors.Is(findErr, gorm.ErrRecordNotFound) {
		return model.RPDBWork{}, fmt.Errorf("find RPDB demo work %s: %w", spec.Slug, findErr)
	}
	if findErr == nil && work.AuthorID != author.ID {
		return model.RPDBWork{}, fmt.Errorf("RPDB demo slug %s belongs to user %d", spec.Slug, work.AuthorID)
	}
	extra, extraErr := rpdbDemoWorkExtra(spec)
	if extraErr != nil {
		return model.RPDBWork{}, fmt.Errorf("encode RPDB demo work %s extra: %w", spec.Slug, extraErr)
	}

	values := map[string]interface{}{
		"author_id":              author.ID,
		"type":                   spec.Type,
		"title":                  spec.Title,
		"slug":                   spec.Slug,
		"summary":                spec.Summary,
		"content":                spec.Content,
		"content_type":           "html",
		"cover_image":            spec.ImageURL,
		"cover_image_updated_at": now,
		"rp_use_cases":           spec.RPUseCases,
		"effect_description":     spec.Effect,
		"restrictions":           `{"level":"any","group":"roleplay"}`,
		"extra":                  extra,
		"game_version":           "11.2.7",
		"expansion":              spec.Expansion,
		"availability_status":    "available",
		"bind_type":              "account",
		"faction":                spec.Faction,
		"armor_type":             spec.ArmorType,
		"status":                 model.RPDBStatusPublished,
		"is_public":              true,
		"review_status":          model.RPDBReviewApproved,
		"reviewer_id":            reviewer.ID,
		"review_comment":         "RPDB 演示数据自动审核通过",
		"reviewed_at":            now,
		"version":                1,
	}
	if errors.Is(findErr, gorm.ErrRecordNotFound) {
		reviewerID := reviewer.ID
		work = model.RPDBWork{
			AuthorID:            author.ID,
			Type:                spec.Type,
			Title:               spec.Title,
			Slug:                spec.Slug,
			Summary:             spec.Summary,
			Content:             spec.Content,
			ContentType:         "html",
			CoverImage:          spec.ImageURL,
			CoverImageUpdatedAt: &now,
			RPUseCases:          spec.RPUseCases,
			EffectDescription:   spec.Effect,
			Restrictions:        `{"level":"any","group":"roleplay"}`,
			Extra:               extra,
			GameVersion:         "11.2.7",
			Expansion:           spec.Expansion,
			AvailabilityStatus:  "available",
			BindType:            "account",
			Faction:             spec.Faction,
			ArmorType:           spec.ArmorType,
			Status:              model.RPDBStatusPublished,
			IsPublic:            true,
			ReviewStatus:        model.RPDBReviewApproved,
			ReviewerID:          &reviewerID,
			ReviewComment:       "RPDB 演示数据自动审核通过",
			ReviewedAt:          &now,
			Version:             1,
		}
		if err := tx.Create(&work).Error; err != nil {
			return model.RPDBWork{}, fmt.Errorf("create RPDB demo work %s: %w", spec.Slug, err)
		}
	} else if err := tx.Model(&work).Updates(values).Error; err != nil {
		return model.RPDBWork{}, fmt.Errorf("update RPDB demo work %s: %w", spec.Slug, err)
	}
	if err := tx.First(&work, work.ID).Error; err != nil {
		return model.RPDBWork{}, fmt.Errorf("reload RPDB demo work %s: %w", spec.Slug, err)
	}
	return work, nil
}

func ensureRPDBDemoWorkChildren(
	tx *gorm.DB,
	work model.RPDBWork,
	author model.User,
	reviewer model.User,
	spec rpdbDemoWorkSpec,
	tags map[string]model.Tag,
	now time.Time,
) error {
	references, err := ensureRPDBDemoReferences(tx, work, spec)
	if err != nil {
		return err
	}
	if err := ensureRPDBDemoMedia(tx, work, author, reviewer, spec, now); err != nil {
		return err
	}
	if work.Type == model.RPDBWorkTypeHomeShowcase {
		if err := deleteLegacyRPDBDemoHomeGuideSteps(tx, work, spec); err != nil {
			return err
		}
	} else {
		if err := ensureRPDBDemoGuideSteps(tx, work, spec); err != nil {
			return err
		}
	}
	if work.Type == model.RPDBWorkTypeTransmog {
		if err := ensureRPDBDemoTransmogSlots(tx, work, references); err != nil {
			return err
		}
	}
	if err := tx.Where("work_id = ?", work.ID).Delete(&model.RPDBTag{}).Error; err != nil {
		return fmt.Errorf("clear tags for %s: %w", spec.Slug, err)
	}
	for _, name := range spec.Tags {
		tag, ok := tags[name]
		if !ok {
			return fmt.Errorf("RPDB demo work %s references missing tag %s", spec.Slug, name)
		}
		association := model.RPDBTag{WorkID: work.ID, TagID: tag.ID, AddedBy: author.ID}
		if err := tx.Where("work_id = ? AND tag_id = ?", work.ID, tag.ID).FirstOrCreate(&association).Error; err != nil {
			return fmt.Errorf("attach tag %s to %s: %w", name, spec.Slug, err)
		}
	}
	return nil
}

func deleteLegacyRPDBDemoHomeGuideSteps(tx *gorm.DB, work model.RPDBWork, spec rpdbDemoWorkSpec) error {
	if err := tx.Where("work_id = ?", work.ID).Delete(&model.RPDBGuideStep{}).Error; err != nil {
		return fmt.Errorf("delete legacy home guide steps for %s: %w", spec.Slug, err)
	}
	return nil
}

func ensureRPDBDemoReferences(tx *gorm.DB, work model.RPDBWork, spec rpdbDemoWorkSpec) ([]model.RPDBReference, error) {
	references := make([]model.RPDBReference, 0, len(spec.ReferenceIDs))
	for index, externalID := range spec.ReferenceIDs {
		var reference model.RPDBReference
		err := tx.Where(
			"work_id = ? AND external_type = ? AND external_id = ?",
			work.ID,
			spec.ReferenceType,
			externalID,
		).First(&reference).Error
		name := spec.Title
		if len(spec.ReferenceIDs) > 1 {
			name = fmt.Sprintf("%s 部件 %d", spec.Title, index+1)
		}
		urlKind := "item"
		if spec.ReferenceType == "decor" {
			urlKind = "object"
		}
		values := map[string]interface{}{
			"name":       name,
			"icon":       fmt.Sprintf("inv_misc_questionmark_%02d", index+1),
			"quality":    []string{"rare", "epic", "uncommon", "common"}[index%4],
			"source":     "wowhead",
			"url":        fmt.Sprintf("https://www.wowhead.com/%s=%s", urlKind, externalID),
			"locale":     "zhCN",
			"is_primary": index == 0,
			"sort_order": index + 1,
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			reference = model.RPDBReference{
				WorkID:       work.ID,
				ExternalType: spec.ReferenceType,
				ExternalID:   externalID,
				Name:         name,
			}
			if err := tx.Create(&reference).Error; err != nil {
				return nil, fmt.Errorf("create reference %s for %s: %w", externalID, spec.Slug, err)
			}
			if err := tx.Model(&reference).Updates(values).Error; err != nil {
				return nil, fmt.Errorf("complete reference %s for %s: %w", externalID, spec.Slug, err)
			}
		} else if err != nil {
			return nil, fmt.Errorf("find reference %s for %s: %w", externalID, spec.Slug, err)
		} else if err := tx.Model(&reference).Updates(values).Error; err != nil {
			return nil, fmt.Errorf("update reference %s for %s: %w", externalID, spec.Slug, err)
		}
		references = append(references, reference)
	}
	return references, nil
}

func ensureRPDBDemoMedia(tx *gorm.DB, work model.RPDBWork, author, reviewer model.User, spec rpdbDemoWorkSpec, now time.Time) error {
	var media model.RPDBMedia
	err := tx.Where("work_id = ? AND url = ?", work.ID, spec.ImageURL).First(&media).Error
	meta := fmt.Sprintf(`{"seed":"rpdb-demo","slug":%q}`, spec.Slug)
	values := map[string]interface{}{
		"author_id":      author.ID,
		"type":           "image",
		"thumbnail_url":  spec.ImageURL,
		"caption":        spec.Title + " 演示图",
		"sort_order":     1,
		"width":          1600,
		"height":         900,
		"meta":           meta,
		"review_status":  model.RPDBReviewApproved,
		"reviewer_id":    reviewer.ID,
		"review_comment": "RPDB 演示媒体自动审核通过",
		"reviewed_at":    now,
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		authorID := author.ID
		reviewerID := reviewer.ID
		media = model.RPDBMedia{
			WorkID:        work.ID,
			AuthorID:      &authorID,
			Type:          "image",
			URL:           spec.ImageURL,
			ThumbnailURL:  spec.ImageURL,
			Caption:       spec.Title + " 演示图",
			SortOrder:     1,
			Width:         1600,
			Height:        900,
			Meta:          meta,
			ReviewStatus:  model.RPDBReviewApproved,
			ReviewerID:    &reviewerID,
			ReviewComment: "RPDB 演示媒体自动审核通过",
			ReviewedAt:    &now,
		}
		if err := tx.Create(&media).Error; err != nil {
			return fmt.Errorf("create media for %s: %w", spec.Slug, err)
		}
		return nil
	} else if err != nil {
		return fmt.Errorf("find media for %s: %w", spec.Slug, err)
	}
	if err := tx.Model(&media).Updates(values).Error; err != nil {
		return fmt.Errorf("update media for %s: %w", spec.Slug, err)
	}
	return nil
}

func ensureRPDBDemoGuideSteps(tx *gorm.DB, work model.RPDBWork, spec rpdbDemoWorkSpec) error {
	steps := []model.RPDBGuideStep{
		{
			SortOrder:    1,
			Title:        "确认角色设定",
			Body:         "根据作品摘要确认角色身份、阵营与使用场景，提前准备相关台词。",
			Zone:         spec.Zone,
			MapID:        spec.MapID,
			X:            spec.X,
			Y:            spec.Y,
			Label:        spec.Title + " 起点",
			Prerequisite: "完成基础角色设定",
		},
		{
			SortOrder:    2,
			Title:        "收集核心素材",
			Body:         "按引用列表收集核心物品或布置素材，并记录可替代方案。",
			Zone:         spec.Zone,
			MapID:        spec.MapID,
			X:            spec.X + 1.25,
			Y:            spec.Y + 1.10,
			Label:        spec.Title + " 素材点",
			Prerequisite: "确认引用条目",
		},
		{
			SortOrder:    3,
			Title:        "完成演出检查",
			Body:         "检查外观、道具、互动说明和队伍分工，确认后保存到个人清单。",
			Zone:         spec.Zone,
			MapID:        spec.MapID,
			X:            spec.X + 2.05,
			Y:            spec.Y + 0.65,
			Label:        spec.Title + " 完成点",
			Prerequisite: "核心素材齐备",
		},
	}
	for index := range steps {
		step := &steps[index]
		step.WorkID = work.ID
		step.Meta = fmt.Sprintf(`{"seed_key":%q}`, fmt.Sprintf("%s-step-%d", spec.Slug, step.SortOrder))
		var storedSteps []model.RPDBGuideStep
		if err := tx.Where("work_id = ? AND sort_order = ?", work.ID, step.SortOrder).
			Order("id ASC").
			Find(&storedSteps).Error; err != nil {
			return fmt.Errorf("find guide step %d for %s: %w", step.SortOrder, spec.Slug, err)
		}
		if len(storedSteps) == 0 {
			if err := tx.Create(step).Error; err != nil {
				return fmt.Errorf("create guide step %d for %s: %w", step.SortOrder, spec.Slug, err)
			}
			continue
		}

		stored := storedSteps[0]
		if err := tx.Model(&stored).Updates(map[string]interface{}{
			"sort_order":   step.SortOrder,
			"title":        step.Title,
			"body":         step.Body,
			"zone":         step.Zone,
			"map_id":       step.MapID,
			"x":            step.X,
			"y":            step.Y,
			"label":        step.Label,
			"prerequisite": step.Prerequisite,
			"meta":         step.Meta,
		}).Error; err != nil {
			return fmt.Errorf("update guide step %d for %s: %w", step.SortOrder, spec.Slug, err)
		}
		if len(storedSteps) > 1 {
			duplicateIDs := make([]uint, 0, len(storedSteps)-1)
			for _, duplicate := range storedSteps[1:] {
				duplicateIDs = append(duplicateIDs, duplicate.ID)
			}
			if err := tx.Where("id IN ?", duplicateIDs).Delete(&model.RPDBGuideStep{}).Error; err != nil {
				return fmt.Errorf("delete duplicate guide step %d for %s: %w", step.SortOrder, spec.Slug, err)
			}
		}
	}
	return nil
}

func ensureRPDBDemoTransmogSlots(tx *gorm.DB, work model.RPDBWork, references []model.RPDBReference) error {
	slots := []string{"head", "shoulder", "chest", "main_hand"}
	for index, slot := range slots {
		if index >= len(references) {
			return fmt.Errorf("transmog work %s has only %d references", work.Slug, len(references))
		}
		var stored model.RPDBTransmogSlot
		err := tx.Where("work_id = ? AND slot = ?", work.ID, slot).First(&stored).Error
		values := map[string]interface{}{
			"reference_id": references[index].ID,
			"role":         "required",
			"variant":      "default",
			"note":         fmt.Sprintf("%s 的%s部位", work.Title, slot),
			"sort_order":   index + 1,
		}
		if errors.Is(err, gorm.ErrRecordNotFound) {
			stored = model.RPDBTransmogSlot{WorkID: work.ID, Slot: slot}
			if err := tx.Create(&stored).Error; err != nil {
				return fmt.Errorf("create transmog slot %s for %s: %w", slot, work.Slug, err)
			}
		} else if err != nil {
			return fmt.Errorf("find transmog slot %s for %s: %w", slot, work.Slug, err)
		}
		if err := tx.Model(&stored).Updates(values).Error; err != nil {
			return fmt.Errorf("update transmog slot %s for %s: %w", slot, work.Slug, err)
		}
	}
	return nil
}

func ensureRPDBDemoInteractions(tx *gorm.DB, work model.RPDBWork, curator, explorer model.User) error {
	users := []model.User{curator, explorer}
	for _, user := range users {
		like := model.RPDBLike{WorkID: work.ID, UserID: user.ID}
		if err := tx.Where("work_id = ? AND user_id = ?", work.ID, user.ID).FirstOrCreate(&like).Error; err != nil {
			return fmt.Errorf("seed like for %s: %w", work.Slug, err)
		}
		favorite := model.RPDBFavorite{WorkID: work.ID, UserID: user.ID}
		if err := tx.Where("work_id = ? AND user_id = ?", work.ID, user.ID).FirstOrCreate(&favorite).Error; err != nil {
			return fmt.Errorf("seed favorite for %s: %w", work.Slug, err)
		}
		view := model.RPDBView{WorkID: work.ID, UserID: user.ID}
		if err := tx.Where("work_id = ? AND user_id = ?", work.ID, user.ID).FirstOrCreate(&view).Error; err != nil {
			return fmt.Errorf("seed view for %s: %w", work.Slug, err)
		}
		verification := model.RPDBVerification{
			WorkID:      work.ID,
			UserID:      user.ID,
			WorkVersion: work.Version,
			Result:      "valid",
			Comment:     "已按演示步骤核验，内容可用于 RP 场景。",
		}
		if err := tx.Where("work_id = ? AND user_id = ?", work.ID, user.ID).
			Assign(verification).
			FirstOrCreate(&verification).Error; err != nil {
			return fmt.Errorf("seed verification for %s: %w", work.Slug, err)
		}
	}

	comments := []struct {
		Author  model.User
		Content string
	}{
		{Author: curator, Content: fmt.Sprintf("演示评论：%s 的结构清晰，适合直接用于跑团。", work.Title)},
		{Author: explorer, Content: fmt.Sprintf("演示评论：已把 %s 加入收集清单，坐标步骤很实用。", work.Title)},
	}
	for commentIndex, spec := range comments {
		var comment model.RPDBComment
		err := tx.Where(
			"work_id = ? AND author_id = ? AND content = ?",
			work.ID,
			spec.Author.ID,
			spec.Content,
		).First(&comment).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			comment = model.RPDBComment{
				WorkID:   work.ID,
				AuthorID: spec.Author.ID,
				Content:  spec.Content,
				Status:   "published",
			}
			if err := tx.Create(&comment).Error; err != nil {
				return fmt.Errorf("seed comment for %s: %w", work.Slug, err)
			}
		} else if err != nil {
			return fmt.Errorf("find demo comment for %s: %w", work.Slug, err)
		}
		liker := users[(commentIndex+1)%len(users)]
		commentLike := model.RPDBCommentLike{CommentID: comment.ID, UserID: liker.ID}
		if err := tx.Where("comment_id = ? AND user_id = ?", comment.ID, liker.ID).FirstOrCreate(&commentLike).Error; err != nil {
			return fmt.Errorf("seed comment like for %s: %w", work.Slug, err)
		}
		var likeCount int64
		if err := tx.Model(&model.RPDBCommentLike{}).Where("comment_id = ?", comment.ID).Count(&likeCount).Error; err != nil {
			return fmt.Errorf("count comment likes for %s: %w", work.Slug, err)
		}
		if err := tx.Model(&comment).Update("like_count", likeCount).Error; err != nil {
			return fmt.Errorf("update comment like count for %s: %w", work.Slug, err)
		}
	}

	return nil
}

func ensureRPDBDemoList(tx *gorm.DB, owner model.User, works []model.RPDBWork) error {
	var list model.RPDBList
	err := tx.Where("user_id = ? AND is_default = ?", owner.ID, true).Order("id ASC").First(&list).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		err = tx.Where("user_id = ? AND name = ?", owner.ID, "RPDB 演示收集清单").First(&list).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			list = model.RPDBList{
				UserID:      owner.ID,
				Name:        "RPDB 演示收集清单",
				Description: "包含道具、幻化与家园展示的 RPDB 演示收集清单。",
				IsDefault:   true,
				IsPublic:    true,
			}
			if err := tx.Create(&list).Error; err != nil {
				return fmt.Errorf("create RPDB demo list: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("find RPDB demo list: %w", err)
		} else if err := tx.Model(&list).Update("is_default", true).Error; err != nil {
			return fmt.Errorf("make RPDB demo list default: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("find default RPDB list: %w", err)
	}

	statuses := []string{
		model.RPDBListStatusWanted,
		model.RPDBListStatusFarming,
		model.RPDBListStatusOwned,
		model.RPDBListStatusPaused,
	}
	for index, work := range works {
		entry := model.RPDBListEntry{
			ListID:    list.ID,
			WorkID:    work.ID,
			Status:    statuses[index%len(statuses)],
			Priority:  len(works) - index,
			Quantity:  1,
			Note:      "RPDB 演示清单条目",
			SortOrder: index + 1,
		}
		if err := tx.Where("list_id = ? AND work_id = ?", list.ID, work.ID).
			Assign(entry).
			FirstOrCreate(&entry).Error; err != nil {
			return fmt.Errorf("add %s to RPDB demo list: %w", work.Slug, err)
		}
	}
	var count int64
	if err := tx.Model(&model.RPDBListEntry{}).Where("list_id = ?", list.ID).Count(&count).Error; err != nil {
		return fmt.Errorf("count RPDB demo list entries: %w", err)
	}
	if err := tx.Model(&list).Update("item_count", count).Error; err != nil {
		return fmt.Errorf("update RPDB demo list count: %w", err)
	}
	return nil
}

func ensureRPDBDemoSet(tx *gorm.DB, owner model.User, works []model.RPDBWork) error {
	var set model.RPDBSet
	err := tx.Where("author_id = ? AND name = ?", owner.ID, "艾泽拉斯 RP 灵感精选").First(&set).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		set = model.RPDBSet{AuthorID: owner.ID, Name: "艾泽拉斯 RP 灵感精选"}
		if err := tx.Create(&set).Error; err != nil {
			return fmt.Errorf("create RPDB demo set: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("find RPDB demo set: %w", err)
	}
	if err := tx.Model(&set).Updates(map[string]interface{}{
		"description":   "适合快速浏览 RPDB 核心内容形态的公开精选。",
		"cover_image":   "/uploads/rpdb/demo/home-01.jpg",
		"type":          "mixed",
		"is_public":     true,
		"status":        model.RPDBStatusPublished,
		"review_status": model.RPDBReviewApproved,
	}).Error; err != nil {
		return fmt.Errorf("update RPDB demo set: %w", err)
	}
	for index, work := range works {
		entry := model.RPDBSetWork{
			SetID:     set.ID,
			WorkID:    work.ID,
			Role:      "featured",
			SortOrder: index + 1,
			Note:      "RPDB 演示精选",
		}
		if err := tx.Where("set_id = ? AND work_id = ?", set.ID, work.ID).
			Assign(entry).
			FirstOrCreate(&entry).Error; err != nil {
			return fmt.Errorf("add %s to RPDB demo set: %w", work.Slug, err)
		}
	}
	var count int64
	if err := tx.Model(&model.RPDBSetWork{}).Where("set_id = ?", set.ID).Count(&count).Error; err != nil {
		return fmt.Errorf("count RPDB demo set works: %w", err)
	}
	if err := tx.Model(&set).Update("item_count", count).Error; err != nil {
		return fmt.Errorf("update RPDB demo set count: %w", err)
	}
	return nil
}

func refreshRPDBDemoTagCounts(tx *gorm.DB, tags map[string]model.Tag) error {
	for name, tag := range tags {
		var count int64
		if err := tx.Model(&model.RPDBTag{}).Where("tag_id = ?", tag.ID).Count(&count).Error; err != nil {
			return fmt.Errorf("count usage for RPDB demo tag %s: %w", name, err)
		}
		if err := tx.Model(&tag).Update("usage_count", count).Error; err != nil {
			return fmt.Errorf("update usage for RPDB demo tag %s: %w", name, err)
		}
	}
	return nil
}

func refreshRPDBDemoWorkCounts(tx *gorm.DB, workID uint, now time.Time) error {
	count := func(target interface{}, query string, args ...interface{}) (int64, error) {
		var value int64
		err := tx.Model(target).Where(query, args...).Count(&value).Error
		return value, err
	}

	views, err := count(&model.RPDBView{}, "work_id = ?", workID)
	if err != nil {
		return fmt.Errorf("count views for work %d: %w", workID, err)
	}
	likes, err := count(&model.RPDBLike{}, "work_id = ?", workID)
	if err != nil {
		return fmt.Errorf("count likes for work %d: %w", workID, err)
	}
	favorites, err := count(&model.RPDBFavorite{}, "work_id = ?", workID)
	if err != nil {
		return fmt.Errorf("count favorites for work %d: %w", workID, err)
	}
	comments, err := count(&model.RPDBComment{}, "work_id = ? AND status = ?", workID, "published")
	if err != nil {
		return fmt.Errorf("count comments for work %d: %w", workID, err)
	}
	lists, err := count(&model.RPDBListEntry{}, "work_id = ?", workID)
	if err != nil {
		return fmt.Errorf("count list entries for work %d: %w", workID, err)
	}
	media, err := count(&model.RPDBMedia{}, "work_id = ?", workID)
	if err != nil {
		return fmt.Errorf("count media for work %d: %w", workID, err)
	}
	valid, err := count(&model.RPDBVerification{}, "work_id = ? AND result = ?", workID, "valid")
	if err != nil {
		return fmt.Errorf("count valid verifications for work %d: %w", workID, err)
	}
	outdated, err := count(&model.RPDBVerification{}, "work_id = ? AND result = ?", workID, "outdated")
	if err != nil {
		return fmt.Errorf("count outdated verifications for work %d: %w", workID, err)
	}

	verificationStatus := model.RPDBVerificationUnverified
	if valid >= 2 && valid > outdated {
		verificationStatus = model.RPDBVerificationVerified
	} else if outdated > valid && outdated >= 1 {
		verificationStatus = model.RPDBVerificationStale
	} else if valid > 0 && outdated > 0 {
		verificationStatus = model.RPDBVerificationDisputed
	}
	updates := map[string]interface{}{
		"view_count":          views,
		"like_count":          likes,
		"favorite_count":      favorites,
		"comment_count":       comments,
		"list_count":          lists,
		"media_count":         media,
		"verified_count":      valid,
		"outdated_count":      outdated,
		"verification_status": verificationStatus,
	}
	if valid > 0 {
		updates["last_verified_at"] = now
	}
	if err := tx.Model(&model.RPDBWork{}).Where("id = ?", workID).Updates(updates).Error; err != nil {
		return fmt.Errorf("update interaction counts for work %d: %w", workID, err)
	}
	return nil
}

func rpdbDemoWorkSpecs() []rpdbDemoWorkSpec {
	return []rpdbDemoWorkSpec{
		{
			Slug: "rpdb-demo-item-01", Type: model.RPDBWorkTypeItemShowcase, Title: "月影旅灯",
			Summary:    "适合夜间巡逻、旅店守夜与野外营地的便携 RP 道具。",
			Content:    "<p>一盏带有月白色滤片的旅行灯，附带点灯、调光与熄灯动作建议。</p>",
			RPUseCases: "夜间巡逻；营地守夜；旅店迎宾", Effect: "为暗光场景提供明确的视觉焦点与互动触发点。",
			Expansion: "The War Within", Faction: "neutral", Zone: "暮色森林", MapID: "47", X: 74.2, Y: 45.1,
			ImageURL: "/uploads/rpdb/demo/item-01.jpg", ReferenceIDs: []string{"19019"}, ReferenceType: "item",
			Tags: []string{"联盟风格", "洛丹伦风格", "军旅哨站风格"},
		},
		{
			Slug: "rpdb-demo-item-02", Type: model.RPDBWorkTypeItemShowcase, Title: "秘银调查工具箱",
			Summary:    "面向侦探、工程师与城市守卫角色的线索采集工具组合。",
			Content:    "<p>工具箱包含测量尺、样本瓶、炭笔和封蜡，可拆分成多人调查互动。</p>",
			RPUseCases: "案件调查；遗迹勘察；工程检修", Effect: "把抽象调查过程拆分为可表演的动作步骤。",
			Expansion: "Dragonflight", Faction: "alliance", Zone: "暴风城", MapID: "84", X: 62.4, Y: 31.8,
			ImageURL: "/uploads/rpdb/demo/item-02.jpg", ReferenceIDs: []string{"6948"}, ReferenceType: "item",
			Tags: []string{"联盟风格", "侏儒工程风格", "暴风城风格"},
		},
		{
			Slug: "rpdb-demo-item-03", Type: model.RPDBWorkTypeItemShowcase, Title: "旅店留言簿",
			Summary:    "用于酒馆、旅店与公会据点的持续剧情留言载体。",
			Content:    "<p>每页提供日期、署名、目的地与留言区域，适合串联异步剧情。</p>",
			RPUseCases: "旅店经营；委托发布；异步剧情", Effect: "让离线角色也能通过留言参与持续剧情。",
			Expansion: "The War Within", Faction: "neutral", Zone: "伯拉勒斯", MapID: "1161", X: 74.8, Y: 22.6,
			ImageURL: "/uploads/rpdb/demo/item-03.jpg", ReferenceIDs: []string{"38682"}, ReferenceType: "item",
			Tags: []string{"库尔提拉斯风格", "海港酒馆风格", "联盟风格"},
		},
		{
			Slug: "rpdb-demo-item-04", Type: model.RPDBWorkTypeItemShowcase, Title: "炼金师便携茶具",
			Summary:    "把药剂学、茶会与野外休整结合起来的轻量 RP 套件。",
			Content:    "<p>茶具包含小炉、滤网、试剂瓶与两只旅行杯，可用于药草辨识小游戏。</p>",
			RPUseCases: "野外休整；炼金教学；社交茶会", Effect: "将补给休整转化为有节奏的社交桥段。",
			Expansion: "Mists of Pandaria", Faction: "neutral", Zone: "四风谷", MapID: "376", X: 53.5, Y: 50.7,
			ImageURL: "/uploads/rpdb/demo/item-04.jpg", ReferenceIDs: []string{"86569"}, ReferenceType: "item",
			Tags: []string{"熊猫人风格", "炼金工坊风格", "荒野游侠风格"},
		},
		{
			Slug: "rpdb-demo-transmog-01", Type: model.RPDBWorkTypeTransmog, Title: "暮色守夜人",
			Summary:    "低饱和皮甲与冷色武器构成的夜巡守卫幻化。",
			Content:    "<p>强调轻装、兜帽与旧式武器，适合边境巡逻和调查剧情。</p>",
			RPUseCases: "夜巡守卫；赏金猎人；边境调查", Effect: "建立可靠但不过度华丽的基层守卫形象。",
			Expansion: "Cataclysm", Faction: "alliance", ArmorType: "leather", Zone: "暮色森林", MapID: "47", X: 72.1, Y: 46.4,
			ImageURL: "/uploads/rpdb/demo/transmog-01.jpg", ReferenceIDs: []string{"31026", "31023", "31029", "30865"}, ReferenceType: "item",
			Tags: []string{"联盟风格", "洛丹伦风格", "荒野游侠风格"},
		},
		{
			Slug: "rpdb-demo-transmog-02", Type: model.RPDBWorkTypeTransmog, Title: "银月秘法使",
			Summary:    "金红配色与轻型法杖组合的辛多雷学者幻化。",
			Content:    "<p>保留银月城礼仪感，同时降低战斗装饰，适合学术与外交场景。</p>",
			RPUseCases: "秘法研究；外交会谈；学院教学", Effect: "强化高等精灵学者的秩序感与辨识度。",
			Expansion: "The Burning Crusade", Faction: "horde", ArmorType: "cloth", Zone: "银月城", MapID: "110", X: 56.3, Y: 50.2,
			ImageURL: "/uploads/rpdb/demo/transmog-02.jpg", ReferenceIDs: []string{"34339", "34202", "34399", "34337"}, ReferenceType: "item",
			Tags: []string{"部落风格", "银月城风格", "达拉然风格"},
		},
		{
			Slug: "rpdb-demo-transmog-03", Type: model.RPDBWorkTypeTransmog, Title: "库尔提拉斯航海官",
			Summary:    "深蓝长衣、铜色肩饰与军官佩剑组成的海军风幻化。",
			Content:    "<p>适用于舰队会议、港口巡查和远洋探险开场。</p>",
			RPUseCases: "舰队军官；港务巡查；远洋探险", Effect: "快速传达航海经验与指挥身份。",
			Expansion: "Battle for Azeroth", Faction: "alliance", ArmorType: "mail", Zone: "伯拉勒斯", MapID: "1161", X: 66.8, Y: 24.9,
			ImageURL: "/uploads/rpdb/demo/transmog-03.jpg", ReferenceIDs: []string{"160110", "160111", "160112", "160113"}, ReferenceType: "item",
			Tags: []string{"库尔提拉斯风格", "联盟风格", "海港酒馆风格"},
		},
		{
			Slug: "rpdb-demo-transmog-04", Type: model.RPDBWorkTypeTransmog, Title: "德鲁斯瓦猎巫者",
			Summary:    "厚重皮革、骨饰与暗色火枪组合的林地猎手幻化。",
			Content:    "<p>以耐候装备和民俗护符表现长期在诅咒林地行动的经验。</p>",
			RPUseCases: "猎巫调查；林地护卫；怪物追踪", Effect: "为阴森民俗剧情提供鲜明的职业轮廓。",
			Expansion: "Battle for Azeroth", Faction: "alliance", ArmorType: "mail", Zone: "德鲁斯瓦", MapID: "896", X: 37.4, Y: 49.8,
			ImageURL: "/uploads/rpdb/demo/transmog-04.jpg", ReferenceIDs: []string{"159392", "159393", "159394", "159395"}, ReferenceType: "item",
			Tags: []string{"德鲁斯瓦风格", "库尔提拉斯风格", "暗影诅咒风格"},
		},
		{
			Slug: "rpdb-demo-home-01", Type: model.RPDBWorkTypeHomeShowcase, Title: "暴风城旧城区侦探事务所",
			Summary:    "包含接待区、证物墙与档案桌的小型城市调查据点。",
			Content:    "<p>空间按接待、问询、档案和秘密出口四个叙事区域组织。</p>",
			RPUseCases: "案件委托；线索复盘；秘密会面", Effect: "在有限空间内支持连续调查剧情。",
			Expansion: "The War Within", Faction: "alliance", Zone: "暴风城", MapID: "84", X: 66.1, Y: 61.3,
			ImageURL: "/uploads/rpdb/demo/home-01.jpg", ReferenceIDs: []string{"900101", "900102"}, ReferenceType: "decor",
			Tags:   []string{"暴风城风格", "联盟风格", "贵族沙龙风格"},
			Server: "国服-主宰之剑", Region: "暴风城旧城区", HomeStyle: "城市侦探事务所", ShareCode: "RPBOX-HOME-01",
			VisitNotes: "工作日晚间开放参观，请从接待区开始浏览。", CopyStatus: "copyable", VisitStatus: "appointment", SpaceType: "indoor",
		},
		{
			Slug: "rpdb-demo-home-02", Type: model.RPDBWorkTypeHomeShowcase, Title: "银月城占星师书房",
			Summary:    "以星盘、卷轴和红金织物构成的秘法研究空间。",
			Content:    "<p>中央星盘负责演示，周边书架承担资料检索与私人谈话功能。</p>",
			RPUseCases: "占星咨询；秘法教学；学术辩论", Effect: "把研究过程转化为可分工的多人互动。",
			Expansion: "Midnight", Faction: "horde", Zone: "银月城", MapID: "110", X: 58.7, Y: 42.5,
			ImageURL: "/uploads/rpdb/demo/home-02.jpg", ReferenceIDs: []string{"900201", "900202"}, ReferenceType: "decor",
			Tags:   []string{"银月城风格", "部落风格", "达拉然风格"},
			Server: "国服-凤凰之神", Region: "银月城皇家贸易区", HomeStyle: "辛多雷占星书房", ShareCode: "RPBOX-HOME-02",
			VisitNotes: "开放学术参观，星盘区域请勿放置大型坐骑。", CopyStatus: "reference_only", VisitStatus: "open", SpaceType: "indoor",
		},
		{
			Slug: "rpdb-demo-home-03", Type: model.RPDBWorkTypeHomeShowcase, Title: "伯拉勒斯海港酒馆",
			Summary:    "面向水手、公会与旅行者的开放式海港酒馆布置。",
			Content:    "<p>吧台、公告板、演奏角和临海座位形成四条自然互动路线。</p>",
			RPUseCases: "公会集会；委托发布；旅行者社交", Effect: "支持高流动性的多人社交和剧情切入。",
			Expansion: "Battle for Azeroth", Faction: "alliance", Zone: "伯拉勒斯", MapID: "1161", X: 75.6, Y: 21.8,
			ImageURL: "/uploads/rpdb/demo/home-03.jpg", ReferenceIDs: []string{"900301", "900302"}, ReferenceType: "decor",
			Tags:   []string{"库尔提拉斯风格", "海港酒馆风格", "联盟风格"},
			Server: "国服-白银之手", Region: "伯拉勒斯港务区", HomeStyle: "海港酒馆", ShareCode: "RPBOX-HOME-03",
			VisitNotes: "全天开放公共区域，二层包间需提前预约。", CopyStatus: "copyable", VisitStatus: "open", SpaceType: "mixed",
		},
		{
			Slug: "rpdb-demo-home-04", Type: model.RPDBWorkTypeHomeShowcase, Title: "德鲁斯瓦林地祭坛",
			Summary:    "由石台、护符、烛火与药草组成的低照度民俗祭坛。",
			Content:    "<p>祭坛保留环形站位与观察通道，适合仪式、调查和冲突剧情。</p>",
			RPUseCases: "民俗仪式；诅咒调查；林地守护", Effect: "为暗色民俗剧情提供明确的空间秩序。",
			Expansion: "Battle for Azeroth", Faction: "neutral", Zone: "德鲁斯瓦", MapID: "896", X: 33.8, Y: 52.1,
			ImageURL: "/uploads/rpdb/demo/home-04.jpg", ReferenceIDs: []string{"900401", "900402"}, ReferenceType: "decor",
			Tags:   []string{"德鲁斯瓦风格", "暗影诅咒风格", "荒野游侠风格"},
			Server: "国服-罗宁", Region: "德鲁斯瓦林地", HomeStyle: "林地民俗祭坛", ShareCode: "RPBOX-HOME-04",
			VisitNotes: "仅在剧情活动期间开放，参观者请保持低照度设置。", CopyStatus: "reference_only", VisitStatus: "event_only", SpaceType: "outdoor",
		},
	}
}

func rpdbDemoWorkExtra(spec rpdbDemoWorkSpec) (string, error) {
	extra := map[string]string{
		"seed": "rpdb-demo",
		"slug": spec.Slug,
	}
	if spec.Type == model.RPDBWorkTypeHomeShowcase {
		extra["server"] = spec.Server
		extra["region"] = spec.Region
		extra["home_style"] = spec.HomeStyle
		extra["share_code"] = spec.ShareCode
		extra["visit_notes"] = spec.VisitNotes
		extra["copy_status"] = spec.CopyStatus
		extra["visit_status"] = spec.VisitStatus
		extra["space_type"] = spec.SpaceType
	}
	data, err := json.Marshal(extra)
	if err != nil {
		return "", err
	}
	return string(data), nil
}
