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
