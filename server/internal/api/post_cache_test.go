package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	internalcache "github.com/rpbox/server/internal/cache"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
)

func newPostCacheTestServer(t *testing.T) (*Server, *miniredis.Miniredis) {
	t.Helper()

	db := testutil.NewTestDB(t,
		&model.User{},
		&model.Post{},
		&model.Tag{},
		&model.PostTag{},
		&model.PostLike{},
		&model.PostFavorite{},
		&model.PostEditRequest{},
		&model.Comment{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
		&model.UserActivityLog{},
		&model.AdminActionLog{},
		&model.Notification{},
	)
	server := newTestServer(t, db)
	redisServer := miniredis.RunT(t)
	redisClient := redis.NewClient(&redis.Options{Addr: redisServer.Addr()})
	server.cache = internalcache.NewRedisCache(redisClient, internalcache.Options{})
	t.Cleanup(func() { _ = server.cache.Close() })
	return server, redisServer
}

func TestPostApprovalInvalidatesEveryViewerListCache(t *testing.T) {
	server, _ := newPostCacheTestServer(t)

	author := model.User{Username: "post-author", Email: "post-author@example.com", PassHash: "hash", Role: "user"}
	moderator := model.User{Username: "post-moderator", Email: "post-moderator@example.com", PassHash: "hash", Role: "moderator"}
	cachedViewer := model.User{Username: "cached-viewer", Email: "cached-viewer@example.com", PassHash: "hash", Role: "user"}
	freshViewer := model.User{Username: "fresh-viewer", Email: "fresh-viewer@example.com", PassHash: "hash", Role: "user"}
	if err := database.DB.Create(&[]*model.User{&author, &moderator, &cachedViewer, &freshViewer}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	created := performRequest(server.router, http.MethodPost, "/api/v1/posts", map[string]interface{}{
		"title":    "刚发布的新帖子",
		"content":  "等待版主审核的正文",
		"category": "other",
		"status":   "published",
	}, newTestToken(t, author))
	if created.Code != http.StatusCreated {
		t.Fatalf("create post: code=%d body=%s", created.Code, created.Body.String())
	}
	var pendingPost model.Post
	if err := json.Unmarshal(created.Body.Bytes(), &pendingPost); err != nil {
		t.Fatalf("decode created post: %v", err)
	}
	if pendingPost.Status != "pending" || pendingPost.ReviewStatus != "pending" {
		t.Fatalf("expected pending post, got status=%q review=%q", pendingPost.Status, pendingPost.ReviewStatus)
	}

	warmed := performRequest(server.router, http.MethodGet, "/api/v1/posts", nil, newTestToken(t, cachedViewer))
	if warmed.Code != http.StatusOK {
		t.Fatalf("warm cached viewer list: code=%d body=%s", warmed.Code, warmed.Body.String())
	}
	assertPostListContains(t, warmed, pendingPost.ID, false)

	versionBeforeApproval, err := server.cache.Version(context.Background(), postListCacheName)
	if err != nil {
		t.Fatalf("read cache version before approval: %v", err)
	}
	approved := performRequest(server.router, http.MethodPost, "/api/v1/moderator/review/posts/"+strconv.FormatUint(uint64(pendingPost.ID), 10), map[string]interface{}{
		"action": "approve",
	}, newTestToken(t, moderator))
	if approved.Code != http.StatusOK {
		t.Fatalf("approve post: code=%d body=%s", approved.Code, approved.Body.String())
	}
	versionAfterApproval, err := server.cache.Version(context.Background(), postListCacheName)
	if err != nil {
		t.Fatalf("read cache version after approval: %v", err)
	}
	if versionAfterApproval <= versionBeforeApproval {
		t.Fatalf("approval did not advance list cache generation: before=%d after=%d", versionBeforeApproval, versionAfterApproval)
	}

	for name, viewer := range map[string]model.User{
		"previously cached account": cachedViewer,
		"fresh account":             freshViewer,
	} {
		t.Run(name, func(t *testing.T) {
			response := performRequest(server.router, http.MethodGet, "/api/v1/posts", nil, newTestToken(t, viewer))
			if response.Code != http.StatusOK {
				t.Fatalf("list posts: code=%d body=%s", response.Code, response.Body.String())
			}
			assertPostListContains(t, response, pendingPost.ID, true)
		})
	}
}

func TestPostListCacheIdentitySeparatesEveryFilter(t *testing.T) {
	base := postListParams{
		UserID: 1, Page: 1, PageSize: 20, SortBy: "created_at", Order: "desc",
		Search: "search", AuthorName: "author", Region: "region", Address: "address",
		GuildID: "", TagID: "1", AuthorID: "2", Status: "published", Category: "other",
		ExcludeCategory: "event",
	}
	baseSuffix, err := postListCacheSuffix(base)
	if err != nil {
		t.Fatalf("base cache identity: %v", err)
	}

	pinned := false
	cases := map[string]func(*postListParams){
		"viewer":           func(p *postListParams) { p.UserID = 2 },
		"page":             func(p *postListParams) { p.Page = 2 },
		"page size":        func(p *postListParams) { p.PageSize = 50 },
		"sort":             func(p *postListParams) { p.SortBy = "like_count" },
		"order":            func(p *postListParams) { p.Order = "asc" },
		"search":           func(p *postListParams) { p.Search = "other search" },
		"author name":      func(p *postListParams) { p.AuthorName = "other author" },
		"region":           func(p *postListParams) { p.Region = "other region" },
		"address":          func(p *postListParams) { p.Address = "other address" },
		"guild":            func(p *postListParams) { p.GuildID = "3" },
		"tag":              func(p *postListParams) { p.TagID = "4" },
		"author":           func(p *postListParams) { p.AuthorID = "5" },
		"status":           func(p *postListParams) { p.Status = "all" },
		"category":         func(p *postListParams) { p.Category = "novel" },
		"exclude category": func(p *postListParams) { p.ExcludeCategory = "other" },
		"pinned":           func(p *postListParams) { p.IsPinned = &pinned },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			changed := base
			mutate(&changed)
			suffix, err := postListCacheSuffix(changed)
			if err != nil {
				t.Fatalf("cache identity: %v", err)
			}
			if suffix == baseSuffix {
				t.Fatalf("filter %q reused the base cache identity", name)
			}
		})
	}

	left := base
	left.Search = "x|author_name=y"
	left.AuthorName = "z"
	right := base
	right.Search = "x"
	right.AuthorName = "y|author_name=z"
	leftSuffix, _ := postListCacheSuffix(left)
	rightSuffix, _ := postListCacheSuffix(right)
	if leftSuffix == rightSuffix {
		t.Fatal("canonical cache identity collided for delimiter-like filter values")
	}
}

