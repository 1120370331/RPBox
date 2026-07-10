package service

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/rpbox/server/internal/cache"
	"github.com/rpbox/server/internal/model"
	"gorm.io/gorm"
)

const (
	postListGlobalVersionName = "post:list:global"
	postListViewerVersionName = "post:list:viewer"
	postListCandidateSchema   = "candidate-v1"
)

// PostListQuery describes a public post candidate query.
type PostListQuery struct {
	ViewerID   uint
	Page       int
	PageSize   int
	SortBy     string
	Order      string
	Search     string
	AuthorName string
	Region     string
	Address    string
	TagID      string
	AuthorID   string
	Status     string
	Category   string
	IsPinned   *bool
}

// PostListCandidatePage contains the ordered post IDs for one result page.
type PostListCandidatePage struct {
	IDs   []uint `json:"ids"`
	Total int64  `json:"total"`
}

// PostListService loads and caches public post candidate pages.
type PostListService struct {
	db    *gorm.DB
	cache cache.Cache
}

// NewPostListService creates a public post candidate service.
func NewPostListService(db *gorm.DB, cacheClient cache.Cache) *PostListService {
	return &PostListService{db: db, cache: cacheClient}
}

// Cacheable reports whether candidate ordering is stable enough to cache.
func (q PostListQuery) Cacheable() bool {
	switch strings.ToLower(strings.TrimSpace(q.SortBy)) {
	case "like_count", "view_count":
		return false
	default:
		return true
	}
}

// Candidates returns ordered public post IDs and the total matching count.
func (s *PostListService) Candidates(ctx context.Context, query PostListQuery) (PostListCandidatePage, error) {
	if s == nil || s.db == nil {
		return PostListCandidatePage{}, errors.New("post list service: database is not configured")
	}

	normalized := query.normalized()
	loader := func(context.Context) (interface{}, error) {
		return s.loadCandidates(ctx, normalized)
	}
	loadDirect := func() (PostListCandidatePage, error) {
		value, err := loader(ctx)
		if err != nil {
			return PostListCandidatePage{}, err
		}
		return value.(PostListCandidatePage), nil
	}

	if s.cache == nil || !normalized.Cacheable() {
		return loadDirect()
	}

	globalVersion, err := s.cache.Version(ctx, postListGlobalVersionName)
	if err != nil {
		return loadDirect()
	}
	viewerVersion, err := s.cache.Version(ctx, viewerVersionName(normalized.ViewerID))
	if err != nil {
		return loadDirect()
	}

	key := cache.Key(
		"post",
		"list",
		postListCandidateSchema,
		fmt.Sprintf("g%d", globalVersion),
		fmt.Sprintf("viewer-%d-v%d", normalized.ViewerID, viewerVersion),
		normalized.cacheSuffix(),
	)
	var page PostListCandidatePage
	if err := s.cache.Fetch(ctx, key, cache.TTL["post:list"], &page, loader); err != nil {
		return loadDirect()
	}
	return page, nil
}

// InvalidateGlobal advances the shared public post candidate version.
func (s *PostListService) InvalidateGlobal(ctx context.Context) error {
	if s == nil || s.cache == nil {
		return nil
	}
	_, err := s.cache.BumpVersion(ctx, postListGlobalVersionName)
	return err
}

// InvalidateViewer advances one viewer's visibility candidate version.
func (s *PostListService) InvalidateViewer(ctx context.Context, viewerID uint) error {
	if s == nil || s.cache == nil {
		return nil
	}
	_, err := s.cache.BumpVersion(ctx, viewerVersionName(viewerID))
	return err
}

