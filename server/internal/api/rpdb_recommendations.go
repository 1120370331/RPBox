package api

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/cache"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
)

const (
	rpdbRecommendationLikeWeight       = 8
	rpdbRecommendationFavoriteWeight   = 12
	rpdbRecommendationViewWeight       = 2
	rpdbRecommendationListWeight       = 10
	rpdbRecommendationCreatorWeight    = 7
	rpdbRecommendationSameAuthorWeight = 18
)

type rpdbRecommendationSignals struct {
	Likes      int  `json:"likes"`
	Favorites  int  `json:"favorites"`
	Views      int  `json:"views"`
	Lists      int  `json:"lists"`
	Creators   int  `json:"creators"`
	SameAuthor bool `json:"same_author"`
}

type rpdbRecommendationScore struct {
	Score   int
	Signals rpdbRecommendationSignals
}

type rpdbRecommendation struct {
	rpdbWorkCard
	RecommendationScore   int                       `json:"recommendation_score"`
	RecommendationReasons []string                  `json:"recommendation_reasons"`
	RecommendationSignals rpdbRecommendationSignals `json:"recommendation_signals"`
}

type rpdbRecommendationResponse struct {
	Recommendations []rpdbRecommendation `json:"recommendations"`
}

type rpdbRecommendationInteraction struct {
	WorkID uint
	UserID uint
}

func (s *Server) listRPDBWorkRecommendations(c *gin.Context) {
	workID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || workID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的作品 ID"})
		return
	}
	limit := parsePositiveInt(c.Query("limit"), 6)
	if limit > 12 {
		limit = 12
	}

	viewerID := optionalRPDBUserID(c)
	var current model.RPDBWork
	if err := database.DB.First(&current, uint(workID)).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "查询作品失败"})
		return
	}
	if !canViewRPDBWork(current, viewerID) || isContentHidden(viewerID, reportTargetRPDBWork, current.ID) {
		c.JSON(http.StatusNotFound, gin.H{"error": "作品不存在"})
		return
	}

	var cacheKey string
	if viewerID == 0 && s.cache != nil {
		if version, err := s.cache.Version(c.Request.Context(), rpdbListCacheName); err == nil {
			cacheKey = cache.VersionedKey(
				rpdbListCacheName,
				version,
				cache.HashKey(fmt.Sprintf("recommendations:%d:%d", current.ID, limit)),
			)
			var cached rpdbRecommendationResponse
			if err := s.cache.Get(c.Request.Context(), cacheKey, &cached); err == nil {
				c.JSON(http.StatusOK, cached)
				return
			}
		}
	}

	recommendations, err := buildRPDBRecommendations(current, viewerID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载相关推荐失败"})
		return
	}
	response := rpdbRecommendationResponse{Recommendations: recommendations}
	if cacheKey != "" {
		_ = s.cache.Set(c.Request.Context(), cacheKey, response, cache.TTL["rpdb:list"])
	}
	c.JSON(http.StatusOK, response)
}

