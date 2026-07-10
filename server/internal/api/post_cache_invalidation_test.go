package api

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"

	miniredis "github.com/alicebob/miniredis/v2"
	"github.com/gin-gonic/gin"
	"github.com/rpbox/server/internal/database"
	"github.com/rpbox/server/internal/model"
	"github.com/rpbox/server/internal/testutil"
	"github.com/rpbox/server/pkg/auth"
	"gorm.io/gorm"
)

func TestPostListCandidateCacheHydratesLiveFields(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Post{},
		&model.PostTag{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
	)
	author := model.User{Username: "author", Email: "author@example.com", PassHash: "hash"}
	viewer := model.User{Username: "viewer", Email: "viewer@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&author, &viewer}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	older := createCacheTestPost(t, db, author.ID, "older", time.Now().Add(-time.Minute), 1)
	newer := createCacheTestPost(t, db, author.ID, "newer", time.Now(), 2)
	server := newCachedPostTestServer(t, db)
	token := newTestToken(t, viewer)

	defaultPath := "/api/v1/posts?page=1&page_size=20"
	likePath := defaultPath + "&sort=like_count&order=desc"

	initialDefault := getPostListPayload(t, server, defaultPath, token)
	assertPostOrder(t, initialDefault, newer.ID, older.ID)
	initialLikes := getPostListPayload(t, server, likePath, token)
	assertPostOrder(t, initialLikes, newer.ID, older.ID)

	future := time.Now().Add(time.Hour)
	if err := db.Model(&model.Post{}).
		Where("id = ?", older.ID).
		Updates(map[string]interface{}{
			"like_count": 9,
			"created_at": future,
		}).Error; err != nil {
		t.Fatalf("update live fields: %v", err)
	}

	hydrated := getPostListPayload(t, server, defaultPath, token)
	assertPostOrder(t, hydrated, newer.ID, older.ID)
	if hydrated.Posts[1].LikeCount != 9 {
		t.Fatalf("expected live like_count=9, got %d", hydrated.Posts[1].LikeCount)
	}

	liveLikes := getPostListPayload(t, server, likePath, token)
	assertPostOrder(t, liveLikes, older.ID, newer.ID)
}

func TestPostListCandidateCacheFailsClosedWhenPostBecomesPrivate(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Post{},
		&model.PostTag{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
	)
	author := model.User{Username: "author", Email: "author@example.com", PassHash: "hash"}
	viewer := model.User{Username: "viewer", Email: "viewer@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&author, &viewer}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}

	post := createCacheTestPost(t, db, author.ID, "private after cache", time.Now(), 0)
	server := newCachedPostTestServer(t, db)
	token := newTestToken(t, viewer)
	path := "/api/v1/posts?page=1&page_size=20"

	assertPostOrder(t, getPostListPayload(t, server, path, token), post.ID)
	if err := db.Model(&model.Post{}).
		Where("id = ?", post.ID).
		Update("is_public", false).Error; err != nil {
		t.Fatalf("make post private: %v", err)
	}

	assertPostOrder(t, getPostListPayload(t, server, path, token))
}

func TestPostCacheInvalidationModeratorPin(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Post{},
		&model.PostTag{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
		&model.AdminActionLog{},
	)
	moderator := model.User{
		Username: "moderator",
		Email:    "moderator@example.com",
		PassHash: "hash",
		Role:     "moderator",
	}
	author := model.User{Username: "author", Email: "author@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&moderator, &author}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	post := createCacheTestPost(t, db, author.ID, "pin me", time.Now(), 0)
	server := newCachedPostTestServer(t, db)
	token := newTestToken(t, moderator)

	unpinnedPath := "/api/v1/posts?is_pinned=false"
	assertPostOrder(t, getPostListPayload(t, server, unpinnedPath, token), post.ID)

	response := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/moderator/manage/posts/"+strconv.FormatUint(uint64(post.ID), 10)+"/pin",
		nil,
		token,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("pin post returned %d: %s", response.Code, response.Body.String())
	}

	assertPostOrder(t, getPostListPayload(t, server, unpinnedPath, token))
	assertPostOrder(t, getPostListPayload(t, server, "/api/v1/posts?is_pinned=true", token), post.ID)
}

func TestPostCacheInvalidationTagChanges(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Post{},
		&model.Tag{},
		&model.PostTag{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
	)
	author := model.User{Username: "author", Email: "author@example.com", PassHash: "hash"}
	tag := model.Tag{Name: "event"}
	if err := db.Create(&author).Error; err != nil {
		t.Fatalf("create author: %v", err)
	}
	if err := db.Create(&tag).Error; err != nil {
		t.Fatalf("create tag: %v", err)
	}
	post := createCacheTestPost(t, db, author.ID, "tag me", time.Now(), 0)
	server := newCachedPostTestServer(t, db)
	token := newTestToken(t, author)
	tagPath := "/api/v1/posts?tag_id=" + strconv.FormatUint(uint64(tag.ID), 10)

	assertPostOrder(t, getPostListPayload(t, server, tagPath, token))
	addResponse := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/posts/"+strconv.FormatUint(uint64(post.ID), 10)+"/tags",
		map[string]uint{"tag_id": tag.ID},
		token,
	)
	if addResponse.Code != http.StatusOK {
		t.Fatalf("add tag returned %d: %s", addResponse.Code, addResponse.Body.String())
	}
	assertPostOrder(t, getPostListPayload(t, server, tagPath, token), post.ID)

	removeResponse := performRequest(
		server.router,
		http.MethodDelete,
		"/api/v1/posts/"+strconv.FormatUint(uint64(post.ID), 10)+"/tags/"+strconv.FormatUint(uint64(tag.ID), 10),
		nil,
		token,
	)
	if removeResponse.Code != http.StatusOK {
		t.Fatalf("remove tag returned %d: %s", removeResponse.Code, removeResponse.Body.String())
	}
	assertPostOrder(t, getPostListPayload(t, server, tagPath, token))
}

func TestPostCacheInvalidationViewerBlockIsolation(t *testing.T) {
	db := testutil.NewTestDB(
		t,
		&model.User{},
		&model.Post{},
		&model.PostTag{},
		&model.UserBlock{},
		&model.UserHiddenContent{},
	)
	viewerOne := model.User{Username: "viewer-one", Email: "viewer-one@example.com", PassHash: "hash"}
	viewerTwo := model.User{Username: "viewer-two", Email: "viewer-two@example.com", PassHash: "hash"}
	author := model.User{Username: "author", Email: "author@example.com", PassHash: "hash"}
	if err := db.Create(&[]*model.User{&viewerOne, &viewerTwo, &author}).Error; err != nil {
		t.Fatalf("create users: %v", err)
	}
	post := createCacheTestPost(t, db, author.ID, "block author", time.Now(), 0)
	server := newCachedPostTestServer(t, db)
	viewerOneToken := newTestToken(t, viewerOne)
	viewerTwoToken := newTestToken(t, viewerTwo)

	assertPostOrder(t, getPostListPayload(t, server, "/api/v1/posts", viewerOneToken), post.ID)
	assertPostOrder(t, getPostListPayload(t, server, "/api/v1/posts", viewerTwoToken), post.ID)

	response := performRequest(
		server.router,
		http.MethodPost,
		"/api/v1/user/blocks",
		map[string]uint{"blocked_user_id": author.ID},
		viewerOneToken,
	)
	if response.Code != http.StatusOK {
		t.Fatalf("block user returned %d: %s", response.Code, response.Body.String())
	}

	assertPostOrder(t, getPostListPayload(t, server, "/api/v1/posts", viewerOneToken))
	assertPostOrder(t, getPostListPayload(t, server, "/api/v1/posts", viewerTwoToken), post.ID)

	unblockResponse := performRequest(
		server.router,
		http.MethodDelete,
		"/api/v1/user/blocks/"+strconv.FormatUint(uint64(author.ID), 10),
		nil,
		viewerOneToken,
	)
	if unblockResponse.Code != http.StatusOK {
		t.Fatalf("unblock user returned %d: %s", unblockResponse.Code, unblockResponse.Body.String())
	}
	assertPostOrder(t, getPostListPayload(t, server, "/api/v1/posts", viewerOneToken), post.ID)
}

type postListTestPayload struct {
	Posts []struct {
		ID        uint `json:"id"`
		LikeCount int  `json:"like_count"`
	} `json:"posts"`
	Total int64 `json:"total"`
}

func newCachedPostTestServer(t *testing.T, db *gorm.DB) *Server {
	t.Helper()

	redisServer := miniredis.RunT(t)
	host, port, err := net.SplitHostPort(redisServer.Addr())
	if err != nil {
		t.Fatalf("parse redis address: %v", err)
	}

	gin.SetMode(gin.TestMode)
	gin.DefaultWriter = io.Discard
	gin.DefaultErrorWriter = io.Discard

	cfg := testutil.NewTestConfig(t)
	cfg.Redis.Host = host
	cfg.Redis.Port = port
	auth.Init(cfg.JWT.Secret)
	database.DB = db

	server := NewServer(cfg)
	t.Cleanup(func() {
		if server.cache != nil {
			_ = server.cache.Close()
		}
	})
	return server
}

func createCacheTestPost(
	t *testing.T,
	db *gorm.DB,
	authorID uint,
	title string,
	createdAt time.Time,
	likeCount int,
) model.Post {
	t.Helper()

	post := model.Post{
		AuthorID:     authorID,
		Title:        title,
		Content:      title,
		Status:       "published",
		ReviewStatus: "approved",
		IsPublic:     true,
		LikeCount:    likeCount,
		CreatedAt:    createdAt,
		UpdatedAt:    createdAt,
	}
	if err := db.Create(&post).Error; err != nil {
		t.Fatalf("create post %q: %v", title, err)
	}
	return post
}

func getPostListPayload(t *testing.T, server *Server, path, token string) postListTestPayload {
	t.Helper()

	response := performRequest(server.router, http.MethodGet, path, nil, token)
	if response.Code != http.StatusOK {
		t.Fatalf("GET %s returned %d: %s", path, response.Code, response.Body.String())
	}

	var payload postListTestPayload
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
	return payload
}

func assertPostOrder(t *testing.T, payload postListTestPayload, expected ...uint) {
	t.Helper()

	if payload.Total != int64(len(expected)) {
		t.Fatalf("expected total=%d, got %d", len(expected), payload.Total)
	}
	if len(payload.Posts) != len(expected) {
		t.Fatalf("expected %d posts, got %d", len(expected), len(payload.Posts))
	}
	for index, id := range expected {
		if payload.Posts[index].ID != id {
			t.Fatalf(
				"post %d: expected id=%s, got id=%s",
				index,
				strconv.FormatUint(uint64(id), 10),
				strconv.FormatUint(uint64(payload.Posts[index].ID), 10),
			)
		}
	}
}