func TestPostManagementMutationsInvalidateListCache(t *testing.T) {
	server, _ := newPostCacheTestServer(t)
	moderator := model.User{Username: "cache-moderator", Email: "cache-moderator@example.com", PassHash: "hash", Role: "moderator"}
	author := model.User{Username: "cache-author", Email: "cache-author@example.com", PassHash: "hash", Role: "user"}
	bulkAuthor := model.User{Username: "cache-bulk-author", Email: "cache-bulk-author@example.com", PassHash: "hash", Role: "user"}
	if err := database.DB.Create(&[]*model.User{&moderator, &author, &bulkAuthor}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	posts := []*model.Post{
		{AuthorID: author.ID, Title: "状态管理帖", Content: "content", Status: "published", ReviewStatus: "approved", IsPublic: true},
		{AuthorID: author.ID, Title: "版主删除帖", Content: "content", Status: "published", ReviewStatus: "approved", IsPublic: true},
		{AuthorID: author.ID, Title: "标签管理帖", Content: "content", Status: "published", ReviewStatus: "approved", IsPublic: true},
		{AuthorID: bulkAuthor.ID, Title: "批量管理帖一", Content: "content", Status: "published", ReviewStatus: "approved", IsPublic: true},
		{AuthorID: bulkAuthor.ID, Title: "批量管理帖二", Content: "content", Status: "published", ReviewStatus: "approved", IsPublic: true},
	}
	if err := database.DB.Create(&posts).Error; err != nil {
		t.Fatalf("create posts: %v", err)
	}
	tag := model.Tag{Name: "cache-tag"}
	if err := database.DB.Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}

	moderatorToken := newTestToken(t, moderator)
	authorToken := newTestToken(t, author)
	assertPostMutationBumpsVersion(t, server, http.MethodPost, "/api/v1/moderator/manage/posts/"+strconv.FormatUint(uint64(posts[0].ID), 10)+"/pin", nil, moderatorToken)
	assertPostMutationBumpsVersion(t, server, http.MethodPost, "/api/v1/moderator/manage/posts/"+strconv.FormatUint(uint64(posts[0].ID), 10)+"/feature", nil, moderatorToken)
	assertPostMutationBumpsVersion(t, server, http.MethodPost, "/api/v1/moderator/manage/posts/"+strconv.FormatUint(uint64(posts[0].ID), 10)+"/hide", nil, moderatorToken)
	assertPostMutationBumpsVersion(t, server, http.MethodDelete, "/api/v1/moderator/manage/posts/"+strconv.FormatUint(uint64(posts[1].ID), 10), nil, moderatorToken)
	assertPostMutationBumpsVersion(t, server, http.MethodPost, "/api/v1/posts/"+strconv.FormatUint(uint64(posts[2].ID), 10)+"/tags", map[string]interface{}{"tag_id": tag.ID}, authorToken)
	assertPostMutationBumpsVersion(t, server, http.MethodDelete, "/api/v1/posts/"+strconv.FormatUint(uint64(posts[2].ID), 10)+"/tags/"+strconv.FormatUint(uint64(tag.ID), 10), nil, authorToken)
	assertPostMutationBumpsVersion(t, server, http.MethodPost, "/api/v1/moderator/users/"+strconv.FormatUint(uint64(bulkAuthor.ID), 10)+"/posts/disable", nil, moderatorToken)
	assertPostMutationBumpsVersion(t, server, http.MethodDelete, "/api/v1/moderator/users/"+strconv.FormatUint(uint64(bulkAuthor.ID), 10)+"/posts", nil, moderatorToken)
}