func buildRPDBRecommendations(current model.RPDBWork, viewerID uint, limit int) ([]rpdbRecommendation, error) {
	audience, err := loadRPDBRecommendationAudience(current.ID)
	if err != nil {
		return nil, err
	}
	scores := make(map[uint]*rpdbRecommendationScore)
	addScore := func(workID uint, weight int, update func(*rpdbRecommendationSignals)) {
		if workID == 0 || workID == current.ID {
			return
		}
		entry := scores[workID]
		if entry == nil {
			entry = &rpdbRecommendationScore{}
			scores[workID] = entry
		}
		entry.Score += weight
		update(&entry.Signals)
	}

	if len(audience) > 0 {
		userIDs := make([]uint, 0, len(audience))
		for userID := range audience {
			userIDs = append(userIDs, userID)
		}
		if err := addRPDBRecommendationInteractions(
			database.DB.Model(&model.RPDBLike{}),
			userIDs,
			current.ID,
			func(workID uint) {
				addScore(workID, rpdbRecommendationLikeWeight, func(signals *rpdbRecommendationSignals) {
					signals.Likes++
				})
			},
		); err != nil {
			return nil, err
		}
		if err := addRPDBRecommendationInteractions(
			database.DB.Model(&model.RPDBFavorite{}),
			userIDs,
			current.ID,
			func(workID uint) {
				addScore(workID, rpdbRecommendationFavoriteWeight, func(signals *rpdbRecommendationSignals) {
					signals.Favorites++
				})
			},
		); err != nil {
			return nil, err
		}
		if err := addRPDBRecommendationInteractions(
			database.DB.Model(&model.RPDBViewEvent{}),
			userIDs,
			current.ID,
			func(workID uint) {
				addScore(workID, rpdbRecommendationViewWeight, func(signals *rpdbRecommendationSignals) {
					signals.Views++
				})
			},
		); err != nil {
			return nil, err
		}

		var listInteractions []rpdbRecommendationInteraction
		if err := database.DB.
			Table("rpdb_list_entries").
			Select("DISTINCT rpdb_list_entries.work_id, rpdb_lists.user_id").
			Joins("JOIN rpdb_lists ON rpdb_lists.id = rpdb_list_entries.list_id").
			Where("rpdb_lists.user_id IN ? AND rpdb_list_entries.work_id <> ?", userIDs, current.ID).
			Scan(&listInteractions).Error; err != nil {
			return nil, err
		}
		for _, item := range listInteractions {
			addScore(item.WorkID, rpdbRecommendationListWeight, func(signals *rpdbRecommendationSignals) {
				signals.Lists++
			})
		}

		var authored []model.RPDBWork
		if err := database.DB.
			Select("id").
			Where("author_id IN ? AND id <> ?", userIDs, current.ID).
			Where("status = ? AND review_status = ? AND is_public = ?",
				model.RPDBStatusPublished,
				model.RPDBReviewApproved,
				true,
			).
			Find(&authored).Error; err != nil {
			return nil, err
		}
		for _, work := range authored {
			addScore(work.ID, rpdbRecommendationCreatorWeight, func(signals *rpdbRecommendationSignals) {
				signals.Creators++
			})
		}
	}

	var sameAuthorWorks []model.RPDBWork
	if err := database.DB.
		Select("id").
		Where("author_id = ? AND id <> ?", current.AuthorID, current.ID).
		Where("status = ? AND review_status = ? AND is_public = ?",
			model.RPDBStatusPublished,
			model.RPDBReviewApproved,
			true,
		).
		Find(&sameAuthorWorks).Error; err != nil {
		return nil, err
	}
	for _, work := range sameAuthorWorks {
		addScore(work.ID, rpdbRecommendationSameAuthorWeight, func(signals *rpdbRecommendationSignals) {
			signals.SameAuthor = true
		})
	}

	if len(scores) == 0 {
		return []rpdbRecommendation{}, nil
	}
	candidateIDs := make([]uint, 0, len(scores))
	for workID := range scores {
		candidateIDs = append(candidateIDs, workID)
	}
	query := database.DB.
		Where("id IN ?", candidateIDs).
		Where("status = ? AND review_status = ? AND is_public = ?",
			model.RPDBStatusPublished,
			model.RPDBReviewApproved,
			true,
		)
	if hiddenIDs, err := hiddenContentIDs(viewerID, reportTargetRPDBWork); err == nil && len(hiddenIDs) > 0 {
		query = query.Where("id NOT IN ?", hiddenIDs)
	}
	var works []model.RPDBWork
	if err := query.Find(&works).Error; err != nil {
		return nil, err
	}
	sort.SliceStable(works, func(i, j int) bool {
		left := scores[works[i].ID]
		right := scores[works[j].ID]
		if left.Score != right.Score {
			return left.Score > right.Score
		}
		leftEngagement := works[i].FavoriteCount*4 + works[i].ListCount*4 + works[i].LikeCount*3 + works[i].ViewCount
		rightEngagement := works[j].FavoriteCount*4 + works[j].ListCount*4 + works[j].LikeCount*3 + works[j].ViewCount
		if leftEngagement != rightEngagement {
			return leftEngagement > rightEngagement
		}
		return works[i].UpdatedAt.After(works[j].UpdatedAt)
	})
	if len(works) > limit {
		works = works[:limit]
	}

	cards, err := buildRPDBWorkCards(works, viewerID)
	if err != nil {
		return nil, err
	}
	result := make([]rpdbRecommendation, 0, len(cards))
	for _, card := range cards {
		score := scores[card.ID]
		result = append(result, rpdbRecommendation{
			rpdbWorkCard:          card,
			RecommendationScore:   score.Score,
			RecommendationReasons: rpdbRecommendationReasons(score.Signals),
			RecommendationSignals: score.Signals,
		})
	}
	return result, nil
}