func (s *PostListService) loadCandidates(ctx context.Context, query PostListQuery) (PostListCandidatePage, error) {
	db := s.db.WithContext(ctx)
	filtered := db.Model(&model.Post{}).
		Where("posts.status = ?", "published").
		Where("posts.review_status = ?", "approved").
		Where("posts.is_public = ?", true)

	if query.ViewerID != 0 {
		blockedAuthors := db.Model(&model.UserBlock{}).
			Select("blocked_user_id").
			Where("blocker_id = ?", query.ViewerID)
		filtered = filtered.Where("posts.author_id NOT IN (?)", blockedAuthors)

		hiddenPosts := db.Model(&model.UserHiddenContent{}).
			Select("target_id").
			Where("user_id = ? AND target_type = ?", query.ViewerID, "post")
		filtered = filtered.Where("posts.id NOT IN (?)", hiddenPosts)
	}

	if query.Category != "" {
		filtered = filtered.Where("posts.category = ?", query.Category)
	}
	if query.Search != "" || query.AuthorName != "" {
		filtered = filtered.Joins("JOIN users ON users.id = posts.author_id")
	}
	if query.Search != "" {
		likeKeyword := "%" + query.Search + "%"
		filtered = filtered.Where(
			"(posts.title LIKE ? OR posts.content LIKE ? OR users.username LIKE ? OR posts.region LIKE ? OR posts.address LIKE ?)",
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
			likeKeyword,
		)
	}
	if query.AuthorName != "" {
		filtered = filtered.Where("users.username LIKE ?", "%"+query.AuthorName+"%")
	}
	if query.Region != "" {
		filtered = filtered.Where("posts.region LIKE ?", "%"+query.Region+"%")
	}
	if query.Address != "" {
		filtered = filtered.Where("posts.address LIKE ?", "%"+query.Address+"%")
	}
	if query.TagID != "" {
		taggedPosts := db.Model(&model.PostTag{}).
			Select("post_id").
			Where("tag_id = ?", query.TagID)
		filtered = filtered.Where("posts.id IN (?)", taggedPosts)
	}
	if query.AuthorID != "" {
		filtered = filtered.Where("posts.author_id = ?", query.AuthorID)
	}
	if query.IsPinned != nil {
		filtered = filtered.Where("posts.is_pinned = ?", *query.IsPinned)
	}

	var total int64
	if err := filtered.Count(&total).Error; err != nil {
		return PostListCandidatePage{}, err
	}

	sortColumn := "posts.created_at"
	switch query.SortBy {
	case "like_count":
		sortColumn = "posts.like_count"
	case "view_count":
		sortColumn = "posts.view_count"
	}

	var ids []uint
	if err := filtered.
		Order(sortColumn+" "+query.Order).
		Order("posts.id "+query.Order).
		Offset((query.Page-1)*query.PageSize).
		Limit(query.PageSize).
		Pluck("posts.id", &ids).Error; err != nil {
		return PostListCandidatePage{}, err
	}
	if ids == nil {
		ids = []uint{}
	}

	return PostListCandidatePage{IDs: ids, Total: total}, nil
}

func (q PostListQuery) normalized() PostListQuery {
	normalized := q
	if normalized.Page < 1 {
		normalized.Page = 1
	}
	if normalized.PageSize < 1 {
		normalized.PageSize = 20
	}

	switch strings.ToLower(strings.TrimSpace(normalized.SortBy)) {
	case "like_count":
		normalized.SortBy = "like_count"
	case "view_count":
		normalized.SortBy = "view_count"
	default:
		normalized.SortBy = "created_at"
	}
	if strings.EqualFold(strings.TrimSpace(normalized.Order), "asc") {
		normalized.Order = "ASC"
	} else {
		normalized.Order = "DESC"
	}

	normalized.Search = strings.TrimSpace(normalized.Search)
	normalized.AuthorName = strings.TrimSpace(normalized.AuthorName)
	normalized.Region = strings.TrimSpace(normalized.Region)
	normalized.Address = strings.TrimSpace(normalized.Address)
	normalized.TagID = strings.TrimSpace(normalized.TagID)
	normalized.AuthorID = strings.TrimSpace(normalized.AuthorID)
	normalized.Category = strings.TrimSpace(normalized.Category)
	normalized.Status = "published"
	return normalized
}

func (q PostListQuery) cacheSuffix() string {
	normalized := q.normalized()
	values := url.Values{}
	values.Set("schema", postListCandidateSchema)
	values.Set("viewer", strconv.FormatUint(uint64(normalized.ViewerID), 10))
	values.Set("page", strconv.Itoa(normalized.Page))
	values.Set("page_size", strconv.Itoa(normalized.PageSize))
	values.Set("sort", normalized.SortBy)
	values.Set("order", normalized.Order)
	values.Set("search", normalized.Search)
	values.Set("author_name", normalized.AuthorName)
	values.Set("region", normalized.Region)
	values.Set("address", normalized.Address)
	values.Set("tag", normalized.TagID)
	values.Set("author", normalized.AuthorID)
	values.Set("status", normalized.Status)
	values.Set("category", normalized.Category)
	if normalized.IsPinned == nil {
		values.Set("pinned", "any")
	} else {
		values.Set("pinned", strconv.FormatBool(*normalized.IsPinned))
	}
	return cache.HashKey(values.Encode())
}

func viewerVersionName(viewerID uint) string {
	return fmt.Sprintf("%s:%d", postListViewerVersionName, viewerID)
}