func TestPostListInvalidationSurvivesCanceledRequestContext(t *testing.T) {
	server, _ := newPostCacheTestServer(t)
	before, err := server.cache.Version(context.Background(), postListCacheName)
	if err != nil {
		t.Fatalf("read initial version: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	server.bumpPostListCache(ctx)

	after, err := server.cache.Version(context.Background(), postListCacheName)
	if err != nil {
		t.Fatalf("read invalidated version: %v", err)
	}
	if after <= before {
		t.Fatalf("canceled request context prevented invalidation: before=%d after=%d", before, after)
	}
}

func TestCreatePostSucceedsWhenRedisIsUnavailable(t *testing.T) {
	server, _ := newPostCacheTestServer(t)
	author := model.User{Username: "offline-cache-author", Email: "offline-cache-author@example.com", PassHash: "hash", Role: "user"}
	if err := database.DB.Create(&author).Error; err != nil {
		t.Fatalf("create author: %v", err)
	}
	if err := server.cache.Close(); err != nil {
		t.Fatalf("close cache: %v", err)
	}

	response := performRequest(server.router, http.MethodPost, "/api/v1/posts", map[string]interface{}{
		"title":    "Redis 不可用时的帖子",
		"content":  "数据库写入仍应成功",
		"category": "other",
		"status":   "published",
	}, newTestToken(t, author))
	if response.Code != http.StatusCreated {
		t.Fatalf("create post with unavailable cache: code=%d body=%s", response.Code, response.Body.String())
	}
}

func assertPostListContains(t *testing.T, response *httptest.ResponseRecorder, postID uint, expected bool) {
	t.Helper()
	var payload struct {
		Posts []model.Post `json:"posts"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode post list: %v", err)
	}
	found := false
	for _, post := range payload.Posts {
		if post.ID == postID {
			found = true
			break
		}
	}
	if found != expected {
		t.Fatalf("post %d presence=%v, expected %v", postID, found, expected)
	}
}

func assertPostMutationBumpsVersion(t *testing.T, server *Server, method, path string, body interface{}, token string) {
	t.Helper()
	before, err := server.cache.Version(context.Background(), postListCacheName)
	if err != nil {
		t.Fatalf("read cache version before %s %s: %v", method, path, err)
	}
	response := performRequest(server.router, method, path, body, token)
	if response.Code < http.StatusOK || response.Code >= http.StatusMultipleChoices {
		t.Fatalf("mutation %s %s failed: code=%d body=%s", method, path, response.Code, response.Body.String())
	}
	after, err := server.cache.Version(context.Background(), postListCacheName)
	if err != nil {
		t.Fatalf("read cache version after %s %s: %v", method, path, err)
	}
	if after <= before {
		t.Fatalf("mutation %s %s did not invalidate list cache: before=%d after=%d", method, path, before, after)
	}
}