func loadRPDBRecommendationAudience(workID uint) (map[uint]struct{}, error) {
	audience := make(map[uint]struct{})
	addUsers := func(query *gorm.DB) error {
		var userIDs []uint
		if err := query.Distinct().Pluck("user_id", &userIDs).Error; err != nil {
			return err
		}
		for _, userID := range userIDs {
			if userID != 0 {
				audience[userID] = struct{}{}
			}
		}
		return nil
	}
	if err := addUsers(database.DB.Model(&model.RPDBLike{}).Where("work_id = ?", workID)); err != nil {
		return nil, err
	}
	if err := addUsers(database.DB.Model(&model.RPDBFavorite{}).Where("work_id = ?", workID)); err != nil {
		return nil, err
	}
	if err := addUsers(database.DB.Model(&model.RPDBViewEvent{}).Where("work_id = ?", workID)); err != nil {
		return nil, err
	}
	var listUserIDs []uint
	if err := database.DB.
		Table("rpdb_list_entries").
		Select("DISTINCT rpdb_lists.user_id").
		Joins("JOIN rpdb_lists ON rpdb_lists.id = rpdb_list_entries.list_id").
		Where("rpdb_list_entries.work_id = ?", workID).
		Pluck("rpdb_lists.user_id", &listUserIDs).Error; err != nil {
		return nil, err
	}
	for _, userID := range listUserIDs {
		if userID != 0 {
			audience[userID] = struct{}{}
		}
	}
	return audience, nil
}

func addRPDBRecommendationInteractions(query *gorm.DB, userIDs []uint, currentWorkID uint, add func(uint)) error {
	var interactions []rpdbRecommendationInteraction
	if err := query.
		Select("DISTINCT work_id, user_id").
		Where("user_id IN ? AND work_id <> ?", userIDs, currentWorkID).
		Scan(&interactions).Error; err != nil {
		return err
	}
	for _, item := range interactions {
		add(item.WorkID)
	}
	return nil
}

func rpdbRecommendationReasons(signals rpdbRecommendationSignals) []string {
	reasons := make([]string, 0, 4)
	if signals.SameAuthor {
		reasons = append(reasons, "同作者作品")
	}
	if signals.Favorites > 0 {
		reasons = append(reasons, fmt.Sprintf("%d 位相关玩家收藏", signals.Favorites))
	}
	if signals.Lists > 0 {
		reasons = append(reasons, fmt.Sprintf("%d 位相关玩家加入清单", signals.Lists))
	}
	if signals.Likes > 0 {
		reasons = append(reasons, fmt.Sprintf("%d 位相关玩家点赞", signals.Likes))
	}
	if signals.Creators > 0 {
		reasons = append(reasons, "相关玩家创作")
	}
	if signals.Views > 0 {
		reasons = append(reasons, fmt.Sprintf("%d 位相关玩家浏览", signals.Views))
	}
	return reasons
}
