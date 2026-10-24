package api

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"gorm.io/gorm"
)

const homeFeedSize = 3

type homeContent struct {
	ID             uint       `json:"id"`
	Kind           string     `json:"kind"`
	Title          string     `json:"title"`
	AuthorID       uint       `json:"-"`
	AuthorName     string     `json:"author_name"`
	ViewCount      *int       `json:"view_count,omitempty"`
	LikeCount      int        `json:"like_count"`
	FavoriteCount  int        `json:"favorite_count"`
	CommentCount   *int       `json:"comment_count,omitempty"`
	ListCount      *int       `json:"list_count,omitempty"`
	Downloads      *int       `json:"downloads,omitempty"`
	Category       string     `json:"-"`
	HasCover       bool       `json:"has_cover"`
	CoverURL       string     `json:"cover_url"`
	CoverUpdatedAt *time.Time `json:"cover_updated_at,omitempty"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CreatedAt      time.Time  `json:"-"`
	WeeklyScore    float64    `json:"weekly_score"`
	Affinity       float64    `json:"-"`
	Reason         string     `json:"reason"`
}

type homeFeedResponse struct {
	Entries      []homeContent `json:"entries"`
	Personalized bool          `json:"personalized"`
	HasMore      bool          `json:"has_more"`
}

// All identifiers come from this fixed table, never from request input.
type homeSource struct {
	kind, table, title, category, cover, coverUpdated, targetType string
	likes, favorites, views, contentID, viewTime                  string
	tags                                                          string
}

var homeSources = []homeSource{
	{"post", "posts", "title", "category", "cover_image", "cover_image_updated_at", reportTargetPost, "post_likes", "post_favorites", "post_views", "post_id", "updated_at", "post_tags"},
	{"rpdb", "rpdb_works", "title", "type", "cover_image", "cover_image_updated_at", reportTargetRPDBWork, "rpdb_likes", "rpdb_favorites", "rpdb_view_events", "work_id", "created_at", "rpdb_tags"},
	{"item", "items", "name", "type", "preview_image", "preview_image_updated_at", reportTargetItem, "item_likes", "item_favorites", "item_views", "item_id", "updated_at", "item_tags"},
}

type homeInteraction struct {
	Kind       string
	ID         uint
	Weight     float64
	OccurredAt time.Time
}

type homePreferences struct {
	kinds      map[string]float64
	categories map[string]float64
	authors    map[string]float64
	tags       map[string]float64
	seen       map[string]bool
}

func (s *Server) getHomeFeatured(c *gin.Context) {
	s.serveHomeFeed(c, false)
}

func (s *Server) getHomeRecommendations(c *gin.Context) {
	s.serveHomeFeed(c, true)
}

func (s *Server) serveHomeFeed(c *gin.Context, personalized bool) {
	excluded := map[string]bool{}
	if personalized {
		parts := strings.Split(c.Query("exclude"), ",")
		if len(parts) > 60 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "排除内容过多"})
			return
		}
		for _, part := range parts {
			if part == "" {
				continue
			}
			kind, rawID, ok := strings.Cut(part, ":")
			id, err := strconv.ParseUint(rawID, 10, 32)
			if !ok || err != nil || id == 0 || (kind != "post" && kind != "rpdb" && kind != "item") {
				c.JSON(http.StatusBadRequest, gin.H{"error": "无效的内容标识"})
				return
			}
			excluded[fmt.Sprintf("%s:%d", kind, id)] = true
		}
	}
	response, err := buildHomeFeed(database.DB.WithContext(c.Request.Context()), c.GetUint("userID"), personalized, excluded, time.Now())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "加载首页内容失败"})
		return
	}
	c.Header("Cache-Control", "private, no-store")
	c.JSON(http.StatusOK, response)
}

func buildHomeFeed(db *gorm.DB, viewerID uint, personalized bool, excluded map[string]bool, now time.Time) (homeFeedResponse, error) {
	response := homeFeedResponse{Entries: []homeContent{}}
	blocked, err := getBlockedUserIDs(viewerID)
	if err != nil {
		return response, err
	}
	hidden, err := getHiddenContentIDMap(viewerID, reportTargetPost, reportTargetItem, reportTargetRPDBWork)
	if err != nil {
		return response, err
	}
	preferences := homePreferences{kinds: map[string]float64{}, categories: map[string]float64{}, authors: map[string]float64{}, tags: map[string]float64{}, seen: map[string]bool{}}
	if personalized {
		if err := preferences.load(db, viewerID, now); err != nil {
			return response, err
		}
		response.Personalized = len(preferences.seen) > 0
	}
	candidates := []homeContent{}
	for _, source := range homeSources {
		query := db.Table(source.table+" AS content").
			Joins("JOIN users AS author ON author.id = content.author_id").
			Where("content.status = ? AND content.review_status = ? AND content.is_public = ? AND author.account_deleted_at IS NULL", "published", "approved", true)
		if source.kind == "rpdb" {
			query = query.Where("COALESCE(content.visibility, '') IN ?", []string{"", "public"})
		}
		if len(blocked) > 0 {
			query = query.Where("content.author_id NOT IN ?", blocked)
		}
		if len(hidden[source.targetType]) > 0 {
			ids := make([]uint, 0, len(hidden[source.targetType]))
			for id := range hidden[source.targetType] {
				ids = append(ids, id)
			}
			query = query.Where("content.id NOT IN ?", ids)
		}
		if personalized {
			query = query.Where("content.author_id <> ?", viewerID)
		}
		activity := homeWeeklyActivity(db, source, now)
		query = query.Joins("LEFT JOIN (?) AS activity ON activity.content_id = content.id", activity)
		coverURL := "''"
		if source.kind == "rpdb" {
			coverURL = "CASE WHEN content.cover_image LIKE 'https://%' OR content.cover_image LIKE 'http://%' OR content.cover_image LIKE '/uploads/%' OR content.cover_image LIKE 'uploads/%' THEN content.cover_image ELSE '' END"
		}
		selectSQL := fmt.Sprintf("content.id, ? AS kind, content.%s AS title, content.author_id, author.username AS author_name, content.%s AS category, (COALESCE(content.%s, '') <> '') AS has_cover, %s AS cover_url, content.%s AS cover_updated_at, content.updated_at, content.created_at, COALESCE(activity.score, 0) AS weekly_score", source.title, source.category, source.cover, coverURL, source.coverUpdated)
		selectSQL += ", COALESCE(content.like_count, 0) AS like_count, COALESCE(content.favorite_count, 0) AS favorite_count"
		switch source.kind {
		case "item":
			selectSQL += ", COALESCE(content.downloads, 0) AS downloads, NULL AS view_count, NULL AS comment_count, NULL AS list_count"
		case "rpdb":
			selectSQL += ", COALESCE(content.view_count, 0) AS view_count, COALESCE(content.comment_count, 0) AS comment_count, COALESCE(content.list_count, 0) AS list_count, NULL AS downloads"
		default:
			selectSQL += ", COALESCE(content.view_count, 0) AS view_count, COALESCE(content.comment_count, 0) AS comment_count, NULL AS list_count, NULL AS downloads"
		}
		args := []interface{}{source.kind}
		if personalized {
			categorySQL, categoryArgs := homePreferenceCase("content."+source.category, preferences.categories, source.kind+":")
			authorSQL, authorArgs := homePreferenceCase("content.author_id", preferences.authors, "")
			tagSQL, tagArgs := homePreferenceCase("tag_id", preferences.tags, "")
			tagQuery := db.Table(source.tags).Select(source.contentID+" AS content_id, SUM("+tagSQL+") AS affinity", tagArgs...).Group(source.contentID)
			query = query.Joins("LEFT JOIN (?) AS tags ON tags.content_id = content.id", tagQuery)
			selectSQL += ", (? + " + categorySQL + " + " + authorSQL + " + COALESCE(tags.affinity, 0)) AS affinity"
			args = append(args, preferences.kinds[source.kind])
			args = append(args, categoryArgs...)
			args = append(args, authorArgs...)
			query = query.Order("affinity DESC")
		} else {
			selectSQL += ", 0 AS affinity"
		}
		var rows []homeContent
		limit := homeFeedSize
		if personalized {
			limit = 60 // bounded pool per source, ranked by affinity before popularity
		}
		if err := query.Select(selectSQL, args...).Order("weekly_score DESC, content.created_at DESC, content.id DESC").Limit(limit).Scan(&rows).Error; err != nil {
			return response, err
		}
		candidates = append(candidates, rows...)
	}
	score := func(item homeContent) float64 {
		if !personalized {
			return item.WeeklyScore
		}
		value := item.Affinity + math.Log1p(item.WeeklyScore)*2 + 2/(1+math.Max(0, now.Sub(item.CreatedAt).Hours()/24)/7)
		if preferences.seen[fmt.Sprintf("%s:%d", item.Kind, item.ID)] {
			value *= 0.15 // prefer discovery over repeatedly recommending consumed content
		}
		return value
	}
	sort.Slice(candidates, func(i, j int) bool {
		left, right := candidates[i], candidates[j]
		if score(left) != score(right) {
			return score(left) > score(right)
		}
		if !left.CreatedAt.Equal(right.CreatedAt) {
			return left.CreatedAt.After(right.CreatedAt)
		}
		if left.Kind != right.Kind {
			return left.Kind < right.Kind
		}
		return left.ID > right.ID
	})
	response.HasMore = len(candidates) > homeFeedSize
	for _, item := range candidates {
		if excluded[fmt.Sprintf("%s:%d", item.Kind, item.ID)] {
			continue
		}
		item.Reason = "popular"
		if personalized && item.Affinity > 0 {
			item.Reason = "interest"
		}
		response.Entries = append(response.Entries, item)
		if len(response.Entries) == homeFeedSize {
			break
		}
	}
	return response, nil
}

func homeWeeklyActivity(db *gorm.DB, source homeSource, now time.Time) *gorm.DB {
	counts := func(table, column string, weight int) *gorm.DB {
		return db.Table(table).Select(source.contentID+" AS content_id, COUNT(*) * ? AS points", weight).
			Where(column+" >= ? AND "+column+" <= ?", now.Add(-7*24*time.Hour), now).Group(source.contentID)
	}
	// Views are unique per user for posts/items and counted once per user/day for RPDB.
	return db.Table("(? UNION ALL ? UNION ALL ?) AS interactions", counts(source.views, source.viewTime, 1), counts(source.likes, "created_at", 3), counts(source.favorites, "created_at", 5)).
		Select("content_id, SUM(points) AS score").Group("content_id")
}

// homePreferenceCase emits parameterized, bounded affinity expressions.
func homePreferenceCase(column string, weights map[string]float64, prefix string) (string, []interface{}) {
	keys := []string{}
	for key := range weights {
		if strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if weights[keys[i]] != weights[keys[j]] {
			return weights[keys[i]] > weights[keys[j]]
		}
		return keys[i] < keys[j]
	})
	if len(keys) > 40 {
		keys = keys[:40]
	}
	if len(keys) == 0 {
		return "0", nil
	}
	var sql strings.Builder
	sql.WriteString("CASE " + column)
	args := []interface{}{}
	for _, key := range keys {
		sql.WriteString(" WHEN ? THEN CAST(? AS DOUBLE PRECISION)")
		var value interface{} = strings.TrimPrefix(key, prefix)
		if prefix == "" {
			value, _ = strconv.ParseUint(key, 10, 64)
		}
		args = append(args, value, weights[key])
	}
	sql.WriteString(" ELSE 0 END")
	return sql.String(), args
}

func (preferences *homePreferences) load(db *gorm.DB, viewerID uint, now time.Time) error {
	for _, source := range homeSources {
		queries := []*gorm.DB{}
		for _, signal := range []struct {
			table, column string
			weight        int
		}{{source.views, source.viewTime, 1}, {source.likes, "created_at", 3}, {source.favorites, "created_at", 5}} {
			queries = append(queries, db.Table(signal.table).Select("? AS kind, "+source.contentID+" AS id, CAST(? AS DOUBLE PRECISION) AS weight, "+signal.column+" AS occurred_at", source.kind, signal.weight).
				Where("user_id = ? AND "+signal.column+" >= ? AND "+signal.column+" <= ?", viewerID, now.Add(-90*24*time.Hour), now))
		}
		var interactions []homeInteraction
		if err := db.Table("(? UNION ALL ? UNION ALL ?) AS signals", queries[0], queries[1], queries[2]).
			Select("kind, id, weight, occurred_at").Order("occurred_at DESC").Limit(200).Scan(&interactions).Error; err != nil {
			return err
		}
		if len(interactions) == 0 {
			continue
		}
		weights, ids := map[uint]float64{}, []uint{}
		for _, interaction := range interactions {
			weight := math.Min(interaction.Weight, 9) / (1 + now.Sub(interaction.OccurredAt).Hours()/24/30)
			if _, exists := weights[interaction.ID]; !exists {
				ids = append(ids, interaction.ID)
			}
			weights[interaction.ID] = math.Min(9, weights[interaction.ID]+weight)
			preferences.seen[fmt.Sprintf("%s:%d", source.kind, interaction.ID)] = true
		}
		var contents []struct {
			ID, AuthorID uint
			Category     string
		}
		if err := db.Table(source.table).Select("id, author_id, "+source.category+" AS category").Where("id IN ?", ids).Scan(&contents).Error; err != nil {
			return err
		}
		for _, content := range contents {
			weight := weights[content.ID]
			preferences.kinds[source.kind] += weight * 0.5
			preferences.categories[source.kind+":"+content.Category] += weight * 2
			preferences.authors[strconv.FormatUint(uint64(content.AuthorID), 10)] += weight
		}
		var tags []struct{ ContentID, TagID uint }
		if err := db.Table(source.tags).Select(source.contentID+" AS content_id, tag_id").Where(source.contentID+" IN ?", ids).Scan(&tags).Error; err != nil {
			return err
		}
		for _, tag := range tags {
			preferences.tags[strconv.FormatUint(uint64(tag.TagID), 10)] += weights[tag.ContentID] * 3
		}
	}
	return nil
}
